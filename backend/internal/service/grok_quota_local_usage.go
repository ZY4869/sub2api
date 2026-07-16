package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

const grokFreeQuotaWindow = 24 * time.Hour

func grokLocalUsageForQuota(ctx context.Context, repo UsageLogRepository, accountID int64, billing *xai.BillingSummary, now time.Time) (*WindowStats, *WindowStats, *WindowStats) {
	if grokBillingHasAuthoritativeQuota(billing) {
		weekly, monthly := grokLocalUsageForBilling(ctx, repo, accountID, billing, now)
		return nil, weekly, monthly
	}
	return grokLocalUsage24h(ctx, repo, accountID, now), nil, nil
}

func grokLocalUsage24h(ctx context.Context, repo UsageLogRepository, accountID int64, now time.Time) *WindowStats {
	if repo == nil || accountID <= 0 {
		return nil
	}
	start := now.UTC().Add(-grokFreeQuotaWindow)
	stats, err := repo.GetAccountWindowStats(ctx, accountID, start)
	if err != nil {
		slog.Warn("grok_rolling_24h_usage_query_failed", "account_id", accountID, "window_start", start, "error", err)
		return nil
	}
	return windowStatsFromAccountStats(stats)
}

func grokLocalUsageForBilling(ctx context.Context, repo UsageLogRepository, accountID int64, billing *xai.BillingSummary, now time.Time) (*WindowStats, *WindowStats) {
	var weekly *WindowStats
	var monthly *WindowStats
	if repo == nil || accountID <= 0 {
		return weekly, monthly
	}
	if start, ok := currentGrokBillingWindow(billing, true, now); ok {
		if stats, err := repo.GetAccountWindowStats(ctx, accountID, start); err == nil {
			weekly = windowStatsFromAccountStats(stats)
		} else {
			slog.Warn("grok_window_usage_query_failed", "account_id", accountID, "window_start", start, "error", err)
		}
	}
	if start, ok := currentGrokBillingWindow(billing, false, now); ok {
		if stats, err := repo.GetAccountWindowStats(ctx, accountID, start); err == nil {
			monthly = windowStatsFromAccountStats(stats)
		} else {
			slog.Warn("grok_monthly_usage_query_failed", "account_id", accountID, "window_start", start, "error", err)
		}
	}
	return weekly, monthly
}

func currentGrokBillingWindow(billing *xai.BillingSummary, weekly bool, now time.Time) (time.Time, bool) {
	if billing == nil {
		return time.Time{}, false
	}
	startRaw, endRaw := billing.BillingPeriodStart, billing.BillingPeriodEnd
	if weekly {
		if billing.PeriodType != "weekly" {
			return time.Time{}, false
		}
		startRaw, endRaw = billing.PeriodStart, billing.PeriodEnd
	}
	start, startErr := parseTime(strings.TrimSpace(startRaw))
	end, endErr := parseTime(strings.TrimSpace(endRaw))
	if startErr != nil || endErr != nil || now.Before(start) || !now.Before(end) {
		return time.Time{}, false
	}
	return start, true
}
