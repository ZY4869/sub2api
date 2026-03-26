package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type blacklistAccountRepoStub struct {
	AccountRepository
	account          *Account
	markBlacklisted  int
	lastReasonCode   string
	lastReasonMsg    string
	lastBlacklisted  time.Time
	lastPurgeAt      time.Time
}

func (s *blacklistAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.account == nil || s.account.ID != id {
		return nil, ErrAccountNotFound
	}
	return s.account, nil
}

func (s *blacklistAccountRepoStub) MarkBlacklisted(_ context.Context, id int64, reasonCode, reasonMessage string, blacklistedAt, purgeAt time.Time) error {
	s.markBlacklisted++
	s.lastReasonCode = reasonCode
	s.lastReasonMsg = reasonMessage
	s.lastBlacklisted = blacklistedAt
	s.lastPurgeAt = purgeAt
	s.account.ID = id
	s.account.LifecycleState = AccountLifecycleBlacklisted
	s.account.LifecycleReasonCode = reasonCode
	s.account.LifecycleReasonMessage = reasonMessage
	s.account.BlacklistedAt = &blacklistedAt
	s.account.BlacklistPurgeAt = &purgeAt
	s.account.Status = StatusDisabled
	s.account.Schedulable = false
	return nil
}

func TestAdminServiceBlacklistAccountMarksLifecycleAndRetention(t *testing.T) {
	repo := &blacklistAccountRepoStub{
		account: &Account{
			ID:             42,
			Name:           "openai-42",
			Status:         StatusActive,
			Schedulable:    true,
			LifecycleState: AccountLifecycleNormal,
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.BlacklistAccount(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, AccountLifecycleBlacklisted, account.LifecycleState)
	require.Equal(t, "manual_blacklist", account.LifecycleReasonCode)
	require.Equal(t, "Added to blacklist by admin", account.LifecycleReasonMessage)
	require.Equal(t, 1, repo.markBlacklisted)
	require.Equal(t, "manual_blacklist", repo.lastReasonCode)
	require.Equal(t, "Added to blacklist by admin", repo.lastReasonMsg)
	require.WithinDuration(t, repo.lastBlacklisted.Add(AccountBlacklistRetention), repo.lastPurgeAt, 2*time.Second)
}

func TestAdminServiceBlacklistAccountIsIdempotentForBlacklistedAccount(t *testing.T) {
	now := time.Now().UTC()
	purgeAt := now.Add(AccountBlacklistRetention)
	repo := &blacklistAccountRepoStub{
		account: &Account{
			ID:                     84,
			Name:                   "openai-84",
			Status:                 StatusDisabled,
			Schedulable:            false,
			LifecycleState:         AccountLifecycleBlacklisted,
			LifecycleReasonCode:    "account_deactivated",
			LifecycleReasonMessage: "already blacklisted",
			BlacklistedAt:          &now,
			BlacklistPurgeAt:       &purgeAt,
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.BlacklistAccount(context.Background(), 84)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, 0, repo.markBlacklisted)
	require.Equal(t, "already blacklisted", account.LifecycleReasonMessage)
}
