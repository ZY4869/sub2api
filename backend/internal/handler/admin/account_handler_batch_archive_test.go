package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExecuteBatchArchiveAccountsCreatesArchiveGroupWithoutDisablingAccounts(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 11, Name: "kiro-1", Platform: service.PlatformKiro, Status: service.StatusActive},
		{ID: 12, Name: "kiro-2", Platform: service.PlatformKiro, Status: service.StatusActive},
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := handler.executeBatchArchiveAccounts(context.Background(), &BatchArchiveAccountsRequest{
		AccountIDs: []int64{11, 12},
		GroupName:  "Kiro Archive",
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.ArchivedCount)
	require.Equal(t, 0, result.FailedCount)
	require.Equal(t, "Kiro Archive", result.ArchiveGroupName)
	require.NotZero(t, result.ArchiveGroupID)
	require.NotNil(t, adminSvc.lastBulkUpdateInput)
	require.Equal(t, []int64{11, 12}, adminSvc.lastBulkUpdateInput.AccountIDs)
	require.Empty(t, adminSvc.lastBulkUpdateInput.Status)
	require.Equal(t, service.AccountLifecycleArchived, adminSvc.lastBulkUpdateInput.LifecycleState)
	require.NotNil(t, adminSvc.lastBulkUpdateInput.GroupIDs)
	require.Equal(t, []int64{result.ArchiveGroupID}, *adminSvc.lastBulkUpdateInput.GroupIDs)
}

func TestExecuteBatchArchiveAccountsRejectsMixedPlatforms(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 21, Name: "openai-1", Platform: service.PlatformOpenAI, Status: service.StatusActive},
		{ID: 22, Name: "kiro-1", Platform: service.PlatformKiro, Status: service.StatusActive},
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := handler.executeBatchArchiveAccounts(context.Background(), &BatchArchiveAccountsRequest{
		AccountIDs: []int64{21, 22},
		GroupName:  "Shared Archive",
	})
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, "ACCOUNT_BATCH_ARCHIVE_MIXED_PLATFORM", infraerrors.Reason(err))
}

func TestExecuteBatchArchiveAccountsRejectsArchiveGroupPlatformConflict(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 31, Name: "openai-1", Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}
	adminSvc.groups = append(adminSvc.groups, service.Group{
		ID:       88,
		Name:     "Shared Archive",
		Platform: service.PlatformAnthropic,
		Status:   service.StatusActive,
	})
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := handler.executeBatchArchiveAccounts(context.Background(), &BatchArchiveAccountsRequest{
		AccountIDs: []int64{31},
		GroupName:  "Shared Archive",
	})
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, "ACCOUNT_BATCH_CREATE_ARCHIVE_GROUP_PLATFORM_CONFLICT", infraerrors.Reason(err))
}

func TestAccountHandlerBatchArchiveHTTP(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.strictAccountLookup = true
	adminSvc.accounts = []service.Account{
		{ID: 41, Name: "openai-1", Platform: service.PlatformOpenAI, Status: service.StatusActive},
		{ID: 42, Name: "openai-2", Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/batch-archive", handler.BatchArchiveAccounts)

	body, err := json.Marshal(map[string]any{
		"account_ids": []int64{41, 42},
		"group_name":  "OpenAI Archive",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-archive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int                        `json:"code"`
		Data BatchArchiveAccountsResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 2, resp.Data.ArchivedCount)
	require.Equal(t, 0, resp.Data.FailedCount)
	require.Equal(t, "OpenAI Archive", resp.Data.ArchiveGroupName)
}
