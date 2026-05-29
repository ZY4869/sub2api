package service

import "context"

// CalculateImageCost calculates image-generation cost.
// model: requested model name, used to resolve default pricing.
// imageSize: image size such as "1K", "2K", or "4K".
// imageCount: generated image count.
// groupConfig: optional group override pricing, nil means using defaults.
// rateMultiplier: billing rate multiplier.
func (s *BillingService) CalculateImageCost(model string, imageSize string, imageCount int, groupConfig *ImagePriceConfig, rateMultiplier float64) *CostBreakdown {
	return s.CalculateImageCostWithServiceTierWithContext(context.Background(), model, imageSize, imageCount, groupConfig, rateMultiplier, "")
}

func (s *BillingService) CalculateImageCostWithServiceTier(model string, imageSize string, imageCount int, groupConfig *ImagePriceConfig, rateMultiplier float64, serviceTier string) *CostBreakdown {
	return s.CalculateImageCostWithServiceTierWithContext(context.Background(), model, imageSize, imageCount, groupConfig, rateMultiplier, serviceTier)
}

func (s *BillingService) CalculateImageCostWithServiceTierWithContext(
	ctx context.Context,
	model string,
	imageSize string,
	imageCount int,
	groupConfig *ImagePriceConfig,
	rateMultiplier float64,
	serviceTier string,
) *CostBreakdown {
	if imageCount <= 0 {
		return finalizeCostBreakdownCurrency(&CostBreakdown{}, nil)
	}

	unitPrice, pricing := s.getImageUnitPriceWithPricingWithContext(ctx, model, imageSize, groupConfig, serviceTier)
	totalCost := unitPrice * float64(imageCount)
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}
	actualCost := totalCost * rateMultiplier

	return finalizeCostBreakdownCurrency(&CostBreakdown{
		TotalCost:  totalCost,
		ActualCost: actualCost,
	}, pricing)
}

// CalculateVideoRequestCost calculates one-shot video request billing using model pricing.
func (s *BillingService) CalculateVideoRequestCost(model string, rateMultiplier float64) *CostBreakdown {
	return s.CalculateVideoRequestCostWithContext(context.Background(), model, rateMultiplier)
}

func (s *BillingService) CalculateVideoRequestCostWithContext(ctx context.Context, model string, rateMultiplier float64) *CostBreakdown {
	unitPrice := 0.0
	var pricing *ModelPricing
	if resolved, err := s.getPricingForBillingWithContext(ctx, model); err == nil && resolved != nil && resolved.OutputPricePerVideoRequest > 0 {
		pricing = resolved
		unitPrice = pricing.OutputPricePerVideoRequest
	}
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}
	return finalizeCostBreakdownCurrency(&CostBreakdown{
		TotalCost:  unitPrice,
		ActualCost: unitPrice * rateMultiplier,
	}, pricing)
}

func (s *BillingService) getImageUnitPriceWithPricingWithContext(
	ctx context.Context,
	model string,
	imageSize string,
	groupConfig *ImagePriceConfig,
	serviceTier string,
) (float64, *ModelPricing) {
	if groupConfig != nil {
		switch imageSize {
		case "1K":
			if groupConfig.Price1K != nil {
				return *groupConfig.Price1K, nil
			}
		case "2K":
			if groupConfig.Price2K != nil {
				return *groupConfig.Price2K, nil
			}
		case "4K":
			if groupConfig.Price4K != nil {
				return *groupConfig.Price4K, nil
			}
		}
	}
	pricing, _ := s.getPricingForBillingWithContext(ctx, model)
	basePrice := 0.0
	if pricing != nil {
		basePrice = pricing.OutputPricePerImage
		switch normalizeBillingServiceTier(serviceTier) {
		case BillingServiceTierPriority:
			if pricing.OutputPricePerImagePriority > 0 {
				basePrice = pricing.OutputPricePerImagePriority
			}
		case BillingServiceTierFlex:
			if basePrice > 0 {
				basePrice *= serviceTierCostMultiplier(BillingServiceTierFlex)
			}
		}
	}
	if basePrice <= 0 {
		basePrice = 0.134
		pricing = nil
	}
	if imageSize == "2K" {
		return basePrice * 1.5, pricing
	}
	if imageSize == "4K" {
		return basePrice * 2, pricing
	}
	return basePrice, pricing
}
