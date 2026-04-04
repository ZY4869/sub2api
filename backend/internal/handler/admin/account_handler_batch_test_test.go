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

func decodeBatchTestModelsResponse(t *testing.T, rec *httptest.ResponseRecorder) []service.AvailableTestModel {
	t.Helper()
	var payload struct {
		Data []service.AvailableTestModel `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	return payload.Data
}

func TestAccountHandlerGetBatchTestModelsReturnsIntersection(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 101, Name: "openai-101", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive},
		{ID: 102, Name: "gemini-102", Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey, Status: service.StatusActive},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/batch-test-models", handler.GetBatchTestModels)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-test-models", bytes.NewReader([]byte(`{"account_ids":[101,102]}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	expected := service.IntersectAvailableTestModels(
		service.BuildAvailableTestModels(context.Background(), &adminSvc.accounts[0], nil),
		service.BuildAvailableTestModels(context.Background(), &adminSvc.accounts[1], nil),
	)
	require.Equal(t, expected, decodeBatchTestModelsResponse(t, rec))
}

func TestAccountHandlerBatchTestAutoUsesFirstAvailableModel(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 111, Name: "openai-111", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive},
	}
	accountTestSvc := &stubAccountTestService{
		detailed: &service.BackgroundAccountTestResult{
			Status:       "success",
			ResponseText: "ok",
			LatencyMs:    88,
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, accountTestSvc, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/batch-test", handler.BatchTest)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-test", bytes.NewReader([]byte(`{"account_ids":[111],"model_input_mode":"auto","test_mode":"health_check"}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	expectedModels := service.BuildAvailableTestModels(context.Background(), &adminSvc.accounts[0], nil)
	require.NotEmpty(t, expectedModels)
	require.Equal(t, int64(111), accountTestSvc.lastInput.AccountID)
	require.Equal(t, expectedModels[0].ID, accountTestSvc.lastInput.ModelID)
	require.Equal(t, expectedModels[0].SourceProtocol, accountTestSvc.lastInput.SourceProtocol)
	require.Equal(t, string(service.AccountTestModeHealthCheck), accountTestSvc.lastInput.TestMode)
	require.Contains(t, rec.Body.String(), `"status":"success"`)
	require.Contains(t, rec.Body.String(), `"resolved_model_id":"`+expectedModels[0].ID+`"`)
}

func TestAccountHandlerBatchTestPassesPromptAndReturnsBlacklistState(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 121, Name: "anthropic-121", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive},
	}
	accountTestSvc := &stubAccountTestService{
		detailed: &service.BackgroundAccountTestResult{
			Status:                  "failed",
			ErrorMessage:            "Unauthorized",
			ResponseText:            "Unauthorized",
			LatencyMs:               144,
			ResolvedModelID:         "claude-sonnet-4-5",
			ResolvedPlatform:        service.PlatformAnthropic,
			ResolvedSourceProtocol:  service.PlatformAnthropic,
			BlacklistAdviceDecision: string(service.BlacklistAdviceAutoBlacklisted),
			CurrentLifecycleState:   service.AccountLifecycleBlacklisted,
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, accountTestSvc, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/batch-test", handler.BatchTest)

	body := []byte(`{"account_ids":[121],"model_input_mode":"manual","manual_model_id":"claude-sonnet-4-5","source_protocol":"anthropic","prompt":"hello","test_mode":"real_forward"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "claude-sonnet-4-5", accountTestSvc.lastInput.ModelID)
	require.Equal(t, "anthropic", accountTestSvc.lastInput.SourceProtocol)
	require.Equal(t, "hello", accountTestSvc.lastInput.Prompt)
	require.Equal(t, string(service.AccountTestModeRealForward), accountTestSvc.lastInput.TestMode)
	require.Contains(t, rec.Body.String(), `"blacklist_advice_decision":"auto_blacklisted"`)
	require.Contains(t, rec.Body.String(), `"current_lifecycle_state":"blacklisted"`)
}
