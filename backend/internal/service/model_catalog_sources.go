package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

type defaultModelMetadata struct {
	provider  string
	mode      string
	platforms map[string]struct{}
}

type modelCatalogRecord struct {
	model                           string
	canonicalModelID                string
	pricingLookupModelID            string
	displayName                     string
	iconKey                         string
	provider                        string
	mode                            string
	defaultAvailable                bool
	defaultPlatforms                []string
	accessSources                   []string
	upstreamPricing                 *ModelCatalogPricing
	basePricingSource               string
	officialOverridePricing         *ModelPricingOverride
	officialPricing                 *ModelCatalogPricing
	saleOverridePricing             *ModelPricingOverride
	salePricing                     *ModelCatalogPricing
	supportsPromptCaching           bool
	supportsServiceTier             bool
	longContextInputTokenThreshold  int
	longContextInputCostMultiplier  float64
	longContextOutputCostMultiplier float64
}

func (s *ModelCatalogService) buildCatalogRecords(ctx context.Context) (map[string]*modelCatalogRecord, error) {
	defaultRegistry := s.buildDefaultModelRegistry()
	entries := s.loadCatalogEntries(ctx)
	officialOverrides := s.loadOfficialPriceOverrides(ctx)
	saleOverrides := s.loadSalePriceOverrides(ctx)
	records := make(map[string]*modelCatalogRecord, len(entries))

	for _, entry := range entries {
		record := ensureCatalogRecord(records, entry.Model)
		record.canonicalModelID = entry.CanonicalModelID
		record.pricingLookupModelID = entry.PricingLookupModelID
		record.displayName = entry.DisplayName
		record.provider = entry.Provider
		record.mode = entry.Mode
		applyDefaultMetadata(record, resolveDefaultModelMetadata(defaultRegistry, modelCatalogRecordLookupCandidates(record)...))
		if pricing, ok := s.resolveDynamicPricing(record); ok {
			record.upstreamPricing = pricingFromLiteLLM(pricing)
			record.basePricingSource = ModelCatalogPricingSourceDynamic
			mergeRecordMetadata(record, pricing.LiteLLMProvider, pricing.Mode)
			record.supportsPromptCaching = pricing.SupportsPromptCaching
			record.supportsServiceTier = pricing.SupportsServiceTier
			record.longContextInputTokenThreshold = pricing.LongContextInputTokenThreshold
			record.longContextInputCostMultiplier = pricing.LongContextInputCostMultiplier
			record.longContextOutputCostMultiplier = pricing.LongContextOutputCostMultiplier
		} else if pricing, ok := s.resolveFallbackPricing(record); ok {
			record.upstreamPricing = pricingFromBilling(pricing)
			record.basePricingSource = ModelCatalogPricingSourceFallback
			record.longContextInputTokenThreshold = pricing.LongContextInputThreshold
			record.longContextInputCostMultiplier = pricing.LongContextInputMultiplier
			record.longContextOutputCostMultiplier = pricing.LongContextOutputMultiplier
		}
	}
	for model, override := range officialOverrides {
		record, ok := records[normalizeModelCatalogAlias(model)]
		if !ok {
			continue
		}
		record.officialOverridePricing = override
		mergeRecordMetadata(record, inferModelProvider(model), inferModelMode(model, record.mode))
	}
	for model, override := range saleOverrides {
		record, ok := records[normalizeModelCatalogAlias(model)]
		if !ok {
			continue
		}
		record.saleOverridePricing = override
		mergeRecordMetadata(record, inferModelProvider(model), inferModelMode(model, record.mode))
	}
	for _, record := range records {
		if record.provider == "" {
			record.provider = inferModelProvider(record.model)
		}
		if record.mode == "" {
			record.mode = inferModelMode(record.model, record.mode)
		}
		if record.basePricingSource == "" {
			record.basePricingSource = ModelCatalogPricingSourceNone
		}
		if record.displayName == "" {
			record.displayName = FormatModelCatalogDisplayName(record.model)
		}
		record.iconKey = InferModelCatalogIconKey(record.model)
		record.officialPricing = applyPricingOverride(record.upstreamPricing, record.officialOverridePricing)
		record.salePricing = applyPricingOverride(record.officialPricing, record.saleOverridePricing)
	}
	s.populateCatalogAccessSources(ctx, records)
	return records, nil
}

