package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *imageBatchRepository) ListItemsForUser(ctx context.Context, userID int64, jobID string, limit int) ([]service.ImageBatchItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.job_id, i.custom_id, i.prompt, i.n, i.size, i.status, i.output_count,
			i.friendly_error, i.error_id, i.metadata::text, i.created_at, i.updated_at
		FROM image_batch_items i
		JOIN image_batch_jobs j ON j.id = i.job_id
		WHERE j.user_id = $1 AND j.id = $2 AND j.deleted_at IS NULL
		ORDER BY i.id ASC
		LIMIT `+intLiteral(limit), userID, jobID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.ImageBatchItem, 0)
	for rows.Next() {
		item, err := scanImageBatchItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *imageBatchRepository) GetItemForUser(ctx context.Context, userID int64, jobID string, customID string) (*service.ImageBatchItem, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT i.id, i.job_id, i.custom_id, i.prompt, i.n, i.size, i.status, i.output_count,
			i.friendly_error, i.error_id, i.metadata::text, i.created_at, i.updated_at
		FROM image_batch_items i
		JOIN image_batch_jobs j ON j.id = i.job_id
		WHERE j.user_id = $1 AND j.id = $2 AND i.custom_id = $3 AND j.deleted_at IS NULL
	`, userID, jobID, customID)
	item, err := scanImageBatchItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrImageBatchNotFound
	}
	return item, err
}

func (r *imageBatchRepository) GetOutputForUser(ctx context.Context, userID int64, jobID string, customID string) (*service.ImageBatchOutput, error) {
	row := r.db.QueryRowContext(ctx, imageBatchOutputSelectSQL()+`
		JOIN image_batch_jobs j ON j.id = o.job_id
		WHERE j.user_id = $1 AND o.job_id = $2 AND o.custom_id = $3
			AND j.deleted_at IS NULL AND o.deleted_at IS NULL
		ORDER BY o.id ASC
		LIMIT 1
	`, userID, jobID, customID)
	output, err := scanImageBatchOutput(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrImageBatchOutputNotFound
	}
	return output, err
}

func (r *imageBatchRepository) ListOutputsForUser(ctx context.Context, userID int64, jobID string) ([]service.ImageBatchOutput, error) {
	rows, err := r.db.QueryContext(ctx, imageBatchOutputSelectSQL()+`
		JOIN image_batch_jobs j ON j.id = o.job_id
		WHERE j.user_id = $1 AND o.job_id = $2 AND j.deleted_at IS NULL AND o.deleted_at IS NULL
		ORDER BY o.custom_id ASC, o.id ASC
	`, userID, jobID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.ImageBatchOutput, 0)
	for rows.Next() {
		output, err := scanImageBatchOutput(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *output)
	}
	return out, rows.Err()
}

func (r *imageBatchRepository) SoftDeleteJob(ctx context.Context, userID int64, jobID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE image_batch_jobs
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, jobID, userID)
	return rowsAffectedOrNotFound(res, err)
}

func (r *imageBatchRepository) MarkOutputsDeleted(ctx context.Context, userID int64, jobID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE image_batch_outputs
		SET deleted_at = NOW()
		WHERE job_id = $1 AND deleted_at IS NULL
			AND EXISTS (SELECT 1 FROM image_batch_jobs WHERE id = $1 AND user_id = $2)
	`, jobID, userID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE image_batch_jobs
		SET status = $3, outputs_deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`, jobID, userID, string(service.ImageBatchJobOutputDeleted))
	if err := rowsAffectedOrNotFound(res, err); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *imageBatchRepository) CleanupExpiredOutputs(ctx context.Context, before time.Time, limit int) (int64, error) {
	if before.IsZero() {
		return 0, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	res, err := r.db.ExecContext(ctx, `
		WITH candidates AS (
			SELECT id
			FROM image_batch_jobs
			WHERE deleted_at IS NULL
				AND status = 'completed'
				AND outputs_deleted_at IS NULL
				AND completed_at IS NOT NULL
				AND completed_at < $1
			ORDER BY completed_at ASC
			LIMIT `+intLiteral(limit)+`
			FOR UPDATE SKIP LOCKED
		),
		deleted_outputs AS (
			UPDATE image_batch_outputs o
			SET deleted_at = NOW()
			FROM candidates c
			WHERE o.job_id = c.id AND o.deleted_at IS NULL
			RETURNING o.job_id
		)
		UPDATE image_batch_jobs j
		SET status = $2, outputs_deleted_at = NOW(), updated_at = NOW()
		FROM candidates c
		WHERE j.id = c.id
	`, before, string(service.ImageBatchJobOutputDeleted))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func imageBatchOutputSelectSQL() string {
	return `
		SELECT o.id, o.job_id, o.item_id, o.custom_id, o.content_type, o.storage_backend,
			o.content, o.size_bytes, o.sha256, o.metadata::text, o.created_at
		FROM image_batch_outputs o
	`
}

func jsonText(value map[string]any) string {
	raw, _ := json.Marshal(nonNilMap(value))
	return string(raw)
}
