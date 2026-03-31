package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type googleBatchArchiveObjectRepository struct {
	sql sqlExecutor
}

func NewGoogleBatchArchiveObjectRepository(sqlDB *sql.DB) service.GoogleBatchArchiveObjectRepository {
	return &googleBatchArchiveObjectRepository{sql: sqlDB}
}

func (r *googleBatchArchiveObjectRepository) Upsert(ctx context.Context, object *service.GoogleBatchArchiveObject) error {
	if r == nil || r.sql == nil || object == nil {
		return nil
	}
	metadata := object.MetadataJSON
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = r.sql.ExecContext(ctx, `
		INSERT INTO google_batch_archive_objects (
			job_id,
			public_resource_kind,
			public_resource_name,
			execution_resource_name,
			storage_backend,
			relative_path,
			content_type,
			size_bytes,
			sha256,
			is_result_payload,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW(), NULL)
		ON CONFLICT (public_resource_kind, public_resource_name) WHERE deleted_at IS NULL
		DO UPDATE SET
			job_id = EXCLUDED.job_id,
			execution_resource_name = EXCLUDED.execution_resource_name,
			storage_backend = EXCLUDED.storage_backend,
			relative_path = EXCLUDED.relative_path,
			content_type = EXCLUDED.content_type,
			size_bytes = EXCLUDED.size_bytes,
			sha256 = EXCLUDED.sha256,
			is_result_payload = EXCLUDED.is_result_payload,
			metadata_json = EXCLUDED.metadata_json,
			updated_at = NOW(),
			deleted_at = NULL
	`, object.JobID, object.PublicResourceKind, object.PublicResourceName, object.ExecutionResourceName, object.StorageBackend, object.RelativePath, object.ContentType, object.SizeBytes, object.SHA256, object.IsResultPayload, metadataJSON)
	return err
}

func (r *googleBatchArchiveObjectRepository) GetByPublicResource(ctx context.Context, publicResourceKind string, publicResourceName string) (*service.GoogleBatchArchiveObject, error) {
	if r == nil || r.sql == nil {
		return nil, sql.ErrNoRows
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT
			id,
			job_id,
			public_resource_kind,
			public_resource_name,
			execution_resource_name,
			storage_backend,
			relative_path,
			content_type,
			size_bytes,
			sha256,
			is_result_payload,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM google_batch_archive_objects
		WHERE public_resource_kind = $1
			AND public_resource_name = $2
			AND deleted_at IS NULL
	`, strings.TrimSpace(publicResourceKind), strings.TrimSpace(publicResourceName))
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
	return scanGoogleBatchArchiveObjectRow(rows)
}

func (r *googleBatchArchiveObjectRepository) ListByJobID(ctx context.Context, jobID int64) ([]*service.GoogleBatchArchiveObject, error) {
	if r == nil || r.sql == nil {
		return nil, nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT
			id,
			job_id,
			public_resource_kind,
			public_resource_name,
			execution_resource_name,
			storage_backend,
			relative_path,
			content_type,
			size_bytes,
			sha256,
			is_result_payload,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM google_batch_archive_objects
		WHERE job_id = $1
			AND deleted_at IS NULL
		ORDER BY id ASC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*service.GoogleBatchArchiveObject
	for rows.Next() {
		item, err := scanGoogleBatchArchiveObjectRow(rows)
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

func (r *googleBatchArchiveObjectRepository) SoftDeleteByJobID(ctx context.Context, jobID int64) error {
	if r == nil || r.sql == nil {
		return nil
	}
	_, err := r.sql.ExecContext(ctx, `
		UPDATE google_batch_archive_objects
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE job_id = $1 AND deleted_at IS NULL
	`, jobID)
	return err
}

type googleBatchArchiveObjectScanner interface {
	Scan(dest ...any) error
}

func scanGoogleBatchArchiveObjectRow(scanner googleBatchArchiveObjectScanner) (*service.GoogleBatchArchiveObject, error) {
	var (
		item         service.GoogleBatchArchiveObject
		deletedAt    sql.NullTime
		metadataJSON []byte
	)
	if err := scanner.Scan(
		&item.ID,
		&item.JobID,
		&item.PublicResourceKind,
		&item.PublicResourceName,
		&item.ExecutionResourceName,
		&item.StorageBackend,
		&item.RelativePath,
		&item.ContentType,
		&item.SizeBytes,
		&item.SHA256,
		&item.IsResultPayload,
		&metadataJSON,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
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
