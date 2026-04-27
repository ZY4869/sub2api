package service

import (
	"context"
	"encoding/json"
	"math"
	"sort"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func cloneModelPricingOverride(override *ModelPricingOverride) *ModelPricingOverride {
	if override == nil {
		return nil
	}
	copy := *override
	copy.USDToCNYRate = cloneBillingFloat64(override.USDToCNYRate)
	copy.FXLockedAt = cloneBillingTime(override.FXLockedAt)
	return &copy
}

func loadModelPricingOverridesBySetting(ctx context.Context, settingRepo SettingRepository, settingKey string) map[string]*ModelPricingOverride {
	if settingRepo == nil {
		return map[string]*ModelPricingOverride{}
	}
	raw, err := settingRepo.GetValue(ctx, settingKey)
	if err != nil || raw == "" {
		return map[string]*ModelPricingOverride{}
	}
	var overrides map[string]*ModelPricingOverride
	if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
		logger.FromContext(ctx).Warn("model catalog: invalid override json", zap.String("setting_key", settingKey), zap.Error(err))
		return map[string]*ModelPricingOverride{}
	}
	normalized := make(map[string]*ModelPricingOverride, len(overrides))
	for model, override := range overrides {
		key := NormalizeModelCatalogModelID(model)
		if key == "" || override == nil || pricingEmpty(&override.ModelCatalogPricing) {
			continue
		}
		normalized[key] = cloneModelPricingOverride(override)
	}
	return normalized
}

func persistModelPricingOverridesBySetting(ctx context.Context, settingRepo SettingRepository, settingKey string, overrides map[string]*ModelPricingOverride) error {
	if settingRepo == nil {
		return nil
	}
	if len(overrides) == 0 {
		return settingRepo.Delete(ctx, settingKey)
	}
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make(map[string]*ModelPricingOverride, len(keys))
	for _, key := range keys {
		ordered[key] = overrides[key]
	}
	payload, err := json.Marshal(ordered)
	if err != nil {
		return err
	}
	return settingRepo.Set(ctx, settingKey, string(payload))
}

func pricingFromLiteLLM(pricing *LiteLLMModelPricing) *ModelCatalogPricing {
	if pricing == nil {
		return nil
	}
	return &ModelCatalogPricing{
		Currency:                                 normalizeBillingCurrency(pricing.Currency),
		USDToCNYRate:                             modelCatalogFloat64Ptr(pricing.USDToCNYRate),
		FXRateDate:                               pricing.FXRateDate,
		FXLockedAt:                               cloneBillingTime(pricing.FXLockedAt),
		InputCostPerToken:                        modelCatalogFloat64Ptr(pricing.InputCostPerToken),
		InputCostPerTokenPriority:                modelCatalogFloat64Ptr(pricing.InputCostPerTokenPriority),
		InputTokenThreshold:                      modelCatalogPositiveIntPtr(pricing.InputTokenThreshold),
		InputCostPerTokenAboveThreshold:          modelCatalogFloat64Ptr(pricing.InputCostPerTokenAboveThreshold),
		InputCostPerTokenPriorityAboveThreshold:  modelCatalogFloat64Ptr(pricing.InputCostPerTokenPriorityAboveThreshold),
		OutputCostPerToken:                       modelCatalogFloat64Ptr(pricing.OutputCostPerToken),
		OutputCostPerTokenPriority:               modelCatalogFloat64Ptr(pricing.OutputCostPerTokenPriority),
		OutputTokenThreshold:                     modelCatalogPositiveIntPtr(pricing.OutputTokenThreshold),
		OutputCostPerTokenAboveThreshold:         modelCatalogFloat64Ptr(pricing.OutputCostPerTokenAboveThreshold),
		OutputCostPerTokenPriorityAboveThreshold: modelCatalogFloat64Ptr(pricing.OutputCostPerTokenPriorityAboveThreshold),
		CacheCreationInputTokenCost:              modelCatalogFloat64Ptr(pricing.CacheCreationInputTokenCost),
		CacheCreationInputTokenCostAbove1hr:      modelCatalogFloat64Ptr(pricing.CacheCreationInputTokenCostAbove1hr),
		CacheReadInputTokenCost:                  modelCatalogFloat64Ptr(pricing.CacheReadInputTokenCost),
		CacheReadInputTokenCostPriority:          modelCatalogFloat64Ptr(pricing.CacheReadInputTokenCostPriority),
		OutputCostPerImage:                       modelCatalogFloat64Ptr(pricing.OutputCostPerImage),
		OutputCostPerImagePriority:               modelCatalogFloat64Ptr(pricing.OutputCostPerImagePriority),
		OutputCostPerVideoRequest:                modelCatalogFloat64Ptr(pricing.OutputCostPerVideoRequest),
	}
}

