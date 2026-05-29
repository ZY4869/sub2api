package dto

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var accountListLiteCredentialAllowlist = map[string]struct{}{
	"plan_type":               {},
	"plan_type_raw":           {},
	"plan_type_label":         {},
	"pro_multiplier":          {},
	"subscription_expires_at": {},
	"tier_id":                 {},
	"oauth_type":              {},
	"gemini_api_variant":      {},
}

const accountCredentialMaskedValue = "__sub2api_credential_redacted__"

var accountCredentialSensitiveKeys = map[string]struct{}{
	"access_token":                {},
	"api_key":                     {},
	"async_bearer_token":          {},
	"client_secret":               {},
	"direct_token":                {},
	"id_token":                    {},
	"password":                    {},
	"private_key":                 {},
	"refresh_token":               {},
	"secret":                      {},
	"secret_access_key":           {},
	"sso_token":                   {},
	"token":                       {},
	"vertex_service_account_json": {},
}

func RedactAccountCredentials(credentials map[string]any) map[string]any {
	if credentials == nil {
		return nil
	}
	redacted := make(map[string]any, len(credentials))
	for key, value := range credentials {
		if isSensitiveAccountCredentialKey(key) && hasCredentialValue(value) {
			redacted[key] = accountCredentialMaskedValue
			continue
		}
		redacted[key] = value
	}
	return redacted
}

func isSensitiveAccountCredentialKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if _, ok := accountCredentialSensitiveKeys[normalized]; ok {
		return true
	}
	return strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "api_key") ||
		strings.Contains(normalized, "apikey") ||
		strings.Contains(normalized, "private_key")
}

func hasCredentialValue(value any) bool {
	if value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
}

func isAccountActiveUsageAvailable(a *service.Account) bool {
	if a == nil {
		return false
	}

	switch service.EffectiveProtocol(a) {
	case service.PlatformAnthropic:
		return a.Type == service.AccountTypeOAuth &&
			strings.TrimSpace(a.GetCredential("access_token")) != ""
	case service.PlatformOpenAI:
		return a.Type == service.AccountTypeOAuth &&
			strings.TrimSpace(a.GetOpenAIAccessToken()) != ""
	default:
		return false
	}
}
