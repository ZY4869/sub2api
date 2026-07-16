//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardAsAnthropic_ProtocolGatewayChatPreferenceUsesNativeChatUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"claude-code-gpt","max_tokens":32,"stop_sequences":["END"],"messages":[{"role":"user","content":"hello"}],"stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	resp := newJSONResponse(http.StatusOK, `{"id":"chatcmpl_1","object":"chat.completion","model":"gpt-5.4","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":7,"completion_tokens":2,"total_tokens":9}}`)
	resp.Header.Set("x-request-id", "req_anthropic_chat_bridge")
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:         upstream,
		cfg:                  &config.Config{},
		toolCorrector:        NewCodexToolCorrector(),
		responseHeaderFilter: compileResponseHeaderFilter(&config.Config{}),
	}
	account := newProtocolGatewayOpenAIChatAccountForTest()

	result, err := svc.ForwardAsAnthropic(context.Background(), c, account, body, "", "gpt-5.4")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/v1/chat/completions", upstream.requests[0].URL.Path)
	require.Equal(t, "claude-code-gpt", result.Model)
	require.Equal(t, "gpt-5.4", result.UpstreamModel)
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)

	forwardedBody, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.Equal(t, "gpt-5.4", gjson.GetBytes(forwardedBody, "model").String())
	require.Equal(t, "hello", gjson.GetBytes(forwardedBody, "messages.0.content").String())
	require.Equal(t, int64(128), gjson.GetBytes(forwardedBody, "max_completion_tokens").Int())
	require.Equal(t, "END", gjson.GetBytes(forwardedBody, "stop.0").String())
	require.False(t, gjson.GetBytes(forwardedBody, "stream").Bool())

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "message", gjson.Get(recorder.Body.String(), "type").String())
	require.Equal(t, "claude-code-gpt", gjson.Get(recorder.Body.String(), "model").String())
	require.Equal(t, "OK", gjson.Get(recorder.Body.String(), "content.0.text").String())
	require.Equal(t, "end_turn", gjson.Get(recorder.Body.String(), "stop_reason").String())
}

func TestForwardAsAnthropic_ProtocolGatewayChatPreferenceStreamsAnthropicEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"claude-code-gpt","max_tokens":32,"messages":[{"role":"user","content":"hello"}],"stream":true}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	resp := newJSONResponse(http.StatusOK, "")
	resp.Header.Set("Content-Type", "text/event-stream")
	resp.Header.Set("x-request-id", "req_anthropic_chat_bridge_stream")
	resp.Body = io.NopCloser(strings.NewReader(
		"data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt-5.4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"}}]}\n\n" +
			"data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt-5.4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"OK\"}}]}\n\n" +
			"data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt-5.4\",\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":1,\"total_tokens\":5}}\n\n" +
			"data: [DONE]\n\n",
	))
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:         upstream,
		cfg:                  &config.Config{},
		toolCorrector:        NewCodexToolCorrector(),
		responseHeaderFilter: compileResponseHeaderFilter(&config.Config{}),
	}

	result, err := svc.ForwardAsAnthropic(context.Background(), c, newProtocolGatewayOpenAIChatAccountForTest(), body, "", "gpt-5.4")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/v1/chat/completions", upstream.requests[0].URL.Path)
	require.Equal(t, 4, result.Usage.InputTokens)
	require.Equal(t, 1, result.Usage.OutputTokens)
	require.True(t, result.Stream)

	out := recorder.Body.String()
	require.Contains(t, out, "event: message_start")
	require.Contains(t, out, "event: content_block_delta")
	require.Contains(t, out, `"text":"OK"`)
	require.Contains(t, out, "event: message_stop")
}

func newProtocolGatewayOpenAIChatAccountForTest() *Account {
	return &Account{
		ID:          401,
		Name:        "protocol-gateway-openai-chat",
		Platform:    PlatformProtocolGateway,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://api.openai.com",
		},
		Extra: map[string]any{
			"gateway_protocol":              GatewayProtocolOpenAI,
			"gateway_openai_request_format": GatewayOpenAIRequestFormatChatCompletions,
		},
	}
}
