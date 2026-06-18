package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerResetQuotaReturnsLatestOpenAIResetCreditsExtra(t *testing.T) {
	t.Parallel()

	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{{
		ID:        7301,
		Name:      "OpenAI OAuth",
		Platform:  service.PlatformOpenAI,
		Type:      service.AccountTypeOAuth,
		Status:    service.StatusActive,
		CreatedAt: time.Now().UTC(),
		Extra: map[string]any{
			"openai_rate_limit_reset_credits_available_count": float64(2),
			"openai_rate_limit_reset_credits_status":          "available",
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
	require.Equal(t, float64(2), resp.Data.Extra["openai_rate_limit_reset_credits_available_count"])
	require.Equal(t, "available", resp.Data.Extra["openai_rate_limit_reset_credits_status"])
}

func TestAccountHandlerResetQuotaReturnsStableOpenAIResetCreditConflictReason(t *testing.T) {
	t.Parallel()

	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{{
		ID:       7302,
		Name:     "OpenAI OAuth",
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
	}}
	adminSvc.resetAccountQuotaErr = infraerrors.New(
		http.StatusConflict,
		"OPENAI_RESET_CREDITS_NO_CREDIT",
		"没有可用的 OpenAI 真实重置次数",
	)
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/admin/accounts/:id/reset-quota", handler.ResetQuota)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/7302/reset-quota", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	require.Equal(t, []int64{7302}, adminSvc.resetAccountQuotaIDs)
	require.Contains(t, rec.Body.String(), "OPENAI_RESET_CREDITS_NO_CREDIT")
}
