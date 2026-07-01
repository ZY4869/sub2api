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

type countTokensHTTPUpstreamRecorder struct {
	httpUpstreamRecorder
	accountID          int64
	accountConcurrency int
}

func (u *countTokensHTTPUpstreamRecorder) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	u.accountID = accountID
	u.accountConcurrency = accountConcurrency
	return u.httpUpstreamRecorder.Do(req, proxyURL, accountID, accountConcurrency)
}

func (u *countTokensHTTPUpstreamRecorder) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, tlsProfile *TLSFingerprintProfile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func TestForwardAnthropicCountTokensCompatUsesResponsesInputTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-5.4","max_tokens":64,"messages":[{"role":"user","content":"hello"}],"stream":true}`)
	rec, c := newCompatGatewayTestContext(http.MethodPost, "/v1/messages/count_tokens", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"input_tokens":17}`)),
	}}
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardAnthropicCountTokensCompat(context.Background(), c, newCompatForwardAccount(), body, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 17, result.InputTokens)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"input_tokens":17}`, rec.Body.String())
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "/v1/responses/input_tokens", upstream.lastReq.URL.Path)
	require.Equal(t, "gpt-5.4", gjson.GetBytes(upstream.lastBody, "model").String())
	require.False(t, gjson.GetBytes(upstream.lastBody, "stream").Exists())
	require.False(t, gjson.GetBytes(upstream.lastBody, "max_output_tokens").Exists())
}

func TestForwardAnthropicCountTokensCompatRejectsInvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-5.4",`)
	rec, c := newCompatGatewayTestContext(http.MethodPost, "/v1/messages/count_tokens", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"input_tokens":17}`)),
	}}
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardAnthropicCountTokensCompat(context.Background(), c, newCompatForwardAccount(), body, "")

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_request_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	require.Nil(t, upstream.lastReq)
}

func TestForwardAnthropicCountTokensCompatMapsUpstream429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	rec, c := newCompatGatewayTestContext(http.MethodPost, "/v1/messages/count_tokens", body)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"too many requests"}}`)),
	}}
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardAnthropicCountTokensCompat(context.Background(), c, newCompatForwardAccount(), body, "")

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "upstream_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	require.Equal(t, "Rate limit exceeded", gjson.GetBytes(rec.Body.Bytes(), "error.message").String())
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "/v1/responses/input_tokens", upstream.lastReq.URL.Path)
}

func TestReadResponsesInputTokensResultFallbacks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "usage input tokens", body: `{"usage":{"input_tokens":23}}`, want: 23},
		{name: "total tokens", body: `{"total_tokens":31}`, want: 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}

			result, err := readResponsesInputTokensResult(resp, nil, 1<<20, func(status int, errType string, message string) {
				t.Fatalf("unexpected writeError: status=%d type=%s message=%s", status, errType, message)
			})

			require.NoError(t, err)
			require.Equal(t, tt.want, result.InputTokens)
		})
	}
}

func TestForwardAnthropicCountTokensCompatPassesAccountRuntimeMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	rec, c := newCompatGatewayTestContext(http.MethodPost, "/v1/messages/count_tokens", body)
	upstream := &countTokensHTTPUpstreamRecorder{httpUpstreamRecorder: httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"input_tokens":19}`)),
	}}}
	account := newCompatForwardAccount()
	account.ID = 42
	account.Concurrency = 7
	svc := &OpenAIGatewayService{
		httpUpstream: upstream,
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardAnthropicCountTokensCompat(context.Background(), c, account, body, "")

	require.NoError(t, err)
	require.Equal(t, 19, result.InputTokens)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), upstream.accountID)
	require.Equal(t, 7, upstream.accountConcurrency)
}

func TestSanitizeOpenAIResponsesInputTokensBodyDropsGenerationOnlyFields(t *testing.T) {
	body := sanitizeOpenAIResponsesInputTokensBody([]byte(`{"model":"gpt-5.4","input":"hello","stream":true,"store":true,"max_output_tokens":100,"prompt_cache_key":"pc","previous_response_id":"resp_1","metadata":{"a":"b"}}`))

	require.Equal(t, "gpt-5.4", gjson.GetBytes(body, "model").String())
	require.Equal(t, "hello", gjson.GetBytes(body, "input").String())
	require.False(t, gjson.GetBytes(body, "stream").Exists())
	require.False(t, gjson.GetBytes(body, "store").Exists())
	require.False(t, gjson.GetBytes(body, "max_output_tokens").Exists())
	require.False(t, gjson.GetBytes(body, "prompt_cache_key").Exists())
	require.False(t, gjson.GetBytes(body, "previous_response_id").Exists())
	require.False(t, gjson.GetBytes(body, "metadata").Exists())
}
