package service

import (
	"context"
	"log/slog"
	"time"
)

// PreCheckUsageBatch performs quota precheck for multiple accounts in one request.
// Returned map value=false means the account should be skipped.
func (s *RateLimitService) PreCheckUsageBatch(ctx context.Context, accounts []*Account, requestedModel string) (map[int64]bool, error) {
	result := make(map[int64]bool, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		result[account.ID] = true
	}

	if len(accounts) == 0 || requestedModel == "" {
		return result, nil
	}
	if s.usageRepo == nil || s.geminiQuotaService == nil {
		return result, nil
	}

	modelClass := geminiModelClassFromName(requestedModel)
	now := time.Now()
	dailyStart := geminiDailyWindowStart(now)
	minuteStart := now.Truncate(time.Minute)

	type quotaAccount struct {
		account *Account
		quota   GeminiQuota
	}
	quotaAccounts := make([]quotaAccount, 0, len(accounts))
	for _, account := range accounts {
		if account == nil || EffectiveProtocol(account) != PlatformGemini {
			continue
		}
		quota, ok := s.geminiQuotaService.QuotaForAccount(ctx, account)
		if !ok {
			continue
		}
		quotaAccounts = append(quotaAccounts, quotaAccount{
			account: account,
			quota:   quota,
		})
	}
	if len(quotaAccounts) == 0 {
		return result, nil
	}

	// Daily precheck uses cached totals with batch DB fallback.
	dailyTotalsByID := make(map[int64]GeminiUsageTotals, len(quotaAccounts))
	dailyMissIDs := make([]int64, 0, len(quotaAccounts))
	for _, item := range quotaAccounts {
		limit := geminiDailyLimit(item.quota, modelClass)
		if limit <= 0 {
			continue
		}
		accountID := item.account.ID
		if totals, ok := s.getGeminiUsageTotals(accountID, dailyStart, now); ok {
			dailyTotalsByID[accountID] = totals
			continue
		}
		dailyMissIDs = append(dailyMissIDs, accountID)
	}
	if len(dailyMissIDs) > 0 {
		totalsBatch, err := s.getGeminiUsageTotalsBatch(ctx, dailyMissIDs, dailyStart, now)
		if err != nil {
			return result, err
		}
		for _, accountID := range dailyMissIDs {
			totals := totalsBatch[accountID]
			dailyTotalsByID[accountID] = totals
			s.setGeminiUsageTotals(accountID, dailyStart, now, totals)
		}
	}
	for _, item := range quotaAccounts {
		limit := geminiDailyLimit(item.quota, modelClass)
		if limit <= 0 {
			continue
		}
		accountID := item.account.ID
		used := geminiUsedRequests(item.quota, modelClass, dailyTotalsByID[accountID], true)
		if used >= limit {
			resetAt := geminiDailyResetTime(now)
			slog.Info("gemini_precheck_daily_quota_reached_batch", "account_id", accountID, "used", used, "limit", limit, "reset_at", resetAt)
			result[accountID] = false
		}
	}

	minuteIDs := make([]int64, 0, len(quotaAccounts))
	for _, item := range quotaAccounts {
		accountID := item.account.ID
		if !result[accountID] || geminiMinuteLimit(item.quota, modelClass) <= 0 {
			continue
		}
		minuteIDs = append(minuteIDs, accountID)
	}
	if len(minuteIDs) == 0 {
		return result, nil
	}

	minuteTotalsByID, err := s.getGeminiUsageTotalsBatch(ctx, minuteIDs, minuteStart, now)
	if err != nil {
		return result, err
	}
	for _, item := range quotaAccounts {
		accountID := item.account.ID
		if !result[accountID] {
			continue
		}
		limit := geminiMinuteLimit(item.quota, modelClass)
		if limit <= 0 {
			continue
		}
		used := geminiUsedRequests(item.quota, modelClass, minuteTotalsByID[accountID], false)
		if used >= limit {
			resetAt := minuteStart.Add(time.Minute)
			slog.Info("gemini_precheck_minute_quota_reached_batch", "account_id", accountID, "used", used, "limit", limit, "reset_at", resetAt)
			result[accountID] = false
		}
	}

	return result, nil
}
