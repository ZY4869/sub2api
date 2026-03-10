package service

import (
	"regexp"
	"strings"
)

var (
	modelCatalogDateSuffixPattern = regexp.MustCompile(`-(?:\d{8}|\d{4}-\d{2}-\d{2})$`)
	openAIReasoningModelPattern   = regexp.MustCompile(`^o\d`)
)

func FormatModelCatalogDisplayName(model string) string {
	canonical := CanonicalizeModelNameForPricing(model)
	if canonical == "" {
		return ""
	}
	trimmed := modelCatalogDateSuffixPattern.ReplaceAllString(canonical, "")
	parts := strings.Split(trimmed, "-")
	if len(parts) == 0 {
		return canonical
	}
	parts[0] = formatModelCatalogBrand(parts[0])
	return strings.Join(parts, "-")
}

func InferModelCatalogIconKey(model string) string {
	canonical := CanonicalizeModelNameForPricing(model)
	switch {
	case strings.HasPrefix(canonical, "claude"):
		return "claude"
	case strings.HasPrefix(canonical, "gemini"):
		return "gemini"
	case strings.HasPrefix(canonical, "gpt"), strings.HasPrefix(canonical, "sora"), strings.HasPrefix(canonical, "codex"), openAIReasoningModelPattern.MatchString(canonical):
		return "chatgpt"
	default:
		return ""
	}
}

func formatModelCatalogBrand(value string) string {
	switch value {
	case "claude":
		return "Claude"
	case "gpt":
		return "GPT"
	case "gemini":
		return "Gemini"
	case "sora":
		return "Sora"
	case "codex":
		return "Codex"
	default:
		if openAIReasoningModelPattern.MatchString(value) {
			return strings.ToUpper(value)
		}
		if value == "" {
			return value
		}
		return strings.ToUpper(value[:1]) + value[1:]
	}
}
