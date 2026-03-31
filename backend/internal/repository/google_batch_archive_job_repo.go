package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type googleBatchArchiveJobRepository struct {
	sql sqlExecutor
}

func NewGoogleBatchArchiveJobRepository(sqlDB *sql.DB) service.GoogleBatchArchiveJobRepository {
	return &googleBatchArchiveJobRepository{sql: sqlDB}
}

func (r *googleBatchArchiveJobRepository) Upsert(ctx context.Context, job *service.GoogleBatchArchiveJob) error {
	if r == nil || r.sql == nil || job == nil {
		return nil
	}
	metadata := job.MetadataJSON
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = r.sql.ExecContext(ctx, `
		INSERT INTO google_batch_archive_jobs (
			public_batch_name,
			public_protocol,
			execution_provider_family,
			execution_batch_name,
			source_account_id,
			execution_account_id,
			api_key_id,
			group_id,
			user_id,
			requested_model,
			conversion_direction,
			state,
			official_expires_at,
			prefetch_due_at,
			last_public_result_access_at,
			next_poll_at,
			poll_attempts,
			archive_state,
			billing_settlement_state,
			retention_expires_at,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, NOW(), NOW(), NULL
		)
		ON CONFLICT (public_batch_name) WHERE deleted_at IS NULL
		DO UPDATE SET
			public_protocol = EXCLUDED.public_protocol,
			execution_provider_family = EXCLUDED.execution_provider_family,
			execution_batch_name = EXCLUDED.execution_batch_name,
			source_account_id = EXCLUDED.source_account_id,
			execution_account_id = EXCLUDED.execution_account_id,
			api_key_id = EXCLUDED.api_key_id,
			group_id = EXCLUDED.group_id,
			user_id = EXCLUDED.user_id,
			requested_model = EXCLUDED.requested_model,
			conversion_direction = EXCLUDED.conversion_direction,
			state = EXCLUDED.state,
			official_expires_at = EXCLUDED.official_expires_at,
			prefetch_due_at = EXCLUDED.prefetch_due_at,
			last_public_result_access_at = EXCLUDED.last_public_result_access_at,
			next_poll_at = EXCLUDED.next_poll_at,
			poll_attempts = EXCLUDED.poll_attempts,
			archive_state = EXCLUDED.archive_state,
			billing_settlement_state = EXCLUDED.billing_settlement_state,
			retention_expires_at = EXCLUDED.retention_expires_at,
			metadata_json = EXCLUDED.metadata_json,
			updated_at = NOW(),
			deleted_at = NULL
	`, job.PublicBatchName, job.PublicProtocol, job.ExecutionProviderFamily, job.ExecutionBatchName, job.SourceAccountID, job.ExecutionAccountID, job.APIKeyID, job.GroupID, job.UserID, job.RequestedModel, job.ConversionDirection, job.State, archiveNullTime(job.OfficialExpiresAt), archiveNullTime(job.PrefetchDueAt), archiveNullTime(job.LastPublicResultAccessAt), archiveNullTime(job.NextPollAt), job.PollAttempts, job.ArchiveState, job.BillingSettlementState, archiveNullTime(job.RetentionExpiresAt), metadataJSON)
	return err
}

