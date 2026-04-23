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
)

func TestForwardNativeImagesGeneration_StreamPassthroughCountsCompletedImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	body := []byte(`{"model":"gpt-image-2","prompt":"hi","size":"1024x1024","stream":true}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	resp := newJSONResponse(http.StatusOK, "")
	resp.Header.Set("Content-Type", "text/event-stream")
	resp.Header.Set("x-request-id", "req-native-images-stream")
	resp.Body = io.NopCloser(strings.NewReader(
		"data: {\"type\":\"image_generation.partial_image\",\"partial_image\":\"AAAA\",\"partial_image_index\":0}\n\n" +
			"data: {\"type\":\"image_generation.completed\",\"b64_json\":\"QUJD\"}\n\n" +
			"data: {\"type\":\"image_generation.completed\",\"url\":\"https://example.com/img.png\"}\n\n" +
			"data: [DONE]\n\n",
	))

	upstream := &queuedHTTPUpstream{responses: []*http.Response{resp}}
	svc := &OpenAIGatewayService{httpUpstream: upstream, cfg: &config.Config{}}
	account := &Account{
		ID:          101,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "test-token",
			"base_url": "https://api.openai.com",
		},
	}

	result, err := svc.ForwardNativeImagesGeneration(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.Equal(t, 2, result.ImageCount)
	require.Equal(t, "image", result.MediaType)
	require.Equal(t, OpenAIImageSizeTier1K, result.ImageSize)
	require.Equal(t, "/v1/images/generations", upstream.requests[0].URL.Path)
	require.Equal(t, "text/event-stream", upstream.requests[0].Header.Get("Accept"))
	require.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	require.Contains(t, recorder.Body.String(), "data: {\"type\":\"image_generation.completed\"")
}
