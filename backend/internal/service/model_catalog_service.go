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
	settingRepo          SettingRepository
	adminService         AdminService
	billingService       *BillingService
	billingCenterService *BillingCenterService
	pricingService       *PricingService
	exchangeRateService  *ModelCatalogExchangeRateService
	modelRegistryService *ModelRegistryService
	cfg                  *config.Config
}

func NewModelCatalogService(
	settingRepo SettingRepository,
	adminService AdminService,
	billingService *BillingService,
	pricingService *PricingService,
	cfg *config.Config,
) *ModelCatalogService {
	service := &ModelCatalogService{
		settingRepo:         settingRepo,
		adminService:        adminService,
		billingService:      billingService,
		pricingService:      pricingService,
		exchangeRateService: newModelCatalogExchangeRateService(nil),
		cfg:                 cfg,
	}
	service.billingCenterService = NewBillingCenterService(settingRepo, billingService)
	service.billingCenterService.SetModelCatalogService(service)
	if billingService != nil {
		billingService.SetBillingCenterService(service.billingCenterService)
		service.billingCenterService.syncBillingServiceOverrides(context.Background())
	}
	return service
}

func (s *ModelCatalogService) SetModelRegistryService(modelRegistryService *ModelRegistryService) {
	s.modelRegistryService = modelRegistryService
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
	items = dedupeModelCatalogItems(items)
	sort.Slice(items, func(i, j int) bool {
		if items[i].DisplayName == items[j].DisplayName {
			return items[i].Model < items[j].Model
		}
		return items[i].DisplayName < items[j].DisplayName
	})
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
	if NormalizeModelCatalogModelID(model) == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	records, err := s.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	record, ok := resolveModelCatalogRecord(records, model)
	if !ok {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	routeReferences, err := s.collectRouteReferences(ctx, record)
	if err != nil {
		return nil, err
	}
	return &ModelCatalogDetail{
		ModelCatalogItem:        recordToModelCatalogItem(record),
		UpstreamPricing:         cloneCatalogPricing(record.upstreamPricing),
		OfficialOverridePricing: cloneModelPricingOverride(record.officialOverridePricing),
		SaleOverridePricing:     cloneModelPricingOverride(record.saleOverridePricing),
		BasePricing:             cloneCatalogPricing(record.officialPricing),
		OverridePricing:         cloneModelPricingOverride(record.saleOverridePricing),
		RouteReferences:         routeReferences,
		RouteReferenceCount:     len(routeReferences),
	}, nil
}

func (s *ModelCatalogService) CopyOfficialPricingToSale(ctx context.Context, actor ModelCatalogActor, model string) (*ModelCatalogDetail, error) {
	detail, err := s.GetModelDetail(ctx, model)
	if err != nil {
		return nil, err
	}
	if detail.OfficialPricing == nil || pricingEmpty(detail.OfficialPricing) {
		return nil, infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_EMPTY", "official pricing is empty")
	}
	payload := UpsertModelPricingOverrideInput{
		Model:               detail.Model,
		ModelCatalogPricing: *cloneCatalogPricing(detail.OfficialPricing),
	}
	return s.UpsertPricingOverride(ctx, actor, payload)
}

func (s *ModelCatalogService) UpsertOfficialPricingOverride(ctx context.Context, actor ModelCatalogActor, input UpsertModelPricingOverrideInput) (*ModelCatalogDetail, error) {
	return s.upsertPricingOverrideByLayer(ctx, actor, input, true)
}

func (s *ModelCatalogService) UpsertPricingOverride(ctx context.Context, actor ModelCatalogActor, input UpsertModelPricingOverrideInput) (*ModelCatalogDetail, error) {
	return s.upsertPricingOverrideByLayer(ctx, actor, input, false)
}

func (s *ModelCatalogService) upsertPricingOverrideByLayer(ctx context.Context, actor ModelCatalogActor, input UpsertModelPricingOverrideInput, official bool) (*ModelCatalogDetail, error) {
	overrideKey := NormalizeModelCatalogModelID(input.Model)
	if overrideKey == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	records, err := s.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	record, ok := resolveModelCatalogRecord(records, input.Model)
	if !ok {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	if isGeminiBillingCompatModel(record.model) {
		return nil, infraerrors.BadRequest(
			"GEMINI_PRICING_OVERRIDE_DEPRECATED",
			"Gemini legacy pricing override is deprecated; update the Gemini billing matrix instead",
		)
	}
	if err := validateOverridePricing(input.ModelCatalogPricing); err != nil {
		return nil, err
	}

	var overrides map[string]*ModelPricingOverride
	var currentEffective *ModelCatalogPricing
	if official {
		overrides = s.loadOfficialPriceOverrides(ctx)
		currentEffective = cloneCatalogPricing(record.officialPricing)
	} else {
		overrides = s.loadSalePriceOverrides(ctx)
		currentEffective = cloneCatalogPricing(record.salePricing)
	}
	if currentEffective == nil {
		currentEffective = &ModelCatalogPricing{}
	}
	nextEffective := cloneCatalogPricing(currentEffective)
	mergeCatalogPricing(nextEffective, &input.ModelCatalogPricing)
	if err := validateTieredPricingConfiguration(nextEffective); err != nil {
		return nil, err
	}

	existingLegacy := cloneModelPricingOverride(overrides[overrideKey])
	override := existingLegacy
	if override == nil {
		override = &ModelPricingOverride{}
	}
	mergeCatalogPricing(&override.ModelCatalogPricing, &input.ModelCatalogPricing)
	override.UpdatedAt = time.Now().UTC()
	override.UpdatedByUserID = actor.UserID
	override.UpdatedByEmail = strings.TrimSpace(actor.Email)
	overrides[overrideKey] = override

	if official {
		if err := s.persistOfficialPriceOverrides(ctx, overrides); err != nil {
			return nil, err
		}
	} else {
		if err := s.persistSalePriceOverrides(ctx, overrides); err != nil {
			return nil, err
		}
	}
	s.billingCenterService.syncBillingServiceOverrides(ctx)
	return s.GetModelDetail(ctx, overrideKey)
}

func (s *ModelCatalogService) clearGeminiLegacyPricingOverrideLayer(ctx context.Context, model string, layer string) error {
	if s == nil || !isGeminiBillingCompatModel(model) {
		return nil
	}
	alias := NormalizeModelCatalogModelID(model)
	if alias == "" {
		alias = CanonicalizeModelNameForPricing(model)
	}
	if alias == "" {
		return nil
	}
	switch normalizeBillingDimension(layer, BillingLayerSale) {
	case BillingLayerOfficial:
		overrides := s.loadOfficialPriceOverrides(ctx)
		if _, ok := overrides[alias]; !ok {
			return nil
		}
		delete(overrides, alias)
		return s.persistOfficialPriceOverrides(ctx, overrides)
	default:
		overrides := s.loadSalePriceOverrides(ctx)
		if _, ok := overrides[alias]; !ok {
			return nil
		}
		delete(overrides, alias)
		return s.persistSalePriceOverrides(ctx, overrides)
	}
}

func (s *ModelCatalogService) DeleteOfficialPricingOverride(ctx context.Context, _ ModelCatalogActor, model string) error {
	return s.deletePricingOverrideByLayer(ctx, model, true)
}

func (s *ModelCatalogService) DeletePricingOverride(ctx context.Context, _ ModelCatalogActor, model string) error {
	return s.deletePricingOverrideByLayer(ctx, model, false)
}

func (s *ModelCatalogService) deletePricingOverrideByLayer(ctx context.Context, model string, official bool) error {
	alias := NormalizeModelCatalogModelID(model)
	if alias == "" {
		return infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	layer := BillingLayerSale
	if official {
		layer = BillingLayerOfficial
	}
	records, err := s.buildCatalogRecords(ctx)
	if err != nil {
		return err
	}
	record, ok := resolveModelCatalogRecord(records, model)
	if !ok {
		return infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	if isGeminiBillingCompatModel(record.model) {
		rules := loadBillingRulesBySetting(ctx, s.settingRepo, SettingKeyBillingCenterRules)
		filteredRules, removedRules := deleteGeminiCompatRules(rules, record, layer)
		foundOverride := false
		if official {
			overrides := s.loadOfficialPriceOverrides(ctx)
			if _, ok := overrides[alias]; ok {
				foundOverride = true
				delete(overrides, alias)
				if err := s.persistOfficialPriceOverrides(ctx, overrides); err != nil {
					return err
				}
			}
		} else {
			overrides := s.loadSalePriceOverrides(ctx)
			if _, ok := overrides[alias]; ok {
				foundOverride = true
				delete(overrides, alias)
				if err := s.persistSalePriceOverrides(ctx, overrides); err != nil {
					return err
				}
			}
		}
		if !foundOverride && !removedRules {
			return infraerrors.NotFound("MODEL_OVERRIDE_NOT_FOUND", "pricing override not found")
		}
		if removedRules {
			if err := persistBillingRulesBySetting(ctx, s.settingRepo, SettingKeyBillingCenterRules, filteredRules); err != nil {
				return err
			}
		}
		s.billingCenterService.syncBillingServiceOverrides(ctx)
		return nil
	}
	if official {
		overrides := s.loadOfficialPriceOverrides(ctx)
		if _, ok := overrides[alias]; !ok {
			return infraerrors.NotFound("MODEL_OVERRIDE_NOT_FOUND", "official pricing override not found")
		}
		delete(overrides, alias)
		if err := s.persistOfficialPriceOverrides(ctx, overrides); err != nil {
			return err
		}
		s.billingCenterService.syncBillingServiceOverrides(ctx)
		return nil
	}
	overrides := s.loadSalePriceOverrides(ctx)
	if _, ok := overrides[alias]; !ok {
		return infraerrors.NotFound("MODEL_OVERRIDE_NOT_FOUND", "sale pricing override not found")
	}
	delete(overrides, alias)
	if err := s.persistSalePriceOverrides(ctx, overrides); err != nil {
		return err
	}
	s.billingCenterService.syncBillingServiceOverrides(ctx)
	return nil
}

func (s *ModelCatalogService) GetUSDCNYExchangeRate(ctx context.Context, force bool) (*ModelCatalogExchangeRate, error) {
	if s.exchangeRateService == nil {
		s.exchangeRateService = newModelCatalogExchangeRateService(nil)
	}
	if force {
		return s.exchangeRateService.RefreshUSDCNY(ctx)
	}
	return s.exchangeRateService.GetUSDCNY(ctx)
}

func recordToModelCatalogItem(record *modelCatalogRecord) ModelCatalogItem {
	pricingSource := record.basePricingSource
	if record.officialOverridePricing != nil || record.saleOverridePricing != nil {
		pricingSource = ModelCatalogPricingSourceOverride
	}
	if pricingSource == "" {
		pricingSource = ModelCatalogPricingSourceNone
	}
	salePricing := cloneCatalogPricing(record.salePricing)
	return ModelCatalogItem{
		Model:                           NormalizeModelCatalogModelID(record.model),
		DisplayName:                     record.displayName,
		IconKey:                         record.iconKey,
		Provider:                        record.provider,
		Mode:                            record.mode,
		DefaultAvailable:                record.defaultAvailable,
		DefaultPlatforms:                append([]string(nil), record.defaultPlatforms...),
		AccessSources:                   append([]string(nil), record.accessSources...),
		PricingSource:                   pricingSource,
		BasePricingSource:               record.basePricingSource,
		HasOverride:                     record.officialOverridePricing != nil || record.saleOverridePricing != nil,
		OfficialPricing:                 cloneCatalogPricing(record.officialPricing),
		SalePricing:                     salePricing,
		EffectivePricing:                salePricing,
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
	if keyword := strings.TrimSpace(filter.Search); keyword != "" {
		normalizedKeyword := NormalizeModelCatalogModelID(keyword)
		resolvedKeyword := normalizeModelCatalogAlias(keyword)
		if !contains(item.Model, keyword) &&
			!contains(item.DisplayName, keyword) &&
			!contains(item.Provider, keyword) &&
			(normalizedKeyword == "" || (!contains(item.Model, normalizedKeyword) && !contains(item.DisplayName, normalizedKeyword))) &&
			(resolvedKeyword == "" || (!contains(item.Model, resolvedKeyword) && !contains(item.DisplayName, resolvedKeyword))) {
			return false
		}
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