func (r *googleBatchArchiveJobRepository) GetByID(ctx context.Context, id int64) (*service.GoogleBatchArchiveJob, error) {
	return r.getOne(ctx, `
		SELECT
			id,
			public_batch_name,
			public_protocol,
			execution_provider_family,
			execution_batch_name,
			source_account_id,
			execution_account_id,
			api_key_id,
			group_id,
			user_id,
			requested_model,
			conversion_direction,
			state,
			official_expires_at,
			prefetch_due_at,
			last_public_result_access_at,
			next_poll_at,
			poll_attempts,
			archive_state,
			billing_settlement_state,
			retention_expires_at,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM google_batch_archive_jobs
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
}

func (r *googleBatchArchiveJobRepository) GetByPublicBatchName(ctx context.Context, publicBatchName string) (*service.GoogleBatchArchiveJob, error) {
	return r.getOne(ctx, `
		SELECT
			id,
			public_batch_name,
			public_protocol,
			execution_provider_family,
			execution_batch_name,
			source_account_id,
			execution_account_id,
			api_key_id,
			group_id,
			user_id,
			requested_model,
			conversion_direction,
			state,
			official_expires_at,
			prefetch_due_at,
			last_public_result_access_at,
			next_poll_at,
			poll_attempts,
			archive_state,
			billing_settlement_state,
			retention_expires_at,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM google_batch_archive_jobs
		WHERE public_batch_name = $1 AND deleted_at IS NULL
	`, strings.TrimSpace(publicBatchName))
}

func (r *googleBatchArchiveJobRepository) GetByExecutionBatchName(ctx context.Context, executionBatchName string) (*service.GoogleBatchArchiveJob, error) {
	return r.getOne(ctx, `
		SELECT
			id,
			public_batch_name,
			public_protocol,
			execution_provider_family,
			execution_batch_name,
			source_account_id,
			execution_account_id,
			api_key_id,
			group_id,
			user_id,
			requested_model,
			conversion_direction,
			state,
			official_expires_at,
			prefetch_due_at,
			last_public_result_access_at,
			next_poll_at,
			poll_attempts,
			archive_state,
			billing_settlement_state,
			retention_expires_at,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM google_batch_archive_jobs
		WHERE execution_batch_name = $1 AND deleted_at IS NULL
	`, strings.TrimSpace(executionBatchName))
}

func (r *googleBatchArchiveJobRepository) ListDueForPoll(ctx context.Context, before time.Time, limit int) ([]*service.GoogleBatchArchiveJob, error) {
	return r.list(ctx, `
		SELECT
			id,
			public_batch_name,
			public_protocol,
			execution_provider_family,
			execution_batch_name,
			source_account_id,
			execution_account_id,
			api_key_id,
			group_id,
			user_id,
			requested_model,
			conversion_direction,
			state,
			official_expires_at,
			prefetch_due_at,
			last_public_result_access_at,
			next_poll_at,
			poll_attempts,
			archive_state,
			billing_settlement_state,
			retention_expires_at,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM google_batch_archive_jobs
		WHERE next_poll_at IS NOT NULL
			AND next_poll_at <= $1
			AND deleted_at IS NULL
		ORDER BY next_poll_at ASC, id ASC
		LIMIT $2
	`, before, normalizeArchiveListLimit(limit))
}

func (r *googleBatchArchiveJobRepository) ListDueForPrefetch(ctx context.Context, before time.Time, limit int) ([]*service.GoogleBatchArchiveJob, error) {
	return r.list(ctx, `
		SELECT
			id,
			public_batch_name,
			public_protocol,
			execution_provider_family,
			execution_batch_name,
			source_account_id,
			execution_account_id,
			api_key_id,
			group_id,
			user_id,
			requested_model,
			conversion_direction,
			state,
			official_expires_at,
			prefetch_due_at,
			last_public_result_access_at,
			next_poll_at,
			poll_attempts,
			archive_state,
			billing_settlement_state,
			retention_expires_at,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM google_batch_archive_jobs
		WHERE prefetch_due_at IS NOT NULL
			AND prefetch_due_at <= $1
			AND last_public_result_access_at IS NULL
			AND deleted_at IS NULL
		ORDER BY prefetch_due_at ASC, id ASC
		LIMIT $2
	`, before, normalizeArchiveListLimit(limit))
}

func (r *googleBatchArchiveJobRepository) ListExpiredForCleanup(ctx context.Context, before time.Time, limit int) ([]*service.GoogleBatchArchiveJob, error) {
	return r.list(ctx, `
		SELECT
			id,
			public_batch_name,
			public_protocol,
			execution_provider_family,
			execution_batch_name,
			source_account_id,
			execution_account_id,
			api_key_id,
			group_id,
			user_id,
			requested_model,
			conversion_direction,
			state,
			official_expires_at,
			prefetch_due_at,
			last_public_result_access_at,
			next_poll_at,
			poll_attempts,
			archive_state,
			billing_settlement_state,
			retention_expires_at,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM google_batch_archive_jobs
		WHERE retention_expires_at IS NOT NULL
			AND retention_expires_at <= $1
			AND deleted_at IS NULL
		ORDER BY retention_expires_at ASC, id ASC
		LIMIT $2
	`, before, normalizeArchiveListLimit(limit))
}

func (r *googleBatchArchiveJobRepository) TouchLastPublicResultAccess(ctx context.Context, id int64, accessedAt time.Time) error {
	if r == nil || r.sql == nil {
		return nil
	}
	_, err := r.sql.ExecContext(ctx, `
		UPDATE google_batch_archive_jobs
		SET last_public_result_access_at = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, accessedAt)
	return err
}

