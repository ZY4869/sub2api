package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelRegistryService_ListProviderSummaries_SortsAndPaginates(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	baselineItems, baselineTotal, err := svc.ListProviderSummaries(context.Background(), 1, 500)
	require.NoError(t, err)
	maxTotalCount := 0
	for _, item := range baselineItems {
		if item.TotalCount > maxTotalCount {
			maxTotalCount = item.TotalCount
		}
	}

	providerCounts := []struct {
		provider       string
		platform       string
		totalCount     int
		availableCount int
	}{
		{provider: "zz-provider-summary-top", platform: PlatformOpenAI, totalCount: maxTotalCount + 3, availableCount: 2},
		{provider: "yy-provider-summary-mid", platform: PlatformAnthropic, totalCount: maxTotalCount + 2, availableCount: 1},
		{provider: "xx-provider-summary-low", platform: PlatformGemini, totalCount: maxTotalCount + 1, availableCount: 1},
	}

	activateModels := make([]string, 0, 4)
	for _, provider := range providerCounts {
		for index := 0; index < provider.totalCount; index++ {
			modelID := provider.provider + "-" + string(rune('a'+(index%26))) + "-" + string(rune('a'+((index/26)%26))) + "-" + string(rune('a'+((index/676)%26)))
			_, err = svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
				ID:        modelID,
				Provider:  provider.provider,
				Platforms: []string{provider.platform},
				ExposedIn: []string{"runtime"},
			})
			require.NoError(t, err)
			if index < provider.availableCount {
				activateModels = append(activateModels, modelID)
			}
		}
	}

	_, err = svc.ActivateModels(context.Background(), activateModels)
	require.NoError(t, err)

	items, total, err := svc.ListProviderSummaries(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, baselineTotal+3, total)
	require.Len(t, items, 2)
	require.Equal(t, ModelRegistryProviderSummary{
		Provider:       "zz-provider-summary-top",
		TotalCount:     maxTotalCount + 3,
		AvailableCount: 2,
	}, items[0])
	require.Equal(t, ModelRegistryProviderSummary{
		Provider:       "yy-provider-summary-mid",
		TotalCount:     maxTotalCount + 2,
		AvailableCount: 1,
	}, items[1])

	items, total, err = svc.ListProviderSummaries(context.Background(), 2, 2)
	require.NoError(t, err)
	require.Equal(t, baselineTotal+3, total)
	require.Len(t, items, 2)
	require.Equal(t, ModelRegistryProviderSummary{
		Provider:       "xx-provider-summary-low",
		TotalCount:     maxTotalCount + 1,
		AvailableCount: 1,
	}, items[0])
}

