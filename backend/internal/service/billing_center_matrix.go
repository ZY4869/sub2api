package service

import (
	"fmt"
	"strings"
)

const (
	geminiMatrixRuleIDPrefix = "gemini_matrix"
	geminiMatrixRulePriority = 5000
)

var geminiMatrixSurfaces = []string{
	BillingSurfaceGeminiNative,
	BillingSurfaceOpenAICompat,
	BillingSurfaceGeminiLive,
	BillingSurfaceInteractions,
	BillingSurfaceVertexExisting,
}

var geminiMatrixServiceTiers = []string{
	BillingServiceTierStandard,
	BillingServiceTierFlex,
	BillingServiceTierPriority,
}

var geminiMatrixChargeSlots = []string{
	BillingChargeSlotTextInput,
	BillingChargeSlotTextInputLongContext,
	BillingChargeSlotTextOutput,
	BillingChargeSlotTextOutputLongContext,
	BillingChargeSlotAudioInput,
	BillingChargeSlotAudioOutput,
	BillingChargeSlotCacheCreate,
	BillingChargeSlotCacheRead,
	BillingChargeSlotCacheStorageTokenHour,
	BillingChargeSlotImageOutput,
	BillingChargeSlotVideoRequest,
	BillingChargeSlotFileSearchEmbeddingToken,
	BillingChargeSlotFileSearchRetrievalToken,
	BillingChargeSlotGroundingSearchRequest,
	BillingChargeSlotGroundingMapsRequest,
}

type geminiMatrixSlotRuleSpec struct {
	unit      string
	operation string
	matchers  BillingRuleMatchers
}

var geminiMatrixSlotSpecs = map[string]geminiMatrixSlotRuleSpec{
	BillingChargeSlotTextInput: {
		unit:      BillingUnitInputToken,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			InputModality: "text",
			ContextWindow: BillingContextWindowStandard,
		},
	},
	BillingChargeSlotTextInputLongContext: {
		unit:      BillingUnitInputToken,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			InputModality: "text",
			ContextWindow: BillingContextWindowLong,
		},
	},
	BillingChargeSlotTextOutput: {
		unit:      BillingUnitOutputToken,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			OutputModality: "text",
			ContextWindow:  BillingContextWindowStandard,
		},
	},
	BillingChargeSlotTextOutputLongContext: {
		unit:      BillingUnitOutputToken,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			OutputModality: "text",
			ContextWindow:  BillingContextWindowLong,
		},
	},
	BillingChargeSlotAudioInput: {
		unit:      BillingUnitInputToken,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			InputModality: "audio",
		},
	},
	BillingChargeSlotAudioOutput: {
		unit:      BillingUnitOutputToken,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			OutputModality: "audio",
		},
	},
	BillingChargeSlotCacheCreate: {
		unit:      BillingUnitCacheCreateToken,
		operation: "cache_usage",
		matchers: BillingRuleMatchers{
			CachePhase: "create",
		},
	},
	BillingChargeSlotCacheRead: {
		unit:      BillingUnitCacheReadToken,
		operation: "cache_usage",
		matchers: BillingRuleMatchers{
			CachePhase: "read",
		},
	},
	BillingChargeSlotCacheStorageTokenHour: {
		unit:      BillingUnitCacheStorageTokenHour,
		operation: "cache_storage",
	},
	BillingChargeSlotImageOutput: {
		unit:      BillingUnitImage,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			OutputModality: "image",
		},
	},
	BillingChargeSlotVideoRequest: {
		unit:      BillingUnitVideoRequest,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			OutputModality: "video",
		},
	},
	BillingChargeSlotFileSearchEmbeddingToken: {
		unit:      BillingUnitFileSearchEmbedding,
		operation: "file_search_embedding",
	},
	BillingChargeSlotFileSearchRetrievalToken: {
		unit:      BillingUnitFileSearchRetrieval,
		operation: "file_search_retrieval",
	},
	BillingChargeSlotGroundingSearchRequest: {
		unit:      BillingUnitGroundingSearchRequest,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			GroundingKind: "search",
		},
	},
	BillingChargeSlotGroundingMapsRequest: {
		unit:      BillingUnitGroundingMapsRequest,
		operation: "generate_content",
		matchers: BillingRuleMatchers{
			GroundingKind: "maps",
		},
	},
}

