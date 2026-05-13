package service

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAvailableTestModelsFromProbeSnapshot_UsesRegistryMetadata(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:           "snapshot-image-model",
		DisplayName:  "Snapshot Image Model",
		Provider:     PlatformOpenAI,
		Platforms:    []string{PlatformOpenAI},
		Modalities:   []string{"image"},
		Capabilities: []string{"image_generation"},
		UIPriority:   1,
		ExposedIn:    []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"snapshot-image-model"})
	require.NoError(t, err)

	models := AvailableTestModelsFromProbeSnapshot(
		context.Background(),
		&Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
		registrySvc,
		&AccountModelProbeSnapshot{Models: []string{"snapshot-image-model"}},
	)
	require.Len(t, models, 1)
	require.Equal(t, "snapshot-image-model", models[0].ID)
	require.Equal(t, "Snapshot Image Model", models[0].DisplayName)
	require.Equal(t, "image", models[0].Mode)
	require.Equal(t, PlatformOpenAI, models[0].Provider)
}

func TestBuildAccountModelSupportCacheKey_ChangesWithAccountScopeAndRegistryVersion(t *testing.T) {
	resetAccountModelSupportRuntimeCaches()
	repo := newAccountModelImportSettingRepoStub()
	registrySvc := NewModelRegistryService(repo)

	accountA := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"alias-a": "model-a",
			},
		},
	}
	accountB := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"alias-b": "model-b",
			},
		},
	}

	keyA := buildAccountModelSupportCacheKey(context.Background(), registrySvc, accountA)
	keyB := buildAccountModelSupportCacheKey(context.Background(), registrySvc, accountB)
	require.NotEmpty(t, keyA)
	require.NotEmpty(t, keyB)
	require.NotEqual(t, keyA, keyB)

	repo.values[SettingKeyModelRegistryEntries] = `[{"id":"model-a"}]`
	accountModelSupportRegistryVersion = sync.Map{}
	keyAAfterRegistryChange := buildAccountModelSupportCacheKey(context.Background(), registrySvc, accountA)
	require.NotEqual(t, keyA, keyAAfterRegistryChange)
}

func TestIsRequestedModelSupportedByAccount_ReusesCachedExplicitSupportSet(t *testing.T) {
	resetAccountModelSupportRuntimeCaches()
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "cached-shared-model",
		DisplayName: "Cached Shared Model",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		UIPriority:  1,
		ExposedIn:   []string{"test", "runtime"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"cached-shared-model"})
	require.NoError(t, err)

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"friendly-model": "cached-shared-model",
			},
		},
	}

	require.True(t, isRequestedModelSupportedByAccount(context.Background(), registrySvc, account, "friendly-model"))
	require.False(t, isRequestedModelSupportedByAccount(context.Background(), registrySvc, account, "cached-shared-model"))
	accountModelSupportCacheMu.RLock()
	firstCacheSize := len(accountModelSupportCache)
	accountModelSupportCacheMu.RUnlock()
	require.Equal(t, 1, firstCacheSize)

	require.True(t, isRequestedModelSupportedByAccount(context.Background(), registrySvc, account, "friendly-model"))
	require.False(t, isRequestedModelSupportedByAccount(context.Background(), registrySvc, account, "cached-shared-model"))
	accountModelSupportCacheMu.RLock()
	secondCacheSize := len(accountModelSupportCache)
	accountModelSupportCacheMu.RUnlock()
	require.Equal(t, 1, secondCacheSize)
}

func TestIsRequestedModelSupportedByAccount_HardRemovedModelAlwaysRejected(t *testing.T) {
	resetAccountModelSupportRuntimeCaches()
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	account := &Account{
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"model_scope_v2": map[string]any{
				"policy_mode": "whitelist",
				"entries": []map[string]any{
					{
						"display_model_id": "gemini-3-pro-preview",
						"target_model_id":  "gemini-3-pro-preview",
					},
				},
			},
		},
	}

	require.False(t, isRequestedModelSupportedByAccount(context.Background(), registrySvc, account, "gemini-3-pro-preview"))
}

func TestIsRequestedModelSupportedByAccount_Phase2HardRemovedModelsAlwaysRejected(t *testing.T) {
	resetAccountModelSupportRuntimeCaches()
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	tests := []struct {
		name     string
		platform string
		modelID  string
	}{
		{name: "anthropic old shell", platform: PlatformAnthropic, modelID: "claude-sonnet-4-5"},
		{name: "anthropic dated shell", platform: PlatformAnthropic, modelID: "claude-haiku-4-5-20251001"},
		{name: "deepseek deprecated shell", platform: PlatformDeepSeek, modelID: "deepseek-chat"},
		{name: "grok deprecated shell", platform: PlatformGrok, modelID: "grok-4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{
				Platform: tt.platform,
				Type:     AccountTypeAPIKey,
				Extra: map[string]any{
					"model_scope_v2": map[string]any{
						"policy_mode": "whitelist",
						"entries": []map[string]any{
							{
								"display_model_id": tt.modelID,
								"target_model_id":  tt.modelID,
							},
						},
					},
				},
			}

			require.False(t, isRequestedModelSupportedByAccount(context.Background(), registrySvc, account, tt.modelID))
		})
	}
}
