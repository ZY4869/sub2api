package service

import (
	"context"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func (s *BillingCenterService) ListPricingProviders(ctx context.Context) ([]BillingPricingProviderGroup, error) {
	if s == nil || s.modelCatalogService == nil {
		return []BillingPricingProviderGroup{}, nil
	}
	records, err := s.modelCatalogService.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	grouped := map[string]*BillingPricingProviderGroup{}
	rules := s.ListRules(ctx)
	for _, record := range records {
		if record == nil {
			continue
		}
		provider := strings.TrimSpace(record.provider)
		if provider == "" {
			provider = "unknown"
		}
		group := grouped[provider]
		if group == nil {
			group = &BillingPricingProviderGroup{
				Provider: provider,
				Label:    strings.ToUpper(provider[:1]) + provider[1:],
			}
			grouped[provider] = group
		}
		group.TotalCount++
		group.OfficialCount += len(pricingItemsForRecord(record, BillingLayerOfficial, rules))
		group.SaleCount += len(pricingItemsForRecord(record, BillingLayerSale, rules))
	}
	items := make([]BillingPricingProviderGroup, 0, len(grouped))
	for _, group := range grouped {
		items = append(items, *group)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Label == items[j].Label {
			return items[i].Provider < items[j].Provider
		}
		return items[i].Label < items[j].Label
	})
	return items, nil
}

func (s *BillingCenterService) ListPricingModels(ctx context.Context, filter BillingPricingListFilter) ([]BillingPricingListItem, int64, error) {
	if s == nil || s.modelCatalogService == nil {
		return []BillingPricingListItem{}, 0, nil
	}
	records, err := s.modelCatalogService.buildCatalogRecords(ctx)
	if err != nil {
		return nil, 0, err
	}
	rules := s.ListRules(ctx)
	items := make([]BillingPricingListItem, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		item := BillingPricingListItem{
			Model:         NormalizeModelCatalogModelID(record.model),
			DisplayName:   record.displayName,
			Provider:      record.provider,
			Mode:          record.mode,
			Capabilities:  billingPricingCapabilitiesForRecord(record),
			OfficialCount: len(pricingItemsForRecord(record, BillingLayerOfficial, rules)),
			SaleCount:     len(pricingItemsForRecord(record, BillingLayerSale, rules)),
		}
		item.PriceItemCount = item.OfficialCount + item.SaleCount
		if matchesBillingPricingFilter(item, filter) {
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].DisplayName == items[j].DisplayName {
			return items[i].Model < items[j].Model
		}
		return items[i].DisplayName < items[j].DisplayName
	})
	total := int64(len(items))
	page, pageSize := normalizeListPagination(filter.Page, filter.PageSize)
	if pageSize > 100 {
		pageSize = 100
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []BillingPricingListItem{}, total, nil
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], total, nil
}

func matchesBillingPricingFilter(item BillingPricingListItem, filter BillingPricingListFilter) bool {
	contains := func(value string, keyword string) bool {
		return strings.Contains(strings.ToLower(value), strings.ToLower(strings.TrimSpace(keyword)))
	}
	if keyword := strings.TrimSpace(filter.Search); keyword != "" &&
		!contains(item.Model, keyword) &&
		!contains(item.DisplayName, keyword) &&
		!contains(item.Provider, keyword) {
		return false
	}
	if provider := strings.TrimSpace(filter.Provider); provider != "" && !strings.EqualFold(provider, item.Provider) {
		return false
	}
	if mode := strings.TrimSpace(filter.Mode); mode != "" && !strings.EqualFold(mode, item.Mode) {
		return false
	}
	return true
}

