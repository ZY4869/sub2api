package repository

import (
	"fmt"
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	dbpredicate "github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type accountLimitedSQLColumns struct {
	Platform         string
	Credentials      string
	Extra            string
	RateLimitResetAt string
	SessionWindowEnd string
}

func activeAccountLimitedSQL(rateLimitResetAt string) string {
	return fmt.Sprintf("(%s IS NOT NULL AND %s > NOW())", rateLimitResetAt, rateLimitResetAt)
}

func openAIFamilySQL(platform string) string {
	return fmt.Sprintf("LOWER(BTRIM(COALESCE(%s, ''))) IN ('%s', '%s')", platform, service.PlatformOpenAI, service.PlatformCopilot)
}

func openAIProPlanSQL(credentials string) string {
	return fmt.Sprintf("LOWER(BTRIM(COALESCE(%s->>'plan_type', ''))) = 'pro'", credentials)
}

func nullTrimmedJSONText(column, key string) string {
	return fmt.Sprintf("NULLIF(BTRIM(%s->>'%s'), '')", column, key)
}

func nullTrimmedJSONPathText(column string, path ...string) string {
	return fmt.Sprintf("NULLIF(BTRIM(%s#>>'{%s}'), '')", column, strings.Join(path, ","))
}

func codexUsedPercentSQL(extra, key string) string {
	return fmt.Sprintf("COALESCE((%s)::double precision, 0)", nullTrimmedJSONText(extra, key))
}

func codexResetAtSQL(extra, resetAtKey, resetAfterKey, scope string) string {
	resetAtText := nullTrimmedJSONText(extra, resetAtKey)
	updatedAtText := nullTrimmedJSONText(extra, "codex_usage_updated_at")
	resetAfterText := nullTrimmedJSONText(extra, resetAfterKey)
	modelResetAtText := nullTrimmedJSONPathText(extra, "model_rate_limits", scope, "rate_limit_reset_at")
	return fmt.Sprintf(`COALESCE(
		(%s)::timestamptz,
		CASE
			WHEN %s IS NOT NULL AND %s IS NOT NULL
			THEN (%s)::timestamptz + ((%s)::int * INTERVAL '1 second')
		END,
		(%s)::timestamptz
	)`, resetAtText, updatedAtText, resetAfterText, updatedAtText, resetAfterText, modelResetAtText)
}

func codexScopeActiveSQL(extra, usedPercentKey, resetAtKey, resetAfterKey, scope string) string {
	resetAt := codexResetAtSQL(extra, resetAtKey, resetAfterKey, scope)
	return fmt.Sprintf("((%s) >= 100 AND (%s) IS NOT NULL AND (%s) > NOW())", codexUsedPercentSQL(extra, usedPercentKey), resetAt, resetAt)
}

func accountDisplayRateLimitSQL(cols accountLimitedSQLColumns) string {
	persisted := activeAccountLimitedSQL(cols.RateLimitResetAt)
	if cols.Platform == "" || cols.Credentials == "" {
		return persisted
	}

	isOpenAI := openAIFamilySQL(cols.Platform)
	isPro := openAIProPlanSQL(cols.Credentials)
	normalActive := fmt.Sprintf("(%s OR %s)",
		codexScopeActiveSQL(cols.Extra, "codex_7d_used_percent", "codex_7d_reset_at", "codex_7d_reset_after_seconds", "gpt-5.3-codex"),
		codexScopeActiveSQL(cols.Extra, "codex_5h_used_percent", "codex_5h_reset_at", "codex_5h_reset_after_seconds", "gpt-5.3-codex"),
	)
	sparkActive := fmt.Sprintf("(%s OR %s)",
		codexScopeActiveSQL(cols.Extra, "codex_spark_7d_used_percent", "codex_spark_7d_reset_at", "codex_spark_7d_reset_after_seconds", "gpt-5.3-codex-spark"),
		codexScopeActiveSQL(cols.Extra, "codex_spark_5h_used_percent", "codex_spark_5h_reset_at", "codex_spark_5h_reset_after_seconds", "gpt-5.3-codex-spark"),
	)
	return fmt.Sprintf(`(%s OR (%s AND ((%s AND %s AND %s) OR (NOT %s AND (%s OR %s)))))`,
		persisted,
		isOpenAI,
		isPro,
		normalActive,
		sparkActive,
		isPro,
		normalActive,
		sparkActive,
	)
}

