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
	require.False(t, gjson.GetBytes(clean, "prompt_cache_key").Exists())
	require.False(t, gjson.GetBytes(clean, "previous_response_id").Exists())
	require.False(t, gjson.GetBytes(clean, "safety_identifier").Exists())
	require.False(t, gjson.GetBytes(clean, "service_tier").Exists())
	require.False(t, gjson.GetBytes(clean, "metadata").Exists())
	require.False(t, gjson.GetBytes(clean, "include").Exists())
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
