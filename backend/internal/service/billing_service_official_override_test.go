package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCalculateCost_OfficialPricingOverridesSaleFallback(t *testing.T) {
	svc := NewBillingService(&config.Config{}, &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"layered-model": {
				InputCostPerToken:  1e-6,
				OutputCostPerToken: 2e-6,
			},
		},
	})

	svc.ReplaceModelOfficialPriceOverrides(map[string]*ModelPricingOverride{
		"layered-model": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken: modelCatalogFloat64Ptr(5e-6),
			},
		},
	})

	cost, err := svc.CalculateCost("layered-model", UsageTokens{InputTokens: 10, OutputTokens: 10}, 1.0)
	require.NoError(t, err)
	require.InDelta(t, 10*5e-6, cost.InputCost, 1e-12)
	require.InDelta(t, 10*2e-6, cost.OutputCost, 1e-12)

	svc.ReplaceModelPriceOverrides(map[string]*ModelPricingOverride{
		"layered-model": {
			ModelCatalogPricing: ModelCatalogPricing{
				OutputCostPerToken: modelCatalogFloat64Ptr(9e-6),
			},
		},
	})

	layeredCost, err := svc.CalculateCost("layered-model", UsageTokens{InputTokens: 10, OutputTokens: 10}, 1.0)
	require.NoError(t, err)
	require.InDelta(t, 10*5e-6, layeredCost.InputCost, 1e-12)
	require.InDelta(t, 10*9e-6, layeredCost.OutputCost, 1e-12)
}
