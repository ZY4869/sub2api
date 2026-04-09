package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardAsAnthropic_WritesLocalizedCompatError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.4","max_tokens":128,"system":123,"messages":[{"role":"user","content":"hello"}],"stream":false}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	c.Request = req

	svc := &OpenAIGatewayService{}
	result, err := svc.ForwardAsAnthropic(context.Background(), c, nil, body, "", "")
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.NotEmpty(t, gjson.GetBytes(recorder.Body.Bytes(), "error.message").String())
	require.Equal(t, apicompat.CompatReasonAnthropicSystemInvalid, gjson.GetBytes(recorder.Body.Bytes(), "error.reason").String())
	require.Equal(t, apicompat.CompatReasonAnthropicSystemInvalid, gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
}

func TestForwardAsChatCompletions_WritesLocalizedCompatError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":123}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	c.Request = req

	svc := &OpenAIGatewayService{}
	result, err := svc.ForwardAsChatCompletions(context.Background(), c, nil, body, "", "")
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.NotEmpty(t, gjson.GetBytes(recorder.Body.Bytes(), "error.message").String())
	require.Equal(t, apicompat.CompatReasonChatUserContentInvalid, gjson.GetBytes(recorder.Body.Bytes(), "error.reason").String())
	require.Equal(t, apicompat.CompatReasonChatUserContentInvalid, gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
}

func TestForwardAsAnthropic_WritesRuntimeRegistryMissingCompatError(t *testing.T) {
	withCompatRuntimeRegistryPolicyRemoved(t, compatRuntimePolicyAnthropicMessagesToResponses, func() {
		gin.SetMode(gin.TestMode)

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		body := []byte(`{"model":"gpt-5.4","max_tokens":128,"messages":[{"role":"user","content":"hello"}],"stream":false}`)
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
		req.Header.Set("Accept-Language", "en")
		c.Request = req

		svc := &OpenAIGatewayService{}
		result, err := svc.ForwardAsAnthropic(context.Background(), c, nil, body, "", "")
		require.Error(t, err)
		require.Nil(t, result)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.NotEmpty(t, gjson.GetBytes(recorder.Body.Bytes(), "error.message").String())
		require.Equal(t, apicompat.CompatReasonRuntimeRegistryMissing, gjson.GetBytes(recorder.Body.Bytes(), "error.reason").String())
		require.Equal(t, apicompat.CompatReasonRuntimeRegistryMissing, gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
	})
}

func TestForwardAsChatCompletions_WritesRuntimeRegistryMissingCompatError(t *testing.T) {
	withCompatRuntimeRegistryPolicyRemoved(t, compatRuntimePolicyChatCompletionsToResponses, func() {
		gin.SetMode(gin.TestMode)

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
		req.Header.Set("Accept-Language", "en")
		c.Request = req

		svc := &OpenAIGatewayService{}
		result, err := svc.ForwardAsChatCompletions(context.Background(), c, nil, body, "", "")
		require.Error(t, err)
		require.Nil(t, result)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.NotEmpty(t, gjson.GetBytes(recorder.Body.Bytes(), "error.message").String())
		require.Equal(t, apicompat.CompatReasonRuntimeRegistryMissing, gjson.GetBytes(recorder.Body.Bytes(), "error.reason").String())
		require.Equal(t, apicompat.CompatReasonRuntimeRegistryMissing, gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
	})
}

func TestForwardAsChatCompletions_WritesUpstreamRequestFailureReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`)
	recorder, c := newCompatGatewayTestContext(http.MethodPost, "/v1/chat/completions", body)
	svc := &OpenAIGatewayService{
		httpUpstream: &httpUpstreamRecorder{err: errors.New("dial tcp upstream.internal:443: connect failed")},
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardAsChatCompletions(context.Background(), c, newCompatForwardAccount(), body, "", "")
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Equal(t, apicompat.CompatReasonUpstreamRequestFailed, gjson.GetBytes(recorder.Body.Bytes(), "error.reason").String())
	require.Equal(t, apicompat.CompatReasonUpstreamRequestFailed, gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
}

func TestHandleChatBufferedStreamingResponse_WritesTerminalEventMissingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder, c := newCompatGatewayTestContext(http.MethodPost, "/v1/chat/completions", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"response.output_text.delta\"}\n\ndata: [DONE]\n\n")),
	}

	result, err := (&OpenAIGatewayService{}).handleChatBufferedStreamingResponse(resp, c, "gpt-5.4", "gpt-5.4", time.Now())
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Equal(t, apicompat.CompatReasonUpstreamTerminalEventMissing, gjson.GetBytes(recorder.Body.Bytes(), "error.reason").String())
	require.Equal(t, apicompat.CompatReasonUpstreamTerminalEventMissing, gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
}

func TestForwardAsAnthropic_WritesUpstreamRequestFailureReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"gpt-5.4","max_tokens":32,"messages":[{"role":"user","content":"hello"}],"stream":false}`)
	recorder, c := newCompatGatewayTestContext(http.MethodPost, "/v1/messages", body)
	svc := &OpenAIGatewayService{
		httpUpstream: &httpUpstreamRecorder{err: errors.New("dial tcp upstream.internal:443: connect failed")},
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardAsAnthropic(context.Background(), c, newCompatForwardAccount(), body, "", "")
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Equal(t, apicompat.CompatReasonUpstreamRequestFailed, gjson.GetBytes(recorder.Body.Bytes(), "error.reason").String())
	require.Equal(t, apicompat.CompatReasonUpstreamRequestFailed, gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
}

func TestHandleAnthropicBufferedStreamingResponse_WritesTerminalEventMissingReason(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder, c := newCompatGatewayTestContext(http.MethodPost, "/v1/messages", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"response.output_text.delta\"}\n\ndata: [DONE]\n\n")),
	}

	result, err := (&OpenAIGatewayService{}).handleAnthropicBufferedStreamingResponse(resp, c, "gpt-5.4", "gpt-5.4", time.Now())
	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadGateway, recorder.Code)
	require.Equal(t, apicompat.CompatReasonUpstreamTerminalEventMissing, gjson.GetBytes(recorder.Body.Bytes(), "error.reason").String())
	require.Equal(t, apicompat.CompatReasonUpstreamTerminalEventMissing, gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
}

func withCompatRuntimeRegistryPolicyRemoved(t *testing.T, policyID string, fn func()) {
	t.Helper()

	original := compatRuntimeRegistry
	filtered := make([]CompatRuntimeRegistryEntry, 0, len(original))
	for _, entry := range original {
		if entry.PolicyID == policyID {
			continue
		}
		filtered = append(filtered, entry)
	}
	compatRuntimeRegistry = filtered
	defer func() {
		compatRuntimeRegistry = original
	}()

	fn()
}

func newCompatGatewayTestContext(method string, target string, body []byte) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Accept-Language", "en")
	return recorder, c
}

func newCompatForwardAccount() *Account {
	return &Account{
		ID:          1,
		Name:        "compat-openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
	}
}