func TestModelRegistryService_List_CategoryLatestSortsByCategoryThenPriority(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	inputs := []UpsertModelRegistryEntryInput{
		{
			ID:           "provider-sort-text-old",
			Provider:     "provider-sort",
			Platforms:    []string{PlatformOpenAI},
			Modalities:   []string{"text"},
			Capabilities: []string{"text"},
			UIPriority:   20,
			ExposedIn:    []string{"runtime"},
		},
		{
			ID:           "provider-sort-image-new",
			Provider:     "provider-sort",
			Platforms:    []string{PlatformOpenAI},
			Modalities:   []string{"text", "image"},
			Capabilities: []string{"image_generation"},
			UIPriority:   2,
			ExposedIn:    []string{"runtime"},
		},
		{
			ID:           "provider-sort-audio-new",
			Provider:     "provider-sort",
			Platforms:    []string{PlatformOpenAI},
			Modalities:   []string{"audio"},
			Capabilities: []string{"audio_understanding"},
			UIPriority:   1,
			ExposedIn:    []string{"runtime"},
		},
		{
			ID:           "provider-sort-video-new",
			Provider:     "provider-sort",
			Platforms:    []string{PlatformOpenAI},
			Modalities:   []string{"video"},
			Capabilities: []string{"video_generation"},
			UIPriority:   1,
			ExposedIn:    []string{"runtime"},
		},
		{
			ID:           "provider-sort-text-new",
			Provider:     "provider-sort",
			Platforms:    []string{PlatformOpenAI},
			Modalities:   []string{"text"},
			Capabilities: []string{"text"},
			UIPriority:   1,
			ExposedIn:    []string{"runtime"},
		},
	}
	for _, input := range inputs {
		_, err := svc.UpsertEntry(context.Background(), input)
		require.NoError(t, err)
	}

	items, total, err := svc.List(context.Background(), ModelRegistryListFilter{
		Provider:          "provider-sort",
		Availability:      "all",
		SortMode:          "category_latest",
		IncludeHidden:     true,
		IncludeTombstoned: true,
		Page:              1,
		PageSize:          20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 5, total)
	require.Len(t, items, 5)
	require.Equal(t, []string{
		"provider-sort-text-new",
		"provider-sort-text-old",
		"provider-sort-image-new",
		"provider-sort-video-new",
		"provider-sort-audio-new",
	}, []string{items[0].ID, items[1].ID, items[2].ID, items[3].ID, items[4].ID})
}

func TestModelRegistryService_BatchSyncExposures_MergesAndIsIdempotent(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	_, err := svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:        "gpt-test-sync",
		Platforms: []string{PlatformOpenAI},
		ExposedIn: []string{"runtime", "test"},
	})
	require.NoError(t, err)

	result, err := svc.BatchSyncExposures(context.Background(), BatchSyncModelRegistryExposuresInput{
		Models:    []string{"gpt-test-sync"},
		Exposures: []string{"whitelist", "use_key", "runtime"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"whitelist", "use_key", "runtime"}, result.Exposures)
	require.Equal(t, "add", result.Mode)
	require.Equal(t, 1, result.UpdatedCount)
	require.Zero(t, result.SkippedCount)
	require.Zero(t, result.FailedCount)
	require.Equal(t, []string{"gpt-test-sync"}, result.UpdatedModels)

	detail, err := svc.GetDetail(context.Background(), "gpt-test-sync")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"runtime", "test", "whitelist", "use_key"}, detail.ExposedIn)

	result, err = svc.BatchSyncExposures(context.Background(), BatchSyncModelRegistryExposuresInput{
		Models:    []string{"gpt-test-sync"},
		Exposures: []string{"use_key", "whitelist", "runtime"},
	})
	require.NoError(t, err)
	require.Zero(t, result.UpdatedCount)
	require.Equal(t, 1, result.SkippedCount)
	require.Equal(t, []string{"gpt-test-sync"}, result.SkippedModels)
}

func TestModelRegistryService_BatchSyncExposures_RemoveModeRemovesTargetsOnly(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	_, err := svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:        "gpt-test-remove",
		Platforms: []string{PlatformOpenAI},
		ExposedIn: []string{"runtime", "test", "whitelist"},
	})
	require.NoError(t, err)

	result, err := svc.BatchSyncExposures(context.Background(), BatchSyncModelRegistryExposuresInput{
		Models:    []string{"gpt-test-remove"},
		Exposures: []string{"test"},
		Mode:      "remove",
	})
	require.NoError(t, err)
	require.Equal(t, "remove", result.Mode)
	require.Equal(t, 1, result.UpdatedCount)

	detail, err := svc.GetDetail(context.Background(), "gpt-test-remove")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"runtime", "whitelist"}, detail.ExposedIn)
}

func TestModelRegistryService_BatchSyncExposures_SkipsTombstonedAndMissingModels(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	_, err := svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:        "gpt-test-deleted",
		Platforms: []string{PlatformOpenAI},
		ExposedIn: []string{"runtime"},
	})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteEntry(context.Background(), "gpt-test-deleted"))

	result, err := svc.BatchSyncExposures(context.Background(), BatchSyncModelRegistryExposuresInput{
		Models:    []string{"gpt-test-deleted", "missing-model"},
		Exposures: []string{"whitelist"},
	})
	require.NoError(t, err)
	require.Zero(t, result.UpdatedCount)
	require.Equal(t, 2, result.SkippedCount)
	require.ElementsMatch(t, []string{"gpt-test-deleted", "missing-model"}, result.SkippedModels)
}

