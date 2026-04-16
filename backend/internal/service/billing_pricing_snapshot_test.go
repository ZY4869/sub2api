package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBillingCenterService_MigratesLegacyPricingIntoSnapshot(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelCatalogEntries] = mustModelCatalogJSON(t, []ModelCatalogEntry{
		{
			Model:                "gpt-5.4",
			DisplayName:          "GPT-5.4",
			Provider:             PlatformOpenAI,
			Mode:                 "chat",
			CanonicalModelID:     "gpt-5.4",
			PricingLookupModelID: "gpt-5.4",
		},
	})
	repo.values[SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gpt-5.4": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken:               modelCatalogFloat64Ptr(1.5e-6),
				OutputCostPerToken:              modelCatalogFloat64Ptr(6e-6),
				InputTokenThreshold:             modelCatalogIntPtr(200000),
				InputCostPerTokenAboveThreshold: modelCatalogFloat64Ptr(2.4e-6),
			},
		},
	})
	repo.values[SettingKeyBillingCenterRules] = mustModelCatalogJSON(t, []BillingRule{
		{
			ID:            "rule_batch_input",
			Provider:      PlatformOpenAI,
			Layer:         BillingLayerOfficial,
			Surface:       BillingSurfaceAny,
			OperationType: "generate_content",
			BatchMode:     BillingBatchModeBatch,
			Matchers: BillingRuleMatchers{
				Models: []string{"gpt-5.4"},
			},
			Unit:    BillingUnitInputToken,
			Price:   0.75e-6,
			Enabled: true,
		},
		{
			ID:            "rule_grounding_search",
			Provider:      PlatformGemini,
			Layer:         BillingLayerSale,
			Surface:       BillingSurfaceGeminiNative,
			OperationType: "generate_content",
			Matchers: BillingRuleMatchers{
				Models: []string{"gpt-5.4"},
			},
			Unit:    BillingUnitGroundingSearchRequest,
			Price:   0.12,
			Enabled: true,
		},
	})
	repo.values[SettingKeyModelPricingCurrencies] = mustModelCatalogJSON(t, map[string]*BillingPricingCurrencyPreference{
		"gpt-5.4": {Currency: ModelPricingCurrencyCNY},
	})

	svc := NewModelCatalogService(repo, nil, NewBillingService(&config.Config{}, nil), nil, &config.Config{})

	details, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gpt-5.4"})
	require.NoError(t, err)
	require.Len(t, details, 1)
	require.Equal(t, ModelPricingCurrencyCNY, details[0].Currency)
	require.True(t, details[0].OfficialForm.TieredEnabled)
	require.NotNil(t, details[0].OfficialForm.InputPrice)
	require.NotNil(t, details[0].OfficialForm.InputPriceAboveThreshold)
	require.NotNil(t, details[0].OfficialForm.Special.BatchInputPrice)
	require.InDelta(t, 1.5e-6, *details[0].OfficialForm.InputPrice, 1e-12)
	require.InDelta(t, 2.4e-6, *details[0].OfficialForm.InputPriceAboveThreshold, 1e-12)
	require.InDelta(t, 0.75e-6, *details[0].OfficialForm.Special.BatchInputPrice, 1e-12)
	require.NotNil(t, details[0].SaleForm.Special.GroundingSearch)
	require.InDelta(t, 0.12, *details[0].SaleForm.Special.GroundingSearch, 1e-12)

	snapshot := loadBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot)
	require.NotNil(t, snapshot)
	require.NotEmpty(t, snapshot.Models)
	model, ok, _ := billingPricingSnapshotModel(snapshot, "gpt-5.4")
	require.True(t, ok)
	require.Equal(t, ModelPricingCurrencyCNY, model.Currency)
	require.Greater(t, model.OfficialCount, 0)
	require.Greater(t, model.SaleCount, 0)
}

func TestBillingCenterService_RefreshPricingCatalog_MergesNewModelsWithoutDroppingExistingPricing(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelCatalogEntries] = mustModelCatalogJSON(t, []ModelCatalogEntry{
		{
			Model:                "gpt-5.4",
			DisplayName:          "GPT-5.4",
			Provider:             PlatformOpenAI,
			Mode:                 "chat",
			CanonicalModelID:     "gpt-5.4",
			PricingLookupModelID: "gpt-5.4",
		},
	})
	repo.values[SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gpt-5.4": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken:  modelCatalogFloat64Ptr(1.75e-6),
				OutputCostPerToken: modelCatalogFloat64Ptr(6.5e-6),
			},
		},
	})
	repo.values[SettingKeyModelPricingCurrencies] = mustModelCatalogJSON(t, map[string]*BillingPricingCurrencyPreference{
		"gpt-5.4": {Currency: ModelPricingCurrencyCNY},
	})
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{}}

	svc := NewModelCatalogService(repo, nil, NewBillingService(&config.Config{}, nil), pricingService, &config.Config{})

	initialDetails, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gpt-5.4"})
	require.NoError(t, err)
	require.Len(t, initialDetails, 1)
	require.Equal(t, ModelPricingCurrencyCNY, initialDetails[0].Currency)
	require.NotNil(t, initialDetails[0].OfficialForm.InputPrice)
	require.InDelta(t, 1.75e-6, *initialDetails[0].OfficialForm.InputPrice, 1e-12)

	pricingService.pricingData["claude-3.7-sonnet-refresh-new"] = &LiteLLMModelPricing{
		InputCostPerToken:  3e-6,
		OutputCostPerToken: 15e-6,
		LiteLLMProvider:    PlatformAnthropic,
		Mode:               "chat",
	}

	result, err := svc.billingCenterService.RefreshPricingCatalog(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.GreaterOrEqual(t, result.TotalModels, 2)
	require.GreaterOrEqual(t, result.ProviderCount, 2)

	refreshedDetails, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gpt-5.4", "claude-3.7-sonnet-refresh-new"})
	require.NoError(t, err)
	require.Len(t, refreshedDetails, 2)
	require.Equal(t, "gpt-5.4", refreshedDetails[0].Model)
	require.Equal(t, ModelPricingCurrencyCNY, refreshedDetails[0].Currency)
	require.NotNil(t, refreshedDetails[0].OfficialForm.InputPrice)
	require.InDelta(t, 1.75e-6, *refreshedDetails[0].OfficialForm.InputPrice, 1e-12)
	require.Equal(t, "claude-3.7-sonnet-refresh-new", refreshedDetails[1].Model)
	require.Equal(t, PlatformAnthropic, refreshedDetails[1].Provider)
	require.Equal(t, ModelPricingCurrencyUSD, refreshedDetails[1].Currency)
}

