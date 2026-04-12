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
