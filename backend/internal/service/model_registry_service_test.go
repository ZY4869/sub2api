package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

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
}

func TestModelRegistryService_UpsertEntry_NormalizesLegacyCapabilities(t *testing.T) {
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	entry, err := svc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:           "sora-test-capability",
		Platforms:    []string{PlatformSora},
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