func (r *googleBatchArchiveJobRepository) SoftDelete(ctx context.Context, id int64) error {
	if r == nil || r.sql == nil {
		return nil
	}
	_, err := r.sql.ExecContext(ctx, `
		UPDATE google_batch_archive_jobs
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return err
}

func (r *googleBatchArchiveJobRepository) getOne(ctx context.Context, query string, args ...any) (*service.GoogleBatchArchiveJob, error) {
	if r == nil || r.sql == nil {
		return nil, sql.ErrNoRows
	}
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}
	return scanGoogleBatchArchiveJobRow(rows)
}

func (r *googleBatchArchiveJobRepository) list(ctx context.Context, query string, args ...any) ([]*service.GoogleBatchArchiveJob, error) {
	if r == nil || r.sql == nil {
		return nil, nil
	}
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*service.GoogleBatchArchiveJob
	for rows.Next() {
		item, err := scanGoogleBatchArchiveJobRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type googleBatchArchiveJobScanner interface {
	Scan(dest ...any) error
}

func scanGoogleBatchArchiveJobRow(scanner googleBatchArchiveJobScanner) (*service.GoogleBatchArchiveJob, error) {
	var (
		item         service.GoogleBatchArchiveJob
		apiKeyID     sql.NullInt64
		groupID      sql.NullInt64
		userID       sql.NullInt64
		officialAt   sql.NullTime
		prefetchAt   sql.NullTime
		lastAccessAt sql.NullTime
		nextPollAt   sql.NullTime
		retentionAt  sql.NullTime
		deletedAt    sql.NullTime
		metadataJSON []byte
	)
	if err := scanner.Scan(
		&item.ID,
		&item.PublicBatchName,
		&item.PublicProtocol,
		&item.ExecutionProviderFamily,
		&item.ExecutionBatchName,
		&item.SourceAccountID,
		&item.ExecutionAccountID,
		&apiKeyID,
		&groupID,
		&userID,
		&item.RequestedModel,
		&item.ConversionDirection,
		&item.State,
		&officialAt,
		&prefetchAt,
		&lastAccessAt,
		&nextPollAt,
		&item.PollAttempts,
		&item.ArchiveState,
		&item.BillingSettlementState,
		&retentionAt,
		&metadataJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}
	if apiKeyID.Valid {
		item.APIKeyID = &apiKeyID.Int64
	}
	if groupID.Valid {
		item.GroupID = &groupID.Int64
	}
	if userID.Valid {
		item.UserID = &userID.Int64
	}
	if officialAt.Valid {
		item.OfficialExpiresAt = &officialAt.Time
	}
	if prefetchAt.Valid {
		item.PrefetchDueAt = &prefetchAt.Time
	}
	if lastAccessAt.Valid {
		item.LastPublicResultAccessAt = &lastAccessAt.Time
	}
	if nextPollAt.Valid {
		item.NextPollAt = &nextPollAt.Time
	}
	if retentionAt.Valid {
		item.RetentionExpiresAt = &retentionAt.Time
	}
	if deletedAt.Valid {
		item.DeletedAt = &deletedAt.Time
	}
	if len(metadataJSON) > 0 {
		_ = json.Unmarshal(metadataJSON, &item.MetadataJSON)
	}
	if item.MetadataJSON == nil {
		item.MetadataJSON = map[string]any{}
	}
	return &item, nil
}

func normalizeArchiveListLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func archiveNullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}
