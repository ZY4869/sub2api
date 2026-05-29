package service

import (
	"context"
	"fmt"
	"time"
)

func (s *AccountUsageService) getGeminiUsage(ctx context.Context, account *Account) (*UsageInfo, error) {
	now := time.Now()
	usage := &UsageInfo{
		UpdatedAt: &now,
	}

	if s.geminiQuotaService == nil || s.usageLogRepo == nil {
		return usage, nil
	}

	quota, ok := s.geminiQuotaService.QuotaForAccount(ctx, account)
	if !ok {
		return usage, nil
	}

	dayStart := geminiDailyWindowStart(now)
	stats, err := s.usageLogRepo.GetModelStatsWithFilters(ctx, dayStart, now, 0, 0, account.ID, 0, 0, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get gemini usage stats failed: %w", err)
	}

	dayTotals := geminiAggregateUsage(stats)
	dailyResetAt := geminiDailyResetTime(now)

	// Daily window (RPD)
	if quota.SharedRPD > 0 {
		totalReq := dayTotals.ProRequests + dayTotals.FlashRequests
		totalTokens := dayTotals.ProTokens + dayTotals.FlashTokens
		totalCost := dayTotals.ProCost + dayTotals.FlashCost
		usage.GeminiSharedDaily = buildGeminiUsageProgress(totalReq, quota.SharedRPD, dailyResetAt, totalTokens, totalCost, now)
	} else {
		usage.GeminiProDaily = buildGeminiUsageProgress(dayTotals.ProRequests, quota.ProRPD, dailyResetAt, dayTotals.ProTokens, dayTotals.ProCost, now)
		usage.GeminiFlashDaily = buildGeminiUsageProgress(dayTotals.FlashRequests, quota.FlashRPD, dailyResetAt, dayTotals.FlashTokens, dayTotals.FlashCost, now)
	}

	// Minute window (RPM) - fixed-window approximation: current minute [truncate(now), truncate(now)+1m)
	minuteStart := now.Truncate(time.Minute)
	minuteResetAt := minuteStart.Add(time.Minute)
	minuteStats, err := s.usageLogRepo.GetModelStatsWithFilters(ctx, minuteStart, now, 0, 0, account.ID, 0, 0, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get gemini minute usage stats failed: %w", err)
	}
	minuteTotals := geminiAggregateUsage(minuteStats)

	if quota.SharedRPM > 0 {
		totalReq := minuteTotals.ProRequests + minuteTotals.FlashRequests
		totalTokens := minuteTotals.ProTokens + minuteTotals.FlashTokens
		totalCost := minuteTotals.ProCost + minuteTotals.FlashCost
		usage.GeminiSharedMinute = buildGeminiUsageProgress(totalReq, quota.SharedRPM, minuteResetAt, totalTokens, totalCost, now)
	} else {
		usage.GeminiProMinute = buildGeminiUsageProgress(minuteTotals.ProRequests, quota.ProRPM, minuteResetAt, minuteTotals.ProTokens, minuteTotals.ProCost, now)
		usage.GeminiFlashMinute = buildGeminiUsageProgress(minuteTotals.FlashRequests, quota.FlashRPM, minuteResetAt, minuteTotals.FlashTokens, minuteTotals.FlashCost, now)
	}

	return usage, nil
}

func buildGeminiUsageProgress(used, limit int64, resetAt time.Time, tokens int64, cost float64, now time.Time) *UsageProgress {
	// limit <= 0 means "no local quota window" (unknown or unlimited).
	if limit <= 0 {
		return nil
	}
	utilization := (float64(used) / float64(limit)) * 100
	remainingSeconds := int(resetAt.Sub(now).Seconds())
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}
	resetCopy := resetAt
	return &UsageProgress{
		Utilization:      utilization,
		ResetsAt:         &resetCopy,
		RemainingSeconds: remainingSeconds,
		UsedRequests:     used,
		LimitRequests:    limit,
		WindowStats: &WindowStats{
			Requests: used,
			Tokens:   tokens,
			Cost:     cost,
		},
	}
}
