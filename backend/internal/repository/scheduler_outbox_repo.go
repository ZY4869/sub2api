package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type schedulerOutboxRepository struct {
	db *sql.DB
}

const schedulerOutboxDedupWindow = time.Second

func NewSchedulerOutboxRepository(db *sql.DB) service.SchedulerOutboxRepository {
	return &schedulerOutboxRepository{db: db}
}

func (r *schedulerOutboxRepository) ListAfter(ctx context.Context, afterID int64, limit int) ([]service.SchedulerOutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_type, account_id, group_id, payload, created_at
		FROM scheduler_outbox
		WHERE id > $1
		ORDER BY id ASC
		LIMIT $2
	`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	events := make([]service.SchedulerOutboxEvent, 0, limit)
	for rows.Next() {
		var (
			payloadRaw []byte
			accountID  sql.NullInt64
			groupID    sql.NullInt64
			event      service.SchedulerOutboxEvent
		)
		if err := rows.Scan(&event.ID, &event.EventType, &accountID, &groupID, &payloadRaw, &event.CreatedAt); err != nil {
			return nil, err
		}
		if accountID.Valid {
			v := accountID.Int64
			event.AccountID = &v
		}
		if groupID.Valid {
			v := groupID.Int64
			event.GroupID = &v
		}
		if len(payloadRaw) > 0 {
			var payload map[string]any
			if err := json.Unmarshal(payloadRaw, &payload); err != nil {
				return nil, err
			}
			event.Payload = payload
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *schedulerOutboxRepository) MaxID(ctx context.Context) (int64, error) {
	var maxID int64
	if err := r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM scheduler_outbox").Scan(&maxID); err != nil {
		return 0, err
	}
	return maxID, nil
}

func (r *schedulerOutboxRepository) DeleteThrough(ctx context.Context, maxID int64) error {
	if maxID <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, "DELETE FROM scheduler_outbox WHERE id <= $1", maxID)
	return err
}

func enqueueSchedulerOutbox(ctx context.Context, exec sqlExecutor, eventType string, accountID *int64, groupID *int64, payload any) error {
	if exec == nil {
		return nil
	}
	var payloadArg any
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		payloadArg = encoded
	}
	query := `
		INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload, dedup_key)
		VALUES ($1, $2, $3, $4, $5)
	`
	dedupKey := schedulerOutboxDedupKey(eventType, accountID, groupID)
	args := []any{eventType, accountID, groupID, payloadArg, dedupKey}
	if schedulerOutboxEventSupportsDedup(eventType) {
		query = `
			INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload, dedup_key)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (dedup_key) WHERE dedup_key IS NOT NULL DO NOTHING
		`
		if dedupKey == nil {
			query = `
			INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload, dedup_key)
			SELECT $1, $2, $3, $4, $5
			WHERE NOT EXISTS (
				SELECT 1
				FROM scheduler_outbox
				WHERE event_type = $1
					AND account_id IS NOT DISTINCT FROM $2
					AND group_id IS NOT DISTINCT FROM $3
					AND created_at >= NOW() - make_interval(secs => $6)
			)
		`
			args = append(args, schedulerOutboxDedupWindow.Seconds())
		}
	}
	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

func schedulerOutboxDedupKey(eventType string, accountID *int64, groupID *int64) any {
	if !schedulerOutboxEventSupportsDedup(eventType) {
		return nil
	}
	switch eventType {
	case service.SchedulerOutboxEventAccountChanged,
		service.SchedulerOutboxEventAccountGroupsChanged:
		if accountID != nil && *accountID > 0 {
			return fmt.Sprintf("%s:account:%d", eventType, *accountID)
		}
	case service.SchedulerOutboxEventGroupChanged:
		if groupID != nil && *groupID > 0 {
			return fmt.Sprintf("%s:group:%d", eventType, *groupID)
		}
	case service.SchedulerOutboxEventFullRebuild:
		return eventType
	}
	return nil
}

func schedulerOutboxEventSupportsDedup(eventType string) bool {
	switch eventType {
	case service.SchedulerOutboxEventAccountChanged,
		service.SchedulerOutboxEventAccountGroupsChanged,
		service.SchedulerOutboxEventGroupChanged,
		service.SchedulerOutboxEventFullRebuild:
		return true
	default:
		return false
	}
}
