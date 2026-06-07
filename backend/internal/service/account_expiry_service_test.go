//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type accountExpiryRepoStub struct {
	accounts         []Account
	accountsByID     map[int64]*Account
	listFilterCalls  int
	updateExtraCalls []expiryRepoUpdateExtraCall
	updateCalls      []*Account
	updateErr        error
	markBlacklisted  []expiryRepoBlacklistCall
}

type expiryRepoUpdateExtraCall struct {
	id      int64
	updates map[string]any
}

type expiryRepoBlacklistCall struct {
	id            int64
	reasonCode    string
	reasonMessage string
	blacklistedAt time.Time
	purgeAt       time.Time
}

func (r *accountExpiryRepoStub) Create(context.Context, *Account) error { panic("unexpected") }
func (r *accountExpiryRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if account, ok := r.accountsByID[id]; ok {
		return account, nil
	}
	return nil, ErrAccountNotFound
}
func (r *accountExpiryRepoStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ExistsByID(context.Context, int64) (bool, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) FindByExtraField(context.Context, string, any) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) Update(_ context.Context, account *Account) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	cloned := *account
	cloned.Credentials = cloneStringAnyMap(account.Credentials)
	cloned.Extra = cloneStringAnyMap(account.Extra)
	r.updateCalls = append(r.updateCalls, &cloned)
	if r.accountsByID == nil {
		r.accountsByID = map[int64]*Account{}
	}
	r.accountsByID[account.ID] = &cloned
	return nil
}
func (r *accountExpiryRepoStub) Delete(context.Context, int64) error { panic("unexpected") }
func (r *accountExpiryRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, _, _ string, _ int64, lifecycle, _ string) ([]Account, *pagination.PaginationResult, error) {
	r.listFilterCalls++
	if NormalizeAccountLifecycleInput(lifecycle) == AccountLifecycleAll {
		return append([]Account(nil), r.accounts...), nil, nil
	}
	filtered := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if NormalizeAccountLifecycleInput(account.LifecycleState) != NormalizeAccountLifecycleInput(lifecycle) {
			continue
		}
		filtered = append(filtered, account)
	}
	return filtered, nil, nil
}
func (r *accountExpiryRepoStub) GetStatusSummary(context.Context, AccountStatusSummaryFilters) (*AccountStatusSummary, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListActive(context.Context) ([]Account, error) { panic("unexpected") }
func (r *accountExpiryRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) UpdateLastUsed(context.Context, int64) error { panic("unexpected") }
func (r *accountExpiryRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) SetError(context.Context, int64, string) error { panic("unexpected") }
func (r *accountExpiryRepoStub) ClearError(context.Context, int64) error       { panic("unexpected") }
func (r *accountExpiryRepoStub) SetSchedulable(context.Context, int64, bool) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) BindGroups(context.Context, int64, []int64) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ClearRateLimit(context.Context, int64) error { panic("unexpected") }
func (r *accountExpiryRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ClearModelRateLimits(context.Context, int64) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	r.updateExtraCalls = append(r.updateExtraCalls, expiryRepoUpdateExtraCall{
		id:      id,
		updates: cloneStringAnyMap(updates),
	})
	return nil
}
func (r *accountExpiryRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) MarkBlacklisted(_ context.Context, id int64, reasonCode, reasonMessage string, blacklistedAt, purgeAt time.Time) error {
	r.markBlacklisted = append(r.markBlacklisted, expiryRepoBlacklistCall{
		id:            id,
		reasonCode:    reasonCode,
		reasonMessage: reasonMessage,
		blacklistedAt: blacklistedAt,
		purgeAt:       purgeAt,
	})
	return nil
}
func (r *accountExpiryRepoStub) RestoreBlacklisted(context.Context, int64) error { panic("unexpected") }
func (r *accountExpiryRepoStub) ListBlacklistedIDs(context.Context) ([]int64, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ListBlacklistedForPurge(context.Context, time.Time, int) ([]Account, error) {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	panic("unexpected")
}
func (r *accountExpiryRepoStub) ResetQuotaUsed(context.Context, int64) error { panic("unexpected") }

type accountExpiryProbeExecutorStub struct {
	result *BackgroundAccountTestResult
	err    error
	calls  []ScheduledTestExecutionInput
}

func (s *accountExpiryProbeExecutorStub) RunTestBackgroundDetailed(_ context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
	s.calls = append(s.calls, input)
	return s.result, s.err
}

type periodicJobLeaderGateStub struct {
	allow bool
	calls int
	jobs  []string
}

func (g *periodicJobLeaderGateStub) RunIfLeader(ctx context.Context, jobName string, _ time.Duration, run func(context.Context)) bool {
	g.calls++
	g.jobs = append(g.jobs, jobName)
	if !g.allow {
		return false
	}
	run(ctx)
	return true
}

func TestAccountExpiryService_RunLeaderOnce_SkipsWhenNotLeader(t *testing.T) {
	repo := &accountExpiryRepoStub{}
	svc := NewAccountExpiryService(repo, &accountExpiryProbeExecutorStub{}, time.Minute)
	gate := &periodicJobLeaderGateStub{}
	svc.SetLeaderGate(gate)

	ok := svc.runLeaderOnce(context.Background())

	require.False(t, ok)
	require.Equal(t, 1, gate.calls)
	require.Equal(t, []string{accountExpiryJobName}, gate.jobs)
	require.Zero(t, repo.listFilterCalls)
}

func TestAccountExpiryService_RunLeaderOnce_RunsWhenLeader(t *testing.T) {
	repo := &accountExpiryRepoStub{}
	svc := NewAccountExpiryService(repo, &accountExpiryProbeExecutorStub{}, time.Minute)
	gate := &periodicJobLeaderGateStub{allow: true}
	svc.SetLeaderGate(gate)

	ok := svc.runLeaderOnce(context.Background())

	require.True(t, ok)
	require.Equal(t, 1, gate.calls)
	require.Equal(t, 1, repo.listFilterCalls)
}

func TestAccountExpiryService_RunOnce_WaitsUntilBlockingWindowEnds(t *testing.T) {
	now := time.Date(2026, 5, 8, 8, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-2 * time.Hour)
	windowEnd := now.Add(30 * time.Minute)
	account := Account{
		ID:                 1,
		Status:             StatusActive,
		Schedulable:        true,
		AutoPauseOnExpired: true,
		LifecycleState:     AccountLifecycleNormal,
		ExpiresAt:          &expiresAt,
		SessionWindowEnd:   &windowEnd,
		Extra:              map[string]any{},
	}
	repo := &accountExpiryRepoStub{
		accounts:     []Account{account},
		accountsByID: map[int64]*Account{1: cloneAccountForBackgroundProbe(&account)},
	}
	executor := &accountExpiryProbeExecutorStub{}
	svc := NewAccountExpiryService(repo, executor, time.Minute)
	svc.SetNow(func() time.Time { return now })

	svc.runOnce(context.Background())

	require.Empty(t, executor.calls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountExpiryProbeStatusWaiting, repo.updateExtraCalls[0].updates[accountExpiryProbeStatusKey])
	require.Equal(t, windowEnd.Format(time.RFC3339), repo.updateExtraCalls[0].updates[accountExpiryProbeNextCheckAtKey])
	require.Contains(t, repo.updateExtraCalls[0].updates[accountExpiryProbeSummaryKey], "session window")
}

func TestAccountExpiryService_RunOnce_ExtendsExpiryAndSetsTemporaryPriority(t *testing.T) {
	now := time.Date(2026, 5, 8, 9, 0, 0, 0, time.UTC)
	oldExpiry := now.Add(-3 * time.Hour)
	account := Account{
		ID:                 2,
		Status:             StatusDisabled,
		Schedulable:        false,
		AutoPauseOnExpired: true,
		LifecycleState:     AccountLifecycleNormal,
		ExpiresAt:          &oldExpiry,
		Extra: map[string]any{
			accountExpiryProbeExtensionDaysKey: 3,
		},
	}
	repo := &accountExpiryRepoStub{
		accounts:     []Account{account},
		accountsByID: map[int64]*Account{2: cloneAccountForBackgroundProbe(&account)},
	}
	executor := &accountExpiryProbeExecutorStub{
		result: &BackgroundAccountTestResult{Status: "success"},
	}
	svc := NewAccountExpiryService(repo, executor, time.Minute)
	svc.SetNow(func() time.Time { return now })

	svc.runOnce(context.Background())

	require.Len(t, executor.calls, 1)
	require.Len(t, repo.updateCalls, 1)
	updated := repo.updateCalls[0]
	expectedExpiry := now.Add(72 * time.Hour)
	require.NotNil(t, updated.ExpiresAt)
	require.Equal(t, expectedExpiry, updated.ExpiresAt.UTC())
	require.True(t, updated.Schedulable)
	require.Equal(t, StatusActive, updated.Status)
	require.Equal(t, AccountExpiryProbeStatusSuccess, updated.Extra[accountExpiryProbeStatusKey])
	require.Equal(t, expectedExpiry.Format(time.RFC3339), updated.Extra[accountExpiryProbePriorityUntilKey])
}

func TestAccountExpiryService_RunOnce_DisablesAccountOnFailedProbe(t *testing.T) {
	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-1 * time.Hour)
	account := Account{
		ID:                 3,
		Status:             StatusActive,
		Schedulable:        true,
		AutoPauseOnExpired: true,
		LifecycleState:     AccountLifecycleNormal,
		ExpiresAt:          &expiresAt,
		Extra:              map[string]any{},
	}
	repo := &accountExpiryRepoStub{
		accounts:     []Account{account},
		accountsByID: map[int64]*Account{3: cloneAccountForBackgroundProbe(&account)},
	}
	executor := &accountExpiryProbeExecutorStub{
		result: &BackgroundAccountTestResult{
			Status:       "failed",
			ErrorMessage: "temporary upstream failure",
		},
	}
	svc := NewAccountExpiryService(repo, executor, time.Minute)
	svc.SetNow(func() time.Time { return now })

	svc.runOnce(context.Background())

	require.Len(t, repo.updateCalls, 1)
	updated := repo.updateCalls[0]
	require.False(t, updated.Schedulable)
	require.Equal(t, StatusDisabled, updated.Status)
	require.Equal(t, AccountExpiryProbeStatusDisabled, updated.Extra[accountExpiryProbeStatusKey])
	require.Equal(t, "temporary upstream failure", updated.Extra[accountExpiryProbeSummaryKey])
	require.Empty(t, repo.markBlacklisted)
}

