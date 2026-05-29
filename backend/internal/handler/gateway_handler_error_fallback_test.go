package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayEnsureForwardErrorResponse_WritesFallbackWhenNotWritten(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	h := &GatewayHandler{}
	wrote := h.ensureForwardErrorResponse(c, false)

	require.True(t, wrote)
	require.Equal(t, http.StatusBadGateway, w.Code)

	var parsed map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "error", parsed["type"])
	errorObj, ok := parsed["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "upstream_error", errorObj["type"])
	assert.Equal(t, "Upstream request failed", errorObj["message"])
}

func TestGatewayEnsureForwardErrorResponse_DoesNotOverrideWrittenResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.String(http.StatusTeapot, "already written")

	h := &GatewayHandler{}
	wrote := h.ensureForwardErrorResponse(c, false)

	require.False(t, wrote)
	require.Equal(t, http.StatusTeapot, w.Code)
	assert.Equal(t, "already written", w.Body.String())
}

func TestGatewayEnsureForwardErrorResponse_AppendsResponsesFailedWhenWritten(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/responses/compact", nil)
	c.Set(ctxKeyInboundEndpoint, EndpointResponses)
	_, err := c.Writer.Write([]byte(":\n\n"))
	require.NoError(t, err)

	h := &GatewayHandler{}
	wrote := h.ensureForwardErrorResponse(c, false)

	require.True(t, wrote)
	body := w.Body.String()
	assert.Contains(t, body, "event: response.failed")
	assert.Contains(t, body, `"type":"response.failed"`)
	assert.Contains(t, body, `"status":"failed"`)
	assert.Contains(t, body, `"code":"upstream_error"`)
}

func TestBillingErrorDetails_UserPlatformQuotaExceededReturns429AndMetrics(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	err := service.ErrUserPlatformQuotaExceeded.WithMetadata(map[string]string{
		"platform": "OpenAI",
		"cycle":    "Daily",
	})

	status, code, message := billingErrorDetails(err)

	require.Equal(t, http.StatusTooManyRequests, status)
	require.Equal(t, "USER_PLATFORM_QUOTA_EXCEEDED", code)
	require.NotEmpty(t, message)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.UserPlatformQuotaRejectionTotal)
	require.Equal(t, int64(1), snapshot.UserPlatformQuotaRejectionByPlatform["openai"])
	require.Equal(t, int64(1), snapshot.UserPlatformQuotaRejectionByCycle["daily"])
}

func TestGatewayHandleStreamingAwareError_AppendsResponsesFailedForResponsePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		path          string
		streamStarted bool
	}{
		{name: "v1 responses stream started", path: "/v1/responses", streamStarted: true},
		{name: "bare responses stream started", path: "/responses", streamStarted: true},
		{name: "codex backend responses stream started", path: "/backend-api/codex/responses", streamStarted: true},
		{name: "compact responses stream started", path: "/responses/compact", streamStarted: true},
		{name: "declared sse before stream started", path: "/v1/responses", streamStarted: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, tt.path, nil)
			c.Set(ctxKeyInboundEndpoint, EndpointResponses)
			c.Header("Content-Type", "text/event-stream")

			h := &GatewayHandler{}
			h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", tt.streamStarted)

			body := w.Body.String()
			assert.Contains(t, body, "event: response.failed")
			assert.Contains(t, body, `"type":"response.failed"`)
			assert.Contains(t, body, `"status":"failed"`)
		})
	}
}

func TestGatewayHandleStreamingAwareError_NonResponsesKeepsGenericSSEError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Header("Content-Type", "text/event-stream")

	h := &GatewayHandler{}
	h.handleStreamingAwareError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", true)

	body := w.Body.String()
	assert.NotContains(t, body, "event: response.failed")
	assert.Contains(t, body, `"type":"error"`)
	assert.Contains(t, body, `"type":"upstream_error"`)
}
