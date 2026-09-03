package service

import "strings"

const (
	openaiPlatformChatCompletionsURL = "https://api.openai.com/v1/chat/completions"
	openaiPlatformEmbeddingsURL      = "https://api.openai.com/v1/embeddings"
	openaiPlatformAlphaSearchURL     = "https://api.openai.com/v1/alpha/search"
	chatgptCodexAlphaSearchURL       = "https://chatgpt.com/backend-api/codex/alpha/search"
	openaiPlatformImagesURL          = "https://api.openai.com/v1/images"
	deepseekDefaultAPIBaseURL        = "https://api.deepseek.com"
	openRouterDefaultAPIBaseURL      = "https://openrouter.ai/api/v1"
)

func isChatGPTOpenAIOAuthAccount(account *Account) bool {
	return account != nil && account.Type == AccountTypeOAuth && (account.Platform == "" || account.Platform == PlatformOpenAI)
}

func resolveOpenAIResponsesTargetURL(account *Account, validateBaseURL func(string) (string, error)) (string, error) {
	if account == nil {
		return openaiPlatformAPIURL, nil
	}
	if isChatGPTOpenAIOAuthAccount(account) {
		return chatgptCodexURL, nil
	}

	baseURL := strings.TrimSpace(resolveOpenAICompatibleBaseURL(account))
	if baseURL == "" {
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

func resolveOpenAIEmbeddingsTargetURL(account *Account, validateBaseURL func(string) (string, error)) (string, error) {
	if account == nil {
		return openaiPlatformEmbeddingsURL, nil
	}

	baseURL := strings.TrimSpace(resolveOpenAICompatibleBaseURL(account))
	if baseURL == "" {
		return openaiPlatformEmbeddingsURL, nil
	}
	if validateBaseURL != nil && baseURL != "" {
		validatedURL, err := validateBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		baseURL = validatedURL
	}

	return buildOpenAIEmbeddingsURLForPlatform(baseURL), nil
}

func resolveOpenAIAlphaSearchTargetURL(account *Account, validateBaseURL func(string) (string, error)) (string, error) {
	if account == nil {
		return openaiPlatformAlphaSearchURL, nil
	}
	if account.IsOpenAIOAuth() {
		return chatgptCodexAlphaSearchURL, nil
	}

	baseURL := strings.TrimSpace(account.GetOpenAIBaseURL())
	if baseURL == "" {
		return openaiPlatformAlphaSearchURL, nil
	}
	if validateBaseURL != nil {
		validatedURL, err := validateBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		baseURL = validatedURL
	}

	return buildOpenAIAlphaSearchURL(baseURL), nil
}

func resolveOpenAIChatCompletionsTargetURL(account *Account, validateBaseURL func(string) (string, error)) (string, error) {
	if account == nil {
		return openaiPlatformChatCompletionsURL, nil
	}

	baseURL := strings.TrimSpace(resolveOpenAICompatibleBaseURL(account))
	if baseURL == "" {
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

func resolveDeepSeekChatCompletionsTargetURL(account *Account, validateBaseURL func(string) (string, error), beta bool) (string, error) {
	baseURL := strings.TrimSpace(resolveOpenAICompatibleBaseURL(account))
	if validateBaseURL != nil && baseURL != "" {
		validatedURL, err := validateBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		baseURL = validatedURL
	}
	return buildDeepSeekOpenAITextURL(baseURL, "/chat/completions", beta), nil
}

func resolveDeepSeekCompletionsTargetURL(account *Account, validateBaseURL func(string) (string, error)) (string, error) {
	baseURL := strings.TrimSpace(resolveOpenAICompatibleBaseURL(account))
	if validateBaseURL != nil && baseURL != "" {
		validatedURL, err := validateBaseURL(baseURL)
		if err != nil {
			return "", err
		}
		baseURL = validatedURL
	}
	return buildDeepSeekOpenAITextURL(baseURL, "/completions", true), nil
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

	baseURL := strings.TrimSpace(resolveOpenAICompatibleBaseURL(account))
	if baseURL == "" {
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

func buildOpenAIEmbeddingsURLForPlatform(baseURL string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		return openaiPlatformEmbeddingsURL
	}
	if strings.HasSuffix(normalized, "/embeddings") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/embeddings") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/embeddings"
	}
	return normalized + "/v1/embeddings"
}

func buildOpenAIAlphaSearchURL(baseURL string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		return openaiPlatformAlphaSearchURL
	}
	if strings.HasSuffix(normalized, "/alpha/search") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/alpha/search") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/alpha/search"
	}
	return normalized + "/v1/alpha/search"
}

func buildOpenAIResponsesURLForPlatform(baseURL string, platform string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		return openaiPlatformAPIURL
	}
	if strings.HasSuffix(normalized, "/responses") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/responses") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/responses"
	}
	return normalized + "/v1/responses"
}

