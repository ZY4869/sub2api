package handler

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func buildOpsRequestTraceInput(ops *service.OpsService, c *gin.Context, writer *opsRequestTraceCaptureWriter, startedAt time.Time) *service.OpsRecordRequestTraceInput {
	if c == nil || c.Request == nil {
		return nil
	}

	apiKey, _ := middleware2.GetAPIKeyFromContext(c)
	subject, _ := middleware2.GetAuthSubjectFromContext(c)

	var userID *int64
	if subject.UserID > 0 {
		userID = &subject.UserID
	}
	var apiKeyID *int64
	var groupID *int64
	if apiKey != nil {
		apiKeyID = &apiKey.ID
		groupID = apiKey.GroupID
		if userID == nil && apiKey.User != nil && apiKey.User.ID > 0 {
			userID = &apiKey.User.ID
		}
	}

	var accountID *int64
	if value, ok := c.Get(opsAccountIDKey); ok {
		if parsed, ok := value.(int64); ok && parsed > 0 {
			accountID = &parsed
		}
	}

	requestBody := getOpsTraceRequestBody(c)
	responseBody := writer.BytesCopy()
	requestHeadersJSON := marshalOpsTraceHeaders(filterOpsTraceHeaders(c.Request.Header, opsTraceRequestHeaderAllowlist))
	responseHeadersJSON := marshalOpsTraceHeaders(filterOpsTraceHeaders(c.Writer.Header(), opsTraceResponseHeaderAllowlist))

	normalize, usage := buildOpsTraceNormalizeResult(c, apiKey, requestBody, responseBody)
	statusCode := c.Writer.Status()
	previewLimit := resolveOpsRequestTracePreviewLimit(ops, c.Request.Context())
	requestContentType := strings.TrimSpace(c.Request.Header.Get("Content-Type"))
	if requestContentType == "" {
		requestContentType = "application/json"
	}
	responseContentType := strings.TrimSpace(c.Writer.Header().Get("Content-Type"))
	inboundKeyFields := service.ExtractOpsTraceKeyFieldsFromBytes(requestBody)
	inboundRequestJSON := service.BuildOpsTracePayloadEnvelopeJSONFromBytes(
		requestBody,
		previewLimit,
		service.OpsTracePayloadStateCaptured,
		"inbound_request_capture",
		requestContentType,
	)
	normalizedKeyFields := service.ExtractOpsTraceKeyFieldsFromPayload(map[string]any{"normalize": normalize})
	normalizedRequestJSON := service.GetOpsTraceNormalizedRequestJSON(c)
	if normalizedRequestJSON == nil {
		normalizedRequestJSON = buildOpsTraceCanonicalNormalizedRequestJSON(normalize)
	}
	upstreamRequestKeyFields := service.ExtractOpsTraceKeyFieldsFromBytes(requestBody)
	upstreamRequestJSON := service.GetOpsTraceUpstreamRequestJSON(c)
	if upstreamRequestJSON == nil && len(requestBody) > 0 {
		upstreamRequestJSON = service.BuildOpsTracePayloadEnvelopeJSONFromBytesWithKeyFields(
			requestBody,
			previewLimit,
			service.OpsTracePayloadStateRawOnly,
			"inbound_request_fallback",
			requestContentType,
			upstreamRequestKeyFields,
		)
	}
	upstreamResponseKeyFields := service.ExtractOpsTraceKeyFieldsFromBytes(responseBody)
	upstreamResponseJSON := service.GetOpsTraceUpstreamResponseJSON(c)
	if upstreamResponseJSON == nil && len(responseBody) > 0 {
		upstreamResponseJSON = service.BuildOpsTracePayloadEnvelopeJSONFromBytesWithKeyFields(
			responseBody,
			previewLimit,
			service.OpsTracePayloadStateRawOnly,
			"gateway_response_fallback",
			responseContentType,
			upstreamResponseKeyFields,
		)
	}
	gatewayResponseKeyFields := service.ExtractOpsTraceKeyFieldsFromBytes(responseBody)
	gatewayResponseJSON := service.GetOpsTraceGatewayResponseJSON(c)
	if gatewayResponseJSON == nil && len(responseBody) > 0 {
		gatewayResponseJSON = service.BuildOpsTracePayloadEnvelopeJSONFromBytesWithKeyFields(
			responseBody,
			previewLimit,
			service.OpsTracePayloadStateCaptured,
			"gateway_response_capture",
			responseContentType,
			gatewayResponseKeyFields,
		)
	}
	toolTraceJSON := service.GetOpsTraceToolTraceJSON(c)
	if toolTraceJSON == nil {
		toolTraceJSON = buildOpsTraceToolSummaryJSON(normalize)
	}
	input := &service.OpsRecordRequestTraceInput{
		RequestID:          strings.TrimSpace(firstNonEmptyString(c.Writer.Header().Get("X-Request-Id"), c.Writer.Header().Get("x-request-id"))),
		ClientRequestID:    readContextString(c, ctxkey.ClientRequestID),
		UpstreamRequestID:  normalize.UpstreamRequestID,
		UserID:             userID,
		APIKeyID:           apiKeyID,
		AccountID:          accountID,
		GroupID:            groupID,
		StatusCode:         statusCode,
		UpstreamStatusCode: readOpsTraceStatusCode(c),
		DurationMs:         time.Since(startedAt).Milliseconds(),
		TTFTMs:             getContextLatencyMs(c, service.OpsTimeToFirstTokenMsKey),
		InputTokens:        usage.inputTokens,
		OutputTokens:       usage.outputTokens,
		TotalTokens:        usage.totalTokens,
		Trace: service.GatewayTraceContext{
			Normalize:                  normalize,
			InboundRequestJSON:         inboundRequestJSON,
			NormalizedRequestJSON:      normalizedRequestJSON,
			UpstreamRequestJSON:        upstreamRequestJSON,
			UpstreamResponseJSON:       upstreamResponseJSON,
			GatewayResponseJSON:        gatewayResponseJSON,
			ToolTraceJSON:              toolTraceJSON,
			RequestHeadersJSON:         requestHeadersJSON,
			ResponseHeadersJSON:        responseHeadersJSON,
			InboundRequestKeyFields:    inboundKeyFields,
			NormalizedRequestKeyFields: normalizedKeyFields,
			UpstreamRequestKeyFields:   upstreamRequestKeyFields,
			UpstreamResponseKeyFields:  upstreamResponseKeyFields,
			GatewayResponseKeyFields:   gatewayResponseKeyFields,
			ToolTraceKeyFields:         service.ExtractOpsTraceKeyFieldsFromPayload(map[string]any{"normalize": normalize}),
			RawRequest:                 requestBody,
			RawResponse:                responseBody,
		},
		CreatedAt: time.Now().UTC(),
	}
	recordImageRouteRuntimeMetrics(normalize, statusCode, input.UpstreamStatusCode, input.DurationMs)
	return input
}

