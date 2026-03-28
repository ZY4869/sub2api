package repository

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *accountRepository) GetStatusSummary(ctx context.Context, filters service.AccountStatusSummaryFilters) (*service.AccountStatusSummary, error) {
	normalized := normalizeAdminAccountListFilters(filters.Platform, filters.AccountType, "", filters.Search, filters.GroupID, filters.Lifecycle, filters.PrivacyMode)
	normalized.LimitedView = service.NormalizeAccountLimitedViewInput(filters.LimitedView)
	normalized.LimitedReason = service.NormalizeAccountRateLimitReasonInput(filters.LimitedReason)
	normalized.RuntimeView = service.NormalizeAccountRuntimeViewInput(filters.RuntimeView)
	runtimeFilters := service.AccountRuntimeFiltersFromContext(ctx)
	if runtimeFilters.RuntimeView == service.AccountRuntimeViewInUseOnly {
		normalized.RuntimeView = runtimeFilters.RuntimeView
		normalized.CandidateAccountIDs = runtimeFilters.CandidateAccountIDs
	}
	summary := &service.AccountStatusSummary{
		ByStatus: map[string]int64{
			"active":   0,
			"inactive": 0,
			"error":    0,
		},
		ByPlatform: map[string]int64{},
	}
	if normalized.RuntimeView == service.AccountRuntimeViewInUseOnly && len(normalized.CandidateAccountIDs) == 0 {
		return summary, nil
	}

	baseWhere := []string{"a.deleted_at IS NULL"}
	baseArgs := make([]any, 0, 6)
	baseWhere, baseArgs, _ = appendAdminAccountFilterWhereClauses(baseWhere, baseArgs, 8, normalized, "a", true)
	reasonExpr := accountRateLimitReasonSQL(accountLimitedSQLColumns{
		Extra:            "f.extra",
		RateLimitResetAt: "f.rate_limit_reset_at",
		SessionWindowEnd: "f.session_window_end",
	})
	aggregateQuery := `
		WITH filtered AS (
			SELECT
				a.id,
				a.platform,
				a.status,
				a.schedulable,
				a.rate_limit_reset_at,
				a.temp_unschedulable_until,
				a.overload_until,
				a.extra,
				a.session_window_end
			FROM accounts a
			WHERE ` + strings.Join(baseWhere, " AND ") + `
		),
		classified AS (
			SELECT
				f.id,
				f.platform,
				f.status,
				f.schedulable,
				f.rate_limit_reset_at,
				f.temp_unschedulable_until,
				f.overload_until,
				` + reasonExpr + ` AS rate_limit_reason
			FROM filtered f
		)
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = $1) AS active_count,
			COUNT(*) FILTER (WHERE status = $2 OR status = $4) AS inactive_count,
			COUNT(*) FILTER (WHERE status = $3) AS error_count,
			COUNT(*) FILTER (
				WHERE status = $1
					AND schedulable = TRUE
					AND (rate_limit_reset_at IS NULL OR rate_limit_reset_at <= NOW())
					AND (temp_unschedulable_until IS NULL OR temp_unschedulable_until <= NOW())
					AND (overload_until IS NULL OR overload_until <= NOW())
			) AS dispatchable_count,
			COUNT(*) FILTER (WHERE rate_limit_reset_at IS NOT NULL AND rate_limit_reset_at > NOW()) AS rate_limited_count,
			COUNT(*) FILTER (WHERE temp_unschedulable_until IS NOT NULL AND temp_unschedulable_until > NOW()) AS temp_unschedulable_count,
			COUNT(*) FILTER (WHERE overload_until IS NOT NULL AND overload_until > NOW()) AS overloaded_count,
			COUNT(*) FILTER (WHERE schedulable = FALSE) AS paused_count,
			COUNT(*) FILTER (WHERE rate_limit_reason <> '') AS limited_total,
			COUNT(*) FILTER (WHERE rate_limit_reason = $5) AS limited_rate_429,
			COUNT(*) FILTER (WHERE rate_limit_reason = $6) AS limited_usage_5h,
			COUNT(*) FILTER (WHERE rate_limit_reason = $7) AS limited_usage_7d
		FROM classified
	`
	aggregateArgs := append([]any{
		service.StatusActive,
		service.StatusDisabled,
		service.StatusError,
		"inactive",
		service.AccountRateLimitReason429,
		service.AccountRateLimitReasonUsage5h,
		service.AccountRateLimitReasonUsage7d,
	}, baseArgs...)
	var activeCount int64
	var inactiveCount int64
	var errorCount int64
	if err := r.sql.QueryRowContext(ctx, aggregateQuery, aggregateArgs...).Scan(
		&summary.Total,
		&activeCount,
		&inactiveCount,
		&errorCount,
		&summary.DispatchableCount,
		&summary.RateLimited,
		&summary.TempUnschedulable,
		&summary.Overloaded,
		&summary.Paused,
		&summary.LimitedBreakdown.Total,
		&summary.LimitedBreakdown.Rate429,
		&summary.LimitedBreakdown.Usage5h,
		&summary.LimitedBreakdown.Usage7d,
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