func (s *BillingCenterService) GetPricingDetails(ctx context.Context, models []string) ([]BillingPricingSheetDetail, error) {
	if s == nil || s.modelCatalogService == nil {
		return []BillingPricingSheetDetail{}, nil
	}
	records, err := s.modelCatalogService.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	rules := s.ListRules(ctx)
	items := make([]BillingPricingSheetDetail, 0, len(models))
	for _, model := range models {
		record, ok := resolveModelCatalogRecord(records, model)
		if !ok || record == nil {
			return nil, infraerrors.NotFound("BILLING_MODEL_NOT_FOUND", "billing model not found")
		}
		officialItems := pricingItemsForRecord(record, BillingLayerOfficial, rules)
		saleItems := pricingItemsForRecord(record, BillingLayerSale, rules)
		combinedItems := append(append([]BillingPriceItem(nil), officialItems...), saleItems...)
		metadata := billingPricingMetadataForRecord(record, combinedItems)
		items = append(items, BillingPricingSheetDetail{
			Model:                           NormalizeModelCatalogModelID(record.model),
			DisplayName:                     record.displayName,
			Provider:                        record.provider,
			Mode:                            record.mode,
			Currency:                        defaultModelPricingCurrency(record.pricingCurrency),
			InputSupported:                  metadata.InputSupported,
			OutputChargeSlot:                metadata.OutputChargeSlot,
			SupportsPromptCaching:           record.supportsPromptCaching,
			SupportsServiceTier:             record.supportsServiceTier,
			LongContextInputTokenThreshold:  record.longContextInputTokenThreshold,
			LongContextInputCostMultiplier:  record.longContextInputCostMultiplier,
			LongContextOutputCostMultiplier: record.longContextOutputCostMultiplier,
			Capabilities:                    billingPricingCapabilitiesForRecord(record),
			OfficialForm:                    billingPricingLayerFormFromItemsWithMetadata(metadata, officialItems),
			SaleForm:                        billingPricingLayerFormFromItemsWithMetadata(metadata, saleItems),
			OfficialItems:                   officialItems,
			SaleItems:                       saleItems,
		})
	}
	return items, nil
}

func (s *BillingCenterService) SavePricingLayer(ctx context.Context, actor ModelCatalogActor, input UpsertBillingPricingLayerInput) (*BillingPricingSheetDetail, error) {
	if s == nil || s.modelCatalogService == nil {
		return nil, infraerrors.ServiceUnavailable("BILLING_CENTER_UNAVAILABLE", "billing center service unavailable")
	}
	layer := normalizeBillingDimension(input.Layer, BillingLayerSale)
	if layer != BillingLayerOfficial && layer != BillingLayerSale {
		return nil, infraerrors.BadRequest("BILLING_LAYER_INVALID", "layer must be official or sale")
	}
	records, err := s.modelCatalogService.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	record, ok := resolveModelCatalogRecord(records, input.Model)
	if !ok || record == nil {
		return nil, infraerrors.NotFound("BILLING_MODEL_NOT_FOUND", "billing model not found")
	}
	rules := s.ListRules(ctx)
	currentItems := pricingItemsForRecord(record, layer, rules)
	metadata := billingPricingMetadataForRecord(record, currentItems)
	currency := normalizeModelPricingCurrency(input.Currency)
	if currency == "" {
		currency = defaultModelPricingCurrency(record.pricingCurrency)
	}
	form := BillingPricingLayerForm{}
	switch {
	case input.Form != nil:
		form = cloneBillingPricingLayerForm(*input.Form)
	case len(input.Items) > 0:
		form = billingPricingLayerFormFromItemsWithMetadata(metadata, input.Items)
	}
	if err := validateBillingPricingLayerForm(form); err != nil {
		return nil, err
	}
	items := billingPricingItemsFromForm(metadata, layer, form)

	legacyPricing := flatPricingFromItems(items)
	if err := validateFlatPricingForSave(legacyPricing); err != nil {
		return nil, err
	}
	if err := s.modelCatalogService.ReplacePricingOverrideLayer(ctx, actor, record.model, layer == BillingLayerOfficial, legacyPricing); err != nil {
		return nil, err
	}

	rules = deleteGeneratedPricingRules(rules, record.model, layer)
	rules, _ = deleteGeminiCompatRules(rules, record, layer)
	if isGeminiBillingCompatModel(record.model) {
		matrix := geminiMatrixFromSimpleForm(form)
		rules = replaceGeminiMatrixRules(rules, record, layer, matrix)
	} else {
		rules, _ = deleteGeminiMatrixRules(rules, record.model, layer)
	}
	rules = append(rules, billingPricingRulesFromForm(record, layer, items)...)
	if err := persistBillingRulesBySetting(ctx, s.settingRepo, SettingKeyBillingCenterRules, rules); err != nil {
		return nil, err
	}
	if err := s.modelCatalogService.saveModelPricingCurrency(ctx, actor, record.model, currency); err != nil {
		return nil, err
	}
	s.syncBillingServiceOverrides(ctx)
	logger.FromContext(ctx).Info(
		"billing pricing layer normalized save",
		zap.String("component", "service.billing_center"),
		zap.String("model", record.model),
		zap.String("layer", layer),
		zap.String("currency", currency),
		zap.Bool("input_supported", metadata.InputSupported),
		zap.String("output_charge_slot", metadata.OutputChargeSlot),
		zap.Bool("special_enabled", form.SpecialEnabled),
		zap.Bool("tiered_enabled", form.TieredEnabled),
	)

	details, err := s.GetPricingDetails(ctx, []string{record.model})
	if err != nil || len(details) == 0 {
		return nil, err
	}
	return &details[0], nil
}

