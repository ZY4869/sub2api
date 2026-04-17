package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type usageRepairRepository struct {
	sql sqlExecutor
}

func NewUsageRepairRepository(sqlDB *sql.DB) service.UsageRepairRepository {
	return &usageRepairRepository{sql: sqlDB}
}

func (r *usageRepairRepository) CreateTask(ctx context.Context, task *service.UsageRepairTask) error {
	if task == nil {
		return nil
	}
	query := `
		INSERT INTO usage_repair_tasks (
			kind,
			days,
			status,
			created_by,
			processed_rows,
			repaired_rows,
			skipped_rows
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	return scanSingleRow(ctx, r.sql, query, []any{
		task.Kind,
		task.Days,
		task.Status,
		task.CreatedBy,
		task.ProcessedRows,
		task.RepairedRows,
		task.SkippedRows,
	}, &task.ID, &task.CreatedAt, &task.UpdatedAt)
}

func (r *usageRepairRepository) ListTasks(ctx context.Context, params pagination.PaginationParams) ([]service.UsageRepairTask, *pagination.PaginationResult, error) {
	var total int64
	if err := scanSingleRow(ctx, r.sql, "SELECT COUNT(*) FROM usage_repair_tasks", nil, &total); err != nil {
		return nil, nil, err
	}
	if total == 0 {
		return []service.UsageRepairTask{}, paginationResultFromTotal(0, params), nil
	}

	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, kind, days, status, created_by, processed_rows, repaired_rows, skipped_rows,
			error_message, canceled_by, canceled_at, started_at, finished_at, created_at, updated_at
		FROM usage_repair_tasks
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	tasks := make([]service.UsageRepairTask, 0)
	for rows.Next() {
		task, err := scanUsageRepairTask(rows)
		if err != nil {
			return nil, nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return tasks, paginationResultFromTotal(total, params), nil
}

func (r *usageRepairRepository) ClaimNextPendingTask(ctx context.Context, staleRunningAfterSeconds int64) (*service.UsageRepairTask, error) {
	if staleRunningAfterSeconds <= 0 {
		staleRunningAfterSeconds = int64((30 * time.Minute).Seconds())
	}
	query := `
		WITH next AS (
			SELECT id
			FROM usage_repair_tasks
			WHERE status = $1
				OR (
					status = $2
					AND started_at IS NOT NULL
					AND started_at < NOW() - ($3 * interval '1 second')
				)
			ORDER BY created_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE usage_repair_tasks AS tasks
		SET status = $4,
			started_at = NOW(),
			finished_at = NULL,
			error_message = NULL,
			processed_rows = 0,
			repaired_rows = 0,
			skipped_rows = 0,
			updated_at = NOW()
		FROM next
		WHERE tasks.id = next.id
		RETURNING tasks.id, tasks.kind, tasks.days, tasks.status, tasks.created_by,
			tasks.processed_rows, tasks.repaired_rows, tasks.skipped_rows,
			tasks.error_message, tasks.canceled_by, tasks.canceled_at,
			tasks.started_at, tasks.finished_at, tasks.created_at, tasks.updated_at
	`
	rows, err := r.sql.QueryContext(ctx, query,
		service.UsageRepairStatusPending,
		service.UsageRepairStatusRunning,
		staleRunningAfterSeconds,
		service.UsageRepairStatusRunning,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	task, err := scanUsageRepairTask(rows)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *usageRepairRepository) GetTaskStatus(ctx context.Context, taskID int64) (string, error) {
	var status string
	if err := scanSingleRow(ctx, r.sql, "SELECT status FROM usage_repair_tasks WHERE id = $1", []any{taskID}, &status); err != nil {
		return "", err
	}
	return status, nil
}

func (r *usageRepairRepository) UpdateTaskProgress(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE usage_repair_tasks
		SET processed_rows = $1,
			repaired_rows = $2,
			skipped_rows = $3,
			updated_at = NOW()
		WHERE id = $4
	`, processedRows, repairedRows, skippedRows, taskID)
	return err
}

