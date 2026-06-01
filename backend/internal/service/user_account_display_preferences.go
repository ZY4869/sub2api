package service

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	AccountTodayStatsWindowToday  = "today"
	AccountTodayStatsWindowWeekly = "weekly"
	AccountTodayStatsWindowTotal  = "total"

	AccountGroupDisplayModeFull = "full"
	AccountGroupDisplayModeIcon = "icon"
)

var defaultAccountTodayStatsWindows = []string{
	AccountTodayStatsWindowToday,
	AccountTodayStatsWindowWeekly,
	AccountTodayStatsWindowTotal,
}

func DefaultAccountTodayStatsWindows() []string {
	return append([]string(nil), defaultAccountTodayStatsWindows...)
}

func NormalizeAccountTodayStatsWindows(values []string) []string {
	if len(values) == 0 {
		return DefaultAccountTodayStatsWindows()
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(defaultAccountTodayStatsWindows))
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ToLower(value))
		if !isValidAccountTodayStatsWindow(normalized) {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return DefaultAccountTodayStatsWindows()
	}
	return out
}

func ValidateAccountTodayStatsWindows(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, accountTodayStatsWindowsInvalidError("account_today_stats_windows must contain at least one of today, weekly, total")
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ToLower(value))
		if !isValidAccountTodayStatsWindow(normalized) {
			return nil, accountTodayStatsWindowsInvalidError("account_today_stats_windows must only contain today, weekly, total")
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil, accountTodayStatsWindowsInvalidError("account_today_stats_windows must contain at least one of today, weekly, total")
	}
	return out, nil
}

func accountTodayStatsWindowsInvalidError(message string) error {
	return infraerrors.BadRequest("ACCOUNT_TODAY_STATS_WINDOWS_INVALID", message)
}

func isValidAccountTodayStatsWindow(value string) bool {
	switch value {
	case AccountTodayStatsWindowToday, AccountTodayStatsWindowWeekly, AccountTodayStatsWindowTotal:
		return true
	default:
		return false
	}
}

func NormalizeAccountGroupDisplayMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case AccountGroupDisplayModeIcon:
		return AccountGroupDisplayModeIcon
	case AccountGroupDisplayModeFull:
		return AccountGroupDisplayModeFull
	default:
		return AccountGroupDisplayModeFull
	}
}

func ValidateAccountGroupDisplayMode(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case AccountGroupDisplayModeFull, AccountGroupDisplayModeIcon:
		return normalized, nil
	default:
		return "", infraerrors.BadRequest(
			"ACCOUNT_GROUP_DISPLAY_MODE_INVALID",
			"account_group_display_mode must be one of full, icon",
		)
	}
}
