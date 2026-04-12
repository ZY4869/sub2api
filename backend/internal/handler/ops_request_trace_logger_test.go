package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildOpsTraceNormalizeResult_UsesAccountAwareOpenAIEndpointPair(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(ctxKeyInboundEndpoint, EndpointResponses)
	setOpsEndpointContext(c, "gpt-5.4", service.RequestTypeSync)
	setOpsSelectedAccountDetails(c, &service.Account{
		ID:       42,
		Platform: service.PlatformProtocolGateway,
		Type:     service.AccountTypeAPIKey,
		Extra: map[string]any{
			"gateway_protocol":              service.GatewayProtocolOpenAI,
			"gateway_openai_request_format": service.GatewayOpenAIRequestFormatChatCompletions,
		},
	})

	normalize, usage := buildOpsTraceNormalizeResult(
		c,
		nil,
		[]byte(`{"model":"gpt-5.4"}`),
		[]byte(`{"id":"chatcmpl_1","model":"gpt-5.4","usage":{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}}`),
	)

	require.Equal(t, service.PlatformProtocolGateway, normalize.Platform)
	require.Equal(t, EndpointResponses, normalize.ProtocolIn)
	require.Equal(t, EndpointChatCompletions, normalize.ProtocolOut)
	require.Equal(t, "openai_compat", normalize.Channel)
	require.Equal(t, "/v1/responses", normalize.RoutePath)
	require.Equal(t, "gpt-5.4", normalize.ActualUpstreamModel)
	require.Equal(t, "chatcmpl_1", normalize.UpstreamRequestID)
	require.Equal(t, 7, usage.inputTokens)
	require.Equal(t, 3, usage.outputTokens)
	require.Equal(t, 10, usage.totalTokens)
}

func TestEnrichOpsTraceResponseMetadata_GeminiCanonicalEndpointUsesGoogleParser(t *testing.T) {
	result := &service.ProtocolNormalizeResult{}
	usage := &opsTraceUsage{}

	enrichOpsTraceResponseMetadata(map[string]any{
		"responseId":   "resp-gemini-1",
		"modelVersion": "gemini-2.5-pro",
		"usageMetadata": map[string]any{
			"promptTokenCount":     12,
			"candidatesTokenCount": 5,
			"totalTokenCount":      17,
		},
	}, EndpointGeminiModels, result, usage)

	require.Equal(t, "resp-gemini-1", result.UpstreamRequestID)
	require.Equal(t, "gemini-2.5-pro", result.ActualUpstreamModel)
	require.Equal(t, 12, usage.inputTokens)
	require.Equal(t, 5, usage.outputTokens)
	require.Equal(t, 17, usage.totalTokens)
}
