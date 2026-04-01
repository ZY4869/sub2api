package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type googleBatchQuotaReservationRepository struct {
	sql sqlExecutor
}

func NewGoogleBatchQuotaReservationRepository(sqlDB *sql.DB) service.GoogleBatchQuotaReservationRepository {
	return &googleBatchQuotaReservationRepository{sql: sqlDB}
}

func (r *googleBatchQuotaReservationRepository) Upsert(ctx context.Context, reservation *service.GoogleBatchQuotaReservation) error {
	if r == nil || r.sql == nil || reservation == nil {
		return nil
	}
	metadata := reservation.MetadataJSON
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = r.sql.ExecContext(ctx, `
		INSERT INTO google_batch_quota_reservations (
			provider_family,
			account_id,
			resource_name,
			model_family,
			reserved_tokens,
			status,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), NULL)
		ON CONFLICT (resource_name) WHERE deleted_at IS NULL
		DO UPDATE SET
			provider_family = EXCLUDED.provider_family,
			account_id = EXCLUDED.account_id,
			model_family = EXCLUDED.model_family,
			reserved_tokens = EXCLUDED.reserved_tokens,
			status = EXCLUDED.status,
			metadata_json = EXCLUDED.metadata_json,
			updated_at = NOW(),
			deleted_at = NULL
	`, reservation.ProviderFamily, reservation.AccountID, reservation.ResourceName, reservation.ModelFamily, reservation.ReservedTokens, reservation.Status, metadataJSON)
	return err
}

func (r *googleBatchQuotaReservationRepository) GetByResourceName(ctx context.Context, resourceName string) (*service.GoogleBatchQuotaReservation, error) {
	if r == nil || r.sql == nil {
		return nil, sql.ErrNoRows
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT
			id,
			provider_family,
			account_id,
			resource_name,
			model_family,
			reserved_tokens,
			status,
			metadata_json,
			created_at,
			updated_at,
			deleted_at
		FROM google_batch_quota_reservations
		WHERE resource_name = $1
			AND deleted_at IS NULL
	`, resourceName)
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
	return scanGoogleBatchQuotaReservationRow(rows)
}

func (r *googleBatchQuotaReservationRepository) ReleaseByResourceName(ctx context.Context, resourceName string, status string) error {
	if r == nil || r.sql == nil {
		return nil
	}
	if status == "" {
		status = service.GoogleBatchQuotaReservationStatusReleased
	}
	_, err := r.sql.ExecContext(ctx, `
		UPDATE google_batch_quota_reservations
		SET status = $2,
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE resource_name = $1
			AND deleted_at IS NULL
	`, resourceName, status)
	return err
}

func (r *googleBatchQuotaReservationRepository) SumActiveReservedTokens(ctx context.Context, providerFamily string, accountID int64, modelFamily string) (int64, error) {
	if r == nil || r.sql == nil {
		return 0, nil
	}
	var sum sql.NullInt64
	err := scanSingleRow(ctx, r.sql, `
		SELECT COALESCE(SUM(reserved_tokens), 0)
		FROM google_batch_quota_reservations
		WHERE provider_family = $1
			AND account_id = $2
			AND ($3 = '' OR model_family = $3)
			AND status = $4
			AND deleted_at IS NULL
	`, []any{providerFamily, accountID, modelFamily, service.GoogleBatchQuotaReservationStatusActive}, &sum)
	if err != nil {
		return 0, err
	}
	if !sum.Valid {
		return 0, nil
	}
	return sum.Int64, nil
}

type googleBatchQuotaReservationScanner interface {
	Scan(dest ...any) error
}

func scanGoogleBatchQuotaReservationRow(scanner googleBatchQuotaReservationScanner) (*service.GoogleBatchQuotaReservation, error) {
	var (
		reservation  service.GoogleBatchQuotaReservation
		metadataJSON []byte
		deletedAt    sql.NullTime
	)
	if err := scanner.Scan(
		&reservation.ID,
		&reservation.ProviderFamily,
		&reservation.AccountID,
		&reservation.ResourceName,
		&reservation.ModelFamily,
		&reservation.ReservedTokens,
		&reservation.Status,
		&metadataJSON,
		&reservation.CreatedAt,
		&reservation.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}
	if deletedAt.Valid {
		reservation.DeletedAt = &deletedAt.Time
	}
	if len(metadataJSON) > 0 {
		_ = json.Unmarshal(metadataJSON, &reservation.MetadataJSON)
	}
	if reservation.MetadataJSON == nil {
		reservation.MetadataJSON = map[string]any{}
	}
	return &reservation, nil
}
