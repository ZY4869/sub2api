package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	UsageRepairKindClaudeRequestMetadata = "claude_request_metadata"

	UsageRepairStatusPending   = "pending"
	UsageRepairStatusRunning   = "running"
	UsageRepairStatusSucceeded = "succeeded"
	UsageRepairStatusFailed    = "failed"
	UsageRepairStatusCanceled  = "canceled"
)

type UsageRepairTask struct {
	ID            int64
	Kind          string
	Days          int
	Status        string
	CreatedBy     int64
	ProcessedRows int64
	RepairedRows  int64
	SkippedRows   int64
	ErrorMsg      *string
	CanceledBy    *int64
	CanceledAt    *time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type UsageRepairTaskPatch struct {
	InboundEndpoint  *string
	UpstreamEndpoint *string
	ThinkingEnabled  *bool
	ReasoningEffort  *string
}

func (p UsageRepairTaskPatch) IsEmpty() bool {
	return p.InboundEndpoint == nil &&
		p.UpstreamEndpoint == nil &&
		p.ThinkingEnabled == nil &&
		p.ReasoningEffort == nil
}

type ClaudeUsageRepairCandidate struct {
	UsageID             int64
	RequestID           string
	CreatedAt           time.Time
	Model               string
	RequestedModel      string
	UpstreamModel       string
	InboundEndpoint     *string
	UpstreamEndpoint    *string
	ThinkingEnabled     *bool
	ReasoningEffort     *string
	TraceRoutePath      string
	TraceUpstreamPath   string
	TraceHasThinking    *bool
	TraceInboundJSON    string
	TraceNormalizedJSON string
}

type UsageRepairRepository interface {
	CreateTask(ctx context.Context, task *UsageRepairTask) error
	ListTasks(ctx context.Context, params pagination.PaginationParams) ([]UsageRepairTask, *pagination.PaginationResult, error)
	ClaimNextPendingTask(ctx context.Context, staleRunningAfterSeconds int64) (*UsageRepairTask, error)
	GetTaskStatus(ctx context.Context, taskID int64) (string, error)
	UpdateTaskProgress(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64) error
	CancelTask(ctx context.Context, taskID int64, canceledBy int64) (bool, error)
	MarkTaskSucceeded(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64) error
	MarkTaskFailed(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64, errorMsg string) error
	ListClaudeRequestMetadataCandidates(ctx context.Context, since time.Time, afterID int64, limit int) ([]ClaudeUsageRepairCandidate, error)
	ApplyClaudeRequestMetadataPatch(ctx context.Context, usageID int64, patch UsageRepairTaskPatch) (bool, error)
}