func pricingFromBilling(pricing *ModelPricing) *ModelCatalogPricing {
	if pricing == nil {
		return nil
	}
	return &ModelCatalogPricing{
		Currency:                                 normalizeBillingCurrency(pricing.Currency),
		USDToCNYRate:                             modelCatalogFloat64Ptr(pricing.USDToCNYRate),
		FXRateDate:                               pricing.FXRateDate,
		FXLockedAt:                               cloneBillingTime(pricing.FXLockedAt),
		InputCostPerToken:                        modelCatalogFloat64Ptr(pricing.InputPricePerToken),
		InputCostPerTokenPriority:                modelCatalogFloat64Ptr(pricing.InputPricePerTokenPriority),
		InputTokenThreshold:                      modelCatalogPositiveIntPtr(pricing.InputTokenThreshold),
		InputCostPerTokenAboveThreshold:          modelCatalogFloat64Ptr(pricing.InputPricePerTokenAboveThreshold),
		InputCostPerTokenPriorityAboveThreshold:  modelCatalogFloat64Ptr(pricing.InputPricePerTokenPriorityAboveThreshold),
		OutputCostPerToken:                       modelCatalogFloat64Ptr(pricing.OutputPricePerToken),
		OutputCostPerTokenPriority:               modelCatalogFloat64Ptr(pricing.OutputPricePerTokenPriority),
		OutputTokenThreshold:                     modelCatalogPositiveIntPtr(pricing.OutputTokenThreshold),
		OutputCostPerTokenAboveThreshold:         modelCatalogFloat64Ptr(pricing.OutputPricePerTokenAboveThreshold),
		OutputCostPerTokenPriorityAboveThreshold: modelCatalogFloat64Ptr(pricing.OutputPricePerTokenPriorityAboveThreshold),
		CacheCreationInputTokenCost:              modelCatalogFloat64Ptr(pricing.CacheCreationPricePerToken),
		CacheCreationInputTokenCostAbove1hr:      modelCatalogFloat64Ptr(pricing.CacheCreation1hPrice),
		CacheReadInputTokenCost:                  modelCatalogFloat64Ptr(pricing.CacheReadPricePerToken),
		CacheReadInputTokenCostPriority:          modelCatalogFloat64Ptr(pricing.CacheReadPricePerTokenPriority),
		OutputCostPerImage:                       modelCatalogFloat64Ptr(pricing.OutputPricePerImage),
		OutputCostPerImagePriority:               modelCatalogFloat64Ptr(pricing.OutputPricePerImagePriority),
		OutputCostPerVideoRequest:                modelCatalogFloat64Ptr(pricing.OutputPricePerVideoRequest),
	}
}

func applyPricingOverride(base *ModelCatalogPricing, override *ModelPricingOverride) *ModelCatalogPricing {
	effective := cloneCatalogPricing(base)
	if effective == nil {
		effective = &ModelCatalogPricing{}
	}
	if override == nil {
		if pricingEmpty(effective) {
			return nil
		}
		return effective
	}
	mergeCatalogPricing(effective, &override.ModelCatalogPricing)
	if pricingEmpty(effective) {
		return nil
	}
	return effective
}

func cloneCatalogPricing(pricing *ModelCatalogPricing) *ModelCatalogPricing {
	if pricing == nil {
		return nil
	}
	copy := *pricing
	copy.USDToCNYRate = cloneBillingFloat64(pricing.USDToCNYRate)
	copy.FXLockedAt = cloneBillingTime(pricing.FXLockedAt)
	return &copy
}

