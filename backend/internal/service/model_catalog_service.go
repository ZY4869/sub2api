package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type ModelCatalogService struct {
	settingRepo    SettingRepository
	adminService   AdminService
	billingService *BillingService
	pricingService *PricingService
	cfg            *config.Config
}

func NewModelCatalogService(
	settingRepo SettingRepository,
	adminService AdminService,
	billingService *BillingService,
	pricingService *PricingService,
	cfg *config.Config,
) *ModelCatalogService {
	service := &ModelCatalogService{
		settingRepo:    settingRepo,
		adminService:   adminService,
		billingService: billingService,
		pricingService: pricingService,
		cfg:            cfg,
	}
	if billingService != nil {
		billingService.ReplaceModelPriceOverrides(service.loadPriceOverrides(context.Background()))
	}
	return service
}

func (s *ModelCatalogService) ListModels(ctx context.Context, filter ModelCatalogListFilter) ([]ModelCatalogItem, int64, error) {
	records, err := s.buildCatalogRecords(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]ModelCatalogItem, 0, len(records))
	for _, record := range records {
		item := recordToModelCatalogItem(record)
		if matchesModelCatalogFilter(item, filter) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Model < items[j].Model })
	total := int64(len(items))
	page, pageSize := normalizeListPagination(filter.Page, filter.PageSize)
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []ModelCatalogItem{}, total, nil
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], total, nil
}

func (s *ModelCatalogService) GetModelDetail(ctx context.Context, model string) (*ModelCatalogDetail, error) {
	canonicalModel := CanonicalizeModelNameForPricing(model)
	if canonicalModel == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	records, err := s.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	record, ok := records[canonicalModel]
	if !ok {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	routeReferences, err := s.collectRouteReferences(ctx, canonicalModel, record.mode)
	if err != nil {
		return nil, err
	}
	detail := &ModelCatalogDetail{
		ModelCatalogItem:    recordToModelCatalogItem(record),
		BasePricing:         cloneCatalogPricing(record.basePricing),
		OverridePricing:     record.overridePricing,
		RouteReferences:     routeReferences,
		RouteReferenceCount: len(routeReferences),
	}
	return detail, nil
}

func (s *ModelCatalogService) UpsertPricingOverride(
	ctx context.Context,
	actor ModelCatalogActor,
	input UpsertModelPricingOverrideInput,
) (*ModelCatalogDetail, error) {
	canonicalModel := CanonicalizeModelNameForPricing(input.Model)
	if canonicalModel == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	records, err := s.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	record, ok := records[canonicalModel]
	if !ok {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	if err := validateOverridePricing(input.ModelCatalogPricing); err != nil {
		return nil, err
	}
	overrides := s.loadPriceOverrides(ctx)
	override := overrides[canonicalModel]
	if override == nil {
		override = &ModelPricingOverride{}
	}
	effectivePricing := cloneCatalogPricing(record.basePricing)
	if effectivePricing == nil {
		effectivePricing = &ModelCatalogPricing{}
	}
	if override != nil {
		mergeCatalogPricing(effectivePricing, &override.ModelCatalogPricing)
	}
	mergeCatalogPricing(effectivePricing, &input.ModelCatalogPricing)
	if err := validateTieredPricingConfiguration(effectivePricing); err != nil {
		return nil, err
	}
	mergeCatalogPricing(&override.ModelCatalogPricing, &input.ModelCatalogPricing)
	override.UpdatedAt = time.Now().UTC()
	override.UpdatedByUserID = actor.UserID
	override.UpdatedByEmail = strings.TrimSpace(actor.Email)
	overrides[canonicalModel] = override
	if err := s.persistPriceOverrides(ctx, overrides); err != nil {
		return nil, err
	}
	if s.billingService != nil {
		s.billingService.ReplaceModelPriceOverrides(overrides)
	}
	return s.GetModelDetail(ctx, canonicalModel)
}

func (s *ModelCatalogService) DeletePricingOverride(ctx context.Context, _ ModelCatalogActor, model string) error {
	canonicalModel := CanonicalizeModelNameForPricing(model)
	if canonicalModel == "" {
		return infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	overrides := s.loadPriceOverrides(ctx)
	delete(overrides, canonicalModel)
	if err := s.persistPriceOverrides(ctx, overrides); err != nil {
		return err
	}
	if s.billingService != nil {
		s.billingService.ReplaceModelPriceOverrides(overrides)
	}
	return nil
}

func recordToModelCatalogItem(record *modelCatalogRecord) ModelCatalogItem {
	effectivePricing := applyPricingOverride(record.basePricing, record.overridePricing)
	pricingSource := record.basePricingSource
	if record.overridePricing != nil {
		pricingSource = ModelCatalogPricingSourceOverride
	}
	if pricingSource == "" {
		pricingSource = ModelCatalogPricingSourceNone
	}
	return ModelCatalogItem{
		Model:                           record.model,
		Provider:                        record.provider,
		Mode:                            record.mode,
		DefaultAvailable:                record.defaultAvailable,
		DefaultPlatforms:                record.defaultPlatforms,
		PricingSource:                   pricingSource,
		BasePricingSource:               record.basePricingSource,
		HasOverride:                     record.overridePricing != nil,
		EffectivePricing:                effectivePricing,
		SupportsPromptCaching:           record.supportsPromptCaching,
		SupportsServiceTier:             record.supportsServiceTier,
		LongContextInputTokenThreshold:  record.longContextInputTokenThreshold,
		LongContextInputCostMultiplier:  record.longContextInputCostMultiplier,
		LongContextOutputCostMultiplier: record.longContextOutputCostMultiplier,
	}
}

func matchesModelCatalogFilter(item ModelCatalogItem, filter ModelCatalogListFilter) bool {
	contains := func(value string, keyword string) bool {
		return strings.Contains(strings.ToLower(value), strings.ToLower(strings.TrimSpace(keyword)))
	}
	if keyword := strings.TrimSpace(filter.Search); keyword != "" && !contains(item.Model, keyword) && !contains(item.Provider, keyword) {
		return false
	}
	if provider := strings.TrimSpace(filter.Provider); provider != "" && !strings.EqualFold(provider, item.Provider) {
		return false
	}
	if mode := strings.TrimSpace(filter.Mode); mode != "" && !strings.EqualFold(mode, item.Mode) {
		return false
	}
	if availability := strings.TrimSpace(filter.Availability); availability == "available" && !item.DefaultAvailable {
		return false
	} else if availability == "unavailable" && item.DefaultAvailable {
		return false
	}
	if pricingSource := strings.TrimSpace(filter.PricingSource); pricingSource != "" && !strings.EqualFold(pricingSource, item.PricingSource) {
		return false
	}
	return true
}

func normalizeListPagination(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return page, pageSize
}