func TestAccountExpiryService_RunOnce_BlacklistsWhenAdviceRequiresIt(t *testing.T) {
	now := time.Date(2026, 5, 8, 11, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-2 * time.Hour)
	account := Account{
		ID:                 4,
		Status:             StatusActive,
		Schedulable:        true,
		AutoPauseOnExpired: true,
		LifecycleState:     AccountLifecycleNormal,
		ExpiresAt:          &expiresAt,
		Extra:              map[string]any{},
	}
	repo := &accountExpiryRepoStub{
		accounts:     []Account{account},
		accountsByID: map[int64]*Account{4: cloneAccountForBackgroundProbe(&account)},
	}
	executor := &accountExpiryProbeExecutorStub{
		result: &BackgroundAccountTestResult{
			Status:                  "failed",
			ErrorMessage:            "account is hard banned",
			BlacklistAdviceDecision: string(BlacklistAdviceRecommendBlacklist),
		},
	}
	svc := NewAccountExpiryService(repo, executor, time.Minute)
	svc.SetNow(func() time.Time { return now })

	svc.runOnce(context.Background())

	require.Len(t, repo.markBlacklisted, 1)
	require.Empty(t, repo.updateCalls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountExpiryProbeStatusBlacklisted, repo.updateExtraCalls[0].updates[accountExpiryProbeStatusKey])
	require.Equal(t, "account is hard banned", repo.updateExtraCalls[0].updates[accountExpiryProbeSummaryKey])
}