func newGeminiBillingMatrix() *GeminiBillingMatrix {
	matrix := &GeminiBillingMatrix{
		Surfaces:     append([]string(nil), geminiMatrixSurfaces...),
		ServiceTiers: append([]string(nil), geminiMatrixServiceTiers...),
		ChargeSlots:  append([]string(nil), geminiMatrixChargeSlots...),
		Rows:         make([]GeminiBillingMatrixRow, 0, len(geminiMatrixSurfaces)*len(geminiMatrixServiceTiers)),
	}
	for _, surface := range geminiMatrixSurfaces {
		for _, tier := range geminiMatrixServiceTiers {
			row := GeminiBillingMatrixRow{
				Surface:     surface,
				ServiceTier: tier,
				Slots:       make(map[string]GeminiBillingMatrixCell, len(geminiMatrixChargeSlots)),
			}
			for _, slot := range geminiMatrixChargeSlots {
				row.Slots[slot] = GeminiBillingMatrixCell{}
			}
			matrix.Rows = append(matrix.Rows, row)
		}
	}
	return matrix
}

func normalizeGeminiBillingMatrix(matrix *GeminiBillingMatrix) *GeminiBillingMatrix {
	normalized := newGeminiBillingMatrix()
	if matrix == nil {
		return normalized
	}
	for _, row := range matrix.Rows {
		surface := normalizeBillingSurface(row.Surface)
		tier := normalizeBillingDimension(row.ServiceTier, BillingServiceTierStandard)
		target := geminiMatrixRow(normalized, surface, tier)
		if target == nil {
			continue
		}
		for slot, cell := range row.Slots {
			slot = normalizeBillingDimension(slot, "")
			if !geminiChargeSlotSupported(slot) {
				continue
			}
			if cell.Price != nil {
				target.Slots[slot] = GeminiBillingMatrixCell{
					Price:      modelCatalogFloat64Ptr(*cell.Price),
					RuleID:     strings.TrimSpace(cell.RuleID),
					DerivedVia: strings.TrimSpace(cell.DerivedVia),
				}
				continue
			}
			target.Slots[slot] = GeminiBillingMatrixCell{
				RuleID:     strings.TrimSpace(cell.RuleID),
				DerivedVia: strings.TrimSpace(cell.DerivedVia),
			}
		}
	}
	return normalized
}

func geminiChargeSlotSupported(slot string) bool {
	_, ok := geminiMatrixSlotSpecs[slot]
	return ok
}

func geminiMatrixRow(matrix *GeminiBillingMatrix, surface string, tier string) *GeminiBillingMatrixRow {
	if matrix == nil {
		return nil
	}
	surface = normalizeBillingSurface(surface)
	tier = normalizeBillingDimension(tier, BillingServiceTierStandard)
	for index := range matrix.Rows {
		if matrix.Rows[index].Surface == surface && matrix.Rows[index].ServiceTier == tier {
			return &matrix.Rows[index]
		}
	}
	return nil
}

func geminiMatrixCell(matrix *GeminiBillingMatrix, surface string, tier string, slot string) *GeminiBillingMatrixCell {
	row := geminiMatrixRow(matrix, surface, tier)
	if row == nil || row.Slots == nil {
		return nil
	}
	cell, ok := row.Slots[slot]
	if !ok {
		return nil
	}
	copy := cell
	return &copy
}

func setGeminiMatrixCell(matrix *GeminiBillingMatrix, surface string, tier string, slot string, price *float64, ruleID string, derivedVia string, overwrite bool) {
	row := geminiMatrixRow(matrix, surface, tier)
	if row == nil || !geminiChargeSlotSupported(slot) {
		return
	}
	current := row.Slots[slot]
	if !overwrite && current.Price != nil {
		return
	}
	cell := GeminiBillingMatrixCell{
		RuleID:     strings.TrimSpace(ruleID),
		DerivedVia: strings.TrimSpace(derivedVia),
	}
	if price != nil {
		cell.Price = modelCatalogFloat64Ptr(*price)
	}
	row.Slots[slot] = cell
}

