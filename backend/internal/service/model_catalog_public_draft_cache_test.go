package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestModelCatalogService_GetPublicModelCatalogDraftPayload_UsesCandidateCacheAndSupportsForceRefresh(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	svc := &ModelCatalogService{}
	svc.storePublicModelCatalogSnapshot(&PublicModelCatalogSnapshot{
		ETag:      "test-etag",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		PageSize:  10,
		Items: []PublicModelCatalogItem{
			{
				Model:        "gpt-5.4",
				DisplayName:  "GPT-5.4",
				Provider:     PlatformOpenAI,
				Currency:     ModelPricingCurrencyUSD,
				PriceDisplay: PublicModelCatalogPriceDisplay{Primary: []PublicModelCatalogPriceEntry{{ID: "input", Unit: "token", Value: 1}}},
				MultiplierSummary: PublicModelCatalogMultiplierSummary{
					Enabled: false,
					Kind:    "disabled",
				},
			},
		},
	})

	payload, err := svc.GetPublicModelCatalogDraftPayload(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, payload.AvailableItems, 1)
	require.Equal(t, publicModelCatalogDraftAvailableSourceCache, payload.AvailableSource)
	require.True(t, logSink.ContainsMessageAtLevel("public model catalog draft candidate cache hit", "info"))

	_, err = svc.GetPublicModelCatalogDraftPayload(context.Background(), true)
	require.NoError(t, err)
	require.True(t, logSink.ContainsMessageAtLevel("public model catalog draft candidate snapshot refreshed", "info"))
}

func TestModelCatalogService_PublicCatalogDraftCandidate_PersistedForceAndPublish(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	persisted := publicCatalogCandidateTestSnapshot("persisted-image")
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, persisted))
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(ctx, repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 28, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("gpt-image-2", PlatformOpenAI, "image", true, BillingChargeSlotImageOutput, BillingPricingLayerForm{
				OutputPrice: modelCatalogFloat64Ptr(0.08),
			}),
		},
	}))
	svc := NewModelCatalogService(repo, nil, nil, nil, nil)

	payload, err := svc.GetPublicModelCatalogDraftPayload(ctx, false)
	require.NoError(t, err)
	require.Equal(t, publicModelCatalogDraftAvailableSourcePersisted, payload.AvailableSource)
	require.Equal(t, []string{"persisted-image"}, publicCatalogItemModels(payload.AvailableItems))

	payload, err = svc.GetPublicModelCatalogDraftPayload(ctx, true)
	require.NoError(t, err)
	require.Equal(t, publicModelCatalogDraftAvailableSourceRefreshed, payload.AvailableSource)
	require.Equal(t, []string{"gpt-image-2"}, publicCatalogItemModels(payload.AvailableItems))

	overwritten := loadPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot)
	require.NotNil(t, overwritten)
	require.Equal(t, []string{"gpt-image-2"}, publicCatalogItemModels(overwritten.Items))

	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, persisted))
	summary, err := svc.PublishPublicModelCatalog(ctx, ModelCatalogActor{UserID: 1, Email: "admin@example.test"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"persisted-image"},
		PageSize:       10,
	})
	require.NoError(t, err)
	require.Equal(t, 1, summary.ModelCount)
	published := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)
	require.NotNil(t, published)
	require.Equal(t, []string{"persisted-image"}, publicCatalogItemModels(published.Snapshot.Items))
}

func publicCatalogCandidateTestSnapshot(model string) *PublicModelCatalogSnapshot {
	return &PublicModelCatalogSnapshot{
		ETag:      "etag-" + model,
		UpdatedAt: "2026-04-27T00:00:00Z",
		PageSize:  10,
		Items: []PublicModelCatalogItem{{
			Model:       model,
			DisplayName: model,
			Provider:    PlatformOpenAI,
			Mode:        "image",
			Currency:    ModelPricingCurrencyUSD,
			PriceDisplay: PublicModelCatalogPriceDisplay{
				Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitImage, Value: 0.08}},
			},
			MultiplierSummary: PublicModelCatalogMultiplierSummary{
				Enabled: false,
				Kind:    publicModelCatalogMultiplierDisabled,
			},
		}},
	}
}

func publicCatalogItemModels(items []PublicModelCatalogItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Model)
	}
	return out
}
