package service

import "context"

// CalculateCost computes billing using the model pricing table.
func (s *BillingService) CalculateCost(model string, tokens UsageTokens, rateMultiplier float64) (*CostBreakdown, error) {
	return s.CalculateCostWithServiceTierWithContext(context.Background(), model, tokens, rateMultiplier, "")
}

func (s *BillingService) CalculateCostWithServiceTier(model string, tokens UsageTokens, rateMultiplier float64, serviceTier string) (*CostBreakdown, error) {
	return s.CalculateCostWithServiceTierWithContext(context.Background(), model, tokens, rateMultiplier, serviceTier)
}

func (s *BillingService) CalculateCostWithServiceTierWithContext(
	ctx context.Context,
	model string,
	tokens UsageTokens,
	rateMultiplier float64,
	serviceTier string,
) (*CostBreakdown, error) {
	pricing, err := s.getPricingForBillingWithContext(ctx, model)
	if err != nil {
		return nil, err
	}
	return s.calculateCostWithPricing(pricing, tokens, tokens, rateMultiplier, serviceTier), nil
}

func (s *BillingService) calculateCostWithPricing(
	pricing *ModelPricing,
	billedTokens UsageTokens,
	thresholdTokens UsageTokens,
	rateMultiplier float64,
	serviceTier string,
) *CostBreakdown {
	breakdown := &CostBreakdown{}
	inputPricePerToken := pricing.InputPricePerToken
	outputPricePerToken := pricing.OutputPricePerToken
	cacheReadPricePerToken := pricing.CacheReadPricePerToken
	tierMultiplier := 1.0
	usingPriorityPricing := usePriorityServiceTierPricing(serviceTier, pricing)
	if usingPriorityPricing {
		if pricing.InputPricePerTokenPriority > 0 {
			inputPricePerToken = pricing.InputPricePerTokenPriority
		}
		if pricing.OutputPricePerTokenPriority > 0 {
			outputPricePerToken = pricing.OutputPricePerTokenPriority
		}
		if pricing.CacheReadPricePerTokenPriority > 0 {
			cacheReadPricePerToken = pricing.CacheReadPricePerTokenPriority
		}
	} else {
		tierMultiplier = serviceTierCostMultiplier(serviceTier)
	}

	if usingPriorityPricing {
		inputPricePerToken = resolveTieredTokenPrice(thresholdTokens.InputTokens, inputPricePerToken, pricing.InputTokenThreshold, pricing.InputPricePerTokenPriorityAboveThreshold)
		outputPricePerToken = resolveTieredTokenPrice(thresholdTokens.OutputTokens, outputPricePerToken, pricing.OutputTokenThreshold, pricing.OutputPricePerTokenPriorityAboveThreshold)
	} else {
		inputPricePerToken = resolveTieredTokenPrice(thresholdTokens.InputTokens, inputPricePerToken, pricing.InputTokenThreshold, pricing.InputPricePerTokenAboveThreshold)
		outputPricePerToken = resolveTieredTokenPrice(thresholdTokens.OutputTokens, outputPricePerToken, pricing.OutputTokenThreshold, pricing.OutputPricePerTokenAboveThreshold)
	}

	applyLongContext := s.shouldApplySessionLongContextPricing(thresholdTokens, pricing)
	if applyLongContext {
		inputPricePerToken *= pricing.LongContextInputMultiplier
		outputPricePerToken *= pricing.LongContextOutputMultiplier
		cacheReadPricePerToken *= pricing.LongContextInputMultiplier
	}

	breakdown.InputCost = float64(billedTokens.InputTokens) * inputPricePerToken
	breakdown.OutputCost = float64(billedTokens.OutputTokens) * outputPricePerToken

	if pricing.SupportsCacheBreakdown && (pricing.CacheCreation5mPrice > 0 || pricing.CacheCreation1hPrice > 0) {
		if billedTokens.CacheCreation5mTokens == 0 && billedTokens.CacheCreation1hTokens == 0 && billedTokens.CacheCreationTokens > 0 {
			cacheCreationPrice := pricing.CacheCreation5mPrice
			if applyLongContext {
				cacheCreationPrice *= pricing.LongContextInputMultiplier
			}
			breakdown.CacheCreationCost = float64(billedTokens.CacheCreationTokens) * cacheCreationPrice
		} else {
			cacheCreation5mPrice := pricing.CacheCreation5mPrice
			cacheCreation1hPrice := pricing.CacheCreation1hPrice
			if applyLongContext {
				cacheCreation5mPrice *= pricing.LongContextInputMultiplier
				cacheCreation1hPrice *= pricing.LongContextInputMultiplier
			}
			breakdown.CacheCreationCost = float64(billedTokens.CacheCreation5mTokens)*cacheCreation5mPrice +
				float64(billedTokens.CacheCreation1hTokens)*cacheCreation1hPrice
		}
	} else {
		cacheCreationPrice := pricing.CacheCreationPricePerToken
		if applyLongContext {
			cacheCreationPrice *= pricing.LongContextInputMultiplier
		}
		breakdown.CacheCreationCost = float64(billedTokens.CacheCreationTokens) * cacheCreationPrice
	}

	breakdown.CacheReadCost = float64(billedTokens.CacheReadTokens) * cacheReadPricePerToken

	if tierMultiplier != 1.0 {
		breakdown.InputCost *= tierMultiplier
		breakdown.OutputCost *= tierMultiplier
		breakdown.CacheCreationCost *= tierMultiplier
		breakdown.CacheReadCost *= tierMultiplier
	}

	breakdown.TotalCost = breakdown.InputCost + breakdown.OutputCost +
		breakdown.CacheCreationCost + breakdown.CacheReadCost
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}
	breakdown.ActualCost = breakdown.TotalCost * rateMultiplier
	return finalizeCostBreakdownCurrency(breakdown, pricing)
}
