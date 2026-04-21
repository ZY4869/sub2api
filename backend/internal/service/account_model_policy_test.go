package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAccountModelProjection_LegacyScopePrefersMappedAliasForWhitelistedTarget(t *testing.T) {
	t.Parallel()

	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	for _, entry := range []UpsertModelRegistryEntryInput{
		{
			ID:          "registry-openai-beta",
			DisplayName: "Registry OpenAI Beta",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime", "whitelist"},
			UIPriority:  1,
		},
		{
			ID:          "registry-openai-gamma",
			DisplayName: "Registry OpenAI Gamma",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime", "whitelist"},
			UIPriority:  2,
		},
	} {
		_, err := registrySvc.UpsertEntry(context.Background(), entry)
		require.NoError(t, err)
	}
	_, err := registrySvc.ActivateModels(context.Background(), []string{"registry-openai-beta", "registry-openai-gamma"})
	require.NoError(t, err)

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"friendly-beta": "registry-openai-beta",
			},
		},
		Extra: map[string]any{
			"model_scope_v2": map[string]any{
				"supported_models_by_provider": map[string]any{
					PlatformOpenAI: []any{"registry-openai-beta"},
				},
			},
		},
	}

	projection := BuildAccountModelProjection(context.Background(), account, registrySvc)
	require.NotNil(t, projection)
	require.True(t, projection.Explicit)
	require.Equal(t, accountModelProjectionSourceScope, projection.Source)
	require.Equal(t, AccountModelPolicyModeMapping, projection.PolicyMode)
	require.Len(t, projection.Entries, 1)
	require.Equal(t, "friendly-beta", projection.Entries[0].DisplayModelID)
	require.Equal(t, "registry-openai-beta", projection.Entries[0].TargetModelID)
	require.Equal(t, AccountModelVisibilityModeAlias, projection.Entries[0].VisibilityMode)
}

func TestBuildAccountModelProjection_CacheTracksAvailabilitySnapshotChanges(t *testing.T) {
	t.Parallel()

	resetAccountModelProjectionCache()
	t.Cleanup(resetAccountModelProjectionCache)

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"friendly-gpt": "gpt-4.1-mini",
			},
		},
		Extra: map[string]any{
			accountModelProbeSnapshotExtraKey: map[string]any{
				"entries": []any{
					map[string]any{
						"display_model_id":   "friendly-gpt",
						"target_model_id":    "gpt-4.1-mini",
						"availability_state": AccountModelAvailabilityUnknown,
						"stale_state":        AccountModelStaleStateUnverified,
						"updated_at":         "2026-04-21T10:00:00Z",
						"source":             AccountModelProbeSnapshotSourceModelScopePreview,
					},
				},
			},
		},
	}

	firstProjection := BuildAccountModelProjection(context.Background(), account, nil)
	require.NotNil(t, firstProjection)
	require.Len(t, firstProjection.Entries, 1)
	require.Equal(t, AccountModelAvailabilityUnknown, firstProjection.Entries[0].AvailabilityState)
	require.Equal(t, AccountModelStaleStateUnverified, firstProjection.Entries[0].StaleState)

	account.Extra[accountModelProbeSnapshotExtraKey] = map[string]any{
		"entries": []any{
			map[string]any{
				"display_model_id":   "friendly-gpt",
				"target_model_id":    "gpt-4.1-mini",
				"availability_state": AccountModelAvailabilityVerified,
				"stale_state":        AccountModelStaleStateFresh,
				"updated_at":         "2026-04-21T10:05:00Z",
				"source":             AccountModelProbeSnapshotSourceManualProbe,
			},
		},
		"updated_at": "2026-04-21T10:05:00Z",
		"source":     AccountModelProbeSnapshotSourceManualProbe,
	}

	secondProjection := BuildAccountModelProjection(context.Background(), account, nil)
	require.NotNil(t, secondProjection)
	require.Len(t, secondProjection.Entries, 1)
	require.Equal(t, AccountModelAvailabilityVerified, secondProjection.Entries[0].AvailabilityState)
	require.Equal(t, AccountModelStaleStateFresh, secondProjection.Entries[0].StaleState)
}
