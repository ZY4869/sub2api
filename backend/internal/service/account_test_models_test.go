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
