package service

import (
	"strconv"
	"strings"
	"time"
)

func buildOpsRequestTraceCSVHeaders(includeRaw bool) []string {
	headers := []string{
		"id", "created_at", "request_id", "client_request_id", "upstream_request_id",
		"platform", "protocol_in", "protocol_out", "channel", "route_path", "request_type",
		"user_id", "api_key_id", "account_id", "group_id",
		"requested_model", "upstream_model", "actual_upstream_model",
		"status", "status_code", "upstream_status_code", "duration_ms", "ttft_ms",
		"input_tokens", "output_tokens", "total_tokens",
		"finish_reason", "prompt_block_reason", "stream", "has_tools", "tool_kinds",
		"has_thinking", "thinking_source", "thinking_level", "thinking_budget",
		"media_resolution", "count_tokens_source", "capture_reason", "sampled", "raw_available",
	}
	if includeRaw {
		headers = append(headers, "raw_request", "raw_response")
	}
	return headers
}

func buildOpsRequestTraceCSVRow(item *OpsRequestTraceListItem) []string {
	return []string{
		strconv.FormatInt(item.ID, 10),
		item.CreatedAt.UTC().Format(time.RFC3339),
		item.RequestID,
		item.ClientRequestID,
		item.UpstreamRequestID,
		item.Platform,
		item.ProtocolIn,
		item.ProtocolOut,
		item.Channel,
		item.RoutePath,
		item.RequestType,
		formatInt64Pointer(item.UserID),
		formatInt64Pointer(item.APIKeyID),
		formatInt64Pointer(item.AccountID),
		formatInt64Pointer(item.GroupID),
		item.RequestedModel,
		item.UpstreamModel,
		item.ActualUpstreamModel,
		item.Status,
		strconv.Itoa(item.StatusCode),
		formatIntPointer(item.UpstreamStatusCode),
		strconv.FormatInt(item.DurationMs, 10),
		formatInt64Pointer(item.TTFTMs),
		strconv.Itoa(item.InputTokens),
		strconv.Itoa(item.OutputTokens),
		strconv.Itoa(item.TotalTokens),
		item.FinishReason,
		item.PromptBlockReason,
		strconv.FormatBool(item.Stream),
		strconv.FormatBool(item.HasTools),
		strings.Join(item.ToolKinds, "|"),
		strconv.FormatBool(item.HasThinking),
		item.ThinkingSource,
		item.ThinkingLevel,
		formatIntPointer(item.ThinkingBudget),
		item.MediaResolution,
		item.CountTokensSource,
		item.CaptureReason,
		strconv.FormatBool(item.Sampled),
		strconv.FormatBool(item.RawAvailable),
	}
}

func formatIntPointer(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func formatInt64Pointer(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}
