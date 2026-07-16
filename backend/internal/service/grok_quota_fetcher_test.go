package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

type grokQuotaAccountRepoStub struct {
	AccountRepository
	account          *Account
	accounts         []Account
	updateExtraCalls []struct {
		id      int64
		updates map[string]any
	}
	casCalls             []int64
	casApplied           bool
	credentialCASCalls   []int64
	credentialCASApplied bool
	lastCredentials      map[string]any
	listAfterIDs         []int64
	listLimits           []int
	listWithCalls        int
}

func (r *grokQuotaAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if r.account != nil && r.account.ID == id {
		account := cloneTestAccount(r.account)
		return account, nil
	}
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			account := cloneTestAccount(&r.accounts[i])
			return account, nil
		}
	}
	return nil, ErrAccountNotFound
}

func (r *grokQuotaAccountRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, _, _, _, _ string, _ int64, _ string, _ string) ([]Account, *pagination.PaginationResult, error) {
	r.listWithCalls++
	limit := params.PageSize
	if limit <= 0 || limit > len(r.accounts) {
		limit = len(r.accounts)
	}
	out := make([]Account, 0, limit)
	for i := len(r.accounts) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, *cloneTestAccount(&r.accounts[i]))
	}
	return out, &pagination.PaginationResult{Page: 1, PageSize: params.PageSize, Total: int64(len(r.accounts)), Pages: 1}, nil
}

func (r *grokQuotaAccountRepoStub) ListActiveGrokOAuthAfterID(_ context.Context, afterID int64, limit int) ([]Account, error) {
	r.listAfterIDs = append(r.listAfterIDs, afterID)
	r.listLimits = append(r.listLimits, limit)
	out := make([]Account, 0, limit)
	for i := range r.accounts {
		account := r.accounts[i]
		if account.ID <= afterID || !account.IsGrokOAuth() || account.Status != StatusActive {
			continue
		}
		out = append(out, *cloneTestAccount(&account))
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (r *grokQuotaAccountRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	r.updateExtraCalls = append(r.updateExtraCalls, struct {
		id      int64
		updates map[string]any
	}{id: id, updates: cloneTestMap(updates)})
	return nil
}

func (r *grokQuotaAccountRepoStub) SetGrokOAuthErrorIfCredentialsUnchanged(_ context.Context, id int64, _ map[string]any, _ string) (bool, error) {
	r.casCalls = append(r.casCalls, id)
	return r.casApplied, nil
}

func (r *grokQuotaAccountRepoStub) UpdateGrokOAuthCredentialsIfCredentialsUnchanged(_ context.Context, id int64, _ map[string]any, credentials map[string]any) (bool, error) {
	r.credentialCASCalls = append(r.credentialCASCalls, id)
	if !r.credentialCASApplied {
		return false, nil
	}
	r.lastCredentials = cloneTestMap(credentials)
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			r.accounts[i].Credentials = cloneTestMap(credentials)
			break
		}
	}
	if r.account != nil && r.account.ID == id {
		r.account.Credentials = cloneTestMap(credentials)
	}
	return true, nil
}

func (r *grokQuotaAccountRepoStub) SetRateLimited(_ context.Context, id int64, resetAt time.Time) error {
	if r.account != nil && r.account.ID == id {
		now := time.Now().UTC()
		r.account.RateLimitedAt = &now
		r.account.RateLimitResetAt = &resetAt
	}
	return nil
}

func (r *grokQuotaAccountRepoStub) SetRateLimitedIfLater(ctx context.Context, id int64, resetAt time.Time) error {
	return r.SetRateLimited(ctx, id, resetAt)
}

type grokQuotaHTTPUpstreamStub struct {
	responses []*http.Response
	requests  []*http.Request
	mu        sync.Mutex
}

func (s *grokQuotaHTTPUpstreamStub) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var body []byte
	if req != nil && req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	if req != nil {
		cloned := req.Clone(req.Context())
		cloned.Body = io.NopCloser(bytes.NewReader(body))
		s.requests = append(s.requests, cloned)
	}
	idx := len(s.requests) - 1
	if idx < len(s.responses) {
		return s.responses[idx], nil
	}
	return newGrokQuotaHTTPResponse(http.StatusOK, `{}`), nil
}

func (s *grokQuotaHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, accountConcurrency)
}

func newGrokQuotaHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}
}

