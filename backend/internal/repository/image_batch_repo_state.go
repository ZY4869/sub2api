package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *imageBatchRepository) TryCancelJob(ctx context.Context, userID int64, jobID string, reason string) (*service.ImageBatchJob, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE image_batch_jobs
		SET status = $3, cancelled_at = NOW(), friendly_error = $4, updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
			AND status NOT IN ('completed','failed','cancelled','output_deleted')
	`, jobID, userID, string(service.ImageBatchJobCancelled), reason)
	if err := rowsAffectedOrConflict(res, err); err != nil {
		return nil, err
	}
	_, _ = r.db.ExecContext(ctx, `
		UPDATE image_batch_items
		SET status = $2, friendly_error = $3, updated_at = NOW()
		WHERE job_id = $1 AND status = 'pending'
	`, jobID, string(service.ImageBatchItemCancelled), reason)
	_ = r.RefreshJobCounts(ctx, jobID)
	return r.GetJobForUser(ctx, userID, jobID)
}

func (r *imageBatchRepository) MarkJobSubmitted(ctx context.Context, jobID string, providerBatchName string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE image_batch_jobs
		SET status = $2, provider_batch_name = $3, submitted_at = COALESCE(submitted_at, NOW()), updated_at = NOW()
		WHERE id = $1 AND status IN ('created','uploading')
	`, jobID, string(service.ImageBatchJobSubmitted), providerBatchName)
	return rowsAffectedOrConflict(res, err)
}

func (r *imageBatchRepository) MarkJobRunning(ctx context.Context, jobID string, providerResultFileName string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE image_batch_jobs
		SET status = $2,
			provider_result_file_name = COALESCE(NULLIF($3, ''), provider_result_file_name),
			started_at = COALESCE(started_at, NOW()),
			updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('completed','failed','cancelled','output_deleted')
	`, jobID, string(service.ImageBatchJobRunning), providerResultFileName)
	return rowsAffectedOrConflict(res, err)
}

func (r *imageBatchRepository) MarkJobIndexing(ctx context.Context, jobID string, providerResultFileName string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE image_batch_jobs
		SET status = $2,
			provider_result_file_name = COALESCE(NULLIF($3, ''), provider_result_file_name),
			updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('completed','failed','cancelled','output_deleted')
	`, jobID, string(service.ImageBatchJobIndexing), providerResultFileName)
	return rowsAffectedOrConflict(res, err)
}

func (r *imageBatchRepository) MarkJobSettling(ctx context.Context, jobID string, message string, errorID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE image_batch_jobs
		SET status = $2,
			friendly_error = $3,
			error_id = $4,
			updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('completed','failed','cancelled','output_deleted')
	`, jobID, string(service.ImageBatchJobSettling), message, errorID)
	return rowsAffectedOrConflict(res, err)
}

func (r *imageBatchRepository) IncrementJobSettlementRetry(ctx context.Context, jobID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		UPDATE image_batch_jobs
		SET metadata = jsonb_set(
				metadata,
				'{settlement_retry_count}',
				to_jsonb((
					CASE
						WHEN COALESCE(metadata->>'settlement_retry_count', '') ~ '^[0-9]+$'
						THEN (metadata->>'settlement_retry_count')::int
						ELSE 0
					END + 1
				)),
				true
			),
			updated_at = NOW()
		WHERE id = $1
			AND deleted_at IS NULL
			AND status = 'settling'
		RETURNING COALESCE(NULLIF(metadata->>'settlement_retry_count', '')::int, 0)
	`, jobID).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrImageBatchConflict
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *imageBatchRepository) FailJob(ctx context.Context, jobID string, message string, errorID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE image_batch_items
		SET status = $2, friendly_error = COALESCE(NULLIF(friendly_error, ''), $3),
			error_id = COALESCE(NULLIF(error_id, ''), $4), updated_at = NOW()
		WHERE job_id = $1 AND status = 'pending'
	`, jobID, string(service.ImageBatchItemFailed), message, errorID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE image_batch_jobs
		SET status = $2, friendly_error = $3, error_id = $4, completed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('completed','failed','cancelled','output_deleted')
	`, jobID, string(service.ImageBatchJobFailed), message, errorID)
	if err := rowsAffectedOrConflict(res, err); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return r.RefreshJobCounts(ctx, jobID)
}

func (r *imageBatchRepository) CompleteJob(ctx context.Context, jobID string, providerResultFileName string) error {
	if err := r.RefreshJobCounts(ctx, jobID); err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE image_batch_jobs
		SET status = $2,
			provider_result_file_name = COALESCE(NULLIF($3, ''), provider_result_file_name),
			friendly_error = '',
			error_id = '',
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('failed','cancelled','output_deleted')
	`, jobID, string(service.ImageBatchJobCompleted), providerResultFileName)
	return rowsAffectedOrConflict(res, err)
}

func rowsAffectedOrNotFound(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrImageBatchNotFound
	}
	return nil
}

func rowsAffectedOrConflict(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	if res == nil {
		return errors.New("missing sql result")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrImageBatchConflict
	}
	return nil
}
