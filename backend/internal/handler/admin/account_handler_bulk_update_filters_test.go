package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerBulkUpdate_FiltersModeResolvesTargets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{ID: 1, Name: "oa-1", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive},
		{ID: 2, Name: "oa-2", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive},
		{ID: 3, Name: "an-1", Platform: service.PlatformAnthropic, Type: service.AccountTypeOAuth, Status: service.StatusActive},
	}

	router := gin.New()
	accountHandler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/bulk-update", accountHandler.BulkUpdate)

	body, _ := json.Marshal(map[string]any{
		"filters": map[string]any{
			"platform": "openai",
		},
		"schedulable": false,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/bulk-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, adminSvc.lastBulkUpdateInput)
	require.ElementsMatch(t, []int64{1, 2}, adminSvc.lastBulkUpdateInput.AccountIDs)
}

func TestAccountHandlerBulkUpdate_FiltersModeRejectsEmptyResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{ID: 1, Name: "oa-1", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive},
	}

	router := gin.New()
	accountHandler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/bulk-update", accountHandler.BulkUpdate)

	body, _ := json.Marshal(map[string]any{
		"filters":     map[string]any{"platform": "anthropic"},
		"schedulable": false,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/bulk-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAccountHandlerBulkUpdate_FiltersModeRejectsMixedTargets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()

	router := gin.New()
	accountHandler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/bulk-update", accountHandler.BulkUpdate)

	body, _ := json.Marshal(map[string]any{
		"account_ids": []int64{1, 2},
		"filters":     map[string]any{"platform": "openai"},
		"schedulable": false,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/bulk-update", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
