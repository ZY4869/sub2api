package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

func (r *usageLogRepository) GetUserSpendingRanking(ctx context.Context, startTime, endTime time.Time, limit int, requestType *int16, metric string) (resp *usagestats.UserSpendingRankingResponse, err error) {
	totalTokensExpr := usageTotalTokensSQL("ul.")
	requestTypeWhere, requestTypeArgs := userSpendingRankingRequestTypeWhere(requestType)
	args := []any{startTime, endTime}
	args = append(args, requestTypeArgs...)
	limitArgIndex := len(args) + 1
	args = append(args, limit)
	query := fmt.Sprintf(`
		WITH user_spend AS (
			SELECT
				ul.user_id,
				u.email,
				COALESCE(SUM(ul.actual_cost_usd_equivalent), 0) as actual_cost_usd,
				COUNT(*) as requests,
				COALESCE(SUM(%s), 0) as tokens
			FROM usage_logs ul
			JOIN users u ON u.id = ul.user_id
			WHERE ul.created_at >= $1 AND ul.created_at <= $2
			%s
			GROUP BY ul.user_id, u.email
		),
		totals AS (
			SELECT
				COALESCE(SUM(actual_cost_usd), 0) as total_actual_cost,
				COALESCE(SUM(requests), 0) as total_requests,
				COALESCE(SUM(tokens), 0) as total_tokens
			FROM user_spend
		)
		SELECT
			us.user_id,
			us.email,
			us.actual_cost_usd,
			us.requests,
			us.tokens,
			t.total_actual_cost,
			t.total_requests,
			t.total_tokens
		FROM user_spend us
		CROSS JOIN totals t
		ORDER BY %s DESC, us.actual_cost_usd DESC, us.requests DESC, us.user_id DESC
		LIMIT $%d
	`, totalTokensExpr, requestTypeWhere, userSpendingRankingOrderColumn(metric), limitArgIndex)

	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			resp = nil
		}
	}()

	out := &usagestats.UserSpendingRankingResponse{
		Ranking: []usagestats.UserSpendingRankingItem{},
		Metric:  userSpendingRankingMetric(metric),
	}
	totalSet := false

	for rows.Next() {
		var row usagestats.UserSpendingRankingItem
		var totalActualCost float64
		var totalRequests int64
		var totalTokens int64
		if scanErr := rows.Scan(
			&row.UserID,
			&row.Email,
			&row.ActualCost,
			&row.Requests,
			&row.Tokens,
			&totalActualCost,
			&totalRequests,
			&totalTokens,
		); scanErr != nil {
			return nil, scanErr
		}
		out.Ranking = append(out.Ranking, row)
		if !totalSet {
			out.TotalActualCost = totalActualCost
			out.TotalRequests = totalRequests
			out.TotalTokens = totalTokens
			totalSet = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func userSpendingRankingRequestTypeWhere(requestType *int16) (string, []any) {
	if requestType == nil {
		return "", nil
	}
	condition, args := buildRequestTypeFilterConditionForColumn(3, "ul.request_type", "ul.stream", "ul.openai_ws_mode", *requestType)
	return "AND " + condition, args
}

func userSpendingRankingMetric(metric string) string {
	switch metric {
	case "requests", "tokens":
		return metric
	default:
		return "actual_cost"
	}
}

func userSpendingRankingOrderColumn(metric string) string {
	switch userSpendingRankingMetric(metric) {
	case "requests":
		return "us.requests"
	case "tokens":
		return "us.tokens"
	default:
		return "us.actual_cost_usd"
	}
}
