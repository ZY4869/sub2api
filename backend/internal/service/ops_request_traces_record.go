package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
	"log"
	"strings"
	"time"
)

func (s *OpsService) RecordRequestTrace(ctx context.Context, input *OpsRecordRequestTraceInput) error {
	if input == nil || !s.IsMonitoringEnabled(ctx) || s.opsRepo == nil {
		return nil
	}

	runtimeCfg := s.getOpsRequestTraceRuntimeConfig(ctx)
	if !runtimeCfg.Enabled {
		return nil
	}

	decision := evaluateOpsRequestTraceDecision(runtimeCfg, input)
	if !decision.Capture {
		return nil
	}

	recordedAt := input.CreatedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	insert := &OpsInsertRequestTraceInput{
		RequestID:           strings.TrimSpace(input.RequestID),
		ClientRequestID:     strings.TrimSpace(input.ClientRequestID),
		UpstreamRequestID:   strings.TrimSpace(firstNonEmptyString(input.UpstreamRequestID, input.Trace.Normalize.UpstreamRequestID)),
		GeminiSurface:       strings.TrimSpace(input.Trace.Normalize.GeminiSurface),
		BillingRuleID:       strings.TrimSpace(input.Trace.Normalize.BillingRuleID),
		ProbeAction:         strings.TrimSpace(input.Trace.Normalize.ProbeAction),
		UserID:              input.UserID,
		APIKeyID:            input.APIKeyID,
		AccountID:           input.AccountID,
		GroupID:             input.GroupID,
		Platform:            strings.TrimSpace(input.Trace.Normalize.Platform),
		ProtocolIn:          strings.TrimSpace(input.Trace.Normalize.ProtocolIn),
		ProtocolOut:         strings.TrimSpace(input.Trace.Normalize.ProtocolOut),
		Channel:             strings.TrimSpace(input.Trace.Normalize.Channel),
		RoutePath:           strings.TrimSpace(input.Trace.Normalize.RoutePath),
		UpstreamPath:        strings.TrimSpace(input.Trace.Normalize.UpstreamPath),
		RequestType:         strings.TrimSpace(input.Trace.Normalize.RequestType),
		RequestedModel:      strings.TrimSpace(input.Trace.Normalize.RequestedModel),
		UpstreamModel:       strings.TrimSpace(input.Trace.Normalize.UpstreamModel),
		ActualUpstreamModel: strings.TrimSpace(input.Trace.Normalize.ActualUpstreamModel),
		Status:              normalizeOpsRequestTraceStatus(input.Status, input.StatusCode),
		StatusCode:          input.StatusCode,
		UpstreamStatusCode:  input.UpstreamStatusCode,
		DurationMs:          input.DurationMs,
		TTFTMs:              input.TTFTMs,
		InputTokens:         input.InputTokens,
		OutputTokens:        input.OutputTokens,
		TotalTokens:         input.TotalTokens,
		FinishReason:        strings.TrimSpace(input.Trace.Normalize.FinishReason),
		PromptBlockReason:   strings.TrimSpace(input.Trace.Normalize.PromptBlockReason),
		Stream:              input.Trace.Normalize.Stream,
		HasTools:            input.Trace.Normalize.HasTools,
		ToolKinds:           dedupeNonEmptyStrings(input.Trace.Normalize.ToolKinds),
		HasThinking:         input.Trace.Normalize.HasThinking,
		ThinkingSource:      strings.TrimSpace(input.Trace.Normalize.ThinkingSource),
		ThinkingLevel:       strings.TrimSpace(input.Trace.Normalize.ThinkingLevel),
		ThinkingBudget:      input.Trace.Normalize.ThinkingBudget,
		MediaResolution:     strings.TrimSpace(input.Trace.Normalize.MediaResolution),
		CountTokensSource:   strings.TrimSpace(input.Trace.Normalize.CountTokensSource),
		CaptureReason:       decision.Reason,
		Sampled:             decision.Sampled,
		CreatedAt:           recordedAt,
	}

	normalizationActions := make([]string, 0, 8)
	normalizeJSONBField := func(field string, source string, contentType string, value *string) *string {
		result := normalizeOpsTraceJSONBPayload(value, source, contentType)
		if result.Action != "" {
			normalizationActions = append(normalizationActions, field+":"+result.Action)
		}
		return result.Value
	}

	insert.InboundRequestJSON = normalizeJSONBField(
		"inbound_request",
		"ops_trace_inbound_request_jsonb",
		"application/json",
		input.Trace.InboundRequestJSON,
	)
	if insert.InboundRequestJSON == nil {
		insert.InboundRequestJSON = normalizeJSONBField(
			"inbound_request_fallback",
			"ops_trace_inbound_request_jsonb_fallback",
			"application/json",
			sanitizeTracePayloadForStorage(input.Trace.RawRequest, runtimeCfg.PayloadPreviewLimitBytes, "application/json"),
		)
	}
	insert.NormalizedRequestJSON = normalizeJSONBField(
		"normalized_request",
		"ops_trace_normalized_request_jsonb",
		"application/json",
		input.Trace.NormalizedRequestJSON,
	)
	insert.UpstreamRequestJSON = normalizeJSONBField(
		"upstream_request",
		"ops_trace_upstream_request_jsonb",
		"application/json",
		input.Trace.UpstreamRequestJSON,
	)
	insert.UpstreamResponseJSON = normalizeJSONBField(
		"upstream_response",
		"ops_trace_upstream_response_jsonb",
		"application/json",
		input.Trace.UpstreamResponseJSON,
	)
	insert.GatewayResponseJSON = normalizeJSONBField(
		"gateway_response",
		"ops_trace_gateway_response_jsonb",
		"application/json",
		input.Trace.GatewayResponseJSON,
	)
	if insert.GatewayResponseJSON == nil {
		insert.GatewayResponseJSON = normalizeJSONBField(
			"gateway_response_fallback",
			"ops_trace_gateway_response_jsonb_fallback",
			"application/json",
			sanitizeTracePayloadForStorage(input.Trace.RawResponse, runtimeCfg.PayloadPreviewLimitBytes, ""),
		)
	}
	insert.ToolTraceJSON = normalizeJSONBField(
		"tool_trace",
		"ops_trace_tool_trace_jsonb",
		"application/json",
		input.Trace.ToolTraceJSON,
	)
	insert.RequestHeadersJSON = normalizeJSONBField(
		"request_headers",
		"ops_trace_request_headers_jsonb",
		"application/json",
		input.Trace.RequestHeadersJSON,
	)
	insert.ResponseHeadersJSON = normalizeJSONBField(
		"response_headers",
		"ops_trace_response_headers_jsonb",
		"application/json",
		input.Trace.ResponseHeadersJSON,
	)
	if len(normalizationActions) > 0 {
		logger.FromContext(ctx).Debug(
			"ops request trace jsonb payload normalized",
			zap.String("request_id", insert.RequestID),
			zap.Strings("actions", normalizationActions),
		)
	}

	if decision.RawEnabled {
		if ciphertext, size, truncated, err := buildEncryptedTracePayload(runtimeCfg.EncryptionKey, input.Trace.RawRequest, opsRequestTraceRawRequestLimit); err != nil {
			log.Printf("[Ops] RecordRequestTrace raw request encryption failed: %v", err)
		} else {
			insert.RawRequestCiphertext = ciphertext
			insert.RawRequestBytes = size
			insert.RawRequestTruncated = truncated
		}
		if ciphertext, size, truncated, err := buildEncryptedTracePayload(runtimeCfg.EncryptionKey, input.Trace.RawResponse, opsRequestTraceRawResponseLimit); err != nil {
			log.Printf("[Ops] RecordRequestTrace raw response encryption failed: %v", err)
		} else {
			insert.RawResponseCiphertext = ciphertext
			insert.RawResponseBytes = size
			insert.RawResponseTruncated = truncated
		}
		insert.RawAvailable = len(insert.RawRequestCiphertext) > 0 || len(insert.RawResponseCiphertext) > 0
	}

	insert.SearchText = buildOpsRequestTraceSearchText(insert)
	if _, err := s.opsRepo.InsertRequestTrace(ctx, insert); err != nil {
		log.Printf("[Ops] RecordRequestTrace failed: %v", err)
		return err
	}
	return nil
}
