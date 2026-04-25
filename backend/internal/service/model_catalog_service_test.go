package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type modelCatalogSettingRepoStub struct {
	values map[string]string
}

func (s *modelCatalogSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, nil
}

func (s *modelCatalogSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *modelCatalogSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *modelCatalogSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = s.values[key]
	}
	return result, nil
}

func (s *modelCatalogSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *modelCatalogSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}

func (s *modelCatalogSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestModelCatalogService_ListModelsAndDetailExposeLayeredPricing(t *testing.T) {
	model := "claude-sonnet-4.5"
	pricingLookup := "claude-sonnet-4-5-20250929"
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelCatalogEntries] = mustModelCatalogJSON(t, []ModelCatalogEntry{{
		Model:                model,
		DisplayName:          "Claude Sonnet 4.5",
		Provider:             "anthropic",
		Mode:                 "chat",
		CanonicalModelID:     pricingLookup,
		PricingLookupModelID: pricingLookup,
	}})
	repo.values[SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		pricingLookup: {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken: modelCatalogFloat64Ptr(1.5e-6),
			},
		},
	})
	repo.values[SettingKeyModelPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		pricingLookup: {
			ModelCatalogPricing: ModelCatalogPricing{
				OutputCostPerToken: modelCatalogFloat64Ptr(3.5e-6),
			},
		},
	})

	pricingService := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			pricingLookup: {
				InputCostPerToken:  1e-6,
				OutputCostPerToken: 2e-6,
				LiteLLMProvider:    "anthropic",
				Mode:               "chat",
			},
		},
	}

	svc := NewModelCatalogService(repo, nil, nil, pricingService, &config.Config{})
	items, total, err := svc.ListModels(context.Background(), ModelCatalogListFilter{
		Search:   model,
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	item := items[0]
	require.Equal(t, model, item.Model)
	require.Equal(t, "Claude Sonnet 4.5", item.DisplayName)
	require.Equal(t, "claude", item.IconKey)
	require.Equal(t, ModelCatalogPricingSourceOverride, item.PricingSource)
	require.Equal(t, ModelCatalogPricingSourceDynamic, item.BasePricingSource)
	require.NotNil(t, item.OfficialPricing)
	require.NotNil(t, item.SalePricing)
	require.Equal(t, 1.5e-6, *item.OfficialPricing.InputCostPerToken)
	require.Equal(t, 2e-6, *item.OfficialPricing.OutputCostPerToken)
	require.Equal(t, 1.5e-6, *item.SalePricing.InputCostPerToken)
	require.Equal(t, 3.5e-6, *item.SalePricing.OutputCostPerToken)
	require.Equal(t, *item.SalePricing.OutputCostPerToken, *item.EffectivePricing.OutputCostPerToken)

	detail, err := svc.GetModelDetail(context.Background(), model)
	require.NoError(t, err)
	require.Equal(t, model, detail.Model)
	require.NotNil(t, detail.UpstreamPricing)
	require.NotNil(t, detail.OfficialOverridePricing)
	require.NotNil(t, detail.SaleOverridePricing)
	require.NotNil(t, detail.BasePricing)
	require.NotNil(t, detail.OverridePricing)
	require.Equal(t, 1e-6, *detail.UpstreamPricing.InputCostPerToken)
	require.Equal(t, 2e-6, *detail.UpstreamPricing.OutputCostPerToken)
	require.Equal(t, 1.5e-6, *detail.BasePricing.InputCostPerToken)
	require.Equal(t, 3.5e-6, *detail.OverridePricing.OutputCostPerToken)
	require.Empty(t, detail.RouteReferences)
	require.Zero(t, detail.RouteReferenceCount)
}

func TestModelCatalogService_ListModelsDedupesDisplayNames(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	pricingLookup := "claude-sonnet-4-5-20250929"

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	items, total, err := svc.ListModels(context.Background(), ModelCatalogListFilter{Search: pricingLookup, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "claude-sonnet-4.5", items[0].Model)
	require.NotEmpty(t, items[0].DisplayName)

	items, total, err = svc.ListModels(context.Background(), ModelCatalogListFilter{Search: "claude-sonnet-4.5", Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "claude-sonnet-4.5", items[0].Model)
}

func TestModelCatalogService_SeedFallbackUsesCuratedBaseline(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})

	items, total, err := svc.ListModels(context.Background(), ModelCatalogListFilter{Page: 1, PageSize: 1000})
	require.NoError(t, err)
	require.Greater(t, total, int64(0))

	models := make(map[string]struct{}, len(items))
	for _, item := range items {
		models[item.Model] = struct{}{}
	}

	_, hasAnthropicOfficial := models["claude-opus-4.1"]
	_, hasOldAnthropic := models["claude-opus-4.6"]
	_, hasCurrentCodex := models["gpt-5.3-codex-spark"]
	_, hasLegacyCodex := models["gpt-5-codex"]
	_, hasOldCodex := models["gpt-5.3-codex"]
	require.True(t, hasAnthropicOfficial)
	require.False(t, hasOldAnthropic)
	require.True(t, hasCurrentCodex)
	require.False(t, hasLegacyCodex)
	require.False(t, hasOldCodex)
}

func TestModelCatalogService_PricingBackedSyntheticEntriesAppearInCatalogAndBillingCenter(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	pricingService := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.4-dynamic-only": {
				InputCostPerToken:  1e-6,
				OutputCostPerToken: 2e-6,
				LiteLLMProvider:    PlatformOpenAI,
				Mode:               "chat",
			},
		},
	}
	billingService := NewBillingService(&config.Config{}, nil)
	billingService.fallbackPrices = map[string]*ModelPricing{
		"gpt-5.4-billing-only": {
			InputPricePerToken:  3e-6,
			OutputPricePerToken: 6e-6,
		},
	}

	svc := NewModelCatalogService(repo, nil, billingService, pricingService, &config.Config{})

	dynamicDetail, err := svc.GetModelDetail(context.Background(), "gpt-5.4-dynamic-only")
	require.NoError(t, err)
	require.Equal(t, ModelCatalogPricingSourceDynamic, dynamicDetail.BasePricingSource)
	require.False(t, dynamicDetail.DefaultAvailable)
	require.Equal(t, PlatformOpenAI, dynamicDetail.Provider)

	billingDetail, err := svc.GetModelDetail(context.Background(), "gpt-5.4-billing-only")
	require.NoError(t, err)
	require.Equal(t, ModelCatalogPricingSourceDynamic, billingDetail.BasePricingSource)
	require.False(t, billingDetail.DefaultAvailable)
	require.Equal(t, PlatformOpenAI, billingDetail.Provider)

	items, total, err := svc.billingCenterService.ListPricingModels(context.Background(), BillingPricingListFilter{
		Search:   "gpt-5.4-",
		Page:     1,
		PageSize: 50,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(2))

	models := make(map[string]BillingPricingListItem, len(items))
	for _, item := range items {
		models[item.Model] = item
	}
	_, hasDynamic := models["gpt-5.4-dynamic-only"]
	_, hasFallback := models["gpt-5.4-billing-only"]
	require.True(t, hasDynamic)
	require.True(t, hasFallback)
}

