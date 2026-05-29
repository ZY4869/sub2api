package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const insertOpsErrorLogSQL = `
INSERT INTO ops_error_logs (
  request_id,
  client_request_id,
  user_id,
  api_key_id,
  account_id,
  group_id,
  channel_id,
  client_ip,
  platform,
  model,
  request_path,
  stream,
  inbound_endpoint,
  upstream_endpoint,
  requested_model,
  upstream_model,
  request_type,
  upstream_url,
  user_agent,
  error_phase,
  error_type,
  severity,
  status_code,
  is_business_limited,
  is_count_tokens,
  error_message,
  error_body,
  error_source,
  error_owner,
  upstream_status_code,
  upstream_error_message,
  upstream_error_detail,
  upstream_errors,
  auth_latency_ms,
  routing_latency_ms,
  upstream_latency_ms,
  response_latency_ms,
  time_to_first_token_ms,
  request_body,
  request_body_truncated,
  request_body_bytes,
  request_headers,
  gemini_surface,
  billing_rule_id,
  probe_action,
  is_retryable,
  retry_count,
  created_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48
)`

func (r *opsRepository) InsertErrorLog(ctx context.Context, input *service.OpsInsertErrorLogInput) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if input == nil {
		return 0, fmt.Errorf("nil input")
	}

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		insertOpsErrorLogSQL+" RETURNING id",
		opsInsertErrorLogArgs(input)...,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *opsRepository) BatchInsertErrorLogs(ctx context.Context, inputs []*service.OpsInsertErrorLogInput) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if len(inputs) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, insertOpsErrorLogSQL)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = stmt.Close()
	}()

	var inserted int64
	for _, input := range inputs {
		if input == nil {
			continue
		}
		if _, err = stmt.ExecContext(ctx, opsInsertErrorLogArgs(input)...); err != nil {
			return inserted, err
		}
		inserted++
	}

	if err = tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

func opsInsertErrorLogArgs(input *service.OpsInsertErrorLogInput) []any {
	return []any{
		opsNullString(input.RequestID),
		opsNullString(input.ClientRequestID),
		opsNullInt64(input.UserID),
		opsNullInt64(input.APIKeyID),
		opsNullInt64(input.AccountID),
		opsNullInt64(input.GroupID),
		opsNullInt64(input.ChannelID),
		opsNullString(input.ClientIP),
		opsNullString(input.Platform),
		opsNullString(input.Model),
		opsNullString(input.RequestPath),
		input.Stream,
		opsNullString(input.InboundEndpoint),
		opsNullString(input.UpstreamEndpoint),
		opsNullString(input.RequestedModel),
		opsNullString(input.UpstreamModel),
		opsNullInt16(input.RequestType),
		opsNullString(input.UpstreamURL),
		opsNullString(input.UserAgent),
		input.ErrorPhase,
		input.ErrorType,
		opsNullString(input.Severity),
		opsNullInt(input.StatusCode),
		input.IsBusinessLimited,
		input.IsCountTokens,
		opsNullString(input.ErrorMessage),
		opsNullString(input.ErrorBody),
		opsNullString(input.ErrorSource),
		opsNullString(input.ErrorOwner),
		opsNullInt(input.UpstreamStatusCode),
		opsNullString(input.UpstreamErrorMessage),
		opsNullString(input.UpstreamErrorDetail),
		opsNullString(input.UpstreamErrorsJSON),
		opsNullInt64(input.AuthLatencyMs),
		opsNullInt64(input.RoutingLatencyMs),
		opsNullInt64(input.UpstreamLatencyMs),
		opsNullInt64(input.ResponseLatencyMs),
		opsNullInt64(input.TimeToFirstTokenMs),
		opsNullString(input.RequestBodyJSON),
		input.RequestBodyTruncated,
		opsNullInt(input.RequestBodyBytes),
		opsNullString(input.RequestHeadersJSON),
		opsNullString(input.GeminiSurface),
		opsNullString(input.BillingRuleID),
		opsNullString(input.ProbeAction),
		input.IsRetryable,
		input.RetryCount,
		input.CreatedAt,
	}
}
