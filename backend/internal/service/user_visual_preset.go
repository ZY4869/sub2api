package service

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	VisualPresetClassic = "classic"
	VisualPresetAiry    = "airy"

	VisualPresetPreferenceInherit = "inherit"
)

func NormalizeVisualPreset(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case VisualPresetAiry:
		return VisualPresetAiry
	case VisualPresetClassic:
		fallthrough
	default:
		return VisualPresetClassic
	}
}

func ValidateVisualPreset(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case VisualPresetClassic, VisualPresetAiry:
		return normalized, nil
	default:
		return "", infraerrors.BadRequest(
			"VISUAL_PRESET_INVALID",
			"visual preset must be one of classic, airy",
		)
	}
}

func NormalizeVisualPresetPreference(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case VisualPresetClassic:
		return VisualPresetClassic
	case VisualPresetAiry:
		return VisualPresetAiry
	case VisualPresetPreferenceInherit:
		fallthrough
	default:
		return VisualPresetPreferenceInherit
	}
}

func ValidateVisualPresetPreference(value string, fieldName string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case VisualPresetPreferenceInherit, VisualPresetClassic, VisualPresetAiry:
		return normalized, nil
	default:
		return "", infraerrors.BadRequest(
			"VISUAL_PRESET_PREFERENCE_INVALID",
			fieldName+" must be one of inherit, classic, airy",
		)
	}
}

func ResolveVisualPreset(siteDefault, userPreference, accountOverride string) string {
	effective := NormalizeVisualPreset(siteDefault)
	if preference := NormalizeVisualPresetPreference(userPreference); preference != VisualPresetPreferenceInherit {
		effective = preference
	}
	if override := NormalizeVisualPresetPreference(accountOverride); override != VisualPresetPreferenceInherit {
		effective = override
	}
	return effective
}