func (s *ModelCatalogService) buildDefaultModelRegistry() map[string]*defaultModelMetadata {
	registry := make(map[string]*defaultModelMetadata)
	register := func(model, platform, provider, mode string) {
		key := CanonicalizeModelNameForPricing(model)
		if key == "" {
			return
		}
		meta, ok := registry[key]
		if !ok {
			meta = &defaultModelMetadata{provider: provider, mode: mode, platforms: map[string]struct{}{}}
			registry[key] = meta
		}
		if meta.provider == "" {
			meta.provider = provider
		}
		if meta.mode == "" {
			meta.mode = mode
		}
		meta.platforms[platform] = struct{}{}
	}
	for _, model := range openai.DefaultModelIDs() {
		register(model, PlatformOpenAI, PlatformOpenAI, inferModelMode(model, ""))
	}
	for _, model := range claude.DefaultModelIDs() {
		register(model, PlatformAnthropic, PlatformAnthropic, "chat")
	}
	for _, model := range gemini.DefaultModels() {
		register(model.Name, PlatformGemini, PlatformGemini, inferModelMode(model.Name, ""))
	}
	for _, model := range antigravity.DefaultModels() {
		register(model.ID, PlatformAntigravity, inferModelProvider(model.ID), inferModelMode(model.ID, ""))
	}
	for _, model := range antigravity.DefaultGeminiModels() {
		register(model.Name, PlatformAntigravity, PlatformGemini, inferModelMode(model.Name, ""))
	}
	for _, model := range DefaultSoraModels(s.cfg) {
		register(model.ID, PlatformSora, PlatformOpenAI, inferModelMode(model.ID, ""))
	}
	return registry
}

func modelCatalogRecordLookupCandidates(record *modelCatalogRecord) []string {
	if record == nil {
		return nil
	}
	seen := map[string]struct{}{}
	appendCandidate := func(items []string, value string) []string {
		value = CanonicalizeModelNameForPricing(value)
		if value == "" {
			return items
		}
		if _, ok := seen[value]; ok {
			return items
		}
		seen[value] = struct{}{}
		return append(items, value)
	}
	items := make([]string, 0, 6)
	items = appendCandidate(items, record.pricingLookupModelID)
	items = appendCandidate(items, record.canonicalModelID)
	items = appendCandidate(items, record.model)
	if record.model != "" {
		items = appendCandidate(items, strings.ReplaceAll(record.model, ".", "-"))
	}
	return items
}

func resolveDefaultModelMetadata(registry map[string]*defaultModelMetadata, candidates ...string) *defaultModelMetadata {
	meta := &defaultModelMetadata{platforms: map[string]struct{}{}}
	matched := false
	for _, candidate := range candidates {
		candidate = CanonicalizeModelNameForPricing(candidate)
		if candidate == "" {
			continue
		}
		current, ok := registry[candidate]
		if !ok {
			continue
		}
		matched = true
		if meta.provider == "" {
			meta.provider = current.provider
		}
		if meta.mode == "" {
			meta.mode = current.mode
		}
		for platform := range current.platforms {
			meta.platforms[platform] = struct{}{}
		}
	}
	if !matched {
		return nil
	}
	return meta
}

func (s *ModelCatalogService) resolveDynamicPricing(record *modelCatalogRecord) (*LiteLLMModelPricing, bool) {
	if s.pricingService == nil {
		return nil, false
	}
	for _, candidate := range modelCatalogRecordLookupCandidates(record) {
		if pricing := s.pricingService.GetModelPricing(candidate); pricing != nil {
			return pricing, true
		}
	}
	return nil, false
}

func (s *ModelCatalogService) resolveFallbackPricing(record *modelCatalogRecord) (*ModelPricing, bool) {
	if s.billingService == nil {
		return nil, false
	}
	for _, candidate := range modelCatalogRecordLookupCandidates(record) {
		pricing, err := s.billingService.GetModelPricing(candidate)
		if err == nil && pricing != nil {
			return pricing, true
		}
	}
	return nil, false
}

func modelCatalogRouteMatchCandidates(record *modelCatalogRecord) []string {
	return modelCatalogRecordLookupCandidates(record)
}

