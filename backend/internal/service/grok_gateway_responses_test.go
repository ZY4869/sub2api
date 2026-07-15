package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newGrokAPIKeyResponsesTestAccount() *Account {
	return &Account{
		ID:          77,
		Name:        "grok-apikey",
		Platform:    PlatformGrok,
		Type:        AccountTypeAPIKey,
		Concurrency: 3,
		Credentials: map[string]any{
			"api_key":  "xai-test",
			"base_url": "https://api.x.ai",
			"model_mapping": map[string]any{
				"grok-4": "grok-4-upstream",
			},
		},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func TestSanitizeGrokForwardResponsesDirectOpenAICompatiblePost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","input":"hello","stream":false,"reasoning":{"effort":"max"},"reasoning_effort":"max","prompt_cache_key":"pc","previous_response_id":"resp_1","safety_identifier":"sid","service_tier":"fast","metadata":{"a":"b"},"include":["reasoning.encrypted_content"]}`)
	rec, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/responses", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_1","model":"grok-4-upstream","usage":{"input_tokens":2,"output_tokens":3},"output":[{"content":[{"type":"output_text","text":"ok"}]}]}`)),
	}}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardResponses(context.Background(), c, newGrokAPIKeyResponsesTestAccount(), body, http.MethodPost, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "/v1/responses", upstream.lastReq.URL.Path)
	require.Equal(t, "grok-4-upstream", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "max", gjson.GetBytes(upstream.lastBody, "reasoning.effort").String())
	require.Equal(t, "max", gjson.GetBytes(upstream.lastBody, "reasoning_effort").String())
	require.Equal(t, "pc", gjson.GetBytes(upstream.lastBody, "prompt_cache_key").String())
	for _, key := range []string{"previous_response_id", "safety_identifier", "service_tier", "metadata", "include"} {
		require.False(t, gjson.GetBytes(upstream.lastBody, key).Exists(), "expected %s to be removed", key)
	}
}

func TestGrokAPIKeyForwardUsesOfficialAPIStyleRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","input":"hello"}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/responses", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"req_xai"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_xai","usage":{"input_tokens":2,"output_tokens":3},"output":[{"content":[{"type":"output_text","text":"ok"}]}]}`)),
	}}
	account := newGrokAPIKeyResponsesTestAccount()
	account.Credentials["base_url"] = "https://relay.example.com/xai/"
	account.Credentials["request_headers"] = map[string]any{"X-Test-Route": "grok"}
	account.Proxy = &Proxy{Protocol: "http", Host: "127.0.0.1", Port: 18080}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardResponses(context.Background(), c, account, body, http.MethodPost, "")

	require.NoError(t, err)
	require.Equal(t, GrokRouteModeAPIKey, result.RouteMode)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https", upstream.lastReq.URL.Scheme)
	require.Equal(t, "relay.example.com", upstream.lastReq.URL.Host)
	require.Equal(t, "/xai/v1/responses", upstream.lastReq.URL.Path)
	require.Equal(t, "Bearer xai-test", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "grok", upstream.lastReq.Header.Get("X-Test-Route"))
}

func TestGrokOAuthForwardUsesOfficialXAIEndpointWithAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","input":"hello"}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/responses", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_oauth","usage":{"input_tokens":1,"output_tokens":1},"output":[{"content":[{"type":"output_text","text":"ok"}]}]}`)),
	}}
	account := &Account{
		ID:          88,
		Name:        "grok-oauth",
		Platform:    PlatformGrok,
		Type:        AccountTypeOAuth,
		Concurrency: 2,
		Credentials: map[string]any{
			"access_token": "oauth-token",
		},
		Status:      StatusActive,
		Schedulable: true,
	}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardResponses(context.Background(), c, account, body, http.MethodPost, "")

	require.NoError(t, err)
	require.Equal(t, GrokRouteModeAPIKey, result.RouteMode)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "api.x.ai", upstream.lastReq.URL.Host)
	require.Equal(t, "/v1/responses", upstream.lastReq.URL.Path)
	require.Equal(t, "Bearer oauth-token", upstream.lastReq.Header.Get("Authorization"))
}

func TestGrokAPIKeyForwardChatCompletionsForwardsConversationID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","messages":[{"role":"user","content":"hello"}]}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/chat/completions", body)
	c.Request.Header.Set("x-grok-conv-id", " conv-123 ")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"chatcmpl_1","usage":{"prompt_tokens":2,"completion_tokens":3},"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)),
	}}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardChatCompletions(context.Background(), c, newGrokAPIKeyResponsesTestAccount(), body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "/v1/chat/completions", upstream.lastReq.URL.Path)
	require.Equal(t, "conv-123", upstream.lastReq.Header.Get("x-grok-conv-id"))
}

func TestGrokAPIKeyForwardChatCompletionsDoesNotInventConversationID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","messages":[{"role":"user","content":"hello"}]}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/chat/completions", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"chatcmpl_1","usage":{"prompt_tokens":2,"completion_tokens":3},"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)),
	}}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardChatCompletions(context.Background(), c, newGrokAPIKeyResponsesTestAccount(), body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Empty(t, upstream.lastReq.Header.Get("x-grok-conv-id"))
}

func TestGrokExtractUsageFromJSONReadsCachedTokensVariants(t *testing.T) {
	usage := grokExtractUsageFromJSON([]byte(`{"usage":{"input_tokens":125,"output_tokens":48,"cached_tokens":98}}`))
	require.Equal(t, 125, usage.InputTokens)
	require.Equal(t, 48, usage.OutputTokens)
	require.Equal(t, 98, usage.CacheReadInputTokens)

	usage = grokExtractUsageFromJSON([]byte(`{"usage":{"prompt_tokens":70,"completion_tokens":12,"prompt_tokens_details":{"cached_tokens":31}}}`))
	require.Equal(t, 70, usage.InputTokens)
	require.Equal(t, 12, usage.OutputTokens)
	require.Equal(t, 31, usage.CacheReadInputTokens)
}

func TestGrokParseSSEUsageReadsResponsesCachedTokensVariants(t *testing.T) {
	var usage ClaudeUsage

	grokParseSSEUsage([]byte(`{"type":"response.completed","response":{"usage":{"input_tokens":125,"output_tokens":48,"prompt_tokens_details":{"cached_tokens":98}}}}`), &usage)

	require.Equal(t, 125, usage.InputTokens)
	require.Equal(t, 48, usage.OutputTokens)
	require.Equal(t, 98, usage.CacheReadInputTokens)

	grokParseSSEUsage([]byte(`{"usage":{"input_tokens":130,"output_tokens":50,"cached_tokens":99}}`), &usage)
	require.Equal(t, 130, usage.InputTokens)
	require.Equal(t, 50, usage.OutputTokens)
	require.Equal(t, 99, usage.CacheReadInputTokens)
}