func TestModelCatalogService_LegacyAliasesResolveToCuratedRows(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"claude-sonnet-4-5": {
			ModelCatalogPricing: ModelCatalogPricing{
				OutputCostPerToken: modelCatalogFloat64Ptr(5e-6),
			},
		},
	})
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})

	sonnetDetail, err := svc.GetModelDetail(context.Background(), "claude-sonnet-4.5")
	require.NoError(t, err)
	require.NotNil(t, sonnetDetail.OfficialOverridePricing)
	require.Equal(t, 5e-6, *sonnetDetail.OfficialOverridePricing.OutputCostPerToken)
}

func TestModelCatalogPricingValidationRejectsInvalidOverrides(t *testing.T) {
	tests := []struct {
		name    string
		pricing ModelCatalogPricing
	}{
		{
			name:    "empty override",
			pricing: ModelCatalogPricing{},
		},
		{
			name: "negative price",
			pricing: ModelCatalogPricing{
				InputCostPerToken: modelCatalogFloat64Ptr(-1),
			},
		},
		{
			name: "missing above threshold price",
			pricing: ModelCatalogPricing{
				InputTokenThreshold: modelCatalogIntPtr(200000),
			},
		},
		{
			name: "missing priority above threshold price",
			pricing: ModelCatalogPricing{
				OutputTokenThreshold:             modelCatalogIntPtr(200000),
				OutputCostPerToken:               modelCatalogFloat64Ptr(2e-6),
				OutputCostPerTokenAboveThreshold: modelCatalogFloat64Ptr(4e-6),
				OutputCostPerTokenPriority:       modelCatalogFloat64Ptr(3e-6),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateOverridePricing(test.pricing)
			if test.name == "missing above threshold price" || test.name == "missing priority above threshold price" {
				err = validateTieredPricingConfiguration(&test.pricing)
			}
			require.Error(t, err)
		})
	}

	require.NoError(t, validateOverridePricing(ModelCatalogPricing{
		InputCostPerToken:               modelCatalogFloat64Ptr(1e-6),
		InputTokenThreshold:             modelCatalogIntPtr(200000),
		InputCostPerTokenAboveThreshold: modelCatalogFloat64Ptr(2e-6),
	}))
	require.NoError(t, validateTieredPricingConfiguration(&ModelCatalogPricing{
		InputCostPerToken:               modelCatalogFloat64Ptr(1e-6),
		InputTokenThreshold:             modelCatalogIntPtr(200000),
		InputCostPerTokenAboveThreshold: modelCatalogFloat64Ptr(2e-6),
	}))
}

