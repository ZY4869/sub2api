package service

import "context"

func (s *ModelCatalogService) ListBillingPricingProviders(ctx context.Context) ([]BillingPricingProviderGroup, error) {
	if s == nil || s.billingCenterService == nil {
		return []BillingPricingProviderGroup{}, nil
	}
	return s.billingCenterService.ListPricingProviders(ctx)
}

func (s *ModelCatalogService) ListBillingPricingModels(ctx context.Context, filter BillingPricingListFilter) ([]BillingPricingListItem, int64, error) {
	if s == nil || s.billingCenterService == nil {
		return []BillingPricingListItem{}, 0, nil
	}
	return s.billingCenterService.ListPricingModels(ctx, filter)
}

func (s *ModelCatalogService) GetBillingPricingDetails(ctx context.Context, models []string) ([]BillingPricingSheetDetail, error) {
	if s == nil || s.billingCenterService == nil {
		return []BillingPricingSheetDetail{}, nil
	}
	return s.billingCenterService.GetPricingDetails(ctx, models)
}

func (s *ModelCatalogService) SaveBillingPricingLayer(ctx context.Context, actor ModelCatalogActor, input UpsertBillingPricingLayerInput) (*BillingPricingSheetDetail, error) {
	if s == nil || s.billingCenterService == nil {
		return nil, nil
	}
	return s.billingCenterService.SavePricingLayer(ctx, actor, input)
}

func (s *ModelCatalogService) RefreshBillingPricingCatalog(ctx context.Context) (*BillingPricingRefreshResult, error) {
	if s == nil || s.billingCenterService == nil {
		return nil, nil
	}
	return s.billingCenterService.RefreshPricingCatalog(ctx)
}

func (s *ModelCatalogService) GetBillingPricingAudit(ctx context.Context) (*BillingPricingAudit, error) {
	if s == nil || s.billingCenterService == nil {
		return &BillingPricingAudit{}, nil
	}
	return s.billingCenterService.GetPricingAudit(ctx)
}

func (s *ModelCatalogService) CopyBillingPricingOfficialToSale(ctx context.Context, actor ModelCatalogActor, models []string) ([]BillingPricingSheetDetail, error) {
	if s == nil || s.billingCenterService == nil {
		return []BillingPricingSheetDetail{}, nil
	}
	return s.billingCenterService.CopyPricingItemsOfficialToSale(ctx, actor, models)
}

func (s *ModelCatalogService) ApplyBillingPricingSaleDiscount(ctx context.Context, actor ModelCatalogActor, input BillingBulkApplyRequest) ([]BillingPricingSheetDetail, error) {
	if s == nil || s.billingCenterService == nil {
		return []BillingPricingSheetDetail{}, nil
	}
	return s.billingCenterService.ApplySaleDiscount(ctx, actor, input)
}

func (s *ModelCatalogService) ListBillingRules(ctx context.Context) []BillingRule {
	if s == nil || s.billingCenterService == nil {
		return []BillingRule{}
	}
	return editableBillingRules(s.billingCenterService.ListRules(ctx))
}