func (s *ModelCatalogService) collectRouteReferences(ctx context.Context, record *modelCatalogRecord) ([]ModelCatalogRouteReference, error) {
	if s.adminService == nil {
		return []ModelCatalogRouteReference{}, nil
	}
	if record == nil {
		return []ModelCatalogRouteReference{}, nil
	}
	candidates := modelCatalogRouteMatchCandidates(record)
	groups, err := s.adminService.GetAllGroups(ctx)
	if err != nil {
		return nil, err
	}
	references := make([]ModelCatalogRouteReference, 0)
	for _, group := range groups {
		types := make([]string, 0, 3)
		patterns := make([]string, 0)
		for pattern := range group.ModelRouting {
			for _, candidate := range candidates {
				if matchModelPattern(canonicalizeRoutingPattern(pattern), candidate) {
					types = append(types, "model_routing")
					patterns = append(patterns, pattern)
					break
				}
			}
		}
		for _, candidate := range candidates {
			if CanonicalizeModelNameForPricing(group.DefaultMappedModel) == candidate {
				types = append(types, "default_mapped_model")
				break
			}
		}
		if supportsModelScope(group, record.model, record.mode) {
			types = append(types, "supported_model_scope")
		}
		if len(types) == 0 {
			continue
		}
		sort.Strings(types)
		sort.Strings(patterns)
		references = append(references, ModelCatalogRouteReference{
			GroupID:                group.ID,
			GroupName:              group.Name,
			Platform:               group.Platform,
			ReferenceTypes:         compactStrings(types),
			MatchedRoutingPatterns: patterns,
		})
	}
	sort.Slice(references, func(i, j int) bool {
		if references[i].GroupName == references[j].GroupName {
			return references[i].GroupID < references[j].GroupID
		}
		return references[i].GroupName < references[j].GroupName
	})
	return references, nil
}

func ensureCatalogRecord(records map[string]*modelCatalogRecord, model string) *modelCatalogRecord {
	record, ok := records[model]
	if !ok {
		record = &modelCatalogRecord{model: model}
		records[model] = record
	}
	return record
}

func applyDefaultMetadata(record *modelCatalogRecord, meta *defaultModelMetadata) {
	if record == nil || meta == nil {
		return
	}
	record.defaultAvailable = true
	record.defaultPlatforms = sortedPlatformKeys(meta.platforms)
	mergeRecordMetadata(record, meta.provider, meta.mode)
}

func mergeRecordMetadata(record *modelCatalogRecord, provider string, mode string) {
	if record.provider == "" {
		record.provider = provider
	}
	if record.mode == "" {
		record.mode = mode
	}
}

func sortedPlatformKeys(platforms map[string]struct{}) []string {
	items := make([]string, 0, len(platforms))
	for platform := range platforms {
		items = append(items, platform)
	}
	sort.Strings(items)
	return items
}

func inferModelProvider(model string) string {
	model = CanonicalizeModelNameForPricing(model)
	switch {
	case strings.HasPrefix(model, "claude"):
		return PlatformAnthropic
	case strings.HasPrefix(model, "gemini"):
		return PlatformGemini
	case strings.HasPrefix(model, "gpt"), strings.HasPrefix(model, "sora"), strings.HasPrefix(model, "codex"), openAIReasoningModelPattern.MatchString(model), strings.HasPrefix(model, "prompt-enhance"):
		return PlatformOpenAI
	default:
		return ""
	}
}

func inferModelMode(model string, current string) string {
	if current != "" {
		return current
	}
	model = CanonicalizeModelNameForPricing(model)
	if config, ok := soraModelConfigs[model]; ok {
		return config.Type
	}
	if strings.Contains(model, "image") {
		return "image"
	}
	return "chat"
}

func canonicalizeRoutingPattern(pattern string) string {
	trimmed := strings.TrimSpace(pattern)
	if strings.HasSuffix(trimmed, "*") {
		return CanonicalizeModelNameForPricing(strings.TrimSuffix(trimmed, "*")) + "*"
	}
	return CanonicalizeModelNameForPricing(trimmed)
}

func supportsModelScope(group Group, model string, mode string) bool {
	if group.Platform != PlatformAntigravity || len(group.SupportedModelScopes) == 0 {
		return false
	}
	modelScope := ""
	if strings.HasPrefix(model, "claude") {
		modelScope = "claude"
	} else if strings.HasPrefix(model, "gemini") && mode == "image" {
		modelScope = "gemini_image"
	} else if strings.HasPrefix(model, "gemini") {
		modelScope = "gemini_text"
	}
	for _, scope := range group.SupportedModelScopes {
		if scope == modelScope {
			return true
		}
	}
	return false
}

func compactStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
