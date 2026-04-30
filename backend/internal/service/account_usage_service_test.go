package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	openaipkg "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
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

type openAICodexProbeQueueDialer struct {
	conns     []openAIWSClientConn
	handshake http.Header
	dialCount int
}

func (d *openAICodexProbeQueueDialer) Dial(_ context.Context, _ string, _ http.Header, _ string) (openAIWSClientConn, int, http.Header, error) {
	d.dialCount++
	if len(d.conns) == 0 {
		return nil, 0, nil, errors.New("no ws conns available")
	}
	conn := d.conns[0]
	d.conns = d.conns[1:]
	return conn, 0, cloneHeader(d.handshake), nil
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

func TestAccountUsageService_GetOpenAIUsage_IgnoresSparkWindowsForNonPro(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	extra := map[string]any{
		"codex_5h_used_percent":       12.0,
		"codex_5h_reset_at":           now.Add(2 * time.Hour).Format(time.RFC3339),
		"codex_7d_used_percent":       34.0,
		"codex_7d_reset_at":           now.Add(24 * time.Hour).Format(time.RFC3339),
		codexSpark5hUsedPercentKey:    56.0,
		codexSpark5hResetAtKey:        now.Add(3 * time.Hour).Format(time.RFC3339),
		codexSpark7dUsedPercentKey:    78.0,
		codexSpark7dResetAtKey:        now.Add(48 * time.Hour).Format(time.RFC3339),
		"codex_usage_updated_at":      now.Format(time.RFC3339),
		"rate_limit_reason":           AccountRateLimitReasonUsage5h,
		codexAccountAll7dExhaustedKey: false,
	}

	svc := &AccountUsageService{}
	plusUsage, err := svc.getOpenAIUsage(context.Background(), &Account{
		ID:          4001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Schedulable: true,
		Status:      StatusActive,
		Credentials: map[string]any{
			"plan_type": "plus",
		},
		Extra: cloneStringAnyMap(extra),
	}, false)
	if err != nil {
		t.Fatalf("getOpenAIUsage() error = %v", err)
	}
	if plusUsage.FiveHour == nil || plusUsage.SevenDay == nil {
		t.Fatal("expected normal openai usage windows for non-pro account")
	}
	if plusUsage.SparkFiveHour != nil || plusUsage.SparkSevenDay != nil {
		t.Fatalf("expected non-pro account to ignore spark windows, got %+v", plusUsage)
	}

	proUsage, err := svc.getOpenAIUsage(context.Background(), &Account{
		ID:          4002,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Schedulable: true,
		Status:      StatusActive,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: cloneStringAnyMap(extra),
	}, false)
	if err != nil {
		t.Fatalf("getOpenAIUsage() error = %v", err)
	}
	if proUsage.SparkFiveHour == nil || proUsage.SparkSevenDay == nil {
		t.Fatalf("expected pro account to retain spark windows, got %+v", proUsage)
	}
}

func TestAccountUsageService_GetOpenAIUsage_NonProPlansProbeOnlyNormalScope(t *testing.T) {
	t.Parallel()

	plans := []string{"", "free", "plus", "team", "mystery"}
	for _, plan := range plans {
		plan := plan
		t.Run("plan="+plan, func(t *testing.T) {
			t.Parallel()

			now := time.Now().UTC().Truncate(time.Second)
			probedModels := make([]string, 0, 1)
			svc := &AccountUsageService{
				openAICodexScopeProbe: func(_ context.Context, _ *Account, modelID string) (map[string]any, *time.Time, error) {
					probedModels = append(probedModels, modelID)
					if modelID == openAICodexScopeSpark {
						return nil, nil, fmt.Errorf("non-pro plan should not probe spark scope")
					}
					resetAt := now.Add(24 * time.Hour)
					return map[string]any{
						"codex_usage_updated_at": now.Format(time.RFC3339),
						"codex_5h_used_percent":  12.0,
						"codex_5h_reset_at":      now.Add(2 * time.Hour).Format(time.RFC3339),
						"codex_7d_used_percent":  34.0,
						"codex_7d_reset_at":      resetAt.Format(time.RFC3339),
					}, &resetAt, nil
				},
			}
			credentials := map[string]any{
				"access_token": "token",
			}
			if plan != "" {
				credentials["plan_type"] = plan
			}

			usage, err := svc.getOpenAIUsage(context.Background(), &Account{
				ID:          5009,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Schedulable: true,
				Status:      StatusActive,
				Credentials: credentials,
				Extra:       map[string]any{},
			}, true)
			if err != nil {
				t.Fatalf("getOpenAIUsage() error = %v", err)
			}
			if len(probedModels) != 1 || probedModels[0] != openAICodexScopeNormal {
				t.Fatalf("probe models = %v, want [%q]", probedModels, openAICodexScopeNormal)
			}
			if usage.SparkFiveHour != nil || usage.SparkSevenDay != nil {
				t.Fatalf("expected non-pro plan %q to suppress spark windows, got %+v", plan, usage)
			}
		})
	}
}

func TestShouldRefreshOpenAICodexSnapshot_ProRefreshesWhenSparkWindowsMissing(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	account := &Account{
		ID:       5010,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{
			"codex_usage_updated_at": now.Format(time.RFC3339),
		},
	}
	usage := &UsageInfo{
		FiveHour: &UsageProgress{
			Utilization: 12,
		},
		SevenDay: &UsageProgress{
			Utilization: 34,
		},
	}

	if !shouldRefreshOpenAICodexSnapshot(account, usage, now) {
		t.Fatal("expected pro account to refresh when spark usage windows are missing")
	}

	usage.SparkFiveHour = &UsageProgress{Utilization: 56}
	usage.SparkSevenDay = &UsageProgress{Utilization: 78}
	if !shouldRefreshOpenAICodexSnapshot(account, usage, now) {
		t.Fatal("expected pro account to refresh when spark snapshot timestamp is missing")
	}

	account.Extra[codexSparkUsageUpdatedAtKey] = now.Add(-openAIProbeCacheTTL).Format(time.RFC3339)
	if !shouldRefreshOpenAICodexSnapshot(account, usage, now) {
		t.Fatal("expected pro account to refresh when spark snapshot timestamp is stale")
	}

	account.Extra[codexSparkUsageUpdatedAtKey] = now.Format(time.RFC3339)
	if shouldRefreshOpenAICodexSnapshot(account, usage, now) {
		t.Fatal("did not expect refresh when normal and spark windows are present and fresh")
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

func TestAccountUsageService_GetOpenAIUsage_ForceRefreshProbesNormalAndSpark(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	probedModels := make([]string, 0, 2)
	svc := &AccountUsageService{
		openAICodexScopeProbe: func(_ context.Context, _ *Account, modelID string) (map[string]any, *time.Time, error) {
			probedModels = append(probedModels, modelID)
			switch modelID {
			case openAICodexScopeNormal:
				resetAt := now.Add(24 * time.Hour)
				return map[string]any{
					"codex_usage_updated_at": now.Format(time.RFC3339),
					"codex_5h_used_percent":  12.0,
					"codex_5h_reset_at":      now.Add(2 * time.Hour).Format(time.RFC3339),
					"codex_7d_used_percent":  34.0,
					"codex_7d_reset_at":      resetAt.Format(time.RFC3339),
				}, &resetAt, nil
			case openAICodexScopeSpark:
				resetAt := now.Add(48 * time.Hour)
				return map[string]any{
					"codex_usage_updated_at":   now.Format(time.RFC3339),
					codexSpark5hUsedPercentKey: 56.0,
					codexSpark5hResetAtKey:     now.Add(3 * time.Hour).Format(time.RFC3339),
					codexSpark7dUsedPercentKey: 78.0,
					codexSpark7dResetAtKey:     resetAt.Format(time.RFC3339),
				}, &resetAt, nil
			default:
				return nil, nil, fmt.Errorf("unexpected model %q", modelID)
			}
		},
	}

	usage, err := svc.getOpenAIUsage(context.Background(), &Account{
		ID:       5001,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "token",
			"plan_type":    "pro",
		},
		Extra: map[string]any{},
	}, true)
	if err != nil {
		t.Fatalf("getOpenAIUsage() error = %v", err)
	}
	if len(probedModels) != 2 {
		t.Fatalf("expected 2 probe models, got %v", probedModels)
	}
	if probedModels[0] != openAICodexScopeNormal || probedModels[1] != openAICodexScopeSpark {
		t.Fatalf("probe order = %v, want [%q %q]", probedModels, openAICodexScopeNormal, openAICodexScopeSpark)
	}
	if usage.FiveHour == nil || usage.SevenDay == nil {
		t.Fatalf("expected normal openai windows, got %+v", usage)
	}
	if usage.SparkFiveHour == nil || usage.SparkSevenDay == nil {
		t.Fatalf("expected spark openai windows, got %+v", usage)
	}
	if usage.FiveHour.Utilization != 12.0 || usage.SevenDay.Utilization != 34.0 {
		t.Fatalf("unexpected normal usage windows: %+v", usage)
	}
	if usage.SparkFiveHour.Utilization != 56.0 || usage.SparkSevenDay.Utilization != 78.0 {
		t.Fatalf("unexpected spark usage windows: %+v", usage)
	}
}

func TestResolveOpenAICodexProbeModelID_UsesSparkMapping(t *testing.T) {
	t.Parallel()

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				openAICodexScopeSpark: "gpt-5.3-codex-spark-high",
			},
		},
	}

	if got := resolveOpenAICodexProbeModelID(account, openAICodexScopeSpark); got != "gpt-5.3-codex-spark-high" {
		t.Fatalf("resolveOpenAICodexProbeModelID(spark) = %q, want %q", got, "gpt-5.3-codex-spark-high")
	}
	if got := resolveOpenAICodexProbeModelID(account, openaipkg.DefaultTestModel); got != openaipkg.DefaultTestModel {
		t.Fatalf("resolveOpenAICodexProbeModelID(normal) = %q, want %q", got, openaipkg.DefaultTestModel)
	}
}

