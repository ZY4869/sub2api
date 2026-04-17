package service

import "strings"

func (a *Account) IsBaiduDocumentAI() bool {
	return RoutingPlatformForAccount(a) == PlatformBaiduDocumentAI
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
	return strings.TrimSpace(a.GetCredential("async_bearer_token"))
}

func (a *Account) GetBaiduDocumentAIDirectToken() string {
	if a == nil {
		return ""
	}
	return strings.TrimSpace(a.GetCredential("direct_token"))
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
