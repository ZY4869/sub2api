package service

import "time"

type TracePayloadVisibility string

const (
	TracePayloadVisibilitySanitized TracePayloadVisibility = "sanitized"
	TracePayloadVisibilityRaw       TracePayloadVisibility = "raw"
	TracePayloadVisibilityNone      TracePayloadVisibility = "none"
)

type RequestCaptureDecision struct {
	Capture     bool
	Reason      string
	Sampled     bool
	RawEnabled  bool
	RawReadable bool
}

type ProtocolNormalizeResult struct {
	Platform                         string   `json:"platform"`
	ProtocolIn                       string   `json:"protocol_in"`
	ProtocolOut                      string   `json:"protocol_out"`
	Channel                          string   `json:"channel"`
	RoutePath                        string   `json:"route_path"`
	RequestType                      string   `json:"request_type"`
	RequestedModel                   string   `json:"requested_model"`
	UpstreamModel                    string   `json:"upstream_model"`
	ActualUpstreamModel              string   `json:"actual_upstream_model"`
	FinishReason                     string   `json:"finish_reason"`
	PromptBlockReason                string   `json:"prompt_block_reason"`
	ThinkingSource                   string   `json:"thinking_source"`
	ThinkingLevel                    string   `json:"thinking_level"`
	ThinkingBudget                   *int     `json:"thinking_budget,omitempty"`
	MediaResolution                  string   `json:"media_resolution"`
	ToolKinds                        []string `json:"tool_kinds,omitempty"`
	CountTokensSource                string   `json:"count_tokens_source"`
	UpstreamRequestID                string   `json:"upstream_request_id"`
	GeminiSurface                    string   `json:"gemini_surface,omitempty"`
	GeminiRequestedServiceTier       string   `json:"gemini_requested_service_tier,omitempty"`
	GeminiResolvedServiceTier        string   `json:"gemini_resolved_service_tier,omitempty"`
	GeminiBatchMode                  string   `json:"gemini_batch_mode,omitempty"`
	GeminiCachePhase                 string   `json:"gemini_cache_phase,omitempty"`
	GeminiPublicVersion              string   `json:"gemini_public_version,omitempty"`
	GeminiPublicResource             string   `json:"gemini_public_resource,omitempty"`
	GeminiAliasUsed                  bool     `json:"gemini_alias_used"`
	GeminiModelMetadataSource        string   `json:"gemini_model_metadata_source,omitempty"`
	UpstreamPath                     string   `json:"upstream_path,omitempty"`
	GeminiBillingFallbackReason      string   `json:"gemini_billing_fallback_reason,omitempty"`
	BillingRuleID                    string   `json:"billing_rule_id,omitempty"`
	ProbeAction                      string   `json:"probe_action,omitempty"`
	ImageRouteFamily                 string   `json:"image_route_family,omitempty"`
	ImageAction                      string   `json:"image_action,omitempty"`
	ImageResolvedProvider            string   `json:"image_resolved_provider,omitempty"`
	ImageDisplayModelID              string   `json:"image_display_model_id,omitempty"`
	ImageTargetModelID               string   `json:"image_target_model_id,omitempty"`
	ImageUpstreamEndpoint            string   `json:"image_upstream_endpoint,omitempty"`
	ImageRequestFormat               string   `json:"image_request_format,omitempty"`
	ImageRouteReason                 string   `json:"image_route_reason,omitempty"`
	IncludeServerSideToolInvocations bool     `json:"include_server_side_tool_invocations"`
	HasTools                         bool     `json:"has_tools"`
	HasThinking                      bool     `json:"has_thinking"`
	Stream                           bool     `json:"stream"`
}

