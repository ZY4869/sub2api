package service

import (
	"bytes"
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

type grokGatewayQueuedHTTPUpstream struct {
	responses []*http.Response
	requests  []*http.Request
	callCount int
}

func (s *grokGatewayQueuedHTTPUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	var bodyBytes []byte
	if req != nil && req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	if req != nil {
		cloned := req.Clone(req.Context())
		cloned.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		s.requests = append(s.requests, cloned)
	}
	idx := s.callCount
	s.callCount++
	if idx >= len(s.responses) {
		idx = len(s.responses) - 1
	}
	return s.responses[idx], nil
}

func (s *grokGatewayQueuedHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *TLSFingerprintProfile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, accountConcurrency)
}

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

func TestGrokGatewayRouteModeDistinguishesRuntimeTypes(t *testing.T) {
	svc := &GrokGatewayService{}

	require.Equal(t, GrokRouteModeAPIKey, svc.RouteMode(&Account{
		Platform: PlatformGrok,
		Type:     AccountTypeAPIKey,
	}))
	require.Equal(t, GrokRouteModeOAuthBuild, svc.RouteMode(&Account{
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
	}))
	require.Equal(t, GrokRouteModeSSOReverse, svc.RouteMode(&Account{
		Platform: PlatformGrok,
		Type:     AccountTypeSSO,
	}))
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

func TestGrokOAuthForwardUsesCLIGatewayWithAccessToken(t *testing.T) {
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
			"base_url":     "https://api.x.ai/v1",
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
	require.Equal(t, GrokRouteModeOAuthBuild, result.RouteMode)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "cli-chat-proxy.grok.com", upstream.lastReq.URL.Host)
	require.Equal(t, "/v1/responses", upstream.lastReq.URL.Path)
	require.Equal(t, "Bearer oauth-token", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, grokUpstreamUserAgent, upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, grokCLIVersion, upstream.lastReq.Header.Get("X-Grok-Client-Version"))
	require.Equal(t, "cli-chat-proxy.grok.com", result.EffectiveHost)
	require.Equal(t, "/v1/responses", result.EffectiveEndpoint)
}

func TestGrokAPIKeyRequestRejectsOAuthAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","input":"hello"}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/responses", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_oauth"}`)),
	}}
	account := &Account{
		ID:       188,
		Name:     "grok-oauth",
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"base_url":     "https://api.x.ai/v1",
		},
	}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	resp, meta, err := svc.doAPIKeyRequest(context.Background(), c, account, http.MethodPost, grokEndpointResponses, body)

	require.Error(t, err)
	require.Nil(t, resp)
	require.Nil(t, upstream.lastReq)
	require.Equal(t, GrokRouteModeAPIKey, meta.RouteMode)
}

func TestGrokOAuthForwardChatCompletionsUsesCLIGatewayWithHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4.5","messages":[{"role":"user","content":"hello"}]}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/chat/completions", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-chat"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"chatcmpl_1","usage":{"prompt_tokens":1,"completion_tokens":1},"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)),
	}}
	account := &Account{
		ID:       89,
		Name:     "grok-oauth",
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"base_url":     "https://api.x.ai/v1",
		},
	}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardChatCompletions(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, GrokRouteModeOAuthBuild, result.RouteMode)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://cli-chat-proxy.grok.com/v1/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer oauth-token", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, grokUpstreamUserAgent, upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, grokCLIVersion, upstream.lastReq.Header.Get("X-Grok-Client-Version"))
	require.Equal(t, "cli-chat-proxy.grok.com", result.EffectiveHost)
	require.Equal(t, "/v1/chat/completions", result.EffectiveEndpoint)
	require.Equal(t, "rid-chat", result.UpstreamRequestID)
}

func TestGrokAPIKeyForwardAcceptsV1BaseURLWithoutDuplicatingVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","input":"hello"}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/responses", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_xai","usage":{"input_tokens":1,"output_tokens":1},"output":[{"content":[{"type":"output_text","text":"ok"}]}]}`)),
	}}
	account := newGrokAPIKeyResponsesTestAccount()
	account.Credentials["base_url"] = "https://api.x.ai/v1"
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardResponses(context.Background(), c, account, body, http.MethodPost, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://api.x.ai/v1/responses", upstream.lastReq.URL.String())
	require.Empty(t, upstream.lastReq.Header.Get("X-Grok-Client-Version"))
}

