package admin

import (
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
	stats *usagestats.AccountStats
}

func (s *usageQueryLogRepoStub) GetAccountWindowStats(_ context.Context, _ int64, _ time.Time) (*usagestats.AccountStats, error) {
	if s.stats != nil {
		return s.stats, nil
	}
	return &usagestats.AccountStats{}, nil
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

	t.Run("invalid source returns bad request", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/41/usage?source=bogus", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
