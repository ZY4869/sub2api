package repository

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

var opsSQLPlaceholderPattern = regexp.MustCompile(`\$(\d+)`)

func (r *opsRepository) InsertRequestTrace(ctx context.Context, input *service.OpsInsertRequestTraceInput) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if input == nil {
		return 0, fmt.Errorf("nil input")
	}
	schema, err := r.getOpsRequestTraceSchema(ctx)
	if err != nil {
		return 0, err
	}
	query, args := buildInsertOpsRequestTraceSQLAndArgs(input, schema)
	var id int64
	err = r.db.QueryRowContext(ctx, query+" RETURNING id", args...).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func opsInsertRequestTraceArgs(input *service.OpsInsertRequestTraceInput) []any {
	return opsInsertRequestTraceArgsForSchema(input, defaultOpsRequestTraceSchema())
}

func opsInsertRequestTraceArgsForSchema(input *service.OpsInsertRequestTraceInput, schema opsRequestTraceSchema) []any {
	args := make([]any, 0, 58)
	appendArg := func(value any) {
		args = append(args, value)
	}

	appendArg(opsNullString(input.RequestID))
	appendArg(opsNullString(input.ClientRequestID))
	appendArg(opsNullString(input.UpstreamRequestID))
	if schema.HasGeminiSurface {
		appendArg(opsStringOrEmpty(input.GeminiSurface))
	}
	if schema.HasBillingRuleID {
		appendArg(opsStringOrEmpty(input.BillingRuleID))
	}
	if schema.HasProbeAction {
		appendArg(opsStringOrEmpty(input.ProbeAction))
	}
	appendArg(opsNullInt64(input.UserID))
	appendArg(opsNullInt64(input.APIKeyID))
	appendArg(opsNullInt64(input.AccountID))
	appendArg(opsNullInt64(input.GroupID))
	appendArg(opsStringOrEmpty(input.Platform))
	appendArg(opsStringOrEmpty(input.ProtocolIn))
	appendArg(opsStringOrEmpty(input.ProtocolOut))
	appendArg(opsStringOrEmpty(input.Channel))
	appendArg(opsStringOrEmpty(input.RoutePath))
	if schema.HasUpstreamPath {
		appendArg(opsStringOrEmpty(input.UpstreamPath))
	}
	appendArg(opsStringOrEmpty(input.RequestType))
	appendArg(opsStringOrEmpty(input.RequestedModel))
	appendArg(opsStringOrEmpty(input.UpstreamModel))
	appendArg(opsStringOrEmpty(input.ActualUpstreamModel))
	appendArg(opsStringOrEmpty(input.Status))
	appendArg(input.StatusCode)
	appendArg(opsNullInt(input.UpstreamStatusCode))
	appendArg(input.DurationMs)
	appendArg(opsNullInt64(input.TTFTMs))
	appendArg(input.InputTokens)
	appendArg(input.OutputTokens)
	appendArg(input.TotalTokens)
	appendArg(opsStringOrEmpty(input.FinishReason))
	appendArg(opsStringOrEmpty(input.PromptBlockReason))
	appendArg(input.Stream)
	appendArg(input.HasTools)
	appendArg(opsTextArrayOrEmpty(input.ToolKinds))
	appendArg(input.HasThinking)
	appendArg(opsStringOrEmpty(input.ThinkingSource))
	appendArg(opsStringOrEmpty(input.ThinkingLevel))
	appendArg(opsNullInt(input.ThinkingBudget))
	appendArg(opsStringOrEmpty(input.MediaResolution))
	appendArg(opsStringOrEmpty(input.CountTokensSource))
	appendArg(opsStringOrEmpty(input.CaptureReason))
	appendArg(input.Sampled)
	appendArg(input.RawAvailable)
	appendArg(opsNullString(input.InboundRequestJSON))
	appendArg(opsNullString(input.NormalizedRequestJSON))
	appendArg(opsNullString(input.UpstreamRequestJSON))
	appendArg(opsNullString(input.UpstreamResponseJSON))
	appendArg(opsNullString(input.GatewayResponseJSON))
	appendArg(opsNullString(input.ToolTraceJSON))
	appendArg(opsNullString(input.RequestHeadersJSON))
	appendArg(opsNullString(input.ResponseHeadersJSON))
	appendArg(nullBytes(input.RawRequestCiphertext))
	appendArg(nullBytes(input.RawResponseCiphertext))
	appendArg(opsNullInt(input.RawRequestBytes))
	appendArg(opsNullInt(input.RawResponseBytes))
	appendArg(input.RawRequestTruncated)
	appendArg(input.RawResponseTruncated)
	appendArg(input.SearchText)
	appendArg(input.CreatedAt)
	return args
}