func buildOpsTraceCanonicalNormalizedRequestJSON(normalize service.ProtocolNormalizeResult) *string {
	return service.BuildOpsTracePayloadEnvelopeJSONWithKeyFields(
		service.OpsTracePayloadStateCaptured,
		"canonical_preview",
		map[string]any{"normalize": normalize},
		"application/json",
		false,
		service.ExtractOpsTraceKeyFieldsFromPayload(map[string]any{"normalize": normalize}),
	)
}

func buildOpsTraceToolSummaryJSON(normalize service.ProtocolNormalizeResult) *string {
	if !normalize.HasTools && len(normalize.ToolKinds) == 0 {
		return nil
	}
	payload := map[string]any{
		"has_tools":                            normalize.HasTools,
		"tool_kinds":                           normalize.ToolKinds,
		"include_server_side_tool_invocations": normalize.IncludeServerSideToolInvocations,
	}
	return service.BuildOpsTracePayloadEnvelopeJSONWithKeyFields(
		service.OpsTracePayloadStateCaptured,
		"tool_summary_fallback",
		payload,
		"application/json",
		false,
		service.ExtractOpsTraceKeyFieldsFromPayload(payload),
	)
}

func resolveOpsRequestTracePreviewLimit(ops *service.OpsService, ctx context.Context) int {
	if ops == nil {
		return opsRequestTracePreviewLimit
	}
	if advanced, err := ops.GetOpsAdvancedSettings(ctx); err == nil && advanced != nil && advanced.RequestDetailPayloadPreviewLimitBytes > 0 {
		return advanced.RequestDetailPayloadPreviewLimitBytes
	}
	return opsRequestTracePreviewLimit
}
