package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandler_Create_SchedulesModelProbeRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminSvc := newStubAdminService()
	handler := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	handler.SetAccountModelImportService(service.NewAccountModelImportService(nil, nil, nil, nil))

	router := gin.New()
	router.POST("/api/v1/admin/accounts", handler.Create)

	body := map[string]any{
		"name":     "kiro-create",
		"platform": service.PlatformKiro,
		"type":     service.AccountTypeOAuth,
		"credentials": map[string]any{
			"access_token": "kiro-token",
		},
		"extra": map[string]any{
			"model_scope_v2": buildProbeRefreshScope("claude-sonnet-4.5"),
		},
		"concurrency": 1,
		"priority":    1,
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Eventually(t, func() bool {
		update := latestStubUpdatedAccount(adminSvc)
		return update != nil && hasPolicyUpdateSnapshot(update)
	}, time.Second, 20*time.Millisecond)
}

func TestAccountHandler_Update_SchedulesModelProbeRefreshWhenPolicyChanges(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:          42,
			Name:        "kiro-update",
			Platform:    service.PlatformKiro,
			Type:        service.AccountTypeOAuth,
			Status:      service.StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"access_token": "kiro-token",
			},
			Extra: map[string]any{
				"model_scope_v2": buildProbeRefreshScope("claude-opus-4.1"),
			},
		},
	}

	handler := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	handler.SetAccountModelImportService(service.NewAccountModelImportService(nil, nil, nil, nil))

	router := gin.New()
	router.PATCH("/api/v1/admin/accounts/:id", handler.Update)

	body := map[string]any{
		"extra": map[string]any{
			"model_scope_v2": buildProbeRefreshScope("claude-sonnet-4.5"),
		},
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/accounts/42", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Eventually(t, func() bool {
		adminSvc.mu.Lock()
		defer adminSvc.mu.Unlock()
		if len(adminSvc.updatedAccounts) < 2 {
			return false
		}
		return hasPolicyUpdateSnapshot(adminSvc.updatedAccounts[len(adminSvc.updatedAccounts)-1])
	}, time.Second, 20*time.Millisecond)
}

func buildProbeRefreshScope(modelID string) map[string]any {
	return map[string]any{
		"policy_mode": service.AccountModelPolicyModeWhitelist,
		"entries": []map[string]any{
			{
				"display_model_id": modelID,
				"target_model_id":  modelID,
				"provider":         service.PlatformKiro,
				"visibility_mode":  service.AccountModelVisibilityModeDirect,
			},
		},
	}
}

func latestStubUpdatedAccount(adminSvc *stubAdminService) *service.UpdateAccountInput {
	adminSvc.mu.Lock()
	defer adminSvc.mu.Unlock()
	if len(adminSvc.updatedAccounts) == 0 {
		return nil
	}
	return adminSvc.updatedAccounts[len(adminSvc.updatedAccounts)-1]
}

func hasPolicyUpdateSnapshot(update *service.UpdateAccountInput) bool {
	if update == nil || update.Extra == nil {
		return false
	}
	snapshot, ok := update.Extra["model_probe_snapshot"].(map[string]any)
	if !ok {
		return false
	}
	return snapshot["source"] == service.AccountModelProbeSnapshotSourcePolicyUpdate
}
