package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) GetLatestRetryAttemptForError(ctx context.Context, sourceErrorID int64) (*service.OpsRetryAttempt, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if sourceErrorID <= 0 {
		return nil, fmt.Errorf("invalid source_error_id")
	}

	q := `
SELECT
  id,
  created_at,
  COALESCE(requested_by_user_id, 0),
  source_error_id,
  COALESCE(mode, ''),
  pinned_account_id,
  COALESCE(status, ''),
  started_at,
  finished_at,
  duration_ms,
  success,
  http_status_code,
  upstream_request_id,
  used_account_id,
  response_preview,
  response_truncated,
  result_request_id,
  result_error_id,
  error_message
FROM ops_retry_attempts
WHERE source_error_id = $1
ORDER BY created_at DESC
LIMIT 1`

	var out service.OpsRetryAttempt
	var pinnedAccountID sql.NullInt64
	var requestedBy sql.NullInt64
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	var durationMs sql.NullInt64
	var success sql.NullBool
	var httpStatusCode sql.NullInt64
	var upstreamRequestID sql.NullString
	var usedAccountID sql.NullInt64
	var responsePreview sql.NullString
	var responseTruncated sql.NullBool
	var resultRequestID sql.NullString
	var resultErrorID sql.NullInt64
	var errorMessage sql.NullString

	err := r.db.QueryRowContext(ctx, q, sourceErrorID).Scan(
		&out.ID,
		&out.CreatedAt,
		&requestedBy,
		&out.SourceErrorID,
		&out.Mode,
		&pinnedAccountID,
		&out.Status,
		&startedAt,
		&finishedAt,
		&durationMs,
		&success,
		&httpStatusCode,
		&upstreamRequestID,
		&usedAccountID,
		&responsePreview,
		&responseTruncated,
		&resultRequestID,
		&resultErrorID,
		&errorMessage,
	)
	if err != nil {
		return nil, err
	}
	fillRetryAttemptFromNulls(&out, retryAttemptNulls{
		requestedBy:       requestedBy,
		pinnedAccountID:   pinnedAccountID,
		startedAt:         startedAt,
		finishedAt:        finishedAt,
		durationMs:        durationMs,
		success:           success,
		httpStatusCode:    httpStatusCode,
		upstreamRequestID: upstreamRequestID,
		usedAccountID:     usedAccountID,
		responsePreview:   responsePreview,
		responseTruncated: responseTruncated,
		resultRequestID:   resultRequestID,
		resultErrorID:     resultErrorID,
		errorMessage:      errorMessage,
	})
	return &out, nil
}

func (r *opsRepository) ListRetryAttemptsByErrorID(ctx context.Context, sourceErrorID int64, limit int) ([]*service.OpsRetryAttempt, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if sourceErrorID <= 0 {
		return nil, fmt.Errorf("invalid source_error_id")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	q := `
SELECT
  r.id,
  r.created_at,
  COALESCE(r.requested_by_user_id, 0),
  r.source_error_id,
  COALESCE(r.mode, ''),
  r.pinned_account_id,
  COALESCE(pa.name, ''),
  COALESCE(r.status, ''),
  r.started_at,
  r.finished_at,
  r.duration_ms,
  r.success,
  r.http_status_code,
  r.upstream_request_id,
  r.used_account_id,
  COALESCE(ua.name, ''),
  r.response_preview,
  r.response_truncated,
  r.result_request_id,
  r.result_error_id,
  r.error_message
FROM ops_retry_attempts r
LEFT JOIN accounts pa ON r.pinned_account_id = pa.id
LEFT JOIN accounts ua ON r.used_account_id = ua.id
WHERE r.source_error_id = $1
ORDER BY r.created_at DESC
LIMIT $2`

	rows, err := r.db.QueryContext(ctx, q, sourceErrorID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]*service.OpsRetryAttempt, 0, 16)
	for rows.Next() {
		item, err := scanRetryAttemptRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
