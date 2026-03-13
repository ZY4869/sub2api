package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

const scheduledTestDefaultMaxWorkers = 10

type scheduledTestExecutor interface {
	RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
}

type scheduledTestNotificationSender interface {
	SendNotification(ctx context.Context, message string) error
}

// ScheduledTestRunnerService periodically scans due test plans and executes them.
type ScheduledTestRunnerService struct {
	planRepo       ScheduledTestPlanRepository
	scheduledSvc   *ScheduledTestService
	accountTestSvc scheduledTestExecutor
	rateLimitSvc   *RateLimitService
	accountRepo    AccountRepository
	notifier       scheduledTestNotificationSender
	cfg            *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewScheduledTestRunnerService creates a new runner.
func NewScheduledTestRunnerService(
	planRepo ScheduledTestPlanRepository,
	scheduledSvc *ScheduledTestService,
	accountTestSvc scheduledTestExecutor,
	rateLimitSvc *RateLimitService,
	accountRepo AccountRepository,
	notifier scheduledTestNotificationSender,
	cfg *config.Config,
) *ScheduledTestRunnerService {
	return &ScheduledTestRunnerService{
		planRepo:       planRepo,
		scheduledSvc:   scheduledSvc,
		accountTestSvc: accountTestSvc,
		rateLimitSvc:   rateLimitSvc,
		accountRepo:    accountRepo,
		notifier:       notifier,
		cfg:            cfg,
	}
}

// Start begins the cron ticker (every minute).
func (s *ScheduledTestRunnerService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(scheduledTestCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc("* * * * *", func() { s.runScheduled() })
		if err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] not started (invalid schedule): %v", err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] started (tick=every minute)")
	})
}

// Stop gracefully shuts down the cron scheduler.
func (s *ScheduledTestRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] cron stop timed out")
			}
		}
	})
}

func (s *ScheduledTestRunnerService) runScheduled() {
	// Delay 10s so execution lands at ~:10 of each minute instead of :00.
	time.Sleep(10 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now()
	plans, err := s.planRepo.ListDue(ctx, now)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] ListDue error: %v", err)
		return
	}
	if len(plans) == 0 {
		return
	}

	logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] found %d due plans", len(plans))

	sem := make(chan struct{}, scheduledTestDefaultMaxWorkers)
	var wg sync.WaitGroup

	for _, plan := range plans {
		sem <- struct{}{}
		wg.Add(1)
		go func(p *ScheduledTestPlan) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOnePlan(ctx, p)
		}(plan)
	}

	wg.Wait()
}

func (s *ScheduledTestRunnerService) runOnePlan(ctx context.Context, plan *ScheduledTestPlan) {
	result, err := s.accountTestSvc.RunTestBackground(ctx, plan.AccountID, plan.ModelID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d RunTestBackground error: %v", plan.ID, err)
		result = &ScheduledTestResult{
			Status:       "failed",
			ErrorMessage: err.Error(),
			StartedAt:    time.Now(),
			FinishedAt:   time.Now(),
		}
	}

	if result == nil {
		result = &ScheduledTestResult{
			Status:       "failed",
			ErrorMessage: "scheduled test returned no result",
			StartedAt:    time.Now(),
			FinishedAt:   time.Now(),
		}
	}

	savedResult, saveErr := s.scheduledSvc.SaveResult(ctx, plan.ID, plan.MaxResults, result)
	if saveErr != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d SaveResult error: %v", plan.ID, saveErr)
	}

	lastRunAt := time.Now()
	consecutiveFailures := plan.ConsecutiveFailures
	currentRetryCount := plan.CurrentRetryCount
	shouldRetry := false
	if result.Status == "success" {
		consecutiveFailures = 0
		currentRetryCount = 0
	} else {
		consecutiveFailures++
		currentRetryCount++
		if currentRetryCount < plan.MaxRetries {
			shouldRetry = true
		} else {
			currentRetryCount = 0
		}
	}

	// Auto-recover account if test succeeded and auto_recover is enabled.
	if result.Status == "success" && plan.AutoRecover {
		s.tryRecoverAccount(ctx, plan.AccountID, plan.ID)
	}

	nextRun, nextRunErr := s.computeNextRunAfterResult(plan, lastRunAt, shouldRetry)
	if nextRunErr != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d computeNextRun error: %v", plan.ID, nextRunErr)
		return
	}

	if shouldRetry {
		logger.LegacyPrintf(
			"service.scheduled_test_runner",
			"[ScheduledTestRunner] plan=%d account=%d scheduled retry %d/%d at %s",
			plan.ID,
			plan.AccountID,
			currentRetryCount,
			plan.MaxRetries,
			nextRun.Format(time.RFC3339),
		)
	}

	if err := s.planRepo.UpdateAfterRun(ctx, plan.ID, lastRunAt, nextRun, consecutiveFailures, currentRetryCount); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d UpdateAfterRun error: %v", plan.ID, err)
		return
	}

	if !shouldRetry && s.shouldSendNotification(plan, result, consecutiveFailures) {
		if notifyErr := s.sendNotification(ctx, plan, savedResult, result, consecutiveFailures, nextRun); notifyErr != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d final notification failed: %v", plan.ID, notifyErr)
		}
	}
}

