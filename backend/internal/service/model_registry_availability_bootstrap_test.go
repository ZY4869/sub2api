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
	require.Contains(t, availableSet, "deepseek-v4-flash")
	require.Contains(t, availableSet, "deepseek-v4-pro")
	require.NotContains(t, availableSet, "gpt-5.4-nano")
	require.Contains(t, availableSet, "gemini-3.1-flash-image")
	require.Contains(t, availableSet, "gemini-3.1-flash-image-preview")
	require.Contains(t, availableSet, "gemini-3.1-pro-preview")
	require.Contains(t, availableSet, "gemini-3-pro-image")
	require.Contains(t, availableSet, "gemini-2.5-flash-image")
	require.NotContains(t, availableSet, "gemini-2.5-flash-image-preview")
	require.NotContains(t, availableSet, "gemini-3-pro-preview")
	require.NotContains(t, availableSet, "unknown")

	require.True(t, svc.IsModelAvailable(ctx, "claude-opus-4.1"))
	require.True(t, svc.IsModelAvailable(ctx, "claude-opus-4-6"))
	require.True(t, svc.IsModelAvailable(ctx, "claude-opus-4-7"))
	require.True(t, svc.IsModelAvailable(ctx, "claude-sonnet-4-6"))
	require.False(t, svc.IsModelAvailable(ctx, "claude-sonnet-4-5"))
	require.False(t, svc.IsModelAvailable(ctx, "claude-haiku-4-5"))
	require.True(t, svc.IsModelAvailable(ctx, "gpt-5.4-mini"))

	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260313])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260317])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260416])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260417])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260513])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260511])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260512])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260512Phase2])
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
	firstMarkerV20260513 := repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260513]

	require.True(t, svc.IsModelAvailable(ctx, "gpt-5.4-pro"))
	require.Equal(t, firstAvailable, repo.values[SettingKeyModelRegistryAvailableModels])
	require.Equal(t, firstRuntimeEntries, repo.values[SettingKeyModelRegistryEntries])
	require.Equal(t, firstMarkerV20260313, repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260313])
	require.Equal(t, firstMarkerV20260317, repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260317])
	require.Equal(t, firstMarkerV20260416, repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260416])
	require.Equal(t, firstMarkerV20260417, repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260417])
	require.Equal(t, firstMarkerV20260513, repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260513])
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
	require.Contains(t, availableSet, "deepseek-v4-flash")
	require.Contains(t, availableSet, "deepseek-v4-pro")
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260313])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260317])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260416])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260417])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260513])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260511])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260512])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260512Phase2])
}

func TestModelRegistryService_AvailableBootstrapV20260513_BackfillsDeepSeekV4ModelsForExistingInstances(t *testing.T) {
	ctx := context.Background()
	repo := newAccountModelImportSettingRepoStub()
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModels, `["gpt-4o","deepseek-v3"]`))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModelsBootstrapV20260313, "true"))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModelsBootstrapV20260317, "true"))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModelsBootstrapV20260328, "true"))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModelsBootstrapV20260416, "true"))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModelsBootstrapV20260417, "true"))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryHardRemoveCleanupV20260511, "true"))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryHardRemoveCleanupV20260512, "true"))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryHardRemoveCleanupV20260512Phase2, "true"))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryHardRemoveCleanupV20260512Pricing, "true"))

	svc := NewModelRegistryService(repo)

	require.True(t, svc.IsModelAvailable(ctx, "deepseek-v4-flash"))
	require.True(t, svc.IsModelAvailable(ctx, "deepseek-v4-pro"))

	availableSet, err := svc.loadStringSet(ctx, SettingKeyModelRegistryAvailableModels)
	require.NoError(t, err)
	require.Contains(t, availableSet, "deepseek-v4-flash")
	require.Contains(t, availableSet, "deepseek-v4-pro")
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryAvailableModelsBootstrapV20260513])
}

