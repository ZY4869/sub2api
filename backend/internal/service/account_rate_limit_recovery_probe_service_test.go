//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/stretchr/testify/require"
)

type accountAutoRecoveryProbeRepo struct {
	mockAccountRepoForGemini
	activeAccounts      []Account
	updateExtraCalls    []map[string]any
	markBlacklistedArgs []struct {
		id          int64
		reasonCode  string
		reasonMsg   string
		blacklisted time.Time
		purgeAt     time.Time
	}
}

func (r *accountAutoRecoveryProbeRepo) ListActive(context.Context) ([]Account, error) {
	return append([]Account(nil), r.activeAccounts...), nil
}

func (r *accountAutoRecoveryProbeRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	r.updateExtraCalls = append(r.updateExtraCalls, MergeStringAnyMap(nil, updates))
	return nil
}

func (r *accountAutoRecoveryProbeRepo) MarkBlacklisted(_ context.Context, id int64, reasonCode, reasonMessage string, blacklistedAt, purgeAt time.Time) error {
	r.markBlacklistedArgs = append(r.markBlacklistedArgs, struct {
		id          int64
		reasonCode  string
		reasonMsg   string
		blacklisted time.Time
		purgeAt     time.Time
	}{
		id:          id,
		reasonCode:  reasonCode,
		reasonMsg:   reasonMessage,
		blacklisted: blacklistedAt,
		purgeAt:     purgeAt,
	})
	return nil
}

type accountAutoRecoveryProbeExecutorStub struct {
	result          *BackgroundAccountTestResult
	err             error
	calls           int
	lastProbeAction string
}

func (s *accountAutoRecoveryProbeExecutorStub) RunTestBackgroundDetailed(ctx context.Context, _ ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
	s.calls++
	s.lastProbeAction, _ = ProbeActionMetadataFromContext(ctx)
	return s.result, s.err
}

type accountAutoRecoveryProbeUsageRecordingExecutorStub struct {
	result           *BackgroundAccountTestResult
	calls            int
	lastProbeAction  string
	lastInput        ScheduledTestExecutionInput
	userRepo         *systemUsageUserRepoStub
	apiKeyRepo       *systemUsageAPIKeyRepoStub
	usageRepo        *systemUsageLogRepoStub
}

func (s *accountAutoRecoveryProbeUsageRecordingExecutorStub) RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
	s.calls++
	s.lastInput = input
	s.lastProbeAction, _ = ProbeActionMetadataFromContext(ctx)
	accountTestSvc := &AccountTestService{
		userRepo:     s.userRepo,
		apiKeyRepo:   s.apiKeyRepo,
		usageLogRepo: s.usageRepo,
	}
	if err := accountTestSvc.recordSystemUsage(ctx, input, s.result); err != nil {
		return nil, err
	}
	return s.result, nil
}

type accountAutoRecoveryProbeRealUsageExecutorStub struct {
	service         *AccountTestService
	calls           int
	lastProbeAction string
	lastInput       ScheduledTestExecutionInput
}

func (s *accountAutoRecoveryProbeRealUsageExecutorStub) RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
	s.calls++
	s.lastInput = input
	s.lastProbeAction, _ = ProbeActionMetadataFromContext(ctx)
	return s.service.RunTestBackgroundDetailed(ctx, input)
}

type accountAutoRecoveryProbeRecovererStub struct {
	calls           int
	lastProbeAction string
}

func (s *accountAutoRecoveryProbeRecovererStub) RecoverAccountAfterSuccessfulTest(ctx context.Context, _ int64) (*SuccessfulTestRecoveryResult, error) {
	s.calls++
	s.lastProbeAction, _ = ProbeActionMetadataFromContext(ctx)
	return &SuccessfulTestRecoveryResult{ClearedRateLimit: true}, nil
}

