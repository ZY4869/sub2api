package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *usageLogRepository) ListUsageFilterAPIKeys(ctx context.Context, userID int64, startTime, endTime *time.Time) ([]service.APIKey, error) {
	if userID <= 0 {
		return []service.APIKey{}, nil
	}

	conditions := []string{"ul.user_id = $1"}
	args := []any{userID}
	if startTime != nil {
		conditions = append(conditions, fmt.Sprintf("ul.created_at >= $%d", len(args)+1))
		args = append(args, *startTime)
	}
	if endTime != nil {
		conditions = append(conditions, fmt.Sprintf("ul.created_at < $%d", len(args)+1))
		args = append(args, *endTime)
	}

	query := fmt.Sprintf(`
		SELECT
			ul.api_key_id,
			COALESCE(MAX(ak.user_id), MAX(ul.user_id)) AS user_id,
			COALESCE(NULLIF(MAX(ak.name), ''), '#' || ul.api_key_id::text) AS name,
			COALESCE(BOOL_OR(ak.deleted_at IS NOT NULL), FALSE) AS deleted,
			MAX(ul.created_at) AS last_used_at
		FROM usage_logs ul
		LEFT JOIN api_keys ak ON ak.id = ul.api_key_id
		WHERE %s
		GROUP BY ul.api_key_id
		ORDER BY last_used_at DESC, ul.api_key_id DESC
	`, strings.Join(conditions, " AND "))

	return r.scanUsageAPIKeys(ctx, query, args)
}

func (r *usageLogRepository) SearchUsageAPIKeys(ctx context.Context, userID int64, keyword string, limit int) ([]service.APIKey, error) {
	conditions := []string{}
	args := []any{}

	if userID > 0 {
		conditions = append(conditions, fmt.Sprintf("ul.user_id = $%d", len(args)+1))
		args = append(args, userID)
	}

	trimmedKeyword := strings.TrimSpace(keyword)
	if trimmedKeyword != "" {
		conditions = append(conditions, fmt.Sprintf("(COALESCE(ak.name, '') ILIKE $%d OR ul.api_key_id::text ILIKE $%d)", len(args)+1, len(args)+1))
		args = append(args, "%"+trimmedKeyword+"%")
	}

	if limit <= 0 {
		limit = 30
	}

	query := `
		SELECT
			ul.api_key_id,
			COALESCE(MAX(ak.user_id), MAX(ul.user_id)) AS user_id,
			COALESCE(NULLIF(MAX(ak.name), ''), '#' || ul.api_key_id::text) AS name,
			COALESCE(BOOL_OR(ak.deleted_at IS NOT NULL), FALSE) AS deleted,
			MAX(ul.created_at) AS last_used_at
		FROM usage_logs ul
		LEFT JOIN api_keys ak ON ak.id = ul.api_key_id
	`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += fmt.Sprintf(`
		GROUP BY ul.api_key_id
		ORDER BY last_used_at DESC, ul.api_key_id DESC
		LIMIT $%d
	`, len(args)+1)
	args = append(args, limit)

	return r.scanUsageAPIKeys(ctx, query, args)
}

func (r *usageLogRepository) scanUsageAPIKeys(ctx context.Context, query string, args []any) (result []service.APIKey, err error) {
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()

	result = make([]service.APIKey, 0)
	for rows.Next() {
		var (
			item       service.APIKey
			lastUsedAt time.Time
		)
		if err := rows.Scan(&item.ID, &item.UserID, &item.Name, &item.Deleted, &lastUsedAt); err != nil {
			return nil, err
		}
		item.LastUsedAt = &lastUsedAt
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
