package repository

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *apiKeyRepository) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) (float64, error) {
	updated, err := r.client.APIKey.UpdateOneID(id).
		Where(apikey.DeletedAtIsNil()).
		AddQuotaUsed(amount).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return 0, service.ErrAPIKeyNotFound
		}
		return 0, err
	}
	return updated.QuotaUsed, nil
}

// IncrementQuotaUsedAndGetState atomically increments quota_used, conditionally marks the key
// as quota_exhausted, and returns the latest quota state in one round trip.

func (r *apiKeyRepository) IncrementQuotaUsedAndGetState(ctx context.Context, id int64, amount float64) (*service.APIKeyQuotaUsageState, error) {
	query := `
		UPDATE api_keys
		SET
			quota_used = quota_used + $1,
			status = CASE
				WHEN quota > 0 AND quota_used + $1 >= quota THEN $2
				ELSE status
			END,
			updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING quota_used, quota, key, status
	`

	state := &service.APIKeyQuotaUsageState{}
	if err := scanSingleRow(ctx, r.sql, query, []any{amount, service.StatusAPIKeyQuotaExhausted, id}, &state.QuotaUsed, &state.Quota, &state.Key, &state.Status); err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrAPIKeyNotFound
		}
		return nil, err
	}
	return state, nil
}

func (r *apiKeyRepository) TryReserveImageCount(ctx context.Context, id int64, count int) (bool, error) {
	if count <= 0 {
		return true, nil
	}
	query := `
		UPDATE api_keys
		SET
			image_count_used = image_count_used + $1,
			updated_at = NOW()
		WHERE id = $2
			AND deleted_at IS NULL
			AND image_only_enabled = TRUE
			AND image_count_billing_enabled = TRUE
			AND image_max_count > 0
			AND image_count_used + $1 <= image_max_count
		RETURNING image_count_used
	`
	var nextUsed int
	if err := scanSingleRow(ctx, r.sql, query, []any{count, id}, &nextUsed); err == nil {
		return true, nil
	} else if err != sql.ErrNoRows {
		return false, err
	}

	// Distinguish "not found" from "quota exhausted / disabled / max=0".
	var exists int
	if err := scanSingleRow(ctx, r.sql, `SELECT 1 FROM api_keys WHERE id = $1 AND deleted_at IS NULL`, []any{id}, &exists); err != nil {
		if err == sql.ErrNoRows {
			return false, service.ErrAPIKeyNotFound
		}
		return false, err
	}
	return false, nil
}

func (r *apiKeyRepository) RollbackImageCount(ctx context.Context, id int64, count int) error {
	if count <= 0 {
		return nil
	}
	query := `
		UPDATE api_keys
		SET
			image_count_used = GREATEST(image_count_used - $1, 0),
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING image_count_used
	`
	var nextUsed int
	if err := scanSingleRow(ctx, r.sql, query, []any{count, id}, &nextUsed); err != nil {
		if err == sql.ErrNoRows {
			return service.ErrAPIKeyNotFound
		}
		return err
	}
	return nil
}

func (r *apiKeyRepository) IncrementRateLimitUsage(ctx context.Context, id int64, cost float64) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h = CASE WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW() THEN $1 ELSE usage_5h + $1 END,
			usage_1d = CASE WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW() THEN $1 ELSE usage_1d + $1 END,
			usage_7d = CASE WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW() THEN $1 ELSE usage_7d + $1 END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL`,
		cost, id)
	return err
}

// ResetRateLimitWindows resets expired rate limit windows atomically.

func (r *apiKeyRepository) ResetRateLimitWindows(ctx context.Context, id int64) error {
	_, err := r.sql.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h = CASE WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW() THEN 0 ELSE usage_5h END,
			window_5h_start = CASE WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			usage_1d = CASE WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW() THEN 0 ELSE usage_1d END,
			window_1d_start = CASE WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			usage_7d = CASE WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW() THEN 0 ELSE usage_7d END,
			window_7d_start = CASE WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`,
		id)
	return err
}

// GetRateLimitData returns the current rate limit usage and window start times for an API key.

func (r *apiKeyRepository) GetRateLimitData(ctx context.Context, id int64) (result *service.APIKeyRateLimitData, err error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT usage_5h, usage_1d, usage_7d,
			COALESCE(usage_5h_by_currency, '{}'::jsonb),
			COALESCE(usage_1d_by_currency, '{}'::jsonb),
			COALESCE(usage_7d_by_currency, '{}'::jsonb),
			window_5h_start, window_1d_start, window_7d_start
		FROM api_keys
		WHERE id = $1 AND deleted_at IS NULL`,
		id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	if !rows.Next() {
		return nil, service.ErrAPIKeyNotFound
	}
	data := &service.APIKeyRateLimitData{}
	var usage5hRaw, usage1dRaw, usage7dRaw []byte
	if err := rows.Scan(&data.Usage5h, &data.Usage1d, &data.Usage7d, &usage5hRaw, &usage1dRaw, &usage7dRaw, &data.Window5hStart, &data.Window1dStart, &data.Window7dStart); err != nil {
		return nil, err
	}
	data.Usage5hByCurrency = service.CloneBillingCurrencyMap(parseBillingCurrencyJSONMap(usage5hRaw))
	data.Usage1dByCurrency = service.CloneBillingCurrencyMap(parseBillingCurrencyJSONMap(usage1dRaw))
	data.Usage7dByCurrency = service.CloneBillingCurrencyMap(parseBillingCurrencyJSONMap(usage7dRaw))
	return data, rows.Err()
}
