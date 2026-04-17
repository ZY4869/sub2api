package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type usageRepairRepoStub struct {
	created         []*UsageRepairTask
	listTasks       []UsageRepairTask
	listResult      *pagination.PaginationResult
	statusByID      map[int64]string
	candidates      []ClaudeUsageRepairCandidate
	appliedPatches  []UsageRepairTaskPatch
	succeededTaskID int64
	succeededRows   struct {
		processed int64
		repaired  int64
		skipped   int64
	}
}

func (s *usageRepairRepoStub) CreateTask(ctx context.Context, task *UsageRepairTask) error {
	if task == nil {
		return nil
	}
	if task.ID == 0 {
		task.ID = int64(len(s.created) + 1)
	}
	now := time.Now().UTC()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	clone := *task
	s.created = append(s.created, &clone)
	if s.statusByID == nil {
		s.statusByID = map[int64]string{}
	}
	s.statusByID[task.ID] = task.Status
	return nil
}

func (s *usageRepairRepoStub) ListTasks(ctx context.Context, params pagination.PaginationParams) ([]UsageRepairTask, *pagination.PaginationResult, error) {
	if s.listResult == nil {
		s.listResult = &pagination.PaginationResult{Total: int64(len(s.listTasks)), Page: params.Page, PageSize: params.PageSize}
	}
	return s.listTasks, s.listResult, nil
}

func (s *usageRepairRepoStub) ClaimNextPendingTask(ctx context.Context, staleRunningAfterSeconds int64) (*UsageRepairTask, error) {
	return nil, nil
}

func (s *usageRepairRepoStub) GetTaskStatus(ctx context.Context, taskID int64) (string, error) {
	if s.statusByID == nil {
		return "", sql.ErrNoRows
	}
	status, ok := s.statusByID[taskID]
	if !ok {
		return "", sql.ErrNoRows
	}
	return status, nil
}

func (s *usageRepairRepoStub) UpdateTaskProgress(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64) error {
	return nil
}

func (s *usageRepairRepoStub) CancelTask(ctx context.Context, taskID int64, canceledBy int64) (bool, error) {
	if s.statusByID == nil {
		s.statusByID = map[int64]string{}
	}
	status := s.statusByID[taskID]
	if status != UsageRepairStatusPending && status != UsageRepairStatusRunning {
		return false, nil
	}
	s.statusByID[taskID] = UsageRepairStatusCanceled
	return true, nil
}

func (s *usageRepairRepoStub) MarkTaskSucceeded(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64) error {
	s.succeededTaskID = taskID
	s.succeededRows.processed = processedRows
	s.succeededRows.repaired = repairedRows
	s.succeededRows.skipped = skippedRows
	if s.statusByID == nil {
		s.statusByID = map[int64]string{}
	}
	s.statusByID[taskID] = UsageRepairStatusSucceeded
	return nil
}

func (s *usageRepairRepoStub) MarkTaskFailed(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64, errorMsg string) error {
	if s.statusByID == nil {
		s.statusByID = map[int64]string{}
	}
	s.statusByID[taskID] = UsageRepairStatusFailed
	return nil
}

func (s *usageRepairRepoStub) ListClaudeRequestMetadataCandidates(ctx context.Context, since time.Time, afterID int64, limit int) ([]ClaudeUsageRepairCandidate, error) {
	if afterID > 0 {
		return []ClaudeUsageRepairCandidate{}, nil
	}
	return s.candidates, nil
}

func (s *usageRepairRepoStub) ApplyClaudeRequestMetadataPatch(ctx context.Context, usageID int64, patch UsageRepairTaskPatch) (bool, error) {
	s.appliedPatches = append(s.appliedPatches, patch)
	return true, nil
}

var _ UsageRepairRepository = (*usageRepairRepoStub)(nil)

func TestUsageRepairServiceCreateTaskRejectsRangeOver30(t *testing.T) {
	t.Parallel()

	svc := NewUsageRepairService(&usageRepairRepoStub{}, nil)
	task, err := svc.CreateTask(context.Background(), UsageRepairKindClaudeRequestMetadata, 31, 7)
	require.Nil(t, task)
	require.Error(t, err)
	require.Equal(t, "USAGE_REPAIR_RANGE_TOO_LARGE", infraerrors.Reason(err))
}

func TestBuildClaudeUsageRepairPatchUsesTraceAndExplicitEffortOnly(t *testing.T) {
	t.Parallel()

	hasThinking := true
	candidate := ClaudeUsageRepairCandidate{
		Model:             "claude-sonnet-4",
		TraceRoutePath:    "v1/messages",
		TraceHasThinking:  &hasThinking,
		TraceInboundJSON:  `{"output_config":{"effort":"high"}}`,
		TraceUpstreamPath: "",
	}

	patch := buildClaudeUsageRepairPatch(candidate)
	require.NotNil(t, patch.InboundEndpoint)
	require.Equal(t, EndpointMessages, *patch.InboundEndpoint)
	require.NotNil(t, patch.UpstreamEndpoint)
	require.Equal(t, EndpointMessages, *patch.UpstreamEndpoint)
	require.NotNil(t, patch.ThinkingEnabled)
	require.True(t, *patch.ThinkingEnabled)
	require.NotNil(t, patch.ReasoningEffort)
	require.Equal(t, "high", *patch.ReasoningEffort)
}

func TestBuildClaudeUsageRepairPatchDoesNotInferReasoningWithoutExplicitOutputConfig(t *testing.T) {
	t.Parallel()

	candidate := ClaudeUsageRepairCandidate{
		Model:            "claude-opus-4",
		TraceRoutePath:   "/v1/messages",
		TraceInboundJSON: `{"thinking":{"type":"enabled","budget_tokens":1024}}`,
	}

	patch := buildClaudeUsageRepairPatch(candidate)
	require.NotNil(t, patch.InboundEndpoint)
	require.Nil(t, patch.ReasoningEffort)
}

func TestUsageRepairServiceExecuteClaudeMetadataRepairSkipsEmptyPatch(t *testing.T) {
	t.Parallel()

	enabled := true
	repo := &usageRepairRepoStub{
		statusByID: map[int64]string{9: UsageRepairStatusRunning},
		candidates: []ClaudeUsageRepairCandidate{
			{
				UsageID:          101,
				Model:            "claude-3-7-sonnet",
				InboundEndpoint:  usageRepairStrPtr(EndpointMessages),
				UpstreamEndpoint: usageRepairStrPtr(EndpointMessages),
				ThinkingEnabled:  &enabled,
				ReasoningEffort:  usageRepairStrPtr("medium"),
			},
		},
	}
	svc := NewUsageRepairService(repo, nil)

	svc.executeClaudeMetadataRepair(context.Background(), &UsageRepairTask{
		ID:     9,
		Kind:   UsageRepairKindClaudeRequestMetadata,
		Days:   30,
		Status: UsageRepairStatusRunning,
	})

	require.Equal(t, int64(9), repo.succeededTaskID)
	require.Equal(t, int64(1), repo.succeededRows.processed)
	require.Equal(t, int64(0), repo.succeededRows.repaired)
	require.Equal(t, int64(1), repo.succeededRows.skipped)
	require.Empty(t, repo.appliedPatches)
}

func usageRepairStrPtr(value string) *string {
	return &value
}
