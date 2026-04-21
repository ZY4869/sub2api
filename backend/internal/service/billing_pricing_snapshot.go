package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

type BillingPricingCatalogSnapshot struct {
	UpdatedAt time.Time                      `json:"updated_at"`
	Models    []BillingPricingPersistedModel `json:"models"`
}

type BillingPricingPersistedModel struct {
	Model                           string                     `json:"model"`
	CanonicalModelID                string                     `json:"canonical_model_id,omitempty"`
	PricingLookupModelID            string                     `json:"pricing_lookup_model_id,omitempty"`
	DisplayName                     string                     `json:"display_name,omitempty"`
	Provider                        string                     `json:"provider,omitempty"`
	Mode                            string                     `json:"mode,omitempty"`
	Currency                        string                     `json:"currency"`
	PricingStatus                   BillingPricingStatus       `json:"pricing_status"`
	PricingWarnings                 []string                   `json:"pricing_warnings,omitempty"`
	InputSupported                  bool                       `json:"input_supported"`
	OutputChargeSlot                string                     `json:"output_charge_slot,omitempty"`
	SupportsPromptCaching           bool                       `json:"supports_prompt_caching"`
	SupportsServiceTier             bool                       `json:"supports_service_tier"`
	LongContextInputTokenThreshold  int                        `json:"long_context_input_token_threshold,omitempty"`
	LongContextInputCostMultiplier  float64                    `json:"long_context_input_cost_multiplier,omitempty"`
	LongContextOutputCostMultiplier float64                    `json:"long_context_output_cost_multiplier,omitempty"`
	Capabilities                    BillingPricingCapabilities `json:"capabilities"`
	OfficialForm                    BillingPricingLayerForm    `json:"official_form"`
	SaleForm                        BillingPricingLayerForm    `json:"sale_form"`
	OfficialItems                   []BillingPriceItem         `json:"official_items,omitempty"`
	SaleItems                       []BillingPriceItem         `json:"sale_items,omitempty"`
	OfficialCount                   int                        `json:"official_count"`
	SaleCount                       int                        `json:"sale_count"`
}

type BillingPricingRefreshResult struct {
	UpdatedAt     time.Time `json:"updated_at"`
	TotalModels   int       `json:"total_models"`
	ProviderCount int       `json:"provider_count"`
}

func cloneBillingPricingCatalogSnapshot(snapshot *BillingPricingCatalogSnapshot) *BillingPricingCatalogSnapshot {
	if snapshot == nil {
		return nil
	}
	cloned := &BillingPricingCatalogSnapshot{
		UpdatedAt: snapshot.UpdatedAt,
		Models:    make([]BillingPricingPersistedModel, 0, len(snapshot.Models)),
	}
	for _, model := range snapshot.Models {
		cloned.Models = append(cloned.Models, cloneBillingPricingPersistedModel(model))
	}
	return cloned
}

func cloneBillingPricingPersistedModel(model BillingPricingPersistedModel) BillingPricingPersistedModel {
	cloned := model
	cloned.Model = NormalizeModelCatalogModelID(model.Model)
	cloned.CanonicalModelID = CanonicalizeModelNameForPricing(model.CanonicalModelID)
	cloned.PricingLookupModelID = CanonicalizeModelNameForPricing(model.PricingLookupModelID)
	cloned.Provider = NormalizeModelProvider(model.Provider)
	cloned.Mode = strings.TrimSpace(strings.ToLower(model.Mode))
	cloned.Currency = defaultModelPricingCurrency(model.Currency)
	cloned.PricingStatus = normalizeBillingPricingStatus(model.PricingStatus)
	cloned.PricingWarnings = compactStrings(model.PricingWarnings)
	cloned.OutputChargeSlot = billingDefaultOutputChargeSlot(defaultString(model.OutputChargeSlot, model.Mode))
	cloned.OfficialForm = cloneBillingPricingLayerForm(model.OfficialForm)
	cloned.SaleForm = cloneBillingPricingLayerForm(model.SaleForm)
	cloned.OfficialItems = cloneBillingPriceItems(model.OfficialItems)
	cloned.SaleItems = cloneBillingPriceItems(model.SaleItems)
	metadata := billingPricingMetadataForPersistedModel(cloned)
	if len(cloned.OfficialItems) > 0 {
		cloned.OfficialCount = len(cloned.OfficialItems)
	} else {
		cloned.OfficialCount = billingPricingFormItemCount(metadata, cloned.OfficialForm)
	}
	if len(cloned.SaleItems) > 0 {
		cloned.SaleCount = len(cloned.SaleItems)
	} else {
		cloned.SaleCount = billingPricingFormItemCount(metadata, cloned.SaleForm)
	}
	return cloned
}

