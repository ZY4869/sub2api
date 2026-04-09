package service

import "strings"

var providerLabelOverrides = map[string]string{
	PlatformOpenAI:      "OpenAI-GPT",
	PlatformAnthropic:   "Anthropic-Claude",
	PlatformGemini:      "Google-Gemini",
	PlatformGrok:        "xAI-Grok",
	PlatformAntigravity: "Antigravity",
	PlatformCopilot:     "GitHub-Copilot",
	PlatformKiro:        "Kiro",
}

func ProviderLabelCatalog() map[string]string {
	cloned := make(map[string]string, len(providerLabelOverrides))
	for key, value := range providerLabelOverrides {
		cloned[key] = value
	}
	return cloned
}

func NormalizeModelProvider(provider string) string {
	return strings.TrimSpace(strings.ToLower(provider))
}

func ProviderForPlatform(platform string) string {
	switch NormalizePlatformFamily(platform) {
	case PlatformOpenAI:
		if normalized := NormalizeModelProvider(platform); normalized == PlatformGrok || normalized == PlatformCopilot {
			return normalized
		}
		return PlatformOpenAI
	case PlatformAnthropic:
		if normalized := NormalizeModelProvider(platform); normalized == PlatformKiro {
			return normalized
		}
		return PlatformAnthropic
	default:
		return NormalizeModelProvider(platform)
	}
}

func FormatProviderLabel(provider string) string {
	normalized := NormalizeModelProvider(provider)
	if normalized == "" {
		return "Unknown"
	}
	if label, ok := providerLabelOverrides[normalized]; ok {
		return label
	}

	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return "Unknown"
	}
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "-")
}

func BuildProviderDisplayName(provider string, displayName string) string {
	return BuildProviderDisplayNameWithLabel(provider, "", displayName, "")
}

func BuildProviderDisplayNameWithLabel(provider string, providerLabel string, displayName string, fallbackID string) string {
	label := strings.TrimSpace(providerLabel)
	if label == "" {
		label = strings.TrimSpace(FormatProviderLabel(provider))
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = strings.TrimSpace(fallbackID)
	}
	switch {
	case label == "":
		return name
	case name == "":
		return label
	default:
		return label + " " + name
	}
}

func FinalDisplayNameSortKey(provider string, providerLabel string, displayName string, fallbackID string) string {
	return strings.ToLower(strings.TrimSpace(BuildProviderDisplayNameWithLabel(provider, providerLabel, displayName, fallbackID)))
}
