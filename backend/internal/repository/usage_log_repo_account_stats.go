package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/lib/pq"
)

func (r *usageLogRepository) GetAccountTodayStats(ctx context.Context, accountID int64) (*usagestats.AccountStats, error) {
	today := timezone.Today()
	query := `
		SELECT
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as tokens,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as cost,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as standard_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as user_cost,
			CASE WHEN COUNT(*) = 0 THEN 100 ELSE (COUNT(*) FILTER (WHERE status = 'succeeded')::float / COUNT(*)::float) * 100 END as success_rate,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded' AND duration_ms IS NOT NULL), 0) as average_duration_ms
		FROM usage_logs
		WHERE account_id = $1 AND created_at >= $2
	`
	stats := &usagestats.AccountStats{}
	if err := scanSingleRow(ctx, r.sql, query, []any{accountID, today}, &stats.Requests, &stats.Tokens, &stats.Cost, &stats.StandardCost, &stats.UserCost, &stats.SuccessRate, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	return stats, nil
}
func (r *usageLogRepository) GetAccountWindowStats(ctx context.Context, accountID int64, startTime time.Time) (*usagestats.AccountStats, error) {
	query := `
		SELECT
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as tokens,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as cost,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as standard_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as user_cost,
			CASE WHEN COUNT(*) = 0 THEN 100 ELSE (COUNT(*) FILTER (WHERE status = 'succeeded')::float / COUNT(*)::float) * 100 END as success_rate,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded' AND duration_ms IS NOT NULL), 0) as average_duration_ms
		FROM usage_logs
		WHERE account_id = $1 AND created_at >= $2
	`
	stats := &usagestats.AccountStats{}
	if err := scanSingleRow(ctx, r.sql, query, []any{accountID, startTime}, &stats.Requests, &stats.Tokens, &stats.Cost, &stats.StandardCost, &stats.UserCost, &stats.SuccessRate, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	return stats, nil
}
func (r *usageLogRepository) GetAccountWindowStatsBatch(ctx context.Context, accountIDs []int64, startTime time.Time) (map[int64]*usagestats.AccountStats, error) {
	result := make(map[int64]*usagestats.AccountStats, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	query := `
		SELECT
			account_id,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as tokens,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as cost,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as standard_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as user_cost,
			CASE WHEN COUNT(*) = 0 THEN 100 ELSE (COUNT(*) FILTER (WHERE status = 'succeeded')::float / COUNT(*)::float) * 100 END as success_rate,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded' AND duration_ms IS NOT NULL), 0) as average_duration_ms
		FROM usage_logs
		WHERE account_id = ANY($1) AND created_at >= $2
		GROUP BY account_id
	`
	rows, err := r.sql.QueryContext(ctx, query, pq.Array(accountIDs), startTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var accountID int64
		stats := &usagestats.AccountStats{}
		if err := rows.Scan(&accountID, &stats.Requests, &stats.Tokens, &stats.Cost, &stats.StandardCost, &stats.UserCost, &stats.SuccessRate, &stats.AverageDurationMs); err != nil {
			return nil, err
		}
		result[accountID] = stats
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, accountID := range accountIDs {
		if _, ok := result[accountID]; !ok {
			result[accountID] = &usagestats.AccountStats{}
		}
	}
	return result, nil
}

func (r *usageLogRepository) GetAccountTodayStatsBreakdownBatch(ctx context.Context, accountIDs []int64, todayStart, weekStart time.Time) (map[int64]*usagestats.AccountTodayStatsBreakdown, error) {
	result := make(map[int64]*usagestats.AccountTodayStatsBreakdown, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	query := `
		SELECT
			account_id,
			COUNT(*) FILTER (WHERE created_at >= $2) as today_requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) FILTER (WHERE created_at >= $2), 0) as today_tokens,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)) FILTER (WHERE created_at >= $2), 0) as today_cost,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE created_at >= $2), 0) as today_standard_cost,
			COALESCE(SUM(actual_cost_usd_equivalent) FILTER (WHERE created_at >= $2), 0) as today_user_cost,
			CASE WHEN COUNT(*) FILTER (WHERE created_at >= $2) = 0 THEN 100 ELSE ((COUNT(*) FILTER (WHERE created_at >= $2 AND status = 'succeeded'))::float / (COUNT(*) FILTER (WHERE created_at >= $2))::float) * 100 END as today_success_rate,
			COALESCE(AVG(duration_ms) FILTER (WHERE created_at >= $2 AND status = 'succeeded' AND duration_ms IS NOT NULL), 0) as today_average_duration_ms,
			COUNT(*) FILTER (WHERE created_at >= $3) as weekly_requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) FILTER (WHERE created_at >= $3), 0) as weekly_tokens,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)) FILTER (WHERE created_at >= $3), 0) as weekly_cost,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE created_at >= $3), 0) as weekly_standard_cost,
			COALESCE(SUM(actual_cost_usd_equivalent) FILTER (WHERE created_at >= $3), 0) as weekly_user_cost,
			CASE WHEN COUNT(*) FILTER (WHERE created_at >= $3) = 0 THEN 100 ELSE ((COUNT(*) FILTER (WHERE created_at >= $3 AND status = 'succeeded'))::float / (COUNT(*) FILTER (WHERE created_at >= $3))::float) * 100 END as weekly_success_rate,
			COALESCE(AVG(duration_ms) FILTER (WHERE created_at >= $3 AND status = 'succeeded' AND duration_ms IS NOT NULL), 0) as weekly_average_duration_ms,
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as total_cost,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_standard_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_user_cost,
			CASE WHEN COUNT(*) = 0 THEN 100 ELSE (COUNT(*) FILTER (WHERE status = 'succeeded')::float / COUNT(*)::float) * 100 END as total_success_rate,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded' AND duration_ms IS NOT NULL), 0) as total_average_duration_ms
		FROM usage_logs
		WHERE account_id = ANY($1)
		GROUP BY account_id
	`
	rows, err := r.sql.QueryContext(ctx, query, pq.Array(accountIDs), todayStart, weekStart)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		var accountID int64
		stats := &usagestats.AccountTodayStatsBreakdown{}
		if err := rows.Scan(
			&accountID,
			&stats.Today.Requests,
			&stats.Today.Tokens,
			&stats.Today.Cost,
			&stats.Today.StandardCost,
			&stats.Today.UserCost,
			&stats.Today.SuccessRate,
			&stats.Today.AverageDurationMs,
			&stats.Weekly.Requests,
			&stats.Weekly.Tokens,
			&stats.Weekly.Cost,
			&stats.Weekly.StandardCost,
			&stats.Weekly.UserCost,
			&stats.Weekly.SuccessRate,
			&stats.Weekly.AverageDurationMs,
			&stats.Total.Requests,
			&stats.Total.Tokens,
			&stats.Total.Cost,
			&stats.Total.StandardCost,
			&stats.Total.UserCost,
			&stats.Total.SuccessRate,
			&stats.Total.AverageDurationMs,
		); err != nil {
			return nil, err
		}
		result[accountID] = stats
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, accountID := range accountIDs {
		if _, ok := result[accountID]; !ok {
			result[accountID] = &usagestats.AccountTodayStatsBreakdown{
				Today:  usagestats.AccountStats{SuccessRate: 100},
				Weekly: usagestats.AccountStats{SuccessRate: 100},
				Total:  usagestats.AccountStats{SuccessRate: 100},
			}
		}
	}
	return result, nil
}
