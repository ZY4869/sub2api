package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type archivedBulkUpdateRepoStub struct {
	AccountRepository
	lastIDs     []int64
	lastUpdates AccountBulkUpdate
}

func (s *archivedBulkUpdateRepoStub) BulkUpdate(_ context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	s.lastIDs = append([]int64(nil), ids...)
	s.lastUpdates = updates
	return int64(len(ids)), nil
}

func TestAccountArchivedLifecycleRemainsSchedulable(t *testing.T) {
	account := Account{
		Status:         StatusActive,
		Schedulable:    true,
		LifecycleState: AccountLifecycleArchived,
	}

	require.True(t, IsAccountLifecycleSchedulable(AccountLifecycleArchived))
	require.True(t, account.IsActive())
	require.True(t, account.IsSchedulable())
}

func TestAccountBlacklistedLifecycleRemainsUnschedulable(t *testing.T) {
	account := Account{
		Status:         StatusActive,
		Schedulable:    true,
		LifecycleState: AccountLifecycleBlacklisted,
	}

	require.False(t, IsAccountLifecycleSchedulable(AccountLifecycleBlacklisted))
	require.False(t, account.IsActive())
	require.False(t, account.IsSchedulable())
}

func TestAdminServiceCreateAccountArchivedKeepsActiveStatus(t *testing.T) {
	repo := &kiroAdminAccountRepoStub{}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "archived-account",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		Concurrency:          1,
		Priority:             1,
		LifecycleState:       AccountLifecycleArchived,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, StatusActive, account.Status)
	require.True(t, account.Schedulable)
	require.Equal(t, AccountLifecycleArchived, account.LifecycleState)
	require.NotNil(t, repo.created)
	require.Equal(t, StatusActive, repo.created.Status)
	require.True(t, repo.created.Schedulable)
}

func TestAdminServiceBulkUpdateAccountsArchivedDoesNotForceDisabled(t *testing.T) {
	repo := &archivedBulkUpdateRepoStub{}
	svc := &adminServiceImpl{accountRepo: repo}

	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs:            []int64{1, 2},
		LifecycleState:        AccountLifecycleArchived,
		SkipMixedChannelCheck: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []int64{1, 2}, repo.lastIDs)
	require.NotNil(t, repo.lastUpdates.LifecycleState)
	require.Equal(t, AccountLifecycleArchived, *repo.lastUpdates.LifecycleState)
	require.Nil(t, repo.lastUpdates.Status)
	require.Nil(t, repo.lastUpdates.Schedulable)
}
