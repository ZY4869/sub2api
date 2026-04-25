package service

import "strings"

func isHardRemovedModelID(modelID string) bool {
	normalized := normalizeRegistryID(modelID)
	if normalized == "" {
		return false
	}
	if strings.HasPrefix(normalized, "gpt-5.1") {
		return true
	}
	if strings.HasPrefix(normalized, "gpt-5-codex") {
		return true
	}
	if strings.HasPrefix(normalized, "gpt-5.2-codex") {
		return true
	}
	if strings.HasPrefix(normalized, "gpt-5.3-codex") && !strings.HasPrefix(normalized, "gpt-5.3-codex-spark") {
		return true
	}
	switch normalized {
	case "codex-mini-latest":
		return true
	default:
		return false
	}
}
