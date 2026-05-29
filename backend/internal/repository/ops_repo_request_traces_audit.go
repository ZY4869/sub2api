package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) InsertRequestTraceAudit(ctx context.Context, input *service.OpsInsertRequestTraceAuditInput) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if input == nil {
		return fmt.Errorf("nil input")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO ops_request_trace_audits (
  trace_id,
  operator_id,
  action,
  meta,
  created_at
) VALUES ($1,$2,$3,$4,$5)
`, opsNullInt64(input.TraceID), input.OperatorID, string(input.Action), opsNullString(input.MetaJSON), input.CreatedAt)
	return err
}

func (r *opsRepository) ListRequestTraceAudits(ctx context.Context, traceID int64) ([]*service.OpsRequestTraceAuditLog, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, trace_id, operator_id, action, COALESCE(meta::text, ''), created_at
FROM ops_request_trace_audits
WHERE trace_id = $1
ORDER BY created_at DESC, id DESC
`, traceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.OpsRequestTraceAuditLog, 0, 8)
	for rows.Next() {
		item := &service.OpsRequestTraceAuditLog{}
		var traceIDValue sql.NullInt64
		var action string
		if err := rows.Scan(&item.ID, &traceIDValue, &item.OperatorID, &action, &item.MetaJSON, &item.CreatedAt); err != nil {
			return nil, err
		}
		if traceIDValue.Valid {
			v := traceIDValue.Int64
			item.TraceID = &v
		}
		item.Action = service.OpsRequestTraceAuditAction(action)
		items = append(items, item)
	}
	return items, rows.Err()
}
