package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelRegistryService_AvailableBootstrapAppendsRequestedModelsAndResolvesSeededGPT54Pro(t *testing.T) {
	ctx := context.Background()
	repo := newAccountModelImportSettingRepoStub()
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModels, `["gpt-4o"]`))

	svc := NewModelRegistryService(repo)

	detail, err := svc.GetDetail(ctx, "gpt-5.4-pro")
	require.NoError(t, err)
	require.Equal(t, "seed", detail.Source)
	require.True(t, detail.Available)
	require.Equal(t, "openai", detail.Provider)
	require.ElementsMatch(t, []string{"runtime", "test", "whitelist"}, detail.ExposedIn)

	availableSet, err := svc.loadStringSet(ctx, SettingKeyModelRegistryAvailableModels)
	require.NoError(t, err)
	require.Contains(t, availableSet, "gpt-4o")
	require.Contains(t, availableSet, "claude-opus-4.1")
	require.Contains(t, availableSet, "claude-opus-4-6")
	require.Contains(t, availableSet, "claude-sonnet-4.5")
	require.Contains(t, availableSet, "claude-sonnet-4-6")
	require.Contains(t, availableSet, "claude-haiku-4.5")
	require.Contains(t, availableSet, "gpt-5.2")
	require.Contains(t, availableSet, "gpt-5.4")
	require.Contains(t, availableSet, "gpt-5.4-mini")
	require.Contains(t, availableSet, "gpt-5.4-pro")
	require.Contains(t, availableSet, "claude-opus-4-7")
	require.NotContains(t, availableSet, "gpt-5.4-nano")
	require.Contains(t, availableSet, "gemini-3.1-flash-image")
	require.Contains(t, availableSet, "gemini-3.1-flash-image-preview")
	require.Contains(t, availableSet, "gemini-3.1-pro-preview")
	require.Contains(t, availableSet, "gemini-3-pro-image")
	require.Contains(t, availableSet, "gemini-2.5-flash-image-preview")
	require.Contains(t, availableSet, "gemini-2.5-flash-image")

	require.True(t, svc.IsModelAvailable(ctx, "claude-opus-4.1"))
	require.True(t, svc.IsModelAvailable(ctx, "claude-opus-4-6"))
	require.True(t, svc.IsModelAvailable(ctx, "claude-opus-4-7"))
	require.True(t, svc.IsModelAvailable(ctx, "claude-sonnet-4-6"))
	require.True(t, svc.IsModelAvailable(ctx, "claude-sonnet-4-5"))
	require.True(t, svc.IsModelAvailable(ctx, "claude-haiku-4-5"))
	require.True(t, svc.IsModelAvailable(ctx, "gpt-5.4-mini"))

	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260313])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260317])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260416])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260417])
}

func TestModelRegistryService_AvailableBootstrapIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := newAccountModelImportSettingRepoStub()
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModels, `["gpt-4o"]`))

	svc := NewModelRegistryService(repo)

	require.True(t, svc.IsModelAvailable(ctx, "gpt-5.4-pro"))
	firstAvailable := repo.values[SettingKeyModelRegistryAvailableModels]
	firstRuntimeEntries := repo.values[SettingKeyModelRegistryEntries]
	firstMarkerV20260313 := repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260313]
	firstMarkerV20260317 := repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260317]
	firstMarkerV20260416 := repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260416]
	firstMarkerV20260417 := repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260417]

	require.True(t, svc.IsModelAvailable(ctx, "gpt-5.4-pro"))
	require.Equal(t, firstAvailable, repo.values[SettingKeyModelRegistryAvailableModels])
	require.Equal(t, firstRuntimeEntries, repo.values[SettingKeyModelRegistryEntries])
	require.Equal(t, firstMarkerV20260313, repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260313])
	require.Equal(t, firstMarkerV20260317, repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260317])
	require.Equal(t, firstMarkerV20260416, repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260416])
	require.Equal(t, firstMarkerV20260417, repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260417])
}

func TestModelRegistryService_AvailableBootstrapRunsAfterMigrationWhenSetMissing(t *testing.T) {
	ctx := context.Background()
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)

	require.True(t, svc.IsModelAvailable(ctx, "gpt-5.4-pro"))

	availableSet, err := svc.loadStringSet(ctx, SettingKeyModelRegistryAvailableModels)
	require.NoError(t, err)
	require.Contains(t, availableSet, "gpt-4o")
	require.Contains(t, availableSet, "gpt-5.4-pro")
	require.Contains(t, availableSet, "claude-opus-4.1")
	require.Contains(t, availableSet, "claude-opus-4-6")
	require.Contains(t, availableSet, "claude-opus-4-7")
	require.Contains(t, availableSet, "claude-sonnet-4-6")
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260313])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260317])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260416])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260417])
}
