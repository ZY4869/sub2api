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

func TestForwardAsChatCompletions_ProtocolGatewayChatPreferenceUsesNativeChatUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hi"}],"stream":true}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))

	resp := newJSONResponse(http.StatusOK, "")
	resp.Header.Set("Content-Type", "text/event-stream")
	resp.Header.Set("x-request-id", "req_native_chat")
	resp.Body = io.NopCloser(strings.NewReader(
		"data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt-5.4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"}}]}\n\n" +
			"data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt-5.4\",\"choices\":[],\"usage\":{\"prompt_tokens\":11,\"completion_tokens\":7}}\n\n" +
			"data: [DONE]\n\n",
	))

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          101,
		Platform:    PlatformProtocolGateway,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "test-token",
			"base_url": "https://api.openai.com",
		},
		Extra: map[string]any{
			"gateway_protocol":              GatewayProtocolOpenAI,
			"gateway_openai_request_format": GatewayOpenAIRequestFormatChatCompletions,
		},
	}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, account, body, "", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/v1/chat/completions", upstream.requests[0].URL.Path)
	require.Equal(t, 11, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.OutputTokens)
	require.Contains(t, recorder.Body.String(), `"content":"ok"`)
	require.NotContains(t, recorder.Body.String(), `"usage"`)
}

func TestForwardNativeChatCompletions_DeepSeekPreservesOfficialFieldsAndUsesBetaForPrefixModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{
		"model":"deepseek-v4-flash",
		"stream":true,
		"thinking":{"type":"enabled"},
		"response_format":{"type":"json_object"},
		"logprobs":true,
		"top_logprobs":3,
		"user_id":"user_123",
		"messages":[
			{"role":"user","content":"hi"},
			{"role":"assistant","content":"prefill","prefix":true,"reasoning_content":"draft reasoning"}
		]
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))

	resp := newJSONResponse(http.StatusOK, "")
	resp.Header.Set("Content-Type", "text/event-stream")
	resp.Header.Set("x-request-id", "req_deepseek_native_chat")
	resp.Body = io.NopCloser(strings.NewReader(
		"data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"deepseek-v4-flash\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"}}]}\n\n" +
			"data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"deepseek-v4-flash\",\"choices\":[],\"usage\":{\"prompt_tokens\":9,\"completion_tokens\":4}}\n\n" +
			"data: [DONE]\n\n",
	))

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          201,
		Name:        "deepseek-native",
		Platform:    PlatformDeepSeek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "test-token",
		},
	}

	result, err := svc.ForwardNativeChatCompletions(context.Background(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/beta/chat/completions", upstream.requests[0].URL.Path)
	require.Equal(t, 9, result.Usage.InputTokens)
	require.Equal(t, 4, result.Usage.OutputTokens)

	forwardedBody, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(forwardedBody, "thinking").Exists())
	require.True(t, gjson.GetBytes(forwardedBody, "response_format").Exists())
	require.True(t, gjson.GetBytes(forwardedBody, "logprobs").Bool())
	require.Equal(t, int64(3), gjson.GetBytes(forwardedBody, "top_logprobs").Int())
	require.Equal(t, "user_123", gjson.GetBytes(forwardedBody, "user_id").String())
	require.True(t, gjson.GetBytes(forwardedBody, "messages.1.prefix").Bool())
	require.Equal(t, "draft reasoning", gjson.GetBytes(forwardedBody, "messages.1.reasoning_content").String())
	require.True(t, gjson.GetBytes(forwardedBody, "stream_options.include_usage").Bool())
	require.Contains(t, recorder.Body.String(), `"content":"ok"`)
	require.NotContains(t, recorder.Body.String(), `"usage"`)
}

func TestForwardNativeChatCompletions_DeepSeekStripsBetaOnlyFieldsForUnsupportedModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{
		"model":"deepseek-chat",
		"thinking":{"type":"enabled"},
		"messages":[
			{"role":"user","content":"hi"},
			{"role":"assistant","content":"prefill","prefix":true,"reasoning_content":"draft reasoning"}
		]
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))

	resp := newJSONResponse(http.StatusOK, `{"id":"chatcmpl_1","object":"chat.completion","model":"deepseek-chat","choices":[{"index":0,"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`)

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          202,
		Name:        "deepseek-stable",
		Platform:    PlatformDeepSeek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "test-token",
		},
	}

	result, err := svc.ForwardNativeChatCompletions(context.Background(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/chat/completions", upstream.requests[0].URL.Path)
	require.Equal(t, 5, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)

	forwardedBody, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(forwardedBody, "thinking").Exists())
	require.False(t, gjson.GetBytes(forwardedBody, "messages.1.prefix").Exists())
	require.False(t, gjson.GetBytes(forwardedBody, "messages.1.reasoning_content").Exists())
	require.Contains(t, recorder.Body.String(), `"content":"ok"`)
}
