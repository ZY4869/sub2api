package service

import (
	"context"
	"strings"
)

func (s *BillingService) applyModelSpecificPricingPolicy(model string, pricing *ModelPricing) *ModelPricing {
	if pricing == nil {
		return nil
	}
	if !isOpenAIGPT54Model(model) {
		return pricing
	}
	if pricing.LongContextInputThreshold > 0 && pricing.LongContextInputMultiplier > 0 && pricing.LongContextOutputMultiplier > 0 {
		return pricing
	}
	cloned := *pricing
	if cloned.LongContextInputThreshold <= 0 {
		cloned.LongContextInputThreshold = openAIGPT54LongContextInputThreshold
	}
	if cloned.LongContextInputMultiplier <= 0 {
		cloned.LongContextInputMultiplier = openAIGPT54LongContextInputMultiplier
	}
	if cloned.LongContextOutputMultiplier <= 0 {
		cloned.LongContextOutputMultiplier = openAIGPT54LongContextOutputMultiplier
	}
	return &cloned
}

func (s *BillingService) shouldApplySessionLongContextPricing(tokens UsageTokens, pricing *ModelPricing) bool {
	if pricing == nil || pricing.LongContextInputThreshold <= 0 {
		return false
	}
	if pricing.LongContextInputMultiplier <= 1 && pricing.LongContextOutputMultiplier <= 1 {
		return false
	}
	return longContextInputTokenTotal(tokens) > pricing.LongContextInputThreshold
}

func isOpenAIGPT54Model(model string) bool {
	normalized := normalizeCodexModel(strings.TrimSpace(strings.ToLower(model)))
	base := modelDateVersionSuffixPattern.ReplaceAllString(normalized, "")
	switch base {
	case "gpt-5.4", "gpt-5.4-pro":
		return true
	default:
		return false
	}
}

// CalculateCostWithLongContext calculates cost with a long-context overflow multiplier.
// threshold: threshold value, for example 200000.
// extraMultiplier: multiplier applied to the overflow portion, for example 2.0.
//
// Example:
// cache-read 210k + input 10k = 220k, threshold 200k, multiplier 2.0.
// Split into in-range (200k, 0) and overflow (10k, 10k).
// The in-range portion uses normal pricing and the overflow portion uses the extra multiplier.
func (s *BillingService) CalculateCostWithLongContext(model string, tokens UsageTokens, rateMultiplier float64, threshold int, extraMultiplier float64) (*CostBreakdown, error) {
	return s.CalculateCostWithLongContextWithContext(context.Background(), model, tokens, rateMultiplier, threshold, extraMultiplier)
}

func (s *BillingService) CalculateCostWithLongContextWithContext(
	ctx context.Context,
	model string,
	tokens UsageTokens,
	rateMultiplier float64,
	threshold int,
	extraMultiplier float64,
) (*CostBreakdown, error) {
	// Invalid long-context configuration falls back to normal pricing.
	if threshold <= 0 || extraMultiplier <= 1 {
		return s.CalculateCostWithServiceTierWithContext(ctx, model, tokens, rateMultiplier, "")
	}

	pricing, err := s.getPricingForBillingWithContext(ctx, model)
	if err != nil {
		return nil, err
	}

	// Determine whether input + cache tokens cross the threshold.
	total := longContextInputTokenTotal(tokens)
	if total <= threshold {
		return s.calculateCostWithPricing(pricing, tokens, tokens, rateMultiplier, ""), nil
	}

	// Split tokens into in-range and overflow ranges.
	inRangeTokens, outRangeTokens := splitLongContextInputTokens(tokens, threshold)

	// Price the in-range portion normally.
	inRangeTokens.OutputTokens = tokens.OutputTokens // output tokens stay in the base tier
	inRangeCost := s.calculateCostWithPricing(pricing, inRangeTokens, tokens, rateMultiplier, "")

	// Price the overflow portion with the extra multiplier.
	outRangeCost := s.calculateCostWithPricing(pricing, outRangeTokens, tokens, rateMultiplier*extraMultiplier, "")

	// Merge the two partial cost breakdowns.
	merged := &CostBreakdown{
		InputCost:         inRangeCost.InputCost + outRangeCost.InputCost,
		OutputCost:        inRangeCost.OutputCost,
		CacheCreationCost: inRangeCost.CacheCreationCost + outRangeCost.CacheCreationCost,
		CacheReadCost:     inRangeCost.CacheReadCost + outRangeCost.CacheReadCost,
		TotalCost:         inRangeCost.TotalCost + outRangeCost.TotalCost,
		ActualCost:        inRangeCost.ActualCost + outRangeCost.ActualCost,
	}
	return mergeCostBreakdownsCurrency(merged, inRangeCost, outRangeCost), nil
}

func longContextInputTokenTotal(tokens UsageTokens) int {
	return tokens.CacheReadTokens +
		tokens.CacheCreationTokens +
		tokens.CacheCreation5mTokens +
		tokens.CacheCreation1hTokens +
		tokens.InputTokens
}

func splitLongContextInputTokens(tokens UsageTokens, threshold int) (UsageTokens, UsageTokens) {
	var inRange UsageTokens
	var overflow UsageTokens
	remaining := threshold
	take := func(value int) (int, int) {
		if value <= 0 {
			return 0, 0
		}
		if remaining <= 0 {
			return 0, value
		}
		if value <= remaining {
			remaining -= value
			return value, 0
		}
		in := remaining
		remaining = 0
		return in, value - in
	}

	inRange.CacheReadTokens, overflow.CacheReadTokens = take(tokens.CacheReadTokens)
	inRange.CacheCreationTokens, overflow.CacheCreationTokens = take(tokens.CacheCreationTokens)
	inRange.CacheCreation5mTokens, overflow.CacheCreation5mTokens = take(tokens.CacheCreation5mTokens)
	inRange.CacheCreation1hTokens, overflow.CacheCreation1hTokens = take(tokens.CacheCreation1hTokens)
	inRange.InputTokens, overflow.InputTokens = take(tokens.InputTokens)
	return inRange, overflow
}
