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

func TestExecuteArchiveGroupAccountsReusesArchiveGroupWithoutDisablingAccounts(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.groups = []service.Group{
		{ID: 10, Name: "OpenAI Prod", Platform: service.PlatformOpenAI, Status: service.StatusActive},
		{ID: 11, Name: "OpenAI Archive", Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}
	adminSvc.accounts = []service.Account{
		{ID: 41, Name: "openai-1", Platform: service.PlatformOpenAI, Status: service.StatusActive, GroupIDs: []int64{10}},
		{ID: 42, Name: "openai-2", Platform: service.PlatformOpenAI, Status: service.StatusActive, GroupIDs: []int64{10}},
		{ID: 43, Name: "openai-other", Platform: service.PlatformOpenAI, Status: service.StatusActive, GroupIDs: []int64{999}},
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := handler.executeArchiveGroupAccounts(context.Background(), &ArchiveGroupAccountsRequest{
		SourceGroupID: 10,
		GroupName:     "OpenAI Archive",
	})
	require.NoError(t, err)
	require.Equal(t, int64(10), result.SourceGroupID)
	require.Equal(t, "OpenAI Prod", result.SourceGroupName)
	require.Equal(t, 2, result.ArchivedCount)
	require.Equal(t, 0, result.FailedCount)
	require.Equal(t, int64(11), result.ArchiveGroupID)
	require.Equal(t, "OpenAI Archive", result.ArchiveGroupName)
	require.Equal(t, []int64{41, 42}, result.ArchivedAccountIDs)
	require.NotNil(t, adminSvc.lastBulkUpdateInput)
	require.Equal(t, []int64{41, 42}, adminSvc.lastBulkUpdateInput.AccountIDs)
	require.Empty(t, adminSvc.lastBulkUpdateInput.Status)
	require.Equal(t, service.AccountLifecycleArchived, adminSvc.lastBulkUpdateInput.LifecycleState)
	require.NotNil(t, adminSvc.lastBulkUpdateInput.GroupIDs)
	require.Equal(t, []int64{11}, *adminSvc.lastBulkUpdateInput.GroupIDs)
}

func TestExecuteArchiveGroupAccountsCreatesArchiveGroupWhenMissing(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.groups = []service.Group{
		{ID: 20, Name: "Kiro Prod", Platform: service.PlatformKiro, Status: service.StatusActive},
	}
	adminSvc.accounts = []service.Account{
		{ID: 51, Name: "kiro-1", Platform: service.PlatformKiro, Status: service.StatusActive, GroupIDs: []int64{20}},
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := handler.executeArchiveGroupAccounts(context.Background(), &ArchiveGroupAccountsRequest{
		SourceGroupID: 20,
		GroupName:     "Kiro Archive",
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.ArchivedCount)
	require.NotZero(t, result.ArchiveGroupID)
	require.Equal(t, "Kiro Archive", result.ArchiveGroupName)
	require.Len(t, adminSvc.groups, 2)
}

func TestExecuteArchiveGroupAccountsRejectsEmptySourceGroup(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.groups = []service.Group{
		{ID: 30, Name: "Gemini Empty", Platform: service.PlatformGemini, Status: service.StatusActive},
	}
	adminSvc.accounts = []service.Account{
		{ID: 61, Name: "gemini-1", Platform: service.PlatformGemini, Status: service.StatusActive, GroupIDs: []int64{99}},
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := handler.executeArchiveGroupAccounts(context.Background(), &ArchiveGroupAccountsRequest{
		SourceGroupID: 30,
		GroupName:     "Gemini Archive",
	})
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, "ACCOUNT_GROUP_ARCHIVE_EMPTY", infraerrors.Reason(err))
}

func TestExecuteArchiveGroupAccountsRejectsArchiveGroupPlatformConflict(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.groups = []service.Group{
		{ID: 40, Name: "OpenAI Prod", Platform: service.PlatformOpenAI, Status: service.StatusActive},
		{ID: 41, Name: "Shared Archive", Platform: service.PlatformAnthropic, Status: service.StatusActive},
	}
	adminSvc.accounts = []service.Account{
		{ID: 71, Name: "openai-1", Platform: service.PlatformOpenAI, Status: service.StatusActive, GroupIDs: []int64{40}},
	}
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := handler.executeArchiveGroupAccounts(context.Background(), &ArchiveGroupAccountsRequest{
		SourceGroupID: 40,
		GroupName:     "Shared Archive",
	})
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, "ACCOUNT_BATCH_CREATE_ARCHIVE_GROUP_PLATFORM_CONFLICT", infraerrors.Reason(err))
}

func TestAccountHandlerArchiveGroupHTTP(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.groups = []service.Group{
		{ID: 50, Name: "OpenAI Prod", Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}
	adminSvc.accounts = []service.Account{
		{ID: 81, Name: "openai-1", Platform: service.PlatformOpenAI, Status: service.StatusActive, GroupIDs: []int64{50}},
		{ID: 82, Name: "openai-2", Platform: service.PlatformOpenAI, Status: service.StatusActive, GroupIDs: []int64{50}},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/archive-group", handler.ArchiveGroupAccounts)

	body, err := json.Marshal(map[string]any{
		"source_group_id": int64(50),
		"group_name":      "OpenAI Archive",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/archive-group", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int                        `json:"code"`
		Data ArchiveGroupAccountsResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 2, resp.Data.ArchivedCount)
	require.Equal(t, "OpenAI Prod", resp.Data.SourceGroupName)
	require.Equal(t, "OpenAI Archive", resp.Data.ArchiveGroupName)
}
