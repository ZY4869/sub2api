package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

func buildAccountStatsWindowStarts(ctx context.Context, repo AccountRepository, accounts map[int64]*Account, accountIDs []int64, cycleMode string, now time.Time, periods map[int64]*AccountUsagePeriod) []AccountStatsWindowStart {
	todayStart := timezone.StartOfDay(now)
	calendarWeekStart := todayStart.AddDate(0, 0, -6)
	calendarMonthStart := timezone.StartOfMonth(now)

	out := make([]AccountStatsWindowStart, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		window := AccountStatsWindowStart{
			AccountID:    accountID,
			TodayStart:   todayStart,
			WeeklyStart:  calendarWeekStart,
			MonthlyStart: calendarMonthStart,
		}
		if cycleMode == AccountTodayStatsCycleModeFixed {
			if account := accounts[accountID]; account != nil {
				window.WeeklyStart = fixedWeeklyStart(ctx, repo, account, calendarWeekStart, now)
				window.MonthlyStart = fixedMonthlyStart(account, calendarMonthStart, now, periods[accountID])
			}
		}
		out = append(out, window)
	}
	return out
}

func fixedWeeklyStart(ctx context.Context, repo AccountRepository, account *Account, fallback time.Time, now time.Time) time.Time {
	resetAt := latestFutureResetAt(account, now, "codex_7d_reset_at", codexSpark7dResetAtKey)
	if resetAt.IsZero() {
		return fallback
	}
	syncAccountWeeklyUsagePeriod(ctx, repo, account, resetAt, AccountUsagePeriodSourceUpstreamReset)
	start := resetAt.AddDate(0, 0, -7)
	if start.After(now) {
		return fallback
	}
	return start
}

func fixedMonthlyStart(account *Account, fallback time.Time, now time.Time, period *AccountUsagePeriod) time.Time {
	if period != nil && period.Source != AccountUsagePeriodSourceFallback30D && !period.StartAt.IsZero() {
		end := period.EndAt
		if end == nil || now.Before(*end) || now.Equal(*end) {
			return period.StartAt
		}
	}
	if account == nil {
		return fallback
	}
	createdAt := account.CreatedAt
	if createdAt.IsZero() {
		return fallback
	}
	if account.ExpiresAt != nil && account.ExpiresAt.After(createdAt) {
		// For legacy accounts without period history, the current created_at -> expires_at
		// range is the best local monthly boundary we have.
		if now.Before(*account.ExpiresAt) || sameInstantOrAfter(now, createdAt) {
			return createdAt
		}
	}
	return anchoredThirtyDayStart(createdAt, now)
}

func latestFutureResetAt(account *Account, now time.Time, keys ...string) time.Time {
	var latest time.Time
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		resetAt := account.getExtraTime(key)
		if resetAt.IsZero() || !resetAt.After(now) {
			continue
		}
		if latest.IsZero() || resetAt.After(latest) {
			latest = resetAt
		}
	}
	return latest
}

func anchoredThirtyDayStart(anchor time.Time, now time.Time) time.Time {
	if anchor.IsZero() || now.Before(anchor) {
		return timezone.StartOfMonth(now)
	}
	const period = 30 * 24 * time.Hour
	elapsed := now.Sub(anchor)
	periods := int64(elapsed / period)
	return anchor.Add(time.Duration(periods) * period)
}

func sameInstantOrAfter(value, floor time.Time) bool {
	return value.Equal(floor) || value.After(floor)
}