func TestModelCatalogService_GeminiPricingOverrideDeprecated(t *testing.T) {
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

	_, err := svc.UpsertPricingOverride(context.Background(), ModelCatalogActor{UserID: 1, Email: "gemini@example.com"}, UpsertModelPricingOverrideInput{
		Model: "gemini-3-pro",
		ModelCatalogPricing: ModelCatalogPricing{
			InputCostPerToken:  modelCatalogFloat64Ptr(3e-6),
			OutputCostPerImage: modelCatalogFloat64Ptr(0.04),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "GEMINI_PRICING_OVERRIDE_DEPRECATED")

	_, err = svc.UpsertOfficialPricingOverride(context.Background(), ModelCatalogActor{UserID: 1, Email: "gemini@example.com"}, UpsertModelPricingOverrideInput{
		Model: "gemini-3-pro",
		ModelCatalogPricing: ModelCatalogPricing{
			InputCostPerToken: modelCatalogFloat64Ptr(3e-6),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "GEMINI_PRICING_OVERRIDE_DEPRECATED")
}

func TestModelCatalogService_GeminiLegacySeedStillBuildsMatrixWithoutCanonical(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gemini-2.5-pro": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken:                   modelCatalogFloat64Ptr(3e-6),
				InputTokenThreshold:                 modelCatalogIntPtr(200000),
				InputCostPerTokenAboveThreshold:     modelCatalogFloat64Ptr(5e-6),
				CacheCreationInputTokenCostAbove1hr: modelCatalogFloat64Ptr(7e-6),
			},
		},
	})
	svc, billingService := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-2.5-pro": {
			InputCostPerToken:                   2e-6,
			OutputCostPerToken:                  8e-6,
			CacheCreationInputTokenCost:         1.5e-6,
			CacheReadInputTokenCost:             0.3e-6,
			CacheCreationInputTokenCostAbove1hr: 6e-6,
			LiteLLMProvider:                     PlatformGemini,
			Mode:                                "chat",
			SupportsServiceTier:                 true,
		},
	})

	override := svc.loadSalePriceOverrides(context.Background())["gemini-2.5-pro"]
	require.NotNil(t, override)
	require.NotNil(t, override.InputCostPerToken)
	require.NotNil(t, override.InputTokenThreshold)
	require.Equal(t, 200000, *override.InputTokenThreshold)
	require.Equal(t, 5e-6, *override.InputCostPerTokenAboveThreshold)
	require.Equal(t, 7e-6, *override.CacheCreationInputTokenCostAbove1hr)

	sheet, err := svc.billingCenterService.GetSheet(context.Background(), "gemini-2.5-pro")
	require.NoError(t, err)
	require.NotNil(t, sheet.SaleMatrix)
	cell := geminiMatrixCell(sheet.SaleMatrix, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotTextInput)
	require.NotNil(t, cell)
	require.NotNil(t, cell.Price)
	require.Equal(t, 3e-6, *cell.Price)

	detail, err := svc.GetModelDetail(context.Background(), "gemini-2.5-pro")
	require.NoError(t, err)
	require.NotNil(t, detail.SalePricing)
	require.Equal(t, 3e-6, *detail.SalePricing.InputCostPerToken)
	require.Equal(t, 200000, *detail.SalePricing.InputTokenThreshold)
	require.Equal(t, 5e-6, *detail.SalePricing.InputCostPerTokenAboveThreshold)
	require.Equal(t, 7e-6, *detail.SalePricing.CacheCreationInputTokenCostAbove1hr)

	pricing, err := billingService.getPricingForBilling("gemini-2.5-pro")
	require.NoError(t, err)
	require.Equal(t, 3e-6, pricing.InputPricePerToken)
	require.Equal(t, 200000, pricing.InputTokenThreshold)
	require.Equal(t, 5e-6, pricing.InputPricePerTokenAboveThreshold)
	require.Equal(t, 7e-6, pricing.CacheCreation1hPrice)
}

