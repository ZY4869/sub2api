package repository

import (
	"context"
	"fmt"
	"time"
)

func (r *usageLogRepository) GetUsageTrendWithFilters(ctx context.Context, startTime, endTime time.Time, granularity string, userID, apiKeyID, accountID, groupID, channelID int64, model string, requestType *int16, stream *bool, billingType *int8) (results []TrendDataPoint, err error) {
	if shouldUsePreaggregatedTrend(granularity, userID, apiKeyID, accountID, groupID, channelID, model, requestType, stream, billingType) {
		aggregated, aggregatedErr := r.getUsageTrendFromAggregates(ctx, startTime, endTime, granularity)
		if aggregatedErr == nil && len(aggregated) > 0 {
			return aggregated, nil
		}
	}
	dateFormat := safeDateFormat(granularity)
	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(created_at, '%s') as date,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as actual_cost
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
	`, dateFormat)
	args := []any{startTime, endTime}
	if userID > 0 {
		query += fmt.Sprintf(" AND user_id = $%d", len(args)+1)
		args = append(args, userID)
	}
	if apiKeyID > 0 {
		query += fmt.Sprintf(" AND api_key_id = $%d", len(args)+1)
		args = append(args, apiKeyID)
	}
	if accountID > 0 {
		query += fmt.Sprintf(" AND account_id = $%d", len(args)+1)
		args = append(args, accountID)
	}
	if groupID > 0 {
		query += fmt.Sprintf(" AND group_id = $%d", len(args)+1)
		args = append(args, groupID)
	}
	if channelID > 0 {
		query += fmt.Sprintf(" AND channel_id = $%d", len(args)+1)
		args = append(args, channelID)
	}
	query, args = appendRawUsageLogModelQueryFilter(query, args, model)
	query, args = appendRequestTypeOrStreamQueryFilter(query, args, requestType, stream)
	if billingType != nil {
		query += fmt.Sprintf(" AND billing_type = $%d", len(args)+1)
		args = append(args, int16(*billingType))
	}
	query += " GROUP BY date ORDER BY date ASC"
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
	results, err = scanTrendRows(rows)
	if err != nil {
		return nil, err
	}
	return results, nil
}
func shouldUsePreaggregatedTrend(granularity string, userID, apiKeyID, accountID, groupID, channelID int64, model string, requestType *int16, stream *bool, billingType *int8) bool {
	if granularity != "day" && granularity != "hour" {
		return false
	}
	return userID == 0 && apiKeyID == 0 && accountID == 0 && groupID == 0 && channelID == 0 && model == "" && requestType == nil && stream == nil && billingType == nil
}
func (r *usageLogRepository) getUsageTrendFromAggregates(ctx context.Context, startTime, endTime time.Time, granularity string) (results []TrendDataPoint, err error) {
	dateFormat := safeDateFormat(granularity)
	query := ""
	args := []any{startTime, endTime}
	switch granularity {
	case "hour":
		query = fmt.Sprintf(`
			SELECT
				TO_CHAR(bucket_start, '%s') as date,
				total_requests as requests,
				input_tokens,
				output_tokens,
				cache_creation_tokens,
				cache_read_tokens,
				(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) as total_tokens,
				total_cost as cost,
				actual_cost
			FROM usage_dashboard_hourly
			WHERE bucket_start >= $1 AND bucket_start < $2
			ORDER BY bucket_start ASC
		`, dateFormat)
	case "day":
		query = fmt.Sprintf(`
			SELECT
				TO_CHAR(bucket_date::timestamp, '%s') as date,
				total_requests as requests,
				input_tokens,
				output_tokens,
				cache_creation_tokens,
				cache_read_tokens,
				(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) as total_tokens,
				total_cost as cost,
				actual_cost
			FROM usage_dashboard_daily
			WHERE bucket_date >= $1::date AND bucket_date < $2::date
			ORDER BY bucket_date ASC
		`, dateFormat)
	default:
		return nil, nil
	}
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
	results, err = scanTrendRows(rows)
	if err != nil {
		return nil, err
	}
	return results, nil
}
