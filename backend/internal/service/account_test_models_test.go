package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAvailableTestModels_PrefersReplacementIDForDeprecatedRegistryAliases(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "family-stable",
		DisplayName: "Family Stable",
		Platforms:   []string{PlatformAnthropic},
		UIPriority:  1,
		ExposedIn:   []string{"runtime"},
	})
	require.NoError(t, err)

	_, err = registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "family-stable-20250929",
		DisplayName: "Family Stable",
		Platforms:   []string{PlatformAnthropic},
		UIPriority:  200,
		ExposedIn:   []string{"test"},
		Status:      "deprecated",
		ReplacedBy:  "family-stable",
	})
	require.NoError(t, err)

	_, err = registrySvc.ActivateModels(context.Background(), []string{"family-stable", "family-stable-20250929"})
	require.NoError(t, err)

	account := &Account{
		ID:       991,
		Name:     "kiro-test",
		Platform: PlatformKiro,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
	}

	models := BuildAvailableTestModels(context.Background(), account, registrySvc)
	require.NotEmpty(t, models)

	var target *AvailableTestModel
	for idx := range models {
		if models[idx].ID == "family-stable" {
			target = &models[idx]
			break
		}
	}

	require.NotNil(t, target)
	require.Equal(t, "family-stable", target.CanonicalID)
	require.Equal(t, "stable", target.Status)
	require.Empty(t, target.ReplacedBy)
	require.Empty(t, target.DeprecatedAt)

	for _, model := range models {
		require.NotEqual(t, "family-stable-20250929", model.ID)
	}
}

func TestBuildAvailableTestModels_FiltersCrossPlatformProviderForDirectOpenAIAccounts(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "grok-disguised-openai",
		DisplayName: "Grok Disguised OpenAI",
		Provider:    PlatformGrok,
		Platforms:   []string{PlatformOpenAI},
		UIPriority:  1,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "openai-native-test",
		DisplayName: "OpenAI Native Test",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		UIPriority:  2,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"grok-disguised-openai", "openai-native-test"})
	require.NoError(t, err)

	account := &Account{
		ID:       992,
		Name:     "openai-direct",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
	}

	models := BuildAvailableTestModels(context.Background(), account, registrySvc)
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}

	require.Contains(t, ids, "openai-native-test")
	require.NotContains(t, ids, "grok-disguised-openai")
}

func TestBuildAvailableTestModels_FiltersChatGPTOpenAIModelsByKnownSnapshot(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "gpt-5.4",
		DisplayName: "GPT-5.4",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		UIPriority:  1,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "gpt-5.1-codex-mini",
		DisplayName: "GPT-5.1 Codex Mini",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		UIPriority:  2,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"gpt-5.4", "gpt-5.1-codex-mini"})
	require.NoError(t, err)

	account := &Account{
		ID:       993,
		Name:     "openai-chatgpt-oauth",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Extra: map[string]any{
			"openai_known_models": []string{"gpt-5.4", "gpt-4.1-mini"},
		},
	}

	models := BuildAvailableTestModels(context.Background(), account, registrySvc)
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}

	require.Len(t, ids, 2)
	require.Contains(t, ids, "gpt-5.4")
	require.Contains(t, ids, "gpt-4.1-mini")
	require.NotContains(t, ids, "gpt-5.1-codex-mini")
}

func TestBuildAvailableTestModels_FiltersKnownUnsupportedChatGPTModelsWithoutSnapshot(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "gpt-5.4",
		DisplayName: "GPT-5.4",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		UIPriority:  1,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "gpt-5.1-codex-mini",
		DisplayName: "GPT-5.1 Codex Mini",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		UIPriority:  2,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"gpt-5.4", "gpt-5.1-codex-mini"})
	require.NoError(t, err)

	account := &Account{
		ID:       995,
		Name:     "openai-chatgpt-oauth-no-snapshot",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
	}

	models := BuildAvailableTestModels(context.Background(), account, registrySvc)
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}

	require.Contains(t, ids, "gpt-5.4")
	require.NotContains(t, ids, "gpt-5.1-codex-mini")
}

func TestBuildAvailableTestModels_DoesNotRestrictOpenAIAPIKeyModelsByKnownSnapshot(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "gpt-5.4",
		DisplayName: "GPT-5.4",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		UIPriority:  1,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "gpt-5.1-codex-mini",
		DisplayName: "GPT-5.1 Codex Mini",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		UIPriority:  2,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"gpt-5.4", "gpt-5.1-codex-mini"})
	require.NoError(t, err)

	account := &Account{
		ID:       994,
		Name:     "openai-apikey",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Extra: map[string]any{
			"openai_known_models": []string{"gpt-5.4"},
		},
	}

	models := BuildAvailableTestModels(context.Background(), account, registrySvc)
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}

	require.Contains(t, ids, "gpt-5.4")
	require.Contains(t, ids, "gpt-5.1-codex-mini")
}

func TestBuildManualTestModelCandidates_PrefersManualProviderMetadata(t *testing.T) {
	account := &Account{
		ID:       996,
		Name:     "openai-direct-manual-provider",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Extra: map[string]any{
			"manual_models": []any{
				map[string]any{
					"model_id": "shared-model",
					"provider": "grok",
				},
			},
		},
	}

	candidates := buildManualTestModelCandidates(account, "")
	require.Len(t, candidates, 1)
	require.Equal(t, "shared-model", candidates[0].model.ID)
	require.Equal(t, "grok", candidates[0].model.Provider)
	require.Equal(t, "xAI-Grok", candidates[0].model.ProviderLabel)
}

func TestBuildAvailableTestModels_AppliesExplicitMappingRestrictions(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())

	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "custom-shared-a",
		DisplayName: "Custom Shared A",
		Platforms:   []string{"custom-tests"},
		UIPriority:  1,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "custom-shared-b",
		DisplayName: "Custom Shared B",
		Platforms:   []string{"custom-tests"},
		UIPriority:  2,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"custom-shared-a", "custom-shared-b"})
	require.NoError(t, err)

	account := &Account{
		ID:       997,
		Name:     "mapping-account",
		Platform: "custom-tests",
		Type:     AccountTypeOAuth,
		Status:   StatusActive,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"friendly-a": "custom-shared-a",
			},
		},
	}

	models := BuildAvailableTestModels(context.Background(), account, registrySvc)
	require.Len(t, models, 1)
	require.Equal(t, "custom-shared-a", models[0].ID)
}

func TestBuildAvailableTestModels_UsesManualRowsAsExplicitAvailableModels(t *testing.T) {
	account := &Account{
		ID:       998,
		Name:     "manual-only-account",
		Platform: "custom-tests",
		Type:     AccountTypeAPIKey,
		Status:   StatusActive,
		Extra: map[string]any{
			"manual_models": []any{
				map[string]any{
					"model_id": "manual-allowed",
				},
				map[string]any{
					"model_id": "manual-blocked",
				},
			},
		},
	}

	models := BuildAvailableTestModels(context.Background(), account, nil)
	require.Len(t, models, 2)
	require.Equal(t, "manual-allowed", models[0].ID)
	require.Equal(t, "manual-blocked", models[1].ID)
}
