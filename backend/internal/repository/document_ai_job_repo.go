package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type documentAIJobRepository struct {
	sql sqlExecutor
}

type documentAIJobScanner interface {
	Scan(dest ...any) error
}

func NewDocumentAIJobRepository(sqlDB *sql.DB) service.DocumentAIJobRepository {
	return &documentAIJobRepository{sql: sqlDB}
}

func (r *documentAIJobRepository) Create(ctx context.Context, job *service.DocumentAIJob) error {
	if job == nil {
		return nil
	}
	query := `
		INSERT INTO document_ai_jobs (
			job_id,
			provider_job_id,
			provider_batch_id,
			account_id,
			user_id,
			api_key_id,
			group_id,
			mode,
			model,
			source_type,
			file_name,
			content_type,
			file_size,
			file_hash,
			status,
			provider_result_json,
			normalized_result_json,
			error_code,
			error_message,
			completed_at,
			last_polled_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
		RETURNING id, created_at, updated_at
	`
	return scanSingleRow(ctx, r.sql, query, []any{
		job.JobID,
		job.ProviderJobID,
		job.ProviderBatchID,
		job.AccountID,
		job.UserID,
		job.APIKeyID,
		job.GroupID,
		job.Mode,
		job.Model,
		job.SourceType,
		job.FileName,
		job.ContentType,
		job.FileSize,
		job.FileHash,
		job.Status,
		job.ProviderResultJSON,
		job.NormalizedResultJSON,
		job.ErrorCode,
		job.ErrorMessage,
		job.CompletedAt,
		job.LastPolledAt,
	}, &job.ID, &job.CreatedAt, &job.UpdatedAt)
}

func (r *documentAIJobRepository) GetByJobIDForUser(ctx context.Context, jobID string, userID int64) (*service.DocumentAIJob, error) {
	query := `
		SELECT id, job_id, provider_job_id, provider_batch_id, account_id, user_id, api_key_id, group_id,
			mode, model, source_type, file_name, content_type, file_size, file_hash, status,
			provider_result_json, normalized_result_json, error_code, error_message,
			created_at, updated_at, completed_at, last_polled_at
		FROM document_ai_jobs
		WHERE job_id = $1 AND user_id = $2
		LIMIT 1
	`
	rows, err := r.sql.QueryContext(ctx, query, strings.TrimSpace(jobID), userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}
	job, err := scanDocumentAIJob(rows)
	if err != nil {
		return nil, err
	}
	return &job, rows.Err()
}

func (r *documentAIJobRepository) UpdateAfterSubmit(ctx context.Context, jobID string, providerJobID, providerBatchID *string, status string, providerResultJSON *string) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE document_ai_jobs
		SET provider_job_id = COALESCE($1, provider_job_id),
			provider_batch_id = COALESCE($2, provider_batch_id),
			status = $3,
			provider_result_json = COALESCE($4::jsonb, provider_result_json),
			error_code = NULL,
			error_message = NULL,
			updated_at = NOW()
		WHERE job_id = $5
	`, providerJobID, providerBatchID, status, providerResultJSON, strings.TrimSpace(jobID))
	return err
}

func (r *documentAIJobRepository) ListPollable(ctx context.Context, limit int) ([]service.DocumentAIJob, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, job_id, provider_job_id, provider_batch_id, account_id, user_id, api_key_id, group_id,
			mode, model, source_type, file_name, content_type, file_size, file_hash, status,
			provider_result_json, normalized_result_json, error_code, error_message,
			created_at, updated_at, completed_at, last_polled_at
		FROM document_ai_jobs
		WHERE status IN ($1, $2)
		ORDER BY created_at ASC, id ASC
		LIMIT $3
	`, service.DocumentAIJobStatusPending, service.DocumentAIJobStatusRunning, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	jobs := make([]service.DocumentAIJob, 0, limit)
	for rows.Next() {
		job, err := scanDocumentAIJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *documentAIJobRepository) MarkRunning(ctx context.Context, jobID string, providerResultJSON *string) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE document_ai_jobs
		SET status = $1,
			provider_result_json = COALESCE($2::jsonb, provider_result_json),
			last_polled_at = NOW(),
			updated_at = NOW()
		WHERE job_id = $3
	`, service.DocumentAIJobStatusRunning, providerResultJSON, strings.TrimSpace(jobID))
	return err
}

func (r *documentAIJobRepository) MarkSucceeded(ctx context.Context, jobID string, providerResultJSON, normalizedResultJSON *string) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE document_ai_jobs
		SET status = $1,
			provider_result_json = COALESCE($2::jsonb, provider_result_json),
			normalized_result_json = COALESCE($3::jsonb, normalized_result_json),
			error_code = NULL,
			error_message = NULL,
			completed_at = NOW(),
			last_polled_at = NOW(),
			updated_at = NOW()
		WHERE job_id = $4
	`, service.DocumentAIJobStatusSucceeded, providerResultJSON, normalizedResultJSON, strings.TrimSpace(jobID))
	return err
}

