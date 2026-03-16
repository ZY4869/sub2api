package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

func (r *usageLogRepository) GetUserSpendingRanking(ctx context.Context, startTime, endTime time.Time, limit int) (resp *usagestats.UserSpendingRankingResponse, err error) {
	query := `
		WITH user_spend AS (
			SELECT
				ul.user_id,
				u.email,
				COALESCE(SUM(ul.actual_cost), 0) as actual_cost,
				COUNT(*) as requests,
				COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) as tokens
			FROM usage_logs ul
			JOIN users u ON u.id = ul.user_id
			WHERE ul.created_at >= $1 AND ul.created_at <= $2
			GROUP BY ul.user_id, u.email
		),
		totals AS (
			SELECT
				COALESCE(SUM(actual_cost), 0) as total_actual_cost,
				COALESCE(SUM(requests), 0) as total_requests,
				COALESCE(SUM(tokens), 0) as total_tokens
			FROM user_spend
		)
		SELECT
			us.user_id,
			us.email,
			us.actual_cost,
			us.requests,
			us.tokens,
			t.total_actual_cost,
			t.total_requests,
			t.total_tokens
		FROM user_spend us
		CROSS JOIN totals t
		ORDER BY us.actual_cost DESC, us.requests DESC, us.user_id DESC
		LIMIT $3
	`

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, limit)
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