func TestAccountExpiryService_RunOnce_AutoRenewsFromOriginalExpiry(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	previousExpiry := time.Date(2026, 5, 15, 8, 30, 0, 0, time.UTC)
	cases := []struct {
		name     string
		period   string
		expected time.Time
	}{
		{name: "month", period: AccountAutoRenewPeriodMonth, expected: time.Date(2026, 6, 15, 8, 30, 0, 0, time.UTC)},
		{name: "quarter", period: AccountAutoRenewPeriodQuarter, expected: time.Date(2026, 8, 15, 8, 30, 0, 0, time.UTC)},
		{name: "year", period: AccountAutoRenewPeriodYear, expected: time.Date(2027, 5, 15, 8, 30, 0, 0, time.UTC)},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			accountID := int64(20 + index)
			account := Account{
				ID:                 accountID,
				Status:             StatusDisabled,
				Schedulable:        false,
				AutoPauseOnExpired: true,
				AutoRenewEnabled:   true,
				AutoRenewPeriod:    tc.period,
				LifecycleState:     AccountLifecycleNormal,
				ExpiresAt:          &previousExpiry,
				Extra:              map[string]any{},
			}
			repo := &accountExpiryRepoStub{
				accounts:     []Account{account},
				accountsByID: map[int64]*Account{accountID: cloneAccountForBackgroundProbe(&account)},
			}
			executor := &accountExpiryProbeExecutorStub{}
			svc := NewAccountExpiryService(repo, executor, time.Minute)
			svc.SetNow(func() time.Time { return now })

			svc.runOnce(context.Background())

			require.Empty(t, executor.calls)
			require.Len(t, repo.updateCalls, 1)
			updated := repo.updateCalls[0]
			require.NotNil(t, updated.ExpiresAt)
			require.Equal(t, tc.expected, updated.ExpiresAt.UTC())
			require.False(t, updated.Schedulable)
			require.Equal(t, StatusDisabled, updated.Status)
			require.Equal(t, AccountAutoRenewStatusSuccess, updated.Extra[accountAutoRenewStatusKey])
			require.Equal(t, tc.period, updated.Extra[accountAutoRenewPeriodKey])
			require.Equal(t, previousExpiry.Format(time.RFC3339), updated.Extra[accountAutoRenewPreviousExpiresKey])
			require.Equal(t, tc.expected.Format(time.RFC3339), updated.Extra[accountAutoRenewNextExpiresKey])
			require.Contains(t, updated.Extra[accountAutoRenewSummaryKey], "Auto renewed account")
		})
	}
}

