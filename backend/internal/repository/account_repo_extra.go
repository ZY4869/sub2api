package repository

import (
	"context"
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"strings"
	"time"
)

func (r *accountRepository) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	payload, err := json.Marshal(updates)
	if err != nil {
		return err
	}
	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx, "UPDATE accounts SET extra = COALESCE(extra, '{}'::jsonb) || $1::jsonb, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL", string(payload), id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAccountNotFound
	}
	if shouldEnqueueSchedulerOutboxForExtraUpdates(updates) {
		if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &id, nil, nil); err != nil {
			logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue extra update failed: account=%d err=%v", id, err)
		}
	} else if shouldSyncSchedulerSnapshotForExtraUpdates(updates) {
		r.syncSchedulerAccountSnapshot(ctx, id)
	}
	return nil
}
func shouldEnqueueSchedulerOutboxForExtraUpdates(updates map[string]any) bool {
	if len(updates) == 0 {
		return false
	}
	for key := range updates {
		if isSchedulerNeutralAccountExtraKey(key) {
			continue
		}
		return true
	}
	return false
}
func shouldSyncSchedulerSnapshotForExtraUpdates(updates map[string]any) bool {
	if len(updates) == 0 {
		return false
	}
	for key := range updates {
		if isSchedulerNeutralAccountExtraKey(key) {
			return true
		}
	}
	return codexExtraIndicatesRateLimit(updates, "7d") || codexExtraIndicatesRateLimit(updates, "5h")
}
func isSchedulerNeutralAccountExtraKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if key == "session_window_utilization" {
		return true
	}
	return strings.HasPrefix(key, "codex_")
}
func codexExtraIndicatesRateLimit(updates map[string]any, window string) bool {
	if len(updates) == 0 {
		return false
	}
	usedValue, ok := updates["codex_"+window+"_used_percent"]
	if !ok || !extraValueIndicatesExhausted(usedValue) {
		return false
	}
	return extraValueHasResetMarker(updates["codex_"+window+"_reset_at"]) || extraValueHasPositiveNumber(updates["codex_"+window+"_reset_after_seconds"])
}
func extraValueIndicatesExhausted(value any) bool {
	number, ok := extraValueToFloat64(value)
	return ok && number >= 100-1e-9
}
func extraValueHasPositiveNumber(value any) bool {
	number, ok := extraValueToFloat64(value)
	return ok && number > 0
}
func extraValueHasResetMarker(value any) bool {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case time.Time:
		return !v.IsZero()
	case *time.Time:
		return v != nil && !v.IsZero()
	default:
		return false
	}
}
