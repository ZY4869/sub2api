package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func normalizeRuntimeRegistryEntry(input UpsertModelRegistryEntryInput) (modelregistry.ModelEntry, error) {
	return normalizePersistedEntry(modelregistry.ModelEntry{
		ID:                   input.ID,
		DisplayName:          input.DisplayName,
		Provider:             input.Provider,
		Platforms:            input.Platforms,
		ProtocolIDs:          input.ProtocolIDs,
		Aliases:              input.Aliases,
		PricingLookupIDs:     input.PricingLookupIDs,
		PreferredProtocolIDs: input.PreferredProtocolIDs,
		Modalities:           input.Modalities,
		Capabilities:         input.Capabilities,
		UIPriority:           input.UIPriority,
		ExposedIn:            input.ExposedIn,
		Status:               input.Status,
		DeprecatedAt:         input.DeprecatedAt,
		ReplacedBy:           input.ReplacedBy,
		DeprecationNotice:    input.DeprecationNotice,
	})
}

func normalizePersistedEntry(entry modelregistry.ModelEntry) (modelregistry.ModelEntry, error) {
	entry.ID = normalizeRegistryID(entry.ID)
	if entry.ID == "" {
		return modelregistry.ModelEntry{}, infraerrors.BadRequest("MODEL_REQUIRED", "model id is required")
	}
	entry.DisplayName = strings.TrimSpace(entry.DisplayName)
	if entry.DisplayName == "" {
		entry.DisplayName = FormatModelCatalogDisplayName(entry.ID)
	}
	entry.Provider = strings.TrimSpace(strings.ToLower(entry.Provider))
	if entry.Provider == "" {
		entry.Provider = inferModelProvider(entry.ID)
	}
	if len(entry.Platforms) == 0 {
		entry.Platforms = defaultPlatformsForProvider(entry.Provider)
	}
	entry.Platforms = normalizeStringList(entry.Platforms, normalizeRegistryPlatform)
	if len(entry.Platforms) == 0 {
		entry.Platforms = defaultPlatformsForProvider(entry.Provider)
	}
	entry.ProtocolIDs = normalizeStringList(entry.ProtocolIDs, normalizeRegistryID)
	if len(entry.ProtocolIDs) == 0 {
		entry.ProtocolIDs = []string{entry.ID}
	}
	entry.Aliases = normalizeStringList(mergeRegistryStrings(entry.Aliases, entry.ProtocolIDs...), normalizeRegistryID)
	entry.PricingLookupIDs = normalizeStringList(entry.PricingLookupIDs, normalizeRegistryID)
	if len(entry.PricingLookupIDs) == 0 {
		entry.PricingLookupIDs = []string{entry.ProtocolIDs[0]}
	}
	entry.PreferredProtocolIDs = normalizePreferredProtocolIDs(entry.ID, entry.ProtocolIDs, entry.PreferredProtocolIDs)
	entry.Modalities = normalizeStringList(entry.Modalities, normalizeLowerTrimmed)
	if len(entry.Modalities) == 0 {
		entry.Modalities = defaultModalitiesForMode(inferModelMode(entry.ID, ""))
	}
	capabilities, err := normalizeRegistryCapabilities(entry.Capabilities)
	if err != nil {
		return modelregistry.ModelEntry{}, err
	}
	entry.Capabilities = capabilities
	if len(entry.Capabilities) == 0 {
		entry.Capabilities = defaultCapabilitiesForMode(inferModelMode(entry.ID, ""))
	}
	if entry.UIPriority <= 0 {
		if seedEntry, ok := modelregistry.SeedModelByID(entry.ID); ok {
			entry.UIPriority = seedEntry.UIPriority
		} else {
			entry.UIPriority = 5000
		}
	}
	hadExplicitExposures := entry.ExposedIn != nil
	entry.ExposedIn = normalizeStringList(entry.ExposedIn, normalizeLowerTrimmed)
	if len(entry.ExposedIn) == 0 && !hadExplicitExposures {
		if seedEntry, ok := modelregistry.SeedModelByID(entry.ID); ok && len(seedEntry.ExposedIn) > 0 {
			entry.ExposedIn = append([]string(nil), seedEntry.ExposedIn...)
		} else {
			entry.ExposedIn = []string{"runtime", "whitelist"}
		}
	}
	entry.Status = normalizeRegistryStatus(entry.Status)
	entry.DeprecatedAt = strings.TrimSpace(entry.DeprecatedAt)
	entry.ReplacedBy = normalizeRegistryID(entry.ReplacedBy)
	entry.DeprecationNotice = strings.TrimSpace(entry.DeprecationNotice)
	if entry.Status != "deprecated" {
		entry.DeprecatedAt = ""
		entry.DeprecationNotice = ""
		if entry.Status != "beta" {
			entry.ReplacedBy = ""
		}
	}
	return entry, nil
}

