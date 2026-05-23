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

func TestForwardResponsesAsChatCompletions_UsesChatUpstreamAndReturnsResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-4o","input":"hi","stream":false}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(string(body)))

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-force-chat"}},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"chatcmpl_1","object":"chat.completion","model":"gpt-4o","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`,
			)),
		},
	}
	svc := &OpenAIGatewayService{
		httpUpstream:         upstream,
		cfg:                  &config.Config{},
		toolCorrector:        NewCodexToolCorrector(),
		responseHeaderFilter: compileResponseHeaderFilter(&config.Config{}),
	}
	account := &Account{
		ID:          301,
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

	result, err := svc.ForwardResponsesAsChatCompletions(context.Background(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/v1/chat/completions", upstream.lastReq.URL.Path)
	require.Equal(t, "response", gjson.Get(rec.Body.String(), "object").String())
	require.Equal(t, "ok", gjson.Get(rec.Body.String(), "output.0.content.0.text").String())
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
}