func cloneBillingPriceItems(items []BillingPriceItem) []BillingPriceItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]BillingPriceItem, 0, len(items))
	for _, item := range items {
		copy := item
		copy.ThresholdTokens = cloneBillingInt(item.ThresholdTokens)
		copy.PriceAboveThresh = cloneBillingFloat64(item.PriceAboveThresh)
		copy.FormulaMultiplier = cloneBillingFloat64(item.FormulaMultiplier)
		cloned = append(cloned, copy)
	}
	return cloned
}

func billingPricingMetadataForPersistedModel(model BillingPricingPersistedModel) billingPricingFormMetadata {
	outputSlot := strings.TrimSpace(model.OutputChargeSlot)
	if outputSlot == "" {
		outputSlot = billingDefaultOutputChargeSlot(model.Mode)
	}
	return billingPricingFormMetadata{
		InputSupported:   model.InputSupported,
		OutputChargeSlot: outputSlot,
	}
}

func billingPricingFormItemCount(metadata billingPricingFormMetadata, form BillingPricingLayerForm) int {
	items := dedupeBillingPriceItems(billingPricingItemsFromForm(metadata, BillingLayerSale, form))
	return len(items)
}

func billingPricingPersistedModelFromRecord(record *modelCatalogRecord, rules []BillingRule) BillingPricingPersistedModel {
	officialItems := pricingItemsForRecord(record, BillingLayerOfficial, rules)
	saleItems := pricingItemsForRecord(record, BillingLayerSale, rules)
	combinedItems := append(append([]BillingPriceItem(nil), officialItems...), saleItems...)
	metadata := billingPricingMetadataForRecord(record, combinedItems)
	model := BillingPricingPersistedModel{
		Model:                           NormalizeModelCatalogModelID(record.model),
		CanonicalModelID:                record.canonicalModelID,
		PricingLookupModelID:            record.pricingLookupModelID,
		DisplayName:                     record.displayName,
		Provider:                        record.provider,
		Mode:                            record.mode,
		Currency:                        defaultModelPricingCurrency(record.pricingCurrency),
		PricingStatus:                   billingPricingStatusForRecord(record),
		PricingWarnings:                 billingPricingWarningsForRecord(record),
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
	}
	return cloneBillingPricingPersistedModel(model)
}

