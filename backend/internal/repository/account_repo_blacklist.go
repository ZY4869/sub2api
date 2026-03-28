package repository

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *accountRepository) ListBlacklistedIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id
		FROM accounts
		WHERE deleted_at IS NULL
			AND lifecycle_state = $1
		ORDER BY id ASC
	`, service.AccountLifecycleBlacklisted)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
