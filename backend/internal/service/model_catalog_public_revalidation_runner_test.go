//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPublicModelCatalogRevalidationRunner_RunOnceSkipsWhenDisabled(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, runnerPublishedSnapshot()))
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	runner := NewPublicModelCatalogRevalidationRunner(svc, time.Hour)
	before := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)

	runner.runOnce(ctx)

	after := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)
	require.Equal(t, before.Snapshot.LastRevalidatedAt, after.Snapshot.LastRevalidatedAt)
	require.Equal(t, before.Snapshot.StaleReason, after.Snapshot.StaleReason)
}

func TestPublicModelCatalogRevalidationRunner_RunOnceRevalidatesWhenEnabled(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{
		SettingKeyPublicModelCatalogAutoRevalidateEnabled: "true",
	}}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, runnerPublishedSnapshot()))
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	runner := NewPublicModelCatalogRevalidationRunner(svc, time.Hour)
	before := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)

	runner.runOnce(ctx)

	after := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)
	require.NotEqual(t, before.Snapshot.LastRevalidatedAt, after.Snapshot.LastRevalidatedAt)
	require.Empty(t, after.Snapshot.StaleReason)
}

func TestPublicModelCatalogRevalidationRunner_RunOnceHandlesMissingPublishedSnapshot(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{
		SettingKeyPublicModelCatalogAutoRevalidateEnabled: "true",
	}}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	runner := NewPublicModelCatalogRevalidationRunner(svc, time.Hour)

	require.NotPanics(t, func() {
		runner.runOnce(context.Background())
	})
}

func TestPublicModelCatalogRevalidationRunner_StopPreventsFurtherTicks(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{
		SettingKeyPublicModelCatalogAutoRevalidateEnabled: "true",
	}}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, runnerPublishedSnapshot()))
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	runner := NewPublicModelCatalogRevalidationRunner(svc, 10*time.Millisecond)
	runner.Start()
	require.Eventually(t, func() bool {
		published := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)
		return published != nil && published.Snapshot.LastRevalidatedAt != "2026-05-01T00:00:00Z"
	}, time.Second, 10*time.Millisecond)

	runner.Stop()
	stoppedAt := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot).Snapshot.LastRevalidatedAt
	time.Sleep(40 * time.Millisecond)

	after := loadPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot)
	require.Equal(t, stoppedAt, after.Snapshot.LastRevalidatedAt)
}

func runnerPublishedSnapshot() *PublicModelCatalogPublishedSnapshot {
	return &PublicModelCatalogPublishedSnapshot{
		Snapshot: PublicModelCatalogSnapshot{
			ETag:              "etag-runner",
			UpdatedAt:         "2026-05-01T00:00:00Z",
			PublishedAt:       "2026-05-01T00:00:00Z",
			LastRevalidatedAt: "2026-05-01T00:00:00Z",
			PageSize:          10,
			Items: []PublicModelCatalogItem{{
				Model:             "gpt-5.4",
				PublicModelID:     "gpt-5.4",
				Status:            PublicModelStatusOK,
				AvailabilityState: AccountModelAvailabilityVerified,
				StaleState:        AccountModelStaleStateFresh,
				Currency:          ModelPricingCurrencyUSD,
				PriceDisplay: PublicModelCatalogPriceDisplay{
					Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1e-6}},
				},
				MultiplierSummary: PublicModelCatalogMultiplierSummary{Enabled: false, Kind: publicModelCatalogMultiplierDisabled},
			}},
		},
	}
}
