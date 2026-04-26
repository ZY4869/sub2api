package service

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type pricingRemoteClientStub struct{}

func (pricingRemoteClientStub) FetchPricingJSON(_ context.Context, _ string) ([]byte, error) {
	return nil, errors.New("remote pricing unavailable")
}

func (pricingRemoteClientStub) FetchHashText(_ context.Context, _ string) (string, error) {
	return "", errors.New("remote hash unavailable")
}

func TestParsePricingData_ParsesPriorityAndServiceTierFields(t *testing.T) {
	svc := &PricingService{}
	body := []byte(`{
		"gpt-5.4": {
			"input_cost_per_token": 0.0000025,
			"input_cost_per_token_priority": 0.000005,
			"output_cost_per_token": 0.000015,
			"output_cost_per_token_priority": 0.00003,
			"cache_creation_input_token_cost": 0.0000025,
			"cache_read_input_token_cost": 0.00000025,
			"cache_read_input_token_cost_priority": 0.0000005,
			"supports_service_tier": true,
			"supports_prompt_caching": true,
			"litellm_provider": "openai",
			"mode": "chat"
		}
	}`)

	data, err := svc.parsePricingData(body)
	require.NoError(t, err)
	pricing := data["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 5e-6, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 3e-5, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 5e-7, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestGetModelPricing_Gpt53CodexSparkUsesGpt54Pricing(t *testing.T) {
	gpt54Pricing := &LiteLLMModelPricing{InputCostPerToken: 1}

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.4": gpt54Pricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex-spark")
	require.Same(t, gpt54Pricing, got)
}

func TestGetModelPricing_OpenAIFallbackMatchedLoggedAsDebug(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	gpt54Pricing := &LiteLLMModelPricing{InputCostPerToken: 2}
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.4": gpt54Pricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex-spark")
	require.Same(t, gpt54Pricing, got)

	require.True(t, logSink.ContainsMessageAtLevel("[Pricing] OpenAI fallback matched gpt-5.3-codex-spark -> gpt-5.4", "debug"))
	require.False(t, logSink.ContainsMessageAtLevel("[Pricing] OpenAI fallback matched gpt-5.3-codex-spark -> gpt-5.4", "info"))
	require.False(t, logSink.ContainsMessageAtLevel("[Pricing] OpenAI fallback matched gpt-5.3-codex-spark -> gpt-5.4", "warn"))
}

func TestGetModelPricing_Gpt54UsesStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{},
	}

	got := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, got)
	require.InDelta(t, 2.5e-6, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.5e-5, got.OutputCostPerToken, 1e-12)
	require.InDelta(t, 2.5e-7, got.CacheReadInputTokenCost, 1e-12)
	require.Equal(t, 272000, got.LongContextInputTokenThreshold)
	require.InDelta(t, 2.0, got.LongContextInputCostMultiplier, 1e-12)
	require.InDelta(t, 1.5, got.LongContextOutputCostMultiplier, 1e-12)
}

func TestGetModelPricing_Gpt54MiniAndNanoUseStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{},
	}

	mini := svc.GetModelPricing("gpt-5.4-mini")
	require.NotNil(t, mini)
	require.InDelta(t, 7.5e-7, mini.InputCostPerToken, 1e-12)
	require.InDelta(t, 4.5e-6, mini.OutputCostPerToken, 1e-12)
	require.InDelta(t, 7.5e-8, mini.CacheReadInputTokenCost, 1e-12)
	require.Zero(t, mini.LongContextInputTokenThreshold)

	nano := svc.GetModelPricing("gpt-5.4-nano")
	require.NotNil(t, nano)
	require.InDelta(t, 2e-7, nano.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.25e-6, nano.OutputCostPerToken, 1e-12)
	require.InDelta(t, 2e-8, nano.CacheReadInputTokenCost, 1e-12)
	require.Zero(t, nano.LongContextInputTokenThreshold)
}

func TestGetModelPricing_Gpt45PreviewUsesStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{},
	}

	got := svc.GetModelPricing("gpt-4.5-preview")
	require.NotNil(t, got)
	require.InDelta(t, 7.5e-5, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.5e-4, got.OutputCostPerToken, 1e-12)
	require.InDelta(t, 3.75e-5, got.CacheReadInputTokenCost, 1e-12)
	require.True(t, got.SupportsPromptCaching)
}

func TestGetModelPricing_GptOss120bMediumFallsBackToDefaultModel(t *testing.T) {
	defaultPricing := &LiteLLMModelPricing{InputCostPerToken: 1.25e-6}
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.4": defaultPricing,
		},
	}

	got := svc.GetModelPricing("gpt-oss-120b-medium")
	require.Same(t, defaultPricing, got)
}