func TestShouldRunAccountAutoRecoveryProbe(t *testing.T) {
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(-time.Minute)
	account := &Account{
		ID:               1,
		Status:           StatusActive,
		LifecycleState:   AccountLifecycleNormal,
		RateLimitResetAt: &resetAt,
		Extra: map[string]any{
			"rate_limit_reason": AccountRateLimitReasonUsage7d,
		},
	}

	require.True(t, shouldRunAccountAutoRecoveryProbe(account, now))

	account.Extra[accountAutoRecoveryProbeCheckedAtKey] = now.Format(time.RFC3339)
	require.False(t, shouldRunAccountAutoRecoveryProbe(account, now))
}

func TestAccountRateLimitRecoveryProbeService_RunOnceSuccess(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(-time.Minute)
	repo := &accountAutoRecoveryProbeRepo{
		activeAccounts: []Account{{
			ID:               9,
			Status:           StatusActive,
			LifecycleState:   AccountLifecycleNormal,
			RateLimitResetAt: &resetAt,
			Extra: map[string]any{
				"rate_limit_reason": AccountRateLimitReasonUsage7d,
			},
		}},
	}
	executor := &accountAutoRecoveryProbeExecutorStub{
		result: &BackgroundAccountTestResult{Status: "success"},
	}
	recoverer := &accountAutoRecoveryProbeRecovererStub{}
	svc := NewAccountRateLimitRecoveryProbeService(repo, executor, recoverer, time.Minute)
	svc.now = func() time.Time { return now }

	svc.runOnce(context.Background())

	require.Equal(t, 1, executor.calls)
	require.Equal(t, "test", executor.lastProbeAction)
	require.Equal(t, 1, recoverer.calls)
	require.Equal(t, "recover", recoverer.lastProbeAction)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountAutoRecoveryProbeStatusSuccess, repo.updateExtraCalls[0][accountAutoRecoveryProbeStatusKey])
	require.Equal(t, false, repo.updateExtraCalls[0][accountAutoRecoveryProbeBlacklisted])

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.RecoveryProbeStartedTotal)
	require.Equal(t, int64(1), snapshot.RecoveryProbeSuccessTotal)
	require.Equal(t, int64(0), snapshot.RecoveryProbeRetryTotal)
	require.Equal(t, int64(0), snapshot.RecoveryProbeBlacklistedTotal)
	require.Equal(t, int64(1), snapshot.RecoveryProbeStartedByReason[AccountRateLimitReasonUsage7d])
	require.Equal(t, int64(1), snapshot.RecoveryProbeSuccessByReason["recover"])
}

func TestAccountRateLimitRecoveryProbeService_RunOnceSuccessForUsage7dAll(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(-time.Minute)
	repo := &accountAutoRecoveryProbeRepo{
		activeAccounts: []Account{{
			ID:               12,
			Status:           StatusActive,
			LifecycleState:   AccountLifecycleNormal,
			RateLimitResetAt: &resetAt,
			Extra: map[string]any{
				"rate_limit_reason": AccountRateLimitReasonUsage7dAll,
			},
		}},
	}
	executor := &accountAutoRecoveryProbeExecutorStub{
		result: &BackgroundAccountTestResult{Status: "success"},
	}
	recoverer := &accountAutoRecoveryProbeRecovererStub{}
	svc := NewAccountRateLimitRecoveryProbeService(repo, executor, recoverer, time.Minute)
	svc.now = func() time.Time { return now }

	svc.runOnce(context.Background())

	require.Equal(t, 1, executor.calls)
	require.Equal(t, "test", executor.lastProbeAction)
	require.Equal(t, 1, recoverer.calls)
	require.Equal(t, "recover", recoverer.lastProbeAction)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountAutoRecoveryProbeStatusSuccess, repo.updateExtraCalls[0][accountAutoRecoveryProbeStatusKey])
	require.Equal(t, false, repo.updateExtraCalls[0][accountAutoRecoveryProbeBlacklisted])

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.RecoveryProbeStartedTotal)
	require.Equal(t, int64(1), snapshot.RecoveryProbeSuccessTotal)
	require.Equal(t, int64(1), snapshot.RecoveryProbeStartedByReason[AccountRateLimitReasonUsage7dAll])
	require.Equal(t, int64(1), snapshot.RecoveryProbeSuccessByReason["recover"])
}

