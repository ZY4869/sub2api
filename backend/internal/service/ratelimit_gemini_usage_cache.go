package service

import (
	"context"
	"time"
)

type geminiUsageCacheEntry struct {
	windowStart time.Time
	cachedAt    time.Time
	totals      GeminiUsageTotals
}

type geminiUsageTotalsBatchProvider interface {
	GetGeminiUsageTotalsBatch(ctx context.Context, accountIDs []int64, startTime, endTime time.Time) (map[int64]GeminiUsageTotals, error)
}

const geminiPrecheckCacheTTL = time.Minute

func (s *RateLimitService) getGeminiUsageTotalsBatch(ctx context.Context, accountIDs []int64, start, end time.Time) (map[int64]GeminiUsageTotals, error) {
	result := make(map[int64]GeminiUsageTotals, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}

	ids := make([]int64, 0, len(accountIDs))
	seen := make(map[int64]struct{}, len(accountIDs))
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			continue
		}
		if _, ok := seen[accountID]; ok {
			continue
		}
		seen[accountID] = struct{}{}
		ids = append(ids, accountID)
	}
	if len(ids) == 0 {
		return result, nil
	}

	if batchReader, ok := s.usageRepo.(geminiUsageTotalsBatchProvider); ok {
		stats, err := batchReader.GetGeminiUsageTotalsBatch(ctx, ids, start, end)
		if err != nil {
			return nil, err
		}
		for _, accountID := range ids {
			result[accountID] = stats[accountID]
		}
		return result, nil
	}

	for _, accountID := range ids {
		stats, err := s.usageRepo.GetModelStatsWithFilters(ctx, start, end, 0, 0, accountID, 0, 0, nil, nil, nil)
		if err != nil {
			return nil, err
		}
		result[accountID] = geminiAggregateUsage(stats)
	}
	return result, nil
}

func geminiDailyLimit(quota GeminiQuota, modelClass geminiModelClass) int64 {
	if quota.SharedRPD > 0 {
		return quota.SharedRPD
	}
	if modelClass == geminiModelFlash {
		return quota.FlashRPD
	}
	return quota.ProRPD
}

func geminiMinuteLimit(quota GeminiQuota, modelClass geminiModelClass) int64 {
	if quota.SharedRPM > 0 {
		return quota.SharedRPM
	}
	if modelClass == geminiModelFlash {
		return quota.FlashRPM
	}
	return quota.ProRPM
}

func geminiUsedRequests(quota GeminiQuota, modelClass geminiModelClass, totals GeminiUsageTotals, daily bool) int64 {
	if daily && quota.SharedRPD > 0 {
		return totals.ProRequests + totals.FlashRequests
	}
	if !daily && quota.SharedRPM > 0 {
		return totals.ProRequests + totals.FlashRequests
	}
	if modelClass == geminiModelFlash {
		return totals.FlashRequests
	}
	return totals.ProRequests
}

func (s *RateLimitService) getGeminiUsageTotals(accountID int64, windowStart, now time.Time) (GeminiUsageTotals, bool) {
	s.usageCacheMu.RLock()
	defer s.usageCacheMu.RUnlock()

	entry, ok := s.usageCache[accountID]
	if s.usageCache == nil || !ok || entry == nil {
		return GeminiUsageTotals{}, false
	}
	if !entry.windowStart.Equal(windowStart) || now.Sub(entry.cachedAt) >= geminiPrecheckCacheTTL {
		return GeminiUsageTotals{}, false
	}
	return entry.totals, true
}

func (s *RateLimitService) setGeminiUsageTotals(accountID int64, windowStart, now time.Time, totals GeminiUsageTotals) {
	s.usageCacheMu.Lock()
	defer s.usageCacheMu.Unlock()
	if s.usageCache == nil {
		s.usageCache = make(map[int64]*geminiUsageCacheEntry)
	}
	s.usageCache[accountID] = &geminiUsageCacheEntry{
		windowStart: windowStart,
		cachedAt:    now,
		totals:      totals,
	}
}

// GeminiCooldown returns the fallback cooldown duration for Gemini 429s based on tier.
func (s *RateLimitService) GeminiCooldown(ctx context.Context, account *Account) time.Duration {
	if account == nil || s.geminiQuotaService == nil {
		return 5 * time.Minute
	}
	return s.geminiQuotaService.CooldownForAccount(ctx, account)
}
