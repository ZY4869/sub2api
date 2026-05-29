package service

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// IsAnthropicAPIKeyPassthroughEnabled 返回 Anthropic API Key 账号是否启用“自动透传（仅替换认证）”。
// 字段：accounts.extra.anthropic_passthrough。
// 字段缺失或类型不正确时，按 false（关闭）处理。
func (a *Account) IsAnthropicAPIKeyPassthroughEnabled() bool {
	if a == nil || !a.IsAnthropic() || a.Type != AccountTypeAPIKey || a.Extra == nil {
		return false
	}
	enabled, ok := a.Extra["anthropic_passthrough"].(bool)
	return ok && enabled
}

// IsAnthropicOAuthOrSetupToken 判断是否为 Anthropic OAuth 或 SetupToken 类型账号
// 仅这两类账号支持 5h 窗口额度控制和会话数量控制
func (a *Account) IsAnthropicOAuthOrSetupToken() bool {
	return a.IsAnthropic() && (a.Type == AccountTypeOAuth || a.Type == AccountTypeSetupToken)
}

// IsTLSFingerprintEnabled 检查是否启用 TLS 指纹伪装
// 仅适用于 Anthropic OAuth/SetupToken 类型账号
// 启用后将模拟 Claude Code (Node.js) 客户端的 TLS 握手特征
func (a *Account) IsTLSFingerprintEnabled() bool {
	if a == nil || a.Extra == nil {
		return false
	}
	if a.IsAnthropicOAuthOrSetupToken() {
		if v, ok := a.Extra[enableTLSFingerprintKey]; ok {
			if enabled, ok := v.(bool); ok {
				return enabled
			}
		}
		return false
	}
	if !IsClaudeClientMimicEnabled(a, EffectiveProtocol(a)) {
		return false
	}
	if v, ok := a.Extra[enableTLSFingerprintKey]; ok {
		if enabled, ok := v.(bool); ok {
			return enabled
		}
	}
	return false
}

func (a *Account) GetTLSFingerprintProfileID() int64 {
	if a == nil || a.Extra == nil {
		return 0
	}
	value, ok := a.Extra["tls_fingerprint_profile_id"]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return parsed
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0
		}
		if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// GetUserMsgQueueMode 获取用户消息队列模式
// "serialize" = 串行队列, "throttle" = 软性限速, "" = 未设置（使用全局配置）
func (a *Account) GetUserMsgQueueMode() string {
	if a.Extra == nil {
		return ""
	}
	// 优先读取新字段 user_msg_queue_mode（白名单校验，非法值视为未设置）
	if mode, ok := a.Extra["user_msg_queue_mode"].(string); ok && mode != "" {
		if mode == config.UMQModeSerialize || mode == config.UMQModeThrottle {
			return mode
		}
		return "" // 非法值 fallback 到全局配置
	}
	// 向后兼容: user_msg_queue_enabled: true → "serialize"
	if enabled, ok := a.Extra["user_msg_queue_enabled"].(bool); ok && enabled {
		return config.UMQModeSerialize
	}
	return ""
}

// IsSessionIDMaskingEnabled 检查是否启用会话ID伪装
// 仅适用于 Anthropic OAuth/SetupToken 类型账号
// 启用后将在一段时间内（15分钟）固定 metadata.user_id 中的 session ID，
// 使上游认为请求来自同一个会话
func (a *Account) IsSessionIDMaskingEnabled() bool {
	if a == nil || a.Extra == nil {
		return false
	}
	if a.IsAnthropicOAuthOrSetupToken() {
		if v, ok := a.Extra[sessionIDMaskingEnabledKey]; ok {
			if enabled, ok := v.(bool); ok {
				return enabled
			}
		}
		return false
	}
	if !IsClaudeClientMimicEnabled(a, EffectiveProtocol(a)) {
		return false
	}
	if v, ok := a.Extra[sessionIDMaskingEnabledKey]; ok {
		if enabled, ok := v.(bool); ok {
			return enabled
		}
	}
	return false
}

// IsCustomBaseURLEnabled checks whether a custom relay base URL is enabled.
// It only applies to Anthropic OAuth/SetupToken accounts.
func (a *Account) IsCustomBaseURLEnabled() bool {
	if !a.IsAnthropicOAuthOrSetupToken() || a.Extra == nil {
		return false
	}
	if v, ok := a.Extra["custom_base_url_enabled"]; ok {
		if enabled, ok := v.(bool); ok {
			return enabled
		}
	}
	return false
}

// GetCustomBaseURL returns the custom relay base URL for Anthropic OAuth/SetupToken accounts.
func (a *Account) GetCustomBaseURL() string {
	return strings.TrimSpace(a.GetExtraString("custom_base_url"))
}

// IsCacheTTLOverrideEnabled 检查是否启用缓存 TTL 强制替换
// 仅适用于 Anthropic OAuth/SetupToken 类型账号
// 启用后将所有 cache creation tokens 归入指定的 TTL 类型（5m 或 1h）
func (a *Account) IsCacheTTLOverrideEnabled() bool {
	if !a.IsAnthropicOAuthOrSetupToken() {
		return false
	}
	if a.Extra == nil {
		return false
	}
	if v, ok := a.Extra["cache_ttl_override_enabled"]; ok {
		if enabled, ok := v.(bool); ok {
			return enabled
		}
	}
	return false
}

// GetCacheTTLOverrideTarget 获取缓存 TTL 强制替换的目标类型
// 返回 "5m" 或 "1h"，默认 "5m"
func (a *Account) GetCacheTTLOverrideTarget() string {
	if a.Extra == nil {
		return "5m"
	}
	if v, ok := a.Extra["cache_ttl_override_target"]; ok {
		if target, ok := v.(string); ok && (target == "5m" || target == "1h") {
			return target
		}
	}
	return "5m"
}
