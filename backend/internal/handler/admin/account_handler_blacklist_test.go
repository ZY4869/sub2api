package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerBlacklistHTTP(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 51, Name: "openai-51", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/:id/blacklist", handler.Blacklist)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/51/blacklist", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(51), adminSvc.lastBlacklistedID)
	require.Contains(t, rec.Body.String(), "\"lifecycle_state\":\"blacklisted\"")
	require.Contains(t, rec.Body.String(), "\"lifecycle_reason_code\":\"manual_blacklist\"")
}

func TestAccountHandlerBlacklistPersistsFeedbackPayload(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 52, Name: "openai-52", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/:id/blacklist", handler.Blacklist)

	body := []byte(`{"source":"test_modal","feedback":{"fingerprint":"fp-123","advice_decision":"recommend_blacklist","action":"blacklist","platform":"openai","status_code":401,"error_code":"invalid_api_key","message_keywords":["invalid","key"]}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/52/blacklist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, adminSvc.lastBlacklistInput)
	require.Equal(t, "test_modal", adminSvc.lastBlacklistInput.Source)
	require.NotNil(t, adminSvc.lastBlacklistInput.Feedback)
	require.Equal(t, "fp-123", adminSvc.lastBlacklistInput.Feedback.Fingerprint)
	require.Equal(t, "recommend_blacklist", adminSvc.lastBlacklistInput.Feedback.AdviceDecision)
	require.Equal(t, []string{"invalid", "key"}, adminSvc.lastBlacklistInput.Feedback.MessageKeywords)
}

func TestAccountHandlerBlacklistReturnsNotFound(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/:id/blacklist", handler.Blacklist)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/999/blacklist", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}
