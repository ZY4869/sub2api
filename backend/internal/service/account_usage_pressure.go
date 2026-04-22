package service

import "time"

const (
	accountUsagePressureWindow5h = "5h"
	accountUsagePressureWindow7d = "7d"
)

type accountUsagePressure struct {
	scope       string
	window      string
	windowRank  int
	utilization float64
	resetAt     time.Time
}

func buildAccountUsagePressure(account *Account, now time.Time) *accountUsagePressure {
	if account == nil {
		return nil
	}
	candidates := []*accountUsagePressure{
		buildAnthropicSessionUsagePressure(account, now),
		buildAnthropicSevenDayUsagePressure(account, now),
		buildCodexUsagePressure(account, accountUsagePressureWindow5h, now),
		buildCodexUsagePressure(account, accountUsagePressureWindow7d, now),
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

func buildAnthropicSessionUsagePressure(account *Account, now time.Time) *accountUsagePressure {
	if account == nil || account.SessionWindowEnd == nil || !now.Before(*account.SessionWindowEnd) {
		return nil
	}
	raw, ok := account.Extra["session_window_utilization"]
	if !ok {
		return nil
	}
	return &accountUsagePressure{
		window:      accountUsagePressureWindow5h,
		windowRank:  0,
		utilization: normalizeUsagePressureUtilization(parseExtraFloat64(raw)),
		resetAt:     account.SessionWindowEnd.UTC(),
	}
}

func buildAnthropicSevenDayUsagePressure(account *Account, now time.Time) *accountUsagePressure {
	if account == nil {
		return nil
	}
	rawUtilization, ok := account.Extra["passive_usage_7d_utilization"]
	if !ok {
		return nil
	}
	rawReset, ok := account.Extra["passive_usage_7d_reset"]
	if !ok {
		return nil
	}
	resetAtUnix := int64(parseExtraFloat64(rawReset))
	if resetAtUnix <= 0 {
		return nil
	}
	resetAt := time.Unix(resetAtUnix, 0).UTC()
	if !now.Before(resetAt) {
		return nil
	}
	return &accountUsagePressure{
		window:      accountUsagePressureWindow7d,
		windowRank:  1,
		utilization: normalizeUsagePressureUtilization(parseExtraFloat64(rawUtilization)),
		resetAt:     resetAt,
	}
}

func buildCodexUsagePressure(account *Account, window string, now time.Time) *accountUsagePressure {
	if account == nil {
		return nil
	}
	progress := buildCodexUsageProgressFromExtra(account.Extra, window, now)
	if progress == nil || progress.ResetsAt == nil || !now.Before(*progress.ResetsAt) {
		return nil
	}
	windowRank := 1
	if window == accountUsagePressureWindow5h {
		windowRank = 0
	}
	return &accountUsagePressure{
		window:      window,
		windowRank:  windowRank,
		utilization: normalizeUsagePressureUtilization(progress.Utilization),
		resetAt:     progress.ResetsAt.UTC(),
	}
}

func normalizeUsagePressureUtilization(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value <= 1:
		return value * 100
	case value > 100:
		return 100
	default:
		return value
	}
}

func compareAccountUsagePressure(left, right *Account, now time.Time) int {
	return compareResolvedAccountUsagePressure(buildAccountUsagePressure(left, now), buildAccountUsagePressure(right, now))
}

func compareResolvedAccountUsagePressure(left, right *accountUsagePressure) int {
	if left == nil || right == nil {
		return 0
	}
	if left.windowRank != right.windowRank {
		if left.windowRank < right.windowRank {
			return -1
		}
		return 1
	}
	if left.utilization != right.utilization {
		if left.utilization > right.utilization {
			return -1
		}
		return 1
	}
	if !left.resetAt.Equal(right.resetAt) {
		if left.resetAt.Before(right.resetAt) {
			return -1
		}
		return 1
	}
	return 0
}

func filterByBestAccountUsagePressure(accounts []accountWithLoad, now time.Time) []accountWithLoad {
	if len(accounts) == 0 {
		return accounts
	}
	var best *Account
	for _, candidate := range accounts {
		if buildAccountUsagePressure(candidate.account, now) == nil {
			continue
		}
		if best == nil || compareAccountUsagePressure(candidate.account, best, now) < 0 {
			best = candidate.account
		}
	}
	if best == nil {
		return accounts
	}
	filtered := make([]accountWithLoad, 0, len(accounts))
	for _, candidate := range accounts {
		if compareAccountUsagePressure(candidate.account, best, now) == 0 {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func accountUsagePressureLogValues(pressure *accountUsagePressure) (string, float64, string) {
	if pressure == nil {
		return "", 0, ""
	}
	resetAt := ""
	if !pressure.resetAt.IsZero() {
		resetAt = pressure.resetAt.UTC().Format(time.RFC3339)
	}
	return pressure.window, pressure.utilization, resetAt
}
