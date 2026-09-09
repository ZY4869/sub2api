package service

import (
	"strings"
)

func (a *Account) GetBaseURL() string {
	allowBaseURL := a.Type == AccountTypeAPIKey || (a.Platform == PlatformGrok && a.Type == AccountTypeOAuth)
	if !allowBaseURL {
		return ""
	}
	if a.Platform == PlatformGrok && a.Type == AccountTypeOAuth {
		return defaultGrokCLIBaseURL
	}
	baseURL := a.GetCredential("base_url")
	if baseURL == "" {
		if a.Platform == PlatformGrok {
			return "https://api.x.ai"
		}
		if a.Platform == PlatformDeepSeek {
			return deepSeekAnthropicBaseURL("")
		}
		if a.Platform == PlatformKimi {
			return "https://api.moonshot.cn"
		}
		return "https://api.anthropic.com"
	}
	if a.Platform == PlatformAntigravity {
		return appendPathSuffixOnce(baseURL, "/antigravity")
	}
	if a.Platform == PlatformDeepSeek {
		return deepSeekAnthropicBaseURL(baseURL)
	}
	return baseURL
}

func deepSeekRootBaseURL(baseURL string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		return "https://api.deepseek.com"
	}
	if strings.HasSuffix(strings.ToLower(normalized), "/anthropic") {
		return strings.TrimRight(normalized[:len(normalized)-len("/anthropic")], "/")
	}
	return normalized
}

func deepSeekAnthropicBaseURL(baseURL string) string {
	return deepSeekRootBaseURL(baseURL) + "/anthropic"
}

func (a *Account) GetDeepSeekBaseURL() string {
	if a == nil || a.Type != AccountTypeAPIKey || a.Platform != PlatformDeepSeek {
		return ""
	}
	baseURL := strings.TrimSpace(a.GetCredential("base_url"))
	if baseURL == "" {
		return "https://api.deepseek.com"
	}
	return deepSeekRootBaseURL(baseURL)
}

// GetGeminiBaseURL 返回 Gemini 兼容端点的 base URL。
// Antigravity 平台的 APIKey 账号自动拼接 /antigravity。
func (a *Account) GetGeminiBaseURL(defaultBaseURL string) string {
	baseURL := strings.TrimSpace(a.GetCredential("base_url"))
	if baseURL == "" {
		return defaultBaseURL
	}
	if a.Platform == PlatformAntigravity && a.Type == AccountTypeAPIKey {
		return appendPathSuffixOnce(baseURL, "/antigravity")
	}
	return baseURL
}

func appendPathSuffixOnce(baseURL string, suffix string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	suffix = "/" + strings.Trim(strings.TrimSpace(suffix), "/")
	if normalized == "" || suffix == "/" {
		return normalized
	}
	if strings.HasSuffix(strings.ToLower(normalized), strings.ToLower(suffix)) {
		return normalized
	}
	return normalized + suffix
}

func (a *Account) GetExtraString(key string) string {
	if a.Extra == nil {
		return ""
	}
	if v, ok := a.Extra[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (a *Account) GatewayProtocol() string {
	return GetAccountGatewayProtocol(a)
}

func (a *Account) EffectiveProtocol() string {
	return EffectiveProtocol(a)
}

func (a *Account) GetClaudeUserID() string {
	if v := strings.TrimSpace(a.GetExtraString("claude_user_id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(a.GetExtraString("anthropic_user_id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(a.GetCredential("claude_user_id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(a.GetCredential("anthropic_user_id")); v != "" {
		return v
	}
	return ""
}
