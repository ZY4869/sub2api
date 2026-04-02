package service

import (
	"sort"
	"strings"
)

const (
	GrokModelAuto           = "grok-auto"
	GrokModel3Fast          = "grok-3-fast"
	GrokModel4Expert        = "grok-4-expert"
	GrokModel4Heavy         = "grok-4-heavy"
	GrokModelImagineFast    = "grok-imagine-1.0-fast"
	GrokModelImagine        = "grok-imagine-1.0"
	GrokModelImagineEdit    = "grok-imagine-1.0-edit"
	GrokModelImagineVideo   = "grok-imagine-1.0-video"
	grokMediaTypeImage      = "image"
	grokMediaTypeImageEdit  = "image_edit"
	grokMediaTypeVideo      = "video"
	grokAPIKeyRegistryRoute = "grok"
	grokDefaultPollInterval = 2
	grokDefaultPollTimeout  = 180
)

type grokModelDescriptor struct {
	PublicID                string
	LegacyAliases           []string
	MediaType               string
	PreferredAPIKeyUpstream string
	HeavyOnly               bool
}

var grokCanonicalDescriptors = []grokModelDescriptor{
	{PublicID: GrokModelAuto, LegacyAliases: []string{"grok-beta"}},
	{PublicID: GrokModel3Fast, LegacyAliases: []string{"grok-3-fast-beta"}, PreferredAPIKeyUpstream: "grok-3-fast-beta"},
	{PublicID: GrokModel4Expert, LegacyAliases: []string{"grok-4", "grok-4-0709"}, PreferredAPIKeyUpstream: "grok-4"},
	{PublicID: GrokModel4Heavy, HeavyOnly: true},
	{PublicID: GrokModelImagineFast},
	{PublicID: GrokModelImagine, LegacyAliases: []string{"grok-imagine-image"}, MediaType: grokMediaTypeImage, PreferredAPIKeyUpstream: "grok-imagine-image"},
	{PublicID: GrokModelImagineEdit, MediaType: grokMediaTypeImageEdit},
	{PublicID: GrokModelImagineVideo, LegacyAliases: []string{"grok-imagine-video"}, MediaType: grokMediaTypeVideo, PreferredAPIKeyUpstream: "grok-imagine-video"},
}

var (
	grokCanonicalModelSet          map[string]grokModelDescriptor
	grokLegacyAliasToPublicModelID map[string]string
)

func init() {
	grokCanonicalModelSet = make(map[string]grokModelDescriptor, len(grokCanonicalDescriptors))
	grokLegacyAliasToPublicModelID = make(map[string]string, len(grokCanonicalDescriptors)*2)
	for _, item := range grokCanonicalDescriptors {
		publicID := normalizeRegistryID(item.PublicID)
		item.PublicID = publicID
		grokCanonicalModelSet[publicID] = item
		for _, alias := range item.LegacyAliases {
			if normalized := normalizeRegistryID(alias); normalized != "" {
				grokLegacyAliasToPublicModelID[normalized] = publicID
			}
		}
	}
}

func NormalizeGrokPublicModelID(model string) string {
	normalized := normalizeRegistryID(model)
	if normalized == "" {
		return ""
	}
	if _, ok := grokCanonicalModelSet[normalized]; ok {
		return normalized
	}
	if publicID, ok := grokLegacyAliasToPublicModelID[normalized]; ok {
		return publicID
	}
	return normalized
}

func IsCanonicalGrokPublicModel(model string) bool {
	_, ok := grokCanonicalModelSet[normalizeRegistryID(model)]
	return ok
}

func GrokLegacyAliasesForPublicModel(model string) []string {
	descriptor, ok := grokCanonicalModelSet[NormalizeGrokPublicModelID(model)]
	if !ok || len(descriptor.LegacyAliases) == 0 {
		return nil
	}
	items := make([]string, 0, len(descriptor.LegacyAliases))
	for _, alias := range descriptor.LegacyAliases {
		if normalized := normalizeRegistryID(alias); normalized != "" {
			items = append(items, normalized)
		}
	}
	return items
}

func GrokMediaTypeForModel(model string) string {
	descriptor, ok := grokCanonicalModelSet[NormalizeGrokPublicModelID(model)]
	if !ok {
		return ""
	}
	return descriptor.MediaType
}

func GrokIsVideoModel(model string) bool {
	return GrokMediaTypeForModel(model) == grokMediaTypeVideo
}

func GrokIsImageModel(model string) bool {
	return GrokMediaTypeForModel(model) == grokMediaTypeImage
}

func GrokIsImageEditModel(model string) bool {
	return GrokMediaTypeForModel(model) == grokMediaTypeImageEdit
}