func mergeCatalogPricing(target *ModelCatalogPricing, patch *ModelCatalogPricing) {
	if target == nil || patch == nil {
		return
	}
	if currency := normalizeBillingCurrency(patch.Currency); currency != "" {
		target.Currency = currency
	}
	if patch.USDToCNYRate != nil {
		target.USDToCNYRate = modelCatalogFloat64Ptr(*patch.USDToCNYRate)
	}
	if patch.FXRateDate != "" {
		target.FXRateDate = patch.FXRateDate
	}
	if patch.FXLockedAt != nil {
		target.FXLockedAt = cloneBillingTime(patch.FXLockedAt)
	}
	assignFloat := func(dst **float64, src *float64) {
		if src != nil {
			*dst = modelCatalogFloat64Ptr(*src)
		}
	}
	assignInt := func(dst **int, src *int) {
		if src != nil {
			*dst = modelCatalogIntPtr(*src)
		}
	}
	assignFloat(&target.InputCostPerToken, patch.InputCostPerToken)
	assignFloat(&target.InputCostPerTokenPriority, patch.InputCostPerTokenPriority)
	assignInt(&target.InputTokenThreshold, patch.InputTokenThreshold)
	assignFloat(&target.InputCostPerTokenAboveThreshold, patch.InputCostPerTokenAboveThreshold)
	assignFloat(&target.InputCostPerTokenPriorityAboveThreshold, patch.InputCostPerTokenPriorityAboveThreshold)
	assignFloat(&target.OutputCostPerToken, patch.OutputCostPerToken)
	assignFloat(&target.OutputCostPerTokenPriority, patch.OutputCostPerTokenPriority)
	assignInt(&target.OutputTokenThreshold, patch.OutputTokenThreshold)
	assignFloat(&target.OutputCostPerTokenAboveThreshold, patch.OutputCostPerTokenAboveThreshold)
	assignFloat(&target.OutputCostPerTokenPriorityAboveThreshold, patch.OutputCostPerTokenPriorityAboveThreshold)
	assignFloat(&target.CacheCreationInputTokenCost, patch.CacheCreationInputTokenCost)
	assignFloat(&target.CacheCreationInputTokenCostAbove1hr, patch.CacheCreationInputTokenCostAbove1hr)
	assignFloat(&target.CacheReadInputTokenCost, patch.CacheReadInputTokenCost)
	assignFloat(&target.CacheReadInputTokenCostPriority, patch.CacheReadInputTokenCostPriority)
	assignFloat(&target.OutputCostPerImage, patch.OutputCostPerImage)
	assignFloat(&target.OutputCostPerImagePriority, patch.OutputCostPerImagePriority)
	assignFloat(&target.OutputCostPerVideoRequest, patch.OutputCostPerVideoRequest)
}

func pricingEmpty(pricing *ModelCatalogPricing) bool {
	return pricing == nil ||
		(pricing.InputCostPerToken == nil &&
			pricing.InputCostPerTokenPriority == nil &&
			pricing.InputTokenThreshold == nil &&
			pricing.InputCostPerTokenAboveThreshold == nil &&
			pricing.InputCostPerTokenPriorityAboveThreshold == nil &&
			pricing.OutputCostPerToken == nil &&
			pricing.OutputCostPerTokenPriority == nil &&
			pricing.OutputTokenThreshold == nil &&
			pricing.OutputCostPerTokenAboveThreshold == nil &&
			pricing.OutputCostPerTokenPriorityAboveThreshold == nil &&
			pricing.CacheCreationInputTokenCost == nil &&
			pricing.CacheCreationInputTokenCostAbove1hr == nil &&
			pricing.CacheReadInputTokenCost == nil &&
			pricing.CacheReadInputTokenCostPriority == nil &&
			pricing.OutputCostPerImage == nil &&
			pricing.OutputCostPerImagePriority == nil &&
			pricing.OutputCostPerVideoRequest == nil)
}

