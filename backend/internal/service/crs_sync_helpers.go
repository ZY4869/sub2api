package service

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

func mergeMap(existing map[string]any, updates map[string]any) map[string]any {
	out := make(map[string]any, len(existing)+len(updates))
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range updates {
		out[k] = v
	}
	return out
}

func defaultName(name, id string) string {
	if strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return "CRS " + id
}

func clampPriority(priority int) int {
	if priority < 1 || priority > 100 {
		return 50
	}
	return priority
}

func sanitizeCredentialsMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for k, v := range input {
		// Avoid nil values to keep JSONB cleaner
		if v != nil {
			out[k] = v
		}
	}
	return out
}

func mapCRSStatus(isActive bool, status string) string {
	if !isActive {
		return "inactive"
	}
	if strings.EqualFold(strings.TrimSpace(status), "error") {
		return "error"
	}
	return "active"
}

func normalizeBaseURL(raw string, allowlist []string, allowPrivate bool) (string, error) {
	// Do not require allowlist when it is empty; only run basic URL and SSRF validation.
	requireAllowlist := len(allowlist) > 0
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     allowlist,
		RequireAllowlist: requireAllowlist,
		AllowPrivate:     allowPrivate,
	})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}

// cleanBaseURL removes trailing suffix from base_url in credentials
// Used for both Claude and OpenAI accounts to remove /v1
func cleanBaseURL(credentials map[string]any, suffixToRemove string) {
	if baseURL, ok := credentials["base_url"].(string); ok && baseURL != "" {
		trimmed := strings.TrimSpace(baseURL)
		if strings.HasSuffix(trimmed, suffixToRemove) {
			credentials["base_url"] = strings.TrimSuffix(trimmed, suffixToRemove)
		}
	}
}
