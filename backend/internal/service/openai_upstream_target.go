package service

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	copilotDefaultUserAgent          = "GithubCopilot/1.0"
	copilotDefaultEditorVersion      = "vscode/1.100.0"
	copilotDefaultPluginVersion      = "copilot/1.300.0"
	copilotDefaultIntegrationID      = "vscode-chat"
	copilotDefaultOpenAIIntent       = "conversation-panel"
	copilotGitHubAPIVersion          = "2025-10-01"
	openaiPlatformChatCompletionsURL = "https://api.openai.com/v1/chat/completions"
	openaiPlatformImagesURL          = "https://api.openai.com/v1/images"
)

func isChatGPTOpenAIOAuthAccount(account *Account) bool {
	return account != nil && account.Type == AccountTypeOAuth && (account.Platform == "" || account.Platform == PlatformOpenAI)
}

func isCopilotOAuthAccount(account *Account) bool {
	return account != nil && account.Platform == PlatformCopilot && account.Type == AccountTypeOAuth
}

func resolveOpenAIResponsesTargetURL(account *Account, validateBaseURL func(string) (string, error)) (string, error) {
	if account == nil {
		return openaiPlatformAPIURL, nil
	}
	if isChatGPTOpenAIOAuthAccount(account) {
		return chatgptCodexURL, nil
	}

	baseURL := strings.TrimSpace(account.GetOpenAIBaseURL())
	if baseURL == "" && account.Platform != PlatformCopilot {
		return openaiPlatformAPIURL, nil
	}
	if validateBaseURL != nil && baseURL != "" {
		validatedURL, err := validateBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		baseURL = validatedURL
	}

	return buildOpenAIResponsesURLForPlatform(baseURL, account.Platform), nil
}

func resolveOpenAIChatCompletionsTargetURL(account *Account, validateBaseURL func(string) (string, error)) (string, error) {
	if account == nil {
		return openaiPlatformChatCompletionsURL, nil
	}

	baseURL := strings.TrimSpace(account.GetOpenAIBaseURL())
	if baseURL == "" && account.Platform != PlatformCopilot {
		return openaiPlatformChatCompletionsURL, nil
	}
	if validateBaseURL != nil && baseURL != "" {
		validatedURL, err := validateBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		baseURL = validatedURL
	}

	return buildOpenAIChatCompletionsURLForPlatform(baseURL, account.Platform), nil
}

func resolveOpenAITargetURLForRequestFormat(account *Account, requestFormat string, validateBaseURL func(string) (string, error)) (string, error) {
	switch NormalizeGatewayOpenAIRequestFormat(requestFormat) {
	case GatewayOpenAIRequestFormatChatCompletions:
		return resolveOpenAIChatCompletionsTargetURL(account, validateBaseURL)
	default:
		return resolveOpenAIResponsesTargetURL(account, validateBaseURL)
	}
}

func resolveOpenAIImagesTargetURL(account *Account, validateBaseURL func(string) (string, error), action string) (string, error) {
	if account == nil {
		return buildOpenAIImagesURLForPlatform("", PlatformOpenAI, action), nil
	}

	baseURL := strings.TrimSpace(account.GetOpenAIBaseURL())
	if baseURL == "" && account.Platform != PlatformCopilot {
		return buildOpenAIImagesURLForPlatform("", account.Platform, action), nil
	}
	if validateBaseURL != nil && baseURL != "" {
		validatedURL, err := validateBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		baseURL = validatedURL
	}
	return buildOpenAIImagesURLForPlatform(baseURL, account.Platform, action), nil
}

func buildOpenAIResponsesURLForPlatform(baseURL string, platform string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		if platform == PlatformCopilot {
			return "https://api.githubcopilot.com/responses"
		}
		return openaiPlatformAPIURL
	}
	if strings.HasSuffix(normalized, "/responses") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/responses") {
		return normalized
	}
	if platform == PlatformCopilot {
		if strings.HasSuffix(normalized, "/v1") {
			return normalized + "/responses"
		}
		return normalized + "/responses"
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/responses"
	}
	return normalized + "/v1/responses"
}

func buildOpenAIChatCompletionsURLForPlatform(baseURL string, platform string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		if platform == PlatformCopilot {
			return "https://api.githubcopilot.com/chat/completions"
		}
		return openaiPlatformChatCompletionsURL
	}
	if strings.HasSuffix(normalized, "/chat/completions") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/chat/completions") {
		return normalized
	}
	if platform == PlatformCopilot {
		if strings.HasSuffix(normalized, "/v1") {
			return normalized + "/chat/completions"
		}
		return normalized + "/chat/completions"
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/chat/completions"
	}
	return normalized + "/v1/chat/completions"
}

