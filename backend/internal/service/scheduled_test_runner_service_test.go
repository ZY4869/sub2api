//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type scheduledTestPlanRepoStub struct {
	updateCalls []scheduledTestPlanUpdateCall
}

type scheduledTestPlanUpdateCall struct {
	id                  int64
	lastRunAt           time.Time
	nextRunAt           time.Time
	consecutiveFailures int
	currentRetryCount   int
}

func (s *scheduledTestPlanRepoStub) Create(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	panic("unexpected Create call")
}

func (s *scheduledTestPlanRepoStub) GetByID(ctx context.Context, id int64) (*ScheduledTestPlan, error) {
	panic("unexpected GetByID call")
}

func (s *scheduledTestPlanRepoStub) ListByAccountID(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error) {
	panic("unexpected ListByAccountID call")
}

func (s *scheduledTestPlanRepoStub) ListDue(ctx context.Context, now time.Time) ([]*ScheduledTestPlan, error) {
	panic("unexpected ListDue call")
}

func (s *scheduledTestPlanRepoStub) Update(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	panic("unexpected Update call")
}

func (s *scheduledTestPlanRepoStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *scheduledTestPlanRepoStub) UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time, consecutiveFailures int, currentRetryCount int) error {
	s.updateCalls = append(s.updateCalls, scheduledTestPlanUpdateCall{
		id:                  id,
		lastRunAt:           lastRunAt,
		nextRunAt:           nextRunAt,
		consecutiveFailures: consecutiveFailures,
		currentRetryCount:   currentRetryCount,
	})
	return nil
}

type scheduledTestResultRepoStub struct {
	nextID  int64
	created []*ScheduledTestResult
}

func (s *scheduledTestResultRepoStub) Create(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error) {
	s.nextID++
	clone := *result
	clone.ID = s.nextID
	s.created = append(s.created, &clone)
	return &clone, nil
}

func (s *scheduledTestResultRepoStub) ListByPlanID(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	panic("unexpected ListByPlanID call")
}

func (s *scheduledTestResultRepoStub) PruneOldResults(ctx context.Context, planID int64, keepCount int) error {
	return nil
}

type scheduledTestExecutorStub struct {
	result *ScheduledTestResult
	err    error
}

func (s *scheduledTestExecutorStub) RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	if s.result == nil {
		return nil, s.err
	}
	clone := *s.result
	return &clone, s.err
}

type scheduledTestNotifierStub struct {
	messages []string
}

func (s *scheduledTestNotifierStub) SendNotification(ctx context.Context, message string) error {
	s.messages = append(s.messages, message)
	return nil
}

type scheduledTestAccountRepoStub struct {
	account *Account
}

func (s *scheduledTestAccountRepoStub) Create(ctx context.Context, account *Account) error {
	panic("unexpected Create call")
}

func (s *scheduledTestAccountRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if s.account != nil && s.account.ID == id {
		return s.account, nil
	}
	return nil, ErrAccountNotFound
}

func (s *scheduledTestAccountRepoStub) GetByIDs(ctx context.Context, ids []int64) ([]*Account, error) {
	panic("unexpected GetByIDs call")
}

func (s *scheduledTestAccountRepoStub) ExistsByID(ctx context.Context, id int64) (bool, error) {
	panic("unexpected ExistsByID call")
}

func (s *scheduledTestAccountRepoStub) GetByCRSAccountID(ctx context.Context, crsAccountID string) (*Account, error) {
	panic("unexpected GetByCRSAccountID call")
}

func (s *scheduledTestAccountRepoStub) FindByExtraField(ctx context.Context, key string, value any) ([]Account, error) {
	panic("unexpected FindByExtraField call")
}

func (s *scheduledTestAccountRepoStub) ListCRSAccountIDs(ctx context.Context) (map[string]int64, error) {
	panic("unexpected ListCRSAccountIDs call")
}

func (s *scheduledTestAccountRepoStub) Update(ctx context.Context, account *Account) error {
	panic("unexpected Update call")
}

func (s *scheduledTestAccountRepoStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *scheduledTestAccountRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *scheduledTestAccountRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, lifecycle string, privacyMode string) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *scheduledTestAccountRepoStub) GetStatusSummary(ctx context.Context, filters AccountStatusSummaryFilters) (*AccountStatusSummary, error) {
	panic("unexpected GetStatusSummary call")
}

func (s *scheduledTestAccountRepoStub) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	panic("unexpected ListByGroup call")
}

func (s *scheduledTestAccountRepoStub) ListActive(ctx context.Context) ([]Account, error) {
	panic("unexpected ListActive call")
}

func (s *scheduledTestAccountRepoStub) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected ListByPlatform call")
}

func (s *scheduledTestAccountRepoStub) UpdateLastUsed(ctx context.Context, id int64) error {
	panic("unexpected UpdateLastUsed call")
}

