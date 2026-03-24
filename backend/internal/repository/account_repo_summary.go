package repository

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *accountRepository) GetStatusSummary(ctx context.Context, filters service.AccountStatusSummaryFilters) (*service.AccountStatusSummary, error) {
	normalized := normalizeAdminAccountListFilters(filters.Platform, filters.AccountType, "", filters.Search, filters.GroupID, filters.Lifecycle)
	summary := &service.AccountStatusSummary{
		ByStatus: map[string]int64{
			"active":   0,
			"inactive": 0,
			"error":    0,
		},
		ByPlatform: map[string]int64{},
	}

	baseWhere := []string{"a.deleted_at IS NULL"}
	baseArgs := make([]any, 0, 6)
	baseWhere, baseArgs, _ = appendAdminAccountFilterWhereClauses(baseWhere, baseArgs, 5, normalized, "a", true)
	aggregateQuery := `
		WITH filtered AS (
			SELECT
				a.id,
				a.platform,
				a.status,
				a.schedulable,
				a.rate_limit_reset_at,
				a.temp_unschedulable_until,
				a.overload_until
			FROM accounts a
			WHERE ` + strings.Join(baseWhere, " AND ") + `
		)
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = $1) AS active_count,
			COUNT(*) FILTER (WHERE status = $2 OR status = $4) AS inactive_count,
			COUNT(*) FILTER (WHERE status = $3) AS error_count,
			COUNT(*) FILTER (WHERE rate_limit_reset_at IS NOT NULL AND rate_limit_reset_at > NOW()) AS rate_limited_count,
			COUNT(*) FILTER (WHERE temp_unschedulable_until IS NOT NULL AND temp_unschedulable_until > NOW()) AS temp_unschedulable_count,
			COUNT(*) FILTER (WHERE overload_until IS NOT NULL AND overload_until > NOW()) AS overloaded_count,
			COUNT(*) FILTER (WHERE schedulable = FALSE) AS paused_count
		FROM filtered
	`
	aggregateArgs := append([]any{service.StatusActive, service.StatusDisabled, service.StatusError, "inactive"}, baseArgs...)
	var activeCount int64
	var inactiveCount int64
	var errorCount int64
	if err := r.sql.QueryRowContext(ctx, aggregateQuery, aggregateArgs...).Scan(
		&summary.Total,
		&activeCount,
		&inactiveCount,
		&errorCount,
		&summary.RateLimited,
		&summary.TempUnschedulable,
		&summary.Overloaded,
		&summary.Paused,
	); err != nil {
		return nil, err
	}
	summary.ByStatus["active"] = activeCount
	summary.ByStatus["inactive"] = inactiveCount
	summary.ByStatus["error"] = errorCount

	platformWhere := []string{"a.deleted_at IS NULL"}
	platformArgs := make([]any, 0, 5)
	platformWhere, platformArgs, _ = appendAdminAccountFilterWhereClauses(platformWhere, platformArgs, 1, normalized, "a", false)
	platformQuery := `
		SELECT a.platform, COUNT(*) AS total
		FROM accounts a
		WHERE ` + strings.Join(platformWhere, " AND ") + `
		GROUP BY a.platform
	`
	rows, err := r.sql.QueryContext(ctx, platformQuery, platformArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var platform string
		var total int64
		if err := rows.Scan(&platform, &total); err != nil {
			return nil, err
		}
		summary.ByPlatform[platform] = total
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return summary, nil
}
