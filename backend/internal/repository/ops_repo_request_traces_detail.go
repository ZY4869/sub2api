package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

func (r *opsRepository) GetUsageRequestPreview(ctx context.Context, userID, apiKeyID int64, requestID string) (*service.UsageRequestPreview, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}

	query := `
SELECT
  COALESCE(t.request_id, ''),
  t.created_at,
  COALESCE(t.inbound_request, ''),
  COALESCE(t.normalized_request, ''),
  COALESCE(t.upstream_request, ''),
  COALESCE(t.upstream_response, ''),
  COALESCE(t.gateway_response, ''),
  COALESCE(t.tool_trace, '')
FROM ops_request_traces t
WHERE t.user_id = $1
  AND t.api_key_id = $2
  AND (
    COALESCE(t.request_id, '') = $3
    OR COALESCE(t.client_request_id, '') = $3
    OR COALESCE(t.upstream_request_id, '') = $3
    OR ($4 <> '' AND COALESCE(t.client_request_id, '') = $4)
    OR ($5 <> '' AND COALESCE(t.request_id, '') = $5)
    OR ($5 <> '' AND COALESCE(t.upstream_request_id, '') = $5)
  )
ORDER BY t.created_at DESC, t.id DESC
LIMIT 1`

	preview := &service.UsageRequestPreview{Available: true}
	var capturedAt time.Time
	trimmedRequestID := strings.TrimSpace(requestID)
	clientCandidate := usagePreviewClientRequestID(requestID)
	localCandidate := usagePreviewLocalRequestID(requestID)
	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		apiKeyID,
		trimmedRequestID,
		clientCandidate,
		localCandidate,
	).Scan(
		&preview.RequestID,
		&capturedAt,
		&preview.InboundRequestJSON,
		&preview.NormalizedRequestJSON,
		&preview.UpstreamRequestJSON,
		&preview.UpstreamResponseJSON,
		&preview.GatewayResponseJSON,
		&preview.ToolTraceJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.FromContext(ctx).Info(
				"usage request preview trace match failed",
				zap.String("request_id_kind", usagePreviewRequestIDKind(trimmedRequestID)),
				zap.Bool("has_client_candidate", clientCandidate != ""),
				zap.Bool("has_local_candidate", localCandidate != ""),
				zap.Int64("user_id", userID),
				zap.Int64("api_key_id", apiKeyID),
				zap.String("reason", "no_trace_match"),
			)
		}
		return nil, err
	}
	preview.CapturedAt = &capturedAt
	return preview, nil
}

func usagePreviewClientRequestID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if strings.HasPrefix(requestID, "client:") {
		return strings.TrimSpace(strings.TrimPrefix(requestID, "client:"))
	}
	return ""
}

func usagePreviewLocalRequestID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if strings.HasPrefix(requestID, "local:") {
		return strings.TrimSpace(strings.TrimPrefix(requestID, "local:"))
	}
	return ""
}

func usagePreviewRequestIDKind(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	switch {
	case strings.HasPrefix(requestID, "client:"):
		return "client"
	case strings.HasPrefix(requestID, "local:"):
		return "local"
	case strings.HasPrefix(requestID, "generated:"):
		return "generated"
	case requestID == "":
		return "empty"
	default:
		return "direct"
	}
}

