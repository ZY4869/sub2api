package service

import (
	"context"
	"testing"
	"time"
)

type accountLimitedReasonRepoStub struct {
	stubOpenAIAccountRepo
	setRateLimitedCalls []time.Time
	updateExtraCalls    []map[string]any
}

func (r *accountLimitedReasonRepoStub) SetRateLimited(_ context.Context, _ int64, resetAt time.Time) error {
	r.setRateLimitedCalls = append(r.setRateLimitedCalls, resetAt)
	return nil
}

func (r *accountLimitedReasonRepoStub) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	copied := make(map[string]any, len(updates))
	for key, value := range updates {
		copied[key] = value
	}
	r.updateExtraCalls = append(r.updateExtraCalls, copied)
	return nil
}

func TestAccountRateLimitReasonPrefersStoredReason(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	resetAt := now.Add(30 * time.Minute)
	account := &Account{
		RateLimitResetAt: &resetAt,
		Extra: map[string]any{
			"rate_limit_reason":          AccountRateLimitReasonUsage5h,
			"codex_7d_used_percent":      100.0,
			"session_window_utilization": 1.0,
		},
	}

	if got := AccountRateLimitReason(account, now); got != AccountRateLimitReasonUsage5h {
		t.Fatalf("AccountRateLimitReason() = %q, want %q", got, AccountRateLimitReasonUsage5h)
	}
}

func TestAccountRateLimitReasonInfersOpenAIFallback(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	resetAt := now.Add(30 * time.Minute)

	tests := []struct {
		name        string
		credentials map[string]any
		extra       map[string]any
		want        string
	}{
		{
			name: "all 7d exhausted wins over single scope",
			credentials: map[string]any{
				"plan_type": "pro",
			},
			extra: map[string]any{
				"codex_7d_used_percent":          100.0,
				"codex_7d_reset_at":              resetAt.Format(time.RFC3339),
				"codex_spark_7d_used_percent":    100.0,
				"codex_spark_7d_reset_at":        resetAt.Format(time.RFC3339),
				"codex_account_7d_all_exhausted": true,
			},
			want: AccountRateLimitReasonUsage7dAll,
		},
		{
			name: "7d wins over 5h",
			extra: map[string]any{
				"codex_7d_used_percent": 100.0,
				"codex_5h_used_percent": 100.0,
			},
			want: AccountRateLimitReasonUsage7d,
		},
		{
			name: "5h when only 5h exhausted",
			extra: map[string]any{
				"codex_5h_used_percent": 100.0,
			},
			want: AccountRateLimitReasonUsage5h,
		},
		{
			name:  "fallback to 429 when no usage markers",
			extra: map[string]any{},
			want:  AccountRateLimitReason429,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			account := &Account{
				RateLimitResetAt: &resetAt,
				Credentials:      tt.credentials,
				Extra:            tt.extra,
			}
			if got := AccountRateLimitReason(account, now); got != tt.want {
				t.Fatalf("AccountRateLimitReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAccountRateLimitReasonInfersAnthropicFallback(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	resetAt := now.Add(30 * time.Minute)
	sessionEnd := now.Add(5 * time.Hour)

	tests := []struct {
		name             string
		extra            map[string]any
		sessionWindowEnd *time.Time
		want             string
	}{
		{
			name: "7d passive usage bucket",
			extra: map[string]any{
				"passive_usage_7d_utilization": 1.0,
				"session_window_utilization":   1.0,
			},
			want: AccountRateLimitReasonUsage7d,
		},
		{
			name: "5h session bucket",
			extra: map[string]any{
				"session_window_utilization": 1.0,
			},
			sessionWindowEnd: &sessionEnd,
			want:             AccountRateLimitReasonUsage5h,
		},
		{
			name: "429 fallback without reliable windows",
			extra: map[string]any{
				"session_window_utilization": 1.0,
			},
			want: AccountRateLimitReason429,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			account := &Account{
				RateLimitResetAt: &resetAt,
				SessionWindowEnd: tt.sessionWindowEnd,
				Extra:            tt.extra,
			}
			if got := AccountRateLimitReason(account, now); got != tt.want {
				t.Fatalf("AccountRateLimitReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAccountLimitedFilterNormalization(t *testing.T) {
	t.Parallel()

	ctx := WithAccountLimitedFilters(context.Background(), "limited_only", "usage_7d_all")
	filters := AccountLimitedFiltersFromContext(ctx)
	if filters.LimitedView != AccountLimitedViewLimitedOnly {
		t.Fatalf("LimitedView = %q, want %q", filters.LimitedView, AccountLimitedViewLimitedOnly)
	}
	if filters.LimitedReason != AccountRateLimitReasonUsage7dAll {
		t.Fatalf("LimitedReason = %q, want %q", filters.LimitedReason, AccountRateLimitReasonUsage7dAll)
	}

	filters = AccountLimitedFiltersFromContext(WithAccountLimitedFilters(context.Background(), "???", "nope"))
	if filters.LimitedView != AccountLimitedViewAll {
		t.Fatalf("invalid LimitedView normalized to %q, want %q", filters.LimitedView, AccountLimitedViewAll)
	}
	if filters.LimitedReason != "" {
		t.Fatalf("invalid LimitedReason normalized to %q, want empty string", filters.LimitedReason)
	}
}

func TestSetAccountRateLimitedPersistsNormalizedReason(t *testing.T) {
	t.Parallel()

	repo := &accountLimitedReasonRepoStub{}
	resetAt := time.Date(2026, 3, 26, 13, 0, 0, 0, time.UTC)

	if err := setAccountRateLimited(context.Background(), repo, 42, resetAt, " usage_5h "); err != nil {
		t.Fatalf("setAccountRateLimited() error = %v", err)
	}
	if len(repo.setRateLimitedCalls) != 1 {
		t.Fatalf("SetRateLimited calls = %d, want 1", len(repo.setRateLimitedCalls))
	}
	if len(repo.updateExtraCalls) != 1 {
		t.Fatalf("UpdateExtra calls = %d, want 1", len(repo.updateExtraCalls))
	}
	if got := repo.updateExtraCalls[0]["rate_limit_reason"]; got != AccountRateLimitReasonUsage5h {
		t.Fatalf("rate_limit_reason = %v, want %q", got, AccountRateLimitReasonUsage5h)
	}
}
