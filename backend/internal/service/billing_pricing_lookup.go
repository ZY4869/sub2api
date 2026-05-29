package service

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

// GetModelPricing returns pricing for a model.
func (s *BillingService) GetModelPricing(model string) (*ModelPricing, error) {
	// Normalize the model name to lowercase.
	model = strings.ToLower(model)
	if s.modelRegistryService != nil {
		if pricingModel, ok, err := s.modelRegistryService.ResolvePricingModel(context.Background(), model); err == nil && ok && pricingModel != "" {
			model = pricingModel
		}
	}

	// 1. Prefer dynamic pricing when available.
	if s.pricingService != nil {
		litellmPricing := s.pricingService.GetModelPricing(model)
		if litellmPricing != nil {
			// Enable 5m/1h cache breakdown only when:
			// 1. 1h pricing exists.
			// 2. 1h pricing is greater than 5m pricing to avoid under-charging on bad upstream data.
			price5m := litellmPricing.CacheCreationInputTokenCost
			price1h := litellmPricing.CacheCreationInputTokenCostAbove1hr
			enableBreakdown := price1h > 0 && price1h > price5m
			pricing := s.applyModelSpecificPricingPolicy(model, &ModelPricing{
				Currency:                                  normalizeBillingCurrency(litellmPricing.Currency),
				USDToCNYRate:                              litellmPricing.USDToCNYRate,
				FXRateDate:                                strings.TrimSpace(litellmPricing.FXRateDate),
				FXLockedAt:                                cloneBillingTime(litellmPricing.FXLockedAt),
				InputPricePerToken:                        litellmPricing.InputCostPerToken,
				InputPricePerTokenPriority:                litellmPricing.InputCostPerTokenPriority,
				InputTokenThreshold:                       litellmPricing.InputTokenThreshold,
				InputPricePerTokenAboveThreshold:          litellmPricing.InputCostPerTokenAboveThreshold,
				InputPricePerTokenPriorityAboveThreshold:  litellmPricing.InputCostPerTokenPriorityAboveThreshold,
				OutputPricePerToken:                       litellmPricing.OutputCostPerToken,
				OutputPricePerTokenPriority:               litellmPricing.OutputCostPerTokenPriority,
				OutputTokenThreshold:                      litellmPricing.OutputTokenThreshold,
				OutputPricePerTokenAboveThreshold:         litellmPricing.OutputCostPerTokenAboveThreshold,
				OutputPricePerTokenPriorityAboveThreshold: litellmPricing.OutputCostPerTokenPriorityAboveThreshold,
				OutputPricePerImage:                       litellmPricing.OutputCostPerImage,
				OutputPricePerImagePriority:               litellmPricing.OutputCostPerImagePriority,
				OutputPricePerVideoRequest:                litellmPricing.OutputCostPerVideoRequest,
				CacheCreationPricePerToken:                litellmPricing.CacheCreationInputTokenCost,
				CacheReadPricePerToken:                    litellmPricing.CacheReadInputTokenCost,
				CacheReadPricePerTokenPriority:            litellmPricing.CacheReadInputTokenCostPriority,
				CacheCreation5mPrice:                      price5m,
				CacheCreation1hPrice:                      price1h,
				SupportsCacheBreakdown:                    enableBreakdown,
				LongContextInputThreshold:                 litellmPricing.LongContextInputTokenThreshold,
				LongContextInputMultiplier:                litellmPricing.LongContextInputCostMultiplier,
				LongContextOutputMultiplier:               litellmPricing.LongContextOutputCostMultiplier,
			})
			return pricing, nil
		}
	}

	// 2. Fall back to hardcoded pricing.
	fallback := s.getFallbackPricing(model)
	if fallback != nil {
		key := CanonicalizeModelNameForPricing(model)
		if key == "" {
			key = strings.ToLower(strings.TrimSpace(model))
		}
		if key != "" {
			if _, loaded := s.fallbackPricingLogs.LoadOrStore(key, struct{}{}); !loaded {
				logger.LegacyPrintf("service.billing", "[Debug] [Billing] Using fallback pricing for model: %s", model)
			}
		}
		return s.applyModelSpecificPricingPolicy(model, fallback), nil
	}

	return nil, fmt.Errorf("pricing not found for model: %s", model)
}

