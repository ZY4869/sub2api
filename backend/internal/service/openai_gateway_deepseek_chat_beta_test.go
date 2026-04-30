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

func TestForwardNativeChatCompletions_DeepSeekExplicitBetaTrueUsesBetaWithoutPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{
		"model":"deepseek-v4-pro",
		"beta":true,
		"thinking":{"type":"enabled"},
		"messages":[
			{"role":"user","content":"hi"}
		]
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))

	resp := newJSONResponse(http.StatusOK, `{"id":"chatcmpl_1","object":"chat.completion","model":"deepseek-v4-pro","choices":[{"index":0,"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":6,"completion_tokens":3}}`)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          211,
		Name:        "deepseek-explicit-beta-true",
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

	forwardedBody, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(forwardedBody, "beta").Exists())
	require.True(t, gjson.GetBytes(forwardedBody, "thinking").Exists())
}

func TestForwardNativeChatCompletions_DeepSeekExplicitBetaFalseKeepsStableAndStripsBetaFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{
		"model":"deepseek-v4-flash",
		"beta":false,
		"messages":[
			{"role":"user","content":"hi"},
			{"role":"assistant","content":"prefill","prefix":true,"reasoning_content":"draft reasoning"}
		]
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))

	resp := newJSONResponse(http.StatusOK, `{"id":"chatcmpl_1","object":"chat.completion","model":"deepseek-v4-flash","choices":[{"index":0,"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`)
	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          212,
		Name:        "deepseek-explicit-beta-false",
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

	forwardedBody, err := io.ReadAll(upstream.requests[0].Body)
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(forwardedBody, "beta").Exists())
	require.False(t, gjson.GetBytes(forwardedBody, "messages.1.prefix").Exists())
	require.False(t, gjson.GetBytes(forwardedBody, "messages.1.reasoning_content").Exists())
}

func TestForwardNativeChatCompletions_DeepSeekExplicitBetaTrueRejectsUnsupportedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{
		"model":"deepseek-chat",
		"beta":true,
		"messages":[{"role":"user","content":"hi"}]
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))

	upstream := &queuedHTTPUpstream{}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          213,
		Name:        "deepseek-explicit-beta-invalid-model",
		Platform:    PlatformDeepSeek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "test-token",
		},
	}

	result, err := svc.ForwardNativeChatCompletions(context.Background(), c, account, body, "")
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, 0, upstream.callCount)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "deepseek_chat_beta_model_unsupported", gjson.Get(recorder.Body.String(), "error.reason").String())
}

func TestForwardNativeChatCompletions_DeepSeekRejectsInvalidBetaType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{
		"model":"deepseek-v4-flash",
		"beta":"true",
		"messages":[{"role":"user","content":"hi"}]
	}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body)))

	upstream := &queuedHTTPUpstream{}
	svc := &OpenAIGatewayService{
		httpUpstream:  upstream,
		cfg:           &config.Config{},
		toolCorrector: NewCodexToolCorrector(),
	}
	account := &Account{
		ID:          214,
		Name:        "deepseek-invalid-beta-type",
		Platform:    PlatformDeepSeek,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "test-token",
		},
	}

	result, err := svc.ForwardNativeChatCompletions(context.Background(), c, account, body, "")
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, 0, upstream.callCount)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "invalid_beta", gjson.Get(recorder.Body.String(), "error.reason").String())
}
