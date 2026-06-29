package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageQueryAccountRepoStub struct {
	service.AccountRepository
	account          *service.Account
	updateExtraCalls []map[string]any
}

func (s *usageQueryAccountRepoStub) GetByID(_ context.Context, id int64) (*service.Account, error) {
	if s.account == nil || s.account.ID != id {
		return nil, service.ErrAccountNotFound
	}
	account := *s.account
	account.Credentials = cloneAnyMap(s.account.Credentials)
	account.Extra = cloneAnyMap(s.account.Extra)
	return &account, nil
}

func (s *usageQueryAccountRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if s.account == nil || s.account.ID != id {
		return service.ErrAccountNotFound
	}
	copied := cloneAnyMap(updates)
	s.updateExtraCalls = append(s.updateExtraCalls, copied)
	if s.account.Extra == nil {
		s.account.Extra = map[string]any{}
	}
	for key, value := range copied {
		s.account.Extra[key] = value
	}
	return nil
}

func (s *usageQueryAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	return nil
}

type usageQueryResetCreditReaderStub struct {
	snapshot *service.OpenAIResetCreditsSnapshot
	err      error
	calls    int
}

func (s *usageQueryResetCreditReaderStub) ReadResetCredits(context.Context, *service.Account) (*service.OpenAIResetCreditsSnapshot, error) {
	s.calls++
	return s.snapshot, s.err
}

type usageQueryLogRepoStub struct {
	service.UsageLogRepository
	stats     *usagestats.AccountStats
	breakdown map[int64]*usagestats.AccountTodayStatsBreakdown
}

func (s *usageQueryLogRepoStub) GetAccountWindowStats(_ context.Context, _ int64, _ time.Time) (*usagestats.AccountStats, error) {
	if s.stats != nil {
		return s.stats, nil
	}
	return &usagestats.AccountStats{}, nil
}

func (s *usageQueryLogRepoStub) GetAccountTodayStatsBreakdownBatch(_ context.Context, accountIDs []int64, _ time.Time, _ time.Time) (map[int64]*usagestats.AccountTodayStatsBreakdown, error) {
	result := make(map[int64]*usagestats.AccountTodayStatsBreakdown, len(accountIDs))
	for _, accountID := range accountIDs {
		result[accountID] = s.breakdown[accountID]
	}
	return result, nil
}

func cloneAnyMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func TestAccountHandlerGetUsageSupportsSourceQuery(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	windowStart := now.Add(-30 * time.Minute)
	windowEnd := now.Add(4 * time.Hour)
	passiveReset := now.Add(6 * 24 * time.Hour)
	account := &service.Account{
		ID:                 41,
		Name:               "Claude Setup",
		Platform:           service.PlatformAnthropic,
		Type:               service.AccountTypeSetupToken,
		Status:             service.StatusActive,
		SessionWindowStart: &windowStart,
		SessionWindowEnd:   &windowEnd,
		Extra: map[string]any{
			"session_window_utilization":   0.25,
			"passive_usage_sampled_at":     now.Format(time.RFC3339),
			"passive_usage_7d_utilization": 0.6,
			"passive_usage_7d_reset":       float64(passiveReset.Unix()),
		},
	}
	usageService := service.NewAccountUsageService(
		&usageQueryAccountRepoStub{account: account},
		&usageQueryLogRepoStub{},
		nil,
		nil,
		nil,
		service.NewUsageCache(),
		nil,
	)
	handler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, usageService, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id/usage", handler.GetUsage)

	type usageResponse struct {
		Code int `json:"code"`
		Data struct {
			Source   string `json:"source"`
			FiveHour *struct {
				Utilization float64 `json:"utilization"`
			} `json:"five_hour"`
			SevenDay *struct {
				Utilization float64 `json:"utilization"`
			} `json:"seven_day"`
		} `json:"data"`
	}

	t.Run("passive returns sampled 7d snapshot and ignores force", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/41/usage?source=passive&force=1", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp usageResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, "passive", resp.Data.Source)
		require.NotNil(t, resp.Data.FiveHour)
		require.NotNil(t, resp.Data.SevenDay)
		require.InDelta(t, 25, resp.Data.FiveHour.Utilization, 0.001)
		require.InDelta(t, 60, resp.Data.SevenDay.Utilization, 0.001)
	})

	t.Run("active keeps legacy behavior", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/41/usage?source=active", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp usageResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, "", resp.Data.Source)
		require.NotNil(t, resp.Data.FiveHour)
		require.Nil(t, resp.Data.SevenDay)
		require.InDelta(t, 25, resp.Data.FiveHour.Utilization, 0.001)
	})

	t.Run("empty source stays on legacy active branch", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/41/usage", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp usageResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(t, "", resp.Data.Source)
		require.Nil(t, resp.Data.SevenDay)
	})

	t.Run("active falls back to passive snapshot when anthropic oauth is missing access token", func(t *testing.T) {
		oauthWindowEnd := now.Add(4 * time.Hour)
		oauthAccount := &service.Account{
			ID:               42,
			Name:             "Claude OAuth Missing Token",
			Platform:         service.PlatformAnthropic,
			Type:             service.AccountTypeOAuth,
			Status:           service.StatusActive,
			SessionWindowEnd: &oauthWindowEnd,
			Extra: map[string]any{
				"session_window_utilization":   0.18,
				"passive_usage_sampled_at":     now.Format(time.RFC3339),
				"passive_usage_7d_utilization": 0.42,
				"passive_usage_7d_reset":       float64(passiveReset.Unix()),
			},
		}
		oauthUsageService := service.NewAccountUsageService(
			&usageQueryAccountRepoStub{account: oauthAccount},
			&usageQueryLogRepoStub{},
			nil,
			nil,
			nil,
			service.NewUsageCache(),
			nil,
		)
		oauthHandler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, oauthUsageService, nil, nil, nil, nil, nil, nil)

		oauthRouter := gin.New()
		oauthRouter.GET("/api/v1/admin/accounts/:id/usage", oauthHandler.GetUsage)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/42/usage?source=active", nil)
		oauthRouter.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp usageResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, "passive", resp.Data.Source)
		require.NotNil(t, resp.Data.FiveHour)
		require.NotNil(t, resp.Data.SevenDay)
		require.InDelta(t, 18, resp.Data.FiveHour.Utilization, 0.001)
		require.InDelta(t, 42, resp.Data.SevenDay.Utilization, 0.001)
	})

	t.Run("invalid source returns bad request", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/41/usage?source=bogus", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestAccountHandlerGetUsageReturnsOpenAIResetCreditStatus(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	account := &service.Account{
		ID:       43,
		Name:     "OpenAI OAuth",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
		Credentials: map[string]any{
			"access_token":       "token",
			"chatgpt_account_id": "acct",
		},
		Extra: map[string]any{
			"codex_usage_updated_at":                             now.Format(time.RFC3339),
			"codex_5h_used_percent":                              10,
			"codex_5h_reset_at":                                  now.Add(time.Hour).Format(time.RFC3339),
			"codex_7d_used_percent":                              20,
			"codex_7d_reset_at":                                  now.Add(24 * time.Hour).Format(time.RFC3339),
			"openai_quota_usage_updated_at":                      now.Format(time.RFC3339),
			"openai_rate_limit_reset_credits_status":             "unsupported",
			"openai_rate_limit_reset_credits_unsupported_reason": "OpenAI quota usage did not include reset credits",
		},
	}
	usageService := service.NewAccountUsageService(
		&usageQueryAccountRepoStub{account: account},
		&usageQueryLogRepoStub{},
		nil,
		nil,
		nil,
		service.NewUsageCache(),
		nil,
	)
	handler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, usageService, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id/usage", handler.GetUsage)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/43/usage?source=active", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			OpenAIResetCredits *struct {
				AvailableCount    *int   `json:"available_count"`
				Status            string `json:"status"`
				UnsupportedReason string `json:"unsupported_reason"`
			} `json:"openai_reset_credits"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NotNil(t, resp.Data.OpenAIResetCredits)
	require.Nil(t, resp.Data.OpenAIResetCredits.AvailableCount)
	require.Equal(t, "unsupported", resp.Data.OpenAIResetCredits.Status)
	require.Contains(t, resp.Data.OpenAIResetCredits.UnsupportedReason, "OpenAI quota")
}

func TestAccountHandlerGetUsageDoesNotUseOpenAIQuotaWindowsAfterResetCreditsRead(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	count := 2
	account := &service.Account{
		ID:       45,
		Name:     "OpenAI OAuth",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
		Credentials: map[string]any{
			"access_token":       "token",
			"chatgpt_account_id": "acct",
		},
		Extra: map[string]any{
			"codex_usage_updated_at": now.Add(-time.Minute).Format(time.RFC3339),
			"codex_5h_used_percent":  100,
			"codex_5h_reset_at":      now.Add(2 * time.Hour).Format(time.RFC3339),
			"codex_7d_used_percent":  100,
			"codex_7d_reset_at":      now.Add(48 * time.Hour).Format(time.RFC3339),
		},
	}
	repo := &usageQueryAccountRepoStub{account: account}
	reader := &usageQueryResetCreditReaderStub{
		snapshot: &service.OpenAIResetCreditsSnapshot{
			AvailableCount: &count,
			UpdatedAt:      now,
			Source:         "chatgpt_wham",
			Status:         "available",
			FiveHour: &service.OpenAIQuotaWindowSnapshot{
				Progress: &service.UsageProgress{
					Utilization:      0,
					ResetsAt:         ptrTime(now.Add(5 * time.Hour)),
					RemainingSeconds: 5 * 60 * 60,
				},
				LimitWindowSeconds: 5 * 60 * 60,
			},
			SevenDay: &service.OpenAIQuotaWindowSnapshot{
				Progress: &service.UsageProgress{
					Utilization:      4,
					ResetsAt:         ptrTime(now.Add(7 * 24 * time.Hour)),
					RemainingSeconds: 7 * 24 * 60 * 60,
				},
				LimitWindowSeconds: 7 * 24 * 60 * 60,
			},
		},
	}
	usageService := service.NewAccountUsageService(
		repo,
		&usageQueryLogRepoStub{},
		nil,
		nil,
		nil,
		service.NewUsageCache(),
		nil,
	)
	usageService.SetOpenAIResetCreditService(reader)
	handler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, usageService, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id/usage", handler.GetUsage)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/45/usage?source=active", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			FiveHour *struct {
				Utilization float64    `json:"utilization"`
				ResetsAt    *time.Time `json:"resets_at"`
			} `json:"five_hour"`
			SevenDay *struct {
				Utilization float64    `json:"utilization"`
				ResetsAt    *time.Time `json:"resets_at"`
			} `json:"seven_day"`
			OpenAIResetCredits *struct {
				AvailableCount *int `json:"available_count"`
			} `json:"openai_reset_credits"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 1, reader.calls)
	require.NotNil(t, resp.Data.FiveHour)
	require.NotNil(t, resp.Data.SevenDay)
	require.InDelta(t, 100, resp.Data.FiveHour.Utilization, 0.001)
	require.InDelta(t, 100, resp.Data.SevenDay.Utilization, 0.001)
	require.NotNil(t, resp.Data.FiveHour.ResetsAt)
	require.NotNil(t, resp.Data.SevenDay.ResetsAt)
	require.WithinDuration(t, now.Add(2*time.Hour), *resp.Data.FiveHour.ResetsAt, time.Second)
	require.WithinDuration(t, now.Add(48*time.Hour), *resp.Data.SevenDay.ResetsAt, time.Second)
	require.NotNil(t, resp.Data.OpenAIResetCredits)
	require.NotNil(t, resp.Data.OpenAIResetCredits.AvailableCount)
	require.Equal(t, 2, *resp.Data.OpenAIResetCredits.AvailableCount)
}