func TestAccountExpiryService_RunOnce_AutoRenewsLongExpiredAccountToFuture(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	previousExpiry := time.Date(2026, 1, 15, 8, 30, 0, 0, time.UTC)
	account := Account{
		ID:                 29,
		Status:             StatusActive,
		Schedulable:        true,
		AutoPauseOnExpired: true,
		AutoRenewEnabled:   true,
		AutoRenewPeriod:    AccountAutoRenewPeriodMonth,
		LifecycleState:     AccountLifecycleNormal,
		ExpiresAt:          &previousExpiry,
		Extra:              map[string]any{},
	}
	repo := &accountExpiryRepoStub{
		accounts:     []Account{account},
		accountsByID: map[int64]*Account{29: cloneAccountForBackgroundProbe(&account)},
	}
	executor := &accountExpiryProbeExecutorStub{}
	svc := NewAccountExpiryService(repo, executor, time.Minute)
	svc.SetNow(func() time.Time { return now })

	svc.runOnce(context.Background())

	require.Empty(t, executor.calls)
	require.Len(t, repo.updateCalls, 1)
	updated := repo.updateCalls[0]
	require.NotNil(t, updated.ExpiresAt)
	require.True(t, updated.ExpiresAt.After(now), "auto renewal must produce a dispatchable future expiry")
	require.Equal(t, time.Date(2026, 6, 15, 8, 30, 0, 0, time.UTC), updated.ExpiresAt.UTC())
	require.True(t, updated.IsSchedulable())
	require.Equal(t, AccountAutoRenewStatusSuccess, updated.Extra[accountAutoRenewStatusKey])
	require.Equal(t, previousExpiry.Format(time.RFC3339), updated.Extra[accountAutoRenewPreviousExpiresKey])
	require.Equal(t, updated.ExpiresAt.UTC().Format(time.RFC3339), updated.Extra[accountAutoRenewNextExpiresKey])
}