func (s *BillingCenterService) CopyPricingItemsOfficialToSale(ctx context.Context, actor ModelCatalogActor, models []string) ([]BillingPricingSheetDetail, error) {
	details, err := s.GetPricingDetails(ctx, models)
	if err != nil {
		return nil, err
	}
	updated := make([]BillingPricingSheetDetail, 0, len(details))
	for _, detail := range details {
		form := cloneBillingPricingLayerForm(detail.OfficialForm)
		next, err := s.SavePricingLayer(ctx, actor, UpsertBillingPricingLayerInput{
			Model: detail.Model,
			Layer: BillingLayerSale,
			Form:  &form,
		})
		if err != nil {
			return nil, err
		}
		updated = append(updated, *next)
	}
	return updated, nil
}

func (s *BillingCenterService) ApplySaleDiscount(ctx context.Context, actor ModelCatalogActor, input BillingBulkApplyRequest) ([]BillingPricingSheetDetail, error) {
	if input.DiscountRatio <= 0 {
		return nil, infraerrors.BadRequest("BILLING_DISCOUNT_RATIO_INVALID", "discount ratio must be greater than zero")
	}
	details, err := s.GetPricingDetails(ctx, input.Models)
	if err != nil {
		return nil, err
	}
	itemFilter := make(map[string]struct{}, len(input.ItemIDs))
	for _, itemID := range input.ItemIDs {
		itemFilter[strings.TrimSpace(itemID)] = struct{}{}
	}
	updated := make([]BillingPricingSheetDetail, 0, len(details))
	for _, detail := range details {
		form := applyDiscountToBillingPricingLayerForm(detail.SaleForm, input.DiscountRatio, itemFilter)
		next, err := s.SavePricingLayer(ctx, actor, UpsertBillingPricingLayerInput{
			Model: detail.Model,
			Layer: BillingLayerSale,
			Form:  &form,
		})
		if err != nil {
			return nil, err
		}
		updated = append(updated, *next)
	}
	return updated, nil
}

