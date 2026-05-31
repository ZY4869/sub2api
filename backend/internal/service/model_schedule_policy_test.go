package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/stretchr/testify/require"
)

func TestModelRegistryEntryCurrentlyAvailable_ScheduledAndExpired(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	future := modelregistry.ModelEntry{
		ID:            "future-model",
		AvailableFrom: now.Add(time.Hour).Format(time.RFC3339),
	}
	expired := modelregistry.ModelEntry{
		ID:             "expired-model",
		AvailableUntil: now.Add(-time.Hour).Format(time.RFC3339),
	}
	active := modelregistry.ModelEntry{
		ID:            "active-model",
		AvailableFrom: now.Add(-time.Hour).Format(time.RFC3339),
	}

	require.False(t, modelRegistryEntryCurrentlyAvailable(future, now))
	require.Equal(t, ModelRegistryScheduleScheduled, modelRegistryScheduleStatus(future, now))
	require.False(t, modelRegistryEntryCurrentlyAvailable(expired, now))
	require.Equal(t, ModelRegistryScheduleExpired, modelRegistryScheduleStatus(expired, now))
	require.True(t, modelRegistryEntryCurrentlyAvailable(active, now))
}

func TestPublicModelCatalogItemCurrentlyAvailable_TimeWindow(t *testing.T) {
	item := PublicModelCatalogItem{
		Model: "windowed-model",
		AccessTimePolicy: &TimeAccessPolicy{
			Enabled:  true,
			Timezone: "Asia/Singapore",
			WeeklyWindows: []TimeAccessWindow{{
				Days:  []int{1},
				Start: "08:00",
				End:   "20:00",
			}},
		},
	}

	inside := time.Date(2026, 6, 1, 10, 0, 0, 0, time.FixedZone("SGT", 8*3600))
	outside := time.Date(2026, 6, 1, 22, 0, 0, 0, time.FixedZone("SGT", 8*3600))

	require.True(t, publicModelCatalogItemCurrentlyAvailable(item, inside))
	require.False(t, publicModelCatalogItemCurrentlyAvailable(item, outside))
	require.Equal(t, ModelRegistryScheduleOutOfWindow, publicModelCatalogItemScheduleStatus(item, outside))
}

func TestApplyPublicModelCatalogDraftSchedule_ClearsExistingSchedule(t *testing.T) {
	item := PublicModelCatalogItem{
		Model:          "scheduled-model",
		AvailableFrom:  "2026-06-01T00:00:00Z",
		AvailableUntil: "2026-06-30T00:00:00Z",
		AccessTimePolicy: &TimeAccessPolicy{
			Enabled:  true,
			Timezone: "Asia/Singapore",
			WeeklyWindows: []TimeAccessWindow{{
				Days:  []int{1},
				Start: "08:00",
				End:   "20:00",
			}},
		},
	}

	cleared := applyPublicModelCatalogDraftSchedule(item, PublicModelCatalogEntryDraft{})

	require.Empty(t, cleared.AvailableFrom)
	require.Empty(t, cleared.AvailableUntil)
	require.Nil(t, cleared.AccessTimePolicy)
	require.Equal(t, ModelRegistryScheduleActive, cleared.ScheduleStatus)
}

func TestSanitizePublicModelCatalogItemForPublic_HidesScheduleMetadata(t *testing.T) {
	item := PublicModelCatalogItem{
		Model:          "scheduled-model",
		AvailableFrom:  time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		AvailableUntil: time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		ScheduleStatus: ModelRegistryScheduleActive,
		AccessTimePolicy: &TimeAccessPolicy{
			Enabled:  true,
			Timezone: "Asia/Singapore",
			WeeklyWindows: []TimeAccessWindow{{
				Days:  []int{1},
				Start: "08:00",
				End:   "20:00",
			}},
		},
	}

	sanitized := sanitizePublicModelCatalogItemForPublicWithSource(item, PublicModelCatalogSourcePublished)

	require.Empty(t, sanitized.AvailableFrom)
	require.Empty(t, sanitized.AvailableUntil)
	require.Empty(t, sanitized.ScheduleStatus)
	require.Nil(t, sanitized.AccessTimePolicy)
}

func TestModelCatalogService_ResolvePublishedPublicCatalogEntryStatus_TimeWindowDenied(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	snapshot := &PublicModelCatalogSnapshot{
		Items: []PublicModelCatalogItem{{
			Model:         "scheduled-model",
			PublicModelID: "scheduled-model",
			AvailableFrom: future,
		}},
	}
	etag, err := computePublicModelCatalogETag(snapshot)
	require.NoError(t, err)
	snapshot.ETag = etag

	require.NoError(t, svc.persistPublishedPublicModelCatalogSnapshot(ctx, &PublicModelCatalogPublishedSnapshot{
		Snapshot: *snapshot,
	}))

	entry, status, err := svc.ResolvePublishedPublicCatalogEntryStatus(ctx, "scheduled-model")

	require.NoError(t, err)
	require.Nil(t, entry)
	require.Equal(t, PublicCatalogResolutionTimeWindowDenied, status)
}
