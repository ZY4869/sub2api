package repository

import (
	"context"
	"fmt"
	"time"
)

func (r *usageLogRepository) GetGlobalStats(ctx context.Context, startTime, endTime time.Time) (*UsageStats, error) {
	query := `
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE billing_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(total_cost_usd_equivalent) FILTER (WHERE billing_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded'), 0) as avg_duration_ms
		FROM usage_logs
		WHERE created_at >= $1 AND created_at <= $2
	`
	stats := &UsageStats{}
	if err := scanSingleRow(ctx, r.sql, query, []any{startTime, endTime}, &stats.TotalRequests, &stats.TotalInputTokens, &stats.TotalOutputTokens, &stats.TotalCacheCreationTokens, &stats.TotalCacheReadTokens, &stats.TotalCost, &stats.TotalActualCost, &stats.AdminFreeRequests, &stats.AdminFreeStandardCost, &stats.AverageDurationMs); err != nil {
		return nil, err
	}
	normalizeUsageStatsCacheTotals(stats)
	costByCurrency, actualCostByCurrency, err := queryUsageCostByCurrency(ctx, r.sql, "WHERE created_at >= $1 AND created_at <= $2", []any{startTime, endTime})
	if err != nil {
		return nil, err
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency
	return stats, nil
}

func buildUsageStatsBaseFiltersWithPrefix(filters UsageLogFilters, prefix string) ([]string, []any) {
	conditions := make([]string, 0, 9)
	args := make([]any, 0, 9)
	if filters.UserID > 0 {
		conditions = append(conditions, fmt.Sprintf("%suser_id = $%d", prefix, len(args)+1))
		args = append(args, filters.UserID)
	}
	if filters.APIKeyID > 0 {
		conditions = append(conditions, fmt.Sprintf("%sapi_key_id = $%d", prefix, len(args)+1))
		args = append(args, filters.APIKeyID)
	}
	if filters.AccountID > 0 {
		conditions = append(conditions, fmt.Sprintf("%saccount_id = $%d", prefix, len(args)+1))
		args = append(args, filters.AccountID)
	}
	if filters.GroupID > 0 {
		conditions = append(conditions, fmt.Sprintf("%sgroup_id = $%d", prefix, len(args)+1))
		args = append(args, filters.GroupID)
	}
	if filters.ChannelID > 0 {
		conditions = append(conditions, fmt.Sprintf("%schannel_id = $%d", prefix, len(args)+1))
		args = append(args, filters.ChannelID)
	}
	conditions, args = appendRawUsageLogModelWhereConditionForColumn(conditions, args, prefix+rawUsageLogModelColumn, filters.Model)
	conditions, args = appendRequestTypeOrStreamWhereConditionWithPrefix(conditions, args, prefix, filters.RequestType, filters.Stream)
	if filters.BillingType != nil {
		conditions = append(conditions, fmt.Sprintf("%sbilling_type = $%d", prefix, len(args)+1))
		args = append(args, int16(*filters.BillingType))
	}
	return conditions, args
}

func appendUsageStatsTimeRangeWithPrefix(conditions []string, args []any, prefix string, startTime, endTime *time.Time) ([]string, []any) {
	if startTime != nil {
		conditions = append(conditions, fmt.Sprintf("%screated_at >= $%d", prefix, len(args)+1))
		args = append(args, *startTime)
	}
	if endTime != nil {
		conditions = append(conditions, fmt.Sprintf("%screated_at < $%d", prefix, len(args)+1))
		args = append(args, *endTime)
	}
	return conditions, args
}

func cloneUsageStatsFilters(conditions []string, args []any) ([]string, []any) {
	return append([]string(nil), conditions...), append([]any(nil), args...)
}

func normalizeUsageStatsCacheTotals(stats *UsageStats) {
	if stats == nil {
		return
	}
	stats.TotalCacheTokens = stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
	stats.TotalTokens = stats.TotalInputTokens + stats.TotalOutputTokens + stats.TotalCacheTokens
	totalInputSideTokens := stats.TotalInputTokens + stats.TotalCacheCreationTokens + stats.TotalCacheReadTokens
	if totalInputSideTokens > 0 {
		stats.CacheHitRate = float64(stats.TotalCacheReadTokens) / float64(totalInputSideTokens)
	}
}

func (r *usageLogRepository) queryUsageStatsFromWithConditions(ctx context.Context, fromClause, columnPrefix string, conditions []string, args []any, includeAccountCost bool) (*UsageStats, *float64, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_requests,
			COALESCE(SUM(%[3]sinput_tokens), 0) as total_input_tokens,
			COALESCE(SUM(%[3]soutput_tokens), 0) as total_output_tokens,
			COALESCE(SUM(%[3]scache_creation_tokens), 0) as total_cache_creation_tokens,
			COALESCE(SUM(%[3]scache_read_tokens), 0) as total_cache_read_tokens,
			COALESCE(SUM(%[3]stotal_cost_usd_equivalent), 0) as total_cost,
			COALESCE(SUM(%[3]sactual_cost_usd_equivalent), 0) as total_actual_cost,
			COUNT(*) FILTER (WHERE %[3]sbilling_exempt_reason = 'admin_free') as admin_free_requests,
			COALESCE(SUM(%[3]stotal_cost_usd_equivalent) FILTER (WHERE %[3]sbilling_exempt_reason = 'admin_free'), 0) as admin_free_standard_cost,
			COALESCE(SUM(%[3]stotal_cost_usd_equivalent * COALESCE(%[3]saccount_rate_multiplier, 1)), 0) as total_account_cost,
			COALESCE(AVG(%[3]sduration_ms) FILTER (WHERE %[3]sstatus = 'succeeded'), 0) as avg_duration_ms
		FROM %[1]s
		%[2]s
	`, fromClause, buildWhere(conditions), columnPrefix)

	stats := &UsageStats{}
	var totalAccountCost float64
	if err := scanSingleRow(
		ctx,
		r.sql,
		query,
		args,
		&stats.TotalRequests,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalCacheCreationTokens,
		&stats.TotalCacheReadTokens,
		&stats.TotalCost,
		&stats.TotalActualCost,
		&stats.AdminFreeRequests,
		&stats.AdminFreeStandardCost,
		&totalAccountCost,
		&stats.AverageDurationMs,
	); err != nil {
		return nil, nil, err
	}
	normalizeUsageStatsCacheTotals(stats)

	costByCurrency, actualCostByCurrency, err := queryUsageCostByCurrencyFrom(ctx, r.sql, fromClause, columnPrefix, buildWhere(conditions), args)
	if err != nil {
		return nil, nil, err
	}
	stats.CostByCurrency = costByCurrency
	stats.ActualCostByCurrency = actualCostByCurrency

	if !includeAccountCost {
		return stats, nil, nil
	}
	return stats, &totalAccountCost, nil
}

func (r *usageLogRepository) GetStatsWithFilters(ctx context.Context, filters UsageLogFilters) (*UsageStats, error) {
	platform := normalizeUsagePlatformFilter(filters.Platform)
	withPlatformJoin := platform != ""
	fromClause := "usage_logs"
	columnPrefix := ""
	if withPlatformJoin {
		fromClause = usageLogPlatformJoinFromClause()
		columnPrefix = "ul."
	}
	baseConditions, baseArgs := buildUsageStatsBaseFiltersWithPrefix(filters, columnPrefix)
	if platform != "" {
		baseConditions = append(baseConditions, fmt.Sprintf("%s = $%d", usagePlatformExpression, len(baseArgs)+1))
		baseArgs = append(baseArgs, platform)
	}
	selectedConditions, selectedArgs := cloneUsageStatsFilters(baseConditions, baseArgs)
	selectedConditions, selectedArgs = appendUsageStatsTimeRangeWithPrefix(selectedConditions, selectedArgs, columnPrefix, filters.StartTime, filters.EndTime)

	stats, totalAccountCost, err := r.queryUsageStatsFromWithConditions(ctx, fromClause, columnPrefix, selectedConditions, selectedArgs, filters.AccountID > 0)
	if err != nil {
		return nil, err
	}
	if totalAccountCost != nil {
		stats.TotalAccountCost = totalAccountCost
	}

	if filters.TodayStart != nil || filters.TodayEnd != nil {
		todayConditions, todayArgs := cloneUsageStatsFilters(baseConditions, baseArgs)
		todayConditions, todayArgs = appendUsageStatsTimeRangeWithPrefix(todayConditions, todayArgs, columnPrefix, filters.TodayStart, filters.TodayEnd)
		todayStats, _, todayErr := r.queryUsageStatsFromWithConditions(ctx, fromClause, columnPrefix, todayConditions, todayArgs, false)
		if todayErr != nil {
			return nil, todayErr
		}
		stats.TodayRequests = todayStats.TotalRequests
		stats.TodayInputTokens = todayStats.TotalInputTokens
		stats.TodayOutputTokens = todayStats.TotalOutputTokens
		stats.TodayCacheCreationTokens = todayStats.TotalCacheCreationTokens
		stats.TodayCacheReadTokens = todayStats.TotalCacheReadTokens
		stats.TodayCacheTokens = todayStats.TotalCacheTokens
		stats.TodayTokens = todayStats.TotalTokens
		stats.TodayCacheHitRate = todayStats.CacheHitRate
		stats.TodayCost = todayStats.TotalCost
		stats.TodayActualCost = todayStats.TotalActualCost
		stats.TodayCostByCurrency = todayStats.CostByCurrency
		stats.TodayActualCostByCurrency = todayStats.ActualCostByCurrency
		stats.TodayAverageDurationMs = todayStats.AverageDurationMs
	}

	platformBreakdown, err := r.queryPlatformBreakdown(ctx, filters)
	if err != nil {
		return nil, err
	}
	stats.PlatformBreakdown = platformBreakdown

	return stats, nil
}
