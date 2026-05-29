package service

import (
	"context"
	"log/slog"
	"time"
)

type ErrorPolicyResult int

const (
	ErrorPolicyNone            ErrorPolicyResult = iota // 未命中任何策略，继续默认逻辑
	ErrorPolicySkipped                                  // 自定义错误码开启但未命中，跳过处理
	ErrorPolicyMatched                                  // 自定义错误码命中，应停止调度
	ErrorPolicyTempUnscheduled                          // 临时不可调度规则命中
)

// CheckErrorPolicy 检查自定义错误码和临时不可调度规则。
// 自定义错误码开启时覆盖后续所有逻辑（包括临时不可调度）。
func (s *RateLimitService) CheckErrorPolicy(ctx context.Context, account *Account, statusCode int, responseBody []byte) ErrorPolicyResult {
	if account.IsCustomErrorCodesEnabled() {
		if account.ShouldHandleErrorCode(statusCode) {
			return ErrorPolicyMatched
		}
		slog.Info("account_error_code_skipped", "account_id", account.ID, "status_code", statusCode)
		return ErrorPolicySkipped
	}
	if account.IsPoolMode() {
		return ErrorPolicySkipped
	}
	if s.tryTempUnschedulable(ctx, account, statusCode, responseBody) {
		return ErrorPolicyTempUnscheduled
	}
	return ErrorPolicyNone
}

// HandleUpstreamError 处理上游错误响应，标记账号状态
// 返回是否应该停止该账号的调度
func (s *RateLimitService) markAccountBlacklisted(ctx context.Context, account *Account, match *HardBanMatch) bool {
	if s == nil || s.accountRepo == nil || account == nil || match == nil {
		return false
	}
	now := time.Now()
	purgeAt := now.Add(AccountBlacklistRetention)
	if err := s.accountRepo.MarkBlacklisted(ctx, account.ID, match.ReasonCode, match.ReasonMessage, now, purgeAt); err != nil {
		slog.Warn("account_mark_blacklisted_failed", "account_id", account.ID, "reason_code", match.ReasonCode, "error", err)
		return false
	}
	if s.tempUnschedCache != nil {
		if err := s.tempUnschedCache.DeleteTempUnsched(ctx, account.ID); err != nil {
			slog.Warn("account_mark_blacklisted_clear_temp_unsched_cache_failed", "account_id", account.ID, "error", err)
		}
	}
	slog.Warn("account_marked_blacklisted", "account_id", account.ID, "reason_code", match.ReasonCode, "purge_at", purgeAt)
	return true
}
