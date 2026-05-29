package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

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
