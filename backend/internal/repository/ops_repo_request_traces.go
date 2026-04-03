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

const insertOpsRequestTraceSQL = `
INSERT INTO ops_request_traces (
  request_id,
  client_request_id,
  upstream_request_id,
  user_id,
  api_key_id,
  account_id,
  group_id,
  platform,
  protocol_in,
  protocol_out,
  channel,
  route_path,
  request_type,
  requested_model,
  upstream_model,
  actual_upstream_model,
  status,
  status_code,
  upstream_status_code,
  duration_ms,
  ttft_ms,
  input_tokens,
  output_tokens,
  total_tokens,
  finish_reason,
  prompt_block_reason,
  stream,
  has_tools,
  tool_kinds,
  has_thinking,
  thinking_source,
  thinking_level,
  thinking_budget,
  media_resolution,
  count_tokens_source,
  capture_reason,
  sampled,
  raw_available,
  inbound_request,
  normalized_request,
  upstream_request,
  upstream_response,
  gateway_response,
  tool_trace,
  request_headers,
  response_headers,
  raw_request,
  raw_response,
  raw_request_bytes,
  raw_response_bytes,
  raw_request_truncated,
  raw_response_truncated,
  search_text,
  created_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54
)`

func (r *opsRepository) InsertRequestTrace(ctx context.Context, input *service.OpsInsertRequestTraceInput) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if input == nil {
		return 0, fmt.Errorf("nil input")
	}
	var id int64
	err := r.db.QueryRowContext(ctx, insertOpsRequestTraceSQL+" RETURNING id", opsInsertRequestTraceArgs(input)...).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func opsInsertRequestTraceArgs(input *service.OpsInsertRequestTraceInput) []any {
	return []any{
		opsNullString(input.RequestID),
		opsNullString(input.ClientRequestID),
		opsNullString(input.UpstreamRequestID),
		opsNullInt64(input.UserID),
		opsNullInt64(input.APIKeyID),
		opsNullInt64(input.AccountID),
		opsNullInt64(input.GroupID),
		opsStringOrEmpty(input.Platform),
		opsStringOrEmpty(input.ProtocolIn),
		opsStringOrEmpty(input.ProtocolOut),
		opsStringOrEmpty(input.Channel),
		opsStringOrEmpty(input.RoutePath),
		opsStringOrEmpty(input.RequestType),
		opsStringOrEmpty(input.RequestedModel),
		opsStringOrEmpty(input.UpstreamModel),
		opsStringOrEmpty(input.ActualUpstreamModel),
		opsStringOrEmpty(input.Status),
		input.StatusCode,
		opsNullInt(input.UpstreamStatusCode),
		input.DurationMs,
		opsNullInt64(input.TTFTMs),
		input.InputTokens,
		input.OutputTokens,
		input.TotalTokens,
		opsStringOrEmpty(input.FinishReason),
		opsStringOrEmpty(input.PromptBlockReason),
		input.Stream,
		input.HasTools,
		pq.Array(input.ToolKinds),
		input.HasThinking,
		opsStringOrEmpty(input.ThinkingSource),
		opsStringOrEmpty(input.ThinkingLevel),
		opsNullInt(input.ThinkingBudget),
		opsStringOrEmpty(input.MediaResolution),
		opsStringOrEmpty(input.CountTokensSource),
		opsStringOrEmpty(input.CaptureReason),
		input.Sampled,
		input.RawAvailable,
		opsNullString(input.InboundRequestJSON),
		opsNullString(input.NormalizedRequestJSON),
		opsNullString(input.UpstreamRequestJSON),
		opsNullString(input.UpstreamResponseJSON),
		opsNullString(input.GatewayResponseJSON),
		opsNullString(input.ToolTraceJSON),
		opsNullString(input.RequestHeadersJSON),
		opsNullString(input.ResponseHeadersJSON),
		nullBytes(input.RawRequestCiphertext),
		nullBytes(input.RawResponseCiphertext),
		opsNullInt(input.RawRequestBytes),
		opsNullInt(input.RawResponseBytes),
		input.RawRequestTruncated,
		input.RawResponseTruncated,
		input.SearchText,
		input.CreatedAt,
	}
}

func (r *opsRepository) ListRequestTraces(ctx context.Context, filter *service.OpsRequestTraceFilter) (*service.OpsRequestTraceList, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
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

	where, args := buildOpsRequestTracesWhere(filterCopy)
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

func (r *opsRepository) GetRequestTraceByID(ctx context.Context, id int64) (*service.OpsRequestTraceDetail, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
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

	err := r.db.QueryRowContext(ctx, query, id).Scan(
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
	_, _, startTime, endTime := filter.Normalize()
	filterCopy := &service.OpsRequestTraceFilter{}
	if filter != nil {
		*filterCopy = *filter
	}
	filterCopy.StartTime = &startTime
	filterCopy.EndTime = &endTime

	where, args := buildOpsRequestTracesWhere(filterCopy)
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
		&item.RequestedModel,
		&item.UpstreamModel,
		&item.ActualUpstreamModel,
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
