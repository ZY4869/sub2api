//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAdminService_CreateUser_Success(t *testing.T) {
	repo := &userRepoStub{nextID: 10}
	svc := &adminServiceImpl{userRepo: repo}

	input := &CreateUserInput{
		Email:         "user@test.com",
		Password:      "strong-pass",
		Username:      "tester",
		Notes:         "note",
		Balance:       12.5,
		Concurrency:   7,
		AllowedGroups: []int64{3, 5},
	}

	user, err := svc.CreateUser(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, int64(10), user.ID)
	require.Equal(t, input.Email, user.Email)
	require.Equal(t, input.Username, user.Username)
	require.Equal(t, input.Notes, user.Notes)
	require.Equal(t, input.Balance, user.Balance)
	require.Equal(t, input.Concurrency, user.Concurrency)
	require.Equal(t, input.AllowedGroups, user.AllowedGroups)
	require.Equal(t, APIKeyModelBindingModeModelRequired, user.EffectiveAPIKeyModelBindingMode())
	require.Equal(t, ExternalModelCatalogViewModeFollowKeyBinding, user.ExternalModelCatalogViewMode)
	require.Equal(t, ExternalModelCatalogViewModeModelOnly, user.EffectiveExternalModelCatalogViewMode())
	require.Equal(t, RoleUser, user.Role)
	require.Equal(t, StatusActive, user.Status)
	require.True(t, user.CheckPassword(input.Password))
	require.Len(t, repo.created, 1)
	require.Equal(t, user, repo.created[0])
}

func TestAdminService_CreateUser_StoresExternalModelCatalogViewMode(t *testing.T) {
	repo := &userRepoStub{nextID: 13}
	svc := &adminServiceImpl{userRepo: repo}

	user, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:                        "catalog-view@test.com",
		Password:                     "strong-pass",
		ExternalModelCatalogViewMode: ExternalModelCatalogViewModeGroupFirst,
	})

	require.NoError(t, err)
	require.Equal(t, ExternalModelCatalogViewModeGroupFirst, user.ExternalModelCatalogViewMode)
	require.Equal(t, ExternalModelCatalogViewModeGroupFirst, user.EffectiveExternalModelCatalogViewMode())
	require.Len(t, repo.created, 1)
	require.Equal(t, ExternalModelCatalogViewModeGroupFirst, repo.created[0].ExternalModelCatalogViewMode)
}

func TestAdminService_CreateUser_RejectsInvalidExternalModelCatalogViewMode(t *testing.T) {
	repo := &userRepoStub{nextID: 13}
	svc := &adminServiceImpl{userRepo: repo}

	_, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:                        "catalog-invalid@test.com",
		Password:                     "strong-pass",
		ExternalModelCatalogViewMode: "invalid",
	})

	require.Error(t, err)
	require.Equal(t, "EXTERNAL_MODEL_CATALOG_VIEW_MODE_INVALID", infraerrors.Reason(err))
	require.Empty(t, repo.created)
}

func TestAdminService_CreateUser_StoresGroupAllowedModelBindingMode(t *testing.T) {
	repo := &userRepoStub{nextID: 11}
	svc := &adminServiceImpl{userRepo: repo}

	user, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:                  "group-allowed@test.com",
		Password:               "strong-pass",
		APIKeyModelBindingMode: APIKeyModelBindingModeGroupAllowed,
	})

	require.NoError(t, err)
	require.Equal(t, APIKeyModelBindingModeGroupAllowed, user.EffectiveAPIKeyModelBindingMode())
	require.Len(t, repo.created, 1)
	require.Equal(t, APIKeyModelBindingModeGroupAllowed, repo.created[0].APIKeyModelBindingMode)
}

func TestAdminService_UpdateUser_ModelBindingModeInvalidatesAuthCache(t *testing.T) {
	baseRepo := &userRepoStub{
		user: &User{
			ID:                     12,
			Email:                  "mode@test.com",
			Role:                   RoleUser,
			Status:                 StatusActive,
			Concurrency:            1,
			APIKeyModelBindingMode: APIKeyModelBindingModeModelRequired,
		},
	}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		authCacheInvalidator: invalidator,
	}
	mode := APIKeyModelBindingModeGroupAllowed

	user, err := svc.UpdateUser(context.Background(), 12, &UpdateUserInput{
		APIKeyModelBindingMode: &mode,
	})

	require.NoError(t, err)
	require.Equal(t, APIKeyModelBindingModeGroupAllowed, user.EffectiveAPIKeyModelBindingMode())
	require.Equal(t, []int64{12}, invalidator.userIDs)
	require.Len(t, repo.updated, 1)
	require.Equal(t, APIKeyModelBindingModeGroupAllowed, repo.updated[0].APIKeyModelBindingMode)
}

