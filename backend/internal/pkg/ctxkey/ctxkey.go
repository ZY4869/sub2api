// Package ctxkey defines typed context keys shared across the request path.
package ctxkey

// Key avoids using built-in string keys directly in context values.
type Key string

const (
	ForcePlatform   Key = "ctx_force_platform"
	RequestID       Key = "ctx_request_id"
	ClientRequestID Key = "ctx_client_request_id"
	Model           Key = "ctx_model"
	Platform        Key = "ctx_platform"
	AccountID       Key = "ctx_account_id"
	RetryCount      Key = "ctx_retry_count"

	AccountSwitchCount         Key = "ctx_account_switch_count"
	IsClaudeCodeClient         Key = "ctx_is_claude_code_client"
	ThinkingEnabled            Key = "ctx_thinking_enabled"
	Group                      Key = "ctx_group"
	Groups                     Key = "ctx_groups"
	IsMaxTokensOneHaikuRequest Key = "ctx_is_max_tokens_one_haiku"
	SingleAccountRetry         Key = "ctx_single_account_retry"
	PrefetchedStickyAccountID  Key = "ctx_prefetched_sticky_account_id"
	PrefetchedStickyGroupID    Key = "ctx_prefetched_sticky_group_id"
	ClaudeCodeVersion          Key = "ctx_claude_code_version"
	GeminiPublicProtocol       Key = "ctx_gemini_public_protocol"
	GeminiPublicProtocolStrict Key = "ctx_gemini_public_protocol_strict"
	GeminiMixedProtocolEnabled Key = "ctx_gemini_mixed_protocol_enabled"
)
