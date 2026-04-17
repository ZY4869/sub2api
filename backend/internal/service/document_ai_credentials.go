package service

import "strings"

const defaultBaiduDocumentAIAsyncBaseURL = "https://paddleocr.aistudio-app.com/api/v2/ocr"

func DefaultBaiduDocumentAIAsyncBaseURL() string {
	return defaultBaiduDocumentAIAsyncBaseURL
}

func normalizeBaiduDocumentAICredentialsForStorage(credentials map[string]any) map[string]any {
	if len(credentials) == 0 {
		return map[string]any{
			"async_base_url": defaultBaiduDocumentAIAsyncBaseURL,
		}
	}
	normalized := make(map[string]any, len(credentials)+1)
	asyncBaseURL := strings.TrimRight(strings.TrimSpace(anyString(credentials["async_base_url"])), "/")
	if asyncBaseURL == "" {
		asyncBaseURL = defaultBaiduDocumentAIAsyncBaseURL
	}
	normalized["async_base_url"] = asyncBaseURL
	if value := strings.TrimSpace(anyString(credentials["async_bearer_token"])); value != "" {
		normalized["async_bearer_token"] = value
	}
	if value := strings.TrimSpace(anyString(credentials["direct_token"])); value != "" {
		normalized["direct_token"] = value
	}
	if directAPIURLs := normalizeBaiduDocumentAIDirectAPIURLs(credentials["direct_api_urls"]); len(directAPIURLs) > 0 {
		normalized["direct_api_urls"] = directAPIURLs
	}
	return normalized
}

func normalizeBaiduDocumentAIDirectAPIURLs(raw any) map[string]any {
	normalized := map[string]any{}
	switch typed := raw.(type) {
	case map[string]any:
		for modelID, value := range typed {
			key := strings.TrimSpace(modelID)
			url := strings.TrimSpace(anyString(value))
			if key == "" || url == "" {
				continue
			}
			normalized[key] = strings.TrimRight(url, "/")
		}
	case map[string]string:
		for modelID, value := range typed {
			key := strings.TrimSpace(modelID)
			url := strings.TrimSpace(value)
			if key == "" || url == "" {
				continue
			}
			normalized[key] = strings.TrimRight(url, "/")
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func anyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}
