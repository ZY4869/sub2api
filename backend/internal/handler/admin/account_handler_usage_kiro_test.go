package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerGetUsageRejectsKiroActiveQuery(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	windowStart := now.Add(-30 * time.Minute)
	windowEnd := now.Add(4 * time.Hour)
	passiveReset := now.Add(6 * 24 * time.Hour)
	account := &service.Account{
		ID:                 42,
		Name:               "Kiro OAuth",
		Platform:           service.PlatformKiro,
		Type:               service.AccountTypeOAuth,
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

	t.Run("active returns bad request instead of upstream error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/42/usage?source=active", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		var resp struct {
			Reason  string `json:"reason"`
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(t, "ACCOUNT_USAGE_UNSUPPORTED", resp.Reason)
		require.Contains(t, resp.Message, "kiro oauth accounts do not support active usage query")
	})

	t.Run("passive still returns sampled usage", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/42/usage?source=passive", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
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
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, "passive", resp.Data.Source)
		require.NotNil(t, resp.Data.FiveHour)
		require.NotNil(t, resp.Data.SevenDay)
	})
}