type GatewayTraceContext struct {
	Normalize             ProtocolNormalizeResult `json:"normalize"`
	InboundRequestJSON    *string                 `json:"inbound_request_json,omitempty"`
	NormalizedRequestJSON *string                 `json:"normalized_request_json,omitempty"`
	UpstreamRequestJSON   *string                 `json:"upstream_request_json,omitempty"`
	UpstreamResponseJSON  *string                 `json:"upstream_response_json,omitempty"`
	GatewayResponseJSON   *string                 `json:"gateway_response_json,omitempty"`
	ToolTraceJSON         *string                 `json:"tool_trace_json,omitempty"`
	RequestHeadersJSON    *string                 `json:"request_headers_json,omitempty"`
	ResponseHeadersJSON   *string                 `json:"response_headers_json,omitempty"`
	RawRequest            []byte                  `json:"-"`
	RawResponse           []byte                  `json:"-"`
}

type OpsInsertRequestTraceInput struct {
	RequestID             string
	ClientRequestID       string
	UpstreamRequestID     string
	GeminiSurface         string
	BillingRuleID         string
	ProbeAction           string
	UserID                *int64
	APIKeyID              *int64
	AccountID             *int64
	GroupID               *int64
	Platform              string
	ProtocolIn            string
	ProtocolOut           string
	Channel               string
	RoutePath             string
	UpstreamPath          string
	RequestType           string
	RequestedModel        string
	UpstreamModel         string
	ActualUpstreamModel   string
	Status                string
	StatusCode            int
	UpstreamStatusCode    *int
	DurationMs            int64
	TTFTMs                *int64
	InputTokens           int
	OutputTokens          int
	TotalTokens           int
	FinishReason          string
	PromptBlockReason     string
	Stream                bool
	HasTools              bool
	ToolKinds             []string
	HasThinking           bool
	ThinkingSource        string
	ThinkingLevel         string
	ThinkingBudget        *int
	MediaResolution       string
	CountTokensSource     string
	CaptureReason         string
	Sampled               bool
	RawAvailable          bool
	InboundRequestJSON    *string
	NormalizedRequestJSON *string
	UpstreamRequestJSON   *string
	UpstreamResponseJSON  *string
	GatewayResponseJSON   *string
	ToolTraceJSON         *string
	RequestHeadersJSON    *string
	ResponseHeadersJSON   *string
	RawRequestCiphertext  []byte
	RawResponseCiphertext []byte
	RawRequestBytes       *int
	RawResponseBytes      *int
	RawRequestTruncated   bool
	RawResponseTruncated  bool
	SearchText            string
	CreatedAt             time.Time
}

type OpsRequestTraceAuditAction string

const (
	OpsRequestTraceAuditActionViewRaw   OpsRequestTraceAuditAction = "view_raw"
	OpsRequestTraceAuditActionExportCSV OpsRequestTraceAuditAction = "export_csv"
)

type OpsInsertRequestTraceAuditInput struct {
	TraceID    *int64
	OperatorID int64
	Action     OpsRequestTraceAuditAction
	MetaJSON   *string
	CreatedAt  time.Time
}

