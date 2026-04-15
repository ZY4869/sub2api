package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBillingCenterService_GetPricingDetails_CompactsGeminiLegacyPricingItems(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-2.0-flash": {
			InputCostPerToken:   1e-6,
			OutputCostPerToken:  4e-6,
			LiteLLMProvider:     PlatformGemini,
			Mode:                "chat",
			SupportsServiceTier: true,
		},
	})

	details, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gemini-2.0-flash"})
	require.NoError(t, err)
	require.Len(t, details, 1)

	detail := details[0]
	require.Less(t, len(detail.OfficialItems), 20)
	for _, item := range detail.OfficialItems {
		require.NotContains(t, []string{
			BillingSurfaceOpenAICompat,
			BillingSurfaceGeminiLive,
			BillingSurfaceInteractions,
			BillingSurfaceVertexExisting,
		}, item.Surface)
		require.NotEqual(t, BillingChargeSlotAudioInput, item.ChargeSlot)
		require.NotEqual(t, BillingChargeSlotAudioOutput, item.ChargeSlot)
		require.NotEqual(t, BillingChargeSlotTextInputLongContext, item.ChargeSlot)
		require.NotEqual(t, BillingChargeSlotTextOutputLongContext, item.ChargeSlot)
	}
}

func TestBillingCenterService_GetPricingDetails_CompactsSavedGeminiMatrix(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-2.0-flash": {
			InputCostPerToken:   1e-6,
			OutputCostPerToken:  4e-6,
			LiteLLMProvider:     PlatformGemini,
			Mode:                "chat",
			SupportsServiceTier: true,
		},
	})

	sheet, err := svc.billingCenterService.GetSheet(context.Background(), "gemini-2.0-flash")
	require.NoError(t, err)
	require.NotNil(t, sheet)
	require.NotNil(t, sheet.OfficialMatrix)

	_, err = svc.UpsertBillingSheet(context.Background(), ModelCatalogActor{UserID: 7, Email: "matrix@example.com"}, UpsertModelBillingSheetInput{
		Model:  "gemini-2.0-flash",
		Layer:  BillingLayerOfficial,
		Matrix: sheet.OfficialMatrix,
	})
	require.NoError(t, err)

	details, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gemini-2.0-flash"})
	require.NoError(t, err)
	require.Len(t, details, 1)
	require.Less(t, len(details[0].OfficialItems), 20)
	for _, item := range details[0].OfficialItems {
		require.NotContains(t, []string{
			BillingSurfaceOpenAICompat,
			BillingSurfaceGeminiLive,
			BillingSurfaceInteractions,
			BillingSurfaceVertexExisting,
		}, item.Surface)
	}
}

