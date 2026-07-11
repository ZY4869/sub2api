//go:build unit

package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestAdminService_CreateUser_StoresAdminRole(t *testing.T) {
	repo := &userRepoStub{nextID: 30}
	svc := &adminServiceImpl{userRepo: repo}

	user, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:    "admin@test.com",
		Password: "strong-pass",
		Role:     RoleAdmin,
	})

	require.NoError(t, err)
	require.Equal(t, RoleAdmin, user.Role)
	require.Len(t, repo.created, 1)
	require.Equal(t, RoleAdmin, repo.created[0].Role)
}

func TestAdminService_CreateUser_RejectsInvalidRole(t *testing.T) {
	repo := &userRepoStub{nextID: 31}
	svc := &adminServiceImpl{userRepo: repo}

	_, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:    "invalid-role@test.com",
		Password: "strong-pass",
		Role:     "owner",
	})

	require.Error(t, err)
	require.Equal(t, "USER_ROLE_INVALID", infraerrors.Reason(err))
	require.Empty(t, repo.created)
}

func TestAdminService_UpdateUser_PromotesRoleAndInvalidatesAuthCache(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 41, Role: RoleUser, Status: StatusActive, Concurrency: 1}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{userRepo: repo, authCacheInvalidator: invalidator}
	role := RoleAdmin

	user, err := svc.UpdateUser(context.Background(), 41, &UpdateUserInput{Role: &role})

	require.NoError(t, err)
	require.Equal(t, RoleAdmin, user.Role)
	require.Equal(t, []int64{41}, invalidator.userIDs)
	require.Len(t, repo.updated, 1)
	require.Equal(t, RoleAdmin, repo.updated[0].Role)
}

func TestAdminService_UpdateUser_DemotesAdminWhenAnotherActiveAdminExists(t *testing.T) {
	baseRepo := &userRepoStub{
		user:      &User{ID: 42, Role: RoleAdmin, Status: StatusActive, Concurrency: 1, AdminFreeBilling: true, RequestDetailsReview: true},
		allowList: true,
		listUsers: []User{
			{ID: 42, Role: RoleAdmin, Status: StatusActive},
			{ID: 99, Role: RoleAdmin, Status: StatusActive},
		},
		listResult: &pagination.PaginationResult{Total: 2, Page: 1, PageSize: 2},
	}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{userRepo: repo, authCacheInvalidator: invalidator}
	role := RoleUser

	user, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{Role: &role})

	require.NoError(t, err)
	require.Equal(t, RoleUser, user.Role)
	require.False(t, user.AdminFreeBilling)
	require.True(t, user.RequestDetailsReview)
	require.Equal(t, []int64{42}, invalidator.userIDs)
	require.Len(t, repo.updated, 1)
}

func TestAdminService_UpdateUser_RejectsLastActiveAdminDemotion(t *testing.T) {
	baseRepo := &userRepoStub{
		user:       &User{ID: 43, Role: RoleAdmin, Status: StatusActive, Concurrency: 1},
		allowList:  true,
		listUsers:  []User{{ID: 43, Role: RoleAdmin, Status: StatusActive}},
		listResult: &pagination.PaginationResult{Total: 1, Page: 1, PageSize: 2},
	}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{userRepo: repo, authCacheInvalidator: invalidator}
	role := RoleUser

	_, err := svc.UpdateUser(context.Background(), 43, &UpdateUserInput{Role: &role})

	require.Error(t, err)
	require.Equal(t, "LAST_ADMIN_GUARD", infraerrors.Reason(err))
	require.Empty(t, invalidator.userIDs)
	require.Empty(t, repo.updated)
}

func TestAdminService_UpdateUser_RejectsLastActiveAdminDisable(t *testing.T) {
	baseRepo := &userRepoStub{
		user:       &User{ID: 44, Role: RoleAdmin, Status: StatusActive, Concurrency: 1},
		allowList:  true,
		listUsers:  []User{{ID: 44, Role: RoleAdmin, Status: StatusActive}},
		listResult: &pagination.PaginationResult{Total: 1, Page: 1, PageSize: 2},
	}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{userRepo: repo, authCacheInvalidator: invalidator}

	_, err := svc.UpdateUser(context.Background(), 44, &UpdateUserInput{Status: StatusDisabled})

	require.Error(t, err)
	require.Equal(t, "LAST_ADMIN_GUARD", infraerrors.Reason(err))
	require.Empty(t, invalidator.userIDs)
	require.Empty(t, repo.updated)
}

func TestAdminService_UpdateUser_RejectsInvalidRole(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 45, Role: RoleUser, Status: StatusActive, Concurrency: 1}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	svc := &adminServiceImpl{userRepo: repo}
	role := "owner"

	_, err := svc.UpdateUser(context.Background(), 45, &UpdateUserInput{Role: &role})

	require.Error(t, err)
	require.Equal(t, "USER_ROLE_INVALID", infraerrors.Reason(err))
	require.Empty(t, repo.updated)
}
