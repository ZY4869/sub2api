package service

import "strings"

const (
	OpenAIWSIngressModeOff         = "off"
	OpenAIWSIngressModeShared      = "shared"
	OpenAIWSIngressModeDedicated   = "dedicated"
	OpenAIWSIngressModeCtxPool     = "ctx_pool"
	OpenAIWSIngressModePassthrough = "passthrough"
)

func normalizeOpenAIWSIngressMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case OpenAIWSIngressModeOff:
		return OpenAIWSIngressModeOff
	case OpenAIWSIngressModeCtxPool:
		return OpenAIWSIngressModeCtxPool
	case OpenAIWSIngressModePassthrough:
		return OpenAIWSIngressModePassthrough
	case OpenAIWSIngressModeShared:
		return OpenAIWSIngressModeShared
	case OpenAIWSIngressModeDedicated:
		return OpenAIWSIngressModeDedicated
	default:
		return ""
	}
}

func normalizeOpenAIWSIngressDefaultMode(mode string) string {
	if normalized := normalizeOpenAIWSIngressMode(mode); normalized != "" {
		if normalized == OpenAIWSIngressModeShared || normalized == OpenAIWSIngressModeDedicated {
			return OpenAIWSIngressModeCtxPool
		}
		return normalized
	}
	return OpenAIWSIngressModeCtxPool
}

// ResolveOpenAIResponsesWebSocketV2Mode 返回账号在 WSv2 ingress 下的有效模式（off/ctx_pool/passthrough）。
//
// 优先级：
// 1. 分类型 mode 新字段（string）
// 2. 分类型 enabled 旧字段（bool）
// 3. 兼容 enabled 旧字段（bool）
// 4. defaultMode（非法时回退 ctx_pool）
func (a *Account) ResolveOpenAIResponsesWebSocketV2Mode(defaultMode string) string {
	resolvedDefault := normalizeOpenAIWSIngressDefaultMode(defaultMode)
	if a == nil || !a.IsOpenAI() {
		return OpenAIWSIngressModeOff
	}
	if a.Extra == nil {
		return resolvedDefault
	}

	resolveModeString := func(key string) (string, bool) {
		raw, ok := a.Extra[key]
		if !ok {
			return "", false
		}
		mode, ok := raw.(string)
		if !ok {
			return "", false
		}
		normalized := normalizeOpenAIWSIngressMode(mode)
		if normalized == "" {
			return "", false
		}
		return normalized, true
	}
	resolveBoolMode := func(key string) (string, bool) {
		raw, ok := a.Extra[key]
		if !ok {
			return "", false
		}
		enabled, ok := raw.(bool)
		if !ok {
			return "", false
		}
		if enabled {
			return OpenAIWSIngressModeCtxPool, true
		}
		return OpenAIWSIngressModeOff, true
	}

	if a.IsOpenAIOAuth() {
		if mode, ok := resolveModeString("openai_oauth_responses_websockets_v2_mode"); ok {
			return mode
		}
		if mode, ok := resolveBoolMode("openai_oauth_responses_websockets_v2_enabled"); ok {
			return mode
		}
	}
	if a.IsOpenAIApiKey() {
		if mode, ok := resolveModeString("openai_apikey_responses_websockets_v2_mode"); ok {
			return mode
		}
		if mode, ok := resolveBoolMode("openai_apikey_responses_websockets_v2_enabled"); ok {
			return mode
		}
	}
	if mode, ok := resolveBoolMode("responses_websockets_v2_enabled"); ok {
		return mode
	}
	if mode, ok := resolveBoolMode("openai_ws_enabled"); ok {
		return mode
	}
	// 兼容旧值：shared/dedicated 语义都归并到 ctx_pool。
	if resolvedDefault == OpenAIWSIngressModeShared || resolvedDefault == OpenAIWSIngressModeDedicated {
		return OpenAIWSIngressModeCtxPool
	}
	return resolvedDefault
}
