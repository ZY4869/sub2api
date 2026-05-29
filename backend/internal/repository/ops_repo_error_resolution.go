package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (r *opsRepository) UpdateErrorResolution(ctx context.Context, errorID int64, resolved bool, resolvedByUserID *int64, resolvedRetryID *int64, resolvedAt *time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if errorID <= 0 {
		return fmt.Errorf("invalid error id")
	}

	q := `
UPDATE ops_error_logs
SET
  resolved = $2,
  resolved_at = $3,
  resolved_by_user_id = $4,
  resolved_retry_id = $5
WHERE id = $1`

	at := sql.NullTime{}
	if resolvedAt != nil && !resolvedAt.IsZero() {
		at = sql.NullTime{Time: resolvedAt.UTC(), Valid: true}
	} else if resolved {
		now := time.Now().UTC()
		at = sql.NullTime{Time: now, Valid: true}
	}

	_, err := r.db.ExecContext(
		ctx,
		q,
		errorID,
		resolved,
		at,
		nullInt64(resolvedByUserID),
		nullInt64(resolvedRetryID),
	)
	return err
}
