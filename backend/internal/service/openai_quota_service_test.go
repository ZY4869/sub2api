package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

type openAIQuotaAccountRepoStub struct {
	AccountRepository
	account          *Account
	getErr           error
	updateExtraCalls []map[string]any
	resetQuotaCalls  int
}

func (r *openAIQuotaAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.account == nil || r.account.ID != id {
		return nil, ErrAccountNotFound
	}
	account := *r.account
	account.Credentials = cloneTestMap(r.account.Credentials)
	account.Extra = cloneTestMap(r.account.Extra)
	return &account, nil
}

func (r *openAIQuotaAccountRepoStub) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	r.updateExtraCalls = append(r.updateExtraCalls, cloneTestMap(updates))
	return nil
}

func (r *openAIQuotaAccountRepoStub) ResetQuotaUsed(context.Context, int64) error {
	r.resetQuotaCalls++
	return nil
}

func TestOpenAIQuotaService_QueryUsageUsesWhamUsageEndpoint(t *testing.T) {
	t.Parallel()

	var capturedAuth string
	var capturedAccountID string
	var capturedOriginator string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/backend-api/wham/usage", r.URL.Path)
		capturedAuth = r.Header.Get("Authorization")
		capturedAccountID = r.Header.Get("Chatgpt-Account-Id")
		capturedOriginator = r.Header.Get("Originator")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user_id":"user_1","account_id":"acct_chatgpt","rate_limit_reset_credits":{"available_count":3}}`))
	}))
	defer server.Close()

	svc, repo := newOpenAIQuotaServiceForTest(server.URL)

	usage, err := svc.QueryUsage(context.Background(), 9001)

	require.NoError(t, err)
	require.NotNil(t, usage)
	require.NotNil(t, usage.RateLimitResetCredits)
	require.Equal(t, 3, usage.RateLimitResetCredits.AvailableCount)
	require.NotZero(t, usage.FetchedAt)
	require.Equal(t, "Bearer access-token", capturedAuth)
	require.Equal(t, "acct_chatgpt", capturedAccountID)
	require.Equal(t, openAIQuotaOriginator, capturedOriginator)
	require.Zero(t, repo.resetQuotaCalls)
}

func TestOpenAIQuotaService_ResetCreditPostsRedeemRequestIDOnly(t *testing.T) {
	t.Parallel()

	var capturedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/backend-api/wham/rate-limit-reset-credits/consume", r.URL.Path)
		require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))
		require.Equal(t, "acct_chatgpt", r.Header.Get("Chatgpt-Account-Id"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"success","windows_reset":2,"credit":{"id":"credit_1","status":"redeemed"}}`))
	}))
	defer server.Close()

	svc, repo := newOpenAIQuotaServiceForTest(server.URL)

	result, err := svc.ResetCredit(context.Background(), 9001)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "success", result.Code)
	require.Equal(t, 2, result.WindowsReset)
	require.NotNil(t, result.Credit)
	require.Equal(t, "credit_1", result.Credit.ID)
	require.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, capturedBody["redeem_request_id"])
	require.Len(t, capturedBody, 1)
	require.Zero(t, repo.resetQuotaCalls)
}

func TestOpenAIQuotaService_ReadResetCreditsPersistsWhamSnapshot(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/backend-api/wham/usage", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"rate_limit_reset_credits":{"available_count":5}}`))
	}))
	defer server.Close()

	svc, repo := newOpenAIQuotaServiceForTest(server.URL)

	snapshot, err := svc.ReadResetCredits(context.Background(), repo.account)

	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.NotNil(t, snapshot.AvailableCount)
	require.Equal(t, 5, *snapshot.AvailableCount)
	require.Equal(t, openAIResetCreditsSourceWham, snapshot.Source)
	require.Equal(t, openAIResetCreditsStatusAvailable, snapshot.Status)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, 5, repo.updateExtraCalls[0][openAIResetCreditsAvailableCountExtraKey])
	require.Equal(t, openAIResetCreditsStatusAvailable, repo.updateExtraCalls[0][openAIResetCreditsStatusExtraKey])
	require.NotEmpty(t, repo.updateExtraCalls[0][openAIQuotaUsageUpdatedAtExtraKey])
}

