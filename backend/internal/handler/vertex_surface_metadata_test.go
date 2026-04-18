package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestVertexModels_SetsStrictGeminiSurfaceMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/v1/projects/demo-project/locations/us-central1/publishers/google/models/gemini-2.5-pro:generateContent",
		nil,
	)
	c.Params = gin.Params{{Key: "modelAction", Value: "/gemini-2.5-pro:generateContent"}}

	h := &GatewayHandler{}
	h.VertexModels(c)

	surface, ok := service.GeminiSurfaceMetadataFromContext(c.Request.Context())
	require.True(t, ok)
	require.Equal(t, "vertex_strict", surface)
}

func TestVertexBatchPredictionJobs_SetsStrictGeminiSurfaceMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/v1/projects/demo-project/locations/us-central1/batchPredictionJobs",
		nil,
	)

	h := &GatewayHandler{}
	h.VertexBatchPredictionJobs(c)

	surface, ok := service.GeminiSurfaceMetadataFromContext(c.Request.Context())
	require.True(t, ok)
	require.Equal(t, "vertex_strict", surface)
}
