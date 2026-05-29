package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

func (r *usageLogRepository) GetUserModelStats(ctx context.Context, userID int64, startTime, endTime time.Time) (results []ModelStat, err error) {
	query := `
		SELECT
			model,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as cost,
			COALESCE(SUM(actual_cost_usd_equivalent), 0) as actual_cost
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
		GROUP BY model
		ORDER BY total_tokens DESC
	`
	rows, err := r.sql.QueryContext(ctx, query, userID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			results = nil
		}
	}()
	results, err = scanModelStatsRows(rows)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *usageLogRepository) GetModelStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8) (results []ModelStat, err error) {
	return r.getModelStatsWithFiltersBySource(ctx, startTime, endTime, userID, apiKeyID, accountID, groupID, channelID, requestType, stream, billingType, usagestats.ModelSourceRequested)
}

func (r *usageLogRepository) GetModelStatsWithFiltersBySource(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8, source string) (results []ModelStat, err error) {
	return r.getModelStatsWithFiltersBySource(ctx, startTime, endTime, userID, apiKeyID, accountID, groupID, channelID, requestType, stream, billingType, source)
}

func (r *usageLogRepository) getModelStatsWithFiltersBySource(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8, source string) (results []ModelStat, err error) {
	actualCostExpr := "COALESCE(SUM(actual_cost_usd_equivalent), 0) as actual_cost"
	if accountID > 0 && userID == 0 && apiKeyID == 0 {
		actualCostExpr = "COALESCE(SUM(total_cost_usd_equivalent * COALESCE(account_rate_multiplier, 1)), 0) as actual_cost"
	}
	modelExpr := resolveModelDimensionExpression(source)
	query := fmt.Sprintf(`
		SELECT
			%s as model,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost_usd_equivalent), 0) as cost,
			%s
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
	`, modelExpr, actualCostExpr)
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
	query, args = appendRequestTypeOrStreamQueryFilter(query, args, requestType, stream)
	if billingType != nil {
		query += fmt.Sprintf(" AND billing_type = $%d", len(args)+1)
		args = append(args, int16(*billingType))
	}
	query += fmt.Sprintf(" GROUP BY %s ORDER BY total_tokens DESC", modelExpr)
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
	results, err = scanModelStatsRows(rows)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func resolveModelDimensionExpression(modelType string) string {
	requestedExpr := "COALESCE(NULLIF(TRIM(requested_model), ''), model)"
	switch usagestats.NormalizeModelSource(modelType) {
	case usagestats.ModelSourceUpstream:
		return fmt.Sprintf("COALESCE(NULLIF(TRIM(upstream_model), ''), %s)", requestedExpr)
	case usagestats.ModelSourceMapping:
		return fmt.Sprintf("(%s || ' -> ' || COALESCE(NULLIF(TRIM(upstream_model), ''), %s))", requestedExpr, requestedExpr)
	default:
		return requestedExpr
	}
}
func (r *usageLogRepository) GetGroupStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8) (results []usagestats.GroupStat, err error) {
	query := `
		SELECT
			COALESCE(ul.group_id, 0) as group_id,
			COALESCE(g.name, '') as group_name,
			COUNT(*) as requests,
			COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(ul.total_cost_usd_equivalent), 0) as cost,
			COALESCE(SUM(ul.actual_cost_usd_equivalent), 0) as actual_cost
		FROM usage_logs ul
		LEFT JOIN groups g ON g.id = ul.group_id
		WHERE ul.created_at >= $1 AND ul.created_at < $2
	`
	args := []any{startTime, endTime}
	if userID > 0 {
		query += fmt.Sprintf(" AND ul.user_id = $%d", len(args)+1)
		args = append(args, userID)
	}
	if apiKeyID > 0 {
		query += fmt.Sprintf(" AND ul.api_key_id = $%d", len(args)+1)
		args = append(args, apiKeyID)
	}
	if accountID > 0 {
		query += fmt.Sprintf(" AND ul.account_id = $%d", len(args)+1)
		args = append(args, accountID)
	}
	if groupID > 0 {
		query += fmt.Sprintf(" AND ul.group_id = $%d", len(args)+1)
		args = append(args, groupID)
	}
	if channelID > 0 {
		query += fmt.Sprintf(" AND ul.channel_id = $%d", len(args)+1)
		args = append(args, channelID)
	}
	query, args = appendRequestTypeOrStreamQueryFilter(query, args, requestType, stream)
	if billingType != nil {
		query += fmt.Sprintf(" AND ul.billing_type = $%d", len(args)+1)
		args = append(args, int16(*billingType))
	}
	query += " GROUP BY ul.group_id, g.name ORDER BY total_tokens DESC"
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
	results = make([]usagestats.GroupStat, 0)
	for rows.Next() {
		var row usagestats.GroupStat
		if err := rows.Scan(&row.GroupID, &row.GroupName, &row.Requests, &row.TotalTokens, &row.Cost, &row.ActualCost); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
