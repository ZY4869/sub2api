package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

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