func TestModelCatalogService_GeminiImagePricingSeedsPriorityMatrixCell(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-2.5-flash-image": {
			InputCostPerToken:          3e-7,
			InputCostPerTokenPriority:  5.4e-7,
			OutputCostPerImage:         0.039,
			OutputCostPerImagePriority: 0.0702,
			LiteLLMProvider:            PlatformGemini,
			Mode:                       "image_generation",
			SupportsServiceTier:        true,
		},
	})

	sheet, err := svc.billingCenterService.GetSheet(context.Background(), "gemini-2.5-flash-image")
	require.NoError(t, err)
	require.NotNil(t, sheet)
	require.NotNil(t, sheet.OfficialMatrix)

	standardCell := geminiMatrixCell(sheet.OfficialMatrix, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotImageOutput)
	require.NotNil(t, standardCell)
	require.NotNil(t, standardCell.Price)
	require.InDelta(t, 0.039, *standardCell.Price, 1e-12)

	priorityCell := geminiMatrixCell(sheet.OfficialMatrix, BillingSurfaceGeminiNative, BillingServiceTierPriority, BillingChargeSlotImageOutput)
	require.NotNil(t, priorityCell)
	require.NotNil(t, priorityCell.Price)
	require.InDelta(t, 0.0702, *priorityCell.Price, 1e-12)
}

