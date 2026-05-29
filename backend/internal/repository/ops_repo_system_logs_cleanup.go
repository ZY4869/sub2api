package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) DeleteSystemLogs(ctx context.Context, filter *service.OpsSystemLogCleanupFilter) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if filter == nil {
		filter = &service.OpsSystemLogCleanupFilter{}
	}

	where, args, hasConstraint := buildOpsSystemLogsCleanupWhere(filter)
	if !hasConstraint {
		return 0, fmt.Errorf("cleanup requires at least one filter condition")
	}

	query := "DELETE FROM ops_system_logs l " + where
	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *opsRepository) InsertSystemLogCleanupAudit(ctx context.Context, input *service.OpsSystemLogCleanupAudit) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if input == nil {
		return fmt.Errorf("nil input")
	}
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO ops_system_log_cleanup_audits (
  created_at,
  operator_id,
  conditions,
  deleted_rows
) VALUES ($1,$2,$3,$4)
`, createdAt.UTC(), input.OperatorID, input.Conditions, input.DeletedRows)
	return err
}
