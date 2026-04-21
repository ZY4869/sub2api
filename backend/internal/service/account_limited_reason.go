package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	AccountLimitedViewAll         = "all"
	AccountLimitedViewNormalOnly  = "normal_only"
	AccountLimitedViewLimitedOnly = "limited_only"

	AccountRateLimitReason429        = "rate_429"
	AccountRateLimitReasonUsage5h    = "usage_5h"
	AccountRateLimitReasonUsage7d    = "usage_7d"
	AccountRateLimitReasonUsage7dAll = "usage_7d_all"
)

type accountLimitedFiltersContextKey struct{}

type AccountLimitedFilters struct {
	LimitedView   string
	LimitedReason string
}

func NormalizeAccountLimitedViewInput(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", AccountLimitedViewAll:
		return AccountLimitedViewAll
	case AccountLimitedViewNormalOnly:
		return AccountLimitedViewNormalOnly
	case AccountLimitedViewLimitedOnly:
		return AccountLimitedViewLimitedOnly
	default:
		return AccountLimitedViewAll
	}
}

func NormalizeAccountRateLimitReasonInput(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case AccountRateLimitReason429:
		return AccountRateLimitReason429
	case AccountRateLimitReasonUsage5h:
		return AccountRateLimitReasonUsage5h
	case AccountRateLimitReasonUsage7d:
		return AccountRateLimitReasonUsage7d
	case AccountRateLimitReasonUsage7dAll:
		return AccountRateLimitReasonUsage7dAll
	default:
		return ""
	}
}

func WithAccountLimitedFilters(ctx context.Context, limitedView, limitedReason string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, accountLimitedFiltersContextKey{}, AccountLimitedFilters{
		LimitedView:   NormalizeAccountLimitedViewInput(limitedView),
		LimitedReason: NormalizeAccountRateLimitReasonInput(limitedReason),
	})
}

func AccountLimitedFiltersFromContext(ctx context.Context) AccountLimitedFilters {
	if ctx == nil {
		return AccountLimitedFilters{LimitedView: AccountLimitedViewAll}
	}
	filters, ok := ctx.Value(accountLimitedFiltersContextKey{}).(AccountLimitedFilters)
	if !ok {
		return AccountLimitedFilters{LimitedView: AccountLimitedViewAll}
	}
	filters.LimitedView = NormalizeAccountLimitedViewInput(filters.LimitedView)
	filters.LimitedReason = NormalizeAccountRateLimitReasonInput(filters.LimitedReason)
	return filters
}

func AccountRateLimitReason(account *Account, now time.Time) string {
	if account == nil || account.RateLimitResetAt == nil || !now.Before(*account.RateLimitResetAt) {
		return ""
	}
	if reason := NormalizeAccountRateLimitReasonInput(parseExtraString(account.Extra["rate_limit_reason"])); reason != "" {
		return reason
	}
	if reason := inferAccountRateLimitReason(account, now); reason != "" {
		return reason
	}
	return AccountRateLimitReason429
}

func inferAccountRateLimitReason(account *Account, now time.Time) string {
	if account == nil {
		return ""
	}
	if resetAt, ok := codexAccountAll7dResetAtFromExtra(account, account.Extra, now); ok && resetAt != nil && now.Before(*resetAt) {
		return AccountRateLimitReasonUsage7dAll
	}
	if progress := buildCodexUsageProgressFromExtra(account.Extra, "7d", now); progress != nil && progress.Utilization >= 100 {
		return AccountRateLimitReasonUsage7d
	}
	if passive7dUtilizationExhausted(account.Extra) {
		return AccountRateLimitReasonUsage7d
	}
	if progress := buildCodexUsageProgressFromExtra(account.Extra, "5h", now); progress != nil && progress.Utilization >= 100 {
		return AccountRateLimitReasonUsage5h
	}
	if sessionWindowUtilizationExhausted(account.Extra) && account.SessionWindowEnd != nil && now.Before(*account.SessionWindowEnd) {
		return AccountRateLimitReasonUsage5h
	}
	return AccountRateLimitReason429
}

func passive7dUtilizationExhausted(extra map[string]any) bool {
	if len(extra) == 0 {
		return false
	}
	return parseExtraFloat64(extra["passive_usage_7d_utilization"]) >= 1
}

func sessionWindowUtilizationExhausted(extra map[string]any) bool {
	if len(extra) == 0 {
		return false
	}
	return parseExtraFloat64(extra["session_window_utilization"]) >= 1
}

func parseExtraString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func setAccountRateLimited(ctx context.Context, repo AccountRepository, accountID int64, resetAt time.Time, reason string) error {
	if err := repo.SetRateLimited(ctx, accountID, resetAt); err != nil {
		return err
	}
	normalizedReason := NormalizeAccountRateLimitReasonInput(reason)
	if normalizedReason == "" {
		normalizedReason = AccountRateLimitReason429
	}
	return repo.UpdateExtra(ctx, accountID, map[string]any{
		"rate_limit_reason": normalizedReason,
	})
}