func TestModelCatalogService_GeminiBillingSheetUsesCanonicalRulesAndClearsLegacyOverride(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gemini-3-pro": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken:  modelCatalogFloat64Ptr(3e-6),
				OutputCostPerToken: modelCatalogFloat64Ptr(8e-6),
			},
		},
	})
	svc, billingService := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
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
	textInput := 4e-6
	textOutput := 9e-6
	imageOutput := 0.05
	row := geminiMatrixRow(matrix, BillingSurfaceGeminiNative, BillingServiceTierStandard)
	require.NotNil(t, row)
	row.Slots[BillingChargeSlotTextInput] = GeminiBillingMatrixCell{Price: &textInput}
	row.Slots[BillingChargeSlotTextOutput] = GeminiBillingMatrixCell{Price: &textOutput}
	row.Slots[BillingChargeSlotImageOutput] = GeminiBillingMatrixCell{Price: &imageOutput}

	sheet, err := svc.UpsertBillingSheet(context.Background(), ModelCatalogActor{UserID: 4, Email: "matrix@example.com"}, UpsertModelBillingSheetInput{
		Model:  "gemini-3-pro",
		Layer:  BillingLayerSale,
		Matrix: matrix,
	})
	require.NoError(t, err)
	require.NotNil(t, sheet)
	require.NotNil(t, sheet.SaleMatrix)

	rules := loadBillingRulesBySetting(context.Background(), repo, SettingKeyBillingCenterRules)
	require.Nil(t, buildGeminiCompatPricingOverride("gemini-3-pro-preview", BillingLayerSale, rules))
	_, exists := svc.loadSalePriceOverrides(context.Background())["gemini-3-pro"]
	require.False(t, exists)

	cell := geminiMatrixCell(sheet.SaleMatrix, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotTextInput)
	require.NotNil(t, cell)
	require.NotNil(t, cell.Price)
	require.Equal(t, 4e-6, *cell.Price)

	pricing, err := billingService.getPricingForBilling("gemini-3-pro-preview")
	require.NoError(t, err)
	require.Equal(t, 2e-6, pricing.InputPricePerToken)
	require.Equal(t, 7e-6, pricing.OutputPricePerToken)
	require.Zero(t, pricing.OutputPricePerImage)
}

func TestModelCatalogService_DeleteGeminiPricingOverrideClearsCompatAndLegacy(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gemini-2.5-pro": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken:               modelCatalogFloat64Ptr(3e-6),
				InputTokenThreshold:             modelCatalogIntPtr(200000),
				InputCostPerTokenAboveThreshold: modelCatalogFloat64Ptr(5e-6),
			},
		},
	})
	repo.values[SettingKeyBillingCenterRules] = mustModelCatalogJSON(t, []BillingRule{{
		ID:            "manual_rule_keep",
		Provider:      BillingRuleProviderGemini,
		Layer:         BillingLayerSale,
		Surface:       BillingSurfaceAny,
		OperationType: "",
		ServiceTier:   "",
		BatchMode:     BillingBatchModeAny,
		Matchers:      BillingRuleMatchers{Models: []string{"gemini-2.5-flash"}},
		Unit:          BillingUnitInputToken,
		Price:         9e-6,
		Priority:      1,
		Enabled:       true,
	}})
	svc, _ := newGeminiBillingCatalogService(repo, map[string]*LiteLLMModelPricing{
		"gemini-2.5-pro": {
			InputCostPerToken:           2e-6,
			OutputCostPerToken:          8e-6,
			CacheCreationInputTokenCost: 1e-6,
			CacheReadInputTokenCost:     0.2e-6,
			LiteLLMProvider:             PlatformGemini,
			Mode:                        "chat",
		},
	})

	err := svc.DeletePricingOverride(context.Background(), ModelCatalogActor{}, "gemini-2.5-pro")
	require.NoError(t, err)

	rules := loadBillingRulesBySetting(context.Background(), repo, SettingKeyBillingCenterRules)
	require.Len(t, rules, 1)
	require.Equal(t, "manual_rule_keep", rules[0].ID)
	require.Nil(t, buildGeminiCompatPricingOverride("gemini-2.5-pro", BillingLayerSale, rules))
	_, exists := svc.loadSalePriceOverrides(context.Background())["gemini-2.5-pro"]
	require.False(t, exists)

	detail, err := svc.GetModelDetail(context.Background(), "gemini-2.5-pro")
	require.NoError(t, err)
	require.Nil(t, detail.SaleOverridePricing)
	require.NotNil(t, detail.SalePricing)
	require.Equal(t, 2e-6, *detail.SalePricing.InputCostPerToken)
}

func newGeminiBillingCatalogService(repo *modelCatalogSettingRepoStub, pricing map[string]*LiteLLMModelPricing) (*ModelCatalogService, *BillingService) {
	cfg := &config.Config{}
	pricingService := &PricingService{pricingData: pricing}
	billingService := NewBillingService(cfg, pricingService)
	return NewModelCatalogService(repo, nil, billingService, pricingService, cfg), billingService
}

func mustModelCatalogJSON(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return string(payload)
}
