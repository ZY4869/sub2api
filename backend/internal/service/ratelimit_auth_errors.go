package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
)

func isOpenAIPermanentUnauthorizedDetail(body []byte) bool {
	if len(body) == 0 {
		return false
	}

	var payload struct {
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(payload.Detail), "Unauthorized")
}

// PreCheckUsage proactively checks local quota before dispatching a request.
// Returns false when the account should be skipped.

func (s *RateLimitService) handleAuthError(ctx context.Context, account *Account, errorMsg string) {
	if err := s.accountRepo.SetError(ctx, account.ID, errorMsg); err != nil {
		slog.Warn("account_set_error_failed", "account_id", account.ID, "error", err)
		return
	}
	slog.Warn("account_disabled_auth_error", "account_id", account.ID, "error", errorMsg)
}

// handle403 处理 403 Forbidden 错误
// Antigravity 平台区分 validation/violation/generic 三种类型，均 SetError 永久禁用；
// 其他平台保持原有 SetError 行为。
func (s *RateLimitService) handle403(ctx context.Context, account *Account, upstreamMsg string, responseBody []byte) (shouldDisable bool) {
	if EffectiveProtocol(account) == PlatformAntigravity {
		return s.handleAntigravity403(ctx, account, upstreamMsg, responseBody)
	}
	// 非 Antigravity 平台：保持原有行为
	msg := "Access forbidden (403): account may be suspended or lack permissions"
	if upstreamMsg != "" {
		msg = "Access forbidden (403): " + upstreamMsg
	}
	s.handleAuthError(ctx, account, msg)
	return true
}

// handleAntigravity403 处理 Antigravity 平台的 403 错误
// validation（需要验证）→ 永久 SetError（需人工去 Google 验证后恢复）
// violation（违规封号）→ 永久 SetError（需人工处理）
// generic（通用禁止）→ 永久 SetError
func (s *RateLimitService) handleAntigravity403(ctx context.Context, account *Account, upstreamMsg string, responseBody []byte) (shouldDisable bool) {
	fbType := classifyForbiddenType(string(responseBody))

	switch fbType {
	case forbiddenTypeValidation:
		// VALIDATION_REQUIRED: 永久禁用，需人工去 Google 验证后手动恢复
		msg := "Validation required (403): account needs Google verification"
		if upstreamMsg != "" {
			msg = "Validation required (403): " + upstreamMsg
		}
		if validationURL := extractValidationURL(string(responseBody)); validationURL != "" {
			msg += " | validation_url: " + validationURL
		}
		s.handleAuthError(ctx, account, msg)
		return true

	case forbiddenTypeViolation:
		// 违规封号: 永久禁用，需人工处理
		msg := "Account violation (403): terms of service violation"
		if upstreamMsg != "" {
			msg = "Account violation (403): " + upstreamMsg
		}
		s.handleAuthError(ctx, account, msg)
		return true

	default:
		// 通用 403: 保持原有行为
		msg := "Access forbidden (403): account may be suspended or lack permissions"
		if upstreamMsg != "" {
			msg = "Access forbidden (403): " + upstreamMsg
		}
		s.handleAuthError(ctx, account, msg)
		return true
	}
}

// handleCustomErrorCode 处理自定义错误码，停止账号调度
func (s *RateLimitService) handleCustomErrorCode(ctx context.Context, account *Account, statusCode int, errorMsg string) {
	msg := "Custom error code " + strconv.Itoa(statusCode) + ": " + errorMsg
	if err := s.accountRepo.SetError(ctx, account.ID, msg); err != nil {
		slog.Warn("account_set_error_failed", "account_id", account.ID, "status_code", statusCode, "error", err)
		return
	}
	slog.Warn("account_disabled_custom_error", "account_id", account.ID, "status_code", statusCode, "error", errorMsg)
}

// handle429 处理429限流错误
// 解析响应头获取重置时间，标记账号为限流状态
