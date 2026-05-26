//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminService_CreateGroup_RejectsUnsupportedPlatform(t *testing.T) {
	repo := &groupRepoStubForAdmin{}
	svc := &adminServiceImpl{groupRepo: repo}

	_, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
		Name:     "legacy-copilot-group",
		Platform: "copilot",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "UNSUPPORTED_PLATFORM")
	require.Nil(t, repo.created)
}

func TestAdminService_CreateGroup_RejectsInvalidPlatform(t *testing.T) {
	repo := &groupRepoStubForAdmin{}
	svc := &adminServiceImpl{groupRepo: repo}

	_, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
		Name:     "invalid-group",
		Platform: "unknown-platform",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_PLATFORM")
	require.Nil(t, repo.created)
}

func TestAdminService_CreateGroup_AllowsProtocolGatewayPlatform(t *testing.T) {
	repo := &groupRepoStubForAdmin{}
	svc := &adminServiceImpl{groupRepo: repo}

	group, err := svc.CreateGroup(context.Background(), &CreateGroupInput{
		Name:     "protocol-gateway-group",
		Platform: PlatformProtocolGateway,
	})

	require.NoError(t, err)
	require.NotNil(t, group)
	require.NotNil(t, repo.created)
	require.Equal(t, PlatformProtocolGateway, repo.created.Platform)
}

func TestAdminService_GetGroup_RejectsUnsupportedLegacyPlatform(t *testing.T) {
	repo := &groupRepoStubForAdmin{
		getByID: &Group{
			ID:       99,
			Name:     "legacy-copilot-group",
			Platform: "copilot",
			Status:   StatusActive,
		},
	}
	svc := &adminServiceImpl{groupRepo: repo}

	group, err := svc.GetGroup(context.Background(), 99)
	require.Error(t, err)
	require.Nil(t, group)
	require.Contains(t, err.Error(), "UNSUPPORTED_PLATFORM")
}

func TestAdminService_UpdateGroup_RejectsUnsupportedRequestedPlatform(t *testing.T) {
	repo := &groupRepoStubForAdmin{
		getByID: &Group{
			ID:       1,
			Name:     "anthropic-group",
			Platform: PlatformAnthropic,
			Status:   StatusActive,
		},
	}
	svc := &adminServiceImpl{groupRepo: repo}

	_, err := svc.UpdateGroup(context.Background(), 1, &UpdateGroupInput{
		Platform: "copilot",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "UNSUPPORTED_PLATFORM")
	require.Nil(t, repo.updated)
}

func TestAdminService_UpdateGroup_RejectsUnsupportedLegacyPlatform(t *testing.T) {
	repo := &groupRepoStubForAdmin{
		getByID: &Group{
			ID:       1,
			Name:     "legacy-copilot-group",
			Platform: "copilot",
			Status:   StatusActive,
		},
	}
	svc := &adminServiceImpl{groupRepo: repo}

	_, err := svc.UpdateGroup(context.Background(), 1, &UpdateGroupInput{
		Name: "renamed-group",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "UNSUPPORTED_PLATFORM")
	require.Nil(t, repo.updated)
}

func TestAdminService_DeleteGroup_RejectsUnsupportedLegacyPlatform(t *testing.T) {
	repo := &groupRepoStubForAdmin{
		getByID: &Group{
			ID:       1,
			Name:     "legacy-copilot-group",
			Platform: "copilot",
			Status:   StatusActive,
		},
	}
	svc := &adminServiceImpl{groupRepo: repo}

	err := svc.DeleteGroup(context.Background(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "UNSUPPORTED_PLATFORM")
}
