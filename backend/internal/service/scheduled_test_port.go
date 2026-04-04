package service

import (
	"context"
	"time"
)

const (
	ScheduledTestNotifyPolicyNone        = "none"
	ScheduledTestNotifyPolicyAlways      = "always"
	ScheduledTestNotifyPolicyFailureOnly = "failure_only"
	ScheduledTestModelInputModeCatalog   = "catalog"
	ScheduledTestModelInputModeManual    = "manual"
)

// ScheduledTestPlan represents a scheduled test plan domain model.
type ScheduledTestPlan struct {
	ID                     int64      `json:"id"`
	AccountID              int64      `json:"account_id"`
	ModelID                string     `json:"model_id"`
	ModelInputMode         string     `json:"model_input_mode,omitempty"`
	ManualModelID          string     `json:"manual_model_id,omitempty"`
	RequestAlias           string     `json:"request_alias,omitempty"`
	SourceProtocol         string     `json:"source_protocol,omitempty"`
	CronExpression         string     `json:"cron_expression"`
	Enabled                bool       `json:"enabled"`
	MaxResults             int        `json:"max_results"`
	AutoRecover            bool       `json:"auto_recover"`
	NotifyPolicy           string     `json:"notify_policy"`
	NotifyFailureThreshold int        `json:"notify_failure_threshold"`
	RetryIntervalMinutes   int        `json:"retry_interval_minutes"`
	MaxRetries             int        `json:"max_retries"`
	ConsecutiveFailures    int        `json:"consecutive_failures"`
	CurrentRetryCount      int        `json:"current_retry_count"`
	LastRunAt              *time.Time `json:"last_run_at"`
	NextRunAt              *time.Time `json:"next_run_at"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func (p *ScheduledTestPlan) EffectiveModelID() string {
	if p == nil {
		return ""
	}
	if p.ModelInputMode == ScheduledTestModelInputModeManual {
		return p.ManualModelID
	}
	return p.ModelID
}

// ScheduledTestResult represents a single test execution result.
type ScheduledTestResult struct {
	ID           int64     `json:"id"`
	PlanID       int64     `json:"plan_id"`
	Status       string    `json:"status"`
	ResponseText string    `json:"response_text"`
	ErrorMessage string    `json:"error_message"`
	LatencyMs    int64     `json:"latency_ms"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type BackgroundAccountTestResult struct {
	Status                  string    `json:"status"`
	ResponseText            string    `json:"response_text"`
	ErrorMessage            string    `json:"error_message"`
	LatencyMs               int64     `json:"latency_ms"`
	StartedAt               time.Time `json:"started_at"`
	FinishedAt              time.Time `json:"finished_at"`
	ResolvedModelID         string    `json:"resolved_model_id,omitempty"`
	ResolvedPlatform        string    `json:"resolved_platform,omitempty"`
	ResolvedSourceProtocol  string    `json:"resolved_source_protocol,omitempty"`
	BlacklistAdviceDecision string    `json:"blacklist_advice_decision,omitempty"`
	CurrentLifecycleState   string    `json:"current_lifecycle_state,omitempty"`
}

type ScheduledTestExecutionInput struct {
	AccountID      int64
	ModelID        string
	SourceProtocol string
	RequestAlias   string
	Prompt         string
	TestMode       string
}

// ScheduledTestPlanRepository defines the data access interface for test plans.
type ScheduledTestPlanRepository interface {
	Create(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error)
	GetByID(ctx context.Context, id int64) (*ScheduledTestPlan, error)
	ListByAccountID(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error)
	ListDue(ctx context.Context, now time.Time) ([]*ScheduledTestPlan, error)
	Update(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error)
	Delete(ctx context.Context, id int64) error
	UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time, consecutiveFailures int, currentRetryCount int) error
}

// ScheduledTestResultRepository defines the data access interface for test results.
type ScheduledTestResultRepository interface {
	Create(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error)
	ListByPlanID(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error)
	PruneOldResults(ctx context.Context, planID int64, keepCount int) error
}
