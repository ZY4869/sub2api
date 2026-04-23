//go:build unit

package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIStreamingResponsesToolSetsImageOutputCountMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &OpenAIGatewayService{cfg: &config.Config{}}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	mdCtx := EnsureRequestMetadata(req.Context())
	SetImageRequestSurfaceMetadata(mdCtx, "responses_tool")
	req = req.WithContext(mdCtx)
	c.Request = req

	sse := "" +
		"data: {\"type\":\"response.image_generation_call.partial_image\",\"output_index\":0,\"partial_image\":\"AAA\"}\n\n" +
		"data: {\"type\":\"response.image_generation_call.partial_image\",\"output_index\":0,\"partial_image\":\"AAA\"}\n\n" +
		"data: {\"type\":\"response.image_generation_call.partial_image\",\"output_index\":1,\"partial_image\":\"BBB\"}\n\n" +
		"data: {\"type\":\"response.done\",\"response\":{}}\n\n"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(sse)),
		Header:     http.Header{},
	}

	result, err := svc.handleStreamingResponse(c.Request.Context(), resp, c, &Account{ID: 1}, time.Now(), "model", "model")
	require.NoError(t, err)
	require.NotNil(t, result)

	count, ok := ImageOutputCountMetadataFromContext(c.Request.Context())
	require.True(t, ok)
	require.Equal(t, 2, count)
}
