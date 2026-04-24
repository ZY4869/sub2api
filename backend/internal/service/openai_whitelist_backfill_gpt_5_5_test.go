package service

import (
	"context"
	"sort"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type openAIGPT55WhitelistBackfillRepoStub struct {
	pages            [][]Account
	updateExtraCalls []struct {
		id      int64
		updates map[string]any
	}
}

func (s *openAIGPT55WhitelistBackfillRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, _, _, _, _ string, _ int64, _ string, _ string) ([]Account, *pagination.PaginationResult, error) {
	page := params.Page
	if page < 1 || page > len(s.pages) {
		return nil, &pagination.PaginationResult{Page: page, PageSize: params.PageSize, Pages: len(s.pages)}, nil
	}
	return append([]Account(nil), s.pages[page-1]...), &pagination.PaginationResult{Page: page, PageSize: params.PageSize, Pages: len(s.pages)}, nil
}

func (s *openAIGPT55WhitelistBackfillRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	s.updateExtraCalls = append(s.updateExtraCalls, struct {
		id      int64
		updates map[string]any
	}{id: id, updates: updates})
	return nil
}

func buildOpenAIWhitelistExtra(models ...string) map[string]any {
	entries := make([]AccountModelScopeEntry, 0, len(models))
	for _, model := range models {
		entries = append(entries, AccountModelScopeEntry{
			DisplayModelID: model,
			TargetModelID:  model,
			Provider:       PlatformOpenAI,
			SourceProtocol: PlatformOpenAI,
			VisibilityMode: AccountModelVisibilityModeDirect,
		})
	}
	scope := (&AccountModelScopeV2{
		PolicyMode: AccountModelPolicyModeWhitelist,
		Entries:    entries,
	}).ToMap()
	return map[string]any{
		"model_scope_v2": scope,
	}
}

func extractDisplayModelIDs(t *testing.T, raw any) []string {
	t.Helper()
	scopeMap, ok := raw.(map[string]any)
	require.True(t, ok)
	entriesAny, ok := scopeMap["entries"].([]map[string]any)
	require.True(t, ok)

	models := make([]string, 0, len(entriesAny))
	for _, entry := range entriesAny {
		models = append(models, stringValueFromAny(entry["display_model_id"]))
	}
	sort.Strings(models)
	return models
}

func TestBackfillOpenAIGPT55DefaultWhitelists_UpdatesLegacyWhitelistsForNonFreeAccounts(t *testing.T) {
	t.Parallel()

	repo := &openAIGPT55WhitelistBackfillRepoStub{
		pages: [][]Account{
			{
				{
					ID:       1,
					Platform: PlatformOpenAI,
					Type:     AccountTypeOAuth,
					Credentials: map[string]any{
						"plan_type": "plus",
					},
					Extra: buildOpenAIWhitelistExtra("gpt-5.2", "gpt-5.4", "gpt-5.4-mini"),
				},
				{
					ID:       2,
					Platform: PlatformOpenAI,
					Type:     AccountTypeOAuth,
					Credentials: map[string]any{
						"plan_type": "pro",
					},
					Extra: buildOpenAIWhitelistExtra("gpt-5.2", "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex-spark"),
				},
				{
					ID:       3,
					Platform: PlatformOpenAI,
					Type:     AccountTypeOAuth,
					Credentials: map[string]any{
						"plan_type": "free",
					},
					Extra: buildOpenAIWhitelistExtra("gpt-5.2", "gpt-5.4", "gpt-5.4-mini"),
				},
				{
					ID:       4,
					Platform: PlatformOpenAI,
					Type:     AccountTypeOAuth,
					Credentials: map[string]any{
						"plan_type": "plus",
					},
					Extra: buildOpenAIWhitelistExtra("gpt-5.4"),
				},
			},
		},
	}

	result, err := BackfillOpenAIGPT55DefaultWhitelists(context.Background(), repo, 50)
	require.NoError(t, err)
	require.Equal(t, 4, result.Scanned)
	require.Equal(t, 2, result.Updated)
	require.Len(t, repo.updateExtraCalls, 2)

	updatedAccount1 := repo.updateExtraCalls[0]
	require.Equal(t, int64(1), updatedAccount1.id)
	models1 := extractDisplayModelIDs(t, updatedAccount1.updates["model_scope_v2"])
	require.Equal(t, []string{"gpt-5.2", "gpt-5.4", "gpt-5.4-mini", "gpt-5.5"}, models1)

	updatedAccount2 := repo.updateExtraCalls[1]
	require.Equal(t, int64(2), updatedAccount2.id)
	models2 := extractDisplayModelIDs(t, updatedAccount2.updates["model_scope_v2"])
	require.Equal(t, []string{"gpt-5.2", "gpt-5.3-codex-spark", "gpt-5.4", "gpt-5.4-mini", "gpt-5.5"}, models2)
}

func TestBackfillOpenAIGPT55DefaultWhitelists_IsIdempotent(t *testing.T) {
	t.Parallel()

	repo := &openAIGPT55WhitelistBackfillRepoStub{
		pages: [][]Account{
			{
				{
					ID:       1,
					Platform: PlatformOpenAI,
					Type:     AccountTypeOAuth,
					Credentials: map[string]any{
						"plan_type": "plus",
					},
					Extra: buildOpenAIWhitelistExtra("gpt-5.2", "gpt-5.4", "gpt-5.4-mini", "gpt-5.5"),
				},
				{
					ID:       2,
					Platform: PlatformOpenAI,
					Type:     AccountTypeOAuth,
					Credentials: map[string]any{
						"plan_type": "pro",
					},
					Extra: buildOpenAIWhitelistExtra("gpt-5.2", "gpt-5.4", "gpt-5.4-mini", "gpt-5.5", "gpt-5.3-codex-spark"),
				},
			},
		},
	}

	result, err := BackfillOpenAIGPT55DefaultWhitelists(context.Background(), repo, 50)
	require.NoError(t, err)
	require.Equal(t, 2, result.Scanned)
	require.Equal(t, 0, result.Updated)
	require.Len(t, repo.updateExtraCalls, 0)
}