func TestResolveOpenAICodexProbeModelID_IgnoresSparkMappingToNormalScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mapping map[string]any
	}{
		{
			name: "exact spark mapping to normal model",
			mapping: map[string]any{
				openAICodexScopeSpark: openaipkg.DefaultTestModel,
			},
		},
		{
			name: "wildcard mapping to normal model",
			mapping: map[string]any{
				"*": openaipkg.DefaultTestModel,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			account := &Account{
				ID:       5020,
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"model_mapping": tt.mapping,
				},
			}

			if got := resolveOpenAICodexProbeModelID(account, openAICodexScopeSpark); got != openAICodexScopeSpark {
				t.Fatalf("resolveOpenAICodexProbeModelID(spark) = %q, want %q", got, openAICodexScopeSpark)
			}
		})
	}
}

func TestOpenAICodexProbeHeaders_MatchOfficialCodexClient(t *testing.T) {
	t.Parallel()

	if !openaipkg.IsCodexOfficialClientByHeaders(codexCLIUserAgent, "codex_cli_rs") {
		t.Fatalf("expected probe UA/originator to be recognized as official codex client (ua=%q)", codexCLIUserAgent)
	}
}

func TestAccountUsageService_ProbeOpenAICodexSnapshot_ProKeepsPartialSuccess(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	probedModels := make([]string, 0, 2)
	svc := &AccountUsageService{
		openAICodexScopeProbe: func(_ context.Context, _ *Account, modelID string) (map[string]any, *time.Time, error) {
			probedModels = append(probedModels, modelID)
			if modelID == openAICodexScopeSpark {
				return nil, nil, fmt.Errorf("spark probe failed")
			}
			resetAt := now.Add(24 * time.Hour)
			return map[string]any{
				"codex_usage_updated_at": now.Format(time.RFC3339),
				"codex_5h_used_percent":  20.0,
				"codex_5h_reset_at":      now.Add(2 * time.Hour).Format(time.RFC3339),
				"codex_7d_used_percent":  40.0,
				"codex_7d_reset_at":      resetAt.Format(time.RFC3339),
			}, &resetAt, nil
		},
	}

	updates, resetAt, err := svc.probeOpenAICodexSnapshot(context.Background(), &Account{
		ID:       5002,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "token",
			"plan_type":    "pro",
		},
	})
	if err != nil {
		t.Fatalf("probeOpenAICodexSnapshot() error = %v", err)
	}
	if len(probedModels) != 2 {
		t.Fatalf("expected both normal and spark probes, got %v", probedModels)
	}
	if updates["codex_5h_used_percent"] != 20.0 {
		t.Fatalf("codex_5h_used_percent = %v, want 20", updates["codex_5h_used_percent"])
	}
	if _, ok := updates[codexSpark5hUsedPercentKey]; ok {
		t.Fatalf("did not expect spark updates on failed spark probe: %+v", updates)
	}
	if resetAt == nil {
		t.Fatal("expected resetAt from successful normal probe")
	}
}