func buildInsertOpsRequestTraceSQLAndArgs(input *service.OpsInsertRequestTraceInput, schema opsRequestTraceSchema) (string, []any) {
	columns := make([]string, 0, 58)
	appendColumn := func(name string, supported bool) {
		if supported {
			columns = append(columns, name)
		}
	}

	appendColumn("request_id", true)
	appendColumn("client_request_id", true)
	appendColumn("upstream_request_id", true)
	appendColumn("gemini_surface", schema.HasGeminiSurface)
	appendColumn("billing_rule_id", schema.HasBillingRuleID)
	appendColumn("probe_action", schema.HasProbeAction)
	appendColumn("user_id", true)
	appendColumn("api_key_id", true)
	appendColumn("account_id", true)
	appendColumn("group_id", true)
	appendColumn("platform", true)
	appendColumn("protocol_in", true)
	appendColumn("protocol_out", true)
	appendColumn("channel", true)
	appendColumn("route_path", true)
	appendColumn("upstream_path", schema.HasUpstreamPath)
	appendColumn("request_type", true)
	appendColumn("requested_model", true)
	appendColumn("upstream_model", true)
	appendColumn("actual_upstream_model", true)
	appendColumn("status", true)
	appendColumn("status_code", true)
	appendColumn("upstream_status_code", true)
	appendColumn("duration_ms", true)
	appendColumn("ttft_ms", true)
	appendColumn("input_tokens", true)
	appendColumn("output_tokens", true)
	appendColumn("total_tokens", true)
	appendColumn("finish_reason", true)
	appendColumn("prompt_block_reason", true)
	appendColumn("stream", true)
	appendColumn("has_tools", true)
	appendColumn("tool_kinds", true)
	appendColumn("has_thinking", true)
	appendColumn("thinking_source", true)
	appendColumn("thinking_level", true)
	appendColumn("thinking_budget", true)
	appendColumn("media_resolution", true)
	appendColumn("count_tokens_source", true)
	appendColumn("capture_reason", true)
	appendColumn("sampled", true)
	appendColumn("raw_available", true)
	appendColumn("inbound_request", true)
	appendColumn("normalized_request", true)
	appendColumn("upstream_request", true)
	appendColumn("upstream_response", true)
	appendColumn("gateway_response", true)
	appendColumn("tool_trace", true)
	appendColumn("request_headers", true)
	appendColumn("response_headers", true)
	appendColumn("raw_request", true)
	appendColumn("raw_response", true)
	appendColumn("raw_request_bytes", true)
	appendColumn("raw_response_bytes", true)
	appendColumn("raw_request_truncated", true)
	appendColumn("raw_response_truncated", true)
	appendColumn("search_text", true)
	appendColumn("created_at", true)

	placeholders := make([]string, 0, len(columns))
	for index := range columns {
		placeholders = append(placeholders, "$"+itoa(index+1))
	}

	query := "INSERT INTO ops_request_traces (\n  " + strings.Join(columns, ",\n  ") + "\n) VALUES (\n  " + strings.Join(placeholders, ",") + "\n)"
	return query, opsInsertRequestTraceArgsForSchema(input, schema)
}

func opsTextArrayOrEmpty(items []string) any {
	if items == nil {
		items = []string{}
	}
	return pq.Array(items)
}

func (r *opsRepository) ListRequestTraces(ctx context.Context, filter *service.OpsRequestTraceFilter) (*service.OpsRequestTraceList, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	schema, err := r.getOpsRequestTraceSchema(ctx)
	if err != nil {
		return nil, err
	}
	page, pageSize, startTime, endTime := filter.Normalize()
	filterCopy := &service.OpsRequestTraceFilter{}
	if filter != nil {
		*filterCopy = *filter
	}
	filterCopy.Page = page
	filterCopy.PageSize = pageSize
	filterCopy.StartTime = &startTime
	filterCopy.EndTime = &endTime

	where, args := buildOpsRequestTracesWhereWithSchema(filterCopy, schema)
	countSQL := "SELECT COUNT(*) FROM ops_request_traces t " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, err
	}

	sort := "ORDER BY t.created_at DESC, t.id DESC"
	switch strings.ToLower(strings.TrimSpace(filterCopy.Sort)) {
	case "", "created_at_desc":
	case "duration_desc":
		sort = "ORDER BY t.duration_ms DESC, t.created_at DESC, t.id DESC"
	default:
		return nil, fmt.Errorf("invalid sort")
	}

	offset := (page - 1) * pageSize
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
  COALESCE(a.name, ''),
  COALESCE(g.name, ''),
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
  COALESCE(t.raw_available, false)
