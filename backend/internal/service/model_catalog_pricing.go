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

func pricingFromLiteLLM(pricing *LiteLLMModelPricing) *ModelCatalogPricing {
	if pricing == nil {
		return nil
	}
	return &ModelCatalogPricing{
		InputCostPerToken:                   modelCatalogFloat64Ptr(pricing.InputCostPerToken),
		InputCostPerTokenPriority:           modelCatalogFloat64Ptr(pricing.InputCostPerTokenPriority),
		OutputCostPerToken:                  modelCatalogFloat64Ptr(pricing.OutputCostPerToken),
		OutputCostPerTokenPriority:          modelCatalogFloat64Ptr(pricing.OutputCostPerTokenPriority),
		CacheCreationInputTokenCost:         modelCatalogFloat64Ptr(pricing.CacheCreationInputTokenCost),
		CacheCreationInputTokenCostAbove1hr: modelCatalogFloat64Ptr(pricing.CacheCreationInputTokenCostAbove1hr),
		CacheReadInputTokenCost:             modelCatalogFloat64Ptr(pricing.CacheReadInputTokenCost),
		CacheReadInputTokenCostPriority:     modelCatalogFloat64Ptr(pricing.CacheReadInputTokenCostPriority),
		OutputCostPerImage:                  modelCatalogFloat64Ptr(pricing.OutputCostPerImage),
	}
}

func pricingFromBilling(pricing *ModelPricing) *ModelCatalogPricing {
	if pricing == nil {
		return nil
	}
	return &ModelCatalogPricing{
		InputCostPerToken:                   modelCatalogFloat64Ptr(pricing.InputPricePerToken),
		InputCostPerTokenPriority:           modelCatalogFloat64Ptr(pricing.InputPricePerTokenPriority),
		OutputCostPerToken:                  modelCatalogFloat64Ptr(pricing.OutputPricePerToken),
		OutputCostPerTokenPriority:          modelCatalogFloat64Ptr(pricing.OutputPricePerTokenPriority),
		CacheCreationInputTokenCost:         modelCatalogFloat64Ptr(pricing.CacheCreationPricePerToken),
		CacheCreationInputTokenCostAbove1hr: modelCatalogFloat64Ptr(pricing.CacheCreation1hPrice),
		CacheReadInputTokenCost:             modelCatalogFloat64Ptr(pricing.CacheReadPricePerToken),
		CacheReadInputTokenCostPriority:     modelCatalogFloat64Ptr(pricing.CacheReadPricePerTokenPriority),
		OutputCostPerImage:                  modelCatalogFloat64Ptr(pricing.OutputPricePerImage),
	}
}

func applyPricingOverride(base *ModelCatalogPricing, override *ModelPricingOverride) *ModelCatalogPricing {
	effective := cloneCatalogPricing(base)
	if effective == nil {
		effective = &ModelCatalogPricing{}
	}
	if override == nil {
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
	return &copy
}

func mergeCatalogPricing(target *ModelCatalogPricing, patch *ModelCatalogPricing) {
	if target == nil || patch == nil {
		return
	}
	assignFloat := func(dst **float64, src *float64) {
		if src != nil {
			*dst = modelCatalogFloat64Ptr(*src)
		}
	}
	assignFloat(&target.InputCostPerToken, patch.InputCostPerToken)
	assignFloat(&target.InputCostPerTokenPriority, patch.InputCostPerTokenPriority)
	assignFloat(&target.OutputCostPerToken, patch.OutputCostPerToken)
	assignFloat(&target.OutputCostPerTokenPriority, patch.OutputCostPerTokenPriority)
	assignFloat(&target.CacheCreationInputTokenCost, patch.CacheCreationInputTokenCost)
	assignFloat(&target.CacheCreationInputTokenCostAbove1hr, patch.CacheCreationInputTokenCostAbove1hr)
	assignFloat(&target.CacheReadInputTokenCost, patch.CacheReadInputTokenCost)
	assignFloat(&target.CacheReadInputTokenCostPriority, patch.CacheReadInputTokenCostPriority)
	assignFloat(&target.OutputCostPerImage, patch.OutputCostPerImage)
}

func pricingEmpty(pricing *ModelCatalogPricing) bool {
	return pricing == nil ||
		(pricing.InputCostPerToken == nil &&
			pricing.InputCostPerTokenPriority == nil &&
			pricing.OutputCostPerToken == nil &&
			pricing.OutputCostPerTokenPriority == nil &&
			pricing.CacheCreationInputTokenCost == nil &&
			pricing.CacheCreationInputTokenCostAbove1hr == nil &&
			pricing.CacheReadInputTokenCost == nil &&
			pricing.CacheReadInputTokenCostPriority == nil &&
			pricing.OutputCostPerImage == nil)
}

func (s *ModelCatalogService) loadPriceOverrides(ctx context.Context) map[string]*ModelPricingOverride {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelPriceOverrides)
	if err != nil || raw == "" {
		return map[string]*ModelPricingOverride{}
	}
	var overrides map[string]*ModelPricingOverride
	if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
		logger.FromContext(ctx).Warn("model catalog: invalid override json", zap.Error(err))
		return map[string]*ModelPricingOverride{}
	}
	normalized := make(map[string]*ModelPricingOverride, len(overrides))
	for model, override := range overrides {
		key := CanonicalizeModelNameForPricing(model)
		if key == "" || override == nil || pricingEmpty(&override.ModelCatalogPricing) {
			continue
		}
		normalized[key] = override
	}
	return normalized
}

func (s *ModelCatalogService) persistPriceOverrides(ctx context.Context, overrides map[string]*ModelPricingOverride) error {
	if len(overrides) == 0 {
		return s.settingRepo.Delete(ctx, SettingKeyModelPriceOverrides)
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
	return s.settingRepo.Set(ctx, SettingKeyModelPriceOverrides, string(payload))
}

func validateOverridePricing(pricing ModelCatalogPricing) error {
	values := []*float64{
		pricing.InputCostPerToken,
		pricing.InputCostPerTokenPriority,
		pricing.OutputCostPerToken,
		pricing.OutputCostPerTokenPriority,
		pricing.CacheCreationInputTokenCost,
		pricing.CacheCreationInputTokenCostAbove1hr,
		pricing.CacheReadInputTokenCost,
		pricing.CacheReadInputTokenCostPriority,
		pricing.OutputCostPerImage,
	}
	for _, value := range values {
		if value == nil {
			continue
		}
		if math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
			return infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_INVALID", "pricing override must be a non-negative number")
		}
	}
	if pricingEmpty(&pricing) {
		return infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_EMPTY", "at least one pricing field is required")
	}
	return nil
}

func modelCatalogFloat64Ptr(value float64) *float64 {
	v := value
	return &v
}
