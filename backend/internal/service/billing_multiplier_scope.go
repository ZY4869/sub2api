package service

func normalizeExplicitRateMultiplier(value float64) float64 {
	if value <= 0 {
		return 1
	}
	return value
}

func cloneRateMultiplier(value float64) *float64 {
	v := value
	return &v
}

func billingRuntimeFlatMultiplier(rate float64, flat *float64) float64 {
	if flat == nil {
		return normalizeExplicitRateMultiplier(rate)
	}
	return normalizeExplicitRateMultiplier(*flat)
}

func billingLineActualMultiplier(unit string, tokenMultiplier, flatMultiplier float64) float64 {
	if billingUnitUsesTokenMultiplier(unit) {
		return normalizeExplicitRateMultiplier(tokenMultiplier)
	}
	return normalizeExplicitRateMultiplier(flatMultiplier)
}

func billingUnitUsesTokenMultiplier(unit string) bool {
	switch unit {
	case BillingUnitInputToken,
		BillingUnitOutputToken,
		BillingUnitCacheCreateToken,
		BillingUnitCacheReadToken,
		BillingUnitCacheStorageTokenHour,
		BillingUnitFileSearchEmbedding,
		BillingUnitFileSearchRetrieval:
		return true
	default:
		return false
	}
}

func applyFlatMultiplierToNonTokenSimulationLines(result *BillingSimulationResult, flatMultiplier float64) {
	if result == nil {
		return
	}
	flatMultiplier = normalizeExplicitRateMultiplier(flatMultiplier)
	actualCost := 0.0
	for i := range result.Lines {
		if !billingUnitUsesTokenMultiplier(result.Lines[i].Unit) {
			result.Lines[i].ActualCost = result.Lines[i].Cost * flatMultiplier
		}
		actualCost += result.Lines[i].ActualCost
	}
	result.ActualCost = actualCost
}
