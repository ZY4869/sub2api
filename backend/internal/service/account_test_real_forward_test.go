//go:build unit

package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAccountTestModeDefaultsToRealForward(t *testing.T) {
	require.Equal(t, AccountTestModeRealForward, normalizeAccountTestMode(""))
	require.Equal(t, AccountTestModeRealForward, normalizeAccountTestMode("unexpected"))
	require.Equal(t, AccountTestModeRealForward, normalizeAccountTestMode("real_forward"))
	require.Equal(t, AccountTestModeHealthCheck, normalizeAccountTestMode("health_check"))
}

func TestAccountTestServiceSendResolvedTestRuntimeMetaEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/test", nil)

	svc := &AccountTestService{}
	svc.prepareTestStream(ctx)
	svc.setResolvedTestRuntimeMeta(ctx, accountTestRuntimeMeta{
		Mode:            AccountTestModeHealthCheck,
		RuntimePlatform: PlatformOpenAI,
		SourceProtocol:  PlatformAnthropic,
		SimulatedClient: GatewayClientProfileCodex,
		InboundEndpoint: EndpointMessages,
		CompatPath:      "anthropic->openai:compat_translate",
		TargetProvider:  PlatformOpenAI,
		TargetModelID:   "gpt-5.4",
		ResolvedModelID: "gpt-5.4",
	})
	svc.sendResolvedTestRuntimeMetaEvents(ctx)

	body := recorder.Body.String()
	require.True(t, strings.Contains(body, `"key":"test_mode"`))
	require.True(t, strings.Contains(body, `"value":"health_check"`))
	require.True(t, strings.Contains(body, `"key":"resolved_platform"`))
	require.True(t, strings.Contains(body, `"value":"openai"`))
	require.True(t, strings.Contains(body, `"key":"resolved_protocol"`))
	require.True(t, strings.Contains(body, `"value":"anthropic"`))
	require.True(t, strings.Contains(body, `"key":"simulated_client"`))
	require.True(t, strings.Contains(body, `"value":"codex"`))
	require.True(t, strings.Contains(body, `"key":"inbound_endpoint"`))
	require.True(t, strings.Contains(body, `"value":"/v1/messages"`))
	require.True(t, strings.Contains(body, `"key":"compat_path"`))
	require.True(t, strings.Contains(body, `compat_translate`))
	require.True(t, strings.Contains(body, `"key":"target_provider"`))
	require.True(t, strings.Contains(body, `"value":"openai"`))
	require.True(t, strings.Contains(body, `"key":"target_model_id"`))
	require.True(t, strings.Contains(body, `"value":"gpt-5.4"`))
	require.True(t, strings.Contains(body, `"key":"resolved_model_id"`))
	require.True(t, strings.Contains(body, `"value":"gpt-5.4"`))
}

func TestBuildAccountTestRuntimeMeta_UsesGatewayOpenAIRequestFormatPreference(t *testing.T) {
	meta := buildAccountTestRuntimeMeta(
		&Account{
			Platform: PlatformProtocolGateway,
			Type:     AccountTypeAPIKey,
			Extra: map[string]any{
				"gateway_protocol":              GatewayProtocolOpenAI,
				"gateway_openai_request_format": GatewayOpenAIRequestFormatChatCompletions,
			},
		},
		AccountTestModeHealthCheck,
		PlatformOpenAI,
		PlatformOpenAI,
		"gpt-5.4",
		"gpt-5.4",
		"",
	)

	require.Equal(t, EndpointChatCompletions, meta.InboundEndpoint)
	require.Equal(t, PlatformOpenAI, meta.SourceProtocol)
}

func TestAccountTestService_OpenAIRealForwardUsesGatewayChatPreference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newGatewayTestContext()

	resp := newJSONResponse(http.StatusOK, "")
	resp.Header.Set("Content-Type", "text/event-stream")
	resp.Body = io.NopCloser(strings.NewReader(
		"data: {\"id\":\"chatcmpl_1\",\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n" +
			"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":3}}\n\n" +
			"data: [DONE]\n\n",
	))
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	gatewaySvc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	svc := &AccountTestService{openAIGatewayService: gatewaySvc}
	account := &Account{
		ID:          12,
		Platform:    PlatformProtocolGateway,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "test-token", "base_url": "https://api.openai.com"},
		Extra: map[string]any{
			"gateway_protocol":              GatewayProtocolOpenAI,
			"gateway_openai_request_format": GatewayOpenAIRequestFormatChatCompletions,
		},
	}

	err := svc.testOpenAIRealForwardConnection(ctx, account, "gpt-5.4", "", "")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "/v1/chat/completions", upstream.requests[0].URL.Path)
	require.Contains(t, recorder.Body.String(), "test_complete")
}

func TestAccountTestService_OpenAIRealForwardImageModelEmitsImageEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := newGatewayTestContext()

	resp := newJSONResponse(http.StatusOK, `{"created":123,"data":[{"b64_json":"QUJD"}],"output_format":"png"}`)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	gatewaySvc := &OpenAIGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}
	svc := &AccountTestService{openAIGatewayService: gatewaySvc}
	account := &Account{
		ID:          12,
		Platform:    PlatformProtocolGateway,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "test-token", "base_url": "https://api.openai.com"},
		Extra: map[string]any{
			"gateway_protocol": GatewayProtocolOpenAI,
		},
	}

	err := svc.testOpenAIRealForwardConnection(ctx, account, "gpt-image-2", "draw a tiny orange cat astronaut", "")
	require.NoError(t, err)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "/v1/images/generations", upstream.requests[0].URL.Path)
	require.Contains(t, recorder.Body.String(), `"type":"image"`)
	require.Contains(t, recorder.Body.String(), `"image_url":"data:image/png;base64,QUJD"`)
	require.Contains(t, recorder.Body.String(), `"type":"test_complete"`)
}
