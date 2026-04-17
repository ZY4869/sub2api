package service

import (
	"context"
	"strings"
	"time"
)

type UsageRequestPreview struct {
	Available             bool       `json:"available"`
	RequestID             string     `json:"request_id"`
	CapturedAt            *time.Time `json:"captured_at"`
	InboundRequestJSON    string     `json:"inbound_request_json"`
	NormalizedRequestJSON string     `json:"normalized_request_json"`
	UpstreamRequestJSON   string     `json:"upstream_request_json"`
	UpstreamResponseJSON  string     `json:"upstream_response_json"`
	GatewayResponseJSON   string     `json:"gateway_response_json"`
	ToolTraceJSON         string     `json:"tool_trace_json"`
}

type UsageRequestPreviewReader interface {
	GetUsageRequestPreviewForUsage(ctx context.Context, usage *UsageLog) (*UsageRequestPreview, error)
}

func (s *UsageService) SetRequestPreviewReader(reader UsageRequestPreviewReader) {
	if s == nil {
		return
	}
	s.requestPreviewReader = reader
}

func (s *UsageService) GetRequestPreview(ctx context.Context, usage *UsageLog) (*UsageRequestPreview, error) {
	if usage == nil {
		return normalizeUsageRequestPreview(nil, ""), nil
	}
	if s == nil || s.requestPreviewReader == nil {
		return normalizeUsageRequestPreview(nil, usage.RequestID), nil
	}

	preview, err := s.requestPreviewReader.GetUsageRequestPreviewForUsage(ctx, usage)
	if err != nil {
		return nil, err
	}
	return normalizeUsageRequestPreview(preview, usage.RequestID), nil
}

func newUnavailableUsageRequestPreview(requestID string) *UsageRequestPreview {
	return &UsageRequestPreview{
		Available: false,
		RequestID: strings.TrimSpace(requestID),
	}
}

func normalizeUsageRequestPreview(preview *UsageRequestPreview, fallbackRequestID string) *UsageRequestPreview {
	if preview == nil {
		return newUnavailableUsageRequestPreview(fallbackRequestID)
	}

	normalized := *preview
	normalized.RequestID = strings.TrimSpace(normalized.RequestID)
	if normalized.RequestID == "" {
		normalized.RequestID = strings.TrimSpace(fallbackRequestID)
	}
	normalized.InboundRequestJSON = normalizeUsagePreviewContent(normalized.InboundRequestJSON)
	normalized.NormalizedRequestJSON = normalizeUsagePreviewContent(normalized.NormalizedRequestJSON)
	normalized.UpstreamRequestJSON = normalizeUsagePreviewContent(normalized.UpstreamRequestJSON)
	normalized.UpstreamResponseJSON = normalizeUsagePreviewContent(normalized.UpstreamResponseJSON)
	normalized.GatewayResponseJSON = normalizeUsagePreviewContent(normalized.GatewayResponseJSON)
	normalized.ToolTraceJSON = normalizeUsagePreviewContent(normalized.ToolTraceJSON)
	if !normalized.Available {
		normalized.CapturedAt = nil
	}
	return &normalized
}

func normalizeUsagePreviewContent(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return value
}