func flatPricingFromItems(items []BillingPriceItem) *ModelCatalogPricing {
	pricing := &ModelCatalogPricing{}
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if shouldPersistAsRule(item) {
			continue
		}
		switch item.ChargeSlot {
		case BillingChargeSlotTextInput:
			if item.ServiceTier == BillingServiceTierPriority {
				pricing.InputCostPerTokenPriority = modelCatalogFloat64Ptr(item.Price)
				if item.ThresholdTokens != nil {
					pricing.InputTokenThreshold = modelCatalogIntPtr(*item.ThresholdTokens)
				}
				if item.PriceAboveThresh != nil {
					pricing.InputCostPerTokenPriorityAboveThreshold = modelCatalogFloat64Ptr(*item.PriceAboveThresh)
				}
			} else {
				pricing.InputCostPerToken = modelCatalogFloat64Ptr(item.Price)
				if item.ThresholdTokens != nil {
					pricing.InputTokenThreshold = modelCatalogIntPtr(*item.ThresholdTokens)
				}
				if item.PriceAboveThresh != nil {
					pricing.InputCostPerTokenAboveThreshold = modelCatalogFloat64Ptr(*item.PriceAboveThresh)
				}
			}
		case BillingChargeSlotTextOutput:
			if item.ServiceTier == BillingServiceTierPriority {
				pricing.OutputCostPerTokenPriority = modelCatalogFloat64Ptr(item.Price)
				if item.ThresholdTokens != nil {
					pricing.OutputTokenThreshold = modelCatalogIntPtr(*item.ThresholdTokens)
				}
				if item.PriceAboveThresh != nil {
					pricing.OutputCostPerTokenPriorityAboveThreshold = modelCatalogFloat64Ptr(*item.PriceAboveThresh)
				}
			} else {
				pricing.OutputCostPerToken = modelCatalogFloat64Ptr(item.Price)
				if item.ThresholdTokens != nil {
					pricing.OutputTokenThreshold = modelCatalogIntPtr(*item.ThresholdTokens)
				}
				if item.PriceAboveThresh != nil {
					pricing.OutputCostPerTokenAboveThreshold = modelCatalogFloat64Ptr(*item.PriceAboveThresh)
				}
			}
		case BillingChargeSlotCacheCreate:
			pricing.CacheCreationInputTokenCost = modelCatalogFloat64Ptr(item.Price)
		case BillingChargeSlotCacheRead:
			if item.ServiceTier == BillingServiceTierPriority {
				pricing.CacheReadInputTokenCostPriority = modelCatalogFloat64Ptr(item.Price)
			} else {
				pricing.CacheReadInputTokenCost = modelCatalogFloat64Ptr(item.Price)
			}
		case BillingChargeSlotCacheStorageTokenHour:
			pricing.CacheCreationInputTokenCostAbove1hr = modelCatalogFloat64Ptr(item.Price)
		case BillingChargeSlotImageOutput:
			if item.ServiceTier == BillingServiceTierPriority {
				pricing.OutputCostPerImagePriority = modelCatalogFloat64Ptr(item.Price)
			} else {
				pricing.OutputCostPerImage = modelCatalogFloat64Ptr(item.Price)
			}
		case BillingChargeSlotVideoRequest:
			pricing.OutputCostPerVideoRequest = modelCatalogFloat64Ptr(item.Price)
		}
	}
	if pricingEmpty(pricing) {
		return nil
	}
	return pricing
}

func validateFlatPricingForSave(pricing *ModelCatalogPricing) error {
	if pricing == nil || pricingEmpty(pricing) {
		return nil
	}
	if err := validateOverridePricing(*pricing); err != nil {
		return err
	}
	return validateTieredPricingConfiguration(pricing)
}

func shouldPersistAsRule(item BillingPriceItem) bool {
	return !canPersistAsFlatPricing(item)
}

func canPersistAsFlatPricing(raw BillingPriceItem) bool {
	item := normalizeBillingPriceItem(raw)
	slot := normalizeBillingDimension(item.ChargeSlot, "")
	if slot == "" {
		return false
	}
	if normalizeBillingActualBatchMode(item.BatchMode) == BillingBatchModeBatch || item.ContextWindow != "" {
		return false
	}
	if surface := strings.TrimSpace(strings.ToLower(item.Surface)); surface != "" && surface != BillingSurfaceGeminiNative {
		return false
	}
	if item.InputModality != "" || item.OutputModality != "" || item.CachePhase != "" || item.GroundingKind != "" {
		return false
	}
	if item.OperationType != "" && item.OperationType != operationTypeForChargeSlot(slot) {
		return false
	}
	tier := normalizeBillingServiceTier(item.ServiceTier)
	if tier == BillingServiceTierStandard {
		tier = ""
	}
	switch slot {
	case BillingChargeSlotTextInput, BillingChargeSlotTextOutput:
		if tier != "" && tier != BillingServiceTierPriority {
			return false
		}
		return item.Mode != BillingPriceItemModeProviderRule
	case BillingChargeSlotCacheCreate, BillingChargeSlotVideoRequest:
		return tier == "" && item.Mode != BillingPriceItemModeProviderRule
	case BillingChargeSlotCacheRead, BillingChargeSlotImageOutput:
		if tier != "" && tier != BillingServiceTierPriority {
			return false
		}
		return item.Mode != BillingPriceItemModeProviderRule
	case BillingChargeSlotCacheStorageTokenHour:
		return tier == ""
	default:
		return false
	}
}

