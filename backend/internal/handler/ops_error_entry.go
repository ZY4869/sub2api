package handler

import (
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func buildOpsErrorResponseLogEntry(c *gin.Context, status int, body []byte, parsed parsedOpsError) *service.OpsInsertErrorLogInput {
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
	var accountID *int64
	if v, ok := accountIDV.(int64); ok && v > 0 {
		accountID = &v
	}

	fallbackPlatform := guessPlatformFromPath(c.Request.URL.Path)
	platform := resolveOpsPlatform(apiKey, fallbackPlatform)

	requestID := c.Writer.Header().Get("X-Request-Id")
	if requestID == "" {
		requestID = c.Writer.Header().Get("x-request-id")
	}

	normalizedType := normalizeOpsErrorType(parsed.ErrorType, parsed.Code)
	phase := classifyOpsPhase(normalizedType, parsed.Message, parsed.Code)
	isBusinessLimited := classifyOpsIsBusinessLimited(normalizedType, phase, parsed.Code, status, parsed.Message)

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
		UserAgent:        c.GetHeader("User-Agent"),

		ErrorPhase:        phase,
		ErrorType:         normalizedType,
		Severity:          classifyOpsSeverity(normalizedType, status),
		StatusCode:        status,
		IsBusinessLimited: isBusinessLimited,
		IsCountTokens:     isCountTokensRequest(c),

		ErrorMessage: parsed.Message,
		// Keep the full captured error body (capture is already capped at 64KB) so the
		// service layer can sanitize JSON before truncating for storage.
		ErrorBody:   string(body),
		ErrorSource: classifyOpsErrorSource(phase, parsed.Message),
		ErrorOwner:  classifyOpsErrorOwner(phase, parsed.Message),

		IsRetryable: classifyOpsIsRetryable(normalizedType, status),
		RetryCount:  0,
		CreatedAt:   time.Now(),
	}
	applyOpsLatencyFieldsFromContext(c, entry)
	applyOpsGeminiMetadataFromContext(c, entry)
	applyOpsUpstreamErrorContextToEntry(c, entry)
	applyOpsAPIKeyAndClientIP(c, apiKey, entry)

	// Persist only a minimal, whitelisted set of request headers to improve retry fidelity.
	// Do NOT store Authorization/Cookie/etc.
	entry.RequestHeadersJSON = extractOpsRetryRequestHeaders(c)
	attachOpsRequestBodyToEntry(c, entry)

	return entry
}

func shouldSkipOpsPassthrough(c *gin.Context) bool {
	if c == nil {
		return false
	}
	if v, ok := c.Get(service.OpsSkipPassthroughKey); ok {
		if skip, _ := v.(bool); skip {
			return true
		}
	}
	return false
}

func applyOpsUpstreamErrorContextToEntry(c *gin.Context, entry *service.OpsInsertErrorLogInput) {
	if c == nil || entry == nil {
		return
	}
	if v, ok := c.Get(service.OpsUpstreamStatusCodeKey); ok {
		switch t := v.(type) {
		case int:
			if t > 0 {
				code := t
				entry.UpstreamStatusCode = &code
			}
		case int64:
			if t > 0 {
				code := int(t)
				entry.UpstreamStatusCode = &code
			}
		}
	}
	if v, ok := c.Get(service.OpsUpstreamErrorMessageKey); ok {
		if s, ok := v.(string); ok {
			if msg := strings.TrimSpace(s); msg != "" {
				entry.UpstreamErrorMessage = &msg
			}
		}
	}
	if v, ok := c.Get(service.OpsUpstreamErrorDetailKey); ok {
		if s, ok := v.(string); ok {
			if detail := strings.TrimSpace(s); detail != "" {
				entry.UpstreamErrorDetail = &detail
			}
		}
	}
	if v, ok := c.Get(service.OpsUpstreamErrorsKey); ok {
		if events, ok := v.([]*service.OpsUpstreamErrorEvent); ok && len(events) > 0 {
			entry.UpstreamErrors = events
			entry.UpstreamURL = latestOpsUpstreamURL(events)
			backfillOpsUpstreamErrorContextFromLastEvent(entry, events)
		}
	}
}

func backfillOpsUpstreamErrorContextFromLastEvent(entry *service.OpsInsertErrorLogInput, events []*service.OpsUpstreamErrorEvent) {
	if entry == nil || len(events) == 0 {
		return
	}
	last := events[len(events)-1]
	if last == nil {
		return
	}
	if entry.UpstreamStatusCode == nil && last.UpstreamStatusCode > 0 {
		code := last.UpstreamStatusCode
		entry.UpstreamStatusCode = &code
	}
	if entry.UpstreamErrorMessage == nil && strings.TrimSpace(last.Message) != "" {
		msg := strings.TrimSpace(last.Message)
		entry.UpstreamErrorMessage = &msg
	}
	if entry.UpstreamErrorDetail == nil && strings.TrimSpace(last.Detail) != "" {
		detail := strings.TrimSpace(last.Detail)
		entry.UpstreamErrorDetail = &detail
	}
}
