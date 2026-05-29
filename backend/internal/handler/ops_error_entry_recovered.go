package handler

import (
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func buildRecoveredOpsUpstreamErrorEntry(c *gin.Context, status int) *service.OpsInsertErrorLogInput {
	recovered, ok := collectRecoveredOpsUpstreamContext(c)
	if !ok {
		return nil
	}

	apiKey, _ := middleware2.GetAPIKeyFromContext(c)
	clientRequestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string)

	model, _ := c.Get(opsModelKey)
	streamV, _ := c.Get(opsStreamKey)
	accountIDV, _ := c.Get(opsAccountIDKey)

	var modelName string
	if s, ok := model.(string); ok {
		modelName = s
	}
	stream := false
	if b, ok := streamV.(bool); ok {
		stream = b
	}

	accountID := recovered.accountID
	if accountID == nil {
		if v, ok := accountIDV.(int64); ok && v > 0 {
			accountID = &v
		}
	}

	fallbackPlatform := guessPlatformFromPath(c.Request.URL.Path)
	platform := resolveOpsPlatform(apiKey, fallbackPlatform)

	requestID := c.Writer.Header().Get("X-Request-Id")
	if requestID == "" {
		requestID = c.Writer.Header().Get("x-request-id")
	}

	entry := &service.OpsInsertErrorLogInput{
		RequestID:       requestID,
		ClientRequestID: clientRequestID,

		AccountID: accountID,
		ChannelID: resolveOpsChannelID(c),
		Platform:  platform,
		Model:     modelName,
		RequestPath: func() string {
			if c.Request != nil && c.Request.URL != nil {
				return c.Request.URL.Path
			}
			return ""
		}(),
		Stream:           stream,
		InboundEndpoint:  GetInboundEndpoint(c),
		UpstreamEndpoint: resolveOpsUpstreamEndpoint(c, platform),
		RequestedModel:   modelName,
		UpstreamModel:    resolveOpsUpstreamModel(c),
		RequestType:      resolveOpsRequestType(c, stream),
		UpstreamURL:      latestOpsUpstreamURL(recovered.events),
		UserAgent:        c.GetHeader("User-Agent"),

		ErrorPhase: "upstream",
		ErrorType:  "upstream_error",
		// Severity/retryability should reflect the upstream failure, not the final client status (200).
		Severity:          classifyOpsSeverity("upstream_error", recovered.effectiveStatus),
		StatusCode:        status,
		IsBusinessLimited: false,
		IsCountTokens:     isCountTokensRequest(c),

		ErrorMessage: recovered.messageForLog,
		ErrorBody:    "",
		ErrorSource:  "upstream_http",
		ErrorOwner:   "provider",

		UpstreamStatusCode:   recovered.upstreamStatusCode,
		UpstreamErrorMessage: recovered.upstreamErrorMessage,
		UpstreamErrorDetail:  recovered.upstreamErrorDetail,
		UpstreamErrors:       recovered.events,

		IsRetryable: classifyOpsIsRetryable("upstream_error", recovered.effectiveStatus),
		RetryCount:  0,
		CreatedAt:   time.Now(),
	}
	applyOpsLatencyFieldsFromContext(c, entry)
	applyOpsGeminiMetadataFromContext(c, entry)
	applyOpsAPIKeyAndClientIP(c, apiKey, entry)

	// Store request headers/body only when an upstream error occurred to keep overhead minimal.
	entry.RequestHeadersJSON = extractOpsRetryRequestHeaders(c)
	attachOpsRequestBodyToEntry(c, entry)

	// Skip logging if a passthrough rule with skip_monitoring=true matched.
	if shouldSkipOpsPassthrough(c) {
		return nil
	}

	return entry
}

func buildRecoveredOpsMessage(status int, upstreamErrorMessage *string) string {
	recoveredMsg := "Recovered upstream error"
	if status > 0 {
		recoveredMsg += " " + strconvItoa(status)
	}
	if upstreamErrorMessage != nil && strings.TrimSpace(*upstreamErrorMessage) != "" {
		recoveredMsg += ": " + strings.TrimSpace(*upstreamErrorMessage)
	}
	return truncateString(recoveredMsg, 2048)
}
