package service

import (
	"strings"
	"time"
)

type AccountDisplayRateLimitProjection struct {
	Limited bool
	ResetAt *time.Time
	Reason  string
}

func AccountDisplayRateLimitState(account *Account, now time.Time) AccountDisplayRateLimitProjection {
	if account == nil {
		return AccountDisplayRateLimitProjection{}
	}
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
		resetAt := account.RateLimitResetAt.UTC()
		return AccountDisplayRateLimitProjection{
			Limited: true,
			ResetAt: &resetAt,
			Reason:  AccountRateLimitReason(account, now),
		}
	}
	if !account.IsOpenAI() {
		return AccountDisplayRateLimitProjection{}
	}

	normal := openAICodexDisplayScopeRateLimitState(account, openAICodexScopeNormal, now)
	spark := openAICodexDisplayScopeRateLimitState(account, openAICodexScopeSpark, now)

	if !isOpenAIProPlan(account) {
		if normal.Limited {
			return normal
		}
		if spark.Limited {
			return spark
		}
		return AccountDisplayRateLimitProjection{}
	}
	if !normal.Limited || !spark.Limited {
		return AccountDisplayRateLimitProjection{}
	}

	if resetAt, ok := codexAccountAll7dResetAtFromExtra(account, account.Extra, now); ok && resetAt != nil {
		return AccountDisplayRateLimitProjection{
			Limited: true,
			ResetAt: resetAt,
			Reason:  AccountRateLimitReasonUsage7dAll,
		}
	}

	reason := AccountRateLimitReasonUsage5h
	if normal.Reason == AccountRateLimitReasonUsage7d || spark.Reason == AccountRateLimitReasonUsage7d {
		reason = AccountRateLimitReasonUsage7d
	}
	return AccountDisplayRateLimitProjection{
		Limited: true,
		ResetAt: earlierTimePtr(normal.ResetAt, spark.ResetAt),
		Reason:  reason,
	}
}

func ApplyAccountDisplayRateLimitProjection(account *Account, now time.Time) AccountDisplayRateLimitProjection {
	state := AccountDisplayRateLimitState(account, now)
	if account == nil || !state.Limited {
		return state
	}
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
		return state
	}

	if state.ResetAt != nil {
		resetAt := state.ResetAt.UTC()
		account.RateLimitResetAt = &resetAt
	}
	if account.RateLimitedAt == nil {
		rateLimitedAt := now.UTC()
		account.RateLimitedAt = &rateLimitedAt
	}
	if strings.TrimSpace(state.Reason) != "" {
		if account.Extra == nil {
			account.Extra = map[string]any{}
		}
		account.Extra["rate_limit_reason"] = state.Reason
	}
	return state
}

func openAICodexDisplayScopeRateLimitState(account *Account, scope string, now time.Time) AccountDisplayRateLimitProjection {
	if account == nil {
		return AccountDisplayRateLimitProjection{}
	}

	resetAt := codexRateLimitResetAtFromExtraForScope(account.Extra, scope, now)
	reason := codexRateLimitReasonFromExtraForScope(account.Extra, scope, now)
	if resetAt == nil {
		if fallback := account.modelRateLimitResetAt(scope); fallback != nil && now.Before(*fallback) {
			normalized := fallback.UTC()
			resetAt = &normalized
		}
	}
	if resetAt == nil {
		return AccountDisplayRateLimitProjection{}
	}
	if reason == "" {
		reason = fallbackOpenAICodexScopeRateLimitReason(account.Extra, scope, now)
	}
	if reason == "" {
		reason = AccountRateLimitReasonUsage7d
	}
	return AccountDisplayRateLimitProjection{
		Limited: true,
		ResetAt: resetAt,
		Reason:  reason,
	}
}

func fallbackOpenAICodexScopeRateLimitReason(extra map[string]any, scope string, now time.Time) string {
	if progress := buildScopedCodexUsageProgressFromExtra(extra, scope, "7d", now); progress != nil && progress.ResetsAt != nil && now.Before(*progress.ResetsAt) && progress.Utilization >= 100 {
		return AccountRateLimitReasonUsage7d
	}
	if progress := buildScopedCodexUsageProgressFromExtra(extra, scope, "5h", now); progress != nil && progress.ResetsAt != nil && now.Before(*progress.ResetsAt) && progress.Utilization >= 100 {
		return AccountRateLimitReasonUsage5h
	}
	return ""
}

func earlierTimePtr(left, right *time.Time) *time.Time {
	switch {
	case left == nil:
		return right
	case right == nil:
		return left
	case right.Before(*left):
		return right
	default:
		return left
	}
}
