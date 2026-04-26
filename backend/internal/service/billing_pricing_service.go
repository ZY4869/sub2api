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
	snapshot, err := s.ensureBillingPricingCatalogMigrated(ctx)
	if err != nil {
		return nil, err
	}
	grouped := map[string]*BillingPricingProviderGroup{}
	for _, model := range snapshot.Models {
		provider := strings.TrimSpace(model.Provider)
		if provider == "" {
			provider = "unknown"
		}
		group := grouped[provider]
		if group == nil {
			group = &BillingPricingProviderGroup{
				Provider: provider,
				Label:    FormatProviderLabel(provider),
			}
			grouped[provider] = group
		}
		group.TotalCount++
		group.OfficialCount += model.OfficialCount
		group.SaleCount += model.SaleCount
	}
	items := make([]BillingPricingProviderGroup, 0, len(grouped))
	for _, group := range grouped {
		items = append(items, *group)
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(items[i].Label))
		right := strings.ToLower(strings.TrimSpace(items[j].Label))
		if left == right {
			return items[i].Provider < items[j].Provider
		}
		return left < right
	})
	return items, nil
}

func (s *BillingCenterService) ListPricingModels(ctx context.Context, filter BillingPricingListFilter) ([]BillingPricingListItem, int64, error) {
	if s == nil || s.modelCatalogService == nil {
		return []BillingPricingListItem{}, 0, nil
	}
	snapshot, err := s.ensureBillingPricingCatalogMigrated(ctx)
	if err != nil {
		return nil, 0, err
	}
	_, groupMultiplier, err := s.resolveBillingPreviewGroupMultiplier(ctx, filter.GroupID)
	if err != nil {
		return nil, 0, err
	}
	items := make([]BillingPricingListItem, 0, len(snapshot.Models))
	for _, model := range snapshot.Models {
		item := billingPricingPersistedModelToListItem(model)
		if groupMultiplier != nil {
			previewForm := scaleBillingPricingLayerForm(model.SaleForm, *groupMultiplier)
			item.PreviewGroupID = filter.GroupID
			item.PreviewRateMultiplier = cloneBillingFloat64(groupMultiplier)
			item.PreviewPriceDisplay = billingPricingPreviewPriceDisplay(model, previewForm)
		}
		if matchesBillingPricingFilter(item, filter) {
			items = append(items, item)
		}
	}
	sortBillingPricingListItems(items, filter)
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
	if status := strings.TrimSpace(filter.PricingStatus); status != "" {
		expected := normalizeBillingPricingStatus(BillingPricingStatus(strings.ToLower(status)))
		if normalizeBillingPricingStatus(item.PricingStatus) != expected {
			return false
		}
	}
	return true
}

