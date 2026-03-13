package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var scheduledTestCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// ScheduledTestService provides CRUD operations for scheduled test plans and results.
type ScheduledTestService struct {
	planRepo   ScheduledTestPlanRepository
	resultRepo ScheduledTestResultRepository
}

// NewScheduledTestService creates a new ScheduledTestService.
func NewScheduledTestService(
	planRepo ScheduledTestPlanRepository,
	resultRepo ScheduledTestResultRepository,
) *ScheduledTestService {
	return &ScheduledTestService{
		planRepo:   planRepo,
		resultRepo: resultRepo,
	}
}

// CreatePlan validates the cron expression, computes next_run_at, and persists the plan.
func (s *ScheduledTestService) CreatePlan(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	normalizeScheduledTestPlan(plan)
	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun

	return s.planRepo.Create(ctx, plan)
}

// GetPlan retrieves a plan by ID.
func (s *ScheduledTestService) GetPlan(ctx context.Context, id int64) (*ScheduledTestPlan, error) {
	return s.planRepo.GetByID(ctx, id)
}

// ListPlansByAccount returns all plans for a given account.
func (s *ScheduledTestService) ListPlansByAccount(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error) {
	return s.planRepo.ListByAccountID(ctx, accountID)
}

// UpdatePlan validates cron and updates the plan.
func (s *ScheduledTestService) UpdatePlan(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	normalizeScheduledTestPlan(plan)
	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	plan.NextRunAt = &nextRun

	return s.planRepo.Update(ctx, plan)
}

// DeletePlan removes a plan and its results (via CASCADE).
func (s *ScheduledTestService) DeletePlan(ctx context.Context, id int64) error {
	return s.planRepo.Delete(ctx, id)
}

// ListResults returns the most recent results for a plan.
func (s *ScheduledTestService) ListResults(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.resultRepo.ListByPlanID(ctx, planID, limit)
}

// SaveResult inserts a result and prunes old entries beyond maxResults.
func (s *ScheduledTestService) SaveResult(ctx context.Context, planID int64, maxResults int, result *ScheduledTestResult) (*ScheduledTestResult, error) {
	result.PlanID = planID
	created, err := s.resultRepo.Create(ctx, result)
	if err != nil {
		return nil, err
	}
	if err := s.resultRepo.PruneOldResults(ctx, planID, maxResults); err != nil {
		return created, err
	}
	return created, nil
}

func computeNextRun(cronExpr string, from time.Time) (time.Time, error) {
	sched, err := scheduledTestCronParser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(from), nil
}

func normalizeScheduledTestPlan(plan *ScheduledTestPlan) {
	if plan == nil {
		return
	}
	plan.ModelID = strings.TrimSpace(plan.ModelID)
	plan.CronExpression = strings.TrimSpace(plan.CronExpression)
	if plan.MaxResults <= 0 {
		plan.MaxResults = 50
	}
	switch strings.TrimSpace(plan.NotifyPolicy) {
	case ScheduledTestNotifyPolicyAlways, ScheduledTestNotifyPolicyFailureOnly:
		plan.NotifyPolicy = strings.TrimSpace(plan.NotifyPolicy)
	default:
		plan.NotifyPolicy = ScheduledTestNotifyPolicyNone
	}
	if plan.NotifyFailureThreshold <= 0 {
		plan.NotifyFailureThreshold = 3
	}
	if plan.RetryIntervalMinutes <= 0 {
		plan.RetryIntervalMinutes = 5
	}
	if plan.MaxRetries <= 0 {
		plan.MaxRetries = 3
	}
	if plan.ConsecutiveFailures < 0 {
		plan.ConsecutiveFailures = 0
	}
	if plan.CurrentRetryCount < 0 {
		plan.CurrentRetryCount = 0
	}
}