func billingPricingPersistedModelToDetail(model BillingPricingPersistedModel) BillingPricingSheetDetail {
	cloned := cloneBillingPricingPersistedModel(model)
	metadata := billingPricingMetadataForPersistedModel(cloned)
	officialItems := cloneBillingPriceItems(cloned.OfficialItems)
	if len(officialItems) == 0 {
		officialItems = prepareBillingPriceItemsForEditor(
			billingPricingItemsFromForm(metadata, BillingLayerOfficial, cloned.OfficialForm),
		)
	}
	saleItems := cloneBillingPriceItems(cloned.SaleItems)
	if len(saleItems) == 0 {
		saleItems = prepareBillingPriceItemsForEditor(
			billingPricingItemsFromForm(metadata, BillingLayerSale, cloned.SaleForm),
		)
	}
	return BillingPricingSheetDetail{
		Model:                           cloned.Model,
		DisplayName:                     cloned.DisplayName,
		Provider:                        cloned.Provider,
		Mode:                            cloned.Mode,
		Currency:                        cloned.Currency,
		PricingStatus:                   cloned.PricingStatus,
		PricingWarnings:                 compactStrings(cloned.PricingWarnings),
		InputSupported:                  cloned.InputSupported,
		OutputChargeSlot:                cloned.OutputChargeSlot,
		SupportsPromptCaching:           cloned.SupportsPromptCaching,
		SupportsServiceTier:             cloned.SupportsServiceTier,
		LongContextInputTokenThreshold:  cloned.LongContextInputTokenThreshold,
		LongContextInputCostMultiplier:  cloned.LongContextInputCostMultiplier,
		LongContextOutputCostMultiplier: cloned.LongContextOutputCostMultiplier,
		Capabilities:                    cloned.Capabilities,
		OfficialForm:                    cloned.OfficialForm,
		SaleForm:                        cloned.SaleForm,
		PreviewSaleForm:                 nil,
		OfficialItems:                   officialItems,
		SaleItems:                       saleItems,
	}
}

func billingPricingPersistedModelToListItem(model BillingPricingPersistedModel) BillingPricingListItem {
	cloned := cloneBillingPricingPersistedModel(model)
	return BillingPricingListItem{
		Model:           cloned.Model,
		DisplayName:     cloned.DisplayName,
		Provider:        cloned.Provider,
		Mode:            cloned.Mode,
		Currency:        cloned.Currency,
		PriceItemCount:  cloned.OfficialCount + cloned.SaleCount,
		OfficialCount:   cloned.OfficialCount,
		SaleCount:       cloned.SaleCount,
		PricingStatus:   cloned.PricingStatus,
		PricingWarnings: compactStrings(cloned.PricingWarnings),
		Capabilities:    cloned.Capabilities,
	}
}

func billingPricingPersistedModelToRecord(model BillingPricingPersistedModel) *modelCatalogRecord {
	cloned := cloneBillingPricingPersistedModel(model)
	return &modelCatalogRecord{
		model:                           cloned.Model,
		canonicalModelID:                cloned.CanonicalModelID,
		pricingLookupModelID:            cloned.PricingLookupModelID,
		displayName:                     cloned.DisplayName,
		provider:                        cloned.Provider,
		mode:                            cloned.Mode,
		pricingCurrency:                 cloned.Currency,
		supportsPromptCaching:           cloned.SupportsPromptCaching,
		supportsServiceTier:             cloned.SupportsServiceTier,
		longContextInputTokenThreshold:  cloned.LongContextInputTokenThreshold,
		longContextInputCostMultiplier:  cloned.LongContextInputCostMultiplier,
		longContextOutputCostMultiplier: cloned.LongContextOutputCostMultiplier,
	}
}

func sortBillingPricingPersistedModels(models []BillingPricingPersistedModel) {
	sort.SliceStable(models, func(i, j int) bool {
		if models[i].Model == models[j].Model {
			return models[i].DisplayName < models[j].DisplayName
		}
		return models[i].Model < models[j].Model
	})
}

func loadBillingPricingCatalogSnapshotBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
) *BillingPricingCatalogSnapshot {
	if settingRepo == nil {
		return nil
	}
	raw, err := settingRepo.GetValue(ctx, settingKey)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var snapshot BillingPricingCatalogSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		logger.FromContext(ctx).Warn(
			"billing pricing: invalid catalog snapshot json",
			zap.String("setting_key", settingKey),
			zap.Error(err),
		)
		return nil
	}
	normalized := cloneBillingPricingCatalogSnapshot(&snapshot)
	if normalized == nil {
		return nil
	}
	sortBillingPricingPersistedModels(normalized.Models)
	return normalized
}

func persistBillingPricingCatalogSnapshotBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
	snapshot *BillingPricingCatalogSnapshot,
) error {
	if settingRepo == nil {
		return nil
	}
	if snapshot == nil || len(snapshot.Models) == 0 {
		return settingRepo.Delete(ctx, settingKey)
	}
	normalized := cloneBillingPricingCatalogSnapshot(snapshot)
	sortBillingPricingPersistedModels(normalized.Models)
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return settingRepo.Set(ctx, settingKey, string(payload))
}

