package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type imageBatchRepository struct {
	db *sql.DB
}

func NewImageBatchRepository(db *sql.DB) service.ImageBatchRepository {
	if db == nil {
		return nil
	}
	return &imageBatchRepository{db: db}
}

func (r *imageBatchRepository) CreateJob(ctx context.Context, job *service.ImageBatchJob, items []service.ImageBatchItem) error {
	if r == nil || r.db == nil || job == nil {
		return service.ErrImageBatchInvalidRequest
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	meta, _ := json.Marshal(nonNilMap(job.Metadata))
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO image_batch_jobs (
			id, user_id, api_key_id, group_id, provider, display_model_id, target_model_id,
			size, status, item_count, hold_request_id, idempotency_key_hash, request_fingerprint, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb)
	`, job.ID, job.UserID, job.APIKeyID, job.GroupID, job.Provider, job.DisplayModelID, job.TargetModelID,
		job.Size, string(job.Status), len(items), job.HoldRequestID, job.IdempotencyKeyHash, job.RequestFingerprint, string(meta)); err != nil {
		return err
	}
	for i := range items {
		item := items[i]
		itemMeta, _ := json.Marshal(nonNilMap(item.Metadata))
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO image_batch_items (job_id, custom_id, prompt, n, size, status, metadata)
			VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb)
		`, job.ID, item.CustomID, item.Prompt, item.N, item.Size, string(item.Status), string(itemMeta)); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	stored, err := r.GetJobForUser(ctx, job.UserID, job.ID)
	if err != nil {
		return err
	}
	*job = *stored
	return nil
}

func (r *imageBatchRepository) GetGroupSettings(ctx context.Context, groupID int64) (*service.ImageBatchGroupSettings, error) {
	var out service.ImageBatchGroupSettings
	err := r.db.QueryRowContext(ctx, `
		SELECT image_batch_enabled, image_batch_allowed_providers, image_batch_allowed_models,
			image_batch_max_items, image_batch_max_download_bytes, image_batch_download_concurrency
		FROM groups
		WHERE id = $1 AND deleted_at IS NULL
	`, groupID).Scan(
		&out.Enabled,
		pq.Array(&out.AllowedProviders),
		pq.Array(&out.AllowedModels),
		&out.MaxItems,
		&out.MaxDownloadBytes,
		&out.DownloadConcurrency,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return &service.ImageBatchGroupSettings{}, nil
	}
	if isGroupImageBatchSettingsMissingError(err) {
		return &service.ImageBatchGroupSettings{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *imageBatchRepository) GetJobForUser(ctx context.Context, userID int64, jobID string) (*service.ImageBatchJob, error) {
	return r.getJob(ctx, "user_id = $2", jobID, userID)
}

func (r *imageBatchRepository) GetJobForAPIKey(ctx context.Context, apiKeyID int64, jobID string) (*service.ImageBatchJob, error) {
	return r.getJob(ctx, "api_key_id = $2", jobID, apiKeyID)
}

func (r *imageBatchRepository) getJob(ctx context.Context, ownerClause string, jobID string, ownerID int64) (*service.ImageBatchJob, error) {
	query := imageBatchJobSelectSQL() + " WHERE id = $1 AND " + ownerClause + " AND deleted_at IS NULL"
	row := r.db.QueryRowContext(ctx, query, jobID, ownerID)
	job, err := scanImageBatchJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrImageBatchNotFound
	}
	return job, err
}

func (r *imageBatchRepository) ListJobsForUser(ctx context.Context, userID int64, limit int, before time.Time) ([]service.ImageBatchJob, error) {
	args := []any{userID}
	query := imageBatchJobSelectSQL() + " WHERE user_id = $1 AND deleted_at IS NULL"
	if !before.IsZero() {
		args = append(args, before)
		query += " AND created_at < $2"
	}
	query += " ORDER BY created_at DESC LIMIT " + intLiteral(limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.ImageBatchJob, 0)
	for rows.Next() {
		job, err := scanImageBatchJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *job)
	}
	return out, rows.Err()
}

func nonNilMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	return in
}

func intLiteral(v int) string {
	if v <= 0 {
		return "50"
	}
	if v > 1000 {
		return "1000"
	}
	return strconv.Itoa(v)
}
