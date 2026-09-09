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

func (a *Account) IsKimi() bool {
	return a != nil && EffectiveProtocol(a) == PlatformKimi
}

func (a *Account) IsOpenRouter() bool {
	return EffectiveProtocol(a) == PlatformOpenRouter
}

func (a *Account) IsOpenAITextCompatible() bool {
	return a.IsOpenAI() || a.IsDeepSeek() || a.IsKimi() || a.IsOpenRouter()
}

func (a *Account) IsAnthropic() bool {
	return IsAnthropicFamily(EffectiveProtocol(a))
}

func (a *Account) IsOpenAIOAuth() bool {
	return a.IsOpenAI() && a.Type == AccountTypeOAuth
}

// IsOpenAIPersonalAccessToken reports whether an OpenAI OAuth account uses a
// Codex Personal Access Token. PATs are OAuth-shaped credentials but require
// the Responses web_search fallback for alpha/search.
func (a *Account) IsOpenAIPersonalAccessToken() bool {
	if !a.IsOpenAIOAuth() {
		return false
	}
	for _, key := range []string{"auth_mode", "openai_auth_mode"} {
		value := strings.ToLower(strings.TrimSpace(a.GetCredential(key)))
		if isOpenAIPersonalAccessTokenAuthMode(value) {
			return true
		}
	}
	// Imported Codex credentials can predate the auth_mode marker. Codex PATs
	// use the at-* token family, while regular OAuth access tokens are JWTs.
	accessToken := strings.TrimSpace(a.GetCredential("access_token"))
	if len(accessToken) >= 3 && strings.EqualFold(accessToken[:3], "at-") {
		return true
	}
	return false
}

func (a *Account) IsOpenAIApiKey() bool {
	return a.IsOpenAI() && a.Type == AccountTypeAPIKey
}

func (a *Account) GetOpenAIBaseURL() string {
	if a == nil || (!a.IsOpenAI() && !a.IsKimi()) {
		return ""
	}
	baseURL := strings.TrimSpace(a.GetCredential("base_url"))
	if baseURL != "" {
		return baseURL
	}
	if a.IsKimi() {
		return "https://api.moonshot.cn"
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

func (a *Account) IsChatGPTAccountFedRAMP() bool {
	if !a.IsOpenAIOAuth() || a.Credentials == nil {
		return false
	}
	raw, ok := a.Credentials["chatgpt_account_is_fedramp"]
	if !ok || raw == nil {
		return false
	}
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "1", "yes":
			return true
		}
	}
	return false
}

func (a *Account) GetOpenAIOrganizationID() string {
	if !a.IsOpenAIOAuth() {
		return ""
	}
	return a.GetCredential("organization_id")
}