func cloneTestAccount(account *Account) *Account {
	if account == nil {
		return nil
	}
	next := *account
	next.Credentials = cloneTestMap(account.Credentials)
	next.Extra = cloneTestMap(account.Extra)
	return &next
}

func TestGrokQuotaFetcherBuildUsageInfoFromBillingAndQuotaSnapshots(t *testing.T) {
	limit := float64(xai.SuperGrokLimitCents)
	used := float64(3000)
	usedPercent := float64(20)
	reqLimit := int64(100)
	reqRemaining := int64(25)
	tokLimit := int64(1000)
	tokRemaining := int64(750)
	retryAfter := 30

	account := &Account{
		ID:       91,
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			grokBillingExtraKey: map[string]any{
				"monthly_limit_cents": limit,
				"used_cents":          used,
				"used_percent":        usedPercent,
				"plan":                "SuperGrok",
				"status_code":         http.StatusOK,
				"updated_at":          "2026-07-15T10:00:00Z",
				"fetched_at":          "2026-07-15T10:00:00Z",
			},
			grokQuotaSnapshotExtraKey: map[string]any{
				"requests": map[string]any{
					"limit":      reqLimit,
					"remaining":  reqRemaining,
					"reset_at":   "2026-07-15T11:00:00Z",
					"reset_unix": float64(1784113200),
				},
				"tokens": map[string]any{
					"limit":     tokLimit,
					"remaining": tokRemaining,
				},
				"retry_after_seconds":  retryAfter,
				"entitlement_status":   "active",
				"status_code":          http.StatusTooManyRequests,
				"headers_observed":     true,
				"last_headers_seen_at": "2026-07-15T10:01:00Z",
				"updated_at":           "2026-07-15T10:01:00Z",
			},
		},
	}

	usage := NewGrokQuotaFetcher().BuildUsageInfo(account)

	require.Equal(t, "passive", usage.Source)
	require.NotNil(t, usage.GrokBilling)
	require.Equal(t, "SuperGrok", usage.SubscriptionTier)
	require.NotNil(t, usage.GrokRequestQuota)
	require.EqualValues(t, reqLimit, *usage.GrokRequestQuota.Limit)
	require.EqualValues(t, reqRemaining, *usage.GrokRequestQuota.Remaining)
	require.NotNil(t, usage.GrokTokenQuota)
	require.EqualValues(t, tokRemaining, *usage.GrokTokenQuota.Remaining)
	require.NotNil(t, usage.GrokRetryAfterSeconds)
	require.Equal(t, retryAfter, *usage.GrokRetryAfterSeconds)
	require.Equal(t, "active", usage.GrokEntitlementStatus)
	require.Equal(t, "billing_observed", usage.GrokQuotaSnapshotState)
	require.Equal(t, http.StatusTooManyRequests, usage.GrokLastStatusCode)
	require.Equal(t, "rate_limited", usage.ErrorCode)
	require.NotNil(t, usage.UpdatedAt)
	require.Equal(t, time.Date(2026, 7, 15, 10, 1, 0, 0, time.UTC), *usage.UpdatedAt)
}