func (s *ModelCatalogService) loadOfficialPriceOverrides(ctx context.Context) map[string]*ModelPricingOverride {
	return loadModelPricingOverridesBySetting(ctx, s.settingRepo, SettingKeyModelOfficialPriceOverrides)
}

func (s *ModelCatalogService) loadSalePriceOverrides(ctx context.Context) map[string]*ModelPricingOverride {
	return loadModelPricingOverridesBySetting(ctx, s.settingRepo, SettingKeyModelPriceOverrides)
}

func (s *ModelCatalogService) persistOfficialPriceOverrides(ctx context.Context, overrides map[string]*ModelPricingOverride) error {
	return persistModelPricingOverridesBySetting(ctx, s.settingRepo, SettingKeyModelOfficialPriceOverrides, overrides)
}

func (s *ModelCatalogService) persistSalePriceOverrides(ctx context.Context, overrides map[string]*ModelPricingOverride) error {
	return persistModelPricingOverridesBySetting(ctx, s.settingRepo, SettingKeyModelPriceOverrides, overrides)
}

func validateOverridePricing(pricing ModelCatalogPricing) error {
	values := []*float64{
		pricing.InputCostPerToken,
		pricing.InputCostPerTokenPriority,
		pricing.InputCostPerTokenAboveThreshold,
		pricing.InputCostPerTokenPriorityAboveThreshold,
		pricing.OutputCostPerToken,
		pricing.OutputCostPerTokenPriority,
		pricing.OutputCostPerTokenAboveThreshold,
		pricing.OutputCostPerTokenPriorityAboveThreshold,
		pricing.CacheCreationInputTokenCost,
		pricing.CacheCreationInputTokenCostAbove1hr,
		pricing.CacheReadInputTokenCost,
		pricing.CacheReadInputTokenCostPriority,
		pricing.OutputCostPerImage,
		pricing.OutputCostPerImagePriority,
		pricing.OutputCostPerVideoRequest,
	}
	for _, value := range values {
		if value == nil {
			continue
		}
		if math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
			return infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_INVALID", "pricing override must be a non-negative number")
		}
	}
	thresholds := []*int{pricing.InputTokenThreshold, pricing.OutputTokenThreshold}
	for _, value := range thresholds {
		if value == nil {
			continue
		}
		if *value <= 0 {
			return infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_INVALID", "token threshold must be a positive integer")
		}
	}
	if pricingEmpty(&pricing) {
		return infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_EMPTY", "at least one pricing field is required")
	}
	return nil
}

func validateTieredPricingConfiguration(pricing *ModelCatalogPricing) error {
	if pricing == nil {
		return nil
	}
	if err := validateTierRule(pricing.InputTokenThreshold, pricing.InputCostPerTokenAboveThreshold, pricing.InputCostPerTokenPriority, pricing.InputCostPerTokenPriorityAboveThreshold); err != nil {
		return err
	}
	if err := validateTierRule(pricing.OutputTokenThreshold, pricing.OutputCostPerTokenAboveThreshold, pricing.OutputCostPerTokenPriority, pricing.OutputCostPerTokenPriorityAboveThreshold); err != nil {
		return err
	}
	return nil
}

func validateTierRule(threshold *int, above *float64, priorityBase *float64, priorityAbove *float64) error {
	if threshold == nil {
		return nil
	}
	if *threshold <= 0 {
		return infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_INVALID", "token threshold must be a positive integer")
	}
	if above == nil {
		return infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_INVALID", "tiered pricing requires above-threshold price")
	}
	if priorityBase != nil && *priorityBase >= 0 && priorityAbove == nil {
		return infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_INVALID", "priority tiered pricing requires above-threshold priority price")
	}
	return nil
}

func modelCatalogFloat64Ptr(value float64) *float64 {
	v := value
	return &v
}

func modelCatalogIntPtr(value int) *int {
	v := value
	return &v
}

func modelCatalogPositiveIntPtr(value int) *int {
	if value <= 0 {
		return nil
	}
	return modelCatalogIntPtr(value)
}
