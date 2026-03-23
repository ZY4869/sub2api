package admin

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestParseBatchCreateLineKiroLegacyRegionNormalization(t *testing.T) {
	line, err := parseBatchCreateLine(`{"region":"ap-southeast-1","access_token":"kiro-access"}`, service.PlatformKiro, service.AccountTypeOAuth)
	require.NoError(t, err)
	require.NotNil(t, line)
	require.Equal(t, "kiro-access", line.Credentials["access_token"])
	require.Equal(t, "ap-southeast-1", line.Credentials["api_region"])
	_, hasLegacyRegion := line.Credentials["region"]
	require.False(t, hasLegacyRegion)
}

func TestBuildBatchCreateAccountInputArchiveModeOverridesGroupBinding(t *testing.T) {
	req := &BatchCreateAccountsRequest{
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		GroupIDs: []int64{10, 11},
	}
	line := &batchCreateLineOverrides{
		Credentials: map[string]any{"access_token": "token-value"},
		GroupIDs:    &[]int64{22, 23},
	}

	input, ignoredGroups, err := buildBatchCreateAccountInput(req, line, "openai-batch", 1, &service.Group{ID: 99})
	require.NoError(t, err)
	require.True(t, ignoredGroups)
	require.Equal(t, "openai-batch-001", input.Name)
	require.Equal(t, service.StatusDisabled, input.Status)
	require.Equal(t, []int64{99}, input.GroupIDs)
	require.Equal(t, "token-value", input.Credentials["access_token"])
}

func TestExecuteBatchCreateAccountsArchiveAutoCreatesGroupAndNamesAccounts(t *testing.T) {
	adminSvc := newStubAdminService()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	result, err := handler.executeBatchCreateAccounts(context.Background(), &BatchCreateAccountsRequest{
		Platform:   service.PlatformOpenAI,
		Type:       service.AccountTypeOAuth,
		NamePrefix: "archive-batch",
		Items: []string{
			"token-1",
			`{"name":"named-account","access_token":"token-2","group_ids":[1,2]}`,
		},
		Archive: BatchCreateAccountsArchiveRequest{
			Enabled:   true,
			GroupName: "OpenAI Archive",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, result.CreatedCount)
	require.Equal(t, 0, result.FailedCount)
	require.Equal(t, "OpenAI Archive", result.ArchiveGroupName)
	require.NotNil(t, result.ArchiveGroupID)
	require.Len(t, adminSvc.createdAccounts, 2)
	require.Len(t, adminSvc.groups, 2)
	require.Equal(t, "archive-batch-001", adminSvc.createdAccounts[0].Name)
	require.Equal(t, service.StatusDisabled, adminSvc.createdAccounts[0].Status)
	require.Equal(t, []int64{*result.ArchiveGroupID}, adminSvc.createdAccounts[0].GroupIDs)
	require.Equal(t, "named-account", adminSvc.createdAccounts[1].Name)
	require.Equal(t, []int64{*result.ArchiveGroupID}, adminSvc.createdAccounts[1].GroupIDs)
	require.Contains(t, result.Results[1].Message, "archive mode ignored provided group_ids")
}

func TestResolveBatchCreateArchiveGroupRejectsDifferentPlatformConflict(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.groups = append(adminSvc.groups, service.Group{
		ID:       88,
		Name:     "Shared Archive",
		Platform: service.PlatformAnthropic,
		Status:   service.StatusActive,
	})
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	group, err := handler.resolveBatchCreateArchiveGroup(context.Background(), service.PlatformOpenAI, "Shared Archive")
	require.Nil(t, group)
	require.Error(t, err)
	require.Equal(t, "ACCOUNT_BATCH_CREATE_ARCHIVE_GROUP_PLATFORM_CONFLICT", infraerrors.Reason(err))
}

func TestResolveBatchCreateArchiveGroupReusesSamePlatformGroup(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.groups = append(adminSvc.groups, service.Group{
		ID:       77,
		Name:     "Kiro Archive",
		Platform: service.PlatformKiro,
		Status:   service.StatusActive,
	})
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	group, err := handler.resolveBatchCreateArchiveGroup(context.Background(), service.PlatformKiro, "Kiro Archive")
	require.NoError(t, err)
	require.NotNil(t, group)
	require.Equal(t, int64(77), group.ID)
}
