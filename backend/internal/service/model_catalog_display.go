package service

import (
	"regexp"
	"strings"
)

var (
	modelCatalogDateVersionSuffixPattern = regexp.MustCompile(`-(?:\d{8}|\d{4}-\d{2}-\d{2})(?:-[^-\s]+:\d+)?$`)
	openAIReasoningModelPattern          = regexp.MustCompile(`^o\d`)
)

func NormalizeModelCatalogModelID(model string) string {
	return normalizeModelCatalogAlias(model)
}

func FormatModelCatalogDisplayName(model string) string {
	canonical := NormalizeModelCatalogModelID(model)
	if canonical == "" {
		return ""
	}
	parts := strings.FieldsFunc(canonical, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return canonical
	}
	formatted := make([]string, 0, len(parts))
	for index := 0; index < len(parts); index++ {
		current := parts[index]
		if shouldMergeModelCatalogVersionToken(current, parts, index) {
			formatted = append(formatted, current+"."+parts[index+1])
			index++
			continue
		}
		formatted = append(formatted, formatModelCatalogToken(current, len(formatted) == 0))
	}
	return strings.Join(formatted, " ")
}

func InferModelCatalogIconKey(model string) string {
	canonical := NormalizeModelCatalogModelID(model)
	switch {
	case strings.HasPrefix(canonical, "claude"):
		return "claude"
	case strings.HasPrefix(canonical, "gemini"):
		return "gemini"
	case strings.HasPrefix(canonical, "gpt"), strings.HasPrefix(canonical, "codex"), openAIReasoningModelPattern.MatchString(canonical):
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

func formatModelCatalogToken(value string, isFirst bool) string {
	if isFirst {
		return formatModelCatalogBrand(value)
	}
	if value == "" {
		return value
	}
	if value[0] < 'a' || value[0] > 'z' {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func shouldMergeModelCatalogVersionToken(current string, parts []string, index int) bool {
	if index+1 >= len(parts) {
		return false
	}
	return isShortModelCatalogNumericToken(current) && isShortModelCatalogNumericToken(parts[index+1])
}

func isShortModelCatalogNumericToken(value string) bool {
	if len(value) == 0 || len(value) > 2 {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
