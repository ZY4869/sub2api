package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestMatchesBillingPricingFilter_FiltersByPricingStatus(t *testing.T) {
	item := BillingPricingListItem{
		Model:         "gpt-5.4",
		DisplayName:   "GPT-5.4",
		Provider:      PlatformOpenAI,
		Mode:          "chat",
		PricingStatus: BillingPricingStatusOK,
	}

	filter := BillingPricingListFilter{PricingStatus: string(BillingPricingStatusOK)}
	require.True(t, matchesBillingPricingFilter(item, filter))

	filter.PricingStatus = string(BillingPricingStatusMissing)
	require.False(t, matchesBillingPricingFilter(item, filter))
}

func TestBillingCenterService_ListPricingModels_IncludesOverrideOnlyModels(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gpt-override-only-model": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken:  modelCatalogFloat64Ptr(1.5e-6),
				OutputCostPerToken: modelCatalogFloat64Ptr(6e-6),
			},
		},
	})

	svc := NewModelCatalogService(repo, nil, NewBillingService(&config.Config{}, nil), nil, &config.Config{})

	items, total, err := svc.billingCenterService.ListPricingModels(
		context.Background(),
		BillingPricingListFilter{
			Search:   "gpt-override-only-model",
			Page:     1,
			PageSize: 20,
		},
	)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, items)
	require.Equal(t, "gpt-override-only-model", items[0].Model)
}