func TestAccountHandlerGetUsageResetCreditsReadFailureDoesNotReturnStaleCount(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	account := &service.Account{
		ID:       44,
		Name:     "OpenAI OAuth",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
		Credentials: map[string]any{
			"access_token":       "token",
			"chatgpt_account_id": "acct",
		},
		Extra: map[string]any{
			"codex_usage_updated_at":                             now.Format(time.RFC3339),
			"codex_5h_used_percent":                              10,
			"codex_5h_reset_at":                                  now.Add(time.Hour).Format(time.RFC3339),
			"codex_7d_used_percent":                              20,
			"codex_7d_reset_at":                                  now.Add(24 * time.Hour).Format(time.RFC3339),
			"openai_rate_limit_reset_credits_available_count":    3,
			"openai_rate_limit_reset_credits_updated_at":         now.Add(-11 * time.Minute).Format(time.RFC3339),
			"openai_quota_usage_updated_at":                      now.Add(-11 * time.Minute).Format(time.RFC3339),
			"openai_rate_limit_reset_credits_status":             "available",
			"openai_rate_limit_reset_credits_unsupported_reason": "stale",
		},
	}
	repo := &usageQueryAccountRepoStub{account: account}
	reader := &usageQueryResetCreditReaderStub{err: errors.New("OpenAI quota unavailable")}
	usageService := service.NewAccountUsageService(
		repo,
		&usageQueryLogRepoStub{},
		nil,
		nil,
		nil,
		service.NewUsageCache(),
		nil,
	)
	usageService.SetOpenAIResetCreditService(reader)
	handler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, usageService, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id/usage", handler.GetUsage)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/44/usage?source=active", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			OpenAIResetCredits *struct {
				AvailableCount *int   `json:"available_count"`
				Status         string `json:"status"`
			} `json:"openai_reset_credits"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 1, reader.calls)
	require.NotNil(t, resp.Data.OpenAIResetCredits)
	require.Nil(t, resp.Data.OpenAIResetCredits.AvailableCount)
	require.Equal(t, "unknown_or_unsupported", resp.Data.OpenAIResetCredits.Status)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Contains(t, repo.updateExtraCalls[0], "openai_rate_limit_reset_credits_available_count")
	require.Nil(t, repo.updateExtraCalls[0]["openai_rate_limit_reset_credits_available_count"])
}

func TestAccountHandlerGetBatchTodayStatsIncludesBreakdown(t *testing.T) {
	t.Parallel()

	breakdown := map[int64]*usagestats.AccountTodayStatsBreakdown{
		41: {
			Today:   usagestats.AccountStats{Requests: 1, Tokens: 30, Cost: 1.2, SuccessRate: 100, AverageDurationMs: 120},
			Weekly:  usagestats.AccountStats{Requests: 3, Tokens: 90, Cost: 3.6, SuccessRate: 66.666, AverageDurationMs: 180},
			Monthly: usagestats.AccountStats{Requests: 4, Tokens: 110, Cost: 4.2, SuccessRate: 75, AverageDurationMs: 190},
			Total:   usagestats.AccountStats{Requests: 5, Tokens: 140, Cost: 5.4, SuccessRate: 80, AverageDurationMs: 200},
		},
	}
	usageService := service.NewAccountUsageService(
		nil,
		&usageQueryLogRepoStub{breakdown: breakdown},
		nil,
		nil,
		nil,
		service.NewUsageCache(),
		nil,
	)
	handler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, usageService, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id/today-stats", handler.GetTodayStats)
	router.POST("/api/v1/admin/accounts/today-stats/batch", handler.GetBatchTodayStats)

	singleRec := httptest.NewRecorder()
	singleReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/41/today-stats", nil)
	router.ServeHTTP(singleRec, singleReq)

	require.Equal(t, http.StatusOK, singleRec.Code)
	var singleResp struct {
		Code int                  `json:"code"`
		Data *service.WindowStats `json:"data"`
	}
	require.NoError(t, json.Unmarshal(singleRec.Body.Bytes(), &singleResp))
	require.Equal(t, 0, singleResp.Code)
	require.NotNil(t, singleResp.Data)
	require.Equal(t, int64(1), singleResp.Data.Requests)
	require.NotNil(t, singleResp.Data.Weekly)
	require.Equal(t, int64(3), singleResp.Data.Weekly.Requests)
	require.NotNil(t, singleResp.Data.Monthly)
	require.Equal(t, int64(4), singleResp.Data.Monthly.Requests)
	require.NotNil(t, singleResp.Data.Total)
	require.Equal(t, int64(5), singleResp.Data.Total.Requests)

	body := bytes.NewBufferString(`{"account_ids":[41,42],"cycle_mode":"fixed"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/today-stats/batch", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Stats map[string]*service.WindowStats `json:"stats"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NotNil(t, resp.Data.Stats["41"])
	require.Equal(t, int64(1), resp.Data.Stats["41"].Requests)
	require.NotNil(t, resp.Data.Stats["41"].Weekly)
	require.Equal(t, int64(3), resp.Data.Stats["41"].Weekly.Requests)
	require.NotNil(t, resp.Data.Stats["41"].Monthly)
	require.Equal(t, int64(4), resp.Data.Stats["41"].Monthly.Requests)
	require.NotNil(t, resp.Data.Stats["41"].Total)
	require.Equal(t, int64(5), resp.Data.Stats["41"].Total.Requests)
	require.NotNil(t, resp.Data.Stats["42"])
	require.Equal(t, int64(0), resp.Data.Stats["42"].Requests)
	require.InEpsilon(t, 100, resp.Data.Stats["42"].SuccessRate, 0.0001)
	require.NotNil(t, resp.Data.Stats["42"].Monthly)
	require.InEpsilon(t, 100, resp.Data.Stats["42"].Monthly.SuccessRate, 0.0001)

	invalidBody := bytes.NewBufferString(`{"account_ids":[41],"cycle_mode":"bogus"}`)
	invalidRec := httptest.NewRecorder()
	invalidReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/today-stats/batch", invalidBody)
	invalidReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(invalidRec, invalidReq)
	require.Equal(t, http.StatusBadRequest, invalidRec.Code)
}

func TestAccountHandlerGetBatchTodayStatsFiltersUnavailableAccounts(t *testing.T) {
	t.Parallel()

	breakdown := map[int64]*usagestats.AccountTodayStatsBreakdown{
		41: {
			Today:   usagestats.AccountStats{Requests: 2, Tokens: 60, Cost: 2.4, SuccessRate: 100, AverageDurationMs: 130},
			Weekly:  usagestats.AccountStats{Requests: 4, Tokens: 120, Cost: 4.8, SuccessRate: 100, AverageDurationMs: 150},
			Monthly: usagestats.AccountStats{Requests: 5, Tokens: 150, Cost: 6.0, SuccessRate: 100, AverageDurationMs: 160},
			Total:   usagestats.AccountStats{Requests: 6, Tokens: 180, Cost: 7.2, SuccessRate: 100, AverageDurationMs: 170},
		},
		42: {
			Today: usagestats.AccountStats{Requests: 99, Tokens: 990, Cost: 99},
		},
	}
	usageRepo := &usageQueryLogRepoStub{breakdown: breakdown}
	usageService := service.NewAccountUsageService(
		nil,
		usageRepo,
		nil,
		nil,
		nil,
		service.NewUsageCache(),
		nil,
	)
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{{
		ID:        41,
		Name:      "visible",
		Platform:  service.PlatformAnthropic,
		Type:      service.AccountTypeOAuth,
		Status:    service.StatusActive,
		CreatedAt: time.Now().UTC(),
	}}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, usageService, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/admin/accounts/today-stats/batch", handler.GetBatchTodayStats)

	body := bytes.NewBufferString(`{"account_ids":[41,42,999,41],"cycle_mode":"calendar"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/today-stats/batch", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Stats map[string]*service.WindowStats `json:"stats"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Contains(t, resp.Data.Stats, "41")
	require.NotContains(t, resp.Data.Stats, "42")
	require.NotContains(t, resp.Data.Stats, "999")
	require.Equal(t, int64(2), resp.Data.Stats["41"].Requests)
}
