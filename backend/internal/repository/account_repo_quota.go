package repository

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const nowUTC = `to_char(NOW() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`

func (r *accountRepository) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) error {
	rows, err := r.sql.QueryContext(ctx, `UPDATE accounts SET extra = (
			COALESCE(extra, '{}'::jsonb)
			-- 总额度：始终递增
			|| jsonb_build_object('quota_used', COALESCE((extra->>'quota_used')::numeric, 0) + $1)
			-- 日额度：仅在 quota_daily_limit > 0 时处理
			|| CASE WHEN COALESCE((extra->>'quota_daily_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_daily_used',
					CASE WHEN COALESCE((extra->>'quota_daily_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '24 hours'::interval <= NOW()
					THEN $1
					ELSE COALESCE((extra->>'quota_daily_used')::numeric, 0) + $1 END,
					'quota_daily_start',
					CASE WHEN COALESCE((extra->>'quota_daily_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '24 hours'::interval <= NOW()
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_daily_start', `+nowUTC+`) END
				)
			ELSE '{}'::jsonb END
			-- 周额度：仅在 quota_weekly_limit > 0 时处理
			|| CASE WHEN COALESCE((extra->>'quota_weekly_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_weekly_used',
					CASE WHEN COALESCE((extra->>'quota_weekly_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '168 hours'::interval <= NOW()
					THEN $1
					ELSE COALESCE((extra->>'quota_weekly_used')::numeric, 0) + $1 END,
					'quota_weekly_start',
					CASE WHEN COALESCE((extra->>'quota_weekly_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '168 hours'::interval <= NOW()
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_weekly_start', `+nowUTC+`) END
				)
			ELSE '{}'::jsonb END
		), updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING
			COALESCE((extra->>'quota_used')::numeric, 0),
			COALESCE((extra->>'quota_limit')::numeric, 0)`, amount, id)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()
	var newUsed, limit float64
	if rows.Next() {
		if err := rows.Scan(&newUsed, &limit); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if limit > 0 && newUsed >= limit && (newUsed-amount) < limit {
		if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue quota exceeded failed: account=%d err=%v", id, err)
		}
	}
	return nil
}
