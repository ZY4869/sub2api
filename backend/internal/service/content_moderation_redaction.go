package service

import "regexp"

var (
	urlRedactionPattern    = regexp.MustCompile(`(?i)\bhttps?://[^\s"'<>]+`)
	bearerRedactionPattern = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)
	secretRedactionPattern = regexp.MustCompile(`(?i)\b(api[_-]?key|authorization|token|secret)\s*[:=]\s*["']?[^"',\s]+`)
)

func redactContentModerationSecrets(value string) string {
	value = urlRedactionPattern.ReplaceAllString(value, "[redacted-url]")
	value = bearerRedactionPattern.ReplaceAllString(value, "Bearer [redacted-token]")
	value = secretRedactionPattern.ReplaceAllString(value, "$1=[redacted]")
	return value
}