func validateBillablePricingFX(model string, pricing *ModelPricing) error {
	if pricing == nil {
		return nil
	}
	if normalizeBillingCurrency(pricing.Currency) == ModelPricingCurrencyCNY &&
		(pricing.USDToCNYRate <= 0 || pricing.FXLockedAt == nil || pricing.FXLockedAt.IsZero()) {
		return infraerrors.ServiceUnavailable(
			"BILLING_FX_RATE_UNAVAILABLE",
			"USD/CNY exchange rate is unavailable",
		).WithMetadata(map[string]string{
			"model":    NormalizeModelCatalogModelID(model),
			"currency": ModelPricingCurrencyCNY,
			"fx_state": "pending",
		})
	}
	return nil
}

func (s *BillingService) getPricingForBilling(model string) (*ModelPricing, error) {
	return s.getPricingForBillingWithContext(context.Background(), model)
}

func (s *BillingService) getPricingForBillingWithContext(ctx context.Context, model string) (*ModelPricing, error) {
	pricing, err := s.GetModelPricing(model)
	if err != nil {
		return nil, err
	}
	pricing = applyModelPricingOverride(pricing, s.getModelOfficialPriceOverride(model))
	pricing = applyModelPricingOverride(pricing, s.getModelPriceOverride(model))
	return s.ensureBillablePricingFX(ctx, model, pricing)
}

func (s *BillingService) ensureBillablePricingFX(ctx context.Context, model string, pricing *ModelPricing) (*ModelPricing, error) {
	if err := validateBillablePricingFX(model, pricing); err == nil {
		return pricing, nil
	}
	if pricing == nil || normalizeBillingCurrency(pricing.Currency) != ModelPricingCurrencyCNY {
		return pricing, validateBillablePricingFX(model, pricing)
	}
	if s == nil || s.billingCenterService == nil || s.billingCenterService.modelCatalogService == nil {
		return pricing, validateBillablePricingFX(model, pricing)
	}

	rate, err := s.billingCenterService.modelCatalogService.GetUSDCNYExchangeRate(ctx, false)
	if err != nil || rate == nil || rate.Rate <= 0 {
		return pricing, validateBillablePricingFX(model, pricing)
	}

	meta := modelPricingCurrencyMetadataFromExchangeRate(rate)
	enriched := *pricing
	applyModelPricingCurrencyMetadata(&enriched, meta)
	if persistErr := s.billingCenterService.backfillModelPricingCurrencyFX(ctx, model, meta); persistErr != nil {
		logger.FromContext(ctx).Warn(
			"billing pricing runtime fx backfill persist failed",
			zap.String("component", "service.billing"),
			zap.String("request_id", billingContextRequestID(ctx)),
			zap.String("model", NormalizeModelCatalogModelID(model)),
			zap.Error(persistErr),
		)
	}
	if err := validateBillablePricingFX(model, &enriched); err != nil {
		return &enriched, err
	}
	return &enriched, nil
}

// GetPricingServiceStatus returns pricing service status.
func (s *BillingService) GetPricingServiceStatus() map[string]any {
	if s.pricingService != nil {
		return s.pricingService.GetStatus()
	}
	return map[string]any{
		"model_count":  len(s.fallbackPrices),
		"last_updated": "using fallback",
		"local_hash":   "N/A",
	}
}

// ForceUpdatePricing forces a pricing data refresh.
func (s *BillingService) ForceUpdatePricing() error {
	if s.pricingService != nil {
		return s.pricingService.ForceUpdate()
	}
	return fmt.Errorf("pricing service not initialized")
}