func buildOpenAIImagesURLForPlatform(baseURL string, platform string, action string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	action = strings.Trim(strings.TrimSpace(action), "/")
	if action == "" {
		action = "generations"
	}
	if normalized == "" {
		if platform == PlatformCopilot {
			return "https://api.githubcopilot.com/images/" + action
		}
		return openaiPlatformImagesURL + "/" + action
	}
	if strings.HasSuffix(normalized, "/images/"+action) {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/images/"+action) {
		return normalized
	}
	if strings.HasSuffix(normalized, "/images") {
		return normalized + "/" + action
	}
	if strings.HasSuffix(normalized, "/v1/images") {
		return normalized + "/" + action
	}
	if platform == PlatformCopilot {
		if strings.HasSuffix(normalized, "/v1") {
			return normalized + "/images/" + action
		}
		return normalized + "/images/" + action
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/images/" + action
	}
	return normalized + "/v1/images/" + action
}

func buildOpenAIModelsURLForPlatform(baseURL string, platform string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		if platform == PlatformCopilot {
			return "https://api.githubcopilot.com/models"
		}
		return openAIModelsURL
	}
	if strings.HasSuffix(normalized, "/models") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/models") {
		return normalized
	}
	if platform == PlatformCopilot {
		if strings.HasSuffix(normalized, "/v1") {
			return normalized + "/models"
		}
		return normalized + "/models"
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/models"
	}
	return normalized + "/v1/models"
}

func resolveCopilotRequestUserAgent(account *Account) string {
	if account != nil {
		if custom := strings.TrimSpace(account.GetOpenAIUserAgent()); custom != "" {
			return custom
		}
	}
	return copilotDefaultUserAgent
}

func applyCopilotDefaultHeaders(headers http.Header, account *Account) {
	if headers == nil {
		return
	}
	if headers.Get("Accept") == "" {
		headers.Set("Accept", "application/json")
	}
	if headers.Get("User-Agent") == "" {
		headers.Set("User-Agent", resolveCopilotRequestUserAgent(account))
	}
	if headers.Get("Editor-Version") == "" {
		headers.Set("Editor-Version", copilotDefaultEditorVersion)
	}
	if headers.Get("Editor-Plugin-Version") == "" {
		headers.Set("Editor-Plugin-Version", copilotDefaultPluginVersion)
	}
	if headers.Get("Copilot-Integration-Id") == "" {
		headers.Set("Copilot-Integration-Id", copilotDefaultIntegrationID)
	}
	if headers.Get("Openai-Intent") == "" {
		headers.Set("Openai-Intent", copilotDefaultOpenAIIntent)
	}
	if headers.Get("X-GitHub-Api-Version") == "" {
		headers.Set("X-GitHub-Api-Version", copilotGitHubAPIVersion)
	}
}

func applyCopilotDefaultHeadersMap(headers map[string]string, account *Account) {
	if len(headers) == 0 {
		return
	}
	if strings.TrimSpace(headers["Accept"]) == "" {
		headers["Accept"] = "application/json"
	}
	if strings.TrimSpace(headers["User-Agent"]) == "" {
		headers["User-Agent"] = resolveCopilotRequestUserAgent(account)
	}
	if strings.TrimSpace(headers["Editor-Version"]) == "" {
		headers["Editor-Version"] = copilotDefaultEditorVersion
	}
	if strings.TrimSpace(headers["Editor-Plugin-Version"]) == "" {
		headers["Editor-Plugin-Version"] = copilotDefaultPluginVersion
	}
	if strings.TrimSpace(headers["Copilot-Integration-Id"]) == "" {
		headers["Copilot-Integration-Id"] = copilotDefaultIntegrationID
	}
	if strings.TrimSpace(headers["Openai-Intent"]) == "" {
		headers["Openai-Intent"] = copilotDefaultOpenAIIntent
	}
	if strings.TrimSpace(headers["X-GitHub-Api-Version"]) == "" {
		headers["X-GitHub-Api-Version"] = copilotGitHubAPIVersion
	}
}

func trustedCopilotAPIBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(parsed.Host)) {
	case "api.githubcopilot.com", "api.individual.githubcopilot.com", "api.business.githubcopilot.com", "copilot-proxy.githubusercontent.com":
		return strings.TrimRight(raw, "/")
	default:
		return ""
	}
}

func copilotTokenCacheTTL(expiresAt int64) time.Duration {
	if expiresAt <= 0 {
		return 5 * time.Minute
	}
	expireAtTime := time.Unix(expiresAt, 0)
	until := time.Until(expireAtTime)
	switch {
	case until > openAITokenCacheSkew:
		return until - openAITokenCacheSkew
	case until > time.Minute:
		return until
	default:
		return time.Minute
	}
}
