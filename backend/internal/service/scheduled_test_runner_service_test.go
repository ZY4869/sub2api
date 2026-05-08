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
	input  ScheduledTestExecutionInput
}

func (s *scheduledTestExecutorStub) RunTestBackground(ctx context.Context, input ScheduledTestExecutionInput) (*ScheduledTestResult, error) {
	s.input = input
	if s.result == nil {
		return nil, s.err
	}
	clone := *s.result
	return &clone, s.err
}

type scheduledTestUsageRecordingExecutorStub struct {
	input     ScheduledTestExecutionInput
	userRepo  *systemUsageUserRepoStub
	apiKeyRepo *systemUsageAPIKeyRepoStub
	usageRepo *systemUsageLogRepoStub
}

func (s *scheduledTestUsageRecordingExecutorStub) RunTestBackground(ctx context.Context, input ScheduledTestExecutionInput) (*ScheduledTestResult, error) {
	s.input = input
	accountTestSvc := &AccountTestService{
		userRepo:     s.userRepo,
		apiKeyRepo:   s.apiKeyRepo,
		usageLogRepo: s.usageRepo,
	}
	result := &BackgroundAccountTestResult{
		Status:          "success",
		LatencyMs:       321,
		StartedAt:       time.Date(2026, 5, 8, 9, 0, 0, 0, time.UTC),
		FinishedAt:      time.Date(2026, 5, 8, 9, 0, 1, 0, time.UTC),
		ResolvedModelID: "gpt-5.4",
	}
	if err := accountTestSvc.recordSystemUsage(ctx, input, result); err != nil {
		return nil, err
	}
	return &ScheduledTestResult{
		Status:       "success",
		ResponseText: "ok",
		LatencyMs:    result.LatencyMs,
		StartedAt:    result.StartedAt,
		FinishedAt:   result.FinishedAt,
	}, nil
}

type scheduledTestRealUsageExecutor struct {
	service *AccountTestService
	input   ScheduledTestExecutionInput
}

