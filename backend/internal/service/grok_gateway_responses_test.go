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
	for _, key := range []string{"prompt_cache_key", "previous_response_id", "safety_identifier", "service_tier", "metadata", "include"} {
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