func pricingRulesFromItems(record *modelCatalogRecord, layer string, items []BillingPriceItem) []BillingRule {
	if record == nil {
		return nil
	}
	rules := make([]BillingRule, 0, len(items))
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if !shouldPersistAsRule(item) {
			continue
		}
		rules = append(rules, BillingRule{
			ID:            generatedPricingRuleID(record.model, item),
			Provider:      record.provider,
			Layer:         normalizeBillingDimension(layer, BillingLayerSale),
			Surface:       normalizeBillingSurface(defaultString(item.Surface, BillingSurfaceAny)),
			OperationType: normalizeBillingDimension(item.OperationType, operationTypeForChargeSlot(item.ChargeSlot)),
			ServiceTier:   billingRuleServiceTierForItem(item),
			BatchMode:     normalizeBillingBatchMode(defaultString(item.BatchMode, BillingBatchModeAny)),
			Matchers: BillingRuleMatchers{
				Models:         modelCatalogRecordLookupCandidates(record),
				InputModality:  item.InputModality,
				OutputModality: item.OutputModality,
				CachePhase:     item.CachePhase,
				GroundingKind:  item.GroundingKind,
				ContextWindow:  item.ContextWindow,
			},
			Unit:     item.Unit,
			Price:    item.Price,
			Priority: 2500,
			Enabled:  item.Enabled,
		})
	}
	return rules
}

func billingRuleServiceTierForItem(item BillingPriceItem) string {
	tier := normalizeBillingServiceTier(item.ServiceTier)
	if tier == BillingServiceTierStandard {
		return ""
	}
	return tier
}

func deleteGeneratedPricingRules(rules []BillingRule, model string, layer string) []BillingRule {
	filtered := make([]BillingRule, 0, len(rules))
	for _, rule := range rules {
		if strings.HasPrefix(strings.TrimSpace(rule.ID), billingPricingRuleIDPrefix+"__") &&
			rule.Layer == normalizeBillingDimension(layer, BillingLayerSale) &&
			billingRuleMatchesModel(rule, model) {
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered
}

func generatedPricingRuleID(model string, item BillingPriceItem) string {
	return strings.Join([]string{
		billingPricingRuleIDPrefix,
		normalizeBillingDimension(item.Layer, BillingLayerSale),
		CanonicalizeModelNameForPricing(model),
		normalizeBillingDimension(item.ChargeSlot, ""),
		normalizeBillingDimension(item.ServiceTier, "default"),
		normalizeBillingActualBatchMode(item.BatchMode),
		normalizeBillingSurface(defaultString(item.Surface, BillingSurfaceAny)),
	}, "__")
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (s *ModelCatalogService) ReplacePricingOverrideLayer(ctx context.Context, actor ModelCatalogActor, model string, official bool, pricing *ModelCatalogPricing) error {
	alias := NormalizeModelCatalogModelID(model)
	if alias == "" {
		return infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	if pricing != nil && !pricingEmpty(pricing) {
		if err := validateOverridePricing(*pricing); err != nil {
			return err
		}
		if err := validateTieredPricingConfiguration(pricing); err != nil {
			return err
		}
	}
	var overrides map[string]*ModelPricingOverride
	if official {
		overrides = s.loadOfficialPriceOverrides(ctx)
	} else {
		overrides = s.loadSalePriceOverrides(ctx)
	}
	if pricing == nil || pricingEmpty(pricing) {
		delete(overrides, alias)
	} else {
		overrides[alias] = &ModelPricingOverride{
			ModelCatalogPricing: *cloneCatalogPricing(pricing),
			UpdatedAt:           time.Now().UTC(),
			UpdatedByUserID:     actor.UserID,
			UpdatedByEmail:      strings.TrimSpace(actor.Email),
		}
	}
	if official {
		return s.persistOfficialPriceOverrides(ctx, overrides)
	}
	return s.persistSalePriceOverrides(ctx, overrides)
}
