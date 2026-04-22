package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

var _ AccountRepository = (*opsAvailabilityAccountRepoStub)(nil)

type opsAvailabilityAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (s *opsAvailabilityAccountRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, lifecycle string, privacyMode string) ([]Account, *pagination.PaginationResult, error) {
	filtered := make([]Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		if platform != "" && !strings.EqualFold(account.Platform, platform) {
			continue
		}
		filtered = append(filtered, account)
	}

	limit := params.Limit()
	offset := params.Offset()
	if offset >= len(filtered) {
		return []Account{}, &pagination.PaginationResult{Total: int64(len(filtered)), Page: params.Page, PageSize: limit}, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return append([]Account(nil), filtered[offset:end]...), &pagination.PaginationResult{
		Total:    int64(len(filtered)),
		Page:     params.Page,
		PageSize: limit,
	}, nil
}

func TestOpsServiceGetAccountAvailabilityStats_UsesDisplayRateLimitProjection(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	group := &Group{ID: 9, Name: "OpenAI", Platform: PlatformOpenAI}
	repo := &opsAvailabilityAccountRepoStub{
		accounts: []Account{
			{
				ID:          1,
				Name:        "non-pro-limited",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Extra: map[string]any{
					"codex_7d_used_percent": 100.0,
					"codex_7d_reset_at":     now.Add(3 * time.Hour).Format(time.RFC3339),
				},
				Groups: []*Group{group},
			},
			{
				ID:          2,
				Name:        "pro-partial",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					"codex_7d_used_percent": 100.0,
					"codex_7d_reset_at":     now.Add(4 * time.Hour).Format(time.RFC3339),
				},
				Groups: []*Group{group},
			},
			{
				ID:          3,
				Name:        "pro-double-limited",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					"codex_5h_used_percent":       100.0,
					"codex_5h_reset_at":           now.Add(2 * time.Hour).Format(time.RFC3339),
					"codex_spark_5h_used_percent": 100.0,
					"codex_spark_5h_reset_at":     now.Add(90 * time.Minute).Format(time.RFC3339),
				},
				Groups: []*Group{group},
			},
		},
	}
	svc := &OpsService{accountRepo: repo}

	platformStats, groupStats, accountStats, collectedAt, err := svc.GetAccountAvailabilityStats(context.Background(), "", nil)
	require.NoError(t, err)
	require.NotNil(t, collectedAt)

	openAIStats := platformStats[PlatformOpenAI]
	require.NotNil(t, openAIStats)
	require.EqualValues(t, 3, openAIStats.TotalAccounts)
	require.EqualValues(t, 1, openAIStats.AvailableCount)
	require.EqualValues(t, 2, openAIStats.RateLimitCount)

	groupAvailability := groupStats[group.ID]
	require.NotNil(t, groupAvailability)
	require.EqualValues(t, 3, groupAvailability.TotalAccounts)
	require.EqualValues(t, 1, groupAvailability.AvailableCount)
	require.EqualValues(t, 2, groupAvailability.RateLimitCount)

	require.True(t, accountStats[1].IsRateLimited)
	require.False(t, accountStats[1].IsAvailable)
	require.False(t, accountStats[2].IsRateLimited)
	require.True(t, accountStats[2].IsAvailable)
	require.True(t, accountStats[3].IsRateLimited)
	require.False(t, accountStats[3].IsAvailable)
	require.NotNil(t, accountStats[3].RateLimitResetAt)
}
