package service

import (
	"fmt"
	"strings"
)

const (
	geminiCompatRuleIDPrefix  = "compat_gemini"
	geminiCompatRulePriority  = 8000
	geminiCompatRuleTierBonus = 10
)

func isGeminiBillingCompatModel(model string) bool {
	return strings.HasPrefix(CanonicalizeModelNameForPricing(model), "gemini-")
}

func splitGeminiCompatPricing(pricing ModelCatalogPricing) (ModelCatalogPricing, ModelCatalogPricing) {
	ruleBacked := ModelCatalogPricing{
		InputCostPerToken:               pricing.InputCostPerToken,
		InputCostPerTokenPriority:       pricing.InputCostPerTokenPriority,
		OutputCostPerToken:              pricing.OutputCostPerToken,
		OutputCostPerTokenPriority:      pricing.OutputCostPerTokenPriority,
		CacheCreationInputTokenCost:     pricing.CacheCreationInputTokenCost,
		CacheReadInputTokenCost:         pricing.CacheReadInputTokenCost,
		CacheReadInputTokenCostPriority: pricing.CacheReadInputTokenCostPriority,
		OutputCostPerImage:              pricing.OutputCostPerImage,
		OutputCostPerVideoRequest:       pricing.OutputCostPerVideoRequest,
	}
	legacyOnly := ModelCatalogPricing{
		InputTokenThreshold:                      pricing.InputTokenThreshold,
		InputCostPerTokenAboveThreshold:          pricing.InputCostPerTokenAboveThreshold,
		InputCostPerTokenPriorityAboveThreshold:  pricing.InputCostPerTokenPriorityAboveThreshold,
		OutputTokenThreshold:                     pricing.OutputTokenThreshold,
		OutputCostPerTokenAboveThreshold:         pricing.OutputCostPerTokenAboveThreshold,
		OutputCostPerTokenPriorityAboveThreshold: pricing.OutputCostPerTokenPriorityAboveThreshold,
		CacheCreationInputTokenCostAbove1hr:      pricing.CacheCreationInputTokenCostAbove1hr,
	}
	return ruleBacked, legacyOnly
}

func mergeModelPricingOverrides(base *ModelPricingOverride, patch *ModelPricingOverride) *ModelPricingOverride {
	if base == nil && patch == nil {
		return nil
	}
	merged := &ModelPricingOverride{}
	if base != nil {
		*merged = *cloneModelPricingOverride(base)
	}
	if patch != nil {
		if merged.UpdatedAt.IsZero() && !patch.UpdatedAt.IsZero() {
			merged.UpdatedAt = patch.UpdatedAt
			merged.UpdatedByUserID = patch.UpdatedByUserID
			merged.UpdatedByEmail = patch.UpdatedByEmail
		}
		mergeCatalogPricing(&merged.ModelCatalogPricing, &patch.ModelCatalogPricing)
	}
	if pricingEmpty(&merged.ModelCatalogPricing) {
		return nil
	}
	return merged
}

func buildGeminiCompatPricingOverride(model string, layer string, rules []BillingRule) *ModelPricingOverride {
	if !isGeminiBillingCompatModel(model) {
		return nil
	}
	pricing := ModelCatalogPricing{}
	for _, rule := range rules {
		if !isGeminiCompatRule(rule, layer) || !billingRuleMatchesModel(rule, model) {
			continue
		}
		switch geminiCompatRuleSlot(rule.ID) {
		case "input":
			pricing.InputCostPerToken = modelCatalogFloat64Ptr(rule.Price)
		case "input_priority":
			pricing.InputCostPerTokenPriority = modelCatalogFloat64Ptr(rule.Price)
		case "output":
			pricing.OutputCostPerToken = modelCatalogFloat64Ptr(rule.Price)
		case "output_priority":
			pricing.OutputCostPerTokenPriority = modelCatalogFloat64Ptr(rule.Price)
		case "cache_create":
			pricing.CacheCreationInputTokenCost = modelCatalogFloat64Ptr(rule.Price)
		case "cache_read":
			pricing.CacheReadInputTokenCost = modelCatalogFloat64Ptr(rule.Price)
		case "cache_read_priority":
			pricing.CacheReadInputTokenCostPriority = modelCatalogFloat64Ptr(rule.Price)
		case "image":
			pricing.OutputCostPerImage = modelCatalogFloat64Ptr(rule.Price)
		case "video":
			pricing.OutputCostPerVideoRequest = modelCatalogFloat64Ptr(rule.Price)
		}
	}
	if pricingEmpty(&pricing) {
		return nil
	}
	return &ModelPricingOverride{ModelCatalogPricing: pricing}
}