func (s *scheduledTestRealUsageExecutor) RunTestBackground(ctx context.Context, input ScheduledTestExecutionInput) (*ScheduledTestResult, error) {
	s.input = input
	return s.service.RunTestBackground(ctx, input)
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
	executor := &scheduledTestExecutorStub{result: &ScheduledTestResult{Status: "failed", ErrorMessage: "upstream failed", FinishedAt: time.Now()}}
	runner := NewScheduledTestRunnerService(
		planRepo,
		scheduledSvc,
		executor,
		nil,
		accountRepo,
		notifier,
		&config.Config{},
	)

	plan := &ScheduledTestPlan{
		ID:                   1,
		AccountID:            11,
		ModelID:              "gpt-4o-mini",
		ModelInputMode:       ScheduledTestModelInputModeManual,
		ManualModelID:        "gpt-4o",
		SourceProtocol:       "openai",
		RequestAlias:         "gpt-4o",
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
	require.Equal(t, int64(11), executor.input.AccountID)
	require.Equal(t, "gpt-4o", executor.input.ModelID)
	require.Equal(t, "openai", executor.input.SourceProtocol)
	require.Equal(t, "gpt-4o", executor.input.RequestAlias)
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

func TestScheduledTestRunnerService_InjectsGatewayDefaultsForMixedAccounts(t *testing.T) {
	planRepo := &scheduledTestPlanRepoStub{}
	resultRepo := &scheduledTestResultRepoStub{}
	scheduledSvc := NewScheduledTestService(planRepo, resultRepo)
	notifier := &scheduledTestNotifierStub{}
	accountRepo := &scheduledTestAccountRepoStub{
		account: &Account{
			ID:       11,
			Name:     "Mixed Gateway",
			Platform: PlatformProtocolGateway,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				"gateway_protocol":           GatewayProtocolMixed,
				"gateway_accepted_protocols": []string{PlatformOpenAI, PlatformAnthropic},
				"gateway_test_provider":      PlatformOpenAI,
				"gateway_test_model_id":      "gpt-5.4",
			},
		},
	}
	executor := &scheduledTestExecutorStub{result: &ScheduledTestResult{Status: "success", FinishedAt: time.Now()}}
	runner := NewScheduledTestRunnerService(
		planRepo,
		scheduledSvc,
		executor,
		nil,
		accountRepo,
		notifier,
		&config.Config{},
	)

	plan := &ScheduledTestPlan{
		ID:             4,
		AccountID:      11,
		ModelID:        "shared-model",
		CronExpression: "* * * * *",
		MaxResults:     20,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Equal(t, PlatformOpenAI, executor.input.TargetProvider)
	require.Equal(t, "gpt-5.4", executor.input.TargetModelID)
	require.Empty(t, executor.input.SourceProtocol)
}

func TestScheduledTestRunnerService_ExplicitSourceProtocolStillWinsOverGatewayDefaults(t *testing.T) {
	planRepo := &scheduledTestPlanRepoStub{}
	resultRepo := &scheduledTestResultRepoStub{}
	scheduledSvc := NewScheduledTestService(planRepo, resultRepo)
	notifier := &scheduledTestNotifierStub{}
	accountRepo := &scheduledTestAccountRepoStub{
		account: &Account{
			ID:       11,
			Name:     "Mixed Gateway",
			Platform: PlatformProtocolGateway,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				"gateway_protocol":           GatewayProtocolMixed,
				"gateway_accepted_protocols": []string{PlatformOpenAI, PlatformAnthropic},
				"gateway_test_provider":      PlatformOpenAI,
				"gateway_test_model_id":      "gpt-5.4",
			},
		},
	}
	executor := &scheduledTestExecutorStub{result: &ScheduledTestResult{Status: "success", FinishedAt: time.Now()}}
	runner := NewScheduledTestRunnerService(
		planRepo,
		scheduledSvc,
		executor,
		nil,
		accountRepo,
		notifier,
		&config.Config{},
	)

	plan := &ScheduledTestPlan{
		ID:             5,
		AccountID:      11,
		ModelID:        "claude-sonnet",
		SourceProtocol: PlatformAnthropic,
		CronExpression: "* * * * *",
		MaxResults:     20,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Equal(t, PlatformAnthropic, executor.input.SourceProtocol)
	require.Equal(t, PlatformOpenAI, executor.input.TargetProvider)
	require.Equal(t, "gpt-5.4", executor.input.TargetModelID)
}

func TestScheduledTestRunnerService_RunOnePlan_RecordsUsageLogWithScheduledOperationType(t *testing.T) {
	planRepo := &scheduledTestPlanRepoStub{}
	resultRepo := &scheduledTestResultRepoStub{}
	scheduledSvc := NewScheduledTestService(planRepo, resultRepo)
	notifier := &scheduledTestNotifierStub{}
	accountRepo := &scheduledTestAccountRepoStub{
		account: &Account{
			ID:          77,
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Concurrency: 1,
			Credentials: map[string]any{"api_key": "test-token", "base_url": "https://api.openai.com"},
		},
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
		accountRepo:  accountRepo,
		httpUpstream: upstream,
		cfg:          &config.Config{},
		userRepo:     userRepo,
		apiKeyRepo:   apiKeyRepo,
		usageLogRepo: usageRepo,
	}
	executor := &scheduledTestRealUsageExecutor{
		service: accountTestSvc,
	}
	runner := NewScheduledTestRunnerService(
		planRepo,
		scheduledSvc,
		executor,
		nil,
		accountRepo,
		notifier,
		&config.Config{},
	)

	plan := &ScheduledTestPlan{
		ID:             7,
		AccountID:      77,
		ModelID:        "gpt-5.4",
		CronExpression: "* * * * *",
		MaxResults:     20,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Equal(t, UsageOperationTypeScheduledTest, executor.input.OperationType)
	require.Len(t, usageRepo.logs, 1)
	require.NotNil(t, usageRepo.logs[0].OperationType)
	require.Equal(t, UsageOperationTypeScheduledTest, *usageRepo.logs[0].OperationType)
	require.Equal(t, int64(77), usageRepo.logs[0].AccountID)
	require.Equal(t, userRepo.user.ID, usageRepo.logs[0].UserID)
	require.Equal(t, apiKeyRepo.apiKey.ID, usageRepo.logs[0].APIKeyID)
	require.Zero(t, usageRepo.logs[0].ActualCost)
	require.NotNil(t, usageRepo.logs[0].BillingExemptReason)
	require.Equal(t, BillingExemptReasonAdminFree, *usageRepo.logs[0].BillingExemptReason)
}
