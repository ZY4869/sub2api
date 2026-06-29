package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
	applyOpenAIResetCreditsFromExtra(usage, account.Extra)

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
	if force || shouldRefreshOpenAIResetCreditsSnapshot(account, now) {
		s.readOpenAIResetCredits(ctx, account, usage)
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

func (s *AccountUsageService) readOpenAIResetCredits(ctx context.Context, account *Account, usage *UsageInfo) {
	if s == nil || s.openAIResetCreditService == nil || account == nil || usage == nil {
		return
	}
	snapshot, err := s.openAIResetCreditService.ReadResetCredits(ctx, account)
	if err != nil {
		slog.Warn("openai_reset_credits_usage_read_failed", "account_id", account.ID, "error", err.Error())
		s.applyOpenAIResetCreditsUnknown(ctx, account, usage, time.Now().UTC())
		return
	}
	if snapshot == nil {
		return
	}
	usage.OpenAIResetCredits = &OpenAIResetCreditsInfo{
		AvailableCount:    snapshot.AvailableCount,
		UpdatedAt:         &snapshot.UpdatedAt,
		Source:            snapshot.Source,
		Status:            snapshot.Status,
		UnsupportedReason: snapshot.UnsupportedReason,
	}
}

func (s *AccountUsageService) applyOpenAIResetCreditsUnknown(ctx context.Context, account *Account, usage *UsageInfo, now time.Time) {
	if account == nil || usage == nil {
		return
	}
	snapshot := &OpenAIResetCreditsSnapshot{
		UpdatedAt: now.UTC(),
		Source:    openAIResetCreditsSourceWham,
		Status:    openAIResetCreditsStatusUnknownOrUnsupported,
	}
	usage.OpenAIResetCredits = &OpenAIResetCreditsInfo{
		UpdatedAt: &snapshot.UpdatedAt,
		Source:    snapshot.Source,
		Status:    snapshot.Status,
	}
	updates := openAIResetCreditsExtraFromSnapshot(snapshot)
	mergeAccountExtra(account, updates)
	if s == nil || s.accountRepo == nil || account.ID <= 0 {
		return
	}
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
		slog.Warn("openai_reset_credits_unknown_persist_failed", "account_id", account.ID, "error", err.Error())
	}
}

func applyOpenAIResetCreditsFromExtra(usage *UsageInfo, extra map[string]any) {
	if usage == nil || len(extra) == 0 {
		return
	}
	info := &OpenAIResetCreditsInfo{
		Source: openAIResetCreditsSourceWham,
	}
	count, ok := parseOpenAIResetCreditsExtraCount(extra[openAIResetCreditsAvailableCountExtraKey])
	if ok {
		info.AvailableCount = &count
	}
	if updatedAt, ok := parseOpenAIResetCreditsExtraUpdatedAt(extra[openAIResetCreditsUpdatedAtExtraKey]); ok {
		info.UpdatedAt = &updatedAt
	}
	if updatedAt, ok := parseOpenAIResetCreditsExtraUpdatedAt(extra[openAIQuotaUsageUpdatedAtExtraKey]); ok && info.UpdatedAt == nil {
		info.UpdatedAt = &updatedAt
	}
	info.Status = parseOpenAIResetCreditsExtraStatus(extra, info.AvailableCount != nil)
	info.UnsupportedReason = parseOpenAIResetCreditsExtraUnsupportedReason(extra[openAIResetCreditsUnsupportedReasonExtraKey])
	usage.OpenAIResetCredits = info
}

func shouldRefreshOpenAIResetCreditsSnapshot(account *Account, now time.Time) bool {
	if account == nil || !account.IsOpenAIOAuth() {
		return false
	}
	if account.Extra == nil {
		return true
	}
	if ts, ok := parseOpenAIResetCreditsExtraUpdatedAt(account.Extra[openAIResetCreditsUpdatedAtExtraKey]); ok {
		return now.Sub(ts) >= openAIProbeCacheTTL
	}
	if ts, ok := parseOpenAIResetCreditsExtraUpdatedAt(account.Extra[openAIQuotaUsageUpdatedAtExtraKey]); ok {
		return now.Sub(ts) >= openAIProbeCacheTTL
	}
	return true
}

func parseOpenAIResetCreditsExtraCount(value any) (int, bool) {
	count, ok := parseOpenAIResetCreditsAvailableCount(value)
	return count, ok
}

func parseOpenAIResetCreditsExtraUpdatedAt(value any) (time.Time, bool) {
	if value == nil {
		return time.Time{}, false
	}
	ts, err := parseTime(fmt.Sprint(value))
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}

func parseOpenAIResetCreditsExtraStatus(extra map[string]any, hasCount bool) string {
	raw := strings.TrimSpace(fmt.Sprint(extra[openAIResetCreditsStatusExtraKey]))
	switch raw {
	case openAIResetCreditsStatusAvailable,
		openAIResetCreditsStatusUnknownOrUnsupported,
		openAIResetCreditsStatusUnsupported:
		return raw
	}
	if hasCount {
		return openAIResetCreditsStatusAvailable
	}
	return openAIResetCreditsStatusUnknownOrUnsupported
}

func parseOpenAIResetCreditsExtraUnsupportedReason(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
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