func buildGeminiCompatPricingFromMatrixRules(model string, layer string, rules []BillingRule) (ModelCatalogPricing, bool) {
	canonical := canonicalGeminiMatrixRulesForModel(rules, model, layer)
	if len(canonical) == 0 {
		return ModelCatalogPricing{}, false
	}
	matrix := newGeminiBillingMatrix()
	applyGeminiRulesToMatrix(matrix, canonical, "canonical_rule", true)
	pricing := ModelCatalogPricing{
		InputCostPerToken:           geminiCompatMatrixPrice(matrix, BillingChargeSlotTextInput, BillingServiceTierStandard),
		InputCostPerTokenPriority:   geminiCompatMatrixPrice(matrix, BillingChargeSlotTextInput, BillingServiceTierPriority),
		OutputCostPerToken:          geminiCompatMatrixPrice(matrix, BillingChargeSlotTextOutput, BillingServiceTierStandard),
		OutputCostPerTokenPriority:  geminiCompatMatrixPrice(matrix, BillingChargeSlotTextOutput, BillingServiceTierPriority),
		CacheCreationInputTokenCost: geminiCompatMatrixPrice(matrix, BillingChargeSlotCacheCreate, BillingServiceTierStandard),
		CacheReadInputTokenCost:     geminiCompatMatrixPrice(matrix, BillingChargeSlotCacheRead, BillingServiceTierStandard),
		CacheReadInputTokenCostPriority: geminiCompatMatrixPrice(
			matrix,
			BillingChargeSlotCacheRead,
			BillingServiceTierPriority,
		),
		OutputCostPerImage:        geminiCompatMatrixPrice(matrix, BillingChargeSlotImageOutput, BillingServiceTierStandard),
		OutputCostPerVideoRequest: geminiCompatMatrixPrice(matrix, BillingChargeSlotVideoRequest, BillingServiceTierStandard),
	}
	if pricingEmpty(&pricing) {
		return ModelCatalogPricing{}, false
	}
	return pricing, true
}

func geminiCompatMatrixPrice(matrix *GeminiBillingMatrix, slot string, tier string) *float64 {
	if matrix == nil {
		return nil
	}
	for _, surface := range geminiMatrixSurfaces {
		cell := geminiMatrixCell(matrix, surface, tier, slot)
		if cell == nil || cell.Price == nil {
			continue
		}
		return modelCatalogFloat64Ptr(*cell.Price)
	}
	return nil
}

func replaceGeminiCompatRules(rules []BillingRule, record *modelCatalogRecord, layer string, pricing ModelCatalogPricing) []BillingRule {
	filtered := make([]BillingRule, 0, len(rules))
	for _, rule := range rules {
		if isGeminiCompatRule(rule, layer) && record != nil && billingRuleMatchesModel(rule, record.model) {
			continue
		}
		filtered = append(filtered, rule)
	}
	filtered = append(filtered, buildGeminiCompatRules(record, layer, pricing)...)
	return filtered
}

