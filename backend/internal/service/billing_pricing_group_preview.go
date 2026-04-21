package service

import (
	"context"
)

func cloneBillingPricingLayerFormPtr(form BillingPricingLayerForm) *BillingPricingLayerForm {
	cloned := cloneBillingPricingLayerForm(form)
	return &cloned
}

func scaleBillingFloat64(value *float64, multiplier float64) *float64 {
	if value == nil {
		return nil
	}
	scaled := (*value) * multiplier
	return &scaled
}

func scaleBillingPricingLayerForm(form BillingPricingLayerForm, multiplier float64) BillingPricingLayerForm {
	scaled := cloneBillingPricingLayerForm(form)
	scaled.InputPrice = scaleBillingFloat64(form.InputPrice, multiplier)
	scaled.OutputPrice = scaleBillingFloat64(form.OutputPrice, multiplier)
	scaled.CachePrice = scaleBillingFloat64(form.CachePrice, multiplier)
	scaled.InputPriceAboveThreshold = scaleBillingFloat64(form.InputPriceAboveThreshold, multiplier)
	scaled.OutputPriceAboveThreshold = scaleBillingFloat64(form.OutputPriceAboveThreshold, multiplier)
	scaled.Special.BatchInputPrice = scaleBillingFloat64(form.Special.BatchInputPrice, multiplier)
	scaled.Special.BatchOutputPrice = scaleBillingFloat64(form.Special.BatchOutputPrice, multiplier)
	scaled.Special.BatchCachePrice = scaleBillingFloat64(form.Special.BatchCachePrice, multiplier)
	scaled.Special.GroundingSearch = scaleBillingFloat64(form.Special.GroundingSearch, multiplier)
	scaled.Special.GroundingMaps = scaleBillingFloat64(form.Special.GroundingMaps, multiplier)
	scaled.Special.FileSearchEmbedding = scaleBillingFloat64(form.Special.FileSearchEmbedding, multiplier)
	scaled.Special.FileSearchRetrieval = scaleBillingFloat64(form.Special.FileSearchRetrieval, multiplier)
	return scaled
}

func (s *BillingCenterService) resolveBillingPreviewGroupMultiplier(ctx context.Context, groupID *int64) (*Group, *float64, error) {
	if s == nil || groupID == nil || *groupID <= 0 || s.modelCatalogService == nil || s.modelCatalogService.adminService == nil {
		return nil, nil, nil
	}
	group, err := s.modelCatalogService.adminService.GetGroup(ctx, *groupID)
	if err != nil || group == nil {
		return nil, nil, err
	}
	multiplier := group.RateMultiplier
	return group, &multiplier, nil
}

func billingPricingPreviewPriceDisplay(model BillingPricingPersistedModel, form BillingPricingLayerForm) *PublicModelCatalogPriceDisplay {
	metadata := billingPricingMetadataForPersistedModel(model)
	display := publicModelCatalogPriceDisplayFromForm(metadata, form)
	if len(display.Primary) == 0 && len(display.Secondary) == 0 {
		return nil
	}
	return &display
}
