package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	usageRepairWorkerName       = "usage_repair_worker"
	usageRepairDefaultDays      = 30
	usageRepairMaxDays          = 30
	usageRepairBatchSize        = 500
	usageRepairWorkerInterval   = 10 * time.Second
	usageRepairTaskTimeout      = 30 * time.Minute
)

type UsageRepairService struct {
	repo UsageRepairRepository

	running   int32
	startOnce sync.Once
	stopOnce  sync.Once

	timingWheel  *TimingWheelService
	workerCtx    context.Context
	workerCancel context.CancelFunc
}

func NewUsageRepairService(repo UsageRepairRepository, timingWheel *TimingWheelService) *UsageRepairService {
	workerCtx, workerCancel := context.WithCancel(context.Background())
	return &UsageRepairService{
		repo:         repo,
		timingWheel:  timingWheel,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}
}

func (s *UsageRepairService) Start() {
	if s == nil || s.repo == nil || s.timingWheel == nil {
		logger.LegacyPrintf("service.usage_repair", "[UsageRepair] not started (missing deps)")
		return
	}
	s.startOnce.Do(func() {
		s.timingWheel.ScheduleRecurring(usageRepairWorkerName, usageRepairWorkerInterval, s.runOnce)
		logger.LegacyPrintf("service.usage_repair", "[UsageRepair] started (interval=%s batch_size=%d timeout=%s)", usageRepairWorkerInterval, usageRepairBatchSize, usageRepairTaskTimeout)
	})
}

func (s *UsageRepairService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.workerCancel != nil {
			s.workerCancel()
		}
		if s.timingWheel != nil {
			s.timingWheel.Cancel(usageRepairWorkerName)
		}
		logger.LegacyPrintf("service.usage_repair", "[UsageRepair] stopped")
	})
}

func (s *UsageRepairService) ListTasks(ctx context.Context, params pagination.PaginationParams) ([]UsageRepairTask, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("repair service not ready")
	}
	return s.repo.ListTasks(ctx, params)
}

func (s *UsageRepairService) CreateTask(ctx context.Context, kind string, days int, createdBy int64) (*UsageRepairTask, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("repair service not ready")
	}
	if createdBy <= 0 {
		return nil, infraerrors.BadRequest("USAGE_REPAIR_INVALID_CREATOR", "invalid creator")
	}
	kind = normalizeUsageRepairKind(kind)
	if kind == "" {
		return nil, infraerrors.BadRequest("USAGE_REPAIR_INVALID_KIND", "unsupported repair task kind")
	}
	var err error
	days, err = normalizeUsageRepairDays(days)
	if err != nil {
		return nil, err
	}
	task := &UsageRepairTask{
		Kind:      kind,
		Days:      days,
		Status:    UsageRepairStatusPending,
		CreatedBy: createdBy,
	}
	logger.LegacyPrintf("service.usage_repair", "[UsageRepair] create_task requested: operator=%d kind=%s days=%d", createdBy, kind, days)
	if err := s.repo.CreateTask(ctx, task); err != nil {
		logger.LegacyPrintf("service.usage_repair", "[UsageRepair] create_task persist failed: operator=%d err=%v", createdBy, err)
		return nil, fmt.Errorf("create repair task: %w", err)
	}
	logger.LegacyPrintf("service.usage_repair", "[UsageRepair] create_task persisted: task=%d operator=%d kind=%s days=%d", task.ID, createdBy, task.Kind, task.Days)
	go s.runOnce()
	return task, nil
}

func (s *UsageRepairService) CancelTask(ctx context.Context, taskID int64, canceledBy int64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("repair service not ready")
	}
	if canceledBy <= 0 {
		return infraerrors.BadRequest("USAGE_REPAIR_INVALID_CANCELLER", "invalid canceller")
	}
	status, err := s.repo.GetTaskStatus(ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return infraerrors.New(http.StatusNotFound, "USAGE_REPAIR_TASK_NOT_FOUND", "repair task not found")
		}
		return err
	}
	if status == UsageRepairStatusCanceled {
		return nil
	}
	if status != UsageRepairStatusPending && status != UsageRepairStatusRunning {
		return infraerrors.New(http.StatusConflict, "USAGE_REPAIR_CANCEL_CONFLICT", "repair task cannot be canceled in current status")
	}
	ok, err := s.repo.CancelTask(ctx, taskID, canceledBy)
	if err != nil {
		return err
	}
	if ok {
		logger.LegacyPrintf("service.usage_repair", "[UsageRepair] cancel_task done: task=%d operator=%d", taskID, canceledBy)
		return nil
	}
	return infraerrors.New(http.StatusConflict, "USAGE_REPAIR_CANCEL_CONFLICT", "repair task cannot be canceled in current status")
}