func TestModelRegistryService_HardRemoveCleanupV20260512_CleansLegacyRuntimeState(t *testing.T) {
	ctx := context.Background()
	repo := newAccountModelImportSettingRepoStub()
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModels, `["gpt-4o","gemini-3-pro-preview","gemini-2.5-flash-image-preview","unknown"]`))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryHiddenModels, `["gemini-3-pro-preview","unknown"]`))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryEntries, `[
		{"id":"gemini-3-pro-preview","platforms":["gemini"],"exposed_in":["runtime"]},
		{"id":"gemini-2.5-flash-image-preview","platforms":["antigravity"],"exposed_in":["runtime"]},
		{"id":"unknown","platforms":["gemini"],"exposed_in":["runtime"]},
		{"id":"custom-safe-model","platforms":["openai"],"exposed_in":["runtime"]}
	]`))

	svc := NewModelRegistryService(repo)

	require.True(t, svc.IsModelAvailable(ctx, "gpt-4o"))
	require.False(t, svc.IsModelAvailable(ctx, "gemini-3-pro-preview"))
	require.False(t, svc.IsModelAvailable(ctx, "gemini-2.5-flash-image-preview"))
	require.False(t, svc.IsModelAvailable(ctx, "unknown"))

	availableSet, err := svc.loadStringSet(ctx, SettingKeyModelRegistryAvailableModels)
	require.NoError(t, err)
	require.Contains(t, availableSet, "gpt-4o")
	require.NotContains(t, availableSet, "gemini-3-pro-preview")
	require.NotContains(t, availableSet, "gemini-2.5-flash-image-preview")
	require.NotContains(t, availableSet, "unknown")

	hiddenSet, err := svc.loadStringSet(ctx, SettingKeyModelRegistryHiddenModels)
	require.NoError(t, err)
	require.NotContains(t, hiddenSet, "gemini-3-pro-preview")
	require.NotContains(t, hiddenSet, "unknown")

	tombstones, err := svc.loadStringSet(ctx, SettingKeyModelRegistryTombstones)
	require.NoError(t, err)
	require.Contains(t, tombstones, "gemini-3-pro-preview")
	require.Contains(t, tombstones, "gemini-2.5-flash-image-preview")
	require.Contains(t, tombstones, "unknown")

	runtimeEntries, err := svc.loadRuntimeEntries(ctx)
	require.NoError(t, err)
	runtimeIDs := make([]string, 0, len(runtimeEntries))
	for _, entry := range runtimeEntries {
		runtimeIDs = append(runtimeIDs, entry.ID)
	}
	require.Equal(t, []string{"custom-safe-model"}, runtimeIDs)
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260511])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260512])
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260512Phase2])
}

func TestModelRegistryService_HardRemoveCleanupV20260512Phase2_CleansClaudeDeepSeekAndGrokShells(t *testing.T) {
	ctx := context.Background()
	repo := newAccountModelImportSettingRepoStub()
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModels, `["claude-sonnet-4.5","claude-sonnet-4-5","claude-haiku-4-5","claude-haiku-4-5-20251001","deepseek-v3","deepseek-chat","grok-3-fast","grok-4"]`))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryHiddenModels, `["claude-sonnet-4-5","deepseek-chat","grok-3-fast","grok-4"]`))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryEntries, `[
		{"id":"claude-sonnet-4-5","platforms":["anthropic"],"exposed_in":["runtime"]},
		{"id":"claude-haiku-4-5","platforms":["anthropic"],"exposed_in":["runtime"]},
		{"id":"claude-haiku-4-5-20251001","platforms":["anthropic"],"exposed_in":["runtime"]},
		{"id":"deepseek-chat","platforms":["deepseek"],"exposed_in":["runtime"]},
		{"id":"grok-3-fast","platforms":["grok"],"exposed_in":["runtime"]},
		{"id":"grok-4","platforms":["grok"],"exposed_in":["runtime"]},
		{"id":"custom-safe-model","platforms":["openai"],"exposed_in":["runtime"]}
	]`))

	svc := NewModelRegistryService(repo)

	require.True(t, svc.IsModelAvailable(ctx, "claude-sonnet-4.5"))
	require.False(t, svc.IsModelAvailable(ctx, "claude-sonnet-4-5"))
	require.False(t, svc.IsModelAvailable(ctx, "claude-haiku-4-5"))
	require.False(t, svc.IsModelAvailable(ctx, "claude-haiku-4-5-20251001"))
	require.True(t, svc.IsModelAvailable(ctx, "deepseek-v3"))
	require.False(t, svc.IsModelAvailable(ctx, "deepseek-chat"))
	require.False(t, svc.IsModelAvailable(ctx, "grok-3-fast"))
	require.False(t, svc.IsModelAvailable(ctx, "grok-4"))

	availableSet, err := svc.loadStringSet(ctx, SettingKeyModelRegistryAvailableModels)
	require.NoError(t, err)
	require.Contains(t, availableSet, "claude-sonnet-4.5")
	require.Contains(t, availableSet, "deepseek-v3")
	require.NotContains(t, availableSet, "grok-3-fast")
	require.NotContains(t, availableSet, "claude-sonnet-4-5")
	require.NotContains(t, availableSet, "claude-haiku-4-5")
	require.NotContains(t, availableSet, "claude-haiku-4-5-20251001")
	require.NotContains(t, availableSet, "deepseek-chat")
	require.NotContains(t, availableSet, "grok-4")

	hiddenSet, err := svc.loadStringSet(ctx, SettingKeyModelRegistryHiddenModels)
	require.NoError(t, err)
	require.NotContains(t, hiddenSet, "claude-sonnet-4-5")
	require.NotContains(t, hiddenSet, "deepseek-chat")
	require.NotContains(t, hiddenSet, "grok-3-fast")
	require.NotContains(t, hiddenSet, "grok-4")

	tombstones, err := svc.loadStringSet(ctx, SettingKeyModelRegistryTombstones)
	require.NoError(t, err)
	require.Contains(t, tombstones, "claude-sonnet-4-5")
	require.Contains(t, tombstones, "claude-haiku-4-5")
	require.Contains(t, tombstones, "claude-haiku-4-5-20251001")
	require.Contains(t, tombstones, "deepseek-chat")
	require.Contains(t, tombstones, "grok-3-fast")
	require.Contains(t, tombstones, "grok-4")

	runtimeEntries, err := svc.loadRuntimeEntries(ctx)
	require.NoError(t, err)
	runtimeIDs := make([]string, 0, len(runtimeEntries))
	for _, entry := range runtimeEntries {
		runtimeIDs = append(runtimeIDs, entry.ID)
	}
	require.Equal(t, []string{"custom-safe-model"}, runtimeIDs)
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260512Phase2])
}

