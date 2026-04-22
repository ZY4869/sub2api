package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type blacklistAccountRepoStub struct {
	AccountRepository
	account          *Account
	markBlacklisted  int
	restoreCalls     int
	lastReasonCode   string
	lastReasonMsg    string
	lastBlacklisted  time.Time
	lastPurgeAt      time.Time
	updateExtraCalls []map[string]any
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

func (s *blacklistAccountRepoStub) RestoreBlacklisted(_ context.Context, id int64) error {
	s.restoreCalls++
	s.account.ID = id
	s.account.LifecycleState = AccountLifecycleNormal
	s.account.LifecycleReasonCode = ""
	s.account.LifecycleReasonMessage = ""
	s.account.BlacklistedAt = nil
	s.account.BlacklistPurgeAt = nil
	s.account.Status = StatusActive
	s.account.Schedulable = true
	s.account.ErrorMessage = ""
	return nil
}

func (s *blacklistAccountRepoStub) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	s.updateExtraCalls = append(s.updateExtraCalls, MergeStringAnyMap(nil, updates))
	s.account.Extra = MergeStringAnyMap(s.account.Extra, updates)
	return nil
}

type blacklistCandidateSettingRepoStub struct {
	values map[string]string
}

func (s *blacklistCandidateSettingRepoStub) Get(_ context.Context, _ string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *blacklistCandidateSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *blacklistCandidateSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *blacklistCandidateSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *blacklistCandidateSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *blacklistCandidateSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *blacklistCandidateSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
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

	account, err := svc.BlacklistAccount(context.Background(), 42, nil)
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

	account, err := svc.BlacklistAccount(context.Background(), 84, nil)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, 0, repo.markBlacklisted)
	require.Equal(t, "already blacklisted", account.LifecycleReasonMessage)
}

func TestAdminServiceBlacklistAccountRecordsFeedbackCandidate(t *testing.T) {
	repo := &blacklistAccountRepoStub{
		account: &Account{
			ID:             99,
			Name:           "openai-99",
			Platform:       PlatformOpenAI,
			Type:           AccountTypeAPIKey,
			Status:         StatusActive,
			Schedulable:    true,
			LifecycleState: AccountLifecycleNormal,
		},
	}
	settingRepo := &blacklistCandidateSettingRepoStub{values: map[string]string{}}
	svc := &adminServiceImpl{
		accountRepo:    repo,
		settingService: NewSettingService(settingRepo, &config.Config{}),
	}

	account, err := svc.BlacklistAccount(context.Background(), 99, &BlacklistAccountInput{
		Source: "test_modal",
		Feedback: &BlacklistFeedbackInput{
			Fingerprint:    "abc12345",
			AdviceDecision: string(BlacklistAdviceRecommendBlacklist),
			Action:         "blacklist",
			Platform:       PlatformOpenAI,
			StatusCode:     401,
			ErrorCode:      "invalid_api_key",
			MessageKeywords: []string{
				"invalid",
				"key",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, account)

	raw, ok := settingRepo.values[SettingKeyBlacklistRuleCandidates]
	require.True(t, ok)

	var settings BlacklistRuleCandidateSettings
	require.NoError(t, json.Unmarshal([]byte(raw), &settings))
	require.Len(t, settings.Rules, 1)
	require.Equal(t, "abc12345", settings.Rules[0].Fingerprint)
	require.Equal(t, "openai", settings.Rules[0].Platform)
	require.Equal(t, "invalid_api_key", settings.Rules[0].ErrorCode)
	require.Equal(t, "blacklist", settings.Rules[0].AdminAction)
}

func TestAdminServiceRestoreBlacklistedAccountClearsAutoRecoveryProbeState(t *testing.T) {
	repo := &blacklistAccountRepoStub{
		account: &Account{
			ID:             101,
			Name:           "openai-101",
			Status:         StatusDisabled,
			Schedulable:    false,
			LifecycleState: AccountLifecycleBlacklisted,
			Extra: map[string]any{
				accountAutoRecoveryProbeCheckedAtKey: "2026-04-21T11:31:03Z",
				accountAutoRecoveryProbeStatusKey:    AccountAutoRecoveryProbeStatusBlacklisted,
				accountAutoRecoveryProbeSummaryKey:   "API returned 502: {\"error\":{\"message\":\"Upstream request failed\"}}",
				accountAutoRecoveryProbeBlacklisted:  true,
				accountAutoRecoveryProbeErrorCodeKey: accountRateLimitRecoveryProbeReasonCode,
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	account, err := svc.RestoreBlacklistedAccount(context.Background(), 101)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, 1, repo.restoreCalls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountLifecycleNormal, account.LifecycleState)
	require.Equal(t, StatusActive, account.Status)
	require.Nil(t, AccountAutoRecoveryProbeSummaryFromExtra(account.Extra))
}
