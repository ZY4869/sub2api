package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubAccountTestService struct {
	lastInput service.ScheduledTestExecutionInput
	result    *service.ScheduledTestResult
	err       error
}

func (s *stubAccountTestService) SetModelRegistryService(_ *service.ModelRegistryService) {}

func (s *stubAccountTestService) TestAccountConnection(_ *gin.Context, _ int64, _ string, _ string, _ string, _ string) error {
	panic("unexpected TestAccountConnection call")
}

func (s *stubAccountTestService) RunTestBackground(_ context.Context, input service.ScheduledTestExecutionInput) (*service.ScheduledTestResult, error) {
	s.lastInput = input
	if s.result == nil {
		return nil, s.err
	}
	clone := *s.result
	return &clone, s.err
}

func decodeBlacklistRetestModelsResponse(t *testing.T, rec *httptest.ResponseRecorder) []service.AvailableTestModel {
	t.Helper()
	var payload struct {
		Data []service.AvailableTestModel `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	return payload.Data
}

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

func TestAccountHandlerRetestBlacklistedPassesCatalogModel(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 81, Name: "blacklisted-81", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, LifecycleState: service.AccountLifecycleBlacklisted},
	}
	accountTestSvc := &stubAccountTestService{
		result: &service.ScheduledTestResult{Status: "success", ResponseText: "ok", LatencyMs: 123},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, accountTestSvc, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/blacklist/retest", handler.RetestBlacklisted)

	body := []byte(`{"account_ids":[81],"model_id":"gpt-5.4","model_input_mode":"catalog","source_protocol":"openai"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/blacklist/retest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(81), accountTestSvc.lastInput.AccountID)
	require.Equal(t, "gpt-5.4", accountTestSvc.lastInput.ModelID)
	require.Equal(t, "openai", accountTestSvc.lastInput.SourceProtocol)
	require.Contains(t, rec.Body.String(), `"restored":true`)
}

func TestAccountHandlerRetestBlacklistedPrefersManualModelID(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 82, Name: "blacklisted-82", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, LifecycleState: service.AccountLifecycleBlacklisted},
	}
	accountTestSvc := &stubAccountTestService{
		result: &service.ScheduledTestResult{Status: "success", ResponseText: "ok"},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, accountTestSvc, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/blacklist/retest", handler.RetestBlacklisted)

	body := []byte(`{"account_ids":[82],"model_id":"ignored-model","model_input_mode":"manual","manual_model_id":"claude-sonnet-4-5","source_protocol":"anthropic"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/blacklist/retest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "claude-sonnet-4-5", accountTestSvc.lastInput.ModelID)
	require.Equal(t, "anthropic", accountTestSvc.lastInput.SourceProtocol)
}

func TestAccountHandlerRetestBlacklistedFallsBackWhenModelOmitted(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 83, Name: "blacklisted-83", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, LifecycleState: service.AccountLifecycleBlacklisted},
	}
	accountTestSvc := &stubAccountTestService{
		result: &service.ScheduledTestResult{Status: "success"},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, accountTestSvc, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/blacklist/retest", handler.RetestBlacklisted)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/blacklist/retest", bytes.NewReader([]byte(`{"account_ids":[83]}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "", accountTestSvc.lastInput.ModelID)
	require.Equal(t, "", accountTestSvc.lastInput.SourceProtocol)
}

func TestAccountHandlerRetestBlacklistedModelsDedupesSamePlatform(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 91, Name: "openai-91", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey},
		{ID: 92, Name: "openai-92", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/blacklist/retest-models", handler.RetestBlacklistedModels)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/blacklist/retest-models", bytes.NewReader([]byte(`{"account_ids":[91,92]}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	expected := service.MergeAvailableTestModels(
		service.BuildAvailableTestModels(context.Background(), &adminSvc.accounts[0], nil),
		service.BuildAvailableTestModels(context.Background(), &adminSvc.accounts[1], nil),
	)
	require.Equal(t, expected, decodeBlacklistRetestModelsResponse(t, rec))
}

func TestAccountHandlerRetestBlacklistedModelsSupportsMixedPlatforms(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 93, Name: "openai-93", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey},
		{ID: 94, Name: "gemini-94", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/blacklist/retest-models", handler.RetestBlacklistedModels)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/blacklist/retest-models", bytes.NewReader([]byte(`{"account_ids":[93,94]}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	expected := service.MergeAvailableTestModels(
		service.BuildAvailableTestModels(context.Background(), &adminSvc.accounts[0], nil),
		service.BuildAvailableTestModels(context.Background(), &adminSvc.accounts[1], nil),
	)
	require.Equal(t, expected, decodeBlacklistRetestModelsResponse(t, rec))
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