func TestAccountUsageService_ProbeOpenAICodexSnapshot_Pro_WSEventDistinguishesScopes(t *testing.T) {
	t.Parallel()

	normalConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"codex.rate_limits","rate_limits":{"primary":{"used_percent":4.0,"window_minutes":300,"resets_in_seconds":1800},"secondary":{"used_percent":40.0,"window_minutes":10080,"resets_in_seconds":604800}}}`),
			[]byte(`{"type":"response.completed","response":{"id":"resp_normal_1","model":"gpt-5.3-codex","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}
	sparkConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"codex.rate_limits","rate_limits":{"primary":{"used_percent":7.0,"window_minutes":300,"resets_in_seconds":1200},"secondary":{"used_percent":55.0,"window_minutes":10080,"resets_in_seconds":500000}}}`),
			[]byte(`{"type":"response.completed","response":{"id":"resp_spark_1","model":"gpt-5.3-codex-spark","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}
	dialer := &openAICodexProbeQueueDialer{conns: []openAIWSClientConn{normalConn, sparkConn}}
	svc := &AccountUsageService{openAICodexWSProbeDialer: dialer}

	updates, _, err := svc.probeOpenAICodexSnapshot(context.Background(), &Account{
		ID:       9101,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "token",
			"plan_type":    "pro",
		},
	})
	if err != nil {
		t.Fatalf("probeOpenAICodexSnapshot() error = %v", err)
	}
	if got := parseExtraFloat64(updates["codex_5h_used_percent"]); got != 4.0 {
		t.Fatalf("codex_5h_used_percent = %v, want 4", got)
	}
	if got := parseExtraFloat64(updates[codexSpark5hUsedPercentKey]); got != 7.0 {
		t.Fatalf("codex_spark_5h_used_percent = %v, want 7", got)
	}
	if got := parseExtraFloat64(updates["codex_7d_used_percent"]); got != 40.0 {
		t.Fatalf("codex_7d_used_percent = %v, want 40", got)
	}
	if got := parseExtraFloat64(updates[codexSpark7dUsedPercentKey]); got != 55.0 {
		t.Fatalf("codex_spark_7d_used_percent = %v, want 55", got)
	}
	if len(normalConn.writes) == 0 {
		t.Fatal("expected ws probe to write payload for normal scope")
	}
	if got, _ := normalConn.writes[0]["model"].(string); got != openAICodexScopeNormal {
		t.Fatalf("normal ws probe model = %q, want %q", got, openAICodexScopeNormal)
	}
	if len(sparkConn.writes) == 0 {
		t.Fatal("expected ws probe to write payload for spark scope")
	}
	if got, _ := sparkConn.writes[0]["model"].(string); got != openAICodexScopeSpark {
		t.Fatalf("spark ws probe model = %q, want %q", got, openAICodexScopeSpark)
	}
}

func TestAccountUsageService_ProbeOpenAICodexSnapshotForModel_WSFailFallsBackToHTTPProbe(t *testing.T) {
	t.Parallel()

	dialer := &openAICodexProbeQueueDialer{}
	httpCalls := 0
	svc := &AccountUsageService{
		openAICodexWSProbeDialer: dialer,
		openAICodexScopeProbeHTTP: func(_ context.Context, _ *Account, modelID string) (map[string]any, *time.Time, error) {
			httpCalls++
			if modelID != openAICodexScopeNormal {
				return nil, nil, fmt.Errorf("unexpected modelID %q", modelID)
			}
			now := time.Now().UTC().Truncate(time.Second)
			resetAt := now.Add(24 * time.Hour)
			return map[string]any{
				"codex_usage_updated_at": now.Format(time.RFC3339),
				"codex_5h_used_percent":  12.0,
				"codex_5h_reset_at":      now.Add(2 * time.Hour).Format(time.RFC3339),
				"codex_7d_used_percent":  34.0,
				"codex_7d_reset_at":      resetAt.Format(time.RFC3339),
			}, &resetAt, nil
		},
	}

	updates, _, err := svc.probeOpenAICodexSnapshotForModel(context.Background(), &Account{
		ID:       9102,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "token",
			"plan_type":    "pro",
		},
	}, openAICodexScopeNormal)
	if err != nil {
		t.Fatalf("probeOpenAICodexSnapshotForModel() error = %v", err)
	}
	if httpCalls != 1 {
		t.Fatalf("http fallback calls = %d, want 1", httpCalls)
	}
	if dialer.dialCount != 1 {
		t.Fatalf("ws dial count = %d, want 1", dialer.dialCount)
	}
	if got := parseExtraFloat64(updates["codex_5h_used_percent"]); got != 12.0 {
		t.Fatalf("codex_5h_used_percent = %v, want 12", got)
	}
}

