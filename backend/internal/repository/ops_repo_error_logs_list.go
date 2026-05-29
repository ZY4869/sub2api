package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) ListErrorLogs(ctx context.Context, filter *service.OpsErrorLogFilter) (*service.OpsErrorLogList, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if filter == nil {
		filter = &service.OpsErrorLogFilter{}
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 500 {
		pageSize = 500
	}

	where, args := buildOpsErrorLogsWhere(filter)
	countSQL := "SELECT COUNT(*) FROM ops_error_logs e " + where

	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	argsWithLimit := append(args, pageSize, offset)
	selectSQL := `
SELECT
  e.id,
  e.created_at,
  e.error_phase,
  e.error_type,
  COALESCE(e.error_owner, ''),
  COALESCE(e.error_source, ''),
  e.severity,
  COALESCE(e.upstream_status_code, e.status_code, 0),
  COALESCE(e.platform, ''),
  COALESCE(e.model, ''),
  COALESCE(e.is_retryable, false),
  COALESCE(e.retry_count, 0),
  COALESCE(e.resolved, false),
  e.resolved_at,
  e.resolved_by_user_id,
  COALESCE(u2.email, ''),
  e.resolved_retry_id,
  COALESCE(e.client_request_id, ''),
  COALESCE(e.request_id, ''),
  COALESCE(e.error_message, ''),
  e.user_id,
  COALESCE(u.email, ''),
  e.api_key_id,
  e.account_id,
  COALESCE(a.name, ''),
  e.group_id,
  COALESCE(g.name, ''),
  CASE WHEN e.client_ip IS NULL THEN NULL ELSE e.client_ip::text END,
  COALESCE(e.request_path, ''),
  e.stream,
  COALESCE(e.inbound_endpoint, ''),
  COALESCE(e.upstream_endpoint, ''),
  COALESCE(e.requested_model, ''),
  COALESCE(e.upstream_model, ''),
  e.request_type,
  COALESCE(e.upstream_url, ''),
  COALESCE(e.gemini_surface, ''),
  COALESCE(e.billing_rule_id, ''),
  COALESCE(e.probe_action, '')
FROM ops_error_logs e
LEFT JOIN accounts a ON e.account_id = a.id
LEFT JOIN groups g ON e.group_id = g.id
LEFT JOIN users u ON e.user_id = u.id
LEFT JOIN users u2 ON e.resolved_by_user_id = u2.id
` + where + `
ORDER BY e.created_at DESC
LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)

	rows, err := r.db.QueryContext(ctx, selectSQL, argsWithLimit...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]*service.OpsErrorLog, 0, pageSize)
	for rows.Next() {
		item, err := scanOpsErrorLogListRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &service.OpsErrorLogList{
		Errors:   out,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
