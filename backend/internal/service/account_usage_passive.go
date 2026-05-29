package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// GetPassiveUsage 从 Account.Extra 中的被动采样数据构建 UsageInfo，不调用外部 API。
// 仅适用于 Anthropic OAuth / SetupToken 账号。
func (s *AccountUsageService) GetPassiveUsage(ctx context.Context, accountID int64) (*UsageInfo, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get account failed: %w", err)
	}

	return s.buildPassiveUsageInfo(ctx, account)
}

func (s *AccountUsageService) buildPassiveUsageInfo(ctx context.Context, account *Account) (*UsageInfo, error) {
	if account == nil {
		return nil, fmt.Errorf("account is nil")
	}

	if !account.IsAnthropicOAuthOrSetupToken() {
		return nil, fmt.Errorf("passive usage only supported for Anthropic OAuth/SetupToken accounts")
	}

	// 复用 estimateSetupTokenUsage 构建 5h 窗口（OAuth 和 SetupToken 逻辑一致）
	info := s.estimateSetupTokenUsage(account)
	info.Source = "passive"

	// 设置采样时间
	if raw, ok := account.Extra["passive_usage_sampled_at"]; ok {
		if str, ok := raw.(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				info.UpdatedAt = &t
			}
		}
	}

	// 构建 7d 窗口（从被动采样数据）
	util7d := parseExtraFloat64(account.Extra["passive_usage_7d_utilization"])
	reset7dRaw := parseExtraFloat64(account.Extra["passive_usage_7d_reset"])
	if util7d > 0 || reset7dRaw > 0 {
		var resetAt *time.Time
		var remaining int
		if reset7dRaw > 0 {
			t := time.Unix(int64(reset7dRaw), 0)
			resetAt = &t
			remaining = int(time.Until(t).Seconds())
			if remaining < 0 {
				remaining = 0
			}
		}
		info.SevenDay = &UsageProgress{
			Utilization:      util7d * 100,
			ResetsAt:         resetAt,
			RemainingSeconds: remaining,
		}
	}

	// 添加窗口统计
	s.addWindowStats(ctx, account, info, false)

	return info, nil
}

func (s *AccountUsageService) passiveUsageFallbackForMissingAccessToken(ctx context.Context, account *Account, cause error) (*UsageInfo, bool) {
	if account == nil || !account.IsAnthropicOAuthOrSetupToken() || !isMissingUsageAccessTokenError(cause) {
		return nil, false
	}

	usage, err := s.buildPassiveUsageInfo(ctx, account)
	if err != nil {
		slog.Warn("account_usage_passive_fallback_failed", "account_id", account.ID, "error", err)
		return nil, false
	}

	slog.Info(
		"account_usage_passive_fallback_applied",
		"account_id", account.ID,
		"platform", account.Platform,
		"type", account.Type,
	)

	return usage, true
}

func isMissingUsageAccessTokenError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no access token available")
}

// syncActiveToPassive 将主动查询的最新数据回写到 Extra 被动缓存，
// 这样下次被动加载时能看到最新值。
func (s *AccountUsageService) syncActiveToPassive(ctx context.Context, accountID int64, usage *UsageInfo) {
	extraUpdates := make(map[string]any, 4)

	if usage.FiveHour != nil {
		extraUpdates["session_window_utilization"] = usage.FiveHour.Utilization / 100
	}
	if usage.SevenDay != nil {
		extraUpdates["passive_usage_7d_utilization"] = usage.SevenDay.Utilization / 100
		if usage.SevenDay.ResetsAt != nil {
			extraUpdates["passive_usage_7d_reset"] = usage.SevenDay.ResetsAt.Unix()
		}
	}

	if len(extraUpdates) > 0 {
		extraUpdates["passive_usage_sampled_at"] = time.Now().UTC().Format(time.RFC3339)
		if err := s.accountRepo.UpdateExtra(ctx, accountID, extraUpdates); err != nil {
			slog.Warn("sync_active_to_passive_failed", "account_id", accountID, "error", err)
		}
	}
}
