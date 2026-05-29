package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

func (r *usageLogRepository) queryPlatformBreakdown(ctx context.Context, filters UsageLogFilters) (results []usagestats.PlatformUsageStat, err error) {
	platform := normalizeUsagePlatformFilter(filters.Platform)
	conditions, args := buildUsageStatsBaseFiltersWithPrefix(filters, "ul.")
	if platform != "" {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", usagePlatformExpression, len(args)+1))
		args = append(args, platform)
	}
	conditions, args = appendUsageStatsTimeRangeWithPrefix(conditions, args, "ul.", filters.StartTime, filters.EndTime)

	query := fmt.Sprintf(`
		SELECT
			%s AS platform,
			COUNT(*) AS requests,
			COALESCE(SUM(ul.input_tokens), 0) AS input_tokens,
			COALESCE(SUM(ul.output_tokens), 0) AS output_tokens,
			COALESCE(SUM(ul.cache_creation_tokens + ul.cache_read_tokens), 0) AS cache_tokens,
			COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) AS total_tokens,
			COALESCE(SUM(ul.total_cost_usd_equivalent), 0) AS cost,
			COALESCE(SUM(ul.actual_cost_usd_equivalent), 0) AS actual_cost,
			COALESCE(AVG(ul.duration_ms) FILTER (WHERE ul.status = 'succeeded'), 0) AS average_duration_ms
		FROM %s
		%s
		GROUP BY %s
		ORDER BY total_tokens DESC, requests DESC
	`, usagePlatformExpression, usageLogPlatformJoinFromClause(), buildWhere(conditions), usagePlatformExpression)

	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()

	results = make([]usagestats.PlatformUsageStat, 0)
	for rows.Next() {
		var row usagestats.PlatformUsageStat
		if err := rows.Scan(&row.Platform, &row.Requests, &row.InputTokens, &row.OutputTokens, &row.CacheTokens, &row.TotalTokens, &row.Cost, &row.ActualCost, &row.AverageDurationMs); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