func (s *BillingCenterService) GetPricingDetails(ctx context.Context, models []string, groupID ...*int64) ([]BillingPricingSheetDetail, error) {
	if s == nil || s.modelCatalogService == nil {
		return []BillingPricingSheetDetail{}, nil
	}
	snapshot, err := s.ensureBillingPricingCatalogMigrated(ctx)
	if err != nil {
		return nil, err
	}
	var previewGroupID *int64
	if len(groupID) > 0 {
		previewGroupID = groupID[0]
	}
	_, groupMultiplier, err := s.resolveBillingPreviewGroupMultiplier(ctx, previewGroupID)
	if err != nil {
		return nil, err
	}
	items := make([]BillingPricingSheetDetail, 0, len(models))
	for _, model := range models {
		persisted, ok, _ := billingPricingSnapshotModel(snapshot, model)
		if !ok {
			return nil, infraerrors.NotFound("BILLING_MODEL_NOT_FOUND", "billing model not found")
		}
		detail := billingPricingPersistedModelToDetail(persisted)
		if groupMultiplier != nil {
			previewForm := scaleBillingPricingLayerForm(persisted.SaleForm, *groupMultiplier)
			detail.PreviewGroupID = previewGroupID
			detail.PreviewRateMultiplier = cloneBillingFloat64(groupMultiplier)
			detail.PreviewSaleForm = cloneBillingPricingLayerFormPtr(previewForm)
		}
		items = append(items, detail)
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
	snapshot, err := s.ensureBillingPricingCatalogMigrated(ctx)
	if err != nil {
		return nil, err
	}
	persisted, ok, index := billingPricingSnapshotModel(snapshot, input.Model)
	if !ok {
		return nil, infraerrors.NotFound("BILLING_MODEL_NOT_FOUND", "billing model not found")
	}
	record := billingPricingPersistedModelToRecord(persisted)
	metadata := billingPricingMetadataForPersistedModel(persisted)
	currency := normalizeModelPricingCurrency(input.Currency)
	if currency == "" {
		currency = defaultModelPricingCurrency(persisted.Currency)
	}
	form := BillingPricingLayerForm{}
	switch {
	case input.Form != nil:
		form = cloneBillingPricingLayerForm(*input.Form)
	case len(input.Items) > 0:
		form = billingPricingLayerFormFromItemsWithMetadata(metadata, input.Items)
	}
	form = normalizeBillingPricingLayerFormForLayer(form, layer)
	if err := validateBillingPricingLayerForm(form); err != nil {
		return nil, err
	}
	items := billingPricingItemsFromForm(metadata, layer, form)

	legacyPricing := effectiveFlatPricingFromForm(metadata, form)
	if err := validateFlatPricingForSave(legacyPricing); err != nil {
		return nil, err
	}

	updatedModel := cloneBillingPricingPersistedModel(persisted)
	updatedModel.Currency = currency
	editorItems := prepareBillingPriceItemsForEditor(items)
	switch layer {
	case BillingLayerOfficial:
		updatedModel.OfficialForm = cloneBillingPricingLayerForm(form)
		updatedModel.OfficialItems = cloneBillingPriceItems(editorItems)
		updatedModel.OfficialCount = len(updatedModel.OfficialItems)
	case BillingLayerSale:
		updatedModel.SaleForm = cloneBillingPricingLayerForm(form)
		updatedModel.SaleItems = cloneBillingPriceItems(editorItems)
		updatedModel.SaleCount = len(updatedModel.SaleItems)
	}
	updatedModel = cloneBillingPricingPersistedModel(updatedModel)
	snapshot.Models[index] = updatedModel
	snapshot.UpdatedAt = time.Now().UTC()
	if err := s.persistBillingPricingCatalogSnapshot(ctx, snapshot); err != nil {
		return nil, err
	}

	if err := s.modelCatalogService.ReplacePricingOverrideLayer(ctx, actor, record.model, layer == BillingLayerOfficial, legacyPricing); err != nil {
		return nil, err
	}

	rules := s.ListRules(ctx)
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
		zap.Int("model_count", 1),
		zap.String("layer", layer),
		zap.String("currency", currency),
		zap.Bool("snapshot", true),
		zap.Bool("input_supported", metadata.InputSupported),
		zap.String("output_charge_slot", metadata.OutputChargeSlot),
		zap.Bool("special_enabled", form.SpecialEnabled),
		zap.Bool("tiered_enabled", form.TieredEnabled),
		zap.Bool("multiplier_enabled", form.MultiplierEnabled),
		zap.String("multiplier_mode", string(form.MultiplierMode)),
		zap.Int("affected_item_count", len(editorItems)),
	)

	detail := billingPricingPersistedModelToDetail(snapshot.Models[index])
	if input.GroupID != nil {
		if _, groupMultiplier, previewErr := s.resolveBillingPreviewGroupMultiplier(ctx, input.GroupID); previewErr == nil && groupMultiplier != nil {
			previewForm := scaleBillingPricingLayerForm(snapshot.Models[index].SaleForm, *groupMultiplier)
			detail.PreviewGroupID = input.GroupID
			detail.PreviewRateMultiplier = cloneBillingFloat64(groupMultiplier)
			detail.PreviewSaleForm = cloneBillingPricingLayerFormPtr(previewForm)
		}
	}
	return &detail, nil
}

func billingPricingSnapshotModel(snapshot *BillingPricingCatalogSnapshot, model string) (BillingPricingPersistedModel, bool, int) {
	key := NormalizeModelCatalogModelID(model)
	if snapshot == nil || key == "" {
		return BillingPricingPersistedModel{}, false, -1
	}
	for index, item := range snapshot.Models {
		if NormalizeModelCatalogModelID(item.Model) == key {
			return cloneBillingPricingPersistedModel(item), true, index
		}
	}
	return BillingPricingPersistedModel{}, false, -1
}

const (
	billingPricingSortByDisplayName = "display_name"
	billingPricingSortByProvider    = "provider"
	billingPricingSortOrderAsc      = "asc"
	billingPricingSortOrderDesc     = "desc"
)

func normalizeBillingPricingSortBy(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case billingPricingSortByProvider:
		return billingPricingSortByProvider
	default:
		return billingPricingSortByDisplayName
	}
}

func normalizeBillingPricingSortOrder(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case billingPricingSortOrderDesc:
		return billingPricingSortOrderDesc
	default:
		return billingPricingSortOrderAsc
	}
}

func sortBillingPricingListItems(items []BillingPricingListItem, filter BillingPricingListFilter) {
	sortBy := normalizeBillingPricingSortBy(filter.SortBy)
	sortOrder := normalizeBillingPricingSortOrder(filter.SortOrder)
	sort.SliceStable(items, func(i, j int) bool {
		left := billingPricingPrimarySortKey(items[i], sortBy)
		right := billingPricingPrimarySortKey(items[j], sortBy)
		if left != right {
			if sortOrder == billingPricingSortOrderDesc {
				return left > right
			}
			return left < right
		}
		leftDisplay := billingPricingDisplaySortKey(items[i])
		rightDisplay := billingPricingDisplaySortKey(items[j])
		if leftDisplay != rightDisplay {
			return leftDisplay < rightDisplay
		}
		return strings.ToLower(items[i].Model) < strings.ToLower(items[j].Model)
	})
}

