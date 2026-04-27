package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

type modelCatalogRecord struct {
	model                           string
	canonicalModelID                string
	pricingLookupModelID            string
	displayName                     string
	iconKey                         string
	provider                        string
	mode                            string
	pricingCurrency                 string
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
	pricingStatus                   BillingPricingStatus
	pricingWarnings                 []string
}

func (s *ModelCatalogService) buildCatalogRecords(ctx context.Context) (map[string]*modelCatalogRecord, error) {
	details, err := s.catalogBaselineEntries(ctx)
	if err != nil {
		return nil, err
	}
	officialOverrides := s.loadOfficialPriceOverrides(ctx)
	saleOverrides := s.loadSalePriceOverrides(ctx)
	currencyPrefs := s.loadModelPricingCurrencies(ctx)
	records := make(map[string]*modelCatalogRecord, len(details))

	for _, detail := range details {
		entry := detail.ModelEntry
		record := ensureCatalogRecord(records, entry.ID)
		record.canonicalModelID = normalizeModelCatalogAlias(entry.ID)
		if record.canonicalModelID == "" {
			record.canonicalModelID = CanonicalizeModelNameForPricing(entry.ID)
		}
		record.pricingLookupModelID = firstRegistryString(entry.PricingLookupIDs...)
		if record.pricingLookupModelID == "" {
			record.pricingLookupModelID = firstRegistryString(entry.ProtocolIDs...)
		}
		if record.pricingLookupModelID == "" {
			record.pricingLookupModelID = record.canonicalModelID
		}
		record.displayName = entry.DisplayName
		record.provider = entry.Provider
		record.mode = inferModelMode(entry.ID, "")
		record.defaultAvailable = detail.Available
		record.defaultPlatforms = append([]string(nil), entry.Platforms...)
		if pricing, ok := s.resolveDynamicPricing(record); ok {
			record.upstreamPricing = pricingFromLiteLLM(pricing)
			record.basePricingSource = ModelCatalogPricingSourceDynamic
			record.pricingCurrency = defaultModelPricingCurrency(record.upstreamPricing.Currency)
			mergeRecordMetadata(record, pricing.LiteLLMProvider, pricing.Mode)
			record.supportsPromptCaching = pricing.SupportsPromptCaching
			record.supportsServiceTier = pricing.SupportsServiceTier
			record.longContextInputTokenThreshold = pricing.LongContextInputTokenThreshold
			record.longContextInputCostMultiplier = pricing.LongContextInputCostMultiplier
			record.longContextOutputCostMultiplier = pricing.LongContextOutputCostMultiplier
		} else if pricing, ok := s.resolveFallbackPricing(record); ok {
			record.upstreamPricing = pricingFromBilling(pricing)
			record.basePricingSource = ModelCatalogPricingSourceFallback
			record.pricingCurrency = defaultModelPricingCurrency(record.upstreamPricing.Currency)
			record.longContextInputTokenThreshold = pricing.LongContextInputThreshold
			record.longContextInputCostMultiplier = pricing.LongContextInputMultiplier
			record.longContextOutputCostMultiplier = pricing.LongContextOutputMultiplier
		}
	}
	for model, override := range officialOverrides {
		record, ok := resolveModelCatalogRecord(records, model)
		if !ok || record == nil {
			continue
		}
		record.officialOverridePricing = override
		mergeRecordMetadata(record, inferModelProvider(model), inferModelMode(model, record.mode))
	}
	for model, override := range saleOverrides {
		record, ok := resolveModelCatalogRecord(records, model)
		if !ok || record == nil {
			continue
		}
		record.saleOverridePricing = override
		mergeRecordMetadata(record, inferModelProvider(model), inferModelMode(model, record.mode))
	}
	for model, pref := range currencyPrefs {
		record, ok := resolveModelCatalogRecord(records, model)
		if !ok || record == nil || pref == nil {
			continue
		}
		record.pricingCurrency = defaultModelPricingCurrency(pref.Currency)
		meta := modelPricingMetadataFromPreference(pref)
		if record.officialOverridePricing != nil {
			applyCurrencyMetadataToCatalogPricing(&record.officialOverridePricing.ModelCatalogPricing, meta)
		}
		if record.saleOverridePricing != nil {
			applyCurrencyMetadataToCatalogPricing(&record.saleOverridePricing.ModelCatalogPricing, meta)
		}
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
		if record.saleOverridePricing != nil && normalizeModelPricingCurrency(record.saleOverridePricing.Currency) != "" {
			record.pricingCurrency = defaultModelPricingCurrency(record.saleOverridePricing.Currency)
		} else if record.officialOverridePricing != nil && normalizeModelPricingCurrency(record.officialOverridePricing.Currency) != "" {
			record.pricingCurrency = defaultModelPricingCurrency(record.officialOverridePricing.Currency)
		} else {
			record.pricingCurrency = defaultModelPricingCurrency(record.pricingCurrency)
		}
		record.iconKey = InferModelCatalogIconKey(record.model)
		record.officialPricing = applyPricingOverride(record.upstreamPricing, record.officialOverridePricing)
		record.salePricing = applyPricingOverride(record.officialPricing, record.saleOverridePricing)
	}
	s.populateCatalogAccessSources(ctx, records)
	applyBillingPricingStatus(details, records)
	return records, nil
}

func (s *ModelCatalogService) catalogBaselineEntries(ctx context.Context) ([]modelregistry.AdminModelDetail, error) {
	details := make([]modelregistry.AdminModelDetail, 0)
	if s.modelRegistryService != nil {
		registryDetails, err := s.modelRegistryService.pricingDetails(ctx)
		if err != nil {
			return nil, err
		}
		details = append(details, buildCatalogBaselineAdminDetails(registryDetails, true)...)
	} else {
		for _, entry := range buildCatalogBaselineRegistryEntries(modelregistry.SeedModels(), false) {
			details = append(details, modelregistry.AdminModelDetail{
				ModelEntry: entry,
				Available:  true,
			})
		}
	}
	details = appendBundledModelCatalogSeedDetails(details)
	details = s.appendPricingBackedSyntheticCatalogDetails(ctx, details)
	return details, nil
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

func mergeRecordMetadata(record *modelCatalogRecord, provider string, mode string) {
	if record.provider == "" {
		record.provider = provider
	}
	if record.mode == "" {
		record.mode = mode
	}
}

func inferModelProvider(model string) string {
	model = CanonicalizeModelNameForPricing(model)
	switch {
	case strings.HasPrefix(model, "claude"):
		return PlatformAnthropic
	case strings.HasPrefix(model, "gemini"):
		return PlatformGemini
	case strings.HasPrefix(model, "grok"):
		return PlatformGrok
	case strings.HasPrefix(model, "ernie"), strings.HasPrefix(model, "wenxin"), strings.HasPrefix(model, "baidu"):
		return "baidu"
	case strings.HasPrefix(model, "gpt"), strings.HasPrefix(model, "codex"), openAIReasoningModelPattern.MatchString(model):
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
	if strings.Contains(model, "video") {
		return "video"
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

func firstRegistryString(items ...string) string {
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			return item
		}
	}
	return ""
}