func TestAccountRateLimitRecoveryProbeService_RunOnceSchedulesRetryOnTransientFailure(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(-time.Minute)
	repo := &accountAutoRecoveryProbeRepo{
		activeAccounts: []Account{{
			ID:               10,
			Status:           StatusActive,
			LifecycleState:   AccountLifecycleNormal,
			RateLimitResetAt: &resetAt,
			Extra: map[string]any{
				"rate_limit_reason": AccountRateLimitReasonUsage7d,
			},
		}},
	}
	executor := &accountAutoRecoveryProbeExecutorStub{
		result: &BackgroundAccountTestResult{
			Status:       "failed",
			ErrorMessage: "dial tcp timeout",
		},
	}
	recoverer := &accountAutoRecoveryProbeRecovererStub{}
	svc := NewAccountRateLimitRecoveryProbeService(repo, executor, recoverer, time.Minute)
	svc.now = func() time.Time { return now }

	svc.runOnce(context.Background())

	require.Equal(t, 1, executor.calls)
	require.Equal(t, "test", executor.lastProbeAction)
	require.Equal(t, 0, recoverer.calls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountAutoRecoveryProbeStatusRetryScheduled, repo.updateExtraCalls[0][accountAutoRecoveryProbeStatusKey])
	require.NotNil(t, repo.updateExtraCalls[0][accountAutoRecoveryProbeNextRetryKey])
	require.Empty(t, repo.markBlacklistedArgs)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.RecoveryProbeStartedTotal)
	require.Equal(t, int64(1), snapshot.RecoveryProbeRetryTotal)
	require.Equal(t, int64(1), snapshot.RecoveryProbeRetryByReason["transient_error"])
}

func TestAccountRateLimitRecoveryProbeService_RunOnceSchedulesRetryOnUpstream5xxFailure(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(-time.Minute)
	repo := &accountAutoRecoveryProbeRepo{
		activeAccounts: []Account{{
			ID:               13,
			Status:           StatusActive,
			LifecycleState:   AccountLifecycleNormal,
			RateLimitResetAt: &resetAt,
			Extra: map[string]any{
				"rate_limit_reason": AccountRateLimitReasonUsage7d,
			},
		}},
	}
	executor := &accountAutoRecoveryProbeExecutorStub{
		result: &BackgroundAccountTestResult{
			Status:       "failed",
			ErrorMessage: `API returned 502: {"error":{"message":"Upstream request failed","type":"upstream_error"}}`,
		},
	}
	recoverer := &accountAutoRecoveryProbeRecovererStub{}
	svc := NewAccountRateLimitRecoveryProbeService(repo, executor, recoverer, time.Minute)
	svc.now = func() time.Time { return now }

	svc.runOnce(context.Background())

	require.Equal(t, 1, executor.calls)
	require.Equal(t, 0, recoverer.calls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountAutoRecoveryProbeStatusRetryScheduled, repo.updateExtraCalls[0][accountAutoRecoveryProbeStatusKey])
	require.Empty(t, repo.markBlacklistedArgs)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.RecoveryProbeRetryTotal)
	require.Equal(t, int64(0), snapshot.RecoveryProbeBlacklistedTotal)
	require.Equal(t, int64(1), snapshot.RecoveryProbeRetryByReason["transient_error"])
}

