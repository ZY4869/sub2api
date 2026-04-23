package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestForwardCompatImages_StreamGenerationBridgesResponsesSSE(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"req-image-stream"},
			},
			Body: ioNopCloserString(strings.Join([]string{
				`data: {"type":"response.image_generation_call.partial_image","partial_image_b64":"cGFydGlhbA==","output_format":"png"}`,
				"",
				`data: {"type":"response.completed","response":{"id":"resp_1","status":"completed","model":"gpt-image-2","output":[{"type":"message","role":"assistant","content":[{"type":"output_image","image_url":"data:image/png;base64,ZmLuYWw="}]}],"usage":{"input_tokens":50,"output_tokens":50,"total_tokens":100}}}`,
				"",
			}, "\n")),
		},
	}
	svc := &OpenAIGatewayService{httpUpstream: upstream, toolCorrector: NewCodexToolCorrector()}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-image-2","prompt":"A poster","size":"1024x1024","stream":true,"partial_images":1}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req.WithContext(EnsureRequestMetadata(req.Context()))

	result, err := svc.ForwardCompatImages(c.Request.Context(), c, &Account{
		ID:          11,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
	}, body, "application/json", "generation", "gpt-image-2")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "event: image_generation.partial_image")
	require.Contains(t, rec.Body.String(), `"partial_image_index":0`)
	require.Contains(t, rec.Body.String(), "event: image_generation.completed")
	require.Contains(t, rec.Body.String(), `"b64_json":"ZmLuYWw="`)
	require.Contains(t, rec.Body.String(), `"total_tokens":100`)
	require.Equal(t, 1, result.ImageCount)
	require.Equal(t, OpenAIImageSizeTier1K, result.ImageSize)
}

func TestForwardCompatImages_StreamEditBridgesResponsesSSE(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"x-request-id": []string{"req-image-edit-stream"},
			},
			Body: ioNopCloserString(strings.Join([]string{
				`data: {"type":"response.image_generation_call.partial_image","partial_image_b64":"ZWQtcGFydGlhbA==","output_format":"png"}`,
				"",
				`data: {"type":"response.completed","response":{"id":"resp_2","status":"completed","model":"gpt-image-2","output":[{"type":"message","role":"assistant","content":[{"type":"output_image","image_url":"data:image/png;base64,ZWQtZmluYWw="}]}],"usage":{"input_tokens":12,"output_tokens":6,"total_tokens":18}}}`,
				"",
			}, "\n")),
		},
	}
	svc := &OpenAIGatewayService{httpUpstream: upstream, toolCorrector: NewCodexToolCorrector()}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-image-2","prompt":"Edit this image","images":[{"image_url":"https://example.com/source.png"}],"stream":true}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req.WithContext(EnsureRequestMetadata(req.Context()))

	result, err := svc.ForwardCompatImages(c.Request.Context(), c, &Account{
		ID:          12,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
	}, body, "application/json", "edits", "gpt-image-2")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, rec.Body.String(), "event: image_edit.partial_image")
	require.Contains(t, rec.Body.String(), "event: image_edit.completed")
	require.Contains(t, rec.Body.String(), `"b64_json":"ZWQtZmluYWw="`)
	require.Equal(t, 1, result.ImageCount)
}

func ioNopCloserString(value string) *readCloserString {
	return &readCloserString{Reader: strings.NewReader(value)}
}

type readCloserString struct {
	*strings.Reader
}

func (r *readCloserString) Close() error { return nil }
