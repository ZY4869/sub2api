package service

import (
	"context"
	"fmt"
	"time"
)

func (s *AccountUsageService) getOpenAIUsage(ctx context.Context, account *Account, force bool) (*UsageInfo, error) {
	now := time.Now()
	usage := &UsageInfo{UpdatedAt: &now}

	if account == nil {
		return usage, nil
	}
	syncOpenAICodexRateLimitFromExtra(ctx, s.accountRepo, account, now)
	isPro := isOpenAIProPlan(account)
	applyOpenAIUsageProgressFromExtra(usage, account.Extra, now, isPro)

	shouldProbe := force || shouldRefreshOpenAICodexSnapshot(account, usage, now)
	if shouldProbe && (force || s.shouldProbeOpenAICodexSnapshot(account.ID, now)) {
		probe := s.probeOpenAICodexSnapshot
		if s.openAICodexProbe != nil {
			probe = s.openAICodexProbe
		}
		if updates, resetAt, err := probe(ctx, account); err == nil && (len(updates) > 0 || resetAt != nil) {
			mergeAccountExtra(account, updates)
			if resetAt != nil {
				account.RateLimitResetAt = resetAt
			}
			if usage.UpdatedAt == nil {
				usage.UpdatedAt = &now
			}
			applyOpenAIUsageProgressFromExtra(usage, account.Extra, now, isPro)
		}
	}

	if s.usageLogRepo == nil {
		return usage, nil
	}

	if stats, err := s.usageLogRepo.GetAccountWindowStats(ctx, account.ID, now.Add(-5*time.Hour)); err == nil {
		if usage.FiveHour == nil {
			usage.FiveHour = &UsageProgress{Utilization: 0}
		}
		usage.FiveHour.WindowStats = windowStatsFromAccountStats(stats)
	}

	if stats, err := s.usageLogRepo.GetAccountWindowStats(ctx, account.ID, now.Add(-7*24*time.Hour)); err == nil {
		if usage.SevenDay == nil {
			usage.SevenDay = &UsageProgress{Utilization: 0}
		}
		usage.SevenDay.WindowStats = windowStatsFromAccountStats(stats)
	}

	return usage, nil
}

func applyOpenAIUsageProgressFromExtra(usage *UsageInfo, extra map[string]any, now time.Time, includeSpark bool) {
	if usage == nil {
		return
	}
	usage.FiveHour = buildCodexUsageProgressFromExtra(extra, "5h", now)
	usage.SevenDay = buildCodexUsageProgressFromExtra(extra, "7d", now)
	usage.SparkFiveHour = nil
	usage.SparkSevenDay = nil
	if !includeSpark {
		return
	}
	usage.SparkFiveHour = buildScopedCodexUsageProgressFromExtra(extra, openAICodexScopeSpark, "5h", now)
	usage.SparkSevenDay = buildScopedCodexUsageProgressFromExtra(extra, openAICodexScopeSpark, "7d", now)
}

func shouldRefreshOpenAICodexSnapshot(account *Account, usage *UsageInfo, now time.Time) bool {
	if account == nil {
		return false
	}
	if usage == nil {
		return true
	}
	if usage.FiveHour == nil || usage.SevenDay == nil {
		return true
	}
	if isOpenAIProPlan(account) && (usage.SparkFiveHour == nil || usage.SparkSevenDay == nil) {
		return true
	}
	if isOpenAIProPlan(account) && isOpenAICodexSparkSnapshotStale(account, now) {
		return true
	}
	if account.IsRateLimited() {
		return true
	}
	return isOpenAICodexSnapshotStale(account, now)
}

func isOpenAICodexSnapshotStale(account *Account, now time.Time) bool {
	if account == nil || !isChatGPTOpenAIOAuthAccount(account) || !account.IsOpenAIResponsesWebSocketV2Enabled() {
		return false
	}
	if account.Extra == nil {
		return true
	}
	raw, ok := account.Extra["codex_usage_updated_at"]
	if !ok {
		return true
	}
	ts, err := parseTime(fmt.Sprint(raw))
	if err != nil {
		return true
	}
	return now.Sub(ts) >= openAIProbeCacheTTL
}

func isOpenAICodexSparkSnapshotStale(account *Account, now time.Time) bool {
	if account == nil || !isOpenAIProPlan(account) {
		return false
	}
	if account.Extra == nil {
		return true
	}
	raw, ok := account.Extra[codexSparkUsageUpdatedAtKey]
	if !ok {
		return true
	}
	ts, err := parseTime(fmt.Sprint(raw))
	if err != nil {
		return true
	}
	return now.Sub(ts) >= openAIProbeCacheTTL
}

func (s *AccountUsageService) shouldProbeOpenAICodexSnapshot(accountID int64, now time.Time) bool {
	if s == nil || s.cache == nil || accountID <= 0 {
		return true
	}
	if cached, ok := s.cache.openAIProbeCache.Load(accountID); ok {
		if ts, ok := cached.(time.Time); ok && now.Sub(ts) < openAIProbeCacheTTL {
			return false
		}
	}
	s.cache.openAIProbeCache.Store(accountID, now)
	return true
}
