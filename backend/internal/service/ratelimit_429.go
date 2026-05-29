package service

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

func (s *RateLimitService) handle429(ctx context.Context, account *Account, headers http.Header, responseBody []byte) {
	runtimePlatform := EffectiveProtocol(account)
	// 1. OpenAI 平台：优先尝试解析 x-codex-* 响应头（用于 rate_limit_exceeded）
	if runtimePlatform == PlatformOpenAI {
		if state := s.persistOpenAICodexSnapshot(ctx, account, headers); state != nil {
			if state.AccountResetAt != nil {
				slog.Info("openai_account_rate_limited", "account_id", account.ID, "reason", AccountRateLimitReasonUsage7dAll, "reset_at", *state.AccountResetAt)
				return
			}
			if state.ScopeResetAt != nil {
				slog.Info("openai_codex_scope_rate_limited", "account_id", account.ID, "scope", state.Scope, "reason", state.ScopeReason, "reset_at", *state.ScopeResetAt)
				return
			}
		}
		if resetAt := s.calculateOpenAI429ResetTime(headers); resetAt != nil {
			if err := setAccountRateLimited(ctx, s.accountRepo, account.ID, *resetAt, AccountRateLimitReason429); err != nil {
				slog.Warn("rate_limit_set_failed", "account_id", account.ID, "error", err)
				return
			}
			now := time.Now()
			account.RateLimitedAt = &now
			account.RateLimitResetAt = resetAt
			if account.Extra == nil {
				account.Extra = map[string]any{}
			}
			account.Extra["rate_limit_reason"] = AccountRateLimitReason429
			slog.Info("openai_account_rate_limited", "account_id", account.ID, "reset_at", *resetAt)
			return
		}
	}

	// 2. Anthropic 平台：尝试解析 per-window 头（5h / 7d），选择实际触发的窗口
	if result := calculateAnthropic429ResetTime(headers); result != nil {
		if err := setAccountRateLimited(ctx, s.accountRepo, account.ID, result.resetAt, result.reason); err != nil {
			slog.Warn("rate_limit_set_failed", "account_id", account.ID, "error", err)
			return
		}

		// 更新 session window：优先使用 5h-reset 头精确计算，否则从 resetAt 反推
		windowEnd := result.resetAt
		if result.fiveHourReset != nil {
			windowEnd = *result.fiveHourReset
		}
		windowStart := windowEnd.Add(-5 * time.Hour)
		if err := s.accountRepo.UpdateSessionWindow(ctx, account.ID, &windowStart, &windowEnd, "rejected"); err != nil {
			slog.Warn("rate_limit_update_session_window_failed", "account_id", account.ID, "error", err)
		}

		slog.Info("anthropic_account_rate_limited", "account_id", account.ID, "reset_at", result.resetAt, "reset_in", time.Until(result.resetAt).Truncate(time.Second))
		return
	}

	// 3. 尝试从响应头解析重置时间（Anthropic 聚合头，向后兼容）
	resetTimestamp := headers.Get("anthropic-ratelimit-unified-reset")

	// 4. 如果响应头没有，尝试从响应体解析（OpenAI usage_limit_reached, Gemini）
	if resetTimestamp == "" {
		switch runtimePlatform {
		case PlatformOpenAI:
			// 尝试解析 OpenAI 的 usage_limit_reached 错误
			if resetAt := parseOpenAIRateLimitResetTime(responseBody); resetAt != nil {
				resetTime := time.Unix(*resetAt, 0)
				if err := setAccountRateLimited(ctx, s.accountRepo, account.ID, resetTime, AccountRateLimitReason429); err != nil {
					slog.Warn("rate_limit_set_failed", "account_id", account.ID, "error", err)
					return
				}
				slog.Info("account_rate_limited", "account_id", account.ID, "platform", runtimePlatform, "reset_at", resetTime, "reset_in", time.Until(resetTime).Truncate(time.Second))
				return
			}
		case PlatformGemini, PlatformAntigravity:
			// 尝试解析 Gemini 格式（用于其他平台）
			if resetAt := ParseGeminiRateLimitResetTime(responseBody); resetAt != nil {
				resetTime := time.Unix(*resetAt, 0)
				if err := setAccountRateLimited(ctx, s.accountRepo, account.ID, resetTime, AccountRateLimitReason429); err != nil {
					slog.Warn("rate_limit_set_failed", "account_id", account.ID, "error", err)
					return
				}
				slog.Info("account_rate_limited", "account_id", account.ID, "platform", runtimePlatform, "reset_at", resetTime, "reset_in", time.Until(resetTime).Truncate(time.Second))
				return
			}
		}

		// Anthropic 平台：没有限流重置时间的 429 可能是非真实限流（如 Extra usage required），
		// 不标记账号限流状态，直接透传错误给客户端
		if runtimePlatform == PlatformAnthropic {
			slog.Warn("rate_limit_429_no_reset_time_skipped",
				"account_id", account.ID,
				"platform", runtimePlatform,
				"reason", "no rate limit reset time in headers, likely not a real rate limit")
			return
		}

		// 其他平台：没有重置时间，使用默认5分钟
		resetAt := time.Now().Add(5 * time.Minute)
		slog.Warn("rate_limit_no_reset_time", "account_id", account.ID, "platform", runtimePlatform, "using_default", "5m")
		if err := setAccountRateLimited(ctx, s.accountRepo, account.ID, resetAt, AccountRateLimitReason429); err != nil {
			slog.Warn("rate_limit_set_failed", "account_id", account.ID, "error", err)
		}
		return
	}

	// 解析Unix时间戳
	ts, err := strconv.ParseInt(resetTimestamp, 10, 64)
	if err != nil {
		slog.Warn("rate_limit_reset_parse_failed", "reset_timestamp", resetTimestamp, "error", err)
		resetAt := time.Now().Add(5 * time.Minute)
		if err := setAccountRateLimited(ctx, s.accountRepo, account.ID, resetAt, AccountRateLimitReason429); err != nil {
			slog.Warn("rate_limit_set_failed", "account_id", account.ID, "error", err)
		}
		return
	}

	resetAt := time.Unix(ts, 0)

	// 标记限流状态
	if err := setAccountRateLimited(ctx, s.accountRepo, account.ID, resetAt, AccountRateLimitReason429); err != nil {
		slog.Warn("rate_limit_set_failed", "account_id", account.ID, "error", err)
		return
	}

	// 根据重置时间反推5h窗口
	windowEnd := resetAt
	windowStart := resetAt.Add(-5 * time.Hour)
	if err := s.accountRepo.UpdateSessionWindow(ctx, account.ID, &windowStart, &windowEnd, "rejected"); err != nil {
		slog.Warn("rate_limit_update_session_window_failed", "account_id", account.ID, "error", err)
	}

	slog.Info("account_rate_limited", "account_id", account.ID, "reset_at", resetAt)
}
