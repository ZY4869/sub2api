package service

import "strings"

func applyModelPricingCurrencyMetadata(pricing *ModelPricing, meta ModelPricingCurrencyMetadata) {
	if pricing == nil {
		return
	}
	pricing.Currency = defaultModelPricingCurrency(meta.Currency)
	if meta.USDToCNYRate != nil {
		pricing.USDToCNYRate = *meta.USDToCNYRate
	}
	pricing.FXRateDate = strings.TrimSpace(meta.FXRateDate)
	pricing.FXLockedAt = cloneBillingTime(meta.FXLockedAt)
}

func finalizeCostBreakdownCurrency(breakdown *CostBreakdown, pricing *ModelPricing) *CostBreakdown {
	if breakdown == nil {
		return &CostBreakdown{Currency: ModelPricingCurrencyUSD}
	}
	currency := ModelPricingCurrencyUSD
	if pricing != nil {
		currency = defaultModelPricingCurrency(pricing.Currency)
		breakdown.USDToCNYRate = pricing.USDToCNYRate
		breakdown.FXRateDate = pricing.FXRateDate
		breakdown.FXLockedAt = cloneBillingTime(pricing.FXLockedAt)
	}
	breakdown.Currency = currency
	breakdown.InputCost = NormalizeBillingAmount(breakdown.InputCost)
	breakdown.OutputCost = NormalizeBillingAmount(breakdown.OutputCost)
	breakdown.CacheCreationCost = NormalizeBillingAmount(breakdown.CacheCreationCost)
	breakdown.CacheReadCost = NormalizeBillingAmount(breakdown.CacheReadCost)
	breakdown.TotalCost = NormalizeBillingAmount(breakdown.TotalCost)
	breakdown.ActualCost = NormalizeBillingAmount(breakdown.ActualCost)
	breakdown.TotalCostUSDEquivalent = costUSDEquivalent(breakdown.TotalCost, currency, breakdown.USDToCNYRate)
	breakdown.ActualCostUSDEquivalent = costUSDEquivalent(breakdown.ActualCost, currency, breakdown.USDToCNYRate)
	breakdown.CostByCurrency = normalizedBillingCostMap(currency, breakdown.TotalCost)
	breakdown.ActualCostByCurrency = normalizedBillingCostMap(currency, breakdown.ActualCost)
	return breakdown
}

func mergeCostBreakdownsCurrency(total *CostBreakdown, parts ...*CostBreakdown) *CostBreakdown {
	if total == nil {
		total = &CostBreakdown{}
	}
	for _, part := range parts {
		if part == nil {
			continue
		}
		if total.Currency == "" {
			total.Currency = part.Currency
			total.USDToCNYRate = part.USDToCNYRate
			total.FXRateDate = part.FXRateDate
			total.FXLockedAt = cloneBillingTime(part.FXLockedAt)
		}
		total.TotalCostUSDEquivalent += part.TotalCostUSDEquivalent
		total.ActualCostUSDEquivalent += part.ActualCostUSDEquivalent
		if len(part.CostByCurrency) > 0 {
			if total.CostByCurrency == nil {
				total.CostByCurrency = map[string]float64{}
			}
			for currency, amount := range part.CostByCurrency {
				total.CostByCurrency[currency] += amount
			}
		}
		if len(part.ActualCostByCurrency) > 0 {
			if total.ActualCostByCurrency == nil {
				total.ActualCostByCurrency = map[string]float64{}
			}
			for currency, amount := range part.ActualCostByCurrency {
				total.ActualCostByCurrency[currency] += amount
			}
		}
	}
	if total.Currency == "" {
		total.Currency = ModelPricingCurrencyUSD
	}
	total.TotalCostUSDEquivalent = NormalizeBillingAmount(total.TotalCostUSDEquivalent)
	total.ActualCostUSDEquivalent = NormalizeBillingAmount(total.ActualCostUSDEquivalent)
	for currency, amount := range total.CostByCurrency {
		total.CostByCurrency[currency] = NormalizeBillingAmount(amount)
	}
	for currency, amount := range total.ActualCostByCurrency {
		total.ActualCostByCurrency[currency] = NormalizeBillingAmount(amount)
	}
	return total
}
