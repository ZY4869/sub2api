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
	require.Equal(t, AccountModelPolicyModeWhitelist, scope.PolicyMode)
	require.Equal(t, []AccountModelScopeEntry{
		{
			DisplayModelID: "claude-sonnet-4-5-20250929",
			TargetModelID:  "claude-sonnet-4-5-20250929",
			Provider:       "anthropic",
			VisibilityMode: AccountModelVisibilityModeDirect,
		},
		{
			DisplayModelID: "claude-sonnet-4.5",
			TargetModelID:  "claude-sonnet-4.5",
			Provider:       "anthropic",
			VisibilityMode: AccountModelVisibilityModeDirect,
		},
	}, scope.Entries)

	serialized := scope.ToMap()
	require.Equal(t, AccountModelPolicyModeWhitelist, serialized["policy_mode"])
	require.Equal(t, []map[string]any{
		{
			"display_model_id": "claude-sonnet-4-5-20250929",
			"target_model_id":  "claude-sonnet-4-5-20250929",
			"provider":         "anthropic",
			"visibility_mode":  AccountModelVisibilityModeDirect,
		},
		{
			"display_model_id": "claude-sonnet-4.5",
			"target_model_id":  "claude-sonnet-4.5",
			"provider":         "anthropic",
			"visibility_mode":  AccountModelVisibilityModeDirect,
		},
	}, serialized["entries"])
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
	require.Equal(t, AccountModelPolicyModeMapping, scope.PolicyMode)
	require.Equal(t, []AccountModelScopeEntry{
		{
			DisplayModelID: "claude-sonnet-4-5-20250929",
			TargetModelID:  "claude-sonnet-4.5",
			Provider:       "anthropic",
			VisibilityMode: AccountModelVisibilityModeAlias,
		},
	}, scope.Entries)
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
	require.Equal(t, "claude-sonnet-4.5", mapping["claude-sonnet-4-5-20250929"])
	require.Len(t, mapping, 1)
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