func TestModelRegistryService_HardRemoveCleanupV20260512Pricing_PreservesDeepSeekRuntimeModels(t *testing.T) {
	ctx := context.Background()
	repo := newAccountModelImportSettingRepoStub()
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryAvailableModels, `["gpt-4o","deepseek-v4-flash","deepseek-v4-pro","gemini-3-pro-high"]`))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryHiddenModels, `["deepseek-v4-pro","gemini-3-pro-high"]`))
	require.NoError(t, repo.Set(ctx, SettingKeyModelRegistryEntries, `[
		{"id":"deepseek-v4-flash","platforms":["deepseek"],"exposed_in":["runtime"]},
		{"id":"deepseek-v4-pro","platforms":["deepseek"],"exposed_in":["runtime"]},
		{"id":"gemini-3-pro-high","platforms":["gemini"],"exposed_in":["runtime"]},
		{"id":"custom-safe-model","platforms":["openai"],"exposed_in":["runtime"]}
	]`))

	svc := NewModelRegistryService(repo)

	require.True(t, svc.IsModelAvailable(ctx, "deepseek-v4-flash"))
	require.True(t, svc.IsModelAvailable(ctx, "deepseek-v4-pro"))
	require.False(t, svc.IsModelAvailable(ctx, "gemini-3-pro-high"))

	availableSet, err := svc.loadStringSet(ctx, SettingKeyModelRegistryAvailableModels)
	require.NoError(t, err)
	require.Contains(t, availableSet, "deepseek-v4-flash")
	require.Contains(t, availableSet, "deepseek-v4-pro")
	require.NotContains(t, availableSet, "gemini-3-pro-high")

	hiddenSet, err := svc.loadStringSet(ctx, SettingKeyModelRegistryHiddenModels)
	require.NoError(t, err)
	require.Contains(t, hiddenSet, "deepseek-v4-pro")
	require.NotContains(t, hiddenSet, "gemini-3-pro-high")

	tombstones, err := svc.loadStringSet(ctx, SettingKeyModelRegistryTombstones)
	require.NoError(t, err)
	require.NotContains(t, tombstones, "deepseek-v4-flash")
	require.NotContains(t, tombstones, "deepseek-v4-pro")
	require.Contains(t, tombstones, "gemini-3-pro-high")

	runtimeEntries, err := svc.loadRuntimeEntries(ctx)
	require.NoError(t, err)
	runtimeIDs := make([]string, 0, len(runtimeEntries))
	for _, entry := range runtimeEntries {
		runtimeIDs = append(runtimeIDs, entry.ID)
	}
	require.ElementsMatch(t, []string{"custom-safe-model", "deepseek-v4-flash", "deepseek-v4-pro"}, runtimeIDs)
	require.Equal(t, "true", repo.values[SettingKeyModelRegistryHardRemoveCleanupV20260512Pricing])
}
