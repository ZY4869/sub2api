package service

import "strings"

func (a *Account) IsBedrock() bool {
	return a.Platform == PlatformAnthropic && a.Type == AccountTypeBedrock
}

func (a *Account) IsBedrockAPIKey() bool {
	return a.IsBedrock() && a.GetCredential("auth_mode") == "apikey"
}

// IsAPIKeyOrBedrock 返回账号类型是否支持配额和池模式等特性
func (a *Account) IsAPIKeyOrBedrock() bool {
	return a.Type == AccountTypeAPIKey || a.Type == AccountTypeBedrock
}

func (a *Account) IsOpenAI() bool {
	return IsOpenAIFamily(EffectiveProtocol(a))
}

func (a *Account) IsGrok() bool {
	return EffectiveProtocol(a) == PlatformGrok
}

func (a *Account) IsGrokSSO() bool {
	return a.IsGrok() && a.Type == AccountTypeSSO
}

func (a *Account) IsGrokAPIKey() bool {
	return a.IsGrok() && a.Type == AccountTypeAPIKey
}

func (a *Account) IsGrokOAuth() bool {
	return a.IsGrok() && a.Type == AccountTypeOAuth
}

func (a *Account) IsDeepSeek() bool {
	return EffectiveProtocol(a) == PlatformDeepSeek
}

func (a *Account) IsOpenRouter() bool {
	return EffectiveProtocol(a) == PlatformOpenRouter
}

func (a *Account) IsOpenAITextCompatible() bool {
	return a.IsOpenAI() || a.IsDeepSeek() || a.IsOpenRouter()
}

func (a *Account) IsAnthropic() bool {
	return IsAnthropicFamily(EffectiveProtocol(a))
}

func (a *Account) IsOpenAIOAuth() bool {
	return a.IsOpenAI() && a.Type == AccountTypeOAuth
}

func (a *Account) IsOpenAIApiKey() bool {
	return a.IsOpenAI() && a.Type == AccountTypeAPIKey
}

func (a *Account) GetOpenAIBaseURL() string {
	if !a.IsOpenAI() {
		return ""
	}
	baseURL := strings.TrimSpace(a.GetCredential("base_url"))
	if baseURL != "" {
		return baseURL
	}
	return "https://api.openai.com"
}

func (a *Account) GetOpenRouterBaseURL() string {
	if a == nil || a.Type != AccountTypeAPIKey || a.Platform != PlatformOpenRouter {
		return ""
	}
	baseURL := strings.TrimSpace(a.GetCredential("base_url"))
	if baseURL != "" {
		return baseURL
	}
	return openRouterDefaultAPIBaseURL
}

func (a *Account) GetOpenAIAccessToken() string {
	if !a.IsOpenAI() {
		return ""
	}
	return a.GetCredential("access_token")
}

func (a *Account) GetOpenAIRefreshToken() string {
	if !a.IsOpenAIOAuth() {
		return ""
	}
	return a.GetCredential("refresh_token")
}

func (a *Account) GetOpenAIIDToken() string {
	if !a.IsOpenAIOAuth() {
		return ""
	}
	return a.GetCredential("id_token")
}

func (a *Account) GetOpenAIApiKey() string {
	if !a.IsOpenAIApiKey() {
		return ""
	}
	return a.GetCredential("api_key")
}

func (a *Account) GetOpenRouterAPIKey() string {
	if a == nil || a.Platform != PlatformOpenRouter || a.Type != AccountTypeAPIKey {
		return ""
	}
	return a.GetCredential("api_key")
}

func (a *Account) GetGrokAPIKey() string {
	if !a.IsGrokAPIKey() {
		return ""
	}
	return a.GetCredential("api_key")
}

func (a *Account) GetGrokOAuthAccessToken() string {
	if !a.IsGrokOAuth() {
		return ""
	}
	return a.GetCredential("access_token")
}

func (a *Account) GetGrokSSOToken() string {
	if !a.IsGrokSSO() {
		return ""
	}
	return a.GetCredential("sso_token")
}

func (a *Account) GetOpenAIUserAgent() string {
	if !a.IsOpenAI() {
		return ""
	}
	return a.GetCredential("user_agent")
}

func (a *Account) GetChatGPTAccountID() string {
	if !a.IsOpenAIOAuth() {
		return ""
	}
	return a.GetCredential("chatgpt_account_id")
}

func (a *Account) GetChatGPTUserID() string {
	if !a.IsOpenAIOAuth() {
		return ""
	}
	return a.GetCredential("chatgpt_user_id")
}

func (a *Account) GetOpenAIOrganizationID() string {
	if !a.IsOpenAIOAuth() {
		return ""
	}
	return a.GetCredential("organization_id")
}
