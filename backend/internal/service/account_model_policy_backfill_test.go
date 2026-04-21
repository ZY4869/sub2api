package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type accountModelPolicyBackfillRepoStub struct {
	pages            [][]Account
	updateExtraCalls []struct {
		id      int64
		updates map[string]any
	}
}

func (s *accountModelPolicyBackfillRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, _, _, _, _ string, _ int64, _ string, _ string) ([]Account, *pagination.PaginationResult, error) {
	page := params.Page
	if page < 1 || page > len(s.pages) {
		return nil, &pagination.PaginationResult{Page: page, PageSize: params.PageSize, Pages: len(s.pages)}, nil
	}
	return append([]Account(nil), s.pages[page-1]...), &pagination.PaginationResult{Page: page, PageSize: params.PageSize, Pages: len(s.pages)}, nil
}

func (s *accountModelPolicyBackfillRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	s.updateExtraCalls = append(s.updateExtraCalls, struct {
		id      int64
		updates map[string]any
	}{id: id, updates: updates})
	return nil
}

func TestBuildAccountModelPolicyBackfillUpdates_NormalizesLegacyScopeAndSnapshot(t *testing.T) {
	t.Parallel()

	account := &Account{
		ID:       1,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"friendly-gpt": "gpt-4.1-mini",
			},
		},
		Extra: map[string]any{
			"model_scope_v2": map[string]any{
				"supported_models_by_provider": map[string]any{
					PlatformOpenAI: []any{"gpt-4.1-mini"},
				},
			},
			accountModelProbeSnapshotExtraKey: map[string]any{
				"models":     []any{"gpt-4.1-mini"},
				"updated_at": "2026-04-21T10:00:00Z",
				"source":     AccountModelProbeSnapshotSourceManualProbe,
			},
		},
	}

	updates, scopeChanged, snapshotChanged := BuildAccountModelPolicyBackfillUpdates(
		context.Background(),
		account,
		nil,
		time.Date(2026, 4, 21, 11, 0, 0, 0, time.UTC),
	)
	require.True(t, scopeChanged)
	require.True(t, snapshotChanged)
	require.Contains(t, updates, "model_scope_v2")
	require.Contains(t, updates, accountModelProbeSnapshotExtraKey)

	scopeMap, ok := updates["model_scope_v2"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, AccountModelPolicyModeMapping, scopeMap["policy_mode"])

	snapshotMap, ok := updates[accountModelProbeSnapshotExtraKey].(map[string]any)
	require.True(t, ok)
	entries, ok := snapshotMap["entries"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, entries, 1)
	require.Equal(t, "friendly-gpt", entries[0]["display_model_id"])
	require.Equal(t, "gpt-4.1-mini", entries[0]["target_model_id"])
	require.Equal(t, AccountModelAvailabilityVerified, entries[0]["availability_state"])
}

func TestBuildAccountModelPolicyBackfillUpdates_WritesStructuredScopeForLegacyMappingOnly(t *testing.T) {
	t.Parallel()

	account := &Account{
		ID:       2,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"friendly-gpt": "gpt-4.1-mini",
			},
		},
		Extra: map[string]any{},
	}

	updates, scopeChanged, snapshotChanged := BuildAccountModelPolicyBackfillUpdates(
		context.Background(),
		account,
		nil,
		time.Date(2026, 4, 21, 11, 0, 0, 0, time.UTC),
	)
	require.True(t, scopeChanged)
	require.True(t, snapshotChanged)
	require.Contains(t, updates, "model_scope_v2")

	scopeMap, ok := updates["model_scope_v2"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, AccountModelPolicyModeMapping, scopeMap["policy_mode"])
}

func TestBackfillAccountModelPolicies_IsIdempotent(t *testing.T) {
	t.Parallel()

	repo := &accountModelPolicyBackfillRepoStub{
		pages: [][]Account{
			{
				{
					ID:       1,
					Platform: PlatformOpenAI,
					Type:     AccountTypeAPIKey,
					Credentials: map[string]any{
						"model_mapping": map[string]any{
							"friendly-gpt": "gpt-4.1-mini",
						},
					},
					Extra: map[string]any{
						"model_scope_v2": map[string]any{
							"supported_models_by_provider": map[string]any{
								PlatformOpenAI: []any{"gpt-4.1-mini"},
							},
						},
						accountModelProbeSnapshotExtraKey: map[string]any{
							"models":     []any{"gpt-4.1-mini"},
							"updated_at": "2026-04-21T10:00:00Z",
							"source":     AccountModelProbeSnapshotSourceManualProbe,
						},
					},
				},
			},
		},
	}

	result, err := BackfillAccountModelPolicies(context.Background(), repo, nil, 50)
	require.NoError(t, err)
	require.Equal(t, 1, result.Scanned)
	require.Equal(t, 1, result.Updated)
	require.Equal(t, 1, result.ScopeNormalized)
	require.Equal(t, 1, result.SnapshotRefreshed)
	require.Len(t, repo.updateExtraCalls, 1)

	repo.pages = [][]Account{
		{
			{
				ID:       1,
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"friendly-gpt": "gpt-4.1-mini",
					},
				},
				Extra: repo.updateExtraCalls[0].updates,
			},
		},
	}
	repo.updateExtraCalls = nil

	result, err = BackfillAccountModelPolicies(context.Background(), repo, nil, 50)
	require.NoError(t, err)
	require.Equal(t, 1, result.Scanned)
	require.Equal(t, 0, result.Updated)
	require.Equal(t, 0, result.ScopeNormalized)
	require.Equal(t, 0, result.SnapshotRefreshed)
	require.Len(t, repo.updateExtraCalls, 0)
}