// tryRecoverAccount attempts to recover an account from recoverable runtime state.
func (s *ScheduledTestRunnerService) tryRecoverAccount(ctx context.Context, accountID int64, planID int64) {
	if s.rateLimitSvc == nil {
		return
	}

	recovery, err := s.rateLimitSvc.RecoverAccountAfterSuccessfulTest(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover failed: %v", planID, err)
		return
	}
	if recovery == nil {
		return
	}

	if recovery.ClearedError {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d recovered from error status", planID, accountID)
	}
	if recovery.ClearedRateLimit {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d cleared rate-limit/runtime state", planID, accountID)
	}
}

func (s *ScheduledTestRunnerService) computeNextRunAfterResult(plan *ScheduledTestPlan, lastRunAt time.Time, shouldRetry bool) (time.Time, error) {
	if shouldRetry {
		return lastRunAt.Add(time.Duration(plan.RetryIntervalMinutes) * time.Minute), nil
	}
	return computeNextRun(plan.CronExpression, lastRunAt)
}

func (s *ScheduledTestRunnerService) shouldSendNotification(plan *ScheduledTestPlan, result *ScheduledTestResult, consecutiveFailures int) bool {
	if plan == nil || result == nil {
		return false
	}

	switch plan.NotifyPolicy {
	case ScheduledTestNotifyPolicyAlways:
		return true
	case ScheduledTestNotifyPolicyFailureOnly:
		return result.Status != "success" && consecutiveFailures >= plan.NotifyFailureThreshold
	default:
		return false
	}
}

func (s *ScheduledTestRunnerService) sendNotification(ctx context.Context, plan *ScheduledTestPlan, savedResult *ScheduledTestResult, result *ScheduledTestResult, consecutiveFailures int, nextRun time.Time) error {
	if s.notifier == nil {
		return fmt.Errorf("telegram notifier is not configured")
	}

	message := s.buildNotificationMessage(ctx, plan, savedResult, result, consecutiveFailures, nextRun)
	if err := s.notifier.SendNotification(ctx, message); err != nil {
		return err
	}

	resultID := int64(0)
	if savedResult != nil {
		resultID = savedResult.ID
	}
	logger.LegacyPrintf(
		"service.scheduled_test_runner",
		"[ScheduledTestRunner] plan=%d account=%d result=%d final notification sent status=%s",
		plan.ID,
		plan.AccountID,
		resultID,
		result.Status,
	)
	return nil
}

func (s *ScheduledTestRunnerService) buildNotificationMessage(ctx context.Context, plan *ScheduledTestPlan, savedResult *ScheduledTestResult, result *ScheduledTestResult, consecutiveFailures int, nextRun time.Time) string {
	accountName := fmt.Sprintf("Account #%d", plan.AccountID)
	accountType := "-"
	accountPlatform := "-"
	if s.accountRepo != nil {
		if account, err := s.accountRepo.GetByID(ctx, plan.AccountID); err == nil && account != nil {
			if account.Name != "" {
				accountName = account.Name
			}
			if account.Type != "" {
				accountType = account.Type
			}
			if account.Platform != "" {
				accountPlatform = account.Platform
			}
		}
	}

	resultID := int64(0)
	if savedResult != nil {
		resultID = savedResult.ID
	}

	errorMessage := strings.TrimSpace(result.ErrorMessage)
	if errorMessage == "" {
		errorMessage = "-"
	}

	return fmt.Sprintf(
		"Sub2API scheduled test notification\n\nAccount: %s\nAccount ID: %d\nPlatform / Type: %s / %s\nPlan ID: %d\nResult ID: %d\nModel: %s\nStatus: %s\nLatency: %d ms\nError: %s\nConsecutive failures: %d\nCompleted at: %s\nNext run: %s",
		accountName,
		plan.AccountID,
		accountPlatform,
		accountType,
		plan.ID,
		resultID,
		plan.ModelID,
		s.formatNotificationStatus(result.Status),
		result.LatencyMs,
		errorMessage,
		consecutiveFailures,
		s.formatNotificationTime(result.FinishedAt),
		s.formatNotificationTime(nextRun),
	)
}

func (s *ScheduledTestRunnerService) formatNotificationStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "success":
		return "success"
	case "failed":
		return "failed"
	case "":
		return "unknown"
	default:
		return status
	}
}

func (s *ScheduledTestRunnerService) formatNotificationTime(ts time.Time) string {
	loc := time.Local
	if s.cfg != nil && strings.TrimSpace(s.cfg.Timezone) != "" {
		if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
			loc = parsed
		}
	}
	return ts.In(loc).Format("2006-01-02 15:04:05")
}