func editableBillingRules(rules []BillingRule) []BillingRule {
	filtered := make([]BillingRule, 0, len(rules))
	for _, rule := range rules {
		if isGeminiMatrixRule(rule) {
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered
}

func isGeminiMatrixRule(rule BillingRule) bool {
	return strings.HasPrefix(strings.TrimSpace(rule.ID), geminiMatrixRuleIDPrefix+"__")
}

func geminiMatrixRuleID(model string, layer string, surface string, tier string, slot string) string {
	return fmt.Sprintf(
		"%s__%s__%s__%s__%s__%s",
		geminiMatrixRuleIDPrefix,
		normalizeBillingDimension(layer, BillingLayerSale),
		CanonicalizeModelNameForPricing(model),
		normalizeBillingSurface(surface),
		normalizeBillingDimension(tier, BillingServiceTierStandard),
		normalizeBillingDimension(slot, ""),
	)
}

func geminiMatrixRuleMatches(rule BillingRule, model string, layer string) bool {
	if !isGeminiMatrixRule(rule) || rule.Provider != BillingRuleProviderGemini || rule.Layer != normalizeBillingDimension(layer, BillingLayerSale) {
		return false
	}
	return billingRuleMatchesModel(rule, model)
}

func canonicalGeminiMatrixRulesForModel(rules []BillingRule, model string, layer string) []BillingRule {
	filtered := make([]BillingRule, 0, len(rules))
	for _, rule := range rules {
		if geminiMatrixRuleMatches(rule, model, layer) {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

func replaceGeminiMatrixRules(rules []BillingRule, record *modelCatalogRecord, layer string, matrix *GeminiBillingMatrix) []BillingRule {
	model := ""
	if record != nil {
		model = record.model
	}
	filtered := make([]BillingRule, 0, len(rules))
	for _, rule := range rules {
		if geminiMatrixRuleMatches(rule, model, layer) {
			continue
		}
		filtered = append(filtered, rule)
	}
	filtered = append(filtered, buildGeminiMatrixRules(record, layer, matrix)...)
	sortBillingRules(filtered)
	return filtered
}

func deleteGeminiMatrixRules(rules []BillingRule, model string, layer string) ([]BillingRule, bool) {
	filtered := make([]BillingRule, 0, len(rules))
	removed := false
	for _, rule := range rules {
		if geminiMatrixRuleMatches(rule, model, layer) {
			removed = true
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered, removed
}

func buildGeminiMatrixRules(record *modelCatalogRecord, layer string, matrix *GeminiBillingMatrix) []BillingRule {
	normalized := normalizeGeminiBillingMatrix(matrix)
	model := ""
	if record != nil {
		model = record.model
	}
	modelKey := CanonicalizeModelNameForPricing(model)
	if modelKey == "" {
		return nil
	}
	models := modelCatalogRecordLookupCandidates(record)
	if len(models) == 0 {
		models = []string{modelKey}
	}
	rules := make([]BillingRule, 0, len(normalized.Rows)*len(geminiMatrixChargeSlots))
	for _, row := range normalized.Rows {
		for _, slot := range geminiMatrixChargeSlots {
			cell := row.Slots[slot]
			if cell.Price == nil {
				continue
			}
			spec, ok := geminiMatrixSlotSpecs[slot]
			if !ok {
				continue
			}
			rules = append(rules, BillingRule{
				ID:            geminiMatrixRuleID(modelKey, layer, row.Surface, row.ServiceTier, slot),
				Provider:      BillingRuleProviderGemini,
				Layer:         normalizeBillingDimension(layer, BillingLayerSale),
				Surface:       normalizeBillingSurface(row.Surface),
				OperationType: spec.operation,
				ServiceTier:   normalizeBillingDimension(row.ServiceTier, BillingServiceTierStandard),
				BatchMode:     BillingBatchModeAny,
				Matchers: BillingRuleMatchers{
					Models:         append([]string(nil), models...),
					InputModality:  spec.matchers.InputModality,
					OutputModality: spec.matchers.OutputModality,
					CachePhase:     spec.matchers.CachePhase,
					GroundingKind:  spec.matchers.GroundingKind,
					ContextWindow:  spec.matchers.ContextWindow,
				},
				Unit:     spec.unit,
				Price:    *cell.Price,
				Priority: geminiMatrixRulePriority,
				Enabled:  true,
			})
		}
	}
	sortBillingRules(rules)
	return rules
}

func buildGeminiMatrixForRecord(record *modelCatalogRecord, layer string, rules []BillingRule) *GeminiBillingMatrix {
	matrix := newGeminiBillingMatrix()
	if record == nil || !isGeminiBillingCompatModel(record.model) {
		return matrix
	}
	if canonical := canonicalGeminiMatrixRulesForModel(rules, record.model, layer); len(canonical) > 0 {
		applyGeminiRulesToMatrix(matrix, canonical, "canonical_rule", true)
		return matrix
	}
	compatRules := legacyGeminiCompatRulesForRecord(record, layer, rules)
	if len(compatRules) > 0 {
		applyGeminiRulesToMatrix(matrix, compatRules, "legacy_compat_rule", false)
		deriveGeminiMatrixLongContext(matrix, selectGeminiMatrixPricing(record, layer), record)
		deriveGeminiMatrixAudioAndStorage(matrix, selectGeminiMatrixPricing(record, layer))
		return matrix
	}
	applyPricingToGeminiMatrix(matrix, selectGeminiMatrixPricing(record, layer), record, "legacy_pricing")
	return matrix
}

func selectGeminiMatrixPricing(record *modelCatalogRecord, layer string) *ModelCatalogPricing {
	if record == nil {
		return nil
	}
	switch normalizeBillingDimension(layer, BillingLayerSale) {
	case BillingLayerOfficial:
		if record.officialPricing != nil {
			return cloneCatalogPricing(record.officialPricing)
		}
		return cloneCatalogPricing(record.upstreamPricing)
	default:
		if record.salePricing != nil {
			return cloneCatalogPricing(record.salePricing)
		}
		if record.officialPricing != nil {
			return cloneCatalogPricing(record.officialPricing)
		}
		return cloneCatalogPricing(record.upstreamPricing)
	}
}

func legacyGeminiCompatRulesForRecord(record *modelCatalogRecord, layer string, rules []BillingRule) []BillingRule {
	if record == nil {
		return nil
	}
	filtered := make([]BillingRule, 0, 16)
	for _, rule := range rules {
		if !isGeminiCompatRule(rule, layer) || !billingRuleMatchesModel(rule, record.model) {
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered
}

func applyGeminiRulesToMatrix(matrix *GeminiBillingMatrix, rules []BillingRule, derivedVia string, overwrite bool) {
	for _, rule := range rules {
		slot, ok := geminiMatrixSlotForRule(rule)
		if !ok {
			continue
		}
		for _, surface := range geminiMatrixTargetsForSurface(rule.Surface) {
			for _, tier := range geminiMatrixTargetsForTier(rule.ServiceTier) {
				price := rule.Price
				setGeminiMatrixCell(matrix, surface, tier, slot, &price, rule.ID, derivedVia, overwrite)
			}
		}
	}
}

func geminiMatrixTargetsForSurface(surface string) []string {
	surface = normalizeBillingSurface(surface)
	if !billingRuleUsesExplicitValue(surface) {
		return append([]string(nil), geminiMatrixSurfaces...)
	}
	for _, candidate := range geminiMatrixSurfaces {
		if candidate == surface {
			return []string{candidate}
		}
	}
	return nil
}

func geminiMatrixTargetsForTier(serviceTier string) []string {
	serviceTier = normalizeBillingDimension(serviceTier, BillingServiceTierStandard)
	if !billingRuleUsesExplicitValue(serviceTier) {
		return []string{BillingServiceTierStandard}
	}
	for _, candidate := range geminiMatrixServiceTiers {
		if candidate == serviceTier {
			return []string{candidate}
		}
	}
	return nil
}

func geminiMatrixSlotForRule(rule BillingRule) (string, bool) {
	switch rule.Unit {
	case BillingUnitInputToken:
		if normalizeBillingDimension(rule.Matchers.InputModality, "text") == "audio" {
			return BillingChargeSlotAudioInput, true
		}
		if normalizeBillingDimension(rule.Matchers.ContextWindow, BillingContextWindowStandard) == BillingContextWindowLong {
			return BillingChargeSlotTextInputLongContext, true
		}
		return BillingChargeSlotTextInput, true
	case BillingUnitOutputToken:
		switch normalizeBillingDimension(rule.Matchers.OutputModality, "text") {
		case "audio":
			return BillingChargeSlotAudioOutput, true
		case "text":
			if normalizeBillingDimension(rule.Matchers.ContextWindow, BillingContextWindowStandard) == BillingContextWindowLong {
				return BillingChargeSlotTextOutputLongContext, true
			}
			return BillingChargeSlotTextOutput, true
		default:
			if normalizeBillingDimension(rule.Matchers.ContextWindow, BillingContextWindowStandard) == BillingContextWindowLong {
				return BillingChargeSlotTextOutputLongContext, true
			}
			return BillingChargeSlotTextOutput, true
		}
	case BillingUnitCacheCreateToken:
		return BillingChargeSlotCacheCreate, true
	case BillingUnitCacheReadToken:
		return BillingChargeSlotCacheRead, true
	case BillingUnitCacheStorageTokenHour:
		return BillingChargeSlotCacheStorageTokenHour, true
	case BillingUnitImage:
		return BillingChargeSlotImageOutput, true
	case BillingUnitVideoRequest:
		return BillingChargeSlotVideoRequest, true
	case BillingUnitFileSearchEmbedding:
		return BillingChargeSlotFileSearchEmbeddingToken, true
	case BillingUnitFileSearchRetrieval:
		return BillingChargeSlotFileSearchRetrievalToken, true
	case BillingUnitGroundingSearchRequest:
		return BillingChargeSlotGroundingSearchRequest, true
	case BillingUnitGroundingMapsRequest:
		return BillingChargeSlotGroundingMapsRequest, true
	default:
		return "", false
	}
}

func applyPricingToGeminiMatrix(matrix *GeminiBillingMatrix, pricing *ModelCatalogPricing, record *modelCatalogRecord, derivedVia string) {
	if matrix == nil || pricing == nil {
		return
	}
	for _, surface := range geminiMatrixSurfaces {
		for _, tier := range geminiMatrixServiceTiers {
			assignGeminiMatrixPrice(matrix, surface, tier, BillingChargeSlotTextInput, pricingValueForTier(pricing.InputCostPerToken, pricing.InputCostPerTokenPriority, tier), derivedVia)
			assignGeminiMatrixPrice(matrix, surface, tier, BillingChargeSlotTextOutput, pricingValueForTier(pricing.OutputCostPerToken, pricing.OutputCostPerTokenPriority, tier), derivedVia)
			assignGeminiMatrixPrice(matrix, surface, tier, BillingChargeSlotAudioInput, pricingValueForTier(pricing.InputCostPerToken, pricing.InputCostPerTokenPriority, tier), derivedVia)
			assignGeminiMatrixPrice(matrix, surface, tier, BillingChargeSlotAudioOutput, pricingValueForTier(pricing.OutputCostPerToken, pricing.OutputCostPerTokenPriority, tier), derivedVia)
			assignGeminiMatrixPrice(matrix, surface, tier, BillingChargeSlotCacheCreate, pricingValueForTier(pricing.CacheCreationInputTokenCost, nil, tier), derivedVia)
			assignGeminiMatrixPrice(matrix, surface, tier, BillingChargeSlotCacheRead, pricingValueForTier(pricing.CacheReadInputTokenCost, pricing.CacheReadInputTokenCostPriority, tier), derivedVia)
			assignGeminiMatrixPrice(matrix, surface, tier, BillingChargeSlotImageOutput, pricing.OutputCostPerImage, derivedVia)
			assignGeminiMatrixPrice(matrix, surface, tier, BillingChargeSlotVideoRequest, pricing.OutputCostPerVideoRequest, derivedVia)
			if pricing.CacheCreationInputTokenCostAbove1hr != nil {
				assignGeminiMatrixPrice(matrix, surface, tier, BillingChargeSlotCacheStorageTokenHour, pricing.CacheCreationInputTokenCostAbove1hr, derivedVia)
			}
		}
	}
	deriveGeminiMatrixLongContext(matrix, pricing, record)
}

func deriveGeminiMatrixLongContext(matrix *GeminiBillingMatrix, pricing *ModelCatalogPricing, record *modelCatalogRecord) {
	if matrix == nil || pricing == nil || record == nil || record.longContextInputTokenThreshold <= 0 {
		return
	}
	for _, surface := range geminiMatrixSurfaces {
		for _, tier := range geminiMatrixServiceTiers {
			assignGeminiMatrixPrice(
				matrix,
				surface,
				tier,
				BillingChargeSlotTextInputLongContext,
				longContextPricingValue(
					pricing.InputCostPerToken,
					pricing.InputCostPerTokenPriority,
					pricing.InputCostPerTokenAboveThreshold,
					pricing.InputCostPerTokenPriorityAboveThreshold,
					record.longContextInputCostMultiplier,
					tier,
				),
				"long_context_seed",
			)
			assignGeminiMatrixPrice(
				matrix,
				surface,
				tier,
				BillingChargeSlotTextOutputLongContext,
				longContextPricingValue(
					pricing.OutputCostPerToken,
					pricing.OutputCostPerTokenPriority,
					pricing.OutputCostPerTokenAboveThreshold,
					pricing.OutputCostPerTokenPriorityAboveThreshold,
					record.longContextOutputCostMultiplier,
					tier,
				),
				"long_context_seed",
			)
		}
	}
}

func deriveGeminiMatrixAudioAndStorage(matrix *GeminiBillingMatrix, pricing *ModelCatalogPricing) {
	if matrix == nil {
		return
	}
	for _, surface := range geminiMatrixSurfaces {
		for _, tier := range geminiMatrixServiceTiers {
			if current := geminiMatrixCell(matrix, surface, tier, BillingChargeSlotAudioInput); current == nil || current.Price == nil {
				if fallback := geminiMatrixCell(matrix, surface, tier, BillingChargeSlotTextInput); fallback != nil && fallback.Price != nil {
					value := *fallback.Price
					setGeminiMatrixCell(matrix, surface, tier, BillingChargeSlotAudioInput, &value, "", "audio_from_text", false)
				}
			}
			if current := geminiMatrixCell(matrix, surface, tier, BillingChargeSlotAudioOutput); current == nil || current.Price == nil {
				if fallback := geminiMatrixCell(matrix, surface, tier, BillingChargeSlotTextOutput); fallback != nil && fallback.Price != nil {
					value := *fallback.Price
					setGeminiMatrixCell(matrix, surface, tier, BillingChargeSlotAudioOutput, &value, "", "audio_from_text", false)
				}
			}
			if pricing != nil && pricing.CacheCreationInputTokenCostAbove1hr != nil {
				setGeminiMatrixCell(matrix, surface, tier, BillingChargeSlotCacheStorageTokenHour, pricing.CacheCreationInputTokenCostAbove1hr, "", "legacy_cache_storage", false)
			}
		}
	}
}

func assignGeminiMatrixPrice(matrix *GeminiBillingMatrix, surface string, tier string, slot string, price *float64, derivedVia string) {
	if price == nil {
		return
	}
	setGeminiMatrixCell(matrix, surface, tier, slot, price, "", derivedVia, false)
}

func pricingValueForTier(base *float64, explicitPriority *float64, tier string) *float64 {
	if base == nil {
		if normalizeBillingDimension(tier, BillingServiceTierStandard) == BillingServiceTierPriority && explicitPriority != nil {
			return modelCatalogFloat64Ptr(*explicitPriority)
		}
		return nil
	}
	switch normalizeBillingDimension(tier, BillingServiceTierStandard) {
	case BillingServiceTierPriority:
		if explicitPriority != nil {
			return modelCatalogFloat64Ptr(*explicitPriority)
		}
		return nil
	case BillingServiceTierFlex:
		return modelCatalogFloat64Ptr(*base * serviceTierCostMultiplier(BillingServiceTierFlex))
	default:
		return modelCatalogFloat64Ptr(*base)
	}
}

func longContextPricingValue(base *float64, explicitPriority *float64, above *float64, priorityAbove *float64, multiplier float64, tier string) *float64 {
	switch normalizeBillingDimension(tier, BillingServiceTierStandard) {
	case BillingServiceTierPriority:
		if priorityAbove != nil {
			return modelCatalogFloat64Ptr(*priorityAbove)
		}
		if explicitPriority != nil && multiplier > 0 {
			return modelCatalogFloat64Ptr(*explicitPriority * multiplier)
		}
	case BillingServiceTierFlex:
		if above != nil {
			return modelCatalogFloat64Ptr(*above * serviceTierCostMultiplier(BillingServiceTierFlex))
		}
		if base != nil && multiplier > 0 {
			return modelCatalogFloat64Ptr(*base * multiplier * serviceTierCostMultiplier(BillingServiceTierFlex))
		}
	default:
		if above != nil {
			return modelCatalogFloat64Ptr(*above)
		}
		if base != nil && multiplier > 0 {
			return modelCatalogFloat64Ptr(*base * multiplier)
		}
	}
	return nil
}
