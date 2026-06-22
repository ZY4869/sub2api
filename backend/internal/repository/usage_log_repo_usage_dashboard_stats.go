package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *usageLogRepository) GetUserDashboardStats(ctx context.Context, userID int64) (*UserDashboardStats, error) {
	stats := &UserDashboardStats{}
	today := timezone.Today()
	if err := scanSingleRow(ctx, r.sql, "SELECT COUNT(*) FROM api_keys WHERE user_id = $1 AND deleted_at IS NULL", []any{userID}, &stats.TotalAPIKeys); err != nil {
		return nil, err
	}
	if err := scanSingleRow(ctx, r.sql, "SELECT COUNT(*) FROM api_keys WHERE user_id = $1 AND status = $2 AND deleted_at IS NULL", []any{userID, service.StatusActive}, &stats.ActiveAPIKeys); err != nil {
		return nil, err
	}
	cacheCreationExpr := usageCacheCreationSQL("")
	totalStatsQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(%[1]s), 0) as total_cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE user_id = $1
	`, cacheCreationExpr)
	if err := scanSingleRow(ctx, r.sql, totalStatsQuery, []any{userID}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheCreationTokens, &stats.TotalCacheReadTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
	stats.TotalCacheTokens = stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
	if inputSideTokens := stats.TotalInputTokens + stats.TotalCacheTokens; inputSideTokens > 0 {
		stats.CacheHitRate = float64(stats.TotalCacheReadTokens) / float64(inputSideTokens)
	}
	costByCurrency, actualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE user_id = $1", []any{userID})
	if err != nil {
		return nil, err
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	todayStatsQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as today_requests,
			COALESCE(SUM(input_tokens), 0) as today_input_tokens,
			COALESCE(SUM(output_tokens), 0) as today_output_tokens,
			COALESCE(SUM(%[1]s), 0) as today_cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as today_cache_read_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as today_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as today_actual_cost
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2
	`, cacheCreationExpr)
	if err := scanSingleRow(ctx, r.sql, todayStatsQuery, []any{userID, today}, &stats.TodayRequests, &stats.TodayInputTokens, &stats.TodayOutputTokens, &stats.TodayCacheCreationTokens, &stats.TodayCacheReadTokens, &stats.TodayCost, &stats.TodayActualCost); err != nil {
		return nil, err
	}
	stats.TodayTokens = stats.TodayInputTokens + stats.TodayOutputTokens + stats.TodayCacheCreationTokens + stats.TodayCacheReadTokens
	stats.TodayCacheTokens = stats.TodayCacheCreationTokens + stats.TodayCacheReadTokens
	if inputSideTokens := stats.TodayInputTokens + stats.TodayCacheTokens; inputSideTokens > 0 {
		stats.TodayCacheHitRate = float64(stats.TodayCacheReadTokens) / float64(inputSideTokens)
	}
	todayCostByCurrency, todayActualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE user_id = $1 AND created_at >= $2", []any{userID, today})
	if err != nil {
		return nil, err
	}
	stats.TodayCostByCurrency = todayCostByCurrency
	stats.TodayActualCostByCurrency = todayActualCostByCurrency
	rpm, tpm, err := r.getPerformanceStats(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.Rpm = rpm
	stats.Tpm = tpm
	return stats, nil
}
func (r *usageLogRepository) getPerformanceStatsByAPIKey(ctx context.Context, apiKeyID int64) (rpm, tpm int64, err error) {
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	totalTokensExpr := usageTotalTokensSQL("")
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as request_count,
			COALESCE(SUM(%[1]s), 0) as token_count
		FROM usage_logs
		WHERE created_at >= $1 AND api_key_id = $2
	`, totalTokensExpr)
	args := []any{fiveMinutesAgo, apiKeyID}
	var requestCount int64
	var tokenCount int64
	if err := scanSingleRow(ctx, r.sql, query, args, &requestCount, &tokenCount); err != nil {
		return 0, 0, err
	}
	return requestCount / 5, tokenCount / 5, nil
}
func (r *usageLogRepository) GetAPIKeyDashboardStats(ctx context.Context, apiKeyID int64) (*UserDashboardStats, error) {
	stats := &UserDashboardStats{}
	today := timezone.Today()
	stats.TotalAPIKeys = 1
	stats.ActiveAPIKeys = 1
	cacheCreationExpr := usageCacheCreationSQL("")
	totalStatsQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(%[1]s), 0) as total_cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE api_key_id = $1
	`, cacheCreationExpr)
	if err := scanSingleRow(ctx, r.sql, totalStatsQuery, []any{apiKeyID}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheCreationTokens, &stats.TotalCacheReadTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
	stats.TotalCacheTokens = stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
	if inputSideTokens := stats.TotalInputTokens + stats.TotalCacheTokens; inputSideTokens > 0 {
		stats.CacheHitRate = float64(stats.TotalCacheReadTokens) / float64(inputSideTokens)
	}
	costByCurrency, actualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE api_key_id = $1", []any{apiKeyID})
	if err != nil {
		return nil, err
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	todayStatsQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as today_requests,
			COALESCE(SUM(input_tokens), 0) as today_input_tokens,
			COALESCE(SUM(output_tokens), 0) as today_output_tokens,
			COALESCE(SUM(%[1]s), 0) as today_cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as today_cache_read_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as today_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as today_actual_cost
		FROM usage_logs
		WHERE api_key_id = $1 AND created_at >= $2
	`, cacheCreationExpr)
	if err := scanSingleRow(ctx, r.sql, todayStatsQuery, []any{apiKeyID, today}, &stats.TodayRequests, &stats.TodayInputTokens, &stats.TodayOutputTokens, &stats.TodayCacheCreationTokens, &stats.TodayCacheReadTokens, &stats.TodayCost, &stats.TodayActualCost); err != nil {
		return nil, err
	}
	stats.TodayTokens = stats.TodayInputTokens + stats.TodayOutputTokens + stats.TodayCacheCreationTokens + stats.TodayCacheReadTokens
	stats.TodayCacheTokens = stats.TodayCacheCreationTokens + stats.TodayCacheReadTokens
	if inputSideTokens := stats.TodayInputTokens + stats.TodayCacheTokens; inputSideTokens > 0 {
		stats.TodayCacheHitRate = float64(stats.TodayCacheReadTokens) / float64(inputSideTokens)
	}
	todayCostByCurrency, todayActualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE api_key_id = $1 AND created_at >= $2", []any{apiKeyID, today})
	if err != nil {
		return nil, err
	}
	stats.TodayCostByCurrency = todayCostByCurrency
	stats.TodayActualCostByCurrency = todayActualCostByCurrency
	rpm, tpm, err := r.getPerformanceStatsByAPIKey(ctx, apiKeyID)
	if err != nil {
		return nil, err
	}
	stats.Rpm = rpm
	stats.Tpm = tpm
	return stats, nil
}
