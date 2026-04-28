//go:build unit

package service

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestCalculateOpenAI429ResetTime_7dExhausted(t *testing.T) {
	svc := &RateLimitService{}

	// Simulate headers when 7d limit is exhausted (100% used)
	// Primary = 7d (10080 minutes), Secondary = 5h (300 minutes)
	headers := http.Header{}
	headers.Set("x-codex-primary-used-percent", "100")
	headers.Set("x-codex-primary-reset-after-seconds", "384607") // ~4.5 days
	headers.Set("x-codex-primary-window-minutes", "10080")       // 7 days
	headers.Set("x-codex-secondary-used-percent", "3")
	headers.Set("x-codex-secondary-reset-after-seconds", "17369") // ~4.8 hours
	headers.Set("x-codex-secondary-window-minutes", "300")        // 5 hours

	before := time.Now()
	resetAt := svc.calculateOpenAI429ResetTime(headers)
	after := time.Now()

	if resetAt == nil {
		t.Fatal("expected non-nil resetAt")
	}

	// Should be approximately 384607 seconds from now
	expectedDuration := 384607 * time.Second
	minExpected := before.Add(expectedDuration)
	maxExpected := after.Add(expectedDuration)

	if resetAt.Before(minExpected) || resetAt.After(maxExpected) {
		t.Errorf("resetAt %v not in expected range [%v, %v]", resetAt, minExpected, maxExpected)
	}
}

func TestCalculateOpenAI429ResetTime_5hExhausted(t *testing.T) {
	svc := &RateLimitService{}

	// Simulate headers when 5h limit is exhausted (100% used)
	headers := http.Header{}
	headers.Set("x-codex-primary-used-percent", "50")
	headers.Set("x-codex-primary-reset-after-seconds", "500000")
	headers.Set("x-codex-primary-window-minutes", "10080") // 7 days
	headers.Set("x-codex-secondary-used-percent", "100")
	headers.Set("x-codex-secondary-reset-after-seconds", "3600") // 1 hour
	headers.Set("x-codex-secondary-window-minutes", "300")       // 5 hours

	before := time.Now()
	resetAt := svc.calculateOpenAI429ResetTime(headers)
	after := time.Now()

	if resetAt == nil {
		t.Fatal("expected non-nil resetAt")
	}

	// Should be approximately 3600 seconds from now
	expectedDuration := 3600 * time.Second
	minExpected := before.Add(expectedDuration)
	maxExpected := after.Add(expectedDuration)

	if resetAt.Before(minExpected) || resetAt.After(maxExpected) {
		t.Errorf("resetAt %v not in expected range [%v, %v]", resetAt, minExpected, maxExpected)
	}
}

func TestCalculateOpenAI429ResetTime_NeitherExhausted_UsesMax(t *testing.T) {
	svc := &RateLimitService{}

	// Neither limit at 100%, should use the longer reset time
	headers := http.Header{}
	headers.Set("x-codex-primary-used-percent", "80")
	headers.Set("x-codex-primary-reset-after-seconds", "100000")
	headers.Set("x-codex-primary-window-minutes", "10080")
	headers.Set("x-codex-secondary-used-percent", "90")
	headers.Set("x-codex-secondary-reset-after-seconds", "5000")
	headers.Set("x-codex-secondary-window-minutes", "300")

	before := time.Now()
	resetAt := svc.calculateOpenAI429ResetTime(headers)
	after := time.Now()

	if resetAt == nil {
		t.Fatal("expected non-nil resetAt")
	}

	// Should use the max (100000 seconds from 7d window)
	expectedDuration := 100000 * time.Second
	minExpected := before.Add(expectedDuration)
	maxExpected := after.Add(expectedDuration)

	if resetAt.Before(minExpected) || resetAt.After(maxExpected) {
		t.Errorf("resetAt %v not in expected range [%v, %v]", resetAt, minExpected, maxExpected)
	}
}

func TestCalculateOpenAI429ResetTime_NoCodexHeaders(t *testing.T) {
	svc := &RateLimitService{}

	// No codex headers at all
	headers := http.Header{}
	headers.Set("content-type", "application/json")

	resetAt := svc.calculateOpenAI429ResetTime(headers)

	if resetAt != nil {
		t.Errorf("expected nil resetAt when no codex headers, got %v", resetAt)
	}
}