func (s *BillingCenterService) loadBillingPricingCatalogSnapshot(ctx context.Context) *BillingPricingCatalogSnapshot {
	if s == nil {
		return nil
	}
	return loadBillingPricingCatalogSnapshotBySetting(ctx, s.settingRepo, SettingKeyBillingPricingCatalogSnapshot)
}

func (s *BillingCenterService) persistBillingPricingCatalogSnapshot(ctx context.Context, snapshot *BillingPricingCatalogSnapshot) error {
	if s == nil {
		return nil
	}
	return persistBillingPricingCatalogSnapshotBySetting(ctx, s.settingRepo, SettingKeyBillingPricingCatalogSnapshot, snapshot)
}

func (s *BillingCenterService) buildBillingPricingCatalogSnapshot(ctx context.Context) (*BillingPricingCatalogSnapshot, error) {
	if s == nil || s.modelCatalogService == nil {
		return &BillingPricingCatalogSnapshot{UpdatedAt: time.Now().UTC(), Models: []BillingPricingPersistedModel{}}, nil
	}
	records, err := s.modelCatalogService.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	rules := s.ListRules(ctx)
	models := make([]BillingPricingPersistedModel, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		models = append(models, billingPricingPersistedModelFromRecord(record, rules))
	}
	sortBillingPricingPersistedModels(models)
	summary := summarizeBillingPricingStatuses(models)
	logger.FromContext(ctx).Info(
		"billing pricing catalog audit summary",
		zap.String("component", "service.billing_center"),
		zap.Int("model_count", len(models)),
		zap.Int("pricing_ok_count", summary.ok),
		zap.Int("pricing_fallback_count", summary.fallback),
		zap.Int("pricing_conflict_count", summary.conflict),
		zap.Int("pricing_missing_count", summary.missing),
	)
	return &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Now().UTC(),
		Models:    models,
	}, nil
}

func (s *BillingCenterService) ensureBillingPricingCatalogMigrated(ctx context.Context) (*BillingPricingCatalogSnapshot, error) {
	if snapshot := s.loadBillingPricingCatalogSnapshot(ctx); snapshot != nil && len(snapshot.Models) > 0 {
		if !billingPricingSnapshotNeedsStatusMigration(snapshot) {
			return snapshot, nil
		}
		log := logger.FromContext(ctx)
		log.Info("billing pricing catalog snapshot status migration started", zap.String("component", "service.billing_center"))
		baseline, err := s.buildBillingPricingCatalogSnapshot(ctx)
		if err != nil {
			log.Warn("billing pricing catalog snapshot status migration failed", zap.String("component", "service.billing_center"), zap.Error(err))
			return nil, err
		}
		merged := mergeBillingPricingCatalogSnapshots(snapshot, baseline)
		if err := s.persistBillingPricingCatalogSnapshot(ctx, merged); err != nil {
			log.Warn("billing pricing catalog snapshot status persist failed", zap.String("component", "service.billing_center"), zap.Error(err))
			return nil, err
		}
		log.Info("billing pricing catalog snapshot status migration completed", zap.String("component", "service.billing_center"), zap.Int("model_count", len(merged.Models)))
		return merged, nil
	}
	log := logger.FromContext(ctx)
	log.Info("billing pricing catalog snapshot migration started", zap.String("component", "service.billing_center"))
	snapshot, err := s.buildBillingPricingCatalogSnapshot(ctx)
	if err != nil {
		log.Warn("billing pricing catalog snapshot migration failed", zap.String("component", "service.billing_center"), zap.Error(err))
		return nil, err
	}
	if err := s.persistBillingPricingCatalogSnapshot(ctx, snapshot); err != nil {
		log.Warn("billing pricing catalog snapshot persist failed", zap.String("component", "service.billing_center"), zap.Error(err))
		return nil, err
	}
	log.Info(
		"billing pricing catalog snapshot migration completed",
		zap.String("component", "service.billing_center"),
		zap.Int("model_count", len(snapshot.Models)),
	)
	return snapshot, nil
}