func (r *opsRepository) GetRequestTraceByID(ctx context.Context, id int64) (*service.OpsRequestTraceDetail, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	schema, err := r.getOpsRequestTraceSchema(ctx)
	if err != nil {
		return nil, err
	}
	geminiSurfaceExpr := opsRequestTraceOptionalStringExpr("t.gemini_surface", schema.HasGeminiSurface)
	billingRuleExpr := opsRequestTraceOptionalStringExpr("t.billing_rule_id", schema.HasBillingRuleID)
	probeActionExpr := opsRequestTraceOptionalStringExpr("t.probe_action", schema.HasProbeAction)
	query := `
SELECT
  t.id,
  t.created_at,
  COALESCE(t.request_id, ''),
  COALESCE(t.client_request_id, ''),
  COALESCE(t.upstream_request_id, ''),
  COALESCE(t.platform, ''),
  COALESCE(t.protocol_in, ''),
  COALESCE(t.protocol_out, ''),
  COALESCE(t.channel, ''),
  COALESCE(t.route_path, ''),
  COALESCE(t.request_type, ''),
  t.user_id,
  t.api_key_id,
  t.account_id,
  t.group_id,
  COALESCE(t.requested_model, ''),
  COALESCE(t.upstream_model, ''),
  COALESCE(t.actual_upstream_model, ''),
  ` + geminiSurfaceExpr + `,
  ` + billingRuleExpr + `,
  ` + probeActionExpr + `,
  COALESCE(t.status, ''),
  COALESCE(t.status_code, 0),
  t.upstream_status_code,
  COALESCE(t.duration_ms, 0),
  t.ttft_ms,
  COALESCE(t.input_tokens, 0),
  COALESCE(t.output_tokens, 0),
  COALESCE(t.total_tokens, 0),
  COALESCE(t.finish_reason, ''),
  COALESCE(t.prompt_block_reason, ''),
  COALESCE(t.stream, false),
  COALESCE(t.has_tools, false),
  COALESCE(t.tool_kinds, ARRAY[]::text[]),
  COALESCE(t.has_thinking, false),
  COALESCE(t.thinking_source, ''),
  COALESCE(t.thinking_level, ''),
  t.thinking_budget,
  COALESCE(t.media_resolution, ''),
  COALESCE(t.count_tokens_source, ''),
  COALESCE(t.capture_reason, ''),
  COALESCE(t.sampled, false),
  COALESCE(t.raw_available, false),
  COALESCE(t.inbound_request::text, ''),
  COALESCE(t.normalized_request::text, ''),
  COALESCE(t.upstream_request::text, ''),
  COALESCE(t.upstream_response::text, ''),
  COALESCE(t.gateway_response::text, ''),
  COALESCE(t.tool_trace::text, ''),
  COALESCE(t.request_headers::text, ''),
  COALESCE(t.response_headers::text, '')
FROM ops_request_traces t
WHERE t.id = $1
LIMIT 1`

	out := &service.OpsRequestTraceDetail{}
	var (
		userID             sql.NullInt64
		apiKeyID           sql.NullInt64
		accountID          sql.NullInt64
		groupID            sql.NullInt64
		upstreamStatusCode sql.NullInt64
		ttft               sql.NullInt64
		thinkingBudget     sql.NullInt64
		toolKinds          []string
		inboundRequest     string
		normalizedRequest  string
		upstreamRequest    string
		upstreamResponse   string
		gatewayResponse    string
		toolTrace          string
		requestHeaders     string
		responseHeaders    string
	)

	err = r.db.QueryRowContext(ctx, query, id).Scan(
		&out.ID,
		&out.CreatedAt,
		&out.RequestID,
		&out.ClientRequestID,
		&out.UpstreamRequestID,
		&out.Platform,
		&out.ProtocolIn,
		&out.ProtocolOut,
		&out.Channel,
		&out.RoutePath,
		&out.RequestType,
		&userID,
		&apiKeyID,
		&accountID,
		&groupID,
		&out.RequestedModel,
		&out.UpstreamModel,
		&out.ActualUpstreamModel,
		&out.GeminiSurface,
		&out.BillingRuleID,
		&out.ProbeAction,
		&out.Status,
		&out.StatusCode,
		&upstreamStatusCode,
		&out.DurationMs,
		&ttft,
		&out.InputTokens,
		&out.OutputTokens,
		&out.TotalTokens,
		&out.FinishReason,
		&out.PromptBlockReason,
		&out.Stream,
		&out.HasTools,
		pq.Array(&toolKinds),
		&out.HasThinking,
		&out.ThinkingSource,
		&out.ThinkingLevel,
		&thinkingBudget,
		&out.MediaResolution,
		&out.CountTokensSource,
		&out.CaptureReason,
		&out.Sampled,
		&out.RawAvailable,
		&inboundRequest,
		&normalizedRequest,
		&upstreamRequest,
		&upstreamResponse,
		&gatewayResponse,
		&toolTrace,
		&requestHeaders,
		&responseHeaders,
	)
	if err != nil {
		return nil, err
	}

	out.ToolKinds = toolKinds
	if userID.Valid {
		v := userID.Int64
		out.UserID = &v
	}
	if apiKeyID.Valid {
		v := apiKeyID.Int64
		out.APIKeyID = &v
	}
	if accountID.Valid {
		v := accountID.Int64
		out.AccountID = &v
	}
	if groupID.Valid {
		v := groupID.Int64
		out.GroupID = &v
	}
	if upstreamStatusCode.Valid {
		v := int(upstreamStatusCode.Int64)
		out.UpstreamStatusCode = &v
	}
	if ttft.Valid {
		v := ttft.Int64
		out.TTFTMs = &v
	}
	if thinkingBudget.Valid {
		v := int(thinkingBudget.Int64)
		out.ThinkingBudget = &v
	}

	out.InboundRequestJSON = normalizeJSONText(inboundRequest)
	out.NormalizedRequestJSON = normalizeJSONText(normalizedRequest)
	out.UpstreamRequestJSON = normalizeJSONText(upstreamRequest)
	out.UpstreamResponseJSON = normalizeJSONText(upstreamResponse)
	out.GatewayResponseJSON = normalizeJSONText(gatewayResponse)
	out.ToolTraceJSON = normalizeJSONText(toolTrace)
	out.RequestHeadersJSON = normalizeJSONText(requestHeaders)
	out.ResponseHeadersJSON = normalizeJSONText(responseHeaders)

	return out, nil
}

func (r *opsRepository) GetRequestTraceRawByID(ctx context.Context, id int64) (*service.OpsRequestTraceRawDetail, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	out := &service.OpsRequestTraceRawDetail{}
	var rawReq []byte
	var rawResp []byte
	err := r.db.QueryRowContext(ctx, `
SELECT id, COALESCE(request_id,''), raw_request, raw_response
FROM ops_request_traces
WHERE id = $1
LIMIT 1
`, id).Scan(&out.ID, &out.RequestID, &rawReq, &rawResp)
	if err != nil {
		return nil, err
	}
	out.RawRequest = string(rawReq)
	out.RawResponse = string(rawResp)
	return out, nil
}
