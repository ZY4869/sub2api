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

func TestBillingCenterService_SavePricingLayer_GeminiSimpleFormClearsLegacyOverlay(t *testing.T) {
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

	inputPrice := 1e-6
	outputPrice := 4e-6
	cachePrice := 5e-7
	batchInputPrice := 8e-7
	batchOutputPrice := 1.2e-6
	batchCachePrice := 4e-7
	groundingSearch := 0.12
	tierThreshold := 200000
	inputAboveThreshold := 1.5e-6
	outputAboveThreshold := 5.5e-6

	_, err := svc.billingCenterService.SavePricingLayer(context.Background(), ModelCatalogActor{UserID: 9, Email: "billing@example.com"}, UpsertBillingPricingLayerInput{
		Model: "gemini-2.0-flash",
		Layer: BillingLayerSale,
		Form: &BillingPricingLayerForm{
			InputPrice:     &inputPrice,
			OutputPrice:    &outputPrice,
			CachePrice:     &cachePrice,
			SpecialEnabled: true,
			Special: BillingPricingSimpleSpecial{
				BatchInputPrice:  &batchInputPrice,
				BatchOutputPrice: &batchOutputPrice,
				BatchCachePrice:  &batchCachePrice,
				GroundingSearch:  &groundingSearch,
			},
			TieredEnabled:             true,
			TierThresholdTokens:       &tierThreshold,
			InputPriceAboveThreshold:  &inputAboveThreshold,
			OutputPriceAboveThreshold: &outputAboveThreshold,
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

	nativeGrounding := geminiMatrixCell(sheet.SaleMatrix, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotGroundingSearchRequest)
	require.NotNil(t, nativeGrounding)
	require.NotNil(t, nativeGrounding.Price)
	require.InDelta(t, groundingSearch, *nativeGrounding.Price, 1e-12)

	override := svc.loadSalePriceOverrides(context.Background())["gemini-2.0-flash"]
	require.NotNil(t, override)
	require.NotNil(t, override.InputTokenThreshold)
	require.NotNil(t, override.InputCostPerTokenAboveThreshold)
	require.NotNil(t, override.OutputTokenThreshold)
	require.NotNil(t, override.OutputCostPerTokenAboveThreshold)
	require.Equal(t, tierThreshold, *override.InputTokenThreshold)
	require.Equal(t, tierThreshold, *override.OutputTokenThreshold)
	require.InDelta(t, inputAboveThreshold, *override.InputCostPerTokenAboveThreshold, 1e-12)
	require.InDelta(t, outputAboveThreshold, *override.OutputCostPerTokenAboveThreshold, 1e-12)

	rules := loadBillingRulesBySetting(context.Background(), repo, SettingKeyBillingCenterRules)
	for _, rule := range rules {
		require.NotEqual(t, BillingServiceTierFlex, rule.ServiceTier)
		require.NotEqual(t, BillingServiceTierPriority, rule.ServiceTier)
		require.NotEqual(t, BillingSurfaceInteractions, rule.Surface)
	}

	details, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gemini-2.0-flash"})
	require.NoError(t, err)
	require.Len(t, details, 1)
	require.Less(t, len(details[0].SaleItems), 20)
	require.NotNil(t, details[0].SaleForm.InputPrice)
	require.NotNil(t, details[0].SaleForm.OutputPrice)
	require.NotNil(t, details[0].SaleForm.CachePrice)
	require.True(t, details[0].SaleForm.SpecialEnabled)
	require.InDelta(t, inputPrice, *details[0].SaleForm.InputPrice, 1e-12)
	require.InDelta(t, outputPrice, *details[0].SaleForm.OutputPrice, 1e-12)
	require.InDelta(t, cachePrice, *details[0].SaleForm.CachePrice, 1e-12)
	require.NotNil(t, details[0].SaleForm.Special.BatchInputPrice)
	require.NotNil(t, details[0].SaleForm.Special.BatchOutputPrice)
	require.NotNil(t, details[0].SaleForm.Special.BatchCachePrice)
	require.NotNil(t, details[0].SaleForm.Special.GroundingSearch)
	require.InDelta(t, batchInputPrice, *details[0].SaleForm.Special.BatchInputPrice, 1e-12)
	require.InDelta(t, batchOutputPrice, *details[0].SaleForm.Special.BatchOutputPrice, 1e-12)
	require.InDelta(t, batchCachePrice, *details[0].SaleForm.Special.BatchCachePrice, 1e-12)
	require.InDelta(t, groundingSearch, *details[0].SaleForm.Special.GroundingSearch, 1e-12)

	for _, item := range details[0].SaleItems {
		require.NotEqual(t, BillingServiceTierFlex, item.ServiceTier)
		require.NotEqual(t, BillingServiceTierPriority, item.ServiceTier)
		require.NotEqual(t, BillingSurfaceInteractions, item.Surface)
	}
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

func TestBillingPricingLayerFormFromItemsWithMetadata_CompressesSimpleFields(t *testing.T) {
	tierThreshold := 200000
	inputAbove := 3e-6
	outputAbove := 9e-6

	form := billingPricingLayerFormFromItemsWithMetadata(billingPricingFormMetadata{
		InputSupported:   true,
		OutputChargeSlot: BillingChargeSlotTextOutput,
	}, []BillingPriceItem{
		{
			ChargeSlot:       BillingChargeSlotTextInput,
			Mode:             BillingPriceItemModeTiered,
			Price:            1e-6,
			ThresholdTokens:  &tierThreshold,
			PriceAboveThresh: &inputAbove,
		},
		{
			ChargeSlot:       BillingChargeSlotTextOutput,
			Mode:             BillingPriceItemModeTiered,
			Price:            4e-6,
			ThresholdTokens:  &tierThreshold,
			PriceAboveThresh: &outputAbove,
		},
		{ChargeSlot: BillingChargeSlotCacheCreate, Mode: BillingPriceItemModeBase, Price: 2e-7},
		{ChargeSlot: BillingChargeSlotCacheRead, Mode: BillingPriceItemModeBase, Price: 5e-7},
		{ChargeSlot: BillingChargeSlotCacheStorageTokenHour, Mode: BillingPriceItemModeBase, Price: 3e-7},
		{ChargeSlot: BillingChargeSlotTextInput, Mode: BillingPriceItemModeBatch, BatchMode: BillingBatchModeBatch, Price: 7e-7},
		{ChargeSlot: BillingChargeSlotTextOutput, Mode: BillingPriceItemModeBatch, BatchMode: BillingBatchModeBatch, Price: 8e-7},
		{ChargeSlot: BillingChargeSlotCacheCreate, Mode: BillingPriceItemModeBatch, BatchMode: BillingBatchModeBatch, Price: 4e-7},
		{ChargeSlot: BillingChargeSlotCacheRead, Mode: BillingPriceItemModeBatch, BatchMode: BillingBatchModeBatch, Price: 6e-7},
		{ChargeSlot: BillingChargeSlotGroundingSearchRequest, Mode: BillingPriceItemModeProviderRule, Surface: BillingSurfaceGeminiNative, Price: 0.12},
		{ChargeSlot: BillingChargeSlotFileSearchRetrievalToken, Mode: BillingPriceItemModeProviderRule, Surface: BillingSurfaceGeminiNative, Price: 2.3e-6},
		{ChargeSlot: BillingChargeSlotTextInput, Mode: BillingPriceItemModeServiceTier, ServiceTier: BillingServiceTierPriority, Price: 9e-6},
		{ChargeSlot: BillingChargeSlotAudioInput, Mode: BillingPriceItemModeProviderRule, ServiceTier: BillingServiceTierFlex, Surface: BillingSurfaceInteractions, Price: 8e-7},
	})

	require.NotNil(t, form.InputPrice)
	require.NotNil(t, form.OutputPrice)
	require.NotNil(t, form.CachePrice)
	require.NotNil(t, form.TierThresholdTokens)
	require.NotNil(t, form.InputPriceAboveThreshold)
	require.NotNil(t, form.OutputPriceAboveThreshold)
	require.InDelta(t, 1e-6, *form.InputPrice, 1e-12)
	require.InDelta(t, 4e-6, *form.OutputPrice, 1e-12)
	require.InDelta(t, 5e-7, *form.CachePrice, 1e-12)
	require.Equal(t, tierThreshold, *form.TierThresholdTokens)
	require.InDelta(t, inputAbove, *form.InputPriceAboveThreshold, 1e-12)
	require.InDelta(t, outputAbove, *form.OutputPriceAboveThreshold, 1e-12)
	require.True(t, form.TieredEnabled)
	require.True(t, form.SpecialEnabled)
	require.NotNil(t, form.Special.BatchInputPrice)
	require.NotNil(t, form.Special.BatchOutputPrice)
	require.NotNil(t, form.Special.BatchCachePrice)
	require.NotNil(t, form.Special.GroundingSearch)
	require.NotNil(t, form.Special.FileSearchRetrieval)
	require.InDelta(t, 7e-7, *form.Special.BatchInputPrice, 1e-12)
	require.InDelta(t, 8e-7, *form.Special.BatchOutputPrice, 1e-12)
	require.InDelta(t, 6e-7, *form.Special.BatchCachePrice, 1e-12)
	require.InDelta(t, 0.12, *form.Special.GroundingSearch, 1e-12)
	require.InDelta(t, 2.3e-6, *form.Special.FileSearchRetrieval, 1e-12)
}

func TestBillingPricingItemsFromForm_MapsSharedCacheAndSpecialFields(t *testing.T) {
	inputPrice := 1e-6
	outputPrice := 4e-6
	cachePrice := 5e-7
	batchInputPrice := 7e-7
	batchOutputPrice := 8e-7
	batchCachePrice := 6e-7
	groundingSearch := 0.12
	fileSearchRetrieval := 2.3e-6
	tierThreshold := 200000
	inputAbove := 3e-6
	outputAbove := 9e-6

	items := billingPricingItemsFromForm(billingPricingFormMetadata{
		InputSupported:   true,
		OutputChargeSlot: BillingChargeSlotTextOutput,
	}, BillingLayerSale, BillingPricingLayerForm{
		InputPrice:     &inputPrice,
		OutputPrice:    &outputPrice,
		CachePrice:     &cachePrice,
		SpecialEnabled: true,
		Special: BillingPricingSimpleSpecial{
			BatchInputPrice:     &batchInputPrice,
			BatchOutputPrice:    &batchOutputPrice,
			BatchCachePrice:     &batchCachePrice,
			GroundingSearch:     &groundingSearch,
			FileSearchRetrieval: &fileSearchRetrieval,
		},
		TieredEnabled:             true,
		TierThresholdTokens:       &tierThreshold,
		InputPriceAboveThreshold:  &inputAbove,
		OutputPriceAboveThreshold: &outputAbove,
	})

	require.Len(t, items, 11)
	require.Equal(t, BillingLayerSale, items[0].Layer)

	requireBillingItem(t, items, BillingChargeSlotTextInput, BillingPriceItemModeTiered, "", "", "", 1e-6, &tierThreshold, &inputAbove)
	requireBillingItem(t, items, BillingChargeSlotTextOutput, BillingPriceItemModeTiered, "", "", "", 4e-6, &tierThreshold, &outputAbove)
	requireBillingItem(t, items, BillingChargeSlotCacheCreate, BillingPriceItemModeBase, "", "", "", 5e-7, nil, nil)
	requireBillingItem(t, items, BillingChargeSlotCacheRead, BillingPriceItemModeBase, "", "", "", 5e-7, nil, nil)
	requireBillingItem(t, items, BillingChargeSlotCacheStorageTokenHour, BillingPriceItemModeBase, "", "", "", 5e-7, nil, nil)
	requireBillingItem(t, items, BillingChargeSlotTextInput, BillingPriceItemModeBatch, BillingBatchModeBatch, "", "", 7e-7, nil, nil)
	requireBillingItem(t, items, BillingChargeSlotTextOutput, BillingPriceItemModeBatch, BillingBatchModeBatch, "", "", 8e-7, nil, nil)
	requireBillingItem(t, items, BillingChargeSlotCacheCreate, BillingPriceItemModeBatch, BillingBatchModeBatch, "", "", 6e-7, nil, nil)
	requireBillingItem(t, items, BillingChargeSlotCacheRead, BillingPriceItemModeBatch, BillingBatchModeBatch, "", "", 6e-7, nil, nil)
	requireBillingItem(t, items, BillingChargeSlotGroundingSearchRequest, BillingPriceItemModeProviderRule, "", "", BillingSurfaceGeminiNative, 0.12, nil, nil)
	requireBillingItem(t, items, BillingChargeSlotFileSearchRetrievalToken, BillingPriceItemModeProviderRule, "", "", BillingSurfaceGeminiNative, 2.3e-6, nil, nil)
}

func requireBillingItem(t *testing.T, items []BillingPriceItem, slot string, mode BillingPriceItemMode, batchMode string, serviceTier string, surface string, wantPrice float64, wantThreshold *int, wantAbove *float64) {
	t.Helper()
	for _, item := range items {
		if item.ChargeSlot != slot || item.Mode != mode {
			continue
		}
		if normalizeBillingActualBatchMode(item.BatchMode) != normalizeBillingActualBatchMode(batchMode) {
			continue
		}
		if normalizeBillingServiceTier(item.ServiceTier) != normalizeBillingServiceTier(serviceTier) {
			continue
		}
		if normalizeBillingSurface(item.Surface) != normalizeBillingSurface(surface) {
			continue
		}
		require.InDelta(t, wantPrice, item.Price, 1e-12)
		if wantThreshold == nil {
			require.Nil(t, item.ThresholdTokens)
		} else {
			require.NotNil(t, item.ThresholdTokens)
			require.Equal(t, *wantThreshold, *item.ThresholdTokens)
		}
		if wantAbove == nil {
			require.Nil(t, item.PriceAboveThresh)
		} else {
			require.NotNil(t, item.PriceAboveThresh)
			require.InDelta(t, *wantAbove, *item.PriceAboveThresh, 1e-12)
		}
		return
	}
	t.Fatalf("billing item not found: slot=%s mode=%s batch=%s tier=%s surface=%s", slot, mode, batchMode, serviceTier, surface)
}
