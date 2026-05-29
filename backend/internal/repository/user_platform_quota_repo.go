package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type userPlatformQuotaRepository struct {
	db *sql.DB
}

func NewUserPlatformQuotaRepository(_ *dbent.Client, db *sql.DB) service.UserPlatformQuotaRepository {
	return &userPlatformQuotaRepository{db: db}
}

func (r *userPlatformQuotaRepository) ListByUser(ctx context.Context, userID int64) ([]service.UserPlatformQuota, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return []service.UserPlatformQuota{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, user_id, platform,
       daily_limit_usd, weekly_limit_usd, monthly_limit_usd,
       daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
       daily_window_start, weekly_window_start, monthly_window_start,
       created_at, updated_at
FROM user_platform_quotas
WHERE user_id = $1
ORDER BY platform ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.UserPlatformQuota, 0)
	for rows.Next() {
		item, scanErr := scanUserPlatformQuota(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *userPlatformQuotaRepository) ReplaceForUser(ctx context.Context, userID int64, items []service.UserPlatformQuotaInput) ([]service.UserPlatformQuota, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("user platform quota repository db is nil")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	platforms := make([]string, 0, len(items))
	for _, item := range items {
		platforms = append(platforms, item.Platform)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO user_platform_quotas (
    user_id, platform,
    daily_limit_usd, weekly_limit_usd, monthly_limit_usd,
    daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
    daily_window_start, weekly_window_start, monthly_window_start,
    created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, 0, 0, 0, NULL, NULL, NULL, NOW(), NOW())
ON CONFLICT (user_id, platform) DO UPDATE
SET daily_limit_usd = EXCLUDED.daily_limit_usd,
    weekly_limit_usd = EXCLUDED.weekly_limit_usd,
    monthly_limit_usd = EXCLUDED.monthly_limit_usd,
    updated_at = NOW()
`, userID, item.Platform, nullableFloat(item.DailyLimitUSD), nullableFloat(item.WeeklyLimitUSD), nullableFloat(item.MonthlyLimitUSD)); err != nil {
			return nil, err
		}
	}

	if len(platforms) == 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM user_platform_quotas WHERE user_id = $1`, userID); err != nil {
			return nil, err
		}
	} else if _, err := tx.ExecContext(ctx, `
DELETE FROM user_platform_quotas
WHERE user_id = $1 AND NOT (platform = ANY($2))
`, userID, pq.Array(platforms)); err != nil {
		return nil, err
	}

	rows, err := tx.QueryContext(ctx, `
SELECT id, user_id, platform,
       daily_limit_usd, weekly_limit_usd, monthly_limit_usd,
       daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
       daily_window_start, weekly_window_start, monthly_window_start,
       created_at, updated_at
FROM user_platform_quotas
WHERE user_id = $1
ORDER BY platform ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.UserPlatformQuota, 0)
	for rows.Next() {
		item, scanErr := scanUserPlatformQuota(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

type userPlatformQuotaScanner interface {
	Scan(dest ...any) error
}

func scanUserPlatformQuota(scanner userPlatformQuotaScanner) (service.UserPlatformQuota, error) {
	var item service.UserPlatformQuota
	var dailyLimit, weeklyLimit, monthlyLimit sql.NullFloat64
	var dailyStart, weeklyStart, monthlyStart sql.NullTime
	if err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.Platform,
		&dailyLimit,
		&weeklyLimit,
		&monthlyLimit,
		&item.DailyUsageUSD,
		&item.WeeklyUsageUSD,
		&item.MonthlyUsageUSD,
		&dailyStart,
		&weeklyStart,
		&monthlyStart,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return item, err
	}
	item.DailyLimitUSD = floatPtrFromNull(dailyLimit)
	item.WeeklyLimitUSD = floatPtrFromNull(weeklyLimit)
	item.MonthlyLimitUSD = floatPtrFromNull(monthlyLimit)
	item.DailyWindowStart = timePtrFromNull(dailyStart)
	item.WeeklyWindowStart = timePtrFromNull(weeklyStart)
	item.MonthlyWindowStart = timePtrFromNull(monthlyStart)
	return item, nil
}

func nullableFloat(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}

func floatPtrFromNull(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

func timePtrFromNull(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time.UTC()
	return &v
}