func TestAdminService_UpdateUser_ExternalModelCatalogViewModeInvalidatesAuthCache(t *testing.T) {
	baseRepo := &userRepoStub{
		user: &User{
			ID:                           14,
			Email:                        "catalog-mode@test.com",
			Role:                         RoleUser,
			Status:                       StatusActive,
			Concurrency:                  1,
			APIKeyModelBindingMode:       APIKeyModelBindingModeModelRequired,
			ExternalModelCatalogViewMode: ExternalModelCatalogViewModeFollowKeyBinding,
		},
	}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		authCacheInvalidator: invalidator,
	}
	mode := ExternalModelCatalogViewModeModelOnly

	user, err := svc.UpdateUser(context.Background(), 14, &UpdateUserInput{
		ExternalModelCatalogViewMode: &mode,
	})

	require.NoError(t, err)
	require.Equal(t, ExternalModelCatalogViewModeModelOnly, user.ExternalModelCatalogViewMode)
	require.Equal(t, []int64{14}, invalidator.userIDs)
	require.Len(t, repo.updated, 1)
	require.Equal(t, ExternalModelCatalogViewModeModelOnly, repo.updated[0].ExternalModelCatalogViewMode)
}

func TestAdminService_UpdateUser_RejectsInvalidExternalModelCatalogViewMode(t *testing.T) {
	baseRepo := &userRepoStub{
		user: &User{
			ID:                           15,
			Email:                        "catalog-invalid-update@test.com",
			Role:                         RoleUser,
			Status:                       StatusActive,
			Concurrency:                  1,
			APIKeyModelBindingMode:       APIKeyModelBindingModeModelRequired,
			ExternalModelCatalogViewMode: ExternalModelCatalogViewModeFollowKeyBinding,
		},
	}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		authCacheInvalidator: invalidator,
	}
	mode := "invalid"

	_, err := svc.UpdateUser(context.Background(), 15, &UpdateUserInput{
		ExternalModelCatalogViewMode: &mode,
	})

	require.Error(t, err)
	require.Equal(t, "EXTERNAL_MODEL_CATALOG_VIEW_MODE_INVALID", infraerrors.Reason(err))
	require.Empty(t, invalidator.userIDs)
	require.Empty(t, repo.updated)
}

func TestAdminService_CreateUser_EmailExists(t *testing.T) {
	repo := &userRepoStub{createErr: ErrEmailExists}
	svc := &adminServiceImpl{userRepo: repo}

	_, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:    "dup@test.com",
		Password: "password",
	})
	require.ErrorIs(t, err, ErrEmailExists)
	require.Empty(t, repo.created)
}

func TestAdminService_CreateUser_CreateError(t *testing.T) {
	createErr := errors.New("db down")
	repo := &userRepoStub{createErr: createErr}
	svc := &adminServiceImpl{userRepo: repo}

	_, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:    "user@test.com",
		Password: "password",
	})
	require.ErrorIs(t, err, createErr)
	require.Empty(t, repo.created)
}

func TestAdminService_CreateUser_AssignsDefaultSubscriptions(t *testing.T) {
	repo := &userRepoStub{nextID: 21}
	assigner := &defaultSubscriptionAssignerStub{}
	cfg := &config.Config{
		Default: config.DefaultConfig{
			UserBalance:     0,
			UserConcurrency: 1,
		},
	}
	settingService := NewSettingService(&settingRepoStub{values: map[string]string{
		SettingKeyDefaultSubscriptions: `[{"group_id":5,"validity_days":30}]`,
	}}, cfg)
	svc := &adminServiceImpl{
		userRepo:           repo,
		settingService:     settingService,
		defaultSubAssigner: assigner,
	}

	_, err := svc.CreateUser(context.Background(), &CreateUserInput{
		Email:    "new-user@test.com",
		Password: "password",
	})
	require.NoError(t, err)
	require.Len(t, assigner.calls, 1)
	require.Equal(t, int64(21), assigner.calls[0].UserID)
	require.Equal(t, int64(5), assigner.calls[0].GroupID)
	require.Equal(t, 30, assigner.calls[0].ValidityDays)
}
