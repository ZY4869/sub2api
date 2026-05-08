package service

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	UsageContextBadgeDisplayModeRequestOnly = "request_only"
	UsageContextBadgeDisplayModeNativeOnly  = "native_only"
	UsageContextBadgeDisplayModeBoth        = "both"
)

func NormalizeUserUsageContextBadgeDisplayMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case UsageContextBadgeDisplayModeNativeOnly:
		return UsageContextBadgeDisplayModeNativeOnly
	case UsageContextBadgeDisplayModeBoth:
		return UsageContextBadgeDisplayModeBoth
	case UsageContextBadgeDisplayModeRequestOnly:
		fallthrough
	default:
		return UsageContextBadgeDisplayModeRequestOnly
	}
}

func ValidateUserUsageContextBadgeDisplayMode(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case UsageContextBadgeDisplayModeRequestOnly, UsageContextBadgeDisplayModeNativeOnly, UsageContextBadgeDisplayModeBoth:
		return normalized, nil
	default:
		return "", infraerrors.BadRequest(
			"USER_USAGE_CONTEXT_BADGE_DISPLAY_MODE_INVALID",
			"usage_context_badge_display_mode must be one of request_only, native_only, both",
		)
	}
}
