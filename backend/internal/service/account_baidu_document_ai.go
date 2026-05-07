package service

import "strings"

func (a *Account) IsBaiduDocumentAI() bool {
	return RoutingPlatformForAccount(a) == PlatformBaiduDocumentAI
}

// GetBaiduDocumentAIMode returns the explicit routing hint for this account.
// Supported values: "async" and "direct".
func (a *Account) GetBaiduDocumentAIMode() string {
	if a == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(a.GetExtraString("document_ai_mode"))) {
	case "async", "direct":
		return strings.ToLower(strings.TrimSpace(a.GetExtraString("document_ai_mode")))
	default:
		return ""
	}
}

// IsBaiduDocumentAIAsyncMode reports whether the account should be routed
// through the async flow.
func (a *Account) IsBaiduDocumentAIAsyncMode() bool {
	switch a.GetBaiduDocumentAIMode() {
	case "async":
		return true
	case "direct":
		return false
	default:
		return strings.TrimSpace(a.GetCredential("async_bearer_token")) != ""
	}
}

func (a *Account) GetBaiduDocumentAIAsyncBaseURL() string {
	if a == nil {
		return DefaultBaiduDocumentAIAsyncBaseURL()
	}
	baseURL := strings.TrimSpace(a.GetCredential("async_base_url"))
	if baseURL == "" {
		return DefaultBaiduDocumentAIAsyncBaseURL()
	}
	return strings.TrimRight(baseURL, "/")
}

func (a *Account) GetBaiduDocumentAIAsyncBearerToken() string {
	if a == nil {
		return ""
	}
	if value := strings.TrimSpace(a.GetCredential("async_bearer_token")); value != "" {
		return value
	}
	// Compatibility fallback for legacy records that only persisted the direct token.
	return strings.TrimSpace(a.GetCredential("direct_token"))
}

func (a *Account) GetBaiduDocumentAIDirectToken() string {
	if a == nil {
		return ""
	}
	if value := strings.TrimSpace(a.GetCredential("direct_token")); value != "" {
		return value
	}
	return strings.TrimSpace(a.GetCredential("async_bearer_token"))
}

func (a *Account) GetBaiduDocumentAIDirectAPIURL(model string) string {
	if a == nil || a.Credentials == nil {
		return ""
	}
	raw, ok := a.Credentials["direct_api_urls"]
	if !ok || raw == nil {
		return ""
	}
	normalizedModel := normalizeDocumentAIModelID(model)
	switch typed := raw.(type) {
	case map[string]any:
		for key, value := range typed {
			if normalizeDocumentAIModelID(key) != normalizedModel {
				continue
			}
			url := strings.TrimSpace(anyString(value))
			if url != "" {
				return strings.TrimRight(url, "/")
			}
		}
	case map[string]string:
		for key, value := range typed {
			if normalizeDocumentAIModelID(key) != normalizedModel {
				continue
			}
			url := strings.TrimSpace(value)
			if url != "" {
				return strings.TrimRight(url, "/")
			}
		}
	}
	return ""
}