func deleteGeminiCompatRules(rules []BillingRule, record *modelCatalogRecord, layer string) ([]BillingRule, bool) {
	filtered := make([]BillingRule, 0, len(rules))
	removed := false
	for _, rule := range rules {
		if isGeminiCompatRule(rule, layer) && record != nil && billingRuleMatchesModel(rule, record.model) {
			removed = true
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered, removed
}

func buildGeminiCompatRules(record *modelCatalogRecord, layer string, pricing ModelCatalogPricing) []BillingRule {
	if record == nil || !isGeminiBillingCompatModel(record.model) {
		return nil
	}
	ruleBacked, _ := splitGeminiCompatPricing(pricing)
	if pricingEmpty(&ruleBacked) {
		return nil
	}
	models := geminiCompatModelMatchers(record)
	anchor := CanonicalizeModelNameForPricing(record.model)
	usesPriorityPricing := geminiCompatUsesExplicitPriority(pricing)
	rules := make([]BillingRule, 0, 16)
	appendRule := func(slot string, unit string, serviceTier string, price float64) {
		rules = append(rules, BillingRule{
			ID:            geminiCompatRuleID(anchor, layer, slot),
			Provider:      BillingRuleProviderGemini,
			Layer:         layer,
			Surface:       BillingSurfaceAny,
			OperationType: "",
			ServiceTier:   serviceTier,
			BatchMode:     BillingBatchModeAny,
			Matchers:      BillingRuleMatchers{Models: models},
			Unit:          unit,
			Price:         price,
			Priority:      geminiCompatRulePriority + geminiCompatPriorityOffset(serviceTier),
			Enabled:       true,
		})
	}
	appendTieredTokenRules := func(base *float64, explicit *float64, unit string, baseSlot string, prioritySlot string) {
		if base == nil {
			return
		}
		appendRule(baseSlot, unit, "", *base)
		appendRule(baseSlot+"_flex", unit, "flex", *base*serviceTierCostMultiplier("flex"))
		priorityPrice := *base * serviceTierCostMultiplier("priority")
		priorityID := prioritySlot + "_derived"
		if usesPriorityPricing {
			priorityPrice = *base
			if explicit != nil && *explicit > 0 {
				priorityPrice = *explicit
				priorityID = prioritySlot
			}
		}
		appendRule(priorityID, unit, "priority", priorityPrice)
	}
	appendTieredTokenRules(ruleBacked.InputCostPerToken, ruleBacked.InputCostPerTokenPriority, BillingUnitInputToken, "input", "input_priority")
	appendTieredTokenRules(ruleBacked.OutputCostPerToken, ruleBacked.OutputCostPerTokenPriority, BillingUnitOutputToken, "output", "output_priority")
	appendTieredTokenRules(ruleBacked.CacheCreationInputTokenCost, nil, BillingUnitCacheCreateToken, "cache_create", "cache_create_priority")
	appendTieredTokenRules(ruleBacked.CacheReadInputTokenCost, ruleBacked.CacheReadInputTokenCostPriority, BillingUnitCacheReadToken, "cache_read", "cache_read_priority")
	if ruleBacked.OutputCostPerImage != nil {
		appendRule("image", BillingUnitImage, "", *ruleBacked.OutputCostPerImage)
	}
	if ruleBacked.OutputCostPerVideoRequest != nil {
		appendRule("video", BillingUnitVideoRequest, "", *ruleBacked.OutputCostPerVideoRequest)
	}
	return rules
}

func geminiCompatUsesExplicitPriority(pricing ModelCatalogPricing) bool {
	for _, value := range []*float64{
		pricing.InputCostPerTokenPriority,
		pricing.OutputCostPerTokenPriority,
		pricing.CacheReadInputTokenCostPriority,
	} {
		if value != nil && *value > 0 {
			return true
		}
	}
	return false
}

func geminiCompatModelMatchers(record *modelCatalogRecord) []string {
	seen := map[string]struct{}{}
	models := make([]string, 0, 4)
	for _, candidate := range modelCatalogRecordLookupCandidates(record) {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		models = append(models, candidate)
	}
	return models
}

func geminiCompatRuleID(anchor string, layer string, slot string) string {
	return fmt.Sprintf("%s__%s__%s__%s", geminiCompatRuleIDPrefix, strings.TrimSpace(layer), strings.TrimSpace(anchor), strings.TrimSpace(slot))
}

func geminiCompatRuleSlot(id string) string {
	parts := strings.Split(strings.TrimSpace(id), "__")
	if len(parts) < 4 || parts[0] != geminiCompatRuleIDPrefix {
		return ""
	}
	return parts[len(parts)-1]
}

func isGeminiCompatRule(rule BillingRule, layer string) bool {
	return rule.Provider == BillingRuleProviderGemini &&
		rule.Layer == strings.TrimSpace(strings.ToLower(layer)) &&
		strings.HasPrefix(strings.TrimSpace(rule.ID), geminiCompatRuleIDPrefix+"__")
}

func geminiCompatPriorityOffset(serviceTier string) int {
	if strings.TrimSpace(strings.ToLower(serviceTier)) == "" {
		return 0
	}
	return geminiCompatRuleTierBonus
}
