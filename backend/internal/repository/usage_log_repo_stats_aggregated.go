package repository

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

func (r *usageLogRepository) GetUserStatsAggregated(ctx context.Context, userID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	query := usageStatsAggregatedQuery("user_id = $1 AND created_at >= $2 AND created_at < $3")
	var stats usagestats.UsageStats
	if err := scanSingleRow(ctx, r.sql, query, []any{userID, startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheCreationTokens, &stats.TotalCacheReadTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	normalizeUsageStatsCacheTotals(&stats)
	costByCurrency, actualCostByCurrency, costErr := queryUsageCostByCurrency(ctx, r.sql, "WHERE user_id = $1 AND created_at >= $2 AND created_at < $3", []any{userID, startTime, endTime})
	if costErr != nil {
		return nil, costErr
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return &stats, nil
}
func (r *usageLogRepository) GetAPIKeyStatsAggregated(ctx context.Context, apiKeyID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	query := usageStatsAggregatedQuery("api_key_id = $1 AND created_at >= $2 AND created_at < $3")
	var stats usagestats.UsageStats
	if err := scanSingleRow(ctx, r.sql, query, []any{apiKeyID, startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheCreationTokens, &stats.TotalCacheReadTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	normalizeUsageStatsCacheTotals(&stats)
	costByCurrency, actualCostByCurrency, costErr := queryUsageCostByCurrency(ctx, r.sql, "WHERE api_key_id = $1 AND created_at >= $2 AND created_at < $3", []any{apiKeyID, startTime, endTime})
	if costErr != nil {
		return nil, costErr
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return &stats, nil
}
func (r *usageLogRepository) GetAccountStatsAggregated(ctx context.Context, accountID int64, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	query := usageStatsAggregatedQuery("account_id = $1 AND created_at >= $2 AND created_at < $3")
	var stats usagestats.UsageStats
	if err := scanSingleRow(ctx, r.sql, query, []any{accountID, startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheCreationTokens, &stats.TotalCacheReadTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	normalizeUsageStatsCacheTotals(&stats)
	costByCurrency, actualCostByCurrency, costErr := queryUsageCostByCurrency(ctx, r.sql, "WHERE account_id = $1 AND created_at >= $2 AND created_at < $3", []any{accountID, startTime, endTime})
	if costErr != nil {
		return nil, costErr
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return &stats, nil
}
func (r *usageLogRepository) GetModelStatsAggregated(ctx context.Context, modelName string, startTime, endTime time.Time) (*usagestats.UsageStats, error) {
	query := usageStatsAggregatedQuery("model = $1 AND created_at >= $2 AND created_at < $3")
	var stats usagestats.UsageStats
	if err := scanSingleRow(ctx, r.sql, query, []any{modelName, startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheCreationTokens, &stats.TotalCacheReadTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	normalizeUsageStatsCacheTotals(&stats)
	costByCurrency, actualCostByCurrency, costErr := queryUsageCostByCurrency(ctx, r.sql, "WHERE model = $1 AND created_at >= $2 AND created_at < $3", []any{modelName, startTime, endTime})
	if costErr != nil {
		return nil, costErr
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return &stats, nil
}
func (r *usageLogRepository) GetDailyStatsAggregated(ctx context.Context, userID int64, startTime, endTime time.Time) (result []map[string]any, err error) {
	tzName := resolveUsageStatsTimezone()
	cacheCreationExpr := usageCacheCreationSQL("")
	query := fmt.Sprintf(`
		SELECT
			-- 使用应用时区分组，避免数据库会话时区导致日边界偏移。
			TO_CHAR(created_at AT TIME ZONE $4, 'YYYY-MM-DD') as date,
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(%[1]s + COALESCE(cache_read_tokens, 0)), 0) as total_cache_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY 1
		ORDER BY 1
	`, cacheCreationExpr)
	rows, err := r.sql.QueryContext(ctx, query, userID, startTime, endTime, tzName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()
	result = make([]map[string]any, 0)
	for rows.Next() {
		var (
			date              string
			totalRequests     int64
			totalInputTokens  int64
			totalOutputTokens int64
			totalCacheTokens  int64
			totalCost         float64
			totalActualCost   float64
			avgDurationMs     float64
		)
		if err = rows.Scan(&date, &totalRequests, &totalInputTokens, &totalOutputTokens, &totalCacheTokens, &totalCost, &totalActualCost, &avgDurationMs); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{"date": date, "total_requests": totalRequests, "total_input_tokens": totalInputTokens, "total_output_tokens": totalOutputTokens, "total_cache_tokens": totalCacheTokens, "total_tokens": totalInputTokens + totalOutputTokens + totalCacheTokens, "total_cost": totalCost, "total_actual_cost": totalActualCost, "average_duration_ms": avgDurationMs})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func usageStatsAggregatedQuery(whereClause string) string {
	cacheCreationExpr := usageCacheCreationSQL("")
	return fmt.Sprintf(`
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
		WHERE %[2]s
	`, cacheCreationExpr, whereClause)
}

func resolveUsageStatsTimezone() string {
	tzName := timezone.Name()
	if tzName != "" && tzName != "Local" {
		return tzName
	}
	if envTZ := strings.TrimSpace(os.Getenv("TZ")); envTZ != "" {
		return envTZ
	}
	return "UTC"
}