func TestClassifyGrokOAuthReconcileAccount(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		account    *Account
		wantReason string
		wantAction string
		wantOK     bool
	}{
		{
			name: "missing refresh token blocks",
			account: &Account{
				Platform:    PlatformGrok,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Credentials: map[string]any{"access_token": "a", "expires_at": now.Add(time.Hour).Format(time.RFC3339)},
			},
			wantReason: GrokOAuthReconcileReasonMissingRefreshToken,
			wantAction: GrokOAuthReconcileActionBlock,
			wantOK:     true,
		},
		{
			name: "missing access token refreshes",
			account: &Account{
				Platform:    PlatformGrok,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Credentials: map[string]any{"refresh_token": "r", "expires_at": now.Add(time.Hour).Format(time.RFC3339)},
			},
			wantReason: GrokOAuthReconcileReasonMissingAccessToken,
			wantAction: GrokOAuthReconcileActionRefresh,
			wantOK:     true,
		},
		{
			name: "near expiry refreshes",
			account: &Account{
				Platform: PlatformGrok,
				Type:     AccountTypeOAuth,
				Status:   StatusActive,
				Credentials: map[string]any{
					"access_token":  "a",
					"refresh_token": "r",
					"expires_at":    now.Add(time.Minute).Format(time.RFC3339),
				},
			},
			wantReason: GrokOAuthReconcileReasonNearExpiry,
			wantAction: GrokOAuthReconcileActionRefresh,
			wantOK:     true,
		},
		{
			name: "healthy skips",
			account: &Account{
				Platform: PlatformGrok,
				Type:     AccountTypeOAuth,
				Status:   StatusActive,
				Credentials: map[string]any{
					"access_token":  "a",
					"refresh_token": "r",
					"expires_at":    now.Add(time.Hour).Format(time.RFC3339),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, action, ok := classifyGrokOAuthReconcileAccount(tt.account, 3*time.Minute)
			require.Equal(t, tt.wantReason, reason)
			require.Equal(t, tt.wantAction, action)
			require.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestReconcileGrokOAuth_UsesAscendingAfterIDCursor(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	repo := &grokQuotaAccountRepoStub{accounts: []Account{
		{
			ID:       10,
			Platform: PlatformGrok,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"access_token":  "a",
				"refresh_token": "r",
				"expires_at":    now,
			},
			Extra: map[string]any{
				"model_scope_v2": DefaultGrokBuildTextModelScope().ToMap(),
			},
		},
		{
			ID:          20,
			Platform:    PlatformGrok,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Credentials: map[string]any{"access_token": "a", "expires_at": now},
		},
		{
			ID:          30,
			Platform:    PlatformGrok,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Credentials: map[string]any{"access_token": "a", "expires_at": now},
		},
	}}
	svc := &TokenRefreshService{accountRepo: repo}

	result, err := svc.ReconcileGrokOAuth(context.Background(), GrokOAuthReconcileInput{
		AfterID: 10,
		Limit:   2,
	})

	require.NoError(t, err)
	require.Equal(t, []int64{10}, repo.listAfterIDs)
	require.Equal(t, []int{2}, repo.listLimits)
	require.Equal(t, 2, result.Scanned)
	require.Equal(t, int64(30), result.NextAfterID)
	require.True(t, result.HasMore)
	require.Equal(t, 2, result.Actionable)
	require.Equal(t, 2, result.WouldBlock)
	require.Len(t, result.Items, 2)
	require.Equal(t, int64(20), result.Items[0].AccountID)
	require.Equal(t, int64(30), result.Items[1].AccountID)
	require.Empty(t, repo.updateExtraCalls, "dry-run must not persist scope backfill")
}

func TestReconcileGrokOAuth_ApplyBlocksMissingRefreshAndBackfillsScope(t *testing.T) {
	t.Parallel()

	repo := &grokQuotaAccountRepoStub{
		casApplied: true,
		accounts: []Account{{
			ID:          22,
			Platform:    PlatformGrok,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Credentials: map[string]any{"access_token": "a", "expires_at": time.Now().UTC().Add(time.Hour).Format(time.RFC3339)},
			Extra:       map[string]any{},
		}},
	}
	svc := &TokenRefreshService{accountRepo: repo}

	result, err := svc.ReconcileGrokOAuth(context.Background(), GrokOAuthReconcileInput{
		Apply: true,
		Limit: 10,
	})

	require.NoError(t, err)
	require.False(t, result.DryRun)
	require.Equal(t, 1, result.Scanned)
	require.Equal(t, 1, result.Blocked)
	require.Equal(t, []int64{22}, repo.casCalls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, int64(22), repo.updateExtraCalls[0].id)
	scope, ok := ExtractAccountModelScopeV2(repo.updateExtraCalls[0].updates)
	require.True(t, ok)
	require.Len(t, scope.Entries, len(GrokBuildTextModelIDs()))
}

func TestReconcileGrokOAuth_ApplySkipsWhenCASMisses(t *testing.T) {
	t.Parallel()

	repo := &grokQuotaAccountRepoStub{accounts: []Account{{
		ID:          23,
		Platform:    PlatformGrok,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Credentials: map[string]any{"access_token": "a", "expires_at": time.Now().UTC().Add(time.Hour).Format(time.RFC3339)},
		Extra: map[string]any{
			"model_scope_v2": DefaultGrokBuildTextModelScope().ToMap(),
		},
	}}}
	svc := &TokenRefreshService{accountRepo: repo}

	result, err := svc.ReconcileGrokOAuth(context.Background(), GrokOAuthReconcileInput{
		Apply: true,
		Limit: 10,
	})

	require.NoError(t, err)
	require.Equal(t, 0, result.Blocked)
	require.Equal(t, 1, result.Skipped)
	require.Len(t, result.Items, 1)
	require.Equal(t, GrokOAuthReconcileOutcomeSkipped, result.Items[0].Outcome)
	require.Equal(t, []int64{23}, repo.casCalls)
}

func TestReconcileGrokOAuth_ApplyRefreshesExpiredTokenWithCredentialCAS(t *testing.T) {
	t.Parallel()

	repo := &grokQuotaAccountRepoStub{
		credentialCASApplied: true,
		accounts: []Account{{
			ID:       24,
			Platform: PlatformGrok,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"access_token":  "old-access",
				"refresh_token": "old-refresh",
				"expires_at":    time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
			},
			Extra: map[string]any{
				"model_scope_v2": DefaultGrokBuildTextModelScope().ToMap(),
			},
		}},
	}
	oauth := NewGrokOAuthService(nil, &grokOAuthClientStub{}, nil)
	refreshAPI := NewOAuthRefreshAPI(repo, nil)
	svc := &TokenRefreshService{
		accountRepo: repo,
		refreshAPI:  refreshAPI,
		refreshers:  []TokenRefresher{NewGrokTokenRefresher(oauth)},
	}

	result, err := svc.ReconcileGrokOAuth(context.Background(), GrokOAuthReconcileInput{
		Apply: true,
		Limit: 10,
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Refreshed)
	require.Equal(t, []int64{24}, repo.credentialCASCalls)
	require.Equal(t, "refreshed-access", repo.lastCredentials["access_token"])
	require.Equal(t, "rotated-refresh", repo.lastCredentials["refresh_token"])
	require.NotEmpty(t, repo.lastCredentials["_token_version"])
	require.Len(t, result.Items, 1)
	require.Equal(t, GrokOAuthReconcileOutcomeApplied, result.Items[0].Outcome)
}

func TestReconcileGrokOAuth_ApplyRefreshSkipsOnCredentialCASMiss(t *testing.T) {
	t.Parallel()

	repo := &grokQuotaAccountRepoStub{
		credentialCASApplied: false,
		accounts: []Account{{
			ID:       25,
			Platform: PlatformGrok,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"access_token":  "old-access",
				"refresh_token": "old-refresh",
				"expires_at":    time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
			},
			Extra: map[string]any{
				"model_scope_v2": DefaultGrokBuildTextModelScope().ToMap(),
			},
		}},
	}
	oauth := NewGrokOAuthService(nil, &grokOAuthClientStub{}, nil)
	svc := &TokenRefreshService{
		accountRepo: repo,
		refreshAPI:  NewOAuthRefreshAPI(repo, nil),
		refreshers:  []TokenRefresher{NewGrokTokenRefresher(oauth)},
	}

	result, err := svc.ReconcileGrokOAuth(context.Background(), GrokOAuthReconcileInput{
		Apply: true,
		Limit: 10,
	})

	require.NoError(t, err)
	require.Equal(t, 0, result.Refreshed)
	require.Equal(t, 1, result.Skipped)
	require.Equal(t, []int64{25}, repo.credentialCASCalls)
	require.Nil(t, repo.lastCredentials)
	require.Len(t, result.Items, 1)
	require.Equal(t, GrokOAuthReconcileOutcomeSkipped, result.Items[0].Outcome)
}

func TestGrokQuotaService_QueryQuotaBillingAuthoritativeSkipsResponsesProbe(t *testing.T) {
	t.Parallel()

	repo := &grokQuotaAccountRepoStub{account: newGrokQuotaOAuthAccountForTest(101)}
	billingBody := `{"config":{"currentPeriod":{"type":"weekly","start":"2026-07-15T00:00:00Z","end":"2026-07-22T00:00:00Z"},"creditUsagePercent":12.5,"monthlyLimit":{"val":15000},"used":{"val":3000},"billingPeriodStart":"2026-07-01T00:00:00Z","billingPeriodEnd":"2026-08-01T00:00:00Z"}}`
	upstream := &grokQuotaHTTPUpstreamStub{responses: []*http.Response{
		newGrokQuotaHTTPResponse(http.StatusOK, billingBody),
		newGrokQuotaHTTPResponse(http.StatusOK, billingBody),
	}}
	svc := NewGrokQuotaService(repo, nil, NewGrokTokenProvider(repo, nil), upstream, nil)

	result, err := svc.QueryQuota(context.Background(), 101)

	require.NoError(t, err)
	require.Equal(t, "billing_probe", result.Source)
	require.NotNil(t, result.Billing)
	require.Equal(t, "SuperGrok", result.Billing.Plan)
	require.Len(t, upstream.requests, 2)
	for _, req := range upstream.requests {
		require.Equal(t, http.MethodGet, req.Method)
		require.Contains(t, req.URL.Path, "/billing")
		require.Equal(t, "Bearer access-token", req.Header.Get("Authorization"))
	}
	require.Len(t, repo.updateExtraCalls, 1)
	require.Contains(t, repo.updateExtraCalls[0].updates, grokBillingExtraKey)
}

func TestGrokQuotaService_QueryQuotaFallsBackToActiveProbeAndPersists429Snapshot(t *testing.T) {
	t.Parallel()

	repo := &grokQuotaAccountRepoStub{account: newGrokQuotaOAuthAccountForTest(102)}
	resetAt := time.Now().UTC().Add(time.Hour).Unix()
	limitedResp := newGrokQuotaHTTPResponse(http.StatusTooManyRequests, `{"error":"limited"}`)
	limitedResp.Header.Set("x-ratelimit-limit-requests", "100")
	limitedResp.Header.Set("x-ratelimit-remaining-requests", "0")
	limitedResp.Header.Set("x-ratelimit-reset-requests", strconv.FormatInt(resetAt, 10))
	limitedResp.Header.Set("retry-after", "60")
	upstream := &grokQuotaHTTPUpstreamStub{responses: []*http.Response{
		newGrokQuotaHTTPResponse(http.StatusNotFound, `{"error":"no billing"}`),
		newGrokQuotaHTTPResponse(http.StatusNotFound, `{"error":"no billing"}`),
		limitedResp,
	}}
	svc := NewGrokQuotaService(repo, nil, NewGrokTokenProvider(repo, nil), upstream, nil)

	result, err := svc.QueryQuota(context.Background(), 102)

	require.NoError(t, err)
	require.Equal(t, "active_probe", result.Source)
	require.Equal(t, http.StatusTooManyRequests, result.StatusCode)
	require.True(t, result.HeadersObserved)
	require.Len(t, upstream.requests, 3)
	require.Equal(t, "/v1/responses", upstream.requests[2].URL.Path)
	require.Len(t, repo.updateExtraCalls, 1)
	snapshot, ok := repo.updateExtraCalls[0].updates[grokQuotaSnapshotExtraKey].(*xai.QuotaSnapshot)
	require.True(t, ok)
	require.Equal(t, http.StatusTooManyRequests, snapshot.StatusCode)
	require.NotNil(t, snapshot.Requests)
	require.EqualValues(t, 0, *snapshot.Requests.Remaining)
	require.NotNil(t, snapshot.RetryAfterSeconds)
	require.Equal(t, 60, *snapshot.RetryAfterSeconds)
}

func TestGrokQuotaService_QueryQuotaReturnsUpstreamAuthErrorAfterSnapshotPersist(t *testing.T) {
	t.Parallel()

	repo := &grokQuotaAccountRepoStub{account: newGrokQuotaOAuthAccountForTest(103)}
	unauthorizedResp := newGrokQuotaHTTPResponse(http.StatusUnauthorized, `{"error":"unauthorized"}`)
	unauthorizedResp.Header.Set("x-ratelimit-limit-requests", "100")
	upstream := &grokQuotaHTTPUpstreamStub{responses: []*http.Response{
		newGrokQuotaHTTPResponse(http.StatusNotFound, `{"error":"no billing"}`),
		newGrokQuotaHTTPResponse(http.StatusNotFound, `{"error":"no billing"}`),
		unauthorizedResp,
	}}
	svc := NewGrokQuotaService(repo, nil, NewGrokTokenProvider(repo, nil), upstream, nil)

	result, err := svc.QueryQuota(context.Background(), 103)

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusUnauthorized, infraerrors.Code(err))
	require.Len(t, repo.updateExtraCalls, 1)
	snapshot, ok := repo.updateExtraCalls[0].updates[grokQuotaSnapshotExtraKey].(*xai.QuotaSnapshot)
	require.True(t, ok)
	require.Equal(t, http.StatusUnauthorized, snapshot.StatusCode)
	require.True(t, snapshot.HeadersObserved)
}

func newGrokQuotaOAuthAccountForTest(id int64) *Account {
	return &Account{
		ID:          id,
		Platform:    PlatformGrok,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Concurrency: 2,
		Credentials: map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"expires_at":    time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		},
		Extra: map[string]any{
			"model_scope_v2": DefaultGrokBuildTextModelScope().ToMap(),
		},
	}
}
