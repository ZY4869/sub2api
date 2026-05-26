package service

import (
	"sort"
	"strings"
)

func CanonicalizePlatformValue(platform string) string {
	switch strings.TrimSpace(strings.ToLower(platform)) {
	case "baidu":
		return PlatformBaiduDocumentAI
	default:
		return strings.TrimSpace(strings.ToLower(platform))
	}
}

func NormalizePlatformFamily(platform string) string {
	switch CanonicalizePlatformValue(platform) {
	case PlatformProtocolGateway:
		return PlatformProtocolGateway
	case "claude", PlatformAnthropic, PlatformKiro:
		return PlatformAnthropic
	case PlatformOpenAI:
		return PlatformOpenAI
	case "baidu", PlatformBaiduDocumentAI:
		return PlatformBaiduDocumentAI
	default:
		return strings.TrimSpace(strings.ToLower(platform))
	}
}

func IsAnthropicFamily(platform string) bool {
	return NormalizePlatformFamily(platform) == PlatformAnthropic
}

func IsOpenAIFamily(platform string) bool {
	return NormalizePlatformFamily(platform) == PlatformOpenAI
}

func SupportsMixedChannelPlatform(platform string) bool {
	switch CanonicalizePlatformValue(platform) {
	case PlatformAnthropic, "claude", PlatformAntigravity, PlatformKiro:
		return true
	default:
		return false
	}
}

func DisplayPlatformName(platform string) string {
	switch CanonicalizePlatformValue(platform) {
	case PlatformAntigravity:
		return "Antigravity"
	case PlatformProtocolGateway:
		return "Protocol Gateway"
	case PlatformAnthropic, "claude":
		return "Anthropic"
	case PlatformKiro:
		return "Kiro"
	case PlatformGrok:
		return "Grok"
	case PlatformDeepSeek:
		return "DeepSeek"
	case PlatformOpenRouter:
		return "OpenRouter"
	case PlatformBaiduDocumentAI:
		return "百度文档智能"
	default:
		return ""
	}
}

func IsGrokPlatform(platform string) bool {
	return CanonicalizePlatformValue(platform) == PlatformGrok
}

func IsDeepSeekPlatform(platform string) bool {
	return CanonicalizePlatformValue(platform) == PlatformDeepSeek
}

func PlatformDisplayEnglishName(platform string) string {
	switch CanonicalizePlatformValue(platform) {
	case PlatformAnthropic:
		return "Anthropic"
	case PlatformAntigravity:
		return "Antigravity"
	case PlatformBaiduDocumentAI:
		return "Baidu Document AI"
	case PlatformDeepSeek:
		return "DeepSeek"
	case PlatformGemini:
		return "Google"
	case PlatformGrok:
		return "Grok"
	case PlatformKiro:
		return "Kiro"
	case PlatformOpenAI:
		return "OpenAI"
	case PlatformOpenRouter:
		return "OpenRouter"
	case PlatformProtocolGateway:
		return "Protocol Gateway"
	default:
		return strings.TrimSpace(platform)
	}
}

func SortPlatformKeysForDisplay(platforms []string) []string {
	if len(platforms) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		if IsUnsupportedPrimaryPlatform(platform) {
			continue
		}
		filtered = append(filtered, platform)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left := PlatformDisplayEnglishName(filtered[i])
		right := PlatformDisplayEnglishName(filtered[j])
		if left == right {
			return filtered[i] < filtered[j]
		}
		return left < right
	})
	return filtered
}
