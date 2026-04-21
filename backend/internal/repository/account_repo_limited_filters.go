package repository

import (
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	dbpredicate "github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type accountLimitedSQLColumns struct {
	Extra            string
	RateLimitResetAt string
	SessionWindowEnd string
}

func activeAccountLimitedSQL(rateLimitResetAt string) string {
	return fmt.Sprintf("(%s IS NOT NULL AND %s > NOW())", rateLimitResetAt, rateLimitResetAt)
}

func accountRateLimitReasonSQL(cols accountLimitedSQLColumns) string {
	activeLimited := activeAccountLimitedSQL(cols.RateLimitResetAt)
	storedReason := fmt.Sprintf("COALESCE(NULLIF(BTRIM(%s->>'rate_limit_reason'), ''), '')", cols.Extra)
	all7d := fmt.Sprintf("COALESCE((%s->>'%s')::boolean, FALSE)", cols.Extra, "codex_account_7d_all_exhausted")
	codex7d := fmt.Sprintf("COALESCE((%s->>'codex_7d_used_percent')::double precision, 0)", cols.Extra)
	codex5h := fmt.Sprintf("COALESCE((%s->>'codex_5h_used_percent')::double precision, 0)", cols.Extra)
	passive7d := fmt.Sprintf("COALESCE((%s->>'passive_usage_7d_utilization')::double precision, 0)", cols.Extra)
	session5h := fmt.Sprintf("COALESCE((%s->>'session_window_utilization')::double precision, 0)", cols.Extra)

	return fmt.Sprintf(`CASE
WHEN NOT %s THEN ''
WHEN %s IN ('%s', '%s', '%s', '%s') THEN %s
WHEN %s THEN '%s'
WHEN %s >= 100 OR %s >= 1 THEN '%s'
WHEN %s >= 100 OR %s >= 1 THEN '%s'
ELSE '%s'
END`,
		activeLimited,
		storedReason,
		service.AccountRateLimitReason429,
		service.AccountRateLimitReasonUsage5h,
		service.AccountRateLimitReasonUsage7d,
		service.AccountRateLimitReasonUsage7dAll,
		storedReason,
		all7d,
		service.AccountRateLimitReasonUsage7dAll,
		codex7d,
		passive7d,
		service.AccountRateLimitReasonUsage7d,
		codex5h,
		session5h,
		service.AccountRateLimitReasonUsage5h,
		service.AccountRateLimitReason429,
	)
}

func limitedAccountPredicate(filters adminAccountListFilters) dbpredicate.Account {
	if filters.LimitedView == service.AccountLimitedViewAll && filters.LimitedReason == "" {
		return nil
	}

	return dbpredicate.Account(func(s *entsql.Selector) {
		cols := accountLimitedSQLColumns{
			Extra:            s.C("extra"),
			RateLimitResetAt: s.C("rate_limit_reset_at"),
			SessionWindowEnd: s.C("session_window_end"),
		}

		switch filters.LimitedView {
		case service.AccountLimitedViewNormalOnly:
			s.Where(entsql.ExprP(fmt.Sprintf("(%s IS NULL OR %s <= NOW())", cols.RateLimitResetAt, cols.RateLimitResetAt)))
		case service.AccountLimitedViewLimitedOnly:
			s.Where(entsql.ExprP(activeAccountLimitedSQL(cols.RateLimitResetAt)))
		}

		if filters.LimitedReason != "" {
			s.Where(entsql.ExprP(fmt.Sprintf("(%s) = ?", accountRateLimitReasonSQL(cols)), filters.LimitedReason))
		}
	})
}

func appendAdminLimitedWhereClauses(whereClauses []string, args []any, argIndex int, filters adminAccountListFilters, tableAlias string) ([]string, []any, int) {
	rateLimitResetAt := tableAlias + ".rate_limit_reset_at"

	switch filters.LimitedView {
	case service.AccountLimitedViewNormalOnly:
		whereClauses = append(whereClauses, fmt.Sprintf("(%s IS NULL OR %s <= NOW())", rateLimitResetAt, rateLimitResetAt))
	case service.AccountLimitedViewLimitedOnly:
		whereClauses = append(whereClauses, activeAccountLimitedSQL(rateLimitResetAt))
	}

	if filters.LimitedReason != "" {
		cols := accountLimitedSQLColumns{
			Extra:            tableAlias + ".extra",
			RateLimitResetAt: rateLimitResetAt,
			SessionWindowEnd: tableAlias + ".session_window_end",
		}
		whereClauses = append(whereClauses, fmt.Sprintf("(%s) = $%d", accountRateLimitReasonSQL(cols), argIndex))
		args = append(args, filters.LimitedReason)
		argIndex++
	}

	return whereClauses, args, argIndex
}