func (s *UsageRepairService) runOnce() {
	if s == nil || s.repo == nil {
		return
	}
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&s.running, 0)

	parent := context.Background()
	if s.workerCtx != nil {
		parent = s.workerCtx
	}
	ctx, cancel := context.WithTimeout(parent, usageRepairTaskTimeout)
	defer cancel()

	task, err := s.repo.ClaimNextPendingTask(ctx, int64(usageRepairTaskTimeout.Seconds()))
	if err != nil || task == nil {
		return
	}
	s.executeTask(ctx, task)
}

func (s *UsageRepairService) executeTask(ctx context.Context, task *UsageRepairTask) {
	if task == nil {
		return
	}
	switch task.Kind {
	case UsageRepairKindClaudeRequestMetadata:
		s.executeClaudeMetadataRepair(ctx, task)
	default:
		s.markTaskFailed(task.ID, 0, 0, 0, fmt.Errorf("unsupported repair task kind: %s", task.Kind))
	}
}

func (s *UsageRepairService) executeClaudeMetadataRepair(ctx context.Context, task *UsageRepairTask) {
	since := time.Now().UTC().AddDate(0, 0, -task.Days)
	var afterID, processedRows, repairedRows, skippedRows int64

	for {
		if ctx != nil && ctx.Err() != nil {
			return
		}
		canceled, err := s.isTaskCanceled(ctx, task.ID)
		if err != nil {
			s.markTaskFailed(task.ID, processedRows, repairedRows, skippedRows, err)
			return
		}
		if canceled {
			return
		}

		candidates, err := s.repo.ListClaudeRequestMetadataCandidates(ctx, since, afterID, usageRepairBatchSize)
		if err != nil {
			s.markTaskFailed(task.ID, processedRows, repairedRows, skippedRows, err)
			return
		}
		if len(candidates) == 0 {
			break
		}

		for _, candidate := range candidates {
			afterID = candidate.UsageID
			processedRows++
			patch := buildClaudeUsageRepairPatch(candidate)
			if patch.IsEmpty() {
				skippedRows++
				continue
			}
			repaired, err := s.repo.ApplyClaudeRequestMetadataPatch(ctx, candidate.UsageID, patch)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				s.markTaskFailed(task.ID, processedRows, repairedRows, skippedRows, err)
				return
			}
			if repaired {
				repairedRows++
			} else {
				skippedRows++
			}
		}

		updateCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := s.repo.UpdateTaskProgress(updateCtx, task.ID, processedRows, repairedRows, skippedRows); err != nil {
			logger.LegacyPrintf("service.usage_repair", "[UsageRepair] task progress update failed: task=%d err=%v", task.ID, err)
		}
		cancel()

		if len(candidates) < usageRepairBatchSize {
			break
		}
	}

	updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.repo.MarkTaskSucceeded(updateCtx, task.ID, processedRows, repairedRows, skippedRows); err != nil {
		logger.LegacyPrintf("service.usage_repair", "[UsageRepair] update task succeeded failed: task=%d err=%v", task.ID, err)
	}
}

func (s *UsageRepairService) isTaskCanceled(ctx context.Context, taskID int64) (bool, error) {
	if s == nil || s.repo == nil {
		return false, fmt.Errorf("repair service not ready")
	}
	checkCtx := ctx
	if checkCtx == nil {
		checkCtx = context.Background()
	}
	status, err := s.repo.GetTaskStatus(checkCtx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return status == UsageRepairStatusCanceled, nil
}

func (s *UsageRepairService) markTaskFailed(taskID, processedRows, repairedRows, skippedRows int64, err error) {
	message := strings.TrimSpace(err.Error())
	if len(message) > 500 {
		message = message[:500]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if updateErr := s.repo.MarkTaskFailed(ctx, taskID, processedRows, repairedRows, skippedRows, message); updateErr != nil {
		logger.LegacyPrintf("service.usage_repair", "[UsageRepair] update task failed failed: task=%d err=%v", taskID, updateErr)
	}
}

func normalizeUsageRepairKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", UsageRepairKindClaudeRequestMetadata:
		return UsageRepairKindClaudeRequestMetadata
	default:
		return ""
	}
}

func normalizeUsageRepairDays(days int) (int, error) {
	if days <= 0 {
		return usageRepairDefaultDays, nil
	}
	if days > usageRepairMaxDays {
		return 0, infraerrors.BadRequest("USAGE_REPAIR_RANGE_TOO_LARGE", fmt.Sprintf("days cannot exceed %d", usageRepairMaxDays))
	}
	return days, nil
}
