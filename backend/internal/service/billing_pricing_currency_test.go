package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBillingCenterService_GetPricingDetails_DefaultsCurrencyToUSD(t *testing.T) {
	svc, _ := newBillingPricingCurrencyCatalogServiceForTest(t)

	details, err := svc.billingCenterService.GetPricingDetails(context.Background(), []string{"gpt-5.4"})
	require.NoError(t, err)
	require.Len(t, details, 1)
	require.Equal(t, ModelPricingCurrencyUSD, details[0].Currency)
}

func TestBillingCenterService_SavePricingLayer_PersistsCurrencyPreference(t *testing.T) {
	svc, _ := newBillingPricingCurrencyCatalogServiceForTest(t)

	inputPrice := 1.75e-6
	outputPrice := 6.5e-6
	detail, err := svc.billingCenterService.SavePricingLayer(
		context.Background(),
		ModelCatalogActor{UserID: 9, Email: "billing@example.com"},
		UpsertBillingPricingLayerInput{
			Model:    "gpt-5.4",
			Layer:    BillingLayerOfficial,
			Currency: ModelPricingCurrencyCNY,
			Form: &BillingPricingLayerForm{
				InputPrice:     &inputPrice,
				OutputPrice:    &outputPrice,
				SpecialEnabled: false,
				Special:        BillingPricingSimpleSpecial{},
				TieredEnabled:  false,
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, detail)
	require.Equal(t, ModelPricingCurrencyCNY, detail.Currency)
	require.NotNil(t, detail.OfficialForm.InputPrice)
	require.NotNil(t, detail.OfficialForm.OutputPrice)
	require.InDelta(t, inputPrice, *detail.OfficialForm.InputPrice, 1e-12)
	require.InDelta(t, outputPrice, *detail.OfficialForm.OutputPrice, 1e-12)

	prefs := svc.loadModelPricingCurrencies(context.Background())
	require.Contains(t, prefs, "gpt-5.4")
	require.Equal(t, ModelPricingCurrencyCNY, prefs["gpt-5.4"].Currency)
	require.Equal(t, int64(9), prefs["gpt-5.4"].UpdatedByUserID)
	require.Equal(t, "billing@example.com", prefs["gpt-5.4"].UpdatedByEmail)

	snapshot := loadBillingPricingCatalogSnapshotBySetting(context.Background(), svc.settingRepo, SettingKeyBillingPricingCatalogSnapshot)
	require.NotNil(t, snapshot)
	model, ok, _ := billingPricingSnapshotModel(snapshot, "gpt-5.4")
	require.True(t, ok)
	require.Equal(t, ModelPricingCurrencyCNY, model.Currency)
	require.NotNil(t, model.OfficialForm.InputPrice)
	require.NotNil(t, model.OfficialForm.OutputPrice)
	require.InDelta(t, inputPrice, *model.OfficialForm.InputPrice, 1e-12)
	require.InDelta(t, outputPrice, *model.OfficialForm.OutputPrice, 1e-12)

	override := svc.loadOfficialPriceOverrides(context.Background())["gpt-5.4"]
	require.NotNil(t, override)
	require.NotNil(t, override.InputCostPerToken)
	require.NotNil(t, override.OutputCostPerToken)
	require.InDelta(t, inputPrice, *override.InputCostPerToken, 1e-12)
	require.InDelta(t, outputPrice, *override.OutputCostPerToken, 1e-12)
}

func TestBillingCenterService_CopyAndDiscount_PreservePricingCurrency(t *testing.T) {
	svc, _ := newBillingPricingCurrencyCatalogServiceForTest(t)

	inputPrice := 1.75e-6
	outputPrice := 6.5e-6
	_, err := svc.billingCenterService.SavePricingLayer(
		context.Background(),
		ModelCatalogActor{UserID: 9, Email: "billing@example.com"},
		UpsertBillingPricingLayerInput{
			Model:    "gpt-5.4",
			Layer:    BillingLayerOfficial,
			Currency: ModelPricingCurrencyCNY,
			Form: &BillingPricingLayerForm{
				InputPrice:     &inputPrice,
				OutputPrice:    &outputPrice,
				SpecialEnabled: false,
				Special:        BillingPricingSimpleSpecial{},
				TieredEnabled:  false,
			},
		},
	)
	require.NoError(t, err)

	copied, err := svc.billingCenterService.CopyPricingItemsOfficialToSale(
		context.Background(),
		ModelCatalogActor{UserID: 11, Email: "copy@example.com"},
		[]string{"gpt-5.4"},
	)
	require.NoError(t, err)
	require.Len(t, copied, 1)
	require.Equal(t, ModelPricingCurrencyCNY, copied[0].Currency)

	discounted, err := svc.billingCenterService.ApplySaleDiscount(
		context.Background(),
		ModelCatalogActor{UserID: 12, Email: "discount@example.com"},
		BillingBulkApplyRequest{
			Models:        []string{"gpt-5.4"},
			DiscountRatio: 0.5,
		},
	)
	require.NoError(t, err)
	require.Len(t, discounted, 1)
	require.Equal(t, ModelPricingCurrencyCNY, discounted[0].Currency)

	prefs := svc.loadModelPricingCurrencies(context.Background())
	require.Contains(t, prefs, "gpt-5.4")
	require.Equal(t, ModelPricingCurrencyCNY, prefs["gpt-5.4"].Currency)
}

func newBillingPricingCurrencyCatalogServiceForTest(
	t *testing.T,
) (*ModelCatalogService, *modelCatalogSettingRepoStub) {
	t.Helper()

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
				InputCostPerToken:  modelCatalogFloat64Ptr(1.5e-6),
				OutputCostPerToken: modelCatalogFloat64Ptr(6e-6),
			},
		},
	})
	repo.values[SettingKeyModelPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gpt-5.4": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken:  modelCatalogFloat64Ptr(2e-6),
				OutputCostPerToken: modelCatalogFloat64Ptr(7e-6),
			},
		},
	})

	billingService := NewBillingService(&config.Config{}, nil)
	svc := NewModelCatalogService(repo, nil, billingService, nil, &config.Config{})
	return svc, repo
}
