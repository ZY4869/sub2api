package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetModelPricing_DatedAliasesFallBackToStableAlias(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"claude-sonnet-4-5": {InputCostPerToken: 3e-6},
			"gpt-4o":            {InputCostPerToken: 2.5e-6},
		},
	}

	tests := []struct {
		name  string
		model string
		alias string
	}{
		{name: "compact date suffix", model: "claude-sonnet-4-5-20250929", alias: "claude-sonnet-4-5"},
		{name: "date with version suffix", model: "claude-sonnet-4-5-20250929-v1:0", alias: "claude-sonnet-4-5"},
		{name: "hyphenated date suffix", model: "gpt-4o-2024-11-20", alias: "gpt-4o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Same(t, svc.pricingData[tt.alias], svc.GetModelPricing(tt.model))
		})
	}
}
