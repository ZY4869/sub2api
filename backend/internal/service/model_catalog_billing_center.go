package service

import "context"

func (s *ModelCatalogService) GetBillingCenter(ctx context.Context) (*BillingCenterPayload, error) {
	if s == nil || s.billingCenterService == nil {
		return &BillingCenterPayload{Sheets: []ModelBillingSheet{}, Rules: []BillingRule{}}, nil
	}
	return s.billingCenterService.List(ctx)
}

func (s *ModelCatalogService) UpsertBillingSheet(ctx context.Context, actor ModelCatalogActor, input UpsertModelBillingSheetInput) (*ModelBillingSheet, error) {
	return s.billingCenterService.UpsertSheet(ctx, actor, input)
}

func (s *ModelCatalogService) DeleteBillingSheet(ctx context.Context, actor ModelCatalogActor, model string, layer string) error {
	return s.billingCenterService.DeleteSheet(ctx, actor, model, layer)
}

func (s *ModelCatalogService) UpsertBillingRule(ctx context.Context, input BillingRule) (*BillingRule, error) {
	return s.billingCenterService.UpsertRule(ctx, input)
}

func (s *ModelCatalogService) DeleteBillingRule(ctx context.Context, id string) error {
	return s.billingCenterService.DeleteRule(ctx, id)
}

func (s *ModelCatalogService) SimulateBilling(ctx context.Context, input BillingSimulationInput) (*BillingSimulationResult, error) {
	return s.billingCenterService.Simulate(ctx, input)
}

func (s *ModelCatalogService) CopyBillingSheetOfficialToSale(ctx context.Context, actor ModelCatalogActor, model string) (*ModelBillingSheet, error) {
	return s.billingCenterService.CopyOfficialToSale(ctx, actor, model)
}