func (r *usageRepairRepository) CancelTask(ctx context.Context, taskID int64, canceledBy int64) (bool, error) {
	var id int64
	err := scanSingleRow(ctx, r.sql, `
		UPDATE usage_repair_tasks
		SET status = $1,
			canceled_by = $3,
			canceled_at = NOW(),
			finished_at = NOW(),
			error_message = NULL,
			updated_at = NOW()
		WHERE id = $2
			AND status IN ($4, $5)
		RETURNING id
	`, []any{
		service.UsageRepairStatusCanceled,
		taskID,
		canceledBy,
		service.UsageRepairStatusPending,
		service.UsageRepairStatusRunning,
	}, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *usageRepairRepository) MarkTaskSucceeded(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE usage_repair_tasks
		SET status = $1,
			processed_rows = $2,
			repaired_rows = $3,
			skipped_rows = $4,
			finished_at = NOW(),
			updated_at = NOW()
		WHERE id = $5
	`, service.UsageRepairStatusSucceeded, processedRows, repairedRows, skippedRows, taskID)
	return err
}

func (r *usageRepairRepository) MarkTaskFailed(ctx context.Context, taskID, processedRows, repairedRows, skippedRows int64, errorMsg string) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE usage_repair_tasks
		SET status = $1,
			processed_rows = $2,
			repaired_rows = $3,
			skipped_rows = $4,
			error_message = $5,
			finished_at = NOW(),
			updated_at = NOW()
		WHERE id = $6
	`, service.UsageRepairStatusFailed, processedRows, repairedRows, skippedRows, errorMsg, taskID)
	return err
}

func (r *usageRepairRepository) ApplyClaudeRequestMetadataPatch(ctx context.Context, usageID int64, patch service.UsageRepairTaskPatch) (bool, error) {
	setClauses := make([]string, 0, 4)
	changeClauses := make([]string, 0, 4)
	args := make([]any, 0, 5)

	if patch.InboundEndpoint != nil && strings.TrimSpace(*patch.InboundEndpoint) != "" {
		args = append(args, strings.TrimSpace(*patch.InboundEndpoint))
		placeholder := "$" + itoa(len(args))
		setClauses = append(setClauses, "inbound_endpoint = CASE WHEN COALESCE(inbound_endpoint, '') = '' THEN "+placeholder+" ELSE inbound_endpoint END")
		changeClauses = append(changeClauses, "COALESCE(inbound_endpoint, '') = ''")
	}
	if patch.UpstreamEndpoint != nil && strings.TrimSpace(*patch.UpstreamEndpoint) != "" {
		args = append(args, strings.TrimSpace(*patch.UpstreamEndpoint))
		placeholder := "$" + itoa(len(args))
		setClauses = append(setClauses, "upstream_endpoint = CASE WHEN COALESCE(upstream_endpoint, '') = '' THEN "+placeholder+" ELSE upstream_endpoint END")
		changeClauses = append(changeClauses, "COALESCE(upstream_endpoint, '') = ''")
	}
	if patch.ThinkingEnabled != nil {
		args = append(args, *patch.ThinkingEnabled)
		placeholder := "$" + itoa(len(args))
		setClauses = append(setClauses, "thinking_enabled = CASE WHEN thinking_enabled IS NULL THEN "+placeholder+" ELSE thinking_enabled END")
		changeClauses = append(changeClauses, "thinking_enabled IS NULL")
	}
	if patch.ReasoningEffort != nil && strings.TrimSpace(*patch.ReasoningEffort) != "" {
		args = append(args, strings.TrimSpace(*patch.ReasoningEffort))
		placeholder := "$" + itoa(len(args))
		setClauses = append(setClauses, "reasoning_effort = CASE WHEN COALESCE(reasoning_effort, '') = '' THEN "+placeholder+" ELSE reasoning_effort END")
		changeClauses = append(changeClauses, "COALESCE(reasoning_effort, '') = ''")
	}
	if len(setClauses) == 0 {
		return false, nil
	}

	args = append(args, usageID)
	query := `
		UPDATE usage_logs
		SET ` + strings.Join(setClauses, ",\n\t\t\t") + `
		WHERE id = $` + itoa(len(args)) + `
			AND (` + strings.Join(changeClauses, " OR ") + `)
		RETURNING id
	`

	var updatedID int64
	err := scanSingleRow(ctx, r.sql, query, args, &updatedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func scanUsageRepairTask(scanner interface{ Scan(dest ...any) error }) (service.UsageRepairTask, error) {
	task := service.UsageRepairTask{}
	var (
		errMsg     sql.NullString
		canceledBy sql.NullInt64
		canceledAt sql.NullTime
		startedAt  sql.NullTime
		finishedAt sql.NullTime
	)
	if err := scanner.Scan(
		&task.ID,
		&task.Kind,
		&task.Days,
		&task.Status,
		&task.CreatedBy,
		&task.ProcessedRows,
		&task.RepairedRows,
		&task.SkippedRows,
		&errMsg,
		&canceledBy,
		&canceledAt,
		&startedAt,
		&finishedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return service.UsageRepairTask{}, err
	}
	if errMsg.Valid {
		task.ErrorMsg = &errMsg.String
	}
	if canceledBy.Valid {
		value := canceledBy.Int64
		task.CanceledBy = &value
	}
	if canceledAt.Valid {
		task.CanceledAt = &canceledAt.Time
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		task.FinishedAt = &finishedAt.Time
	}
	return task, nil
}
