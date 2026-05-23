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

func TestForwardDeepSeekCompletions_NonStreamingUsesBetaAndTracksUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"deepseek-v4-flash","prompt":"hello","suffix":" world"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(string(body)))

	resp := newJSONResponse(http.StatusOK, `{"id":"cmpl_1","object":"text_completion","model":"deepseek-v4-flash","choices":[{"index":0,"text":"hello world"}],"usage":{"prompt_tokens":6,"completion_tokens":2}}`)
	resp.Header.Set("x-request-id", "req_deepseek_completion")

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          301,
		Name:        "deepseek-completions",
		Platform:    PlatformDeepSeek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "test-token",
		},
	}

	result, err := svc.ForwardDeepSeekCompletions(context.Background(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/beta/completions", upstream.requests[0].URL.Path)
	require.Equal(t, 6, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
	require.Contains(t, recorder.Body.String(), `"text":"hello world"`)
}

func TestForwardDeepSeekCompletions_StreamingPassesThroughAndTracksUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"deepseek-v4-pro","prompt":"func ", "stream":true}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(string(body)))

	resp := newJSONResponse(http.StatusOK, "")
	resp.Header.Set("Content-Type", "text/event-stream")
	resp.Header.Set("x-request-id", "req_deepseek_completion_stream")
	resp.Body = io.NopCloser(strings.NewReader(
		"data: {\"id\":\"cmpl_1\",\"object\":\"text_completion\",\"model\":\"deepseek-v4-pro\",\"choices\":[{\"index\":0,\"text\":\"ok\"}]}\n\n" +
			"data: {\"id\":\"cmpl_1\",\"object\":\"text_completion\",\"model\":\"deepseek-v4-pro\",\"choices\":[],\"usage\":{\"prompt_tokens\":8,\"completion_tokens\":4}}\n\n" +
			"data: [DONE]\n\n",
	))

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          302,
		Name:        "deepseek-completions-stream",
		Platform:    PlatformDeepSeek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "test-token",
		},
	}

	result, err := svc.ForwardDeepSeekCompletions(context.Background(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/beta/completions", upstream.requests[0].URL.Path)
	require.Equal(t, 8, result.Usage.InputTokens)
	require.Equal(t, 4, result.Usage.OutputTokens)
	require.Contains(t, recorder.Body.String(), `"text":"ok"`)
	require.Contains(t, recorder.Body.String(), `[DONE]`)
}

func TestForwardDeepSeekCompletions_RejectsUnsupportedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"deepseek-chat","prompt":"hello"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(string(body)))

	upstream := &queuedHTTPUpstream{}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          303,
		Name:        "deepseek-completions-invalid",
		Platform:    PlatformDeepSeek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "test-token",
		},
	}

	result, err := svc.ForwardDeepSeekCompletions(context.Background(), c, account, body, "")
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, 0, upstream.callCount)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "unknown_deepseek_model", gjson.Get(recorder.Body.String(), "error.reason").String())
	require.Equal(t, "unknown_deepseek_model", gjson.Get(recorder.Body.String(), "error.code").String())
}

func TestForwardDeepSeekCompletions_VariantCanonicalizesAndInjectsUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"deepseek_v4_pro_free","prompt":"hello","user_id":"client"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(string(body)))
	c.Set("api_key", &APIKey{ID: 12, UserID: 34})

	resp := newJSONResponse(http.StatusOK, `{"id":"cmpl_1","object":"text_completion","model":"deepseek-v4-pro","choices":[{"index":0,"text":"ok"}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{JWT: config.JWTConfig{Secret: "deepseek-secret"}},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          304,
		Name:        "deepseek-completions-variant",
		Platform:    PlatformDeepSeek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1000,
		Credentials: map[string]any{"api_key": "test-token"},
		Extra: map[string]any{
			DeepSeekModelConcurrencyLimitsExtraKey: map[string]any{
				"DEEPSEEK V4 PRO": 500,
			},
		},
	}

	result, err := svc.ForwardDeepSeekCompletions(context.Background(), c, account, body, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "deepseek-v4-pro", result.UpstreamModel)
	require.Equal(t, []int{500}, upstream.concurrencies)

	forwardedBody, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.Equal(t, "deepseek-v4-pro", gjson.GetBytes(forwardedBody, "model").String())
	require.Regexp(t, `^sub2api_[a-f0-9]{40}$`, gjson.GetBytes(forwardedBody, "user_id").String())
	require.NotEqual(t, "client", gjson.GetBytes(forwardedBody, "user_id").String())
}
