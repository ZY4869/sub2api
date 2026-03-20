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

	items, total, err := svc.ListModels(context.Background(), ModelCatalogListFilter{Page: 1, PageSize: 100})
	require.NoError(t, err)
	require.Greater(t, total, int64(0))

	models := make(map[string]struct{}, len(items))
	for _, item := range items {
		models[item.Model] = struct{}{}
	}

	_, hasAnthropicOfficial := models["claude-opus-4.1"]
	_, hasOldAnthropic := models["claude-opus-4.6"]
	_, hasCurrentCodex := models["gpt-5-codex"]
	_, hasOldCodex := models["gpt-5.3-codex"]
	require.True(t, hasAnthropicOfficial)
	require.False(t, hasOldAnthropic)
	require.True(t, hasCurrentCodex)
	require.False(t, hasOldCodex)
}

func TestModelCatalogService_LegacyAliasesResolveToCuratedRows(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
		"gpt-5.3-codex": {
			ModelCatalogPricing: ModelCatalogPricing{
				InputCostPerToken: modelCatalogFloat64Ptr(1.25e-6),
			},
		},
		"claude-sonnet-4-5": {
			ModelCatalogPricing: ModelCatalogPricing{
				OutputCostPerToken: modelCatalogFloat64Ptr(5e-6),
			},
		},
	})
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})

	codexDetail, err := svc.GetModelDetail(context.Background(), "gpt-5-codex")
	require.NoError(t, err)
	require.NotNil(t, codexDetail.OfficialOverridePricing)
	require.Equal(t, 1.25e-6, *codexDetail.OfficialOverridePricing.InputCostPerToken)

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

func mustModelCatalogJSON(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return string(payload)
}
