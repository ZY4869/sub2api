package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *accountRepository) GetActiveUsagePeriods(ctx context.Context, accountIDs []int64, windowType string, at time.Time) (map[int64]*service.AccountUsagePeriod, error) {
	out := make(map[int64]*service.AccountUsagePeriod, len(accountIDs))
	if len(accountIDs) == 0 {
		return out, nil
	}
	windowType = strings.TrimSpace(windowType)
	if windowType == "" {
		return out, nil
	}
	if at.IsZero() {
		at = time.Now()
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT DISTINCT ON (account_id)
			account_id,
			window_type,
			start_at,
			end_at,
			reset_at,
			source
		FROM account_usage_periods
		WHERE account_id = ANY($1)
			AND window_type = $2
			AND start_at <= $3
			AND (end_at IS NULL OR end_at >= $3)
		ORDER BY account_id, start_at DESC
	`, pq.Array(accountIDs), windowType, at.UTC())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var period service.AccountUsagePeriod
		var endAt sql.NullTime
		var resetAt sql.NullTime
		if err := rows.Scan(&period.AccountID, &period.WindowType, &period.StartAt, &endAt, &resetAt, &period.Source); err != nil {
			return nil, err
		}
		if endAt.Valid {
			v := endAt.Time.UTC()
			period.EndAt = &v
		}
		if resetAt.Valid {
			v := resetAt.Time.UTC()
			period.ResetAt = &v
		}
		period.StartAt = period.StartAt.UTC()
		out[period.AccountID] = &period
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *accountRepository) SyncMonthlyUsagePeriod(ctx context.Context, account *service.Account, oldExpiresAt *time.Time, source string) (*service.AccountUsagePeriodSyncResult, error) {
	if account == nil || account.ID <= 0 {
		return nil, nil
	}
	startAt, endAt, source := monthlyUsagePeriodBoundary(account, source)
	if startAt.IsZero() {
		return nil, nil
	}
	result := &service.AccountUsagePeriodSyncResult{
		AccountID:  account.ID,
		WindowType: service.AccountUsagePeriodWindowMonthly,
		Source:     source,
		NewStartAt: usagePeriodTimePtr(startAt),
		NewEndAt:   cloneUsagePeriodTimePtr(endAt),
		NewResetAt: cloneUsagePeriodTimePtr(endAt),
	}
	if oldExpiresAt != nil && !oldExpiresAt.IsZero() {
		oldEnd := oldExpiresAt.UTC()
		result.OldEndAt = usagePeriodTimePtr(oldEnd)
		result.OldResetAt = usagePeriodTimePtr(oldEnd)
		if endAt == nil || !oldEnd.Equal(*endAt) {
			res, err := r.sql.ExecContext(ctx, `
			UPDATE account_usage_periods
			SET end_at = $2::timestamptz,
				updated_at = NOW()
			WHERE account_id = $1
				AND window_type = $3::varchar
				AND end_at IS NULL
				AND start_at < $2::timestamptz
			`, account.ID, oldEnd, service.AccountUsagePeriodWindowMonthly)
			if err != nil {
				return nil, err
			}
			result.Closed = rowsAffected(res) > 0
			if oldEnd.After(startAt) {
				startAt = oldEnd
				result.NewStartAt = usagePeriodTimePtr(startAt)
			}
		}
	}
	if oldExpiresAt == nil && endAt != nil {
		res, err := r.sql.ExecContext(ctx, `
			UPDATE account_usage_periods
			SET end_at = $4::timestamptz,
				reset_at = $4::timestamptz,
				source = $5::varchar,
				updated_at = NOW()
			WHERE account_id = $1
				AND window_type = $2::varchar
				AND end_at IS NULL
				AND start_at = $3::timestamptz
		`, account.ID, service.AccountUsagePeriodWindowMonthly, startAt, *endAt, source)
		if err != nil {
			return nil, err
		}
		if rowsAffected(res) > 0 {
			result.Updated = true
			result.OldStartAt = usagePeriodTimePtr(startAt)
		}
		res, err = r.sql.ExecContext(ctx, `
			UPDATE account_usage_periods
			SET end_at = $3::timestamptz,
				updated_at = NOW()
			WHERE account_id = $1
				AND window_type = $2::varchar
				AND end_at IS NULL
				AND start_at < $3::timestamptz
		`, account.ID, service.AccountUsagePeriodWindowMonthly, startAt)
		if err != nil {
			return nil, err
		}
		result.Closed = result.Closed || rowsAffected(res) > 0
	}
	res, err := r.sql.ExecContext(ctx, `
		INSERT INTO account_usage_periods (account_id, window_type, start_at, end_at, reset_at, source)
		SELECT $1::bigint, $2::varchar, $3::timestamptz, $4::timestamptz, $4::timestamptz, $5::varchar
		WHERE NOT EXISTS (
			SELECT 1
			FROM account_usage_periods
			WHERE account_id = $1
				AND window_type = $2::varchar
				AND start_at = $3::timestamptz
				AND COALESCE(end_at, 'infinity'::timestamptz) = COALESCE($4::timestamptz, 'infinity'::timestamptz)
		)
	`, account.ID, service.AccountUsagePeriodWindowMonthly, startAt, nullableTime(endAt), source)
	if err != nil {
		return nil, err
	}
	result.Inserted = rowsAffected(res) > 0
	return result, nil
}

func (r *accountRepository) SyncWeeklyUsagePeriod(ctx context.Context, account *service.Account, resetAt time.Time, source string) (*service.AccountUsagePeriodSyncResult, error) {
	if account == nil || account.ID <= 0 || resetAt.IsZero() {
		return nil, nil
	}
	resetAt = resetAt.UTC()
	startAt := resetAt.AddDate(0, 0, -7)
	if source == "" {
		source = service.AccountUsagePeriodSourceUpstreamReset
	}
	result := &service.AccountUsagePeriodSyncResult{
		AccountID:  account.ID,
		WindowType: service.AccountUsagePeriodWindowWeekly,
		Source:     source,
		NewStartAt: usagePeriodTimePtr(startAt),
		NewEndAt:   usagePeriodTimePtr(resetAt),
		NewResetAt: usagePeriodTimePtr(resetAt),
	}
	var closedCount int64
	var insertedCount int64
	var oldStartAt sql.NullTime
	var oldEndAt sql.NullTime
	var oldResetAt sql.NullTime
	err := scanSingleRow(ctx, r.sql, `
		WITH closing AS (
			SELECT id, start_at, end_at, reset_at
			FROM account_usage_periods
			WHERE account_id = $1
				AND window_type = $2
				AND reset_at IS NOT NULL
				AND reset_at <> $3
				AND start_at < $3
				AND (end_at IS NULL OR end_at > $3)
			FOR UPDATE
		),
		closed AS (
			UPDATE account_usage_periods
			SET end_at = $3,
				updated_at = NOW()
			WHERE id IN (SELECT id FROM closing)
			RETURNING id
		),
		inserted AS (
		INSERT INTO account_usage_periods (account_id, window_type, start_at, end_at, reset_at, source)
		SELECT $1, $2, $4, $3, $3, $5
		WHERE NOT EXISTS (
			SELECT 1
			FROM account_usage_periods
			WHERE account_id = $1
				AND window_type = $2
				AND reset_at = $3
		)
			RETURNING id
		)
		SELECT
			(SELECT COUNT(*) FROM closed),
			(SELECT start_at FROM closing LIMIT 1),
			(SELECT end_at FROM closing LIMIT 1),
			(SELECT reset_at FROM closing LIMIT 1),
			(SELECT COUNT(*) FROM inserted)
	`, []any{account.ID, service.AccountUsagePeriodWindowWeekly, resetAt, startAt, source}, &closedCount, &oldStartAt, &oldEndAt, &oldResetAt, &insertedCount)
	if err != nil {
		return nil, err
	}
	result.Closed = closedCount > 0
	result.Inserted = insertedCount > 0
	result.OldStartAt = sqlNullTimePtr(oldStartAt)
	result.OldEndAt = sqlNullTimePtr(oldEndAt)
	result.OldResetAt = sqlNullTimePtr(oldResetAt)
	return result, nil
}

func monthlyUsagePeriodBoundary(account *service.Account, source string) (time.Time, *time.Time, string) {
	if account == nil {
		return time.Time{}, nil, source
	}
	startAt := account.CreatedAt.UTC()
	if startAt.IsZero() {
		startAt = time.Now().UTC()
	}
	if account.ExpiresAt != nil && account.ExpiresAt.After(startAt) {
		endAt := account.ExpiresAt.UTC()
		if source == "" {
			source = service.AccountUsagePeriodSourceExpiry
		}
		return startAt, &endAt, source
	}
	if source == "" || source == service.AccountUsagePeriodSourceExpiry {
		source = service.AccountUsagePeriodSourceFallback30D
	}
	return startAt, nil, source
}

func nullableTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC()
}

func rowsAffected(result sql.Result) int64 {
	if result == nil {
		return 0
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0
	}
	return rows
}

func usagePeriodTimePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func cloneUsagePeriodTimePtr(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	return usagePeriodTimePtr(*value)
}

func sqlNullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid || value.Time.IsZero() {
		return nil
	}
	return usagePeriodTimePtr(value.Time)
}