func TestGetModelPricing_GptOss120bMediumFallbackIsDebugOnly(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	defaultPricing := &LiteLLMModelPricing{InputCostPerToken: 1.25e-6}
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.4": defaultPricing,
		},
	}

	require.Same(t, defaultPricing, svc.GetModelPricing("gpt-oss-120b-medium"))
	require.Same(t, defaultPricing, svc.GetModelPricing("gpt-oss-120b-medium"))

	message := "[Pricing] OpenAI fallback to default model gpt-oss-120b-medium -> gpt-5.4"
	require.True(t, logSink.ContainsMessageAtLevel(message, "debug"))
	require.False(t, logSink.ContainsMessageAtLevel(message, "info"))
	require.False(t, logSink.ContainsMessageAtLevel(message, "warn"))
}

func TestGetModelPricing_Gpt54ProUsesStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{},
	}

	got := svc.GetModelPricing("gpt-5.4-pro")
	require.NotNil(t, got)
	require.InDelta(t, 3e-5, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.8e-4, got.OutputCostPerToken, 1e-12)
	require.Equal(t, 272000, got.LongContextInputTokenThreshold)
	require.InDelta(t, 2.0, got.LongContextInputCostMultiplier, 1e-12)
	require.InDelta(t, 1.5, got.LongContextOutputCostMultiplier, 1e-12)
}

func TestParsePricingData_PreservesPriorityAndServiceTierFields(t *testing.T) {
	raw := map[string]any{
		"gpt-5.4": map[string]any{
			"input_cost_per_token":                 2.5e-6,
			"input_cost_per_token_priority":        5e-6,
			"output_cost_per_token":                15e-6,
			"output_cost_per_token_priority":       30e-6,
			"cache_read_input_token_cost":          0.25e-6,
			"cache_read_input_token_cost_priority": 0.5e-6,
			"supports_service_tier":                true,
			"supports_prompt_caching":              true,
			"litellm_provider":                     "openai",
			"mode":                                 "chat",
		},
	}
	body, err := json.Marshal(raw)
	require.NoError(t, err)

	svc := &PricingService{}
	pricingMap, err := svc.parsePricingData(body)
	require.NoError(t, err)

	pricing := pricingMap["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 2.5e-6, pricing.InputCostPerToken, 1e-12)
	require.InDelta(t, 5e-6, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 15e-6, pricing.OutputCostPerToken, 1e-12)
	require.InDelta(t, 30e-6, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.25e-6, pricing.CacheReadInputTokenCost, 1e-12)
	require.InDelta(t, 0.5e-6, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestParsePricingData_PreservesServiceTierPriorityFields(t *testing.T) {
	svc := &PricingService{}
	pricingData, err := svc.parsePricingData([]byte(`{
		"gpt-5.4": {
			"input_cost_per_token": 0.0000025,
			"input_cost_per_token_priority": 0.000005,
			"output_cost_per_token": 0.000015,
			"output_cost_per_token_priority": 0.00003,
			"cache_read_input_token_cost": 0.00000025,
			"cache_read_input_token_cost_priority": 0.0000005,
			"supports_service_tier": true,
			"litellm_provider": "openai",
			"mode": "chat"
		}
	}`))
	require.NoError(t, err)

	pricing := pricingData["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 0.0000025, pricing.InputCostPerToken, 1e-12)
	require.InDelta(t, 0.000005, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.000015, pricing.OutputCostPerToken, 1e-12)
	require.InDelta(t, 0.00003, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.00000025, pricing.CacheReadInputTokenCost, 1e-12)
	require.InDelta(t, 0.0000005, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestParsePricingData_PreservesPriorityImagePrice(t *testing.T) {
	svc := &PricingService{}
	pricingData, err := svc.parsePricingData([]byte(`{
		"gemini-2.5-flash-image": {
			"input_cost_per_token": 0.0000003,
			"input_cost_per_token_priority": 0.00000054,
			"output_cost_per_image": 0.039,
			"output_cost_per_image_priority": 0.0702,
			"supports_service_tier": true,
			"litellm_provider": "vertex_ai-language-models",
			"mode": "image_generation"
		}
	}`))
	require.NoError(t, err)

	pricing := pricingData["gemini-2.5-flash-image"]
	require.NotNil(t, pricing)
	require.InDelta(t, 0.039, pricing.OutputCostPerImage, 1e-12)
	require.InDelta(t, 0.0702, pricing.OutputCostPerImagePriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestPricingService_Initialize_UsesEmbeddedFallbackWhenFallbackFileMissing(t *testing.T) {
	cfg := &config.Config{}
	cfg.Pricing.DataDir = t.TempDir()
	cfg.Pricing.FallbackFile = filepath.Join(cfg.Pricing.DataDir, "missing_fallback.json")

	svc := NewPricingService(cfg, pricingRemoteClientStub{})
	t.Cleanup(svc.Stop)

	require.NoError(t, svc.Initialize())
	require.Greater(t, len(svc.GetPricingSnapshot()), 0)

	pricing := svc.GetModelPricing("claude-3-5-haiku")
	require.NotNil(t, pricing)
	require.Greater(t, pricing.InputCostPerToken, 0.0)
}