func accountRateLimitReasonSQL(cols accountLimitedSQLColumns) string {
	displayLimited := accountDisplayRateLimitSQL(cols)
	persistedLimited := activeAccountLimitedSQL(cols.RateLimitResetAt)
	storedReason := fmt.Sprintf("COALESCE(NULLIF(BTRIM(%s->>'rate_limit_reason'), ''), '')", cols.Extra)
	isOpenAI := openAIFamilySQL(cols.Platform)
	isPro := openAIProPlanSQL(cols.Credentials)
	normal7d := codexScopeActiveSQL(cols.Extra, "codex_7d_used_percent", "codex_7d_reset_at", "codex_7d_reset_after_seconds", "gpt-5.3-codex")
	normal5h := codexScopeActiveSQL(cols.Extra, "codex_5h_used_percent", "codex_5h_reset_at", "codex_5h_reset_after_seconds", "gpt-5.3-codex")
	spark7d := codexScopeActiveSQL(cols.Extra, "codex_spark_7d_used_percent", "codex_spark_7d_reset_at", "codex_spark_7d_reset_after_seconds", "gpt-5.3-codex-spark")
	spark5h := codexScopeActiveSQL(cols.Extra, "codex_spark_5h_used_percent", "codex_spark_5h_reset_at", "codex_spark_5h_reset_after_seconds", "gpt-5.3-codex-spark")
	passive7d := fmt.Sprintf("COALESCE((%s->>'passive_usage_7d_utilization')::double precision, 0)", cols.Extra)
	session5h := fmt.Sprintf("COALESCE((%s->>'session_window_utilization')::double precision, 0)", cols.Extra)

	return fmt.Sprintf(`CASE
WHEN NOT %s THEN ''
WHEN %s THEN CASE
	WHEN %s IN ('%s', '%s', '%s', '%s') THEN %s
	WHEN %s AND %s AND %s AND %s THEN '%s'
	WHEN %s OR %s OR %s >= 1 THEN '%s'
	WHEN %s OR %s OR %s >= 1 THEN '%s'
	ELSE '%s'
END
WHEN %s AND %s AND %s AND %s THEN '%s'
WHEN %s AND %s AND ((%s OR %s) AND (%s OR %s)) THEN CASE
	WHEN %s OR %s THEN '%s'
	ELSE '%s'
END
WHEN %s AND NOT %s AND (%s OR %s) THEN '%s'
WHEN %s AND NOT %s AND (%s OR %s) THEN '%s'
ELSE '%s'
END`,
		displayLimited,
		persistedLimited,
		storedReason,
		service.AccountRateLimitReason429,
		service.AccountRateLimitReasonUsage5h,
		service.AccountRateLimitReasonUsage7d,
		service.AccountRateLimitReasonUsage7dAll,
		storedReason,
		isOpenAI,
		isPro,
		normal7d,
		spark7d,
		service.AccountRateLimitReasonUsage7dAll,
		normal7d,
		spark7d,
		passive7d,
		service.AccountRateLimitReasonUsage7d,
		normal5h,
		spark5h,
		session5h,
		service.AccountRateLimitReasonUsage5h,
		service.AccountRateLimitReason429,
		isOpenAI,
		isPro,
		normal7d,
		spark7d,
		service.AccountRateLimitReasonUsage7dAll,
		isOpenAI,
		isPro,
		normal7d,
		normal5h,
		spark7d,
		spark5h,
		normal7d,
		spark7d,
		service.AccountRateLimitReasonUsage7d,
		service.AccountRateLimitReasonUsage5h,
		isOpenAI,
		isPro,
		normal7d,
		spark7d,
		service.AccountRateLimitReasonUsage7d,
		isOpenAI,
		isPro,
		normal5h,
		spark5h,
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
			Platform:         s.C("platform"),
			Credentials:      s.C("credentials"),
			Extra:            s.C("extra"),
			RateLimitResetAt: s.C("rate_limit_reset_at"),
			SessionWindowEnd: s.C("session_window_end"),
		}

		switch filters.LimitedView {
		case service.AccountLimitedViewNormalOnly:
			s.Where(entsql.ExprP(fmt.Sprintf("NOT (%s)", accountDisplayRateLimitSQL(cols))))
		case service.AccountLimitedViewLimitedOnly:
			s.Where(entsql.ExprP(accountDisplayRateLimitSQL(cols)))
		}

		if filters.LimitedReason != "" {
			s.Where(entsql.ExprP(fmt.Sprintf("(%s) = ?", accountRateLimitReasonSQL(cols)), filters.LimitedReason))
		}
	})
}

func appendAdminLimitedWhereClauses(whereClauses []string, args []any, argIndex int, filters adminAccountListFilters, tableAlias string) ([]string, []any, int) {
	cols := accountLimitedSQLColumns{
		Platform:         tableAlias + ".platform",
		Credentials:      tableAlias + ".credentials",
		Extra:            tableAlias + ".extra",
		RateLimitResetAt: tableAlias + ".rate_limit_reset_at",
		SessionWindowEnd: tableAlias + ".session_window_end",
	}

	switch filters.LimitedView {
	case service.AccountLimitedViewNormalOnly:
		whereClauses = append(whereClauses, fmt.Sprintf("NOT (%s)", accountDisplayRateLimitSQL(cols)))
	case service.AccountLimitedViewLimitedOnly:
		whereClauses = append(whereClauses, accountDisplayRateLimitSQL(cols))
	}

	if filters.LimitedReason != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(%s) = $%d", accountRateLimitReasonSQL(cols), argIndex))
		args = append(args, filters.LimitedReason)
		argIndex++
	}

	return whereClauses, args, argIndex
}