func TestAccountRateLimitRecoveryProbeService_RunOnceBlacklistsExplicitFailure(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(-time.Minute)
	repo := &accountAutoRecoveryProbeRepo{
		activeAccounts: []Account{{
			ID:               11,
			Status:           StatusActive,
			LifecycleState:   AccountLifecycleNormal,
			RateLimitResetAt: &resetAt,
			Extra: map[string]any{
				"rate_limit_reason": AccountRateLimitReasonUsage7d,
			},
		}},
	}
	executor := &accountAutoRecoveryProbeExecutorStub{
		result: &BackgroundAccountTestResult{
			Status:                  "failed",
			ErrorMessage:            "refresh token is invalid",
			BlacklistAdviceDecision: string(BlacklistAdviceRecommendBlacklist),
		},
	}
	recoverer := &accountAutoRecoveryProbeRecovererStub{}
	svc := NewAccountRateLimitRecoveryProbeService(repo, executor, recoverer, time.Minute)
	svc.now = func() time.Time { return now }

	svc.runOnce(context.Background())

	require.Equal(t, 1, executor.calls)
	require.Equal(t, "test", executor.lastProbeAction)
	require.Equal(t, 0, recoverer.calls)
	require.Len(t, repo.markBlacklistedArgs, 1)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountAutoRecoveryProbeStatusBlacklisted, repo.updateExtraCalls[0][accountAutoRecoveryProbeStatusKey])
	require.Equal(t, true, repo.updateExtraCalls[0][accountAutoRecoveryProbeBlacklisted])

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.RecoveryProbeStartedTotal)
	require.Equal(t, int64(1), snapshot.RecoveryProbeBlacklistedTotal)
	require.Equal(t, int64(1), snapshot.RecoveryProbeBlacklistedByReason[string(BlacklistAdviceRecommendBlacklist)])
}

func TestAccountRateLimitRecoveryProbeService_RunOnce_RecordsUsageLogWithAutoRecoveryOperationType(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	resetAt := now.Add(-time.Minute)
	account := &Account{
		ID:             22,
		Status:         StatusActive,
		LifecycleState: AccountLifecycleNormal,
		RateLimitResetAt: &resetAt,
		Platform:       PlatformOpenAI,
		Type:           AccountTypeAPIKey,
		Concurrency:    1,
		Credentials: map[string]any{
			"api_key":  "test-token",
			"base_url": "https://api.openai.com",
		},
		Extra: map[string]any{
			"rate_limit_reason": AccountRateLimitReasonUsage7d,
		},
	}
	repo := &accountAutoRecoveryProbeRepo{
		mockAccountRepoForGemini: mockAccountRepoForGemini{
			accountsByID: map[int64]*Account{
				22: account,
			},
		},
		activeAccounts: []Account{*account},
	}
	userRepo := &systemUsageUserRepoStub{}
	apiKeyRepo := &systemUsageAPIKeyRepoStub{}
	usageRepo := &systemUsageLogRepoStub{}
	upstream := &queuedHTTPUpstream{responses: []*http.Response{
		newJSONResponse(200, ""),
	}}
	upstream.responses[0].Body = io.NopCloser(strings.NewReader(`data: {"type":"response.completed"}

`))
	accountTestSvc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
		cfg:          &config.Config{},
		userRepo:     userRepo,
		apiKeyRepo:   apiKeyRepo,
		usageLogRepo: usageRepo,
	}
	executor := &accountAutoRecoveryProbeRealUsageExecutorStub{
		service: accountTestSvc,
	}
	recoverer := &accountAutoRecoveryProbeRecovererStub{}
	svc := NewAccountRateLimitRecoveryProbeService(repo, executor, recoverer, time.Minute)
	svc.now = func() time.Time { return now }

	svc.runOnce(context.Background())

	require.Equal(t, 1, executor.calls)
	require.Equal(t, "test", executor.lastProbeAction)
	require.Equal(t, UsageOperationTypeAutoRecoveryTest, executor.lastInput.OperationType)
	require.Len(t, usageRepo.logs, 1)
	require.NotNil(t, usageRepo.logs[0].OperationType)
	require.Equal(t, UsageOperationTypeAutoRecoveryTest, *usageRepo.logs[0].OperationType)
	require.Equal(t, int64(22), usageRepo.logs[0].AccountID)
	require.Equal(t, userRepo.user.ID, usageRepo.logs[0].UserID)
	require.Equal(t, apiKeyRepo.apiKey.ID, usageRepo.logs[0].APIKeyID)
	require.Zero(t, usageRepo.logs[0].ActualCost)
	require.NotNil(t, usageRepo.logs[0].BillingExemptReason)
	require.Equal(t, BillingExemptReasonAdminFree, *usageRepo.logs[0].BillingExemptReason)
}
