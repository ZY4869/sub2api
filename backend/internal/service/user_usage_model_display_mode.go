package service

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	UsageModelDisplayModeModelOnly       = "model_only"
	UsageModelDisplayModeDisplayOnly     = "display_only"
	UsageModelDisplayModeDisplayAndModel = "display_and_model"
)

func NormalizeUserUsageModelDisplayMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case UsageModelDisplayModeDisplayOnly:
		return UsageModelDisplayModeDisplayOnly
	case UsageModelDisplayModeDisplayAndModel:
		return UsageModelDisplayModeDisplayAndModel
	case UsageModelDisplayModeModelOnly:
		fallthrough
	default:
		return UsageModelDisplayModeModelOnly
	}
}

func ValidateUserUsageModelDisplayMode(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case UsageModelDisplayModeModelOnly, UsageModelDisplayModeDisplayOnly, UsageModelDisplayModeDisplayAndModel:
		return normalized, nil
	default:
		return "", infraerrors.BadRequest(
			"USER_USAGE_MODEL_DISPLAY_MODE_INVALID",
			"usage_model_display_mode must be one of model_only, display_only, display_and_model",
		)
	}
}
