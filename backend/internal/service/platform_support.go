package service

import (
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const legacyUnsupportedPrimaryPlatformCopilot = "copilot"

var supportedPrimaryGroupPlatforms = map[string]struct{}{
	PlatformAnthropic:       {},
	PlatformAntigravity:     {},
	PlatformBaiduDocumentAI: {},
	PlatformDeepSeek:        {},
	PlatformGemini:          {},
	PlatformGrok:            {},
	PlatformKiro:            {},
	PlatformOpenAI:          {},
}

func IsUnsupportedPrimaryPlatform(platform string) bool {
	return CanonicalizePlatformValue(platform) == legacyUnsupportedPrimaryPlatformCopilot
}

func UnsupportedPrimaryPlatformError(platform string) error {
	normalized := CanonicalizePlatformValue(platform)
	if normalized == "" {
		normalized = strings.TrimSpace(strings.ToLower(platform))
	}
	return infraerrors.BadRequest("UNSUPPORTED_PLATFORM", "platform is no longer supported: "+normalized)
}

func EnsureSupportedPrimaryPlatform(platform string) error {
	if IsUnsupportedPrimaryPlatform(platform) {
		return UnsupportedPrimaryPlatformError(platform)
	}
	return nil
}

func IsSupportedPrimaryGroupPlatform(platform string) bool {
	normalized := CanonicalizePlatformValue(platform)
	if normalized == "" {
		return false
	}
	_, ok := supportedPrimaryGroupPlatforms[normalized]
	return ok
}

func EnsureValidPrimaryGroupPlatform(platform string) error {
	normalized := CanonicalizePlatformValue(platform)
	if err := EnsureSupportedPrimaryPlatform(normalized); err != nil {
		return err
	}
	if !IsSupportedPrimaryGroupPlatform(normalized) {
		return infraerrors.BadRequest("INVALID_PLATFORM", "invalid platform")
	}
	return nil
}

func EnsureSupportedAccountPlatform(account *Account) error {
	if account == nil {
		return nil
	}
	return EnsureSupportedPrimaryPlatform(account.Platform)
}

func IsUnsupportedRuntimePlatform(platform string) bool {
	return IsUnsupportedPrimaryPlatform(platform)
}

func UnsupportedPrimaryAccountPredicateValues() []string {
	return []string{legacyUnsupportedPrimaryPlatformCopilot}
}
