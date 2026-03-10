package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPricingService_ParsePricingDataSupportsLegacy200KThresholdFields(t *testing.T) {
	svc := NewPricingService(&config.Config{}, nil)
	payload := []byte(`{
    "claude-3-5-haiku-20241022": {
      "input_cost_per_token": 0.000001,
      "input_cost_per_token_above_200k_tokens": 0.000002,
      "input_cost_per_token_above_200k_tokens_priority": 0.000003,
      "output_cost_per_token": 0.000004,
      "output_cost_per_token_above_200k_tokens": 0.000005,
      "output_cost_per_token_above_200k_tokens_priority": 0.000006
    }
  }`)

	pricingData, err := svc.parsePricingData(payload)
	require.NoError(t, err)

	pricing := pricingData["claude-3-5-haiku-20241022"]
	require.NotNil(t, pricing)
	require.Equal(t, 200000, pricing.InputTokenThreshold)
	require.Equal(t, 200000, pricing.OutputTokenThreshold)
	require.InDelta(t, 0.000002, pricing.InputCostPerTokenAboveThreshold, 1e-12)
	require.InDelta(t, 0.000003, pricing.InputCostPerTokenPriorityAboveThreshold, 1e-12)
	require.InDelta(t, 0.000005, pricing.OutputCostPerTokenAboveThreshold, 1e-12)
	require.InDelta(t, 0.000006, pricing.OutputCostPerTokenPriorityAboveThreshold, 1e-12)
}