func TestAccountExpiryService_RunOnce_AutoRenewFailureWritesAuditOnly(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	previousExpiry := now.Add(-time.Hour)
	account := Account{
		ID:                 30,
		Status:             StatusActive,
		Schedulable:        true,
		AutoPauseOnExpired: false,
		AutoRenewEnabled:   true,
		AutoRenewPeriod:    "weekly",
		LifecycleState:     AccountLifecycleNormal,
		ExpiresAt:          &previousExpiry,
		Extra:              map[string]any{},
	}
	repo := &accountExpiryRepoStub{
		accounts:     []Account{account},
		accountsByID: map[int64]*Account{30: cloneAccountForBackgroundProbe(&account)},
	}
	executor := &accountExpiryProbeExecutorStub{}
	svc := NewAccountExpiryService(repo, executor, time.Minute)
	svc.SetNow(func() time.Time { return now })

	svc.runOnce(context.Background())

	require.Empty(t, executor.calls)
	require.Empty(t, repo.updateCalls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, int64(30), repo.updateExtraCalls[0].id)
	require.Equal(t, AccountAutoRenewStatusFailed, repo.updateExtraCalls[0].updates[accountAutoRenewStatusKey])
	require.Equal(t, "weekly", repo.updateExtraCalls[0].updates[accountAutoRenewPeriodKey])
	require.Equal(t, previousExpiry.Format(time.RFC3339), repo.updateExtraCalls[0].updates[accountAutoRenewPreviousExpiresKey])
	require.Nil(t, repo.updateExtraCalls[0].updates[accountAutoRenewNextExpiresKey])
	require.Contains(t, repo.updateExtraCalls[0].updates[accountAutoRenewSummaryKey], "auto_renew_period")
}

func TestAccountExpiryService_RunOnce_AutoRenewUpdateFailureStillRunsProbe(t *testing.T) {
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	previousExpiry := time.Date(2026, 5, 15, 8, 30, 0, 0, time.UTC)
	account := Account{
		ID:                 31,
		Status:             StatusActive,
		Schedulable:        true,
		AutoPauseOnExpired: true,
		AutoRenewEnabled:   true,
		AutoRenewPeriod:    AccountAutoRenewPeriodMonth,
		LifecycleState:     AccountLifecycleNormal,
		ExpiresAt:          &previousExpiry,
		Extra:              map[string]any{},
	}
	repo := &accountExpiryRepoStub{
		accounts:     []Account{account},
		accountsByID: map[int64]*Account{31: cloneAccountForBackgroundProbe(&account)},
		updateErr:    errors.New("update failed"),
	}
	executor := &accountExpiryProbeExecutorStub{
		result: &BackgroundAccountTestResult{
			Status:       "failed",
			ErrorMessage: "probe ran after auto renew update failure",
		},
	}
	svc := NewAccountExpiryService(repo, executor, time.Minute)
	svc.SetNow(func() time.Time { return now })

	svc.runOnce(context.Background())

	require.Len(t, executor.calls, 1)
	require.Equal(t, int64(31), executor.calls[0].AccountID)
	require.Empty(t, repo.updateCalls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountAutoRenewStatusFailed, repo.updateExtraCalls[0].updates[accountAutoRenewStatusKey])
	require.Equal(t, previousExpiry, account.ExpiresAt.UTC())
}

func TestShouldRunAccountExpiryProbe_SkipsWhileTemporaryPriorityStillActive(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-24 * time.Hour)
	checkedAt := now.Add(-1 * time.Hour)
	priorityUntil := now.Add(2 * time.Hour)
	account := &Account{
		ID:                 5,
		AutoPauseOnExpired: true,
		ExpiresAt:          &expiresAt,
		Extra: map[string]any{
			accountExpiryProbeCheckedAtKey:     checkedAt.Format(time.RFC3339),
			accountExpiryProbePriorityUntilKey: priorityUntil.Format(time.RFC3339),
		},
	}

	require.False(t, shouldRunAccountExpiryProbe(account, now))
}

func TestShouldRunAccountExpiryProbe_SkipsArchivedAccounts(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-1 * time.Hour)
	account := &Account{
		ID:                 6,
		AutoPauseOnExpired: true,
		ExpiresAt:          &expiresAt,
		LifecycleState:     AccountLifecycleArchived,
	}

	require.False(t, shouldRunAccountExpiryProbe(account, now))
}

func TestShouldRunAccountExpiryProbe_SkipsBlacklistedAccounts(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(-1 * time.Hour)
	account := &Account{
		ID:                 7,
		AutoPauseOnExpired: true,
		ExpiresAt:          &expiresAt,
		LifecycleState:     AccountLifecycleBlacklisted,
	}

	require.False(t, shouldRunAccountExpiryProbe(account, now))
}
