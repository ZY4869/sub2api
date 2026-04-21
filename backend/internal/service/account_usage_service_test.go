package service

import (
	"context"
	"net/http"
	"testing"
	"time"
)

type accountUsageCodexProbeRepo struct {
	stubOpenAIAccountRepo
	updateExtraCh chan map[string]any
	rateLimitCh   chan time.Time
	modelLimitCh  chan struct {
		scope   string
		resetAt time.Time
	}
}

func (r *accountUsageCodexProbeRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	if r.updateExtraCh != nil {
		copied := make(map[string]any, len(updates))
		for k, v := range updates {
			copied[k] = v
		}
		r.updateExtraCh <- copied
	}
	return nil
}

func (r *accountUsageCodexProbeRepo) SetRateLimited(_ context.Context, _ int64, resetAt time.Time) error {
	if r.rateLimitCh != nil {
		r.rateLimitCh <- resetAt
	}
	return nil
}

func (r *accountUsageCodexProbeRepo) SetModelRateLimit(_ context.Context, _ int64, scope string, resetAt time.Time) error {
	if r.modelLimitCh != nil {
		r.modelLimitCh <- struct {
			scope   string
			resetAt time.Time
		}{scope: scope, resetAt: resetAt}
	}
	return nil
}

func TestShouldRefreshOpenAICodexSnapshot(t *testing.T) {
	t.Parallel()

	rateLimitedUntil := time.Now().Add(5 * time.Minute)
	now := time.Now()
	usage := &UsageInfo{
		FiveHour: &UsageProgress{Utilization: 0},
		SevenDay: &UsageProgress{Utilization: 0},
	}

	if !shouldRefreshOpenAICodexSnapshot(&Account{RateLimitResetAt: &rateLimitedUntil}, usage, now) {
		t.Fatal("expected rate-limited account to force codex snapshot refresh")
	}

	if shouldRefreshOpenAICodexSnapshot(&Account{}, usage, now) {
		t.Fatal("expected complete non-rate-limited usage to skip codex snapshot refresh")
	}

	if !shouldRefreshOpenAICodexSnapshot(&Account{}, &UsageInfo{FiveHour: nil, SevenDay: &UsageProgress{}}, now) {
		t.Fatal("expected missing 5h snapshot to require refresh")
	}

	staleAt := now.Add(-(openAIProbeCacheTTL + time.Minute)).Format(time.RFC3339)
	if !shouldRefreshOpenAICodexSnapshot(&Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"openai_oauth_responses_websockets_v2_enabled": true,
			"codex_usage_updated_at":                       staleAt,
		},
	}, usage, now) {
		t.Fatal("expected stale ws snapshot to trigger refresh")
	}
}

func TestExtractOpenAICodexProbeUpdatesAccepts429WithCodexHeaders(t *testing.T) {
	t.Parallel()

	headers := make(http.Header)
	headers.Set("x-codex-primary-used-percent", "100")
	headers.Set("x-codex-primary-reset-after-seconds", "604800")
	headers.Set("x-codex-primary-window-minutes", "10080")
	headers.Set("x-codex-secondary-used-percent", "100")
	headers.Set("x-codex-secondary-reset-after-seconds", "18000")
	headers.Set("x-codex-secondary-window-minutes", "300")

	updates, err := extractOpenAICodexProbeUpdates(&http.Response{StatusCode: http.StatusTooManyRequests, Header: headers})
	if err != nil {
		t.Fatalf("extractOpenAICodexProbeUpdates() error = %v", err)
	}
	if len(updates) == 0 {
		t.Fatal("expected codex probe updates from 429 headers")
	}
	if got := updates["codex_5h_used_percent"]; got != 100.0 {
		t.Fatalf("codex_5h_used_percent = %v, want 100", got)
	}
	if got := updates["codex_7d_used_percent"]; got != 100.0 {
		t.Fatalf("codex_7d_used_percent = %v, want 100", got)
	}
}

func TestExtractOpenAICodexProbeSnapshotAccepts429WithResetAt(t *testing.T) {
	t.Parallel()

	headers := make(http.Header)
	headers.Set("x-codex-primary-used-percent", "100")
	headers.Set("x-codex-primary-reset-after-seconds", "604800")
	headers.Set("x-codex-primary-window-minutes", "10080")
	headers.Set("x-codex-secondary-used-percent", "100")
	headers.Set("x-codex-secondary-reset-after-seconds", "18000")
	headers.Set("x-codex-secondary-window-minutes", "300")

	updates, resetAt, reason, err := extractOpenAICodexProbeSnapshot(&http.Response{StatusCode: http.StatusTooManyRequests, Header: headers})
	if err != nil {
		t.Fatalf("extractOpenAICodexProbeSnapshot() error = %v", err)
	}
	if len(updates) == 0 {
		t.Fatal("expected codex probe updates from 429 headers")
	}
	if resetAt == nil {
		t.Fatal("expected resetAt from exhausted codex headers")
	}
	if reason != AccountRateLimitReasonUsage7d {
		t.Fatalf("reason = %q, want %q", reason, AccountRateLimitReasonUsage7d)
	}
}

