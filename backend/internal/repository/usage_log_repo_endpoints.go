package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

func (r *usageLogRepository) GetEndpointStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) (results []usagestats.EndpointStat, err error) {
	return r.getEndpointStatsWithFilters(ctx, startTime, endTime, "ul.inbound_endpoint", userID, apiKeyID, accountID, groupID, model, requestType, stream, billingType)
}

func (r *usageLogRepository) GetUpstreamEndpointStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) (results []usagestats.EndpointStat, err error) {
	return r.getEndpointStatsWithFilters(ctx, startTime, endTime, "ul.upstream_endpoint", userID, apiKeyID, accountID, groupID, model, requestType, stream, billingType)
}

func (r *usageLogRepository) getEndpointStatsWithFilters(ctx context.Context, startTime, endTime time.Time, endpointField string, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) (results []usagestats.EndpointStat, err error) {
	if r == nil || r.sql == nil {
		return nil, fmt.Errorf("usage log repo is nil")
	}
	endpointField = strings.TrimSpace(endpointField)
	if endpointField == "" {
		return nil, fmt.Errorf("endpointField is empty")
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(%s, '') as endpoint,
			COUNT(*) as requests,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0) as total_cost,
			COALESCE(SUM(actual_cost), 0) as total_actual_cost
		FROM usage_logs ul
		WHERE ul.created_at >= $1 AND ul.created_at <= $2
	`, endpointField)
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
	if model != "" {
		query += fmt.Sprintf(" AND ul.model = $%d", len(args)+1)
		args = append(args, model)
	}
	query, args = appendRequestTypeOrStreamQueryFilter(query, args, requestType, stream)
	if billingType != nil {
		query += fmt.Sprintf(" AND ul.billing_type = $%d", len(args)+1)
		args = append(args, int16(*billingType))
	}

	query += fmt.Sprintf(" GROUP BY %s ORDER BY requests DESC", endpointField)

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

	results = make([]usagestats.EndpointStat, 0)
	for rows.Next() {
		var row usagestats.EndpointStat
		if scanErr := rows.Scan(&row.Endpoint, &row.Requests, &row.TotalTokens, &row.Cost, &row.ActualCost); scanErr != nil {
			return nil, scanErr
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
