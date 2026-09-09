package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/stretchr/testify/require"
)

func TestModelCatalogRefresh20260908_RegistrationAvailabilityAndNoFakeExamples(t *testing.T) {
	ctx := context.Background()
	repo := newAccountModelImportSettingRepoStub()
	svc := NewModelRegistryService(repo)
	for _, id := range []string{"gpt-6-astra", "gpt-5.5-pro", "claude-fable-5-1", "gemini-3.8-flash", "gemini-3.7-flash", "gemini-embedding-2", "text-embedding-3-small", "grok-4.6"} {
		detail, err := svc.GetDetail(ctx, id)
		require.NoError(t, err)
		require.True(t, detail.Available, id)
	}
	for _, id := range []string{"gpt-realtime-2.1", "gemini-3.5-transcribe-live", "grok-imagine-video-1.5", "grok-voice-latest"} {
		detail, err := svc.GetDetail(ctx, id)
		require.NoError(t, err)
		require.False(t, detail.Available, id)
		require.True(t, modelregistry.IsCatalogOnly(detail.ModelEntry))
		_, err = svc.ActivateModels(ctx, []string{id})
		require.Error(t, err)
		synced, err := svc.BatchSyncExposures(ctx, BatchSyncModelRegistryExposuresInput{Models: []string{id}, Exposures: []string{"test"}, Mode: "add"})
		require.NoError(t, err)
		require.Len(t, synced.FailedModels, 1)
		_, ok := selectPublicModelCatalogExampleSpec(PublicModelCatalogItem{Model: id}, "text")
		require.False(t, ok)
	}
	snapshot, err := svc.PublicSnapshot(ctx)
	require.NoError(t, err)
	for _, entry := range snapshot.Models {
		require.False(t, modelregistry.IsCatalogOnly(entry), entry.ID)
	}
	// Reinitialization preserves an explicit deactivation after the dated bootstrap.
	_, err = svc.DeactivateModels(ctx, []string{"gemini-3.8-flash"})
	require.NoError(t, err)
	require.False(t, NewModelRegistryService(repo).IsModelAvailable(ctx, "gemini-3.8-flash"))
}

func TestModelCatalogRefresh20260908_AliasPolicyDoesNotExpandWithLibrary(t *testing.T) {
	ctx := context.Background()
	registry := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	account := &Account{ID: 9922, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: map[string]any{
		"model_scope_v2": map[string]any{"policy_mode": "mapping", "entries": []any{map[string]any{"display_model_id": "my-model", "target_model_id": "gpt-6-astra", "provider": "openai"}}},
		"manual_models":  []any{map[string]any{"model_id": "gpt-5.5-pro"}},
	}}
	models := BuildAvailableTestModels(ctx, account, registry)
	require.Len(t, models, 1)
	require.Equal(t, "my-model", models[0].ID)
	require.True(t, isRequestedModelSupportedByAccount(ctx, registry, account, "my-model"))
	require.False(t, isRequestedModelSupportedByAccount(ctx, registry, account, "gpt-6-astra"))
	require.False(t, isRequestedModelSupportedByAccount(ctx, registry, account, "gpt-5.5-pro"))
}