func TestGrokAPIKeyForwardAcceptsRootBaseURLWithoutDuplicatingVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","input":"hello"}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/responses", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_xai","usage":{"input_tokens":1,"output_tokens":1},"output":[{"content":[{"type":"output_text","text":"ok"}]}]}`)),
	}}
	account := newGrokAPIKeyResponsesTestAccount()
	account.Credentials["base_url"] = "https://api.x.ai"
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardResponses(context.Background(), c, account, body, http.MethodPost, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://api.x.ai/v1/responses", upstream.lastReq.URL.String())
	require.Empty(t, upstream.lastReq.Header.Get("X-Grok-Client-Version"))
	require.Equal(t, "api.x.ai", result.EffectiveHost)
	require.Equal(t, "/v1/responses", result.EffectiveEndpoint)
}

func TestGrokAPIKeyForwardAcceptsCustomV1BaseURLWithoutCLIHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","input":"hello"}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/responses", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"resp_custom","usage":{"input_tokens":1,"output_tokens":1},"output":[{"content":[{"type":"output_text","text":"ok"}]}]}`)),
	}}
	account := newGrokAPIKeyResponsesTestAccount()
	account.Credentials["base_url"] = "https://grok.example.test/v1"
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardResponses(context.Background(), c, account, body, http.MethodPost, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://grok.example.test/v1/responses", upstream.lastReq.URL.String())
	require.Empty(t, upstream.lastReq.Header.Get("X-Grok-Client-Version"))
	require.Equal(t, "grok.example.test", result.EffectiveHost)
	require.Equal(t, "/v1/responses", result.EffectiveEndpoint)
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

func TestGrokGatewayLogsEffectiveMetadataOnUpstreamErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sink, cleanup := captureStructuredLog(t)
	defer cleanup()
	body := []byte(`{"model":"grok-4.5","input":"hello"}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/responses", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-not-found"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"not found"}}`)),
	}}
	account := &Account{
		ID:       90,
		Name:     "grok-oauth",
		Platform: PlatformGrok,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"base_url":     "https://api.x.ai/v1",
		},
	}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	_, err := svc.ForwardResponses(context.Background(), c, account, body, http.MethodPost, "")

	require.Error(t, err)
	require.True(t, sink.ContainsMessage("grok.upstream_http_error"))
	require.True(t, sink.ContainsFieldValue("route_mode", GrokRouteModeOAuthBuild))
	require.True(t, sink.ContainsFieldValue("effective_host", "cli-chat-proxy.grok.com"))
	require.True(t, sink.ContainsFieldValue("effective_endpoint", "/v1/responses"))
	require.True(t, sink.ContainsFieldValue("upstream_status", "404"))
	require.True(t, sink.ContainsFieldValue("upstream_request_id", "rid-not-found"))
}

func TestGrokSSOReverseForwardResultIncludesEffectiveMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","input":"hello"}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/responses", body)
	upstream := &grokGatewayQueuedHTTPUpstream{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(`data: {"result":{"response":{"token":"OK","responseId":"resp_sso","conversationId":"conv_sso"}}}

data: [DONE]
`)),
		},
	}}
	account := &Account{
		ID:       91,
		Name:     "grok-sso",
		Platform: PlatformGrok,
		Type:     AccountTypeSSO,
		Credentials: map[string]any{
			"sso_token": "legacy-token",
		},
		Extra: map[string]any{
			"grok_tier": GrokTierBasic,
		},
	}
	svc := &GrokGatewayService{
		reverseClient: NewGrokReverseClient(upstream, &config.Config{}),
		cfg:           &config.Config{},
	}

	result, err := svc.ForwardResponses(context.Background(), c, account, body, http.MethodPost, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, GrokRouteModeSSOReverse, result.RouteMode)
	require.Equal(t, "grok.com", result.EffectiveHost)
	require.Equal(t, "/rest/app-chat/conversations/new", result.EffectiveEndpoint)
	require.Equal(t, "resp_sso", result.UpstreamRequestID)
	require.Len(t, upstream.requests, 1)
	require.Equal(t, "https://grok.com/rest/app-chat/conversations/new", upstream.requests[0].URL.String())
}

func TestGrokOfficialVideoCreateForwardResultIncludesEffectiveMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-imagine-video","prompt":"make a clip"}`)
	_, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/videos/generations", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-video"}},
		Body:       io.NopCloser(strings.NewReader(`{"request_id":"vid_1","status":"completed","url":"https://cdn.example/video.mp4","model":"grok-imagine-video"}`)),
	}}
	account := newGrokAPIKeyResponsesTestAccount()
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardVideosGeneration(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, GrokRouteModeAPIKey, result.RouteMode)
	require.Equal(t, "api.x.ai", result.EffectiveHost)
	require.Equal(t, "/v1/videos/generations", result.EffectiveEndpoint)
	require.Equal(t, "vid_1", result.UpstreamRequestID)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://api.x.ai/v1/videos/generations", upstream.lastReq.URL.String())
}
