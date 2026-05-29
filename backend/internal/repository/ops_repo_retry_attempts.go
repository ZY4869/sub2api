package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) InsertRetryAttempt(ctx context.Context, input *service.OpsInsertRetryAttemptInput) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if input == nil {
		return 0, fmt.Errorf("nil input")
	}
	if input.SourceErrorID <= 0 {
		return 0, fmt.Errorf("invalid source_error_id")
	}
	if strings.TrimSpace(input.Mode) == "" {
		return 0, fmt.Errorf("invalid mode")
	}

	q := `
INSERT INTO ops_retry_attempts (
  requested_by_user_id,
  source_error_id,
  mode,
  pinned_account_id,
  status,
  started_at
) VALUES (
  $1,$2,$3,$4,$5,$6
) RETURNING id`

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		q,
		opsNullInt64(&input.RequestedByUserID),
		input.SourceErrorID,
		strings.TrimSpace(input.Mode),
		opsNullInt64(input.PinnedAccountID),
		strings.TrimSpace(input.Status),
		input.StartedAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *opsRepository) UpdateRetryAttempt(ctx context.Context, input *service.OpsUpdateRetryAttemptInput) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if input == nil {
		return fmt.Errorf("nil input")
	}
	if input.ID <= 0 {
		return fmt.Errorf("invalid id")
	}

	q := `
UPDATE ops_retry_attempts
SET
  status = $2,
  finished_at = $3,
  duration_ms = $4,
  success = $5,
  http_status_code = $6,
  upstream_request_id = $7,
  used_account_id = $8,
  response_preview = $9,
  response_truncated = $10,
  result_request_id = $11,
  result_error_id = $12,
  error_message = $13
WHERE id = $1`

	_, err := r.db.ExecContext(
		ctx,
		q,
		input.ID,
		strings.TrimSpace(input.Status),
		nullTime(input.FinishedAt),
		input.DurationMs,
		nullBool(input.Success),
		nullInt(input.HTTPStatusCode),
		opsNullString(input.UpstreamRequestID),
		nullInt64(input.UsedAccountID),
		opsNullString(input.ResponsePreview),
		nullBool(input.ResponseTruncated),
		opsNullString(input.ResultRequestID),
		nullInt64(input.ResultErrorID),
		opsNullString(input.ErrorMessage),
	)
	return err
}
