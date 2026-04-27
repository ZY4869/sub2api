package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestBillingCenterService_BuildPricingStatusSnapshotUsesSharedLookupWarningsWithoutFalseConflict(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	pricingService := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gemini-3-flash-preview": {
				InputCostPerToken:  5e-7,
				OutputCostPerToken: 3e-6,
				LiteLLMProvider:    PlatformGemini,
				Mode:               "chat",
			},
			"gpt-5.4-pro": {
				InputCostPerToken:  3e-5,
				OutputCostPerToken: 1.8e-4,
				LiteLLMProvider:    PlatformOpenAI,
				Mode:               "responses",
			},
		},
	}
	svc := NewModelCatalogService(repo, nil, NewBillingService(&config.Config{}, pricingService), pricingService, &config.Config{})

	records, err := svc.buildCatalogRecords(context.Background())
	require.NoError(t, err)

	geminiFlash := records["gemini-3-flash"]
	require.NotNil(t, geminiFlash)
	require.Equal(t, BillingPricingStatusOK, geminiFlash.pricingStatus)
	require.NotNil(t, geminiFlash.upstreamPricing)
	require.Contains(t, geminiFlash.pricingWarnings, `Shared pricing lookup "gemini-3-flash-preview" is reused by 2 models; pricing is sourced from the same upstream entry.`)

	gpt54ProSnapshot := records["gpt-5.4-pro-2026-03-05"]
	require.NotNil(t, gpt54ProSnapshot)
	require.Equal(t, BillingPricingStatusOK, gpt54ProSnapshot.pricingStatus)
	require.NotNil(t, gpt54ProSnapshot.upstreamPricing)
	require.Contains(t, gpt54ProSnapshot.pricingWarnings, `Shared pricing lookup "gpt-5.4-pro" is reused by 2 models; pricing is sourced from the same upstream entry.`)

	geminiFlashImage := records["gemini-3.1-flash-image"]
	require.NotNil(t, geminiFlashImage)
	require.Equal(t, BillingPricingStatusMissing, geminiFlashImage.pricingStatus)
	require.Nil(t, geminiFlashImage.upstreamPricing)
	require.Contains(t, geminiFlashImage.pricingWarnings, "No stable upstream pricing source found.")

	snapshot, err := svc.billingCenterService.buildBillingPricingCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Empty(t, assertUniqueBillingPricingModels(snapshot.Models))

	geminiFlashPersisted, ok, _ := billingPricingSnapshotModel(snapshot, "gemini-3-flash")
	require.True(t, ok)
	require.Equal(t, BillingPricingStatusOK, geminiFlashPersisted.PricingStatus)

	gpt54ProPersisted, ok, _ := billingPricingSnapshotModel(snapshot, "gpt-5.4-pro-2026-03-05")
	require.True(t, ok)
	require.Equal(t, BillingPricingStatusOK, gpt54ProPersisted.PricingStatus)

	geminiFlashImagePersisted, ok, _ := billingPricingSnapshotModel(snapshot, "gemini-3.1-flash-image")
	require.True(t, ok)
	require.Equal(t, BillingPricingStatusMissing, geminiFlashImagePersisted.PricingStatus)
}

func TestDeriveBillingPricingStatus_CNYMissingLockedFXIsNotOK(t *testing.T) {
	record := &modelCatalogRecord{
		model:             "ernie-x1-turbo-32k",
		pricingCurrency:   ModelPricingCurrencyCNY,
		basePricingSource: ModelCatalogPricingSourceDynamic,
		upstreamPricing: &ModelCatalogPricing{
			Currency:          ModelPricingCurrencyCNY,
			InputCostPerToken: modelCatalogFloat64Ptr(0.000004),
		},
	}

	status, warnings := deriveBillingPricingStatus(record, nil, nil)
	require.Equal(t, BillingPricingStatusMissing, status)
	require.Contains(t, warnings, "CNY pricing is missing a locked USD/CNY rate.")

	lockedAt := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	record.upstreamPricing.USDToCNYRate = modelCatalogFloat64Ptr(6.8363)
	record.upstreamPricing.FXLockedAt = &lockedAt
	status, warnings = deriveBillingPricingStatus(record, nil, nil)
	require.Equal(t, BillingPricingStatusOK, status)
	require.NotContains(t, warnings, "CNY pricing is missing a locked USD/CNY rate.")
}
