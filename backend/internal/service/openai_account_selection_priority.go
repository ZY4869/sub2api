package service

import (
	"strings"
	"time"
)

const openAIAccountPlanRankUnknown = -1

func openAIAccountPlanType(account *Account) string {
	if account == nil {
		return ""
	}
	planType := strings.TrimSpace(normalizeOpenAIPlanType(account.GetCredential("plan_type")))
	if planType != "" {
		return planType
	}
	return strings.TrimSpace(account.GetCredential("plan_type"))
}

func resolveOpenAIAccountPlanRank(account *Account) (int, bool) {
	if account == nil || !account.IsOpenAI() || account.Type != AccountTypeOAuth {
		return 0, false
	}

	switch openAIAccountPlanType(account) {
	case "pro":
		return 0, true
	case "team":
		return 1, true
	case "plus":
		return 2, true
	case "free":
		return 3, true
	default:
		return 0, false
	}
}

func resolveOpenAIAccountPlanRankForLog(account *Account) int {
	rank, ok := resolveOpenAIAccountPlanRank(account)
	if !ok {
		return openAIAccountPlanRankUnknown
	}
	return rank
}

func compareOpenAIAccountPlanRank(left, right *Account) (int, bool) {
	leftRank, leftOK := resolveOpenAIAccountPlanRank(left)
	rightRank, rightOK := resolveOpenAIAccountPlanRank(right)
	if !leftOK || !rightOK {
		return 0, false
	}
	switch {
	case leftRank < rightRank:
		return -1, true
	case leftRank > rightRank:
		return 1, true
	default:
		return 0, true
	}
}

func compareOpenAIAccountPlanRankValues(left, right int) int {
	if left == openAIAccountPlanRankUnknown || right == openAIAccountPlanRankUnknown {
		return 0
	}
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func resolveOpenAIAccountUsagePressureScope(account *Account, requestedModel string) string {
	if account == nil || !account.IsOpenAI() {
		return ""
	}
	if resolveOpenAICodexQuotaScope(account, requestedModel) == openAICodexScopeSpark {
		return openAICodexScopeSpark
	}
	return openAICodexScopeNormal
}

func buildOpenAICodexScopeUsagePressure(account *Account, scope string, window string, now time.Time) *accountUsagePressure {
	if account == nil || scope == "" {
		return nil
	}
	progress := buildScopedCodexUsageProgressFromExtra(account.Extra, scope, window, now)
	if progress == nil || progress.ResetsAt == nil || !now.Before(*progress.ResetsAt) {
		return nil
	}

	windowRank := 1
	if window == accountUsagePressureWindow5h {
		windowRank = 0
	}

	return &accountUsagePressure{
		scope:       scope,
		window:      window,
		windowRank:  windowRank,
		utilization: normalizeUsagePressureUtilization(progress.Utilization),
		resetAt:     progress.ResetsAt.UTC(),
	}
}

func buildOpenAIAccountUsagePressure(account *Account, requestedModel string, now time.Time) *accountUsagePressure {
	if account == nil || !account.IsOpenAI() {
		return nil
	}

	scope := resolveOpenAIAccountUsagePressureScope(account, requestedModel)
	candidates := []*accountUsagePressure{
		buildOpenAICodexScopeUsagePressure(account, scope, accountUsagePressureWindow5h, now),
		buildOpenAICodexScopeUsagePressure(account, scope, accountUsagePressureWindow7d, now),
	}

	var best *accountUsagePressure
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		if best == nil || compareResolvedAccountUsagePressure(candidate, best) < 0 {
			best = candidate
		}
	}
	return best
}

func compareOpenAIAccountUsagePressure(left, right *Account, requestedModel string, now time.Time) int {
	return compareResolvedAccountUsagePressure(
		buildOpenAIAccountUsagePressure(left, requestedModel, now),
		buildOpenAIAccountUsagePressure(right, requestedModel, now),
	)
}

func compareAccountsByLastUsed(left, right *Account, preferOAuth bool) int {
	if left == nil || right == nil {
		return 0
	}

	switch {
	case left.LastUsedAt == nil && right.LastUsedAt != nil:
		return -1
	case left.LastUsedAt != nil && right.LastUsedAt == nil:
		return 1
	case left.LastUsedAt == nil && right.LastUsedAt == nil:
		if preferOAuth && left.Type != right.Type {
			if left.Type == AccountTypeOAuth {
				return -1
			}
			return 1
		}
		return 0
	default:
		if left.LastUsedAt.Before(*right.LastUsedAt) {
			return -1
		}
		if right.LastUsedAt.Before(*left.LastUsedAt) {
			return 1
		}
		return 0
	}
}

func compareOpenAIAccountsByPriorityPlanAndPressure(left, right *Account, requestedModel string, now time.Time) int {
	if left == nil || right == nil {
		return 0
	}
	if left.Priority != right.Priority {
		if left.Priority < right.Priority {
			return -1
		}
		return 1
	}
	if planCmp, ok := compareOpenAIAccountPlanRank(left, right); ok && planCmp != 0 {
		return planCmp
	}
	if pressureCmp := compareOpenAIAccountUsagePressure(left, right, requestedModel, now); pressureCmp != 0 {
		return pressureCmp
	}
	return 0
}

func compareOpenAIAccountsForSelection(left, right *Account, requestedModel string, now time.Time) int {
	if cmp := compareOpenAIAccountsByPriorityPlanAndPressure(left, right, requestedModel, now); cmp != 0 {
		return cmp
	}
	return compareAccountsByLastUsed(left, right, false)
}

func compareOpenAIAccountsWithLoad(left, right accountWithLoad, requestedModel string, now time.Time) int {
	if left.account == nil || right.account == nil {
		return 0
	}
	if cmp := compareOpenAIAccountsByPriorityPlanAndPressure(left.account, right.account, requestedModel, now); cmp != 0 {
		return cmp
	}
	if left.loadInfo.LoadRate != right.loadInfo.LoadRate {
		if left.loadInfo.LoadRate < right.loadInfo.LoadRate {
			return -1
		}
		return 1
	}
	return compareAccountsByLastUsed(left.account, right.account, false)
}