func defaultPlatformsForProvider(provider string) []string {
	provider = normalizeRegistryPlatform(provider)
	if provider == "" {
		return nil
	}
	return []string{provider}
}

func defaultModalitiesForMode(mode string) []string {
	if mode == "video" {
		return []string{"text", "video"}
	}
	if mode == "image" {
		return []string{"text", "image"}
	}
	return []string{"text"}
}

func defaultCapabilitiesForMode(mode string) []string {
	if mode == "video" {
		return []string{"video_generation"}
	}
	if mode == "image" {
		return []string{"image_generation"}
	}
	return []string{"text"}
}

func normalizeRegistryID(value string) string {
	return modelregistry.NormalizeID(value)
}

func normalizeRegistryPlatform(value string) string {
	return modelregistry.NormalizePlatform(value)
}

func normalizeRegistryStatus(value string) string {
	switch normalizeLowerTrimmed(value) {
	case "beta", "deprecated":
		return normalizeLowerTrimmed(value)
	default:
		return "stable"
	}
}

func normalizePreferredProtocolIDs(modelID string, protocolIDs []string, raw map[string]string) map[string]string {
	normalized := make(map[string]string)
	for key, value := range raw {
		route := normalizeRegistryRouteKey(key)
		value = normalizeRegistryID(value)
		if route == "" || value == "" {
			continue
		}
		normalized[route] = value
	}
	if normalized["default"] == "" {
		normalized["default"] = normalizeRegistryID(modelID)
	}
	if normalized["anthropic_oauth"] == "" && len(protocolIDs) > 0 {
		normalized["anthropic_oauth"] = normalizeRegistryID(protocolIDs[0])
	}
	if normalized["kiro"] == "" {
		normalized["kiro"] = firstNonEmptyString(
			normalized["anthropic_oauth"],
			firstString(protocolIDs),
			modelID,
		)
	}
	if normalized["anthropic_apikey"] == "" {
		normalized["anthropic_apikey"] = normalizeRegistryID(modelID)
	}
	if normalized["openai"] == "" {
		normalized["openai"] = normalizeRegistryID(modelID)
	}
	if normalized["copilot"] == "" {
		normalized["copilot"] = firstNonEmptyString(
			normalized["openai"],
			modelID,
		)
	}
	if normalized["gemini"] == "" {
		normalized["gemini"] = normalizeRegistryID(modelID)
	}
	if normalized["antigravity"] == "" {
		normalized["antigravity"] = normalizeRegistryID(modelID)
	}
	if normalized["sora"] == "" {
		normalized["sora"] = normalizeRegistryID(modelID)
	}
	return normalized
}

func normalizeRegistryRouteKey(value string) string {
	value = normalizeLowerTrimmed(value)
	switch value {
	case "", "default":
		return "default"
	case "anthropic", "anthropic_oauth", "claude_oauth":
		return "anthropic_oauth"
	case "kiro":
		return "kiro"
	case "anthropic_apikey", "anthropic_api_key", "claude_apikey":
		return "anthropic_apikey"
	case "openai", "copilot", "gemini", "antigravity", "sora":
		return value
	default:
		return value
	}
}

var batchSyncExposureTargets = map[string]struct{}{
	"whitelist": {},
	"use_key":   {},
	"test":      {},
	"runtime":   {},
}

func normalizeBatchSyncExposureMode(mode string) string {
	switch normalizeLowerTrimmed(mode) {
	case "", "add":
		return "add"
	case "remove":
		return "remove"
	case "replace":
		return "replace"
	default:
		return ""
	}
}

func normalizeBatchSyncExposureTargets(exposures []string) ([]string, error) {
	targets := normalizeStringList(exposures, normalizeLowerTrimmed)
	if len(targets) == 0 {
		return nil, infraerrors.BadRequest("MODEL_REGISTRY_EXPOSURE_REQUIRED", "at least one exposure target is required")
	}
	for _, target := range targets {
		if _, ok := batchSyncExposureTargets[target]; !ok {
			return nil, infraerrors.BadRequest("MODEL_REGISTRY_EXPOSURE_INVALID", "invalid exposure target: "+target)
		}
	}
	return targets, nil
}