func (s *scheduledTestAccountRepoStub) BatchUpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	panic("unexpected BatchUpdateLastUsed call")
}

func (s *scheduledTestAccountRepoStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	panic("unexpected SetError call")
}

func (s *scheduledTestAccountRepoStub) ClearError(ctx context.Context, id int64) error {
	panic("unexpected ClearError call")
}

func (s *scheduledTestAccountRepoStub) SetSchedulable(ctx context.Context, id int64, schedulable bool) error {
	panic("unexpected SetSchedulable call")
}

func (s *scheduledTestAccountRepoStub) AutoPauseExpiredAccounts(ctx context.Context, now time.Time) (int64, error) {
	panic("unexpected AutoPauseExpiredAccounts call")
}

func (s *scheduledTestAccountRepoStub) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	panic("unexpected BindGroups call")
}

func (s *scheduledTestAccountRepoStub) ListSchedulable(ctx context.Context) ([]Account, error) {
	panic("unexpected ListSchedulable call")
}

func (s *scheduledTestAccountRepoStub) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupID call")
}

func (s *scheduledTestAccountRepoStub) ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected ListSchedulableByPlatform call")
}

func (s *scheduledTestAccountRepoStub) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupIDAndPlatform call")
}

func (s *scheduledTestAccountRepoStub) ListSchedulableByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	panic("unexpected ListSchedulableByPlatforms call")
}

func (s *scheduledTestAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupIDAndPlatforms call")
}

func (s *scheduledTestAccountRepoStub) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected ListSchedulableUngroupedByPlatform call")
}

func (s *scheduledTestAccountRepoStub) ListSchedulableUngroupedByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	panic("unexpected ListSchedulableUngroupedByPlatforms call")
}

func (s *scheduledTestAccountRepoStub) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	panic("unexpected SetRateLimited call")
}

func (s *scheduledTestAccountRepoStub) SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time) error {
	panic("unexpected SetModelRateLimit call")
}

func (s *scheduledTestAccountRepoStub) SetOverloaded(ctx context.Context, id int64, until time.Time) error {
	panic("unexpected SetOverloaded call")
}

func (s *scheduledTestAccountRepoStub) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	panic("unexpected SetTempUnschedulable call")
}

func (s *scheduledTestAccountRepoStub) ClearTempUnschedulable(ctx context.Context, id int64) error {
	panic("unexpected ClearTempUnschedulable call")
}

func (s *scheduledTestAccountRepoStub) ClearRateLimit(ctx context.Context, id int64) error {
	panic("unexpected ClearRateLimit call")
}

func (s *scheduledTestAccountRepoStub) ClearAntigravityQuotaScopes(ctx context.Context, id int64) error {
	panic("unexpected ClearAntigravityQuotaScopes call")
}

func (s *scheduledTestAccountRepoStub) ClearModelRateLimits(ctx context.Context, id int64) error {
	panic("unexpected ClearModelRateLimits call")
}

func (s *scheduledTestAccountRepoStub) UpdateSessionWindow(ctx context.Context, id int64, start, end *time.Time, status string) error {
	panic("unexpected UpdateSessionWindow call")
}

func (s *scheduledTestAccountRepoStub) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	panic("unexpected UpdateExtra call")
}

func (s *scheduledTestAccountRepoStub) BulkUpdate(ctx context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	panic("unexpected BulkUpdate call")
}

func (s *scheduledTestAccountRepoStub) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) error {
	panic("unexpected IncrementQuotaUsed call")
}

func (s *scheduledTestAccountRepoStub) ResetQuotaUsed(ctx context.Context, id int64) error {
	panic("unexpected ResetQuotaUsed call")
}

func (s *scheduledTestAccountRepoStub) MarkBlacklisted(ctx context.Context, id int64, reasonCode, reasonMessage string, blacklistedAt, purgeAt time.Time) error {
	panic("unexpected MarkBlacklisted call")
}

func (s *scheduledTestAccountRepoStub) RestoreBlacklisted(ctx context.Context, id int64) error {
	panic("unexpected RestoreBlacklisted call")
}

func (s *scheduledTestAccountRepoStub) ListBlacklistedIDs(ctx context.Context) ([]int64, error) {
	panic("unexpected ListBlacklistedIDs call")
}

func (s *scheduledTestAccountRepoStub) ListBlacklistedForPurge(ctx context.Context, now time.Time, limit int) ([]Account, error) {
	panic("unexpected ListBlacklistedForPurge call")
}