func TestBillingCenterService_RefreshPricingCatalog_RetainsSnapshotOnlyModels(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelCatalogEntries] = mustModelCatalogJSON(t, []ModelCatalogEntry{
		{
			Model:                "gpt-5.4",
			DisplayName:          "GPT-5.4",
			Provider:             PlatformOpenAI,
			Mode:                 "chat",
			CanonicalModelID:     "gpt-5.4",
			PricingLookupModelID: "gpt-5.4",
		},
	})
	repo.values[SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gpt-5.4": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken:  modelCatalogFloat64Ptr(1.75e-6),
				OutputCostPerToken: modelCatalogFloat64Ptr(6.5e-6),
			},
		},
	})

	legacyInputPrice := 9.5e-7
	legacyOutputPrice := 3.8e-6
	legacySnapshot := &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			{
				Model:            "legacy-snapshot-only-model",
				DisplayName:      "Legacy Snapshot Only",
				Provider:         PlatformAnthropic,
				Mode:             "chat",
				Currency:         ModelPricingCurrencyCNY,
				InputSupported:   true,
				OutputChargeSlot: BillingChargeSlotTextOutput,
				OfficialForm: BillingPricingLayerForm{
					InputPrice:     &legacyInputPrice,
					OutputPrice:    &legacyOutputPrice,
					SpecialEnabled: false,
					Special:        BillingPricingSimpleSpecial{},
					TieredEnabled:  false,
				},
				SaleForm: BillingPricingLayerForm{
					SpecialEnabled: false,
					Special:        BillingPricingSimpleSpecial{},
					TieredEnabled:  false,
				},
			},
		},
	}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, legacySnapshot))

	svc := NewModelCatalogService(repo, nil, NewBillingService(&config.Config{}, nil), nil, &config.Config{})

	result, err := svc.billingCenterService.RefreshPricingCatalog(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.GreaterOrEqual(t, result.TotalModels, 2)

	details, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{
		"legacy-snapshot-only-model",
		"gpt-5.4",
	})
	require.NoError(t, err)
	require.Len(t, details, 2)
	require.Equal(t, "legacy-snapshot-only-model", details[0].Model)
	require.Equal(t, ModelPricingCurrencyCNY, details[0].Currency)
	require.NotNil(t, details[0].OfficialForm.InputPrice)
	require.NotNil(t, details[0].OfficialForm.OutputPrice)
	require.InDelta(t, legacyInputPrice, *details[0].OfficialForm.InputPrice, 1e-12)
	require.InDelta(t, legacyOutputPrice, *details[0].OfficialForm.OutputPrice, 1e-12)
	require.Equal(t, "gpt-5.4", details[1].Model)
}

func TestBillingCenterService_ListPricingModels_SortsByProvider(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	pricingService := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.4-sort-openai": {
				InputCostPerToken:  1e-6,
				OutputCostPerToken: 2e-6,
				LiteLLMProvider:    PlatformOpenAI,
				Mode:               "chat",
			},
			"claude-3.7-sort-anthropic": {
				InputCostPerToken:  3e-6,
				OutputCostPerToken: 15e-6,
				LiteLLMProvider:    PlatformAnthropic,
				Mode:               "chat",
			},
		},
	}

	svc := NewModelCatalogService(repo, nil, NewBillingService(&config.Config{}, nil), pricingService, &config.Config{})

	ascItems, _, err := svc.billingCenterService.ListPricingModels(context.Background(), BillingPricingListFilter{
		Search:    "sort-",
		SortBy:    "provider",
		SortOrder: "asc",
		Page:      1,
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Len(t, ascItems, 2)
	require.Equal(t, PlatformAnthropic, ascItems[0].Provider)
	require.Equal(t, PlatformOpenAI, ascItems[1].Provider)

	descItems, _, err := svc.billingCenterService.ListPricingModels(context.Background(), BillingPricingListFilter{
		Search:    "sort-",
		SortBy:    "provider",
		SortOrder: "desc",
		Page:      1,
		PageSize:  20,
	})
	require.NoError(t, err)
	require.Len(t, descItems, 2)
	require.Equal(t, PlatformOpenAI, descItems[0].Provider)
	require.Equal(t, PlatformAnthropic, descItems[1].Provider)
}
