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
	require.Empty(t, payload.AvailableItems)

	overwritten := loadPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot)
	require.NotNil(t, overwritten)
	require.Empty(t, overwritten.Items)

	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, publicCatalogCandidateTestSnapshot("gpt-image-2")))
	summary, err := svc.PublishPublicModelCatalog(ctx, ModelCatalogActor{UserID: 1, Email: "admin@example.test"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"gpt-image-2"},
		PageSize:       10,
	})
	require.NoError(t, err)
	require.Equal(t, 1, summary.ModelCount)
	published := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)
	require.NotNil(t, published)
	require.Equal(t, []string{"gpt-image-2"}, publicCatalogItemModels(published.Snapshot.Items))
}

func TestModelCatalogService_PublishedSnapshotKeepsInternalAccountIDButPublicViewsSanitize(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	persisted := publicCatalogCandidateTestSnapshot("gpt-5.4")
	persisted.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	persisted.Items[0].EntryID = "entry-openai-a"
	persisted.Items[0].PublicModelID = "gpt-5.4@team-a"
	persisted.Items[0].Model = "gpt-5.4@team-a"
	persisted.Items[0].BaseModel = "gpt-5.4"
	persisted.Items[0].SourceModelID = "gpt-5.4"
	persisted.Items[0].SourceProtocol = PlatformOpenAI
	persisted.Items[0].SourceAlias = "Team A"
	persisted.Items[0].SourceAccountID = 42
	persisted.Items[0].SourceAccountName = "Real Account Name"
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, persisted))

	svc := NewModelCatalogService(repo, nil, nil, nil, nil)
	summary, err := svc.PublishPublicModelCatalog(ctx, ModelCatalogActor{UserID: 1, Email: "admin@example.test"}, &PublicModelCatalogDraft{
		SelectedEntries: []PublicModelCatalogEntryDraft{{
			EntryID:       "entry-openai-a",
			PublicModelID: "gpt-5.4-public",
			SourceAlias:   "Team A",
			SalePriceDisplay: PublicModelCatalogPriceDisplay{
				Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldOutputPrice, Unit: "token", Value: 9}},
			},
		}},
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Equal(t, 1, summary.ModelCount)

	internal := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)
	require.NotNil(t, internal)
	require.Equal(t, int64(42), internal.Snapshot.Items[0].SourceAccountID)
	require.Equal(t, "Real Account Name", internal.Snapshot.Items[0].SourceAccountName)
	require.Equal(t, int64(42), internal.Details["gpt-5.4-public"].Item.SourceAccountID)
	require.Equal(t, "Real Account Name", internal.Details["gpt-5.4-public"].Item.SourceAccountName)

	entry, ok, err := svc.ResolvePublishedPublicCatalogEntry(ctx, "gpt-5.4-public")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(42), entry.SourceAccountID)
	require.Equal(t, "gpt-5.4", entry.SourceModelID)
	require.Equal(t, 9.0, entry.SalePriceDisplay.Primary[0].Value)

	publicSnapshot, err := svc.PublishedPublicModelCatalogSnapshot(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), publicSnapshot.Items[0].SourceAccountID)
	require.Empty(t, publicSnapshot.Items[0].SourceAccountName)

	publicDetail, err := svc.PublishedPublicModelCatalogDetail(ctx, "gpt-5.4-public")
	require.NoError(t, err)
	require.Equal(t, int64(0), publicDetail.Item.SourceAccountID)
	require.Empty(t, publicDetail.Item.SourceAccountName)
}