func TestCalculateOpenAI429ResetTime_ReversedWindowOrder(t *testing.T) {
	svc := &RateLimitService{}

	// Test when OpenAI sends primary as 5h and secondary as 7d (reversed)
	headers := http.Header{}
	headers.Set("x-codex-primary-used-percent", "100")         // This is 5h
	headers.Set("x-codex-primary-reset-after-seconds", "3600") // 1 hour
	headers.Set("x-codex-primary-window-minutes", "300")       // 5 hours - smaller!
	headers.Set("x-codex-secondary-used-percent", "50")
	headers.Set("x-codex-secondary-reset-after-seconds", "500000")
	headers.Set("x-codex-secondary-window-minutes", "10080") // 7 days - larger!

	before := time.Now()
	resetAt := svc.calculateOpenAI429ResetTime(headers)
	after := time.Now()

	if resetAt == nil {
		t.Fatal("expected non-nil resetAt")
	}

	// Should correctly identify that primary is 5h (smaller window) and use its reset time
	expectedDuration := 3600 * time.Second
	minExpected := before.Add(expectedDuration)
	maxExpected := after.Add(expectedDuration)

	if resetAt.Before(minExpected) || resetAt.After(maxExpected) {
		t.Errorf("resetAt %v not in expected range [%v, %v]", resetAt, minExpected, maxExpected)
	}
}

type openAI429SnapshotRepo struct {
	mockAccountRepoForGemini
	rateLimitedID       int64
	rateLimitedAt       *time.Time
	updatedExtra        map[string]any
	updateExtraCalls    []map[string]any
	modelScope          string
	modelRateLimitCalls []struct {
		scope   string
		resetAt time.Time
	}
	clearRateLimitCalls int
}

func (r *openAI429SnapshotRepo) SetRateLimited(_ context.Context, id int64, resetAt time.Time) error {
	r.rateLimitedID = id
	copyResetAt := resetAt
	r.rateLimitedAt = &copyResetAt
	return nil
}

func (r *openAI429SnapshotRepo) SetModelRateLimit(_ context.Context, _ int64, scope string, resetAt time.Time) error {
	r.modelScope = scope
	r.modelRateLimitCalls = append(r.modelRateLimitCalls, struct {
		scope   string
		resetAt time.Time
	}{scope: scope, resetAt: resetAt})
	return nil
}

func (r *openAI429SnapshotRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	copied := make(map[string]any, len(updates))
	if r.updatedExtra == nil {
		r.updatedExtra = map[string]any{}
	}
	for key, value := range updates {
		copied[key] = value
		r.updatedExtra[key] = value
	}
	r.updateExtraCalls = append(r.updateExtraCalls, copied)
	return nil
}

func (r *openAI429SnapshotRepo) ClearRateLimit(_ context.Context, _ int64) error {
	r.clearRateLimitCalls++
	return nil
}

func buildCodexHeaders(used7d float64, reset7dSeconds int, used5h float64, reset5hSeconds int) http.Header {
	headers := http.Header{}
	headers.Set("x-codex-primary-used-percent", formatUsageTestFloat(used7d))
	headers.Set("x-codex-primary-reset-after-seconds", formatUsageTestInt(reset7dSeconds))
	headers.Set("x-codex-primary-window-minutes", "10080")
	headers.Set("x-codex-secondary-used-percent", formatUsageTestFloat(used5h))
	headers.Set("x-codex-secondary-reset-after-seconds", formatUsageTestInt(reset5hSeconds))
	headers.Set("x-codex-secondary-window-minutes", "300")
	return headers
}

