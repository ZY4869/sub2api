package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type upstreamResourceBindingRepository struct {
	sql sqlExecutor
}

func NewUpstreamResourceBindingRepository(sqlDB *sql.DB) service.UpstreamResourceBindingRepository {
	return &upstreamResourceBindingRepository{sql: sqlDB}
}

func (r *upstreamResourceBindingRepository) Upsert(ctx context.Context, binding *service.UpstreamResourceBinding) error {
	if r == nil || r.sql == nil || binding == nil {
		return nil
	}
	metadata := binding.MetadataJSON
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = r.sql.ExecContext(ctx, `
		INSERT INTO upstream_resource_bindings (
			resource_kind,
			resource_name,
			provider_family,
			account_id,
			api_key_id,
			group_id,
			user_id,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), NULL)
		ON CONFLICT (resource_kind, resource_name)
		DO UPDATE SET
			provider_family = EXCLUDED.provider_family,
			account_id = EXCLUDED.account_id,
			api_key_id = EXCLUDED.api_key_id,
			group_id = EXCLUDED.group_id,
			user_id = EXCLUDED.user_id,
			metadata_json = EXCLUDED.metadata_json,
			updated_at = NOW(),
			deleted_at = NULL
	`, binding.ResourceKind, binding.ResourceName, binding.ProviderFamily, binding.AccountID, binding.APIKeyID, binding.GroupID, binding.UserID, metadataJSON)
	return err
}

func (r *upstreamResourceBindingRepository) Get(ctx context.Context, resourceKind, resourceName string) (*service.UpstreamResourceBinding, error) {
	if r == nil || r.sql == nil {
		return nil, sql.ErrNoRows
	}
	query := `
		SELECT
			id,
			resource_kind,
			resource_name,
			provider_family,
			account_id,
			api_key_id,
			group_id,
			user_id,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM upstream_resource_bindings
		WHERE resource_kind = $1
			AND resource_name = $2
			AND deleted_at IS NULL
	`
	return scanUpstreamResourceBinding(ctx, r.sql, query, []any{resourceKind, resourceName})
}

func (r *upstreamResourceBindingRepository) GetByNames(ctx context.Context, resourceKind string, resourceNames []string) (result []*service.UpstreamResourceBinding, err error) {
	if r == nil || r.sql == nil || len(resourceNames) == 0 {
		return nil, nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT
			id,
			resource_kind,
			resource_name,
			provider_family,
			account_id,
			api_key_id,
			group_id,
			user_id,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM upstream_resource_bindings
		WHERE resource_kind = $1
			AND resource_name = ANY($2)
			AND deleted_at IS NULL
	`, resourceKind, pq.Array(resourceNames))
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	result = make([]*service.UpstreamResourceBinding, 0)
	for rows.Next() {
		binding, scanErr := scanUpstreamResourceBindingRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, binding)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *upstreamResourceBindingRepository) SoftDelete(ctx context.Context, resourceKind, resourceName string) error {
	if r == nil || r.sql == nil {
		return nil
	}
	_, err := r.sql.ExecContext(ctx, `
		UPDATE upstream_resource_bindings
		SET deleted_at = NOW(),
			updated_at = NOW()
		WHERE resource_kind = $1
			AND resource_name = $2
			AND deleted_at IS NULL
	`, resourceKind, resourceName)
	return err
}

func scanUpstreamResourceBinding(ctx context.Context, q sqlQueryer, query string, args []any) (binding *service.UpstreamResourceBinding, err error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, sql.ErrNoRows
	}
	binding, err = scanUpstreamResourceBindingRow(rows)
	return binding, err
}

type upstreamResourceBindingScanner interface {
	Scan(dest ...any) error
}

func scanUpstreamResourceBindingRow(scanner upstreamResourceBindingScanner) (*service.UpstreamResourceBinding, error) {
	var (
		binding      service.UpstreamResourceBinding
		apiKeyID     sql.NullInt64
		groupID      sql.NullInt64
		userID       sql.NullInt64
		deletedAt    sql.NullTime
		metadataJSON []byte
	)
	if err := scanner.Scan(
		&binding.ID,
		&binding.ResourceKind,
		&binding.ResourceName,
		&binding.ProviderFamily,
		&binding.AccountID,
		&apiKeyID,
		&groupID,
		&userID,
		&metadataJSON,
		&binding.CreatedAt,
		&binding.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}
	if apiKeyID.Valid {
		binding.APIKeyID = &apiKeyID.Int64
	}
	if groupID.Valid {
		binding.GroupID = &groupID.Int64
	}
	if userID.Valid {
		binding.UserID = &userID.Int64
	}
	if deletedAt.Valid {
		binding.DeletedAt = &deletedAt.Time
	}
	if len(metadataJSON) > 0 {
		_ = json.Unmarshal(metadataJSON, &binding.MetadataJSON)
	}
	if binding.MetadataJSON == nil {
		binding.MetadataJSON = map[string]any{}
	}
	return &binding, nil
}
