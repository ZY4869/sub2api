package securityaudit

import (
	"regexp"
	"strings"
)

var (
	bearerPattern = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+\-/]+=*`)
	apiKeyPattern = regexp.MustCompile(`(?i)\b(sk|rk|pk|api[_-]?key|token|secret|password)[-_:=\s]+[A-Za-z0-9._~+\-/]{8,}`)
	emailPattern  = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	phonePattern  = regexp.MustCompile(`(?:\+?\d[\d\s().-]{8,}\d)`)
)

func BuildPromptPreview(value string, limit int) string {
	redacted := strings.TrimSpace(RedactPreview(value, limit))
	if redacted == "" {
		return ""
	}
	runes := []rune(redacted)
	if len(runes) < 32 {
		return "***"
	}
	keep := len(runes) / 4
	if keep > 24 {
		keep = 24
	}
	return string(runes[:keep]) + "***…"
}

func BuildFullPrompt(value string, limit int) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\x00", "")
	return TrimRunes(value, limit)
}

func RedactPreview(value string, limit int) string {
	value = bearerPattern.ReplaceAllString(value, "Bearer ***")
	value = apiKeyPattern.ReplaceAllString(value, "$1=***")
	value = emailPattern.ReplaceAllString(value, "***@***")
	value = phonePattern.ReplaceAllString(value, "***PHONE***")
	return TrimRunes(value, limit)
}

func TrimRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}