func formatUsageTestFloat(value float64) string {
	if value == float64(int64(value)) {
		return formatUsageTestInt(int(value))
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatUsageTestInt(value int) string {
	return strconv.Itoa(value)
}

func hasModelScopeCall(repo *openAI429SnapshotRepo, scope string) bool {
	for _, call := range repo.modelRateLimitCalls {
		if call.scope == scope {
			return true
		}
	}
	return false
}

func TestHandle429_OpenAIPersistsCodexSnapshotImmediately(t *testing.T) {
	repo := &openAI429SnapshotRepo{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{ID: 123, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	headers := buildCodexHeaders(100, 604800, 100, 18000)

	svc.handle429(context.Background(), account, headers, nil)

	if repo.rateLimitedID != 0 {
		t.Fatalf("expected no account rate limit, got %d", repo.rateLimitedID)
	}
	if repo.modelScope != openAICodexScopeNormal {
		t.Fatalf("modelScope = %q, want %q", repo.modelScope, openAICodexScopeNormal)
	}
	if len(repo.updatedExtra) == 0 {
		t.Fatal("expected codex snapshot to be persisted on 429")
	}
	if got := repo.updatedExtra["codex_5h_used_percent"]; got != 100.0 {
		t.Fatalf("codex_5h_used_percent = %v, want 100", got)
	}
	if got := repo.updatedExtra["codex_7d_used_percent"]; got != 100.0 {
		t.Fatalf("codex_7d_used_percent = %v, want 100", got)
	}
}

func TestHandle429_OpenAISpark429ScopesToSparkModelLimit(t *testing.T) {
	repo := &openAI429SnapshotRepo{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{
		ID:       124,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
	}

	headers := buildCodexHeaders(100, 604800, 44, 18000)
	ctx := WithOpenAICodexRequestModel(context.Background(), "gpt-5.3-codex-spark")

	svc.handle429(ctx, account, headers, nil)

	if repo.rateLimitedID != 0 {
		t.Fatalf("expected no account rate limit, got %d", repo.rateLimitedID)
	}
	if repo.modelScope != openAICodexScopeSpark {
		t.Fatalf("modelScope = %q, want %q", repo.modelScope, openAICodexScopeSpark)
	}
	if got := repo.updatedExtra[codexSpark5hUsedPercentKey]; got != 44.0 {
		t.Fatalf("codex_spark_5h_used_percent = %v, want 44", got)
	}
	if got := repo.updatedExtra[codexSpark7dUsedPercentKey]; got != 100.0 {
		t.Fatalf("codex_spark_7d_used_percent = %v, want 100", got)
	}
	if _, ok := repo.updatedExtra["codex_7d_used_percent"]; ok {
		t.Fatal("expected spark 429 to avoid normal codex fields")
	}
}

func TestHandle429_OpenAIProNormal429IgnoresModelMappingForSnapshotScope(t *testing.T) {
	repo := &openAI429SnapshotRepo{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{
		ID:       125,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
			"model_mapping": map[string]any{
				"gpt-5.4": "gpt-5.3-codex-spark-high",
			},
		},
	}

	headers := buildCodexHeaders(100, 604800, 44, 18000)
	ctx := WithOpenAICodexRequestModel(context.Background(), "gpt-5.4")

	svc.handle429(ctx, account, headers, nil)

	if repo.rateLimitedID != 0 {
		t.Fatalf("expected no whole-account rate limit, got %d", repo.rateLimitedID)
	}
	if repo.modelScope != openAICodexScopeNormal {
		t.Fatalf("modelScope = %q, want %q", repo.modelScope, openAICodexScopeNormal)
	}
	if got := repo.updatedExtra["codex_5h_used_percent"]; got != 44.0 {
		t.Fatalf("codex_5h_used_percent = %v, want 44", got)
	}
	if got := repo.updatedExtra["codex_7d_used_percent"]; got != 100.0 {
		t.Fatalf("codex_7d_used_percent = %v, want 100", got)
	}
	if _, ok := repo.updatedExtra[codexSpark7dUsedPercentKey]; ok {
		t.Fatal("expected normal 429 to avoid spark codex fields")
	}
}

func TestSyncOpenAICodexRateLimitState_OnlyNormal7dLimitsNormalScope(t *testing.T) {
	repo := &openAI429SnapshotRepo{}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	resetAt := now.Add(7 * 24 * time.Hour).UTC().Truncate(time.Second)
	account := &Account{
		ID:       201,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra:    map[string]any{},
	}

	state := syncOpenAICodexRateLimitState(
		WithOpenAICodexRequestModel(context.Background(), "gpt-5.3-codex"),
		repo,
		account,
		map[string]any{
			"codex_7d_used_percent": 100.0,
			"codex_7d_reset_at":     resetAt.Format(time.RFC3339),
		},
		now,
	)

	if state == nil || state.ScopeResetAt == nil {
		t.Fatal("expected normal scope reset state")
	}
	if state.AccountResetAt != nil {
		t.Fatalf("expected no account reset, got %v", state.AccountResetAt)
	}
	if repo.rateLimitedID != 0 {
		t.Fatalf("expected no account rate limit, got %d", repo.rateLimitedID)
	}
	if len(repo.modelRateLimitCalls) != 1 {
		t.Fatalf("model rate limit calls = %d, want 1", len(repo.modelRateLimitCalls))
	}
	if repo.modelRateLimitCalls[0].scope != openAICodexScopeNormal {
		t.Fatalf("scope = %q, want %q", repo.modelRateLimitCalls[0].scope, openAICodexScopeNormal)
	}
}

func TestSyncOpenAICodexRateLimitState_OnlySpark7dLimitsSparkScope(t *testing.T) {
	repo := &openAI429SnapshotRepo{}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	resetAt := now.Add(7 * 24 * time.Hour).UTC().Truncate(time.Second)
	account := &Account{
		ID:       202,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{},
	}

	state := syncOpenAICodexRateLimitState(
		WithOpenAICodexRequestModel(context.Background(), "gpt-5.3-codex-spark"),
		repo,
		account,
		map[string]any{
			codexSpark7dUsedPercentKey: 100.0,
			codexSpark7dResetAtKey:     resetAt.Format(time.RFC3339),
		},
		now,
	)

	if state == nil || state.ScopeResetAt == nil {
		t.Fatal("expected spark scope reset state")
	}
	if state.AccountResetAt != nil {
		t.Fatalf("expected no account reset, got %v", state.AccountResetAt)
	}
	if repo.rateLimitedID != 0 {
		t.Fatalf("expected no account rate limit, got %d", repo.rateLimitedID)
	}
	if len(repo.modelRateLimitCalls) != 1 {
		t.Fatalf("model rate limit calls = %d, want 1", len(repo.modelRateLimitCalls))
	}
	if repo.modelRateLimitCalls[0].scope != openAICodexScopeSpark {
		t.Fatalf("scope = %q, want %q", repo.modelRateLimitCalls[0].scope, openAICodexScopeSpark)
	}
}

func TestSyncOpenAICodexRateLimitState_Both7dTriggersWholeAccountLimit(t *testing.T) {
	repo := &openAI429SnapshotRepo{}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	normalResetAt := now.Add(24 * time.Hour).UTC().Truncate(time.Second)
	sparkResetAt := now.Add(48 * time.Hour).UTC().Truncate(time.Second)
	account := &Account{
		ID:       203,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{},
	}

	state := syncOpenAICodexRateLimitState(
		WithOpenAICodexRequestModel(context.Background(), "gpt-5.3-codex"),
		repo,
		account,
		map[string]any{
			"codex_7d_used_percent":    100.0,
			"codex_7d_reset_at":        normalResetAt.Format(time.RFC3339),
			codexSpark7dUsedPercentKey: 100.0,
			codexSpark7dResetAtKey:     sparkResetAt.Format(time.RFC3339),
		},
		now,
	)

	if state == nil || state.AccountResetAt == nil {
		t.Fatal("expected account reset state")
	}
	if !state.AccountResetAt.Equal(sparkResetAt) {
		t.Fatalf("account resetAt = %v, want %v", *state.AccountResetAt, sparkResetAt)
	}
	if repo.rateLimitedID != account.ID {
		t.Fatalf("rateLimitedID = %d, want %d", repo.rateLimitedID, account.ID)
	}
	if repo.rateLimitedAt == nil || !repo.rateLimitedAt.Equal(sparkResetAt) {
		t.Fatalf("rateLimitedAt = %v, want %v", repo.rateLimitedAt, sparkResetAt)
	}
	if reason := repo.updatedExtra["rate_limit_reason"]; reason != AccountRateLimitReasonUsage7dAll {
		t.Fatalf("rate_limit_reason = %v, want %q", reason, AccountRateLimitReasonUsage7dAll)
	}
	if !hasModelScopeCall(repo, openAICodexScopeNormal) || !hasModelScopeCall(repo, openAICodexScopeSpark) {
		t.Fatalf("expected both normal and spark model rate limits, got %+v", repo.modelRateLimitCalls)
	}
}

func TestSyncOpenAICodexRateLimitState_Both5hDoesNotTriggerWholeAccountLimit(t *testing.T) {
	repo := &openAI429SnapshotRepo{}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	resetAt := now.Add(5 * time.Hour).UTC().Truncate(time.Second)
	account := &Account{
		ID:       204,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{},
	}

	state := syncOpenAICodexRateLimitState(
		WithOpenAICodexRequestModel(context.Background(), "gpt-5.3-codex"),
		repo,
		account,
		map[string]any{
			"codex_5h_used_percent":    100.0,
			"codex_5h_reset_at":        resetAt.Format(time.RFC3339),
			codexSpark5hUsedPercentKey: 100.0,
			codexSpark5hResetAtKey:     resetAt.Format(time.RFC3339),
		},
		now,
	)

	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.AccountResetAt != nil {
		t.Fatalf("expected no account reset, got %v", *state.AccountResetAt)
	}
	if repo.rateLimitedID != 0 {
		t.Fatalf("expected no account rate limit, got %d", repo.rateLimitedID)
	}
	if !hasModelScopeCall(repo, openAICodexScopeNormal) || !hasModelScopeCall(repo, openAICodexScopeSpark) {
		t.Fatalf("expected both scope model limits, got %+v", repo.modelRateLimitCalls)
	}
}

func TestSyncOpenAICodexRateLimitState_Mixed5hAnd7dDoesNotTriggerWholeAccountLimit(t *testing.T) {
	repo := &openAI429SnapshotRepo{}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	normalResetAt := now.Add(7 * 24 * time.Hour).UTC().Truncate(time.Second)
	sparkResetAt := now.Add(5 * time.Hour).UTC().Truncate(time.Second)
	account := &Account{
		ID:       205,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{},
	}

	state := syncOpenAICodexRateLimitState(
		WithOpenAICodexRequestModel(context.Background(), "gpt-5.3-codex"),
		repo,
		account,
		map[string]any{
			"codex_7d_used_percent":    100.0,
			"codex_7d_reset_at":        normalResetAt.Format(time.RFC3339),
			codexSpark5hUsedPercentKey: 100.0,
			codexSpark5hResetAtKey:     sparkResetAt.Format(time.RFC3339),
		},
		now,
	)

	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.AccountResetAt != nil {
		t.Fatalf("expected no account reset, got %v", *state.AccountResetAt)
	}
	if repo.rateLimitedID != 0 {
		t.Fatalf("expected no account rate limit, got %d", repo.rateLimitedID)
	}
	if !hasModelScopeCall(repo, openAICodexScopeNormal) || !hasModelScopeCall(repo, openAICodexScopeSpark) {
		t.Fatalf("expected both scope model limits, got %+v", repo.modelRateLimitCalls)
	}
}

func TestSyncOpenAICodexRateLimitFromExtra_Usage7dAllWaitsForLaterReset(t *testing.T) {
	repo := &openAI429SnapshotRepo{}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	accountResetAt := now.Add(24 * time.Hour).UTC().Truncate(time.Second)
	account := &Account{
		ID:               206,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		RateLimitResetAt: &accountResetAt,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{
			"rate_limit_reason":           AccountRateLimitReasonUsage7dAll,
			codexAccountAll7dExhaustedKey: true,
			"codex_7d_used_percent":       100.0,
			"codex_7d_reset_at":           now.Add(-time.Minute).Format(time.RFC3339),
			codexSpark7dUsedPercentKey:    100.0,
			codexSpark7dResetAtKey:        accountResetAt.Format(time.RFC3339),
		},
	}

	state := syncOpenAICodexRateLimitFromExtra(context.Background(), repo, account, now)
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.AccountResetAt != nil {
		t.Fatalf("expected account reset to stay persisted instead of recalculated, got %v", state.AccountResetAt)
	}
	if account.RateLimitResetAt == nil || !account.RateLimitResetAt.Equal(accountResetAt) {
		t.Fatalf("account rate limit resetAt = %v, want %v", account.RateLimitResetAt, accountResetAt)
	}
	if repo.clearRateLimitCalls != 0 {
		t.Fatalf("expected usage_7d_all not to clear early, got %d clear calls", repo.clearRateLimitCalls)
	}
}

func TestNormalizedCodexLimits(t *testing.T) {
	// Test the Normalize() method directly
	pUsed := 100.0
	pReset := 384607
	pWindow := 10080
	sUsed := 3.0
	sReset := 17369
	sWindow := 300

	snapshot := &OpenAICodexUsageSnapshot{
		PrimaryUsedPercent:         &pUsed,
		PrimaryResetAfterSeconds:   &pReset,
		PrimaryWindowMinutes:       &pWindow,
		SecondaryUsedPercent:       &sUsed,
		SecondaryResetAfterSeconds: &sReset,
		SecondaryWindowMinutes:     &sWindow,
	}

	normalized := snapshot.Normalize()
	if normalized == nil {
		t.Fatal("expected non-nil normalized")
	}

	// Primary has larger window (10080 > 300), so primary should be 7d
	if normalized.Used7dPercent == nil || *normalized.Used7dPercent != 100.0 {
		t.Errorf("expected Used7dPercent=100, got %v", normalized.Used7dPercent)
	}
	if normalized.Reset7dSeconds == nil || *normalized.Reset7dSeconds != 384607 {
		t.Errorf("expected Reset7dSeconds=384607, got %v", normalized.Reset7dSeconds)
	}
	if normalized.Used5hPercent == nil || *normalized.Used5hPercent != 3.0 {
		t.Errorf("expected Used5hPercent=3, got %v", normalized.Used5hPercent)
	}
	if normalized.Reset5hSeconds == nil || *normalized.Reset5hSeconds != 17369 {
		t.Errorf("expected Reset5hSeconds=17369, got %v", normalized.Reset5hSeconds)
	}
}

func TestNormalizedCodexLimits_OnlyPrimaryData(t *testing.T) {
	// Test when only primary has data, no window_minutes
	pUsed := 80.0
	pReset := 50000

	snapshot := &OpenAICodexUsageSnapshot{
		PrimaryUsedPercent:       &pUsed,
		PrimaryResetAfterSeconds: &pReset,
		// No window_minutes, no secondary data
	}

	normalized := snapshot.Normalize()
	if normalized == nil {
		t.Fatal("expected non-nil normalized")
	}

	// Legacy assumption: primary=7d, secondary=5h
	if normalized.Used7dPercent == nil || *normalized.Used7dPercent != 80.0 {
		t.Errorf("expected Used7dPercent=80, got %v", normalized.Used7dPercent)
	}
	if normalized.Reset7dSeconds == nil || *normalized.Reset7dSeconds != 50000 {
		t.Errorf("expected Reset7dSeconds=50000, got %v", normalized.Reset7dSeconds)
	}
	// Secondary (5h) should be nil
	if normalized.Used5hPercent != nil {
		t.Errorf("expected Used5hPercent=nil, got %v", *normalized.Used5hPercent)
	}
	if normalized.Reset5hSeconds != nil {
		t.Errorf("expected Reset5hSeconds=nil, got %v", *normalized.Reset5hSeconds)
	}
}

func TestNormalizedCodexLimits_OnlySecondaryData(t *testing.T) {
	// Test when only secondary has data, no window_minutes
	sUsed := 60.0
	sReset := 3000

	snapshot := &OpenAICodexUsageSnapshot{
		SecondaryUsedPercent:       &sUsed,
		SecondaryResetAfterSeconds: &sReset,
		// No window_minutes, no primary data
	}

	normalized := snapshot.Normalize()
	if normalized == nil {
		t.Fatal("expected non-nil normalized")
	}

	// Legacy assumption: primary=7d, secondary=5h
	// So secondary goes to 5h
	if normalized.Used5hPercent == nil || *normalized.Used5hPercent != 60.0 {
		t.Errorf("expected Used5hPercent=60, got %v", normalized.Used5hPercent)
	}
	if normalized.Reset5hSeconds == nil || *normalized.Reset5hSeconds != 3000 {
		t.Errorf("expected Reset5hSeconds=3000, got %v", normalized.Reset5hSeconds)
	}
	// Primary (7d) should be nil
	if normalized.Used7dPercent != nil {
		t.Errorf("expected Used7dPercent=nil, got %v", *normalized.Used7dPercent)
	}
}

func TestNormalizedCodexLimits_BothDataNoWindowMinutes(t *testing.T) {
	// Test when both have data but no window_minutes
	pUsed := 100.0
	pReset := 400000
	sUsed := 50.0
	sReset := 10000

	snapshot := &OpenAICodexUsageSnapshot{
		PrimaryUsedPercent:         &pUsed,
		PrimaryResetAfterSeconds:   &pReset,
		SecondaryUsedPercent:       &sUsed,
		SecondaryResetAfterSeconds: &sReset,
		// No window_minutes
	}

	normalized := snapshot.Normalize()
	if normalized == nil {
		t.Fatal("expected non-nil normalized")
	}

	// Legacy assumption: primary=7d, secondary=5h
	if normalized.Used7dPercent == nil || *normalized.Used7dPercent != 100.0 {
		t.Errorf("expected Used7dPercent=100, got %v", normalized.Used7dPercent)
	}
	if normalized.Reset7dSeconds == nil || *normalized.Reset7dSeconds != 400000 {
		t.Errorf("expected Reset7dSeconds=400000, got %v", normalized.Reset7dSeconds)
	}
	if normalized.Used5hPercent == nil || *normalized.Used5hPercent != 50.0 {
		t.Errorf("expected Used5hPercent=50, got %v", normalized.Used5hPercent)
	}
	if normalized.Reset5hSeconds == nil || *normalized.Reset5hSeconds != 10000 {
		t.Errorf("expected Reset5hSeconds=10000, got %v", normalized.Reset5hSeconds)
	}
}

func TestHandle429_AnthropicPlatformUnaffected(t *testing.T) {
	// Verify that Anthropic platform accounts still use the original logic
	// This test ensures we don't break existing Claude account rate limiting

	svc := &RateLimitService{}

	// Simulate Anthropic 429 headers
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-reset", "1737820800") // A future Unix timestamp

	// For Anthropic platform, calculateOpenAI429ResetTime should return nil
	// because it only handles OpenAI platform
	resetAt := svc.calculateOpenAI429ResetTime(headers)

	// Should return nil since there are no x-codex-* headers
	if resetAt != nil {
		t.Errorf("expected nil for Anthropic headers, got %v", resetAt)
	}
}

func TestCalculateOpenAI429ResetTime_UserProvidedScenario(t *testing.T) {
	// This is the exact scenario from the user:
	// codex_7d_used_percent: 100
	// codex_7d_reset_after_seconds: 384607 (约4.5天后重置)
	// codex_5h_used_percent: 3
	// codex_5h_reset_after_seconds: 17369 (约4.8小时后重置)

	svc := &RateLimitService{}

	// Simulate headers matching user's data
	// Note: We need to map the canonical 5h/7d back to primary/secondary
	// Based on typical OpenAI behavior: primary=7d (larger window), secondary=5h (smaller window)
	headers := http.Header{}
	headers.Set("x-codex-primary-used-percent", "100")
	headers.Set("x-codex-primary-reset-after-seconds", "384607")
	headers.Set("x-codex-primary-window-minutes", "10080") // 7 days = 10080 minutes
	headers.Set("x-codex-secondary-used-percent", "3")
	headers.Set("x-codex-secondary-reset-after-seconds", "17369")
	headers.Set("x-codex-secondary-window-minutes", "300") // 5 hours = 300 minutes

	before := time.Now()
	resetAt := svc.calculateOpenAI429ResetTime(headers)
	after := time.Now()

	if resetAt == nil {
		t.Fatal("expected non-nil resetAt for user scenario")
	}

	// Should use the 7d reset time (384607 seconds) since 7d limit is exhausted (100%)
	expectedDuration := 384607 * time.Second
	minExpected := before.Add(expectedDuration)
	maxExpected := after.Add(expectedDuration)

	if resetAt.Before(minExpected) || resetAt.After(maxExpected) {
		t.Errorf("resetAt %v not in expected range [%v, %v]", resetAt, minExpected, maxExpected)
	}

	// Verify it's approximately 4.45 days (384607 seconds)
	duration := resetAt.Sub(before)
	actualDays := duration.Hours() / 24.0

	// 384607 / 86400 = ~4.45 days
	if actualDays < 4.4 || actualDays > 4.5 {
		t.Errorf("expected ~4.45 days, got %.2f days", actualDays)
	}

	t.Logf("User scenario: reset_at=%v, duration=%.2f days", resetAt, actualDays)
}

func TestCalculateOpenAI429ResetTime_5MinFallbackWhenNoReset(t *testing.T) {
	// Test that we return nil when there's used_percent but no reset_after_seconds
	// This should cause the caller to use the default 5-minute fallback

	svc := &RateLimitService{}

	headers := http.Header{}
	headers.Set("x-codex-primary-used-percent", "100")
	// No reset_after_seconds!

	resetAt := svc.calculateOpenAI429ResetTime(headers)

	// Should return nil since there's no reset time available
	if resetAt != nil {
		t.Errorf("expected nil when no reset_after_seconds, got %v", resetAt)
	}
}
