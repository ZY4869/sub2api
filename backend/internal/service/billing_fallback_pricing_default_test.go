package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBillingFallbackPricingChineseProviderFamilies(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)

	tests := []struct {
		model      string
		wantInput  float64
		wantOutput float64
	}{
		{"deepseek-v4-flash", 1.4e-7, 2.8e-7},
		{"deepseek-v4-pro", 4.35e-7, 8.7e-7},
		{"deepseek-chat", 1.4e-7, 2.8e-7},
		{"doubao-1.5-thinking-pro", 8e-7, 2e-6},
		{"kimi-latest", 2e-6, 1e-5},
		{"minimax-m1", 1e-6, 8e-6},
		{"glm-4.6", 5e-7, 5e-7},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			pricing, err := svc.GetModelPricing(tt.model)
			require.NoError(t, err)
			require.NotNil(t, pricing)
			require.InDelta(t, tt.wantInput, pricing.InputPricePerToken, 1e-12)
			require.InDelta(t, tt.wantOutput, pricing.OutputPricePerToken, 1e-12)
		})
	}
}
