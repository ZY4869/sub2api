//go:build unit

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

func newGrokAPIKeyCompatAccount() *Account {
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

func TestBuildGrokMessagesCompatResponsesBodyCleansPayloadAndMapsModel(t *testing.T) {
	body := []byte(`{"model":"grok-4","max_tokens":64,"stream":false,"messages":[{"role":"user","content":"hello"}]}`)

	responsesBody, originalModel, mappedModel, clientStream, err := buildGrokMessagesCompatResponsesBody(newGrokAPIKeyCompatAccount(), body)

	require.NoError(t, err)
	require.Equal(t, "grok-4", originalModel)
	require.Equal(t, "grok-4-upstream", mappedModel)
	require.False(t, clientStream)
	require.Equal(t, "grok-4-upstream", gjson.GetBytes(responsesBody, "model").String())
	require.True(t, gjson.GetBytes(responsesBody, "stream").Bool(), "Grok messages compat should ask upstream for a streaming Responses payload")
	require.False(t, gjson.GetBytes(responsesBody, "store").Exists())

	dirty := []byte(`{"model":"grok-4","input":"hello","prompt_cache_key":"pc","previous_response_id":"resp_1","safety_identifier":"sid","service_tier":"fast","metadata":{"a":"b"},"include":["reasoning.encrypted_content"]}`)
	clean := sanitizeGrokOpenAICompatibleRequestBody(dirty)
	require.Equal(t, "grok-4", gjson.GetBytes(clean, "model").String())
	require.Equal(t, "hello", gjson.GetBytes(clean, "input").String())
	require.Equal(t, "pc", gjson.GetBytes(clean, "prompt_cache_key").String())
	require.False(t, gjson.GetBytes(clean, "previous_response_id").Exists())
	require.False(t, gjson.GetBytes(clean, "safety_identifier").Exists())
	require.False(t, gjson.GetBytes(clean, "service_tier").Exists())
	require.False(t, gjson.GetBytes(clean, "metadata").Exists())
	require.False(t, gjson.GetBytes(clean, "include").Exists())
}

func TestSanitizeGrokOpenAICompatibleRequestBodyPreservesReasoning(t *testing.T) {
	dirty := []byte(`{"model":"grok-4","input":"hello","reasoning":{"effort":"high"},"reasoning_effort":"high","prompt_cache_key":"pc"}`)

	clean := sanitizeGrokOpenAICompatibleRequestBody(dirty)

	require.Equal(t, "high", gjson.GetBytes(clean, "reasoning.effort").String())
	require.Equal(t, "high", gjson.GetBytes(clean, "reasoning_effort").String())
	require.Equal(t, "pc", gjson.GetBytes(clean, "prompt_cache_key").String())
}

func TestSanitizeGrokOpenAICompatibleRequestBodyNormalizesXAIToolShapes(t *testing.T) {
	dirty := []byte(`{
		"model":"grok-4",
		"input":[
			{"type":"custom_tool_call","call_id":"call_1","name":"custom_lookup","input":{"query":"hi"}},
			{"type":"custom_tool_call_output","call_id":"call_1","output":"ok"},
			{"type":"reasoning","summary":[],"content":null,"encrypted_content":null}
		],
		"tools":[
			{"type":"tool_search"},
			{"type":"image_generation"},
			{"type":"custom","name":"apply_patch"},
			{"type":"custom","name":"custom_lookup"},
			{"type":"web_search","external_web_access":true,"search_content_types":["text"]},
			{"type":"namespace","name":"codex_app","tools":[
				{"type":"function","name":"automation_update"},
				{"type":"custom","name":"namespace_custom"}
			]}
		],
		"tool_choice":{"type":"namespace","name":"codex_app"},
		"parallel_tool_calls":true,
		"prompt_cache_key":"pc"
	}`)

	clean := sanitizeGrokOpenAICompatibleRequestBody(dirty)

	tools := gjson.GetBytes(clean, "tools").Array()
	require.Len(t, tools, 4)
	require.Equal(t, "function", gjson.GetBytes(clean, "tools.0.type").String())
	require.Equal(t, "custom_lookup", gjson.GetBytes(clean, "tools.0.name").String())
	require.True(t, gjson.GetBytes(clean, "tools.0.parameters").Exists())
	require.Equal(t, "web_search", gjson.GetBytes(clean, "tools.1.type").String())
	require.False(t, gjson.GetBytes(clean, "tools.1.external_web_access").Exists())
	require.Equal(t, "automation_update", gjson.GetBytes(clean, "tools.2.name").String())
	require.Equal(t, "namespace_custom", gjson.GetBytes(clean, "tools.3.name").String())
	require.False(t, gjson.GetBytes(clean, "tool_choice").Exists())
	require.True(t, gjson.GetBytes(clean, "parallel_tool_calls").Bool())
	require.Equal(t, "pc", gjson.GetBytes(clean, "prompt_cache_key").String())
	require.Equal(t, "function_call", gjson.GetBytes(clean, "input.0.type").String())
	require.Equal(t, "hi", gjson.Parse(gjson.GetBytes(clean, "input.0.arguments").String()).Get("query").String())
	require.False(t, gjson.GetBytes(clean, "input.0.input").Exists())
	require.Equal(t, "function_call_output", gjson.GetBytes(clean, "input.1.type").String())
	require.False(t, gjson.GetBytes(clean, "input.2.content").Exists())
	require.False(t, gjson.GetBytes(clean, "input.2.encrypted_content").Exists())
}

func TestSanitizeGrokOpenAICompatibleRequestBodyDropsOrphanedToolChoice(t *testing.T) {
	dirty := []byte(`{"model":"grok-4","input":"hello","tools":[],"tool_choice":"auto","parallel_tool_calls":true}`)

	clean := sanitizeGrokOpenAICompatibleRequestBody(dirty)

	require.False(t, gjson.GetBytes(clean, "tools").Exists())
	require.False(t, gjson.GetBytes(clean, "tool_choice").Exists())
	require.False(t, gjson.GetBytes(clean, "parallel_tool_calls").Exists())
}

func TestSanitizeGrokOpenAICompatibleRequestBodyKeepsToolChoiceWhenToolsRemain(t *testing.T) {
	dirty := []byte(`{"model":"grok-4","input":"hello","tools":[{"type":"function","name":"lookup"}],"tool_choice":"auto","parallel_tool_calls":true}`)

	clean := sanitizeGrokOpenAICompatibleRequestBody(dirty)

	require.True(t, gjson.GetBytes(clean, "tools").Exists())
	require.Equal(t, "auto", gjson.GetBytes(clean, "tool_choice").String())
	require.True(t, gjson.GetBytes(clean, "parallel_tool_calls").Bool())
}

func TestBuildGrokMessagesCompatResponsesBodyPreservesPromptCacheKey(t *testing.T) {
	body := []byte(`{"model":"grok-4","max_tokens":64,"stream":false,"prompt_cache_key":"pc-msg-1","messages":[{"role":"user","content":"hello"}]}`)

	responsesBody, originalModel, mappedModel, _, err := buildGrokMessagesCompatResponsesBody(newGrokAPIKeyCompatAccount(), body)

	require.NoError(t, err)
	require.Equal(t, "grok-4", originalModel)
	require.Equal(t, "grok-4-upstream", mappedModel)
	require.Equal(t, "pc-msg-1", gjson.GetBytes(responsesBody, "prompt_cache_key").String())
	require.Equal(t, "grok-4-upstream", gjson.GetBytes(responsesBody, "model").String())
}

func TestGrokForwardAnthropicCountTokensCompatUsesResponsesInputTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","messages":[{"role":"user","content":"hello"}]}`)
	rec, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/messages/count_tokens", body)
	upstream := &countTokensHTTPUpstreamRecorder{httpUpstreamRecorder: httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"usage":{"input_tokens":29}}`)),
	}}}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardAnthropicCountTokensCompat(context.Background(), c, newGrokAPIKeyCompatAccount(), body)

	require.NoError(t, err)
	require.Equal(t, 29, result.InputTokens)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "/v1/responses/input_tokens", upstream.lastReq.URL.Path)
	require.Equal(t, "Bearer xai-test", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "grok-4-upstream", gjson.GetBytes(upstream.lastBody, "model").String())
	require.False(t, gjson.GetBytes(upstream.lastBody, "stream").Exists())
	require.False(t, gjson.GetBytes(upstream.lastBody, "metadata").Exists())
	require.Equal(t, int64(77), upstream.accountID)
	require.Equal(t, 3, upstream.accountConcurrency)
}

func TestGrokForwardAnthropicCountTokensCompatFallsBackWhenInputTokensUnsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","messages":[{"role":"user","content":"hello world"}]}`)
	rec, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/messages/count_tokens", body)
	upstream := &countTokensHTTPUpstreamRecorder{httpUpstreamRecorder: httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"/v1/responses/input_tokens not found"}}`)),
	}}}
	svc := &GrokGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardAnthropicCountTokensCompat(context.Background(), c, newGrokAPIKeyCompatAccount(), body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Estimated)
	require.Equal(t, string(geminiCountTokensSourceEstimated), result.Source)
	require.Greater(t, result.InputTokens, 0)
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, gjson.GetBytes(rec.Body.Bytes(), "estimated").Bool())
	require.Equal(t, string(geminiCountTokensSourceEstimated), gjson.GetBytes(rec.Body.Bytes(), "source").String())
	require.Equal(t, string(geminiCountTokensSourceEstimated), rec.Header().Get(geminiCountTokensSourceHeader))
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "/v1/responses/input_tokens", upstream.lastReq.URL.Path)
}

func TestGrokForwardAnthropicCountTokensCompatRejectsSSOAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-4","messages":[{"role":"user","content":"hello"}]}`)
	rec, c := newCompatGatewayTestContext(http.MethodPost, "/grok/v1/messages/count_tokens", body)
	svc := &GrokGatewayService{
		httpUpstream: &httpUpstreamRecorder{},
		cfg:          &config.Config{},
	}
	account := &Account{
		ID:       78,
		Platform: PlatformGrok,
		Type:     AccountTypeSSO,
		Credentials: map[string]any{
			"sso_token": "sso-test",
		},
	}

	result, err := svc.ForwardAnthropicCountTokensCompat(context.Background(), c, account, body)

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, "not_found_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
}