func TestExtractOpenAICodexProbeSnapshotForScope_SparkWritesSparkFields(t *testing.T) {
	t.Parallel()

	headers := make(http.Header)
	headers.Set("x-codex-primary-used-percent", "100")
	headers.Set("x-codex-primary-reset-after-seconds", "604800")
	headers.Set("x-codex-primary-window-minutes", "10080")
	headers.Set("x-codex-secondary-used-percent", "35")
	headers.Set("x-codex-secondary-reset-after-seconds", "18000")
	headers.Set("x-codex-secondary-window-minutes", "300")

	updates, resetAt, reason, err := extractOpenAICodexProbeSnapshotForScope(
		&http.Response{StatusCode: http.StatusTooManyRequests, Header: headers},
		openAICodexScopeSpark,
	)
	if err != nil {
		t.Fatalf("extractOpenAICodexProbeSnapshotForScope() error = %v", err)
	}
	if len(updates) == 0 {
		t.Fatal("expected spark codex probe updates from 429 headers")
	}
	if resetAt == nil {
		t.Fatal("expected resetAt from exhausted spark codex headers")
	}
	if reason != AccountRateLimitReasonUsage7d {
		t.Fatalf("reason = %q, want %q", reason, AccountRateLimitReasonUsage7d)
	}
	if got := updates[codexSpark5hUsedPercentKey]; got != 35.0 {
		t.Fatalf("codex_spark_5h_used_percent = %v, want 35", got)
	}
	if got := updates[codexSpark7dUsedPercentKey]; got != 100.0 {
		t.Fatalf("codex_spark_7d_used_percent = %v, want 100", got)
	}
	if _, ok := updates["codex_5h_used_percent"]; ok {
		t.Fatal("expected spark snapshot to avoid normal codex 5h field")
	}
	if _, ok := updates["codex_7d_used_percent"]; ok {
		t.Fatal("expected spark snapshot to avoid normal codex 7d field")
	}
}

func TestAccountUsageService_PersistOpenAICodexProbeSnapshotSetsScopedRateLimit(t *testing.T) {
	t.Parallel()

	repo := &accountUsageCodexProbeRepo{
		updateExtraCh: make(chan map[string]any, 2),
		rateLimitCh:   make(chan time.Time, 1),
		modelLimitCh: make(chan struct {
			scope   string
			resetAt time.Time
		}, 1),
	}
	svc := &AccountUsageService{accountRepo: repo}
	resetAt := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)
	account := &Account{
		ID:       321,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra:    map[string]any{},
	}

	state := svc.persistOpenAICodexProbeSnapshot(context.Background(), account, map[string]any{
		"codex_7d_used_percent": 100.0,
		"codex_7d_reset_at":     resetAt.Format(time.RFC3339),
	})
	if state == nil || state.ScopeResetAt == nil {
		t.Fatal("expected scoped codex state to be returned")
	}

	select {
	case updates := <-repo.updateExtraCh:
		if got := updates["codex_7d_used_percent"]; got != 100.0 {
			t.Fatalf("codex_7d_used_percent = %v, want 100", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiting for codex probe extra persistence timed out")
	}

	select {
	case got := <-repo.modelLimitCh:
		if got.scope != openAICodexScopeNormal {
			t.Fatalf("scope = %q, want %q", got.scope, openAICodexScopeNormal)
		}
		if got.resetAt.Before(resetAt.Add(-time.Second)) || got.resetAt.After(resetAt.Add(time.Second)) {
			t.Fatalf("model rate limit resetAt = %v, want around %v", got.resetAt, resetAt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiting for codex probe scoped rate-limit persistence timed out")
	}

	select {
	case got := <-repo.rateLimitCh:
		t.Fatalf("unexpected account rate-limit persistence: %v", got)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestBuildCodexUsageProgressFromExtra_ZerosExpiredWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)

	t.Run("expired 5h window zeroes utilization", func(t *testing.T) {
		extra := map[string]any{
			"codex_5h_used_percent": 42.0,
			"codex_5h_reset_at":     "2026-03-16T10:00:00Z", // 2h ago
		}
		progress := buildCodexUsageProgressFromExtra(extra, "5h", now)
		if progress == nil {
			t.Fatal("expected non-nil progress")
		}
		if progress.Utilization != 0 {
			t.Fatalf("expected Utilization=0 for expired window, got %v", progress.Utilization)
		}
		if progress.RemainingSeconds != 0 {
			t.Fatalf("expected RemainingSeconds=0, got %v", progress.RemainingSeconds)
		}
	})

	t.Run("active 5h window keeps utilization", func(t *testing.T) {
		resetAt := now.Add(2 * time.Hour).Format(time.RFC3339)
		extra := map[string]any{
			"codex_5h_used_percent": 42.0,
			"codex_5h_reset_at":     resetAt,
		}
		progress := buildCodexUsageProgressFromExtra(extra, "5h", now)
		if progress == nil {
			t.Fatal("expected non-nil progress")
		}
		if progress.Utilization != 42.0 {
			t.Fatalf("expected Utilization=42, got %v", progress.Utilization)
		}
	})

	t.Run("expired 7d window zeroes utilization", func(t *testing.T) {
		extra := map[string]any{
			"codex_7d_used_percent": 88.0,
			"codex_7d_reset_at":     "2026-03-15T00:00:00Z", // yesterday
		}
		progress := buildCodexUsageProgressFromExtra(extra, "7d", now)
		if progress == nil {
			t.Fatal("expected non-nil progress")
		}
		if progress.Utilization != 0 {
			t.Fatalf("expected Utilization=0 for expired 7d window, got %v", progress.Utilization)
		}
	})
}