func billingPricingPrimarySortKey(item BillingPricingListItem, sortBy string) string {
	switch normalizeBillingPricingSortBy(sortBy) {
	case billingPricingSortByProvider:
		return strings.ToLower(strings.TrimSpace(FormatProviderLabel(item.Provider)))
	default:
		return billingPricingDisplaySortKey(item)
	}
}

func billingPricingDisplaySortKey(item BillingPricingListItem) string {
	displayName := strings.TrimSpace(item.DisplayName)
	if displayName != "" {
		return strings.ToLower(displayName)
	}
	return strings.ToLower(strings.TrimSpace(item.Model))
}

func (s *BillingCenterService) CopyPricingItemsOfficialToSale(ctx context.Context, actor ModelCatalogActor, models []string) ([]BillingPricingSheetDetail, error) {
	details, err := s.GetPricingDetails(ctx, models, nil)
	if err != nil {
		return nil, err
	}
	updated := make([]BillingPricingSheetDetail, 0, len(details))
	for _, detail := range details {
		form := cloneBillingPricingLayerForm(detail.OfficialForm)
		if detail.SaleForm.MultiplierEnabled {
			form.MultiplierEnabled = true
			form.MultiplierMode = detail.SaleForm.MultiplierMode
			form.SharedMultiplier = cloneBillingFloat64(detail.SaleForm.SharedMultiplier)
			form.ItemMultipliers = cloneBillingMultiplierMap(detail.SaleForm.ItemMultipliers)
		}
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
	logger.FromContext(ctx).Info(
		"billing pricing copy official to sale completed",
		zap.String("component", "service.billing_center"),
		zap.Bool("snapshot", true),
		zap.Int("model_count", len(updated)),
	)
	return updated, nil
}

func (s *BillingCenterService) ApplySaleDiscount(ctx context.Context, actor ModelCatalogActor, input BillingBulkApplyRequest) ([]BillingPricingSheetDetail, error) {
	if input.DiscountRatio <= 0 {
		return nil, infraerrors.BadRequest("BILLING_DISCOUNT_RATIO_INVALID", "discount ratio must be greater than zero")
	}
	details, err := s.GetPricingDetails(ctx, input.Models, nil)
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
	logger.FromContext(ctx).Info(
		"billing pricing discount completed",
		zap.String("component", "service.billing_center"),
		zap.Bool("snapshot", true),
		zap.Int("model_count", len(updated)),
	)
	return updated, nil
}

func effectiveFlatPricingFromForm(metadata billingPricingFormMetadata, form BillingPricingLayerForm) *ModelCatalogPricing {
	form = normalizeBillingPricingLayerFormForLayer(form, BillingLayerSale)
	pricing := &ModelCatalogPricing{}

	inputPrice := billingPricingEffectiveFieldValue(form, billingDiscountFieldInputPrice)
	outputPrice := billingPricingEffectiveFieldValue(form, billingDiscountFieldOutputPrice)
	cachePrice := billingPricingEffectiveFieldValue(form, billingDiscountFieldCachePrice)

	if metadata.InputSupported {
		pricing.InputCostPerToken = cloneBillingFloat64(inputPrice)
	}
	switch metadata.OutputChargeSlot {
	case BillingChargeSlotImageOutput:
		pricing.OutputCostPerImage = cloneBillingFloat64(outputPrice)
	case BillingChargeSlotVideoRequest:
		pricing.OutputCostPerVideoRequest = cloneBillingFloat64(outputPrice)
	default:
		pricing.OutputCostPerToken = cloneBillingFloat64(outputPrice)
	}
	if cachePrice != nil {
		pricing.CacheCreationInputTokenCost = cloneBillingFloat64(cachePrice)
		pricing.CacheReadInputTokenCost = cloneBillingFloat64(cachePrice)
		pricing.CacheCreationInputTokenCostAbove1hr = cloneBillingFloat64(cachePrice)
	}
	if form.TieredEnabled && form.TierThresholdTokens != nil {
		if metadata.InputSupported {
			if above := billingPricingEffectiveFieldValue(form, billingDiscountFieldInputPriceAboveThreshold); above != nil {
				pricing.InputTokenThreshold = cloneBillingInt(form.TierThresholdTokens)
				pricing.InputCostPerTokenAboveThreshold = cloneBillingFloat64(above)
			}
		}
		if metadata.OutputChargeSlot == BillingChargeSlotTextOutput {
			if above := billingPricingEffectiveFieldValue(form, billingDiscountFieldOutputPriceAboveThreshold); above != nil {
				pricing.OutputTokenThreshold = cloneBillingInt(form.TierThresholdTokens)
				pricing.OutputCostPerTokenAboveThreshold = cloneBillingFloat64(above)
			}
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
			Unit:              item.Unit,
			Price:             item.Price,
			FormulaSource:     item.FormulaSource,
			FormulaMultiplier: cloneBillingFloat64(item.FormulaMultiplier),
			Priority:          2500,
			Enabled:           item.Enabled,
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
