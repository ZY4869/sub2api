package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountModelScopeV2ExtractAndToMap_RoundTripsSelectedModelIDs(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"model_scope_v2": map[string]any{
			"supported_providers": []any{"anthropic"},
			"supported_models_by_provider": map[string]any{
				"anthropic": []any{"claude-sonnet-4.5"},
			},
			"selected_model_ids": []any{
				"claude-sonnet-4-5-20250929",
				"claude-sonnet-4.5",
			},
			"manual_mapping_rows": []any{},
			"manual_mappings":     map[string]any{},
		},
	}

	scope, ok := ExtractAccountModelScopeV2(input)
	require.True(t, ok)
	require.NotNil(t, scope)
	require.Equal(t, []string{"claude-sonnet-4-5-20250929", "claude-sonnet-4.5"}, scope.SelectedModelIDs)

	serialized := scope.ToMap()
	require.Equal(
		t,
		[]string{"claude-sonnet-4-5-20250929", "claude-sonnet-4.5"},
		serialized["selected_model_ids"],
	)
}

func TestModelRegistryServiceInferAccountModelScopeV2_PreservesIdentityWhitelistSelection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	seedAccountModelScopeAnthropicEntry(t, ctx, svc)

	scope := svc.InferAccountModelScopeV2(ctx, PlatformAnthropic, AccountTypeAPIKey, map[string]string{
		"claude-sonnet-4-5-20250929": "claude-sonnet-4-5-20250929",
	})

	require.NotNil(t, scope)
	require.Equal(t, []string{"anthropic"}, scope.SupportedProviders)
	require.Equal(
		t,
		map[string][]string{"anthropic": []string{"claude-sonnet-4.5"}},
		scope.SupportedModelsByProvider,
	)
	require.Equal(t, []string{"claude-sonnet-4-5-20250929"}, scope.SelectedModelIDs)
	require.Empty(t, scope.ManualMappingRows)
	require.Empty(t, scope.ManualMappings)
}

func TestModelRegistryServiceBuildModelMappingFromScopeV2_FallsBackToSelectedModelIDs(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	seedAccountModelScopeAnthropicEntry(t, ctx, svc)

	mapping, selectedModels, hasScope, err := svc.BuildModelMappingFromScopeV2(
		ctx,
		PlatformAnthropic,
		AccountTypeAPIKey,
		map[string]any{
			"model_scope_v2": map[string]any{
				"selected_model_ids": []any{"claude-sonnet-4-5-20250929"},
			},
		},
	)

	require.NoError(t, err)
	require.True(t, hasScope)
	require.Equal(t, []string{"claude-sonnet-4.5"}, selectedModels)
	require.Equal(t, "claude-sonnet-4.5", mapping["claude-sonnet-4.5"])
	require.Equal(t, "claude-sonnet-4.5", mapping["claude-sonnet-4-5-20250929"])
}

func seedAccountModelScopeAnthropicEntry(t *testing.T, ctx context.Context, svc *ModelRegistryService) {
	t.Helper()

	_, err := svc.UpsertEntry(ctx, UpsertModelRegistryEntryInput{
		ID:          "claude-sonnet-4.5",
		DisplayName: "Claude Sonnet 4.5",
		Provider:    PlatformAnthropic,
		Platforms:   []string{PlatformAnthropic},
		Aliases: []string{
			"claude-sonnet-4-5-20250929",
		},
		ProtocolIDs: []string{
			"claude-sonnet-4-5-20250929",
		},
		UIPriority: 1,
		ExposedIn:  []string{"runtime", "whitelist"},
	})
	require.NoError(t, err)

	_, err = svc.ActivateModels(ctx, []string{"claude-sonnet-4.5"})
	require.NoError(t, err)
}
