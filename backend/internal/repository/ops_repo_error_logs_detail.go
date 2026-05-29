package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) GetErrorLogByID(ctx context.Context, id int64) (*service.OpsErrorLogDetail, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if id <= 0 {
		return nil, fmt.Errorf("invalid id")
	}

	q := `
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
  e.resolved_retry_id,
  COALESCE(e.client_request_id, ''),
  COALESCE(e.request_id, ''),
  COALESCE(e.error_message, ''),
  COALESCE(e.error_body, ''),
  e.upstream_status_code,
  COALESCE(e.upstream_error_message, ''),
  COALESCE(e.upstream_error_detail, ''),
  COALESCE(e.upstream_errors::text, ''),
  e.is_business_limited,
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
  COALESCE(e.probe_action, ''),
  COALESCE(e.user_agent, ''),
  e.auth_latency_ms,
  e.routing_latency_ms,
  e.upstream_latency_ms,
  e.response_latency_ms,
  e.time_to_first_token_ms,
  COALESCE(e.request_body::text, ''),
  e.request_body_truncated,
  e.request_body_bytes,
  COALESCE(e.request_headers::text, '')
FROM ops_error_logs e
LEFT JOIN users u ON e.user_id = u.id
LEFT JOIN accounts a ON e.account_id = a.id
LEFT JOIN groups g ON e.group_id = g.id
WHERE e.id = $1
LIMIT 1`

	var out service.OpsErrorLogDetail
	var statusCode sql.NullInt64
	var upstreamStatusCode sql.NullInt64
	var resolvedAt sql.NullTime
	var resolvedBy sql.NullInt64
	var resolvedRetryID sql.NullInt64
	var clientIP sql.NullString
	var userID sql.NullInt64
	var apiKeyID sql.NullInt64
	var accountID sql.NullInt64
	var groupID sql.NullInt64
	var authLatency sql.NullInt64
	var routingLatency sql.NullInt64
	var upstreamLatency sql.NullInt64
	var responseLatency sql.NullInt64
	var ttft sql.NullInt64
	var requestBodyBytes sql.NullInt64
	var requestType sql.NullInt64

	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&out.ID,
		&out.CreatedAt,
		&out.Phase,
		&out.Type,
		&out.Owner,
		&out.Source,
		&out.Severity,
		&statusCode,
		&out.Platform,
		&out.Model,
		&out.IsRetryable,
		&out.RetryCount,
		&out.Resolved,
		&resolvedAt,
		&resolvedBy,
		&resolvedRetryID,
		&out.ClientRequestID,
		&out.RequestID,
		&out.Message,
		&out.ErrorBody,
		&upstreamStatusCode,
		&out.UpstreamErrorMessage,
		&out.UpstreamErrorDetail,
		&out.UpstreamErrors,
		&out.IsBusinessLimited,
		&userID,
		&out.UserEmail,
		&apiKeyID,
		&accountID,
		&out.AccountName,
		&groupID,
		&out.GroupName,
		&clientIP,
		&out.RequestPath,
		&out.Stream,
		&out.InboundEndpoint,
		&out.UpstreamEndpoint,
		&out.RequestedModel,
		&out.UpstreamModel,
		&requestType,
		&out.UpstreamURL,
		&out.GeminiSurface,
		&out.BillingRuleID,
		&out.ProbeAction,
		&out.UserAgent,
		&authLatency,
		&routingLatency,
		&upstreamLatency,
		&responseLatency,
		&ttft,
		&out.RequestBody,
		&out.RequestBodyTruncated,
		&requestBodyBytes,
		&out.RequestHeaders,
	)
	if err != nil {
		return nil, err
	}

	applyOpsErrorLogDetailNulls(&out, opsErrorLogDetailNulls{
		statusCode:         statusCode,
		upstreamStatusCode: upstreamStatusCode,
		resolvedAt:         resolvedAt,
		resolvedBy:         resolvedBy,
		resolvedRetryID:    resolvedRetryID,
		clientIP:           clientIP,
		userID:             userID,
		apiKeyID:           apiKeyID,
		accountID:          accountID,
		groupID:            groupID,
		authLatency:        authLatency,
		routingLatency:     routingLatency,
		upstreamLatency:    upstreamLatency,
		responseLatency:    responseLatency,
		ttft:               ttft,
		requestBodyBytes:   requestBodyBytes,
		requestType:        requestType,
	})

	// Normalize request_body to empty string when stored as JSON null.
	out.RequestBody = strings.TrimSpace(out.RequestBody)
	if out.RequestBody == "null" {
		out.RequestBody = ""
	}
	// Normalize request_headers to empty string when stored as JSON null.
	out.RequestHeaders = strings.TrimSpace(out.RequestHeaders)
	if out.RequestHeaders == "null" {
		out.RequestHeaders = ""
	}
	// Normalize upstream_errors to empty string when stored as JSON null.
	out.UpstreamErrors = strings.TrimSpace(out.UpstreamErrors)
	if out.UpstreamErrors == "null" {
		out.UpstreamErrors = ""
	}

	return &out, nil
}
