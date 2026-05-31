package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
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
	persisted.RefreshedAt = time.Now().UTC().Format(time.RFC3339)
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

func TestModelCatalogService_GetPublicModelCatalogDraftPayload_SkipsExpiredPersistedCandidate(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	expired := publicCatalogCandidateTestSnapshot("stale-model")
	expired.UpdatedAt = time.Now().Add(-publicModelCatalogDraftLiveTTL - time.Minute).UTC().Format(time.RFC3339)
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, expired))

	svc := NewModelCatalogService(repo, nil, nil, nil, nil)
	fresh := publicCatalogCandidateTestSnapshot("fresh-cache-model")
	svc.storePublicModelCatalogSnapshot(fresh)

	payload, err := svc.GetPublicModelCatalogDraftPayload(ctx, false)
	require.NoError(t, err)
	require.Equal(t, publicModelCatalogDraftAvailableSourceCache, payload.AvailableSource)
	require.Equal(t, []string{"fresh-cache-model"}, publicCatalogItemModels(payload.AvailableItems))
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

func TestModelCatalogService_PublishPublicModelCatalog_FreezesCapabilityMetadata(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	persisted := publicCatalogCandidateTestSnapshot("gpt-5.4")
	persisted.Items[0].ContextWindowTokens = 128000
	persisted.Items[0].Capabilities = []string{"text", "tools"}
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, persisted))

	svc := NewModelCatalogService(repo, nil, nil, nil, nil)
	_, err := svc.PublishPublicModelCatalog(ctx, ModelCatalogActor{UserID: 1, Email: "admin@example.test"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"gpt-5.4"},
		PageSize:       10,
	})
	require.NoError(t, err)

	published := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)
	require.NotNil(t, published)
	item := published.Snapshot.Items[0]
	require.Equal(t, int64(128000), item.ContextWindow.Tokens)
	require.Equal(t, PublicModelCapabilitySourcePublishedSnapshot, item.ContextWindow.Source)
	require.NotEmpty(t, item.ProtocolEndpoints)
	require.NotEmpty(t, item.CapabilityMatrix)
	require.Contains(t, item.RequestProtocols, PlatformOpenAI)
	require.Contains(t, item.Capabilities, "text")
	require.Equal(t, PublicModelCatalogExampleValidationDryRunContract, published.Details["gpt-5.4"].ExampleValidation)
}

func TestSanitizePublicModelCatalogItemForPublic_RemovesSourceAccountAndKeepsCapabilityMetadata(t *testing.T) {
	item := PublicModelCatalogItem{
		Model:             "gpt-5.4-public",
		PublicModelID:     "gpt-5.4-public",
		SourceAccountID:   42,
		SourceAccountName: "private-account",
		SourceAlias:       "private-source",
		SourceModelID:     "gpt-5.4",
		SourceProtocol:    PlatformOpenAI,
		ContextWindow: PublicModelContextWindow{
			Tokens:   128000,
			Source:   PublicModelCapabilitySourceVerifiedProbe,
			Verified: true,
		},
		ProtocolEndpoints: []PublicModelProtocolEndpoint{{
			Key:      "openai.responses",
			Protocol: PlatformOpenAI,
			Support:  PublicModelSupportSupported,
			Source:   PublicModelCapabilitySourceVerifiedProbe,
			Verified: true,
		}},
		CapabilityMatrix: []PublicModelCapabilityMatrixEntry{{
			Capability: "text",
			Protocol:   PlatformOpenAI,
			Endpoint:   "openai.responses",
			Support:    PublicModelSupportSupported,
			Source:     PublicModelCapabilitySourceVerifiedProbe,
			Verified:   true,
		}},
	}

	sanitized := sanitizePublicModelCatalogItemForPublicWithSource(item, PublicModelCatalogSourcePublished)

	require.Zero(t, sanitized.SourceAccountID)
	require.Empty(t, sanitized.SourceAccountName)
	require.Empty(t, sanitized.SourceAlias)
	require.Empty(t, sanitized.SourceModelID)
	require.Equal(t, int64(128000), sanitized.ContextWindow.Tokens)
	require.Equal(t, PublicModelCapabilitySourceVerifiedProbe, sanitized.ContextWindow.Source)
	require.NotEmpty(t, sanitized.ProtocolEndpoints)
	require.NotEmpty(t, sanitized.CapabilityMatrix)
}

func TestModelCatalogService_PublishPublicModelCatalog_RejectsDemoEntry(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	persisted := publicCatalogCandidateTestSnapshot("demo-model")
	persisted.Items[0].IsDemo = true
	persisted.Items[0].CatalogEntrySource = PublicModelCatalogEntrySourceDemo
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, persisted))

	svc := NewModelCatalogService(repo, nil, nil, nil, nil)
	_, err := svc.PublishPublicModelCatalog(ctx, ModelCatalogActor{UserID: 1, Email: "admin@example.test"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"demo-model"},
		PageSize:       10,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "PUBLIC_MODEL_DEMO_ENTRY_FORBIDDEN")
}

func TestModelCatalogService_PublicCatalogReadModesFilterDemoEntries(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	persisted := publicCatalogCandidateTestSnapshot("real-model")
	persisted.Items = append(persisted.Items, clonePublicModelCatalogItem(persisted.Items[0]))
	persisted.Items[1].Model = "demo-model"
	persisted.Items[1].PublicModelID = "demo-model"
	persisted.Items[1].IsDemo = true
	persisted.Items[1].CatalogEntrySource = PublicModelCatalogEntrySourceDemo
	published := &PublicModelCatalogPublishedSnapshot{
		Snapshot: *persisted,
		Details: map[string]PublicModelCatalogDetail{
			"real-model": {Item: persisted.Items[0]},
			"demo-model": {Item: persisted.Items[1]},
		},
	}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, published))
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, persisted))

	disabledSvc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	realSnapshot, err := disabledSvc.PublicModelCatalogSnapshotWithOptions(ctx, PublicModelCatalogReadOptions{CatalogMode: PublicModelCatalogModeDemo})
	require.NoError(t, err)
	require.Equal(t, []string{"real-model"}, publicCatalogItemModels(realSnapshot.Items))
	_, err = disabledSvc.PublicModelCatalogDetailWithOptions(ctx, "demo-model", PublicModelCatalogReadOptions{CatalogMode: PublicModelCatalogModeDemo})
	require.Error(t, err)

	enabledSvc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{
		PublicModelCatalog: config.PublicModelCatalogConfig{DemoMode: true},
	})
	demoSnapshot, err := enabledSvc.PublicModelCatalogSnapshotWithOptions(ctx, PublicModelCatalogReadOptions{CatalogMode: PublicModelCatalogModeDemo})
	require.NoError(t, err)
	require.Equal(t, []string{"demo-model"}, publicCatalogItemModels(demoSnapshot.Items))
	demoDetail, err := enabledSvc.PublicModelCatalogDetailWithOptions(ctx, "demo-model", PublicModelCatalogReadOptions{CatalogMode: PublicModelCatalogModeDemo})
	require.NoError(t, err)
	require.True(t, demoDetail.Item.IsDemo)

	draft, err := enabledSvc.GetPublicModelCatalogDraftPayloadWithOptions(ctx, false, PublicModelCatalogReadOptions{CatalogMode: PublicModelCatalogModeDemo})
	require.NoError(t, err)
	require.Equal(t, []string{"demo-model"}, publicCatalogItemModels(draft.AvailableItems))
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