func TestMaybeWarnOpenAICodexProbeDegenerate_EmitsWarn(t *testing.T) {
	t.Parallel()

	logSink, restore := captureStructuredLog(t)
	defer restore()

	account := &Account{
		ID:       9103,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
	}
	updates := map[string]any{
		"codex_5h_used_percent":        4.0,
		"codex_7d_used_percent":        4.0,
		codexSpark5hUsedPercentKey:     4.0,
		codexSpark7dUsedPercentKey:     4.0,
		"codex_5h_reset_at":            "2026-04-30T00:00:00Z",
		"codex_7d_reset_at":            "2026-05-07T00:00:00Z",
		codexSpark5hResetAtKey:         "2026-04-30T00:00:00Z",
		codexSpark7dResetAtKey:         "2026-05-07T00:00:00Z",
		"codex_usage_updated_at":       "2026-04-30T00:00:00Z",
		codexSparkUsageUpdatedAtKey:    "2026-04-30T00:00:00Z",
		"codex_5h_reset_after_seconds": 1,
		"codex_7d_reset_after_seconds": 1,
	}

	maybeWarnOpenAICodexProbeDegenerate(account, updates)
	if !logSink.ContainsMessageAtLevel("openai_codex_probe_degenerate", "warn") {
		t.Fatal("expected degenerate warning log")
	}
}