func TestModelRegistryService_BatchSyncExposures_RejectsInvalidInput(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	_, err := svc.BatchSyncExposures(context.Background(), BatchSyncModelRegistryExposuresInput{
		Models:    []string{"   "},
		Exposures: []string{"whitelist"},
	})
	require.Error(t, err)

	_, err = svc.BatchSyncExposures(context.Background(), BatchSyncModelRegistryExposuresInput{
		Models:    []string{"gpt-test-sync"},
		Exposures: []string{"invalid"},
	})
	require.Error(t, err)

	_, err = svc.BatchSyncExposures(context.Background(), BatchSyncModelRegistryExposuresInput{
		Models:    []string{"gpt-test-sync"},
		Exposures: []string{"test"},
		Mode:      "invalid",
	})
	require.Error(t, err)
}

func TestModelRegistryService_List_FiltersByExposureAndStatus(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	_, err := svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:        "provider-filter-stable-test",
		Provider:  "provider-filter",
		Platforms: []string{PlatformOpenAI},
		ExposedIn: []string{"runtime", "test"},
	})
	require.NoError(t, err)
	_, err = svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:        "provider-filter-deprecated-test",
		Provider:  "provider-filter",
		Platforms: []string{PlatformOpenAI},
		ExposedIn: []string{"test"},
		Status:    "deprecated",
	})
	require.NoError(t, err)
	_, err = svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:        "provider-filter-runtime-only",
		Provider:  "provider-filter",
		Platforms: []string{PlatformOpenAI},
		ExposedIn: []string{"runtime"},
	})
	require.NoError(t, err)

	items, total, err := svc.List(context.Background(), ModelRegistryListFilter{
		Provider:          "provider-filter",
		Exposure:          "test",
		Status:            "deprecated",
		Availability:      "all",
		IncludeHidden:     true,
		IncludeTombstoned: true,
		Page:              1,
		PageSize:          20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, "provider-filter-deprecated-test", items[0].ID)
}

func TestModelRegistryService_HardDeleteModels_TombstonesAndClearsState(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	_, err := svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:        "gpt-test-hard-delete",
		Platforms: []string{PlatformOpenAI},
		ExposedIn: []string{"runtime"},
	})
	require.NoError(t, err)

	_, err = svc.ActivateModels(context.Background(), []string{"gpt-test-hard-delete", "gpt-3.5-turbo"})
	require.NoError(t, err)

	_, err = svc.SetVisibility(context.Background(), UpdateModelRegistryVisibilityInput{
		Model:  "gpt-test-hard-delete",
		Hidden: true,
	})
	require.NoError(t, err)

	deleted, err := svc.HardDeleteModels(context.Background(), []string{"gpt-3.5-turbo", "gpt-test-hard-delete"})
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-3.5-turbo", "gpt-test-hard-delete"}, deleted)

	runtimeDetail, err := svc.GetDetail(context.Background(), "gpt-test-hard-delete")
	require.NoError(t, err)
	require.True(t, runtimeDetail.Tombstoned)
	require.False(t, runtimeDetail.Available)
	require.False(t, runtimeDetail.Hidden)

	seedDetail, err := svc.GetDetail(context.Background(), "gpt-3.5-turbo")
	require.NoError(t, err)
	require.True(t, seedDetail.Tombstoned)
	require.False(t, seedDetail.Available)
	require.False(t, seedDetail.Hidden)

	repeated, err := svc.HardDeleteModels(context.Background(), []string{"gpt-test-hard-delete", "gpt-3.5-turbo"})
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-3.5-turbo", "gpt-test-hard-delete"}, repeated)
}

func TestModelRegistryService_HardDeleteModels_RejectsEmptyInput(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	_, err := svc.HardDeleteModels(context.Background(), []string{"   "})
	require.Error(t, err)
}

func TestModelRegistryService_UpsertEntry_NormalizesLegacyCapabilities(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	entry, err := svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:           "gpt-test-capability",
		Platforms:    []string{PlatformOpenAI},
		Capabilities: []string{"video", "reasoning", "image", "video"},
		ExposedIn:    []string{"runtime"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"text", "image_generation", "video_generation"}, entry.Capabilities)
}

func TestModelRegistryService_UpsertEntry_RejectsUnknownCapabilities(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	_, err := svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:           "gpt-test-invalid-capability",
		Platforms:    []string{PlatformOpenAI},
		Capabilities: []string{"unsupported_capability"},
		ExposedIn:    []string{"runtime"},
	})
	require.Error(t, err)
}
