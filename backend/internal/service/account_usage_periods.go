package service

import (
	"context"
	"log/slog"
	"time"
)

const (
	AccountUsagePeriodWindowWeekly  = "weekly"
	AccountUsagePeriodWindowMonthly = "monthly"

	AccountUsagePeriodSourceUpstreamReset = "upstream_reset"
	AccountUsagePeriodSourceExpiry        = "expiry"
	AccountUsagePeriodSourceDerived       = "derived"
	AccountUsagePeriodSourceFallback30D   = "fallback_30d"
)

type AccountUsagePeriod struct {
	AccountID  int64
	WindowType string
	StartAt    time.Time
	EndAt      *time.Time
	ResetAt    *time.Time
	Source     string
}

type AccountUsagePeriodSyncResult struct {
	AccountID  int64
	WindowType string
	Source     string
	Inserted   bool
	Updated    bool
	Closed     bool
	OldStartAt *time.Time
	OldEndAt   *time.Time
	OldResetAt *time.Time
	NewStartAt *time.Time
	NewEndAt   *time.Time
	NewResetAt *time.Time
}

type AccountUsagePeriodRepository interface {
	GetActiveUsagePeriods(ctx context.Context, accountIDs []int64, windowType string, at time.Time) (map[int64]*AccountUsagePeriod, error)
	SyncMonthlyUsagePeriod(ctx context.Context, account *Account, oldExpiresAt *time.Time, source string) (*AccountUsagePeriodSyncResult, error)
	SyncWeeklyUsagePeriod(ctx context.Context, account *Account, resetAt time.Time, source string) (*AccountUsagePeriodSyncResult, error)
}

func syncAccountMonthlyUsagePeriod(ctx context.Context, repo AccountRepository, account *Account, oldExpiresAt *time.Time, source string) {
	if account == nil || repo == nil {
		return
	}
	periodRepo, ok := repo.(AccountUsagePeriodRepository)
	if !ok || periodRepo == nil {
		return
	}
	if source == "" {
		source = AccountUsagePeriodSourceDerived
	}
	result, err := periodRepo.SyncMonthlyUsagePeriod(ctx, account, oldExpiresAt, source)
	requestID := usagePeriodRequestID(ctx)
	if err != nil {
		slog.Warn(
			"account_usage_period_monthly_sync_failed",
			"request_id", requestID,
			"account_id", account.ID,
			"window_type", AccountUsagePeriodWindowMonthly,
			"old_expires_at", oldExpiresAt,
			"new_expires_at", account.ExpiresAt,
			"source", source,
			"error", err,
		)
		return
	}
	logAccountUsagePeriodSyncSuccess(requestID, result)
}

func logAccountExpiresAtChanged(ctx context.Context, accountID int64, oldExpiresAt *time.Time, newExpiresAt *time.Time, source string) {
	if accountID <= 0 || sameUsagePeriodTimePtr(oldExpiresAt, newExpiresAt) {
		return
	}
	slog.Info(
		"account_expires_at_changed",
		"request_id", usagePeriodRequestID(ctx),
		"account_id", accountID,
		"window_type", AccountUsagePeriodWindowMonthly,
		"old_expires_at", oldExpiresAt,
		"new_expires_at", newExpiresAt,
		"source", source,
	)
}

func syncAccountWeeklyUsagePeriod(ctx context.Context, repo AccountRepository, account *Account, resetAt time.Time, source string) {
	if account == nil || repo == nil || resetAt.IsZero() {
		return
	}
	periodRepo, ok := repo.(AccountUsagePeriodRepository)
	if !ok || periodRepo == nil {
		return
	}
	if source == "" {
		source = AccountUsagePeriodSourceUpstreamReset
	}
	result, err := periodRepo.SyncWeeklyUsagePeriod(ctx, account, resetAt, source)
	requestID := usagePeriodRequestID(ctx)
	if err != nil {
		slog.Warn(
			"account_usage_period_weekly_sync_failed",
			"request_id", requestID,
			"account_id", account.ID,
			"window_type", AccountUsagePeriodWindowWeekly,
			"reset_at", resetAt,
			"source", source,
			"error", err,
		)
		return
	}
	logAccountUsagePeriodSyncSuccess(requestID, result)
}

func logAccountUsagePeriodSyncSuccess(requestID string, result *AccountUsagePeriodSyncResult) {
	if result == nil || result.AccountID <= 0 || (!result.Inserted && !result.Updated && !result.Closed) {
		return
	}
	slog.Info(
		"account_usage_period_synced",
		"request_id", requestID,
		"account_id", result.AccountID,
		"window_type", result.WindowType,
		"source", result.Source,
		"inserted", result.Inserted,
		"updated", result.Updated,
		"closed", result.Closed,
		"old_start_at", result.OldStartAt,
		"old_end_at", result.OldEndAt,
		"old_reset_at", result.OldResetAt,
		"new_start_at", result.NewStartAt,
		"new_end_at", result.NewEndAt,
		"new_reset_at", result.NewResetAt,
	)
}

func usagePeriodRequestID(ctx context.Context) string {
	return firstNonEmptyString(requestIDFromContext(ctx), "generated:"+generateRequestID())
}

func sameUsagePeriodTimePtr(left *time.Time, right *time.Time) bool {
	if left == nil || left.IsZero() {
		return right == nil || right.IsZero()
	}
	if right == nil || right.IsZero() {
		return false
	}
	return left.UTC().Equal(right.UTC())
}
