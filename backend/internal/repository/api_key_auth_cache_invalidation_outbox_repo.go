package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type apiKeyAuthCacheInvalidationOutboxRepository struct {
	db *sql.DB
}

func NewAPIKeyAuthCacheInvalidationOutboxRepository(db *sql.DB) service.APIKeyAuthCacheInvalidationOutboxRepository {
	return &apiKeyAuthCacheInvalidationOutboxRepository{db: db}
}

func (r *apiKeyAuthCacheInvalidationOutboxRepository) EnqueueInvalidateByKey(ctx context.Context, cacheKey string) error {
	cacheKey = strings.TrimSpace(cacheKey)
	if cacheKey == "" {
		return nil
	}
	return r.enqueue(ctx, service.APIKeyAuthCacheInvalidationOutboxEventKey, cacheKey, nil, nil)
}

func (r *apiKeyAuthCacheInvalidationOutboxRepository) EnqueueInvalidateByUserID(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return nil
	}
	return r.enqueue(ctx, service.APIKeyAuthCacheInvalidationOutboxEventUser, "", &userID, nil)
}

func (r *apiKeyAuthCacheInvalidationOutboxRepository) EnqueueInvalidateByGroupID(ctx context.Context, groupID int64) error {
	if groupID <= 0 {
		return nil
	}
	return r.enqueue(ctx, service.APIKeyAuthCacheInvalidationOutboxEventGroup, "", nil, &groupID)
}

func (r *apiKeyAuthCacheInvalidationOutboxRepository) ListAfter(ctx context.Context, afterID int64, limit int) ([]service.APIKeyAuthCacheInvalidationOutboxEvent, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_type, cache_key, user_id, group_id, created_at
		FROM api_key_auth_cache_invalidation_outbox
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

	events := make([]service.APIKeyAuthCacheInvalidationOutboxEvent, 0, limit)
	for rows.Next() {
		var (
			cacheKey sql.NullString
			userID   sql.NullInt64
			groupID  sql.NullInt64
			event    service.APIKeyAuthCacheInvalidationOutboxEvent
		)
		if err := rows.Scan(&event.ID, &event.EventType, &cacheKey, &userID, &groupID, &event.CreatedAt); err != nil {
			return nil, err
		}
		if cacheKey.Valid {
			event.CacheKey = cacheKey.String
		}
		if userID.Valid {
			v := userID.Int64
			event.UserID = &v
		}
		if groupID.Valid {
			v := groupID.Int64
			event.GroupID = &v
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *apiKeyAuthCacheInvalidationOutboxRepository) DeleteThrough(ctx context.Context, maxID int64) error {
	if r == nil || r.db == nil {
		return nil
	}
	if maxID <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, "DELETE FROM api_key_auth_cache_invalidation_outbox WHERE id <= $1", maxID)
	return err
}

func (r *apiKeyAuthCacheInvalidationOutboxRepository) enqueue(ctx context.Context, eventType, cacheKey string, userID, groupID *int64) error {
	if r == nil || r.db == nil {
		return nil
	}
	dedupKey := authCacheInvalidationOutboxDedupKey(eventType, cacheKey, userID, groupID)
	if dedupKey == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO api_key_auth_cache_invalidation_outbox (event_type, cache_key, user_id, group_id, dedup_key)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (dedup_key) DO NOTHING
	`, eventType, nullableString(cacheKey), userID, groupID, dedupKey)
	return err
}

func authCacheInvalidationOutboxDedupKey(eventType, cacheKey string, userID, groupID *int64) string {
	switch eventType {
	case service.APIKeyAuthCacheInvalidationOutboxEventKey:
		if cacheKey == "" {
			return ""
		}
		return fmt.Sprintf("%s:%s", eventType, cacheKey)
	case service.APIKeyAuthCacheInvalidationOutboxEventUser:
		if userID == nil || *userID <= 0 {
			return ""
		}
		return fmt.Sprintf("%s:%d", eventType, *userID)
	case service.APIKeyAuthCacheInvalidationOutboxEventGroup:
		if groupID == nil || *groupID <= 0 {
			return ""
		}
		return fmt.Sprintf("%s:%d", eventType, *groupID)
	default:
		return ""
	}
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}
