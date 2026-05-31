package repository

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *usageLogRepository) GetModelSuccessRates7d(ctx context.Context, models []string, now time.Time) (map[string]service.ModelSuccessRateSnapshot, error) {
	out := map[string]service.ModelSuccessRateSnapshot{}
	normalized := make([]string, 0, len(models))
	seen := map[string]struct{}{}
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		normalized = append(normalized, model)
	}
	if len(normalized) == 0 {
		return out, nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT
			COALESCE(NULLIF(TRIM(requested_model), ''), model) AS display_model,
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'succeeded') AS succeeded
		FROM usage_logs
		WHERE created_at >= $1
		  AND created_at < $2
		  AND COALESCE(NULLIF(TRIM(requested_model), ''), model) = ANY($3)
		GROUP BY display_model
	`, now.AddDate(0, 0, -7), now, pq.Array(normalized))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			model     string
			total     int64
			succeeded int64
		)
		if err := rows.Scan(&model, &total, &succeeded); err != nil {
			return nil, err
		}
		out[model] = service.ModelSuccessRateSnapshot{
			Rate:   usageModelSuccessRate(total, succeeded),
			Status: usageModelSuccessStatus(total, succeeded),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, model := range normalized {
		if _, ok := out[model]; !ok {
			out[model] = service.ModelSuccessRateSnapshot{Status: "unknown"}
		}
	}
	return out, nil
}

func usageModelSuccessRate(total, succeeded int64) *float64 {
	if total <= 0 {
		return nil
	}
	rate := float64(succeeded) / float64(total)
	return &rate
}

func usageModelSuccessStatus(total, succeeded int64) string {
	rate := usageModelSuccessRate(total, succeeded)
	if rate == nil {
		return "unknown"
	}
	switch {
	case *rate >= 0.98:
		return "healthy"
	case *rate >= 0.90:
		return "warning"
	default:
		return "error"
	}
}
