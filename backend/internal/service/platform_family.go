package service

import "strings"

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
	case PlatformOpenAI, PlatformCopilot:
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
	case PlatformAnthropic, "claude", PlatformAntigravity, PlatformKiro, PlatformCopilot:
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
	case PlatformCopilot:
		return "Copilot"
	case PlatformGrok:
		return "Grok"
	case PlatformDeepSeek:
		return "DeepSeek"
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