func (r *documentAIJobRepository) MarkFailed(ctx context.Context, jobID string, providerResultJSON *string, errorCode, errorMessage string) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE document_ai_jobs
		SET status = $1,
			provider_result_json = COALESCE($2::jsonb, provider_result_json),
			error_code = NULLIF($3, ''),
			error_message = NULLIF($4, ''),
			completed_at = NOW(),
			last_polled_at = NOW(),
			updated_at = NOW()
		WHERE job_id = $5
	`, service.DocumentAIJobStatusFailed, providerResultJSON, strings.TrimSpace(errorCode), strings.TrimSpace(errorMessage), strings.TrimSpace(jobID))
	return err
}

func (r *documentAIJobRepository) TouchLastPolledAt(ctx context.Context, jobID string) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE document_ai_jobs
		SET last_polled_at = NOW(),
			updated_at = NOW()
		WHERE job_id = $1
	`, strings.TrimSpace(jobID))
	return err
}

func scanDocumentAIJob(scanner documentAIJobScanner) (service.DocumentAIJob, error) {
	job := service.DocumentAIJob{}
	var (
		providerJobID        sql.NullString
		providerBatchID      sql.NullString
		accountID            sql.NullInt64
		groupID              sql.NullInt64
		fileName             sql.NullString
		contentType          sql.NullString
		fileSize             sql.NullInt64
		fileHash             sql.NullString
		providerResultJSON   sql.NullString
		normalizedResultJSON sql.NullString
		errorCode            sql.NullString
		errorMessage         sql.NullString
		completedAt          sql.NullTime
		lastPolledAt         sql.NullTime
	)
	if err := scanner.Scan(
		&job.ID,
		&job.JobID,
		&providerJobID,
		&providerBatchID,
		&accountID,
		&job.UserID,
		&job.APIKeyID,
		&groupID,
		&job.Mode,
		&job.Model,
		&job.SourceType,
		&fileName,
		&contentType,
		&fileSize,
		&fileHash,
		&job.Status,
		&providerResultJSON,
		&normalizedResultJSON,
		&errorCode,
		&errorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
		&completedAt,
		&lastPolledAt,
	); err != nil {
		return service.DocumentAIJob{}, err
	}
	if providerJobID.Valid {
		job.ProviderJobID = &providerJobID.String
	}
	if providerBatchID.Valid {
		job.ProviderBatchID = &providerBatchID.String
	}
	if accountID.Valid {
		job.AccountID = &accountID.Int64
	}
	if groupID.Valid {
		job.GroupID = &groupID.Int64
	}
	if fileName.Valid {
		job.FileName = &fileName.String
	}
	if contentType.Valid {
		job.ContentType = &contentType.String
	}
	if fileSize.Valid {
		job.FileSize = &fileSize.Int64
	}
	if fileHash.Valid {
		job.FileHash = &fileHash.String
	}
	if providerResultJSON.Valid {
		job.ProviderResultJSON = &providerResultJSON.String
	}
	if normalizedResultJSON.Valid {
		job.NormalizedResultJSON = &normalizedResultJSON.String
	}
	if errorCode.Valid {
		job.ErrorCode = &errorCode.String
	}
	if errorMessage.Valid {
		job.ErrorMessage = &errorMessage.String
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if lastPolledAt.Valid {
		job.LastPolledAt = &lastPolledAt.Time
	}
	return job, nil
}
