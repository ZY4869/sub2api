package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func billingLineBySlot(lines []BillingSimulationLine, slot string) *BillingSimulationLine {
	for i := range lines {
		if lines[i].ChargeSlot == slot {
			return &lines[i]
		}
	}
	return nil
}

func TestBillingCenterService_SimulateUsesCanonicalMatrixRules(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-3-pro-preview": {
			InputCostPerToken:           2e-6,
			OutputCostPerToken:          7e-6,
			CacheCreationInputTokenCost: 1e-6,
			CacheReadInputTokenCost:     0.2e-6,
			LiteLLMProvider:             PlatformGemini,
			Mode:                        "chat",
			SupportsServiceTier:         true,
		},
	})

	matrix := newGeminiBillingMatrix()
	price := 4e-6
	row := geminiMatrixRow(matrix, BillingSurfaceGeminiNative, BillingServiceTierStandard)
	require.NotNil(t, row)
	row.Slots[BillingChargeSlotTextInput] = GeminiBillingMatrixCell{Price: &price}

	_, err := svc.UpsertBillingSheet(context.Background(), ModelCatalogActor{UserID: 5, Email: "sim@example.com"}, UpsertModelBillingSheetInput{
		Model:  "gemini-3-pro",
		Layer:  BillingLayerSale,
		Matrix: matrix,
	})
	require.NoError(t, err)

	result, err := svc.billingCenterService.Simulate(context.Background(), BillingSimulationInput{
		Provider:      BillingRuleProviderGemini,
		Layer:         BillingLayerSale,
		Model:         "gemini-3-pro-preview",
		Surface:       BillingSurfaceGeminiNative,
		OperationType: "generate_content",
		Charges: BillingSimulationCharges{
			TextInputTokens: 1000,
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Lines, 1)
	require.Len(t, result.MatchedRules, 1)
	require.Nil(t, result.Fallback)
	require.Equal(t, BillingChargeSlotTextInput, result.Lines[0].ChargeSlot)
	require.Equal(t, "gemini_matrix__sale__gemini-3-pro__native__standard__text_input", result.Lines[0].RuleID)
	require.InDelta(t, 0.004, result.TotalCost, 1e-12)
}

func TestBillingCenterService_SimulateAndRuntimeShareLegacyFallback(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-3-pro-preview": {
			InputCostPerToken:           2e-6,
			OutputCostPerToken:          7e-6,
			OutputCostPerImage:          0.04,
			CacheCreationInputTokenCost: 1e-6,
			CacheReadInputTokenCost:     0.2e-6,
			LiteLLMProvider:             PlatformGemini,
			Mode:                        "chat",
			SupportsServiceTier:         true,
		},
	})

	charges := BillingSimulationCharges{
		TextInputTokens:  2000,
		TextOutputTokens: 500,
		ImageOutputs:     2,
	}
	simulated, err := svc.billingCenterService.Simulate(context.Background(), BillingSimulationInput{
		Provider:      BillingRuleProviderGemini,
		Layer:         BillingLayerSale,
		Model:         "gemini-3-pro-preview",
		Surface:       BillingSurfaceGeminiNative,
		OperationType: "generate_content",
		Charges:       charges,
	})
	require.NoError(t, err)

	runtimeResult, err := svc.billingCenterService.CalculateGeminiCost(context.Background(), GeminiBillingCalculationInput{
		Model:           "gemini-3-pro-preview",
		InboundEndpoint: "/v1beta/models/gemini-3-pro-preview:generateContent",
		Charges:         charges,
	})
	require.NoError(t, err)

	require.NotNil(t, simulated.Fallback)
	require.NotNil(t, runtimeResult.Fallback)
	require.Equal(t, "legacy_model_pricing", simulated.Fallback.Policy)
	require.Equal(t, simulated.Fallback.Policy, runtimeResult.Fallback.Policy)
	require.Equal(t, simulated.Fallback.Reason, runtimeResult.Fallback.Reason)
	require.Equal(t, simulated.Fallback.CostLines, runtimeResult.Fallback.CostLines)
	require.InDelta(t, simulated.TotalCost, runtimeResult.TotalCost, 1e-12)
	require.InDelta(t, simulated.ActualCost, runtimeResult.ActualCost, 1e-12)
}

