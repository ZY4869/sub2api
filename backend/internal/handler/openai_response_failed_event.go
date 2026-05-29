package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func isResponsesRequestContext(c *gin.Context) bool {
	if c == nil {
		return false
	}
	inbound := GetInboundEndpoint(c)
	if strings.HasPrefix(strings.TrimSpace(inbound), EndpointResponses) {
		return true
	}
	if c.Request == nil || c.Request.URL == nil {
		return false
	}
	path := strings.TrimRight(strings.TrimSpace(c.Request.URL.Path), "/")
	return path == "/responses" ||
		path == "/v1/responses" ||
		strings.HasPrefix(path, "/responses/") ||
		strings.HasPrefix(path, "/v1/responses/") ||
		strings.HasPrefix(path, "/backend-api/codex/responses")
}

func writeResponsesFailedEvent(c *gin.Context, errType, message string) bool {
	if c == nil || c.Writer == nil {
		return false
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return false
	}
	responseID := responseFailedEventID(c)
	model := responseFailedEventModel(c)
	code := strings.TrimSpace(errType)
	if code == "" {
		code = "upstream_error"
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "Upstream request failed"
	}
	payload := `event: response.failed` + "\n" +
		`data: {"type":"response.failed","response":{"id":` + strconv.Quote(responseID) +
		`,"object":"response","model":` + strconv.Quote(model) +
		`,"status":"failed","output":[],"error":{"code":` + strconv.Quote(code) +
		`,"message":` + strconv.Quote(msg) + `}}}` + "\n\n"
	if _, err := fmt.Fprint(c.Writer, payload); err != nil {
		_ = c.Error(err)
		return false
	}
	flusher.Flush()
	return true
}

func canAppendResponsesFailedEvent(c *gin.Context, streamStarted bool) bool {
	if !isResponsesRequestContext(c) {
		return false
	}
	if streamStarted {
		return true
	}
	if c == nil || c.Writer == nil {
		return false
	}
	if c.Writer.Written() {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(c.Writer.Header().Get("Content-Type")))
	return strings.HasPrefix(contentType, "text/event-stream")
}

func responseFailedEventID(c *gin.Context) string {
	if c != nil && c.Request != nil {
		if id, _ := c.Request.Context().Value(ctxkey.RequestID).(string); strings.TrimSpace(id) != "" {
			return "resp_" + sanitizeResponseFailedID(id)
		}
		if id, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(id) != "" {
			return "resp_" + sanitizeResponseFailedID(id)
		}
	}
	return "resp_" + service.GenerateSafeRequestID()
}

func sanitizeResponseFailedID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return service.GenerateSafeRequestID()
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '-':
			_, _ = b.WriteRune(r)
		}
		if b.Len() >= 96 {
			break
		}
	}
	if b.Len() == 0 {
		return service.GenerateSafeRequestID()
	}
	return b.String()
}

func responseFailedEventModel(c *gin.Context) string {
	if c != nil && c.Request != nil {
		if model := service.OpenAICodexRequestModelForEvent(c.Request.Context()); model != "" {
			return model
		}
	}
	return ""
}
