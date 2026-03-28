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

func TestAccountHandlerBatchDeleteBlacklistedByIDs(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{ID: 61, Name: "blacklisted-61", LifecycleState: service.AccountLifecycleBlacklisted},
		{ID: 62, Name: "active-62", LifecycleState: service.AccountLifecycleNormal},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/blacklist/batch-delete", handler.BatchDeleteBlacklisted)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/blacklist/batch-delete", bytes.NewReader([]byte(`{"ids":[61,62,999]}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{61, 62, 999}, adminSvc.lastBatchDeleteInput.ids)
	require.False(t, adminSvc.lastBatchDeleteInput.deleteAll)
	require.Contains(t, rec.Body.String(), "\"deleted_ids\":[61]")
	require.Contains(t, rec.Body.String(), "\"failed_count\":2")
}

func TestAccountHandlerBatchDeleteBlacklistedDeleteAll(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{ID: 71, Name: "blacklisted-71", LifecycleState: service.AccountLifecycleBlacklisted},
		{ID: 72, Name: "blacklisted-72", LifecycleState: service.AccountLifecycleBlacklisted},
		{ID: 73, Name: "active-73", LifecycleState: service.AccountLifecycleNormal},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/blacklist/batch-delete", handler.BatchDeleteBlacklisted)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/blacklist/batch-delete", bytes.NewReader([]byte(`{"delete_all":true}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, adminSvc.lastBatchDeleteInput.ids)
	require.True(t, adminSvc.lastBatchDeleteInput.deleteAll)
	require.Contains(t, rec.Body.String(), "\"deleted_count\":2")
}

func TestAccountHandlerBatchDeleteBlacklistedRequiresMode(t *testing.T) {
	adminSvc := newStubAdminService()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/blacklist/batch-delete", handler.BatchDeleteBlacklisted)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/blacklist/batch-delete", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAccountHandlerBatchDeleteBlacklistedRejectsMixedMode(t *testing.T) {
	adminSvc := newStubAdminService()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/blacklist/batch-delete", handler.BatchDeleteBlacklisted)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/blacklist/batch-delete", bytes.NewReader([]byte(`{"ids":[88],"delete_all":true}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
