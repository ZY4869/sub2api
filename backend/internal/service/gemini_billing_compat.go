package service

import "strings"

const (
	geminiCompatRuleIDPrefix = "compat_gemini"
)

func isGeminiBillingCompatModel(model string) bool {
	return strings.HasPrefix(CanonicalizeModelNameForPricing(model), "gemini-")
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
