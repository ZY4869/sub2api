package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiV1BetaModels_BatchGenerateContentDispatchesToBatchRelay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-pro:batchGenerateContent", nil)
	c.Params = gin.Params{{Key: "modelAction", Value: "/gemini-2.5-pro:batchGenerateContent"}}
	groupID := int64(1)
	c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
		ID:      1,
		GroupID: &groupID,
		Group:   &service.Group{Platform: service.PlatformGemini},
	})
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{
		UserID:      1,
		Concurrency: 1,
	})

	h := &GatewayHandler{}
	h.GeminiV1BetaModels(c)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Gemini batch service not configured")
}

func TestGeminiV1BetaModels_AntigravityBatchGenerateContentRejects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/antigravity/v1beta/models/gemini-2.5-pro:batchGenerateContent", nil)
	c.Request.Header.Set("Accept-Language", "en")
	c.Params = gin.Params{{Key: "modelAction", Value: "/gemini-2.5-pro:batchGenerateContent"}}
	groupID := int64(1)
	c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
		ID:      1,
		GroupID: &groupID,
		Group:   &service.Group{Platform: service.PlatformAntigravity},
	})
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{
		UserID:      1,
		Concurrency: 1,
	})
	c.Set(string(servermiddleware.ContextKeyForcePlatform), service.PlatformAntigravity)

	h := &GatewayHandler{}
	h.GeminiV1BetaModels(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)

	var payload struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
			Details []struct {
				Type   string `json:"@type"`
				Reason string `json:"reason"`
			} `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusNotFound, payload.Error.Code)
	require.Equal(t, "NOT_FOUND", payload.Error.Status)
	require.Contains(t, payload.Error.Message, "/v1beta/models/{model}:batchGenerateContent")
	require.Len(t, payload.Error.Details, 1)
	require.Equal(t, "type.googleapis.com/google.rpc.ErrorInfo", payload.Error.Details[0].Type)
	require.Equal(t, service.GatewayReasonPublicEndpointUnsupported, payload.Error.Details[0].Reason)
}
