package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
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
	monthStart := timezone.StartOfMonth(todayStart)
	windows := make([]service.AccountStatsWindowStart, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		windows = append(windows, service.AccountStatsWindowStart{
			AccountID:    accountID,
			TodayStart:   todayStart,
			WeeklyStart:  weekStart,
			MonthlyStart: monthStart,
		})
	}
	return r.GetAccountTodayStatsBreakdownBatchByWindows(ctx, windows)
}

func (r *usageLogRepository) GetAccountTodayStatsBreakdownBatchByWindows(ctx context.Context, windows []service.AccountStatsWindowStart) (map[int64]*usagestats.AccountTodayStatsBreakdown, error) {
	result := make(map[int64]*usagestats.AccountTodayStatsBreakdown, len(windows))
	if len(windows) == 0 {
		return result, nil
	}
	accountIDs := make([]int64, 0, len(windows))
	todayStarts := make([]time.Time, 0, len(windows))
	weeklyStarts := make([]time.Time, 0, len(windows))
	monthlyStarts := make([]time.Time, 0, len(windows))
	for _, window := range windows {
		accountIDs = append(accountIDs, window.AccountID)
		todayStarts = append(todayStarts, window.TodayStart)
		weeklyStarts = append(weeklyStarts, window.WeeklyStart)
		monthlyStarts = append(monthlyStarts, window.MonthlyStart)
	}
	query := `
		WITH windows AS (
			SELECT *
			FROM unnest($1::bigint[], $2::timestamptz[], $3::timestamptz[], $4::timestamptz[])
				AS w(account_id, today_start, weekly_start, monthly_start)
		)
		SELECT
			w.account_id,
			COUNT(l.id) FILTER (WHERE l.created_at >= w.today_start) as today_requests,
			COALESCE(SUM(l.input_tokens + l.output_tokens + l.cache_creation_tokens + l.cache_read_tokens) FILTER (WHERE l.created_at >= w.today_start), 0) as today_tokens,
			COALESCE(SUM(l.total_cost_usd_equivalent * COALESCE(l.account_rate_multiplier, 1)) FILTER (WHERE l.created_at >= w.today_start), 0) as today_cost,
			COALESCE(SUM(l.total_cost_usd_equivalent) FILTER (WHERE l.created_at >= w.today_start), 0) as today_standard_cost,
			COALESCE(SUM(l.actual_cost_usd_equivalent) FILTER (WHERE l.created_at >= w.today_start), 0) as today_user_cost,
			CASE WHEN COUNT(l.id) FILTER (WHERE l.created_at >= w.today_start) = 0 THEN 100 ELSE ((COUNT(l.id) FILTER (WHERE l.created_at >= w.today_start AND l.status = 'succeeded'))::float / (COUNT(l.id) FILTER (WHERE l.created_at >= w.today_start))::float) * 100 END as today_success_rate,
			COALESCE(AVG(l.duration_ms) FILTER (WHERE l.created_at >= w.today_start AND l.status = 'succeeded' AND l.duration_ms IS NOT NULL), 0) as today_average_duration_ms,
			COUNT(l.id) FILTER (WHERE l.created_at >= w.weekly_start) as weekly_requests,
			COALESCE(SUM(l.input_tokens + l.output_tokens + l.cache_creation_tokens + l.cache_read_tokens) FILTER (WHERE l.created_at >= w.weekly_start), 0) as weekly_tokens,
			COALESCE(SUM(l.total_cost_usd_equivalent * COALESCE(l.account_rate_multiplier, 1)) FILTER (WHERE l.created_at >= w.weekly_start), 0) as weekly_cost,
			COALESCE(SUM(l.total_cost_usd_equivalent) FILTER (WHERE l.created_at >= w.weekly_start), 0) as weekly_standard_cost,
			COALESCE(SUM(l.actual_cost_usd_equivalent) FILTER (WHERE l.created_at >= w.weekly_start), 0) as weekly_user_cost,
			CASE WHEN COUNT(l.id) FILTER (WHERE l.created_at >= w.weekly_start) = 0 THEN 100 ELSE ((COUNT(l.id) FILTER (WHERE l.created_at >= w.weekly_start AND l.status = 'succeeded'))::float / (COUNT(l.id) FILTER (WHERE l.created_at >= w.weekly_start))::float) * 100 END as weekly_success_rate,
			COALESCE(AVG(l.duration_ms) FILTER (WHERE l.created_at >= w.weekly_start AND l.status = 'succeeded' AND l.duration_ms IS NOT NULL), 0) as weekly_average_duration_ms,
			COUNT(l.id) FILTER (WHERE l.created_at >= w.monthly_start) as monthly_requests,
			COALESCE(SUM(l.input_tokens + l.output_tokens + l.cache_creation_tokens + l.cache_read_tokens) FILTER (WHERE l.created_at >= w.monthly_start), 0) as monthly_tokens,
			COALESCE(SUM(l.total_cost_usd_equivalent * COALESCE(l.account_rate_multiplier, 1)) FILTER (WHERE l.created_at >= w.monthly_start), 0) as monthly_cost,
			COALESCE(SUM(l.total_cost_usd_equivalent) FILTER (WHERE l.created_at >= w.monthly_start), 0) as monthly_standard_cost,
			COALESCE(SUM(l.actual_cost_usd_equivalent) FILTER (WHERE l.created_at >= w.monthly_start), 0) as monthly_user_cost,
			CASE WHEN COUNT(l.id) FILTER (WHERE l.created_at >= w.monthly_start) = 0 THEN 100 ELSE ((COUNT(l.id) FILTER (WHERE l.created_at >= w.monthly_start AND l.status = 'succeeded'))::float / (COUNT(l.id) FILTER (WHERE l.created_at >= w.monthly_start))::float) * 100 END as monthly_success_rate,
			COALESCE(AVG(l.duration_ms) FILTER (WHERE l.created_at >= w.monthly_start AND l.status = 'succeeded' AND l.duration_ms IS NOT NULL), 0) as monthly_average_duration_ms,
			COUNT(l.id) as total_requests,
			COALESCE(SUM(l.input_tokens + l.output_tokens + l.cache_creation_tokens + l.cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(l.total_cost_usd_equivalent * COALESCE(l.account_rate_multiplier, 1)), 0) as total_cost,
			COALESCE(SUM(l.total_cost_usd_equivalent), 0) as total_standard_cost,
			COALESCE(SUM(l.actual_cost_usd_equivalent), 0) as total_user_cost,
			CASE WHEN COUNT(l.id) = 0 THEN 100 ELSE (COUNT(l.id) FILTER (WHERE l.status = 'succeeded')::float / COUNT(l.id)::float) * 100 END as total_success_rate,
			COALESCE(AVG(l.duration_ms) FILTER (WHERE l.status = 'succeeded' AND l.duration_ms IS NOT NULL), 0) as total_average_duration_ms
		FROM windows w
		LEFT JOIN usage_logs l ON l.account_id = w.account_id
		GROUP BY w.account_id
	`
	rows, err := r.sql.QueryContext(ctx, query, pq.Array(accountIDs), pq.Array(todayStarts), pq.Array(weeklyStarts), pq.Array(monthlyStarts))
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
			&stats.Monthly.Requests,
			&stats.Monthly.Tokens,
			&stats.Monthly.Cost,
			&stats.Monthly.StandardCost,
			&stats.Monthly.UserCost,
			&stats.Monthly.SuccessRate,
			&stats.Monthly.AverageDurationMs,
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
			result[accountID] = emptyAccountTodayStatsBreakdown()
		}
	}
	return result, nil
}

func emptyAccountTodayStatsBreakdown() *usagestats.AccountTodayStatsBreakdown {
	return &usagestats.AccountTodayStatsBreakdown{
		Today:   usagestats.AccountStats{SuccessRate: 100},
		Weekly:  usagestats.AccountStats{SuccessRate: 100},
		Monthly: usagestats.AccountStats{SuccessRate: 100},
		Total:   usagestats.AccountStats{SuccessRate: 100},
	}
}
