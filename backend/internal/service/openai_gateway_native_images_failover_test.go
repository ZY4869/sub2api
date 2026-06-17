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

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

func TestForwardNativeImagesGeneration_TransportErrorReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-image-2","prompt":"hi","size":"1024x1024"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	svc := &OpenAIGatewayService{
		httpUpstream: &httpUpstreamRecorder{err: errors.New("proxy connect failed")},
		cfg:          &config.Config{},
	}

	result, err := svc.ForwardNativeImagesGeneration(context.Background(), c, openAINativeImagesFailoverAccount(), body)
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.True(t, failoverErr.TempUnscheduleAccount)
	require.False(t, failoverErr.RetryableOnSameAccount)
	require.Empty(t, recorder.Body.String())
	require.Empty(t, recorder.Result().Header.Get("Content-Type"))
}

func TestForwardNativeImagesGeneration_NonJSONSuccessReturnsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-image-2","prompt":"hi","size":"1024x1024"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	svc := &OpenAIGatewayService{
		httpUpstream: &httpUpstreamRecorder{resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/html"}, "x-request-id": []string{"req-img-non-json"}},
			Body:       io.NopCloser(strings.NewReader("<html>temporarily unavailable</html>")),
		}},
		cfg: &config.Config{},
	}

	result, err := svc.ForwardNativeImagesGeneration(context.Background(), c, openAINativeImagesFailoverAccount(), body)
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusBadGateway, failoverErr.StatusCode)
	require.True(t, failoverErr.RetryableOnSameAccount)
	require.False(t, failoverErr.TempUnscheduleAccount)
	require.Contains(t, string(failoverErr.ResponseBody), "non-JSON")
	require.Empty(t, recorder.Body.String())
	require.Empty(t, recorder.Result().Header.Get("Content-Type"))
}

func TestForwardNativeImagesGeneration_ZstdErrorResponseReturnsFailoverBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body := []byte(`{"model":"gpt-image-2","prompt":"hi","size":"1024x1024"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	upstreamBody := []byte(`{"error":{"type":"server_error","message":"temporary image backend failure"}}`)
	svc := &OpenAIGatewayService{
		httpUpstream: &httpUpstreamRecorder{resp: &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header: http.Header{
				"Content-Type":     []string{"application/json"},
				"Content-Encoding": []string{"zstd"},
				"x-request-id":     []string{"req-img-zstd"},
			},
			Body: io.NopCloser(bytes.NewReader(mustZstdEncodeForTest(t, upstreamBody))),
		}},
		cfg: &config.Config{},
	}

	result, err := svc.ForwardNativeImagesGeneration(context.Background(), c, openAINativeImagesFailoverAccount(), body)
	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusServiceUnavailable, failoverErr.StatusCode)
	require.False(t, failoverErr.RetryableOnSameAccount)
	require.JSONEq(t, string(upstreamBody), string(failoverErr.ResponseBody))
	require.Empty(t, recorder.Body.String())
	require.Empty(t, recorder.Result().Header.Get("Content-Type"))
}

func openAINativeImagesFailoverAccount() *Account {
	return &Account{
		ID:          901,
		Name:        "openai-native-images",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://api.openai.com",
		},
	}
}

func mustZstdEncodeForTest(t *testing.T, body []byte) []byte {
	t.Helper()
	var encoded bytes.Buffer
	zw, err := zstd.NewWriter(&encoded)
	require.NoError(t, err)
	_, err = zw.Write(body)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return encoded.Bytes()
}
