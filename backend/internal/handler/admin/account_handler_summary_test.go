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

func TestAccountHandlerGetStatusSummaryUsesSnakeCaseJSON(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accountSummary = &service.AccountStatusSummary{
		Total: 12,
		ByStatus: map[string]int64{
			"active":   8,
			"inactive": 2,
			"error":    2,
		},
		RateLimited:       3,
		TempUnschedulable: 1,
		Overloaded:        1,
		Paused:            4,
		InUse:             2,
		DispatchableCount: 7,
		ByPlatform: map[string]int64{
			"openai": 7,
			"kiro":   5,
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.GET("/api/v1/admin/accounts/summary", handler.GetStatusSummary)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/summary?group=ungrouped", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total              int64            `json:"total"`
			ByStatus           map[string]int64 `json:"by_status"`
			RateLimited        int64            `json:"rate_limited"`
			TempUnschedulable  int64            `json:"temp_unschedulable"`
			Overloaded         int64            `json:"overloaded"`
			Paused             int64            `json:"paused"`
			InUse              int64            `json:"in_use"`
			RemainingAvailable int64            `json:"remaining_available"`
			ByPlatform         map[string]int64 `json:"by_platform"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(12), resp.Data.Total)
	require.Equal(t, int64(8), resp.Data.ByStatus["active"])
	require.Equal(t, int64(2), resp.Data.ByStatus["inactive"])
	require.Equal(t, int64(2), resp.Data.ByStatus["error"])
	require.Equal(t, int64(3), resp.Data.RateLimited)
	require.Equal(t, int64(1), resp.Data.TempUnschedulable)
	require.Equal(t, int64(1), resp.Data.Overloaded)
	require.Equal(t, int64(4), resp.Data.Paused)
	require.Equal(t, int64(2), resp.Data.InUse)
	require.Equal(t, int64(5), resp.Data.RemainingAvailable)
	require.Equal(t, int64(7), resp.Data.ByPlatform["openai"])
	require.Equal(t, int64(5), resp.Data.ByPlatform["kiro"])
}

func TestAccountHandlerGetStatusSummaryFiltersPrivacyModeAndComputesRemainingAvailable(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:          1,
			Name:        "private-openai",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"privacy_mode": "private",
			},
		},
		{
			ID:          2,
			Name:        "public-openai",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.GET("/api/v1/admin/accounts/summary", handler.GetStatusSummary)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/summary?platform=openai&privacy_mode=private", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total              int64 `json:"total"`
			RemainingAvailable int64 `json:"remaining_available"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(1), resp.Data.Total)
	require.Equal(t, int64(1), resp.Data.RemainingAvailable)
}

func TestAccountHandlerGetStatusSummaryAvailableOnlyMatchesRemainingAvailable(t *testing.T) {
	adminSvc := newStubAdminService()
	future := time.Now().Add(10 * time.Minute)
	adminSvc.accounts = []service.Account{
		{
			ID:          1,
			Name:        "available",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
		},
		{
			ID:               2,
			Name:             "rate-limited",
			Platform:         service.PlatformOpenAI,
			Type:             service.AccountTypeAPIKey,
			Status:           service.StatusActive,
			Schedulable:      true,
			RateLimitResetAt: &future,
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.GET("/api/v1/admin/accounts/summary", handler.GetStatusSummary)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/summary?runtime_view=available_only", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Total              int64 `json:"total"`
			InUse              int64 `json:"in_use"`
			RemainingAvailable int64 `json:"remaining_available"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(1), resp.Data.Total)
	require.Equal(t, int64(0), resp.Data.InUse)
	require.Equal(t, int64(1), resp.Data.RemainingAvailable)
}