func buildOpenAIChatCompletionsURLForPlatform(baseURL string, platform string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		if platform == PlatformDeepSeek {
			return buildDeepSeekOpenAITextURL("", "/chat/completions", false)
		}
		if platform == PlatformOpenRouter {
			return openRouterDefaultAPIBaseURL + "/chat/completions"
		}
		return openaiPlatformChatCompletionsURL
	}
	if strings.HasSuffix(normalized, "/chat/completions") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/chat/completions") {
		return normalized
	}
	if platform == PlatformDeepSeek {
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

func buildDeepSeekOpenAITextURL(baseURL string, path string, beta bool) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		normalized = deepseekDefaultAPIBaseURL
	}
	normalized = trimDeepSeekOpenAICompatSuffix(normalized)
	if beta {
		normalized = normalized + "/beta"
	}
	if strings.HasSuffix(normalized, path) {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1"+path) {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + path
	}
	return normalized + path
}

func trimDeepSeekOpenAICompatSuffix(baseURL string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		return deepseekDefaultAPIBaseURL
	}
	for _, suffix := range []string{"/anthropic", "/beta"} {
		lower := strings.ToLower(normalized)
		if strings.HasSuffix(lower, suffix) {
			normalized = strings.TrimRight(normalized[:len(normalized)-len(suffix)], "/")
		}
	}
	if normalized == "" {
		return deepseekDefaultAPIBaseURL
	}
	return normalized
}

func buildOpenAIImagesURLForPlatform(baseURL string, platform string, action string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	action = strings.Trim(strings.TrimSpace(action), "/")
	if action == "" {
		action = "generations"
	}
	if normalized == "" {
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
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/images/" + action
	}
	return normalized + "/v1/images/" + action
}

func buildOpenAIModelsURLForPlatform(baseURL string, platform string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		if platform == PlatformDeepSeek {
			return deepseekDefaultAPIBaseURL + "/models"
		}
		if platform == PlatformOpenRouter {
			return openRouterDefaultAPIBaseURL + "/models"
		}
		return openAIModelsURL
	}
	if strings.HasSuffix(normalized, "/models") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1/models") {
		return normalized
	}
	if platform == PlatformDeepSeek {
		if strings.HasSuffix(normalized, "/v1") {
			return normalized + "/models"
		}
		return normalized + "/models"
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/models"
	}
	if openAIBaseURLHasVersionSuffix(normalized) {
		return normalized + "/models"
	}
	return normalized + "/v1/models"
}

func openAIBaseURLHasVersionSuffix(baseURL string) bool {
	lower := strings.ToLower(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if lower == "" {
		return false
	}
	lastSlash := strings.LastIndex(lower, "/")
	if lastSlash >= 0 {
		lower = lower[lastSlash+1:]
	}
	if len(lower) < 2 || lower[0] != 'v' {
		return false
	}
	for _, ch := range lower[1:] {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func resolveOpenAICompatibleBaseURL(account *Account) string {
	if account == nil {
		return ""
	}
	if account.Platform == PlatformDeepSeek {
		return account.GetDeepSeekBaseURL()
	}
	if account.Platform == PlatformOpenRouter {
		return account.GetOpenRouterBaseURL()
	}
	return account.GetOpenAIBaseURL()
}