func TestBillingCenterService_LegacyFallbackSupportsLongContextCacheStorageAndAudio(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-2.5-pro": {
			InputCostPerToken:                   2e-6,
			InputTokenThreshold:                 200000,
			InputCostPerTokenAboveThreshold:     5e-6,
			OutputCostPerToken:                  7e-6,
			OutputTokenThreshold:                200,
			OutputCostPerTokenAboveThreshold:    9e-6,
			CacheCreationInputTokenCost:         1e-6,
			CacheCreationInputTokenCostAbove1hr: 6e-6,
			CacheReadInputTokenCost:             0.2e-6,
			LongContextInputTokenThreshold:      200000,
			LongContextInputCostMultiplier:      2.0,
			LongContextOutputCostMultiplier:     1.5,
			LiteLLMProvider:                     PlatformGemini,
			Mode:                                "chat",
			SupportsServiceTier:                 true,
		},
	})

	result, err := svc.billingCenterService.Simulate(context.Background(), BillingSimulationInput{
		Provider:      BillingRuleProviderGemini,
		Layer:         BillingLayerSale,
		Model:         "gemini-2.5-pro",
		Surface:       BillingSurfaceGeminiNative,
		OperationType: "generate_content",
		Charges: BillingSimulationCharges{
			TextInputTokens:           250000,
			TextOutputTokens:          1000,
			AudioInputTokens:          100,
			AudioOutputTokens:         50,
			CacheReadTokens:           1000,
			CacheStorageTokenHours:    10,
			FileSearchEmbeddingTokens: 123,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result.Fallback)
	require.True(t, result.Fallback.Applied)

	textInput := billingLineBySlot(result.Fallback.CostLines, BillingChargeSlotTextInputLongContext)
	require.NotNil(t, textInput)
	require.InDelta(t, 10e-6, textInput.Price, 1e-12)

	textOutput := billingLineBySlot(result.Fallback.CostLines, BillingChargeSlotTextOutputLongContext)
	require.NotNil(t, textOutput)
	require.InDelta(t, 13.5e-6, textOutput.Price, 1e-12)

	audioInput := billingLineBySlot(result.Fallback.CostLines, BillingChargeSlotAudioInput)
	require.NotNil(t, audioInput)
	require.InDelta(t, 5e-6, audioInput.Price, 1e-12)

	audioOutput := billingLineBySlot(result.Fallback.CostLines, BillingChargeSlotAudioOutput)
	require.NotNil(t, audioOutput)
	require.InDelta(t, 9e-6, audioOutput.Price, 1e-12)

	cacheStorage := billingLineBySlot(result.Fallback.CostLines, BillingChargeSlotCacheStorageTokenHour)
	require.NotNil(t, cacheStorage)
	require.InDelta(t, 6e-6, cacheStorage.Price, 1e-12)

	require.Len(t, result.UnmatchedDemands, 1)
	require.Equal(t, BillingChargeSlotFileSearchEmbeddingToken, result.UnmatchedDemands[0].ChargeSlot)
}

func TestBillingCenterService_LegacyFallbackMissesUnsupportedOnlyCharges(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-2.5-pro": {
			InputCostPerToken:       2e-6,
			OutputCostPerToken:      7e-6,
			CacheReadInputTokenCost: 0.2e-6,
			LiteLLMProvider:         PlatformGemini,
			Mode:                    "chat",
		},
	})

	result, err := svc.billingCenterService.Simulate(context.Background(), BillingSimulationInput{
		Provider:      BillingRuleProviderGemini,
		Layer:         BillingLayerSale,
		Model:         "gemini-2.5-pro",
		Surface:       BillingSurfaceGeminiNative,
		OperationType: "generate_content",
		Charges: BillingSimulationCharges{
			FileSearchEmbeddingTokens: 100,
			GroundingMapsQueries:      2,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result.Fallback)
	require.False(t, result.Fallback.Applied)
	require.Equal(t, "legacy_pricing_no_supported_charges", result.Fallback.Reason)
	require.Empty(t, result.Fallback.CostLines)
	require.Zero(t, result.TotalCost)
	require.Len(t, result.UnmatchedDemands, 2)
}
