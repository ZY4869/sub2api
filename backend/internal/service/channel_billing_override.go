package service

import "github.com/Wei-Shaw/sub2api/internal/model"

func applyChannelPricingOverride(
	base *CostBreakdown,
	pricing *GatewayChannelResolvedPricing,
	tokens UsageTokens,
	multiplier float64,
	imageCount int,
) (*CostBreakdown, *int, *float64) {
	if pricing == nil {
		return base, nil, nil
	}

	if base == nil {
		base = &CostBreakdown{}
	}

	switch pricing.BillingMode {
	case model.ChannelBillingModePerRequest:
		amount := nullableFloatValue(pricing.PerRequestPrice)
		if amount <= 0 {
			return base, nil, nil
		}
		return newChannelFlatCost(amount, multiplier), nil, nil
	case model.ChannelBillingModeImage:
		imageOutputTokens := tokens.OutputTokens
		if imageOutputTokens <= 0 {
			imageOutputTokens = imageCount
		}
		amount := nullableFloatValue(pricing.PerRequestPrice)
		if unitPrice := nullableFloatValue(pricing.ImageOutputPrice); unitPrice > 0 && imageOutputTokens > 0 {
			amount += float64(imageOutputTokens) * unitPrice
		}
		if amount <= 0 {
			return base, nil, nil
		}
		cost := newChannelFlatCost(amount, multiplier)
		imageCost := cost.TotalCost
		return cost, &imageOutputTokens, &imageCost
	default:
		cost := *base
		if pricing.InputPrice != nil {
			cost.InputCost = float64(tokens.InputTokens) * *pricing.InputPrice
		}
		if pricing.OutputPrice != nil {
			cost.OutputCost = float64(tokens.OutputTokens) * *pricing.OutputPrice
		}
		if pricing.CacheWritePrice != nil {
			cacheWriteTokens := tokens.CacheCreationTokens + tokens.CacheCreation5mTokens + tokens.CacheCreation1hTokens
			cost.CacheCreationCost = float64(cacheWriteTokens) * *pricing.CacheWritePrice
		}
		if pricing.CacheReadPrice != nil {
			cost.CacheReadCost = float64(tokens.CacheReadTokens) * *pricing.CacheReadPrice
		}
		cost.TotalCost = cost.InputCost + cost.OutputCost + cost.CacheCreationCost + cost.CacheReadCost
		if multiplier <= 0 {
			multiplier = 1
		}
		cost.ActualCost = cost.TotalCost * multiplier
		return &cost, nil, nil
	}
}

func newChannelFlatCost(totalCost float64, multiplier float64) *CostBreakdown {
	if multiplier <= 0 {
		multiplier = 1
	}
	return &CostBreakdown{
		TotalCost:  totalCost,
		ActualCost: totalCost * multiplier,
	}
}

func nullableFloatValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