FROM ops_request_traces t
LEFT JOIN accounts a ON a.id = t.account_id
LEFT JOIN groups g ON g.id = t.group_id
` + where + `
` + sort + `
LIMIT $` + itoa(len(args)+1) + ` OFFSET $` + itoa(len(args)+2)

	rows, err := r.db.QueryContext(ctx, query, append(args, pageSize, offset)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.OpsRequestTraceListItem, 0, pageSize)
	for rows.Next() {
		item, err := scanOpsRequestTraceListItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &service.OpsRequestTraceList{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

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
  AND COALESCE(t.request_id, '') = $3
ORDER BY t.created_at DESC, t.id DESC
LIMIT 1`

	preview := &service.UsageRequestPreview{Available: true}
	var capturedAt time.Time
	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		apiKeyID,
		strings.TrimSpace(requestID),
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
		return nil, err
	}
	preview.CapturedAt = &capturedAt
	return preview, nil
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

func (r *opsRepository) GetRequestTraceSummary(ctx context.Context, filter *service.OpsRequestTraceFilter) (*service.OpsRequestTraceSummary, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	schema, err := r.getOpsRequestTraceSchema(ctx)
	if err != nil {
		return nil, err
	}
	_, _, startTime, endTime := filter.Normalize()
	filterCopy := &service.OpsRequestTraceFilter{}
	if filter != nil {
		*filterCopy = *filter
	}
	filterCopy.StartTime = &startTime
	filterCopy.EndTime = &endTime

	where, args := buildOpsRequestTracesWhereWithSchema(filterCopy, schema)
	summary := &service.OpsRequestTraceSummary{
		StartTime: startTime,
		EndTime:   endTime,
	}

	totalsSQL := `
SELECT
  COUNT(*)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) < 400)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) >= 400)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.stream, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_tools, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_thinking, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.raw_available, false))::bigint,
  COALESCE(AVG(COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.50) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.95) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.99) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0)
FROM ops_request_traces t
` + where
	if err := r.db.QueryRowContext(ctx, totalsSQL, args...).Scan(
		&summary.Totals.RequestCount,
		&summary.Totals.SuccessCount,
		&summary.Totals.ErrorCount,
		&summary.Totals.StreamCount,
		&summary.Totals.ToolCount,
		&summary.Totals.ThinkingCount,
		&summary.Totals.RawAvailableCount,
		&summary.Totals.AvgDurationMs,
		&summary.Totals.P50DurationMs,
		&summary.Totals.P95DurationMs,
		&summary.Totals.P99DurationMs,
	); err != nil {
		return nil, err
	}

	bucketSeconds := opsRequestTraceBucketSeconds(endTime.Sub(startTime))
	trendSQL := `
SELECT
  to_timestamp(floor(extract(epoch from t.created_at) / $1) * $1) AT TIME ZONE 'UTC' AS bucket_start,
  COUNT(*)::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.status_code, 0) >= 400)::bigint,
  COALESCE(PERCENTILE_DISC(0.50) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.95) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0),
  COALESCE(PERCENTILE_DISC(0.99) WITHIN GROUP (ORDER BY COALESCE(t.duration_ms, 0)), 0)
FROM ops_request_traces t
` + shiftSQLPlaceholders(where, 1) + `
GROUP BY 1
ORDER BY 1 ASC`
	trendArgs := append([]any{bucketSeconds}, args...)
	rows, err := r.db.QueryContext(ctx, trendSQL, trendArgs...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	summary.Trend = make([]*service.OpsRequestTraceSummaryPoint, 0, 64)
	for rows.Next() {
		point := &service.OpsRequestTraceSummaryPoint{}
		if err := rows.Scan(
			&point.BucketStart,
			&point.RequestCount,
			&point.ErrorCount,
			&point.P50DurationMs,
			&point.P95DurationMs,
			&point.P99DurationMs,
		); err != nil {
			return nil, err
		}
		summary.Trend = append(summary.Trend, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	summary.StatusDistribution, err = r.listRequestTraceSummaryBreakdown(ctx, where, args, "COALESCE(t.status, '')", "COALESCE(t.status, '')", 8)
	if err != nil {
		return nil, err
	}
	summary.FinishReasonDistribution, err = r.listRequestTraceSummaryBreakdown(ctx, where, args, "COALESCE(NULLIF(t.finish_reason, ''), 'unknown')", "COALESCE(NULLIF(t.finish_reason, ''), 'unknown')", 10)
	if err != nil {
		return nil, err
	}
	summary.ProtocolPairDistribution, err = r.listRequestTraceSummaryBreakdown(ctx, where, args, "COALESCE(NULLIF(t.protocol_in, ''), 'unknown') || ' -> ' || COALESCE(NULLIF(t.protocol_out, ''), 'unknown')", "COALESCE(NULLIF(t.protocol_in, ''), 'unknown') || ' -> ' || COALESCE(NULLIF(t.protocol_out, ''), 'unknown')", 10)
	if err != nil {
		return nil, err
	}
	summary.ModelDistribution, err = r.listRequestTraceSummaryBreakdown(ctx, where, args, "COALESCE(NULLIF(t.requested_model, ''), NULLIF(t.upstream_model, ''), 'unknown')", "COALESCE(NULLIF(t.requested_model, ''), NULLIF(t.upstream_model, ''), 'unknown')", 10)
	if err != nil {
		return nil, err
	}
	summary.CapabilityDistribution, err = r.listRequestTraceCapabilityBreakdown(ctx, where, args)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

func (r *opsRepository) listRequestTraceSummaryBreakdown(ctx context.Context, where string, args []any, keyExpr string, labelExpr string, limit int) ([]*service.OpsRequestTraceSummaryBreakdownItem, error) {
	query := `
SELECT key, label, count
FROM (
  SELECT
    ` + keyExpr + ` AS key,
    ` + labelExpr + ` AS label,
    COUNT(*)::bigint AS count
  FROM ops_request_traces t
` + where + `
  GROUP BY 1,2
) s
WHERE key <> ''
ORDER BY count DESC, label ASC
LIMIT $` + itoa(len(args)+1)
	rows, err := r.db.QueryContext(ctx, query, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	items := make([]*service.OpsRequestTraceSummaryBreakdownItem, 0, limit)
	for rows.Next() {
		item := &service.OpsRequestTraceSummaryBreakdownItem{}
		if err := rows.Scan(&item.Key, &item.Label, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *opsRepository) listRequestTraceCapabilityBreakdown(ctx context.Context, where string, args []any) ([]*service.OpsRequestTraceSummaryBreakdownItem, error) {
	query := `
SELECT
  COUNT(*) FILTER (WHERE COALESCE(t.stream, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_tools, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.has_thinking, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.raw_available, false))::bigint,
  COUNT(*) FILTER (WHERE COALESCE(t.count_tokens_source, '') = 'estimated')::bigint
FROM ops_request_traces t
` + where
	var streamCount, toolCount, thinkingCount, rawCount, estimatedCount int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&streamCount, &toolCount, &thinkingCount, &rawCount, &estimatedCount); err != nil {
		return nil, err
	}
	return []*service.OpsRequestTraceSummaryBreakdownItem{
		{Key: "stream", Label: "stream", Count: streamCount},
		{Key: "tools", Label: "tools", Count: toolCount},
		{Key: "thinking", Label: "thinking", Count: thinkingCount},
		{Key: "raw", Label: "raw", Count: rawCount},
		{Key: "estimated_count_tokens", Label: "estimated_count_tokens", Count: estimatedCount},
	}, nil
}

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

func buildOpsRequestTracesWhere(filter *service.OpsRequestTraceFilter) (string, []any) {
	return buildOpsRequestTracesWhereWithSchema(filter, defaultOpsRequestTraceSchema())
}

func buildOpsRequestTracesWhereWithSchema(filter *service.OpsRequestTraceFilter, schema opsRequestTraceSchema) (string, []any) {
	clauses := make([]string, 0, 20)
	args := make([]any, 0, 20)
	clauses = append(clauses, "1=1")

	if filter == nil {
		return "WHERE " + strings.Join(clauses, " AND "), args
	}

	if filter.StartTime != nil && !filter.StartTime.IsZero() {
		args = append(args, filter.StartTime.UTC())
		clauses = append(clauses, "t.created_at >= $"+itoa(len(args)))
	}
	if filter.EndTime != nil && !filter.EndTime.IsZero() {
		args = append(args, filter.EndTime.UTC())
		clauses = append(clauses, "t.created_at < $"+itoa(len(args)))
	}
	if v := strings.TrimSpace(strings.ToLower(filter.Status)); v != "" {
		switch v {
		case "success":
			clauses = append(clauses, "COALESCE(t.status_code, 0) < 400")
		case "error":
			clauses = append(clauses, "COALESCE(t.status_code, 0) >= 400")
		default:
			args = append(args, v)
			clauses = append(clauses, "LOWER(COALESCE(t.status,'')) = $"+itoa(len(args)))
		}
	}
	addOpsRequestTraceStringFilter := func(column string, value string) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			args = append(args, trimmed)
			clauses = append(clauses, column+" = $"+itoa(len(args)))
		}
	}

	addOpsRequestTraceStringFilter("COALESCE(t.platform,'')", filter.Platform)
	addOpsRequestTraceStringFilter("COALESCE(t.protocol_in,'')", filter.ProtocolIn)
	addOpsRequestTraceStringFilter("COALESCE(t.protocol_out,'')", filter.ProtocolOut)
	addOpsRequestTraceStringFilter("COALESCE(t.channel,'')", filter.Channel)
	addOpsRequestTraceStringFilter("COALESCE(t.route_path,'')", filter.RoutePath)
	addOpsRequestTraceStringFilter("COALESCE(t.request_type,'')", filter.RequestType)
	addOpsRequestTraceStringFilter("COALESCE(t.finish_reason,'')", filter.FinishReason)
	addOpsRequestTraceStringFilter("COALESCE(t.capture_reason,'')", filter.CaptureReason)
	addOpsRequestTraceStringFilter("COALESCE(t.requested_model,'')", filter.RequestedModel)
	addOpsRequestTraceStringFilter("COALESCE(t.upstream_model,'')", filter.UpstreamModel)
	addOpsRequestTraceStringFilter("COALESCE(t.request_id,'')", filter.RequestID)
	addOpsRequestTraceStringFilter("COALESCE(t.client_request_id,'')", filter.ClientRequestID)
	addOpsRequestTraceStringFilter("COALESCE(t.upstream_request_id,'')", filter.UpstreamRequestID)
	addOpsRequestTraceStringFilter(opsRequestTraceOptionalStringExpr("t.gemini_surface", schema.HasGeminiSurface), filter.GeminiSurface)
	addOpsRequestTraceStringFilter(opsRequestTraceOptionalStringExpr("t.billing_rule_id", schema.HasBillingRuleID), filter.BillingRuleID)
	addOpsRequestTraceStringFilter(opsRequestTraceOptionalStringExpr("t.probe_action", schema.HasProbeAction), filter.ProbeAction)

	if filter.UserID != nil && *filter.UserID > 0 {
		args = append(args, *filter.UserID)
		clauses = append(clauses, "t.user_id = $"+itoa(len(args)))
	}
	if filter.APIKeyID != nil && *filter.APIKeyID > 0 {
		args = append(args, *filter.APIKeyID)
		clauses = append(clauses, "t.api_key_id = $"+itoa(len(args)))
	}
	if filter.AccountID != nil && *filter.AccountID > 0 {
		args = append(args, *filter.AccountID)
		clauses = append(clauses, "t.account_id = $"+itoa(len(args)))
	}
	if filter.GroupID != nil && *filter.GroupID > 0 {
		args = append(args, *filter.GroupID)
		clauses = append(clauses, "t.group_id = $"+itoa(len(args)))
	}
	if filter.StatusCode != nil && *filter.StatusCode > 0 {
		args = append(args, *filter.StatusCode)
		clauses = append(clauses, "t.status_code = $"+itoa(len(args)))
	}
	addBoolClause := func(column string, value *bool) {
		if value == nil {
			return
		}
		args = append(args, *value)
		clauses = append(clauses, column+" = $"+itoa(len(args)))
	}
	addBoolClause("COALESCE(t.stream,false)", filter.Stream)
	addBoolClause("COALESCE(t.has_tools,false)", filter.HasTools)
	addBoolClause("COALESCE(t.has_thinking,false)", filter.HasThinking)
	addBoolClause("COALESCE(t.raw_available,false)", filter.RawAvailable)
	addBoolClause("COALESCE(t.sampled,false)", filter.Sampled)

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		args = append(args, like)
		clauses = append(clauses, "(COALESCE(t.search_text,'') ILIKE $"+itoa(len(args))+" OR COALESCE(t.request_id,'') ILIKE $"+itoa(len(args))+" OR COALESCE(t.client_request_id,'') ILIKE $"+itoa(len(args))+" OR COALESCE(t.upstream_request_id,'') ILIKE $"+itoa(len(args))+")")
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

func scanOpsRequestTraceListItem(scanner interface {
	Scan(dest ...any) error
}) (*service.OpsRequestTraceListItem, error) {
	item := &service.OpsRequestTraceListItem{}
	var (
		userID             sql.NullInt64
		apiKeyID           sql.NullInt64
		accountID          sql.NullInt64
		groupID            sql.NullInt64
		upstreamStatusCode sql.NullInt64
		ttft               sql.NullInt64
		thinkingBudget     sql.NullInt64
		toolKinds          []string
	)
	err := scanner.Scan(
		&item.ID,
		&item.CreatedAt,
		&item.RequestID,
		&item.ClientRequestID,
		&item.UpstreamRequestID,
		&item.Platform,
		&item.ProtocolIn,
		&item.ProtocolOut,
		&item.Channel,
		&item.RoutePath,
		&item.RequestType,
		&userID,
		&apiKeyID,
		&accountID,
		&groupID,
		&item.AccountName,
		&item.GroupName,
		&item.RequestedModel,
		&item.UpstreamModel,
		&item.ActualUpstreamModel,
		&item.GeminiSurface,
		&item.BillingRuleID,
		&item.ProbeAction,
		&item.Status,
		&item.StatusCode,
		&upstreamStatusCode,
		&item.DurationMs,
		&ttft,
		&item.InputTokens,
		&item.OutputTokens,
		&item.TotalTokens,
		&item.FinishReason,
		&item.PromptBlockReason,
		&item.Stream,
		&item.HasTools,
		pq.Array(&toolKinds),
		&item.HasThinking,
		&item.ThinkingSource,
		&item.ThinkingLevel,
		&thinkingBudget,
		&item.MediaResolution,
		&item.CountTokensSource,
		&item.CaptureReason,
		&item.Sampled,
		&item.RawAvailable,
	)
	if err != nil {
		return nil, err
	}
	item.ToolKinds = toolKinds
	if userID.Valid {
		v := userID.Int64
		item.UserID = &v
	}
	if apiKeyID.Valid {
		v := apiKeyID.Int64
		item.APIKeyID = &v
	}
	if accountID.Valid {
		v := accountID.Int64
		item.AccountID = &v
	}
	if groupID.Valid {
		v := groupID.Int64
		item.GroupID = &v
	}
	if upstreamStatusCode.Valid {
		v := int(upstreamStatusCode.Int64)
		item.UpstreamStatusCode = &v
	}
	if ttft.Valid {
		v := ttft.Int64
		item.TTFTMs = &v
	}
	if thinkingBudget.Valid {
		v := int(thinkingBudget.Int64)
		item.ThinkingBudget = &v
	}
	return item, nil
}

func normalizeJSONText(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "", "null":
		return ""
	default:
		return value
	}
}

func opsRequestTraceBucketSeconds(window time.Duration) int64 {
	switch {
	case window > 7*24*time.Hour:
		return 24 * 60 * 60
	case window > 24*time.Hour:
		return 60 * 60
	case window > 6*time.Hour:
		return 15 * 60
	default:
		return 5 * 60
	}
}

func nullBytes(v []byte) any {
	if len(v) == 0 {
		return nil
	}
	return v
}

func shiftSQLPlaceholders(query string, offset int) string {
	if offset == 0 || query == "" {
		return query
	}
	return opsSQLPlaceholderPattern.ReplaceAllStringFunc(query, func(placeholder string) string {
		value := strings.TrimPrefix(placeholder, "$")
		number := 0
		for _, ch := range value {
			number = number*10 + int(ch-'0')
		}
		return "$" + itoa(number+offset)
	})
}