func billingPricingSnapshotProviderCount(models []BillingPricingPersistedModel) int {
	seen := map[string]struct{}{}
	for _, model := range models {
		provider := NormalizeModelProvider(model.Provider)
		if provider == "" {
			provider = "unknown"
		}
		seen[provider] = struct{}{}
	}
	return len(seen)
}

func mergeBillingPricingCatalogSnapshots(
	existing *BillingPricingCatalogSnapshot,
	baseline *BillingPricingCatalogSnapshot,
) *BillingPricingCatalogSnapshot {
	if baseline == nil {
		return cloneBillingPricingCatalogSnapshot(existing)
	}
	existingMap := make(map[string]BillingPricingPersistedModel, len(baseline.Models))
	if existing != nil {
		for _, model := range existing.Models {
			existingMap[NormalizeModelCatalogModelID(model.Model)] = cloneBillingPricingPersistedModel(model)
		}
	}
	merged := &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Now().UTC(),
		Models:    make([]BillingPricingPersistedModel, 0, len(baseline.Models)+len(existingMap)),
	}
	seen := map[string]struct{}{}
	for _, baselineModel := range baseline.Models {
		next := cloneBillingPricingPersistedModel(baselineModel)
		key := NormalizeModelCatalogModelID(next.Model)
		if existingModel, ok := existingMap[key]; ok {
			next.Currency = defaultModelPricingCurrency(existingModel.Currency)
			next.OfficialForm = cloneBillingPricingLayerForm(existingModel.OfficialForm)
			next.SaleForm = cloneBillingPricingLayerForm(existingModel.SaleForm)
			next.OfficialItems = cloneBillingPriceItems(existingModel.OfficialItems)
			next.SaleItems = cloneBillingPriceItems(existingModel.SaleItems)
			next = cloneBillingPricingPersistedModel(next)
		}
		seen[key] = struct{}{}
		merged.Models = append(merged.Models, next)
	}
	if existing != nil {
		for _, model := range existing.Models {
			key := NormalizeModelCatalogModelID(model.Model)
			if _, ok := seen[key]; ok {
				continue
			}
			merged.Models = append(merged.Models, cloneBillingPricingPersistedModel(model))
		}
	}
	sortBillingPricingPersistedModels(merged.Models)
	return merged
}

func (s *BillingCenterService) RefreshPricingCatalog(ctx context.Context) (*BillingPricingRefreshResult, error) {
	log := logger.FromContext(ctx)
	log.Info("billing pricing catalog refresh started", zap.String("component", "service.billing_center"))
	existing, err := s.ensureBillingPricingCatalogMigrated(ctx)
	if err != nil {
		log.Warn("billing pricing catalog refresh aborted", zap.String("component", "service.billing_center"), zap.Error(err))
		return nil, err
	}
	baseline, err := s.buildBillingPricingCatalogSnapshot(ctx)
	if err != nil {
		log.Warn("billing pricing catalog refresh failed", zap.String("component", "service.billing_center"), zap.Error(err))
		return nil, err
	}
	merged := mergeBillingPricingCatalogSnapshots(existing, baseline)
	if err := s.persistBillingPricingCatalogSnapshot(ctx, merged); err != nil {
		log.Warn("billing pricing catalog refresh persist failed", zap.String("component", "service.billing_center"), zap.Error(err))
		return nil, err
	}
	result := &BillingPricingRefreshResult{
		UpdatedAt:     merged.UpdatedAt,
		TotalModels:   len(merged.Models),
		ProviderCount: billingPricingSnapshotProviderCount(merged.Models),
	}
	log.Info(
		"billing pricing catalog refresh completed",
		zap.String("component", "service.billing_center"),
		zap.Int("model_count", result.TotalModels),
		zap.Int("provider_count", result.ProviderCount),
	)
	return result, nil
}