func TestScheduledTestRunnerService_RetriesIntermediateFailureWithoutNotification(t *testing.T) {
	planRepo := &scheduledTestPlanRepoStub{}
	resultRepo := &scheduledTestResultRepoStub{}
	scheduledSvc := NewScheduledTestService(planRepo, resultRepo)
	notifier := &scheduledTestNotifierStub{}
	accountRepo := &scheduledTestAccountRepoStub{}
	runner := NewScheduledTestRunnerService(
		planRepo,
		scheduledSvc,
		&scheduledTestExecutorStub{result: &ScheduledTestResult{Status: "failed", ErrorMessage: "upstream failed", FinishedAt: time.Now()}},
		nil,
		accountRepo,
		notifier,
		&config.Config{},
	)

	plan := &ScheduledTestPlan{
		ID:                   1,
		AccountID:            11,
		ModelID:              "gpt-4o",
		CronExpression:       "* * * * *",
		MaxResults:           20,
		NotifyPolicy:         ScheduledTestNotifyPolicyNone,
		RetryIntervalMinutes: 5,
		MaxRetries:           3,
		ConsecutiveFailures:  0,
		CurrentRetryCount:    0,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Len(t, planRepo.updateCalls, 1)
	update := planRepo.updateCalls[0]
	require.Equal(t, 1, update.consecutiveFailures)
	require.Equal(t, 1, update.currentRetryCount)
	require.WithinDuration(t, update.lastRunAt.Add(5*time.Minute), update.nextRunAt, 2*time.Second)
	require.Empty(t, notifier.messages)
}

func TestScheduledTestRunnerService_FinalFailureSendsFailureNotification(t *testing.T) {
	planRepo := &scheduledTestPlanRepoStub{}
	resultRepo := &scheduledTestResultRepoStub{}
	scheduledSvc := NewScheduledTestService(planRepo, resultRepo)
	notifier := &scheduledTestNotifierStub{}
	accountRepo := &scheduledTestAccountRepoStub{
		account: &Account{ID: 11, Name: "Primary OpenAI", Platform: PlatformOpenAI, Type: AccountTypeOAuth},
	}
	runner := NewScheduledTestRunnerService(
		planRepo,
		scheduledSvc,
		&scheduledTestExecutorStub{result: &ScheduledTestResult{Status: "failed", ErrorMessage: "quota exceeded", FinishedAt: time.Now()}},
		nil,
		accountRepo,
		notifier,
		&config.Config{},
	)

	plan := &ScheduledTestPlan{
		ID:                     2,
		AccountID:              11,
		ModelID:                "gpt-4o",
		CronExpression:         "* * * * *",
		MaxResults:             20,
		NotifyPolicy:           ScheduledTestNotifyPolicyFailureOnly,
		NotifyFailureThreshold: 3,
		RetryIntervalMinutes:   5,
		MaxRetries:             3,
		ConsecutiveFailures:    2,
		CurrentRetryCount:      2,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Len(t, planRepo.updateCalls, 1)
	update := planRepo.updateCalls[0]
	require.Equal(t, 3, update.consecutiveFailures)
	require.Equal(t, 0, update.currentRetryCount)
	require.True(t, update.nextRunAt.After(update.lastRunAt))
	require.Len(t, notifier.messages, 1)
	require.Contains(t, notifier.messages[0], "Primary OpenAI")
	require.Contains(t, notifier.messages[0], "quota exceeded")
}

func TestScheduledTestRunnerService_SuccessResetsCountersAndSendsAlwaysNotification(t *testing.T) {
	planRepo := &scheduledTestPlanRepoStub{}
	resultRepo := &scheduledTestResultRepoStub{}
	scheduledSvc := NewScheduledTestService(planRepo, resultRepo)
	notifier := &scheduledTestNotifierStub{}
	accountRepo := &scheduledTestAccountRepoStub{
		account: &Account{ID: 11, Name: "Recovered Account", Platform: PlatformGemini, Type: AccountTypeAPIKey},
	}
	runner := NewScheduledTestRunnerService(
		planRepo,
		scheduledSvc,
		&scheduledTestExecutorStub{result: &ScheduledTestResult{Status: "success", ResponseText: "ok", FinishedAt: time.Now()}},
		nil,
		accountRepo,
		notifier,
		&config.Config{},
	)

	plan := &ScheduledTestPlan{
		ID:                   3,
		AccountID:            11,
		ModelID:              "gemini-2.5-pro",
		CronExpression:       "* * * * *",
		MaxResults:           20,
		NotifyPolicy:         ScheduledTestNotifyPolicyAlways,
		RetryIntervalMinutes: 5,
		MaxRetries:           3,
		ConsecutiveFailures:  2,
		CurrentRetryCount:    1,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Len(t, planRepo.updateCalls, 1)
	update := planRepo.updateCalls[0]
	require.Equal(t, 0, update.consecutiveFailures)
	require.Equal(t, 0, update.currentRetryCount)
	require.True(t, update.nextRunAt.After(update.lastRunAt))
	require.Len(t, notifier.messages, 1)
	require.Contains(t, notifier.messages[0], "Recovered Account")
	require.Contains(t, notifier.messages[0], "success")
}
