package service

import "strings"

func NormalizePlatformFamily(platform string) string {
	switch strings.TrimSpace(strings.ToLower(platform)) {
	case PlatformProtocolGateway:
		return PlatformProtocolGateway
	case "claude", PlatformAnthropic, PlatformKiro:
		return PlatformAnthropic
	case PlatformOpenAI, PlatformCopilot:
		return PlatformOpenAI
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
	switch strings.TrimSpace(strings.ToLower(platform)) {
	case PlatformAnthropic, "claude", PlatformAntigravity, PlatformKiro, PlatformCopilot:
		return true
	default:
		return false
	}
}

func DisplayPlatformName(platform string) string {
	switch strings.TrimSpace(strings.ToLower(platform)) {
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
	default:
		return ""
	}
}
