package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *imageBatchRepository) ClaimNextRunnableJob(ctx context.Context, before time.Time) (*service.ImageBatchJob, []service.ImageBatchItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, imageBatchJobSelectSQL()+`
		WHERE deleted_at IS NULL
			AND status IN ('created','uploading','submitted','running','indexing','settling')
			AND updated_at <= $1
		ORDER BY
			CASE status
				WHEN 'uploading' THEN 0
				WHEN 'created' THEN 1
				WHEN 'submitted' THEN 2
				WHEN 'running' THEN 3
				WHEN 'indexing' THEN 4
				WHEN 'settling' THEN 5
				ELSE 6
			END,
			updated_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, before)
	job, err := scanImageBatchJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if job.Status == service.ImageBatchJobCreated {
		if _, err := tx.ExecContext(ctx, `
			UPDATE image_batch_jobs
			SET status = $2, updated_at = NOW()
			WHERE id = $1 AND status = 'created'
		`, job.ID, string(service.ImageBatchJobUploading)); err != nil {
			return nil, nil, err
		}
		job.Status = service.ImageBatchJobUploading
	}

	items, err := listImageBatchItemsForJob(ctx, tx, job.ID)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return job, items, nil
}

func (r *imageBatchRepository) UpsertOutput(ctx context.Context, output *service.ImageBatchOutput) error {
	if output == nil {
		return service.ErrImageBatchInvalidRequest
	}
	var itemID int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT id
		FROM image_batch_items
		WHERE job_id = $1 AND custom_id = $2
	`, output.JobID, output.CustomID).Scan(&itemID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrImageBatchNotFound
		}
		return err
	}
	output.ItemID = itemID
	metadata, _ := json.Marshal(nonNilMap(output.Metadata))
	return r.db.QueryRowContext(ctx, `
		INSERT INTO image_batch_outputs (
			job_id, item_id, custom_id, content_type, storage_backend, content, size_bytes, sha256, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)
		ON CONFLICT (job_id, custom_id, sha256)
			WHERE deleted_at IS NULL AND sha256 <> ''
		DO UPDATE SET
			content_type = EXCLUDED.content_type,
			storage_backend = EXCLUDED.storage_backend,
			content = EXCLUDED.content,
			size_bytes = EXCLUDED.size_bytes,
			metadata = EXCLUDED.metadata,
			deleted_at = NULL
		RETURNING id, created_at
	`, output.JobID, output.ItemID, output.CustomID, firstNonEmpty(output.ContentType, "image/png"),
		firstNonEmpty(output.StorageBackend, "db"), output.Content, output.SizeBytes, output.SHA256, string(metadata)).
		Scan(&output.ID, &output.CreatedAt)
}

func (r *imageBatchRepository) MarkItemResult(ctx context.Context, jobID string, customID string, status service.ImageBatchItemStatus, outputCount int, message string, errorID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE image_batch_items
		SET status = $3,
			output_count = $4,
			friendly_error = $5,
			error_id = $6,
			updated_at = NOW()
		WHERE job_id = $1 AND custom_id = $2
	`, jobID, customID, string(status), outputCount, message, errorID)
	return rowsAffectedOrNotFound(res, err)
}

func (r *imageBatchRepository) RefreshJobCounts(ctx context.Context, jobID string) error {
	res, err := r.db.ExecContext(ctx, `
		WITH counts AS (
			SELECT
				COUNT(*)::int AS total,
				COUNT(*) FILTER (WHERE status = 'success')::int AS success_count,
				COUNT(*) FILTER (WHERE status = 'failed')::int AS failed_count,
				COUNT(*) FILTER (WHERE status = 'cancelled')::int AS cancelled_count
			FROM image_batch_items
			WHERE job_id = $1
		)
		UPDATE image_batch_jobs j
		SET item_count = counts.total,
			success_count = counts.success_count,
			failed_count = counts.failed_count,
			cancelled_count = counts.cancelled_count,
			updated_at = NOW()
		FROM counts
		WHERE j.id = $1
	`, jobID)
	return rowsAffectedOrNotFound(res, err)
}

func (r *imageBatchRepository) AddEvent(ctx context.Context, jobID string, itemID *int64, eventType string, from string, to string, message string, requestID string, payload map[string]any) error {
	raw, _ := json.Marshal(nonNilMap(payload))
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO image_batch_events (
			job_id, item_id, event_type, status_from, status_to, message, request_id, payload
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb)
	`, jobID, itemID, eventType, from, to, message, requestID, string(raw))
	return err
}

func listImageBatchItemsForJob(ctx context.Context, tx *sql.Tx, jobID string) ([]service.ImageBatchItem, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, job_id, custom_id, prompt, n, size, status, output_count,
			friendly_error, error_id, metadata::text, created_at, updated_at
		FROM image_batch_items
		WHERE job_id = $1
		ORDER BY id ASC
	`, jobID)
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