func TestBillingCenterService_SavePricingLayer_GeminiSpecialItemsOverlayLegacyBaseline(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-2.0-flash": {
			InputCostPerToken:   1e-6,
			OutputCostPerToken:  4e-6,
			LiteLLMProvider:     PlatformGemini,
			Mode:                "chat",
			SupportsServiceTier: true,
		},
	})

	_, err := svc.billingCenterService.SavePricingLayer(context.Background(), ModelCatalogActor{UserID: 9, Email: "billing@example.com"}, UpsertBillingPricingLayerInput{
		Model: "gemini-2.0-flash",
		Layer: BillingLayerSale,
		Items: []BillingPriceItem{
			{
				ID:         "base_input",
				ChargeSlot: BillingChargeSlotTextInput,
				Unit:       BillingUnitInputToken,
				Layer:      BillingLayerSale,
				Mode:       BillingPriceItemModeBase,
				Price:      1e-6,
				Enabled:    true,
			},
			{
				ID:         "base_output",
				ChargeSlot: BillingChargeSlotTextOutput,
				Unit:       BillingUnitOutputToken,
				Layer:      BillingLayerSale,
				Mode:       BillingPriceItemModeBase,
				Price:      4e-6,
				Enabled:    true,
			},
			{
				ID:            "interactions_audio_flex",
				ChargeSlot:    BillingChargeSlotAudioInput,
				Unit:          BillingUnitInputToken,
				Layer:         BillingLayerSale,
				Mode:          BillingPriceItemModeProviderRule,
				ServiceTier:   BillingServiceTierFlex,
				Surface:       BillingSurfaceInteractions,
				OperationType: "generate_content",
				InputModality: "audio",
				Price:         9e-7,
				Enabled:       true,
			},
		},
	})
	require.NoError(t, err)

	sheet, err := svc.billingCenterService.GetSheet(context.Background(), "gemini-2.0-flash")
	require.NoError(t, err)
	require.NotNil(t, sheet)
	require.NotNil(t, sheet.SaleMatrix)

	nativeInput := geminiMatrixCell(sheet.SaleMatrix, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotTextInput)
	require.NotNil(t, nativeInput)
	require.NotNil(t, nativeInput.Price)
	require.InDelta(t, 1e-6, *nativeInput.Price, 1e-12)

	interactionAudio := geminiMatrixCell(sheet.SaleMatrix, BillingSurfaceInteractions, BillingServiceTierFlex, BillingChargeSlotAudioInput)
	require.NotNil(t, interactionAudio)
	require.NotNil(t, interactionAudio.Price)
	require.InDelta(t, 9e-7, *interactionAudio.Price, 1e-12)

	details, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gemini-2.0-flash"})
	require.NoError(t, err)
	require.Len(t, details, 1)
	require.Less(t, len(details[0].SaleItems), 20)

	var foundSpecial bool
	for _, item := range details[0].SaleItems {
		if item.Surface == BillingSurfaceInteractions && item.ChargeSlot == BillingChargeSlotAudioInput {
			foundSpecial = true
			require.InDelta(t, 9e-7, item.Price, 1e-12)
		}
	}
	require.True(t, foundSpecial)
}

func TestBillingCenterService_GetPricingDetails_CompactsOpenAIFlatPricingItems(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gpt-5.4": {
			InputCostPerToken:               2.5e-6,
			InputCostPerTokenPriority:       5e-6,
			OutputCostPerToken:              1.5e-5,
			OutputCostPerTokenPriority:      3e-5,
			CacheCreationInputTokenCost:     0,
			CacheReadInputTokenCost:         2.5e-7,
			CacheReadInputTokenCostPriority: 5e-7,
			LiteLLMProvider:                 PlatformOpenAI,
			Mode:                            "chat",
			SupportsServiceTier:             true,
			SupportsPromptCaching:           true,
		},
	})

	details, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gpt-5.4"})
	require.NoError(t, err)
	require.Len(t, details, 1)

	items := details[0].OfficialItems
	require.Len(t, items, 6)
	for _, item := range items {
		require.NotEqual(t, BillingPriceItemModeBatch, item.Mode)
		require.Greater(t, item.Price, 0.0)
		require.Empty(t, item.Surface)
		require.Empty(t, item.OperationType)
	}
}

func TestBillingCenterService_GetPricingDetails_KeepsExplicitZeroOverrideItems(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gpt-5.4": {
			ModelCatalogPricing: ModelCatalogPricing{
				CacheCreationInputTokenCost: modelCatalogFloat64Ptr(0),
			},
		},
	})
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gpt-5.4": {
			InputCostPerToken:           2.5e-6,
			OutputCostPerToken:          1.5e-5,
			CacheCreationInputTokenCost: 2.5e-6,
			LiteLLMProvider:             PlatformOpenAI,
			Mode:                        "chat",
		},
	})

	details, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gpt-5.4"})
	require.NoError(t, err)
	require.Len(t, details, 1)

	var found bool
	for _, item := range details[0].OfficialItems {
		if item.ChargeSlot == BillingChargeSlotCacheCreate {
			found = true
			require.InDelta(t, 0, item.Price, 1e-12)
		}
	}
	require.True(t, found)
}