func TestOpenAIQuotaService_ReadResetCreditsPersistsWhamWindows(t *testing.T) {
	t.Parallel()

	reset5h := time.Now().Add(5 * time.Hour).UTC().Unix()
	reset7d := time.Now().Add(7 * 24 * time.Hour).UTC().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/backend-api/wham/usage", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{
			"rate_limit": {
				"primary_window": {
					"used_percent": 0,
					"limit_window_seconds": 18000,
					"reset_after_seconds": 18000,
					"reset_at": %d
				},
				"secondary_window": {
					"used_percent": 4,
					"limit_window_seconds": 604800,
					"reset_after_seconds": 604800,
					"reset_at": %d
				}
			},
			"rate_limit_reset_credits": {"available_count": 2}
		}`, reset5h, reset7d)))
	}))
	defer server.Close()

	svc, repo := newOpenAIQuotaServiceForTest(server.URL)

	snapshot, err := svc.ReadResetCredits(context.Background(), repo.account)

	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.NotNil(t, snapshot.FiveHour)
	require.NotNil(t, snapshot.SevenDay)
	require.NotNil(t, snapshot.FiveHour.Progress)
	require.NotNil(t, snapshot.SevenDay.Progress)
	require.InDelta(t, 0, snapshot.FiveHour.Progress.Utilization, 0.001)
	require.InDelta(t, 4, snapshot.SevenDay.Progress.Utilization, 0.001)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, 0.0, repo.updateExtraCalls[0]["codex_5h_used_percent"])
	require.Equal(t, 4.0, repo.updateExtraCalls[0]["codex_7d_used_percent"])
	require.Equal(t, 300, repo.updateExtraCalls[0]["codex_primary_window_minutes"])
	require.Equal(t, 10080, repo.updateExtraCalls[0]["codex_secondary_window_minutes"])
}

func TestOpenAIQuotaService_UpstreamStatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		upstream   int
		wantStatus int
	}{
		{name: "unauthorized", upstream: http.StatusUnauthorized, wantStatus: http.StatusUnauthorized},
		{name: "forbidden", upstream: http.StatusForbidden, wantStatus: http.StatusForbidden},
		{name: "rate limited", upstream: http.StatusTooManyRequests, wantStatus: http.StatusTooManyRequests},
		{name: "server error", upstream: http.StatusInternalServerError, wantStatus: http.StatusBadGateway},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.upstream)
				_, _ = w.Write([]byte(`{"error":"upstream"}`))
			}))
			defer server.Close()

			svc, _ := newOpenAIQuotaServiceForTest(server.URL)

			_, err := svc.QueryUsage(context.Background(), 9001)

			require.Error(t, err)
			require.Equal(t, tt.wantStatus, infraerrors.Code(err))
		})
	}
}

func TestOpenAIQuotaService_RejectsNonOpenAIOAuthAccount(t *testing.T) {
	t.Parallel()

	svc, _ := newOpenAIQuotaServiceWithAccount(&Account{
		ID:       9002,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
	}, "http://127.0.0.1")

	_, err := svc.QueryUsage(context.Background(), 9002)

	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, infraerrors.Code(err))
}

func newOpenAIQuotaServiceForTest(baseURL string) (*OpenAIQuotaService, *openAIQuotaAccountRepoStub) {
	return newOpenAIQuotaServiceWithAccount(&Account{
		ID:       9001,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "access-token",
			"chatgpt_account_id": "acct_chatgpt",
			"expires_at":         time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		},
		Extra: map[string]any{},
	}, baseURL)
}

func newOpenAIQuotaServiceWithAccount(account *Account, baseURL string) (*OpenAIQuotaService, *openAIQuotaAccountRepoStub) {
	repo := &openAIQuotaAccountRepoStub{account: account}
	svc := NewOpenAIQuotaService(repo, nil, NewOpenAITokenProvider(nil, nil, nil), func(string) (*req.Client, error) {
		return req.C(), nil
	})
	svc.usageURL = baseURL + "/backend-api/wham/usage"
	svc.resetURL = baseURL + "/backend-api/wham/rate-limit-reset-credits/consume"
	return svc, repo
}

func cloneTestMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