func GrokAPIKeyPreferredUpstreamModel(publicID string) (string, bool) {
	descriptor, ok := grokCanonicalModelSet[NormalizeGrokPublicModelID(publicID)]
	if !ok {
		return "", false
	}
	upstream := normalizeRegistryID(descriptor.PreferredAPIKeyUpstream)
	if upstream == "" {
		return "", false
	}
	return upstream, true
}

func GrokAPIKeyResolvedUpstreamModel(requestedModel string) string {
	normalizedRequested := normalizeRegistryID(requestedModel)
	if normalizedRequested == "" {
		return ""
	}
	if _, ok := grokLegacyAliasToPublicModelID[normalizedRequested]; ok {
		return normalizedRequested
	}
	if upstream, ok := GrokAPIKeyPreferredUpstreamModel(normalizedRequested); ok {
		return upstream
	}
	return normalizedRequested
}

func GrokDefaultPublicModelIDsForTier(tier string) []string {
	models := make([]string, 0, len(grokCanonicalDescriptors))
	for _, item := range grokCanonicalDescriptors {
		if item.HeavyOnly && NormalizeGrokTierValue(tier) != GrokTierHeavy {
			continue
		}
		models = append(models, item.PublicID)
	}
	return models
}

func normalizeGrokModelMappingForStorage(accountType string, raw map[string]any, tier string) map[string]any {
	if len(raw) == 0 {
		if strings.TrimSpace(strings.ToLower(accountType)) != AccountTypeSSO {
			return raw
		}
		defaults := DefaultGrokModelMappingForTier(tier)
		if len(defaults) == 0 {
			return nil
		}
		return defaults
	}

	result := make(map[string]any, len(raw))
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		target, ok := raw[key].(string)
		if !ok {
			continue
		}
		alias := strings.TrimSpace(key)
		target = strings.TrimSpace(target)
		if alias == "" || target == "" {
			continue
		}
		normalizedAlias := NormalizeGrokPublicModelID(alias)
		if normalizedAlias == "" {
			normalizedAlias = normalizeRegistryID(alias)
		}
		if normalizedAlias == "" {
			continue
		}

		normalizedTarget := normalizeRegistryID(target)
		if strings.TrimSpace(strings.ToLower(accountType)) == AccountTypeSSO {
			normalizedTarget = NormalizeGrokPublicModelID(target)
		} else if publicTarget := NormalizeGrokPublicModelID(target); publicTarget != "" {
			if upstream, ok := GrokAPIKeyPreferredUpstreamModel(publicTarget); ok {
				normalizedTarget = upstream
			}
		}
		if normalizedTarget == "" {
			continue
		}
		result[normalizedAlias] = normalizedTarget
	}
	if len(result) == 0 {
		if strings.TrimSpace(strings.ToLower(accountType)) == AccountTypeSSO {
			return DefaultGrokModelMappingForTier(tier)
		}
		return nil
	}
	return result
}

func grokModelMatchCandidates(model string) []string {
	normalized := normalizeRegistryID(model)
	if normalized == "" {
		return nil
	}
	items := []string{normalized}
	if publicID := NormalizeGrokPublicModelID(normalized); publicID != "" && publicID != normalized {
		items = append(items, publicID)
	}
	if publicID := NormalizeGrokPublicModelID(normalized); publicID != "" {
		for _, alias := range GrokLegacyAliasesForPublicModel(publicID) {
			if alias == "" || alias == normalized {
				continue
			}
			items = append(items, alias)
		}
	}
	return dedupeStrings(items)
}

func canonicalizeGrokDetectedModels(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	result := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		normalized := normalizeRegistryID(model)
		if normalized == "" {
			continue
		}
		publicID := grokPublicModelForDetectedSource(normalized)
		if publicID == "" {
			publicID = NormalizeGrokPublicModelID(normalized)
		}
		if publicID == "" {
			publicID = normalized
		}
		if _, ok := seen[publicID]; ok {
			continue
		}
		seen[publicID] = struct{}{}
		result = append(result, publicID)
	}
	return result
}

func grokPublicModelForDetectedSource(sourceModel string) string {
	normalized := normalizeRegistryID(sourceModel)
	if normalized == "" {
		return ""
	}
	publicID := NormalizeGrokPublicModelID(normalized)
	if publicID == "" || publicID == normalized {
		return normalized
	}
	preferred, ok := GrokAPIKeyPreferredUpstreamModel(publicID)
	if !ok || preferred == "" {
		return normalized
	}
	if preferred != normalized {
		return normalized
	}
	return publicID
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}