func normalizeLowerTrimmed(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func normalizeRegistryCapabilities(items []string) ([]string, error) {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		normalized, err := normalizeRegistryCapability(item)
		if err != nil {
			return nil, err
		}
		if normalized == "" {
			continue
		}
		seen[normalized] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for _, capability := range modelRegistryCapabilityOrder {
		if _, ok := seen[capability]; ok {
			result = append(result, capability)
		}
	}
	return result, nil
}

func normalizeRegistryCapability(value string) (string, error) {
	value = normalizeLowerTrimmed(value)
	if value == "" {
		return "", nil
	}
	if alias, ok := modelRegistryCapabilityAliases[value]; ok {
		value = alias
	}
	for _, capability := range modelRegistryCapabilityOrder {
		if value == capability {
			return value, nil
		}
	}
	return "", infraerrors.BadRequest("MODEL_REGISTRY_CAPABILITY_INVALID", "invalid capability: "+value)
}

func normalizeStringList(items []string, normalize func(string) string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = normalize(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func augmentDiscoveredEntry(entry modelregistry.ModelEntry, sourceModelID string, canonicalModelID string, sourcePlatform string) (modelregistry.ModelEntry, bool, error) {
	updated := modelregistry.ModelEntry{
		ID:               entry.ID,
		DisplayName:      entry.DisplayName,
		Provider:         entry.Provider,
		Platforms:        append([]string(nil), entry.Platforms...),
		ProtocolIDs:      append([]string(nil), entry.ProtocolIDs...),
		Aliases:          append([]string(nil), entry.Aliases...),
		PricingLookupIDs: append([]string(nil), entry.PricingLookupIDs...),
		Modalities:       append([]string(nil), entry.Modalities...),
		Capabilities:     append([]string(nil), entry.Capabilities...),
		UIPriority:       entry.UIPriority,
		ExposedIn:        append([]string(nil), entry.ExposedIn...),
	}
	changed := false
	platforms, err := discoveredPlatforms(updated.Provider, sourcePlatform)
	if err != nil {
		return modelregistry.ModelEntry{}, false, err
	}
	if merged := mergeRegistryStrings(updated.Platforms, platforms...); !sameStringSlice(updated.Platforms, merged) {
		updated.Platforms = merged
		changed = true
	}
	if merged := mergeRegistryStrings(updated.ExposedIn, "runtime", "test"); !sameStringSlice(updated.ExposedIn, merged) {
		updated.ExposedIn = merged
		changed = true
	}
	if sourceModelID != "" && sourceModelID != updated.ID {
		merged := mergeRegistryStrings(updated.ProtocolIDs, sourceModelID)
		if !sameStringSlice(updated.ProtocolIDs, merged) {
			updated.ProtocolIDs = merged
			changed = true
		}
	}
	if pricingID, ok := modelregistry.ResolveToPricingID(canonicalModelID); ok {
		merged := mergeRegistryStrings(updated.PricingLookupIDs, pricingID)
		if !sameStringSlice(updated.PricingLookupIDs, merged) {
			updated.PricingLookupIDs = merged
			changed = true
		}
	}
	normalized, err := normalizePersistedEntry(updated)
	if err != nil {
		return modelregistry.ModelEntry{}, false, err
	}
	return normalized, changed, nil
}

func discoveredPlatforms(provider string, sourcePlatform string) ([]string, error) {
	platform := normalizeRegistryPlatform(sourcePlatform)
	if isRuntimeSupportedPlatform(platform) {
		return []string{platform}, nil
	}
	platforms := defaultPlatformsForProvider(provider)
	if len(platforms) > 0 {
		return platforms, nil
	}
	return nil, infraerrors.BadRequest("MODEL_RUNTIME_PLATFORM_UNSUPPORTED", "unable to infer runtime platform from imported model")
}

func providerOrPlatform(provider string, sourcePlatform string) string {
	provider = normalizeRegistryPlatform(provider)
	if provider != "" {
		return provider
	}
	return normalizeRegistryPlatform(sourcePlatform)
}

func isRuntimeSupportedPlatform(platform string) bool {
	switch normalizeRegistryPlatform(platform) {
	case PlatformOpenAI, PlatformAnthropic, PlatformGemini, PlatformAntigravity, PlatformSora, PlatformKiro, PlatformCopilot, PlatformGrok:
		return true
	default:
		return false
	}
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return normalizeRegistryID(values[0])
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if normalized := normalizeRegistryID(value); normalized != "" {
			return normalized
		}
	}
	return ""
}

func mergeRegistryStrings(current []string, items ...string) []string {
	merged := append([]string(nil), current...)
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		exists := false
		for _, existing := range merged {
			if existing == item {
				exists = true
				break
			}
		}
		if !exists {
			merged = append(merged, item)
		}
	}
	return merged
}

func sameStringSlice(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func compactRegistryStrings(items ...string) []string {
	return normalizeStringList(items, normalizeRegistryID)
}
