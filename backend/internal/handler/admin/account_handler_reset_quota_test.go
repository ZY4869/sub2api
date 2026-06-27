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

func TestAccountHandlerResetQuotaUsesGenericLocalQuotaOnly(t *testing.T) {
	t.Parallel()

	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{{
		ID:        7301,
		Name:      "API Key Account",
		Platform:  service.PlatformAnthropic,
		Type:      service.AccountTypeAPIKey,
		Status:    service.StatusActive,
		CreatedAt: time.Now().UTC(),
		Extra: map[string]any{
			"quota_note": "local",
		},
	}}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/admin/accounts/:id/reset-quota", handler.ResetQuota)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/7301/reset-quota", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{7301}, adminSvc.resetAccountQuotaIDs)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID    int64          `json:"id"`
			Extra map[string]any `json:"extra"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(7301), resp.Data.ID)
	require.Equal(t, "local", resp.Data.Extra["quota_note"])
}

func TestAccountHandlerResetQuotaPropagatesGenericResetError(t *testing.T) {
	t.Parallel()

	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{{
		ID:       7302,
		Name:     "API Key Account",
		Platform: service.PlatformAnthropic,
		Type:     service.AccountTypeAPIKey,
		Status:   service.StatusActive,
	}}
	adminSvc.resetAccountQuotaErr = service.ErrAccountNotFound
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/admin/accounts/:id/reset-quota", handler.ResetQuota)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/7302/reset-quota", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, []int64{7302}, adminSvc.resetAccountQuotaIDs)
	require.Contains(t, rec.Body.String(), "ACCOUNT_NOT_FOUND")
}
