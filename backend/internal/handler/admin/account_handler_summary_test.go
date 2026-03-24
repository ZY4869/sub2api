package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
			Total             int64            `json:"total"`
			ByStatus          map[string]int64 `json:"by_status"`
			RateLimited       int64            `json:"rate_limited"`
			TempUnschedulable int64            `json:"temp_unschedulable"`
			Overloaded        int64            `json:"overloaded"`
			Paused            int64            `json:"paused"`
			ByPlatform        map[string]int64 `json:"by_platform"`
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
	require.Equal(t, int64(7), resp.Data.ByPlatform["openai"])
	require.Equal(t, int64(5), resp.Data.ByPlatform["kiro"])
}
