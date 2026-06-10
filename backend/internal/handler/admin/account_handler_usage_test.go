package admin

import (
	"bytes"
	"context"
	"encoding/json"
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
	account *service.Account
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