func TestModelCatalogService_PublishPublicModelCatalog_MatchesSelectedEntryByStableSourceWhenEntryIDChanges(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	persisted := publicCatalogCandidateTestSnapshot("gpt-5.4")
	persisted.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	persisted.Items[0].EntryID = "entry-current"
	persisted.Items[0].PublicModelID = "gpt-5.4@team-a"
	persisted.Items[0].Model = "gpt-5.4@team-a"
	persisted.Items[0].BaseModel = "gpt-5.4"
	persisted.Items[0].SourceModelID = "gpt-5.4"
	persisted.Items[0].SourceProtocol = PlatformOpenAI
	persisted.Items[0].SourceAlias = "Team A"
	persisted.Items[0].SourceAccountID = 42
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, persisted))

	svc := NewModelCatalogService(repo, nil, nil, nil, nil)
	summary, err := svc.PublishPublicModelCatalog(ctx, ModelCatalogActor{UserID: 1, Email: "admin@example.test"}, &PublicModelCatalogDraft{
		SelectedEntries: []PublicModelCatalogEntryDraft{{
			EntryID:         "entry-stale",
			PublicModelID:   "gpt-5.4-public",
			SourceAccountID: 42,
			SourceAlias:     "Team A",
			SourceModelID:   "gpt-5.4",
			BaseModel:       "gpt-5.4",
			SourceProtocol:  PlatformOpenAI,
			SalePriceDisplay: PublicModelCatalogPriceDisplay{
				Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldOutputPrice, Unit: "token", Value: 9}},
			},
		}},
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Equal(t, 1, summary.ModelCount)

	published := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)
	require.NotNil(t, published)
	require.Equal(t, "entry-current", published.Snapshot.Items[0].EntryID)
	require.Equal(t, "gpt-5.4-public", published.Snapshot.Items[0].Model)
	require.Equal(t, int64(42), published.Snapshot.Items[0].SourceAccountID)
	require.Equal(t, 9.0, published.Snapshot.Items[0].SalePriceDisplay.Primary[0].Value)
}

func TestModelCatalogService_PublishPublicModelCatalog_RejectsUnavailableSelectedEntry(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	persisted := publicCatalogCandidateTestSnapshot("gpt-5.4")
	persisted.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, persisted))

	svc := NewModelCatalogService(repo, nil, nil, nil, nil)
	_, err := svc.PublishPublicModelCatalog(ctx, ModelCatalogActor{UserID: 1, Email: "admin@example.test"}, &PublicModelCatalogDraft{
		SelectedEntries: []PublicModelCatalogEntryDraft{{
			EntryID:       "missing-entry",
			PublicModelID: "gpt-5.4@missing",
		}},
		PageSize: 10,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PUBLIC_MODEL_ENTRY_UNAVAILABLE")
}

func TestModelCatalogService_PublishPublicModelCatalog_RejectsStalePersistedCandidate(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	persisted := publicCatalogCandidateTestSnapshot("gpt-5.4")
	persisted.UpdatedAt = time.Now().Add(-publicModelCatalogDraftLiveTTL - time.Minute).UTC().Format(time.RFC3339)
	persisted.Items[0].EntryID = "entry-stale"
	persisted.Items[0].PublicModelID = "gpt-5.4@stale"
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, persisted))

	svc := NewModelCatalogService(repo, nil, nil, nil, nil)
	_, err := svc.PublishPublicModelCatalog(ctx, ModelCatalogActor{UserID: 1, Email: "admin@example.test"}, &PublicModelCatalogDraft{
		SelectedEntries: []PublicModelCatalogEntryDraft{{
			EntryID:       "entry-stale",
			PublicModelID: "gpt-5.4@stale",
		}},
		PageSize: 10,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PUBLIC_MODEL_ENTRY_UNAVAILABLE")
}

func publicCatalogCandidateTestSnapshot(model string) *PublicModelCatalogSnapshot {
	return &PublicModelCatalogSnapshot{
		ETag:      "etag-" + model,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		PageSize:  10,
		Items: []PublicModelCatalogItem{{
			Model:             model,
			PublicModelID:     model,
			BaseModel:         model,
			SourceModelID:     model,
			DisplayName:       model,
			Provider:          PlatformOpenAI,
			ProviderIconKey:   PlatformOpenAI,
			SourceProtocol:    PlatformOpenAI,
			Status:            PublicModelStatusOK,
			AvailabilityState: AccountModelAvailabilityVerified,
			StaleState:        AccountModelStaleStateFresh,
			LifecycleStatus:   PublicModelLifecycleStable,
			RequestProtocols:  []string{PlatformOpenAI},
			Mode:              inferModelMode(model, ""),
			Currency:          ModelPricingCurrencyUSD,
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