type OpsRequestTraceFilter struct {
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`

	Status            string `json:"status,omitempty"`
	Platform          string `json:"platform,omitempty"`
	ProtocolIn        string `json:"protocol_in,omitempty"`
	ProtocolOut       string `json:"protocol_out,omitempty"`
	Channel           string `json:"channel,omitempty"`
	RoutePath         string `json:"route_path,omitempty"`
	RequestType       string `json:"request_type,omitempty"`
	FinishReason      string `json:"finish_reason,omitempty"`
	CaptureReason     string `json:"capture_reason,omitempty"`
	RequestedModel    string `json:"requested_model,omitempty"`
	UpstreamModel     string `json:"upstream_model,omitempty"`
	RequestID         string `json:"request_id,omitempty"`
	ClientRequestID   string `json:"client_request_id,omitempty"`
	UpstreamRequestID string `json:"upstream_request_id,omitempty"`
	GeminiSurface     string `json:"gemini_surface,omitempty"`
	BillingRuleID     string `json:"billing_rule_id,omitempty"`
	ProbeAction       string `json:"probe_action,omitempty"`
	Query             string `json:"q,omitempty"`

	UserID     *int64 `json:"user_id,omitempty"`
	APIKeyID   *int64 `json:"api_key_id,omitempty"`
	AccountID  *int64 `json:"account_id,omitempty"`
	GroupID    *int64 `json:"group_id,omitempty"`
	StatusCode *int   `json:"status_code,omitempty"`

	Stream       *bool `json:"stream,omitempty"`
	HasTools     *bool `json:"has_tools,omitempty"`
	HasThinking  *bool `json:"has_thinking,omitempty"`
	RawAvailable *bool `json:"raw_available,omitempty"`
	Sampled      *bool `json:"sampled,omitempty"`

	Sort     string `json:"sort,omitempty"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func (f *OpsRequestTraceFilter) Normalize() (page, pageSize int, startTime, endTime time.Time) {
	page = 1
	pageSize = 50
	endTime = time.Now()
	startTime = endTime.Add(-1 * time.Hour)

	if f == nil {
		return page, pageSize, startTime, endTime
	}

	if f.Page > 0 {
		page = f.Page
	}
	if f.PageSize > 0 {
		pageSize = f.PageSize
	}
	if pageSize > 200 {
		pageSize = 200
	}
	if f.EndTime != nil && !f.EndTime.IsZero() {
		endTime = *f.EndTime
	}
	if f.StartTime != nil && !f.StartTime.IsZero() {
		startTime = *f.StartTime
	} else if f.EndTime != nil && !f.EndTime.IsZero() {
		startTime = endTime.Add(-1 * time.Hour)
	}
	if startTime.After(endTime) {
		startTime, endTime = endTime, startTime
	}

	return page, pageSize, startTime, endTime
}

type OpsRequestTraceListItem struct {
	ID                int64     `json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	RequestID         string    `json:"request_id"`
	ClientRequestID   string    `json:"client_request_id"`
	UpstreamRequestID string    `json:"upstream_request_id"`

	Platform    string `json:"platform"`
	ProtocolIn  string `json:"protocol_in"`
	ProtocolOut string `json:"protocol_out"`
	Channel     string `json:"channel"`
	RoutePath   string `json:"route_path"`
	RequestType string `json:"request_type"`

	UserID    *int64 `json:"user_id,omitempty"`
	APIKeyID  *int64 `json:"api_key_id,omitempty"`
	AccountID *int64 `json:"account_id,omitempty"`
	GroupID   *int64 `json:"group_id,omitempty"`

	RequestedModel      string `json:"requested_model"`
	UpstreamModel       string `json:"upstream_model"`
	ActualUpstreamModel string `json:"actual_upstream_model"`
	GeminiSurface       string `json:"gemini_surface,omitempty"`
	BillingRuleID       string `json:"billing_rule_id,omitempty"`
	ProbeAction         string `json:"probe_action,omitempty"`
	Status              string `json:"status"`
	StatusCode          int    `json:"status_code"`
	UpstreamStatusCode  *int   `json:"upstream_status_code,omitempty"`
	DurationMs          int64  `json:"duration_ms"`
	TTFTMs              *int64 `json:"ttft_ms,omitempty"`
	InputTokens         int    `json:"input_tokens"`
	OutputTokens        int    `json:"output_tokens"`
	TotalTokens         int    `json:"total_tokens"`
	FinishReason        string `json:"finish_reason"`
	PromptBlockReason   string `json:"prompt_block_reason"`

	Stream            bool     `json:"stream"`
	HasTools          bool     `json:"has_tools"`
	ToolKinds         []string `json:"tool_kinds,omitempty"`
	HasThinking       bool     `json:"has_thinking"`
	ThinkingSource    string   `json:"thinking_source"`
	ThinkingLevel     string   `json:"thinking_level"`
	ThinkingBudget    *int     `json:"thinking_budget,omitempty"`
	MediaResolution   string   `json:"media_resolution"`
	CountTokensSource string   `json:"count_tokens_source"`

	CaptureReason    string `json:"capture_reason"`
	Sampled          bool   `json:"sampled"`
	RawAvailable     bool   `json:"raw_available"`
	RawAccessAllowed bool   `json:"raw_access_allowed"`
}

type OpsRequestTraceList struct {
	Items    []*OpsRequestTraceListItem `json:"items"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
}

type OpsRequestTraceAuditLog struct {
	ID         int64                      `json:"id"`
	TraceID    *int64                     `json:"trace_id,omitempty"`
	OperatorID int64                      `json:"operator_id"`
	Action     OpsRequestTraceAuditAction `json:"action"`
	MetaJSON   string                     `json:"meta_json"`
	CreatedAt  time.Time                  `json:"created_at"`
}

type OpsRequestTraceDetail struct {
	OpsRequestTraceListItem
	InboundRequestJSON    string                     `json:"inbound_request_json"`
	NormalizedRequestJSON string                     `json:"normalized_request_json"`
	UpstreamRequestJSON   string                     `json:"upstream_request_json"`
	UpstreamResponseJSON  string                     `json:"upstream_response_json"`
	GatewayResponseJSON   string                     `json:"gateway_response_json"`
	ToolTraceJSON         string                     `json:"tool_trace_json"`
	RequestHeadersJSON    string                     `json:"request_headers_json"`
	ResponseHeadersJSON   string                     `json:"response_headers_json"`
	Audits                []*OpsRequestTraceAuditLog `json:"audits"`
}

type OpsRequestTraceRawDetail struct {
	ID          int64  `json:"id"`
	RequestID   string `json:"request_id"`
	RawRequest  string `json:"raw_request"`
	RawResponse string `json:"raw_response"`
}

type OpsRequestTraceSummaryTotals struct {
	RequestCount      int64   `json:"request_count"`
	SuccessCount      int64   `json:"success_count"`
	ErrorCount        int64   `json:"error_count"`
	StreamCount       int64   `json:"stream_count"`
	ToolCount         int64   `json:"tool_count"`
	ThinkingCount     int64   `json:"thinking_count"`
	RawAvailableCount int64   `json:"raw_available_count"`
	AvgDurationMs     float64 `json:"avg_duration_ms"`
	P50DurationMs     int64   `json:"p50_duration_ms"`
	P95DurationMs     int64   `json:"p95_duration_ms"`
	P99DurationMs     int64   `json:"p99_duration_ms"`
}

type OpsRequestTraceSummaryPoint struct {
	BucketStart   time.Time `json:"bucket_start"`
	RequestCount  int64     `json:"request_count"`
	ErrorCount    int64     `json:"error_count"`
	P50DurationMs int64     `json:"p50_duration_ms"`
	P95DurationMs int64     `json:"p95_duration_ms"`
	P99DurationMs int64     `json:"p99_duration_ms"`
}

type OpsRequestTraceSummaryBreakdownItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int64  `json:"count"`
}

type OpsRequestTraceSummary struct {
	StartTime                time.Time                              `json:"start_time"`
	EndTime                  time.Time                              `json:"end_time"`
	Totals                   OpsRequestTraceSummaryTotals           `json:"totals"`
	Trend                    []*OpsRequestTraceSummaryPoint         `json:"trend"`
	StatusDistribution       []*OpsRequestTraceSummaryBreakdownItem `json:"status_distribution"`
	FinishReasonDistribution []*OpsRequestTraceSummaryBreakdownItem `json:"finish_reason_distribution"`
	ProtocolPairDistribution []*OpsRequestTraceSummaryBreakdownItem `json:"protocol_pair_distribution"`
	ModelDistribution        []*OpsRequestTraceSummaryBreakdownItem `json:"model_distribution"`
	CapabilityDistribution   []*OpsRequestTraceSummaryBreakdownItem `json:"capability_distribution"`
	RawAccessAllowed         bool                                   `json:"raw_access_allowed"`
}
