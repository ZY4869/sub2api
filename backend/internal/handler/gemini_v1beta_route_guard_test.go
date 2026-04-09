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

func TestGeminiV1BetaModels_ModelActionParseFailuresReturnRouteMismatch(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		rawPath     string
		modelAction string
		wantKey     string
	}{
		{
			name:        "missing path",
			rawPath:     "/v1beta/models/",
			modelAction: "/",
			wantKey:     "gateway.gemini.model_action_path_missing",
		},
		{
			name:        "invalid path",
			rawPath:     "/v1beta/models/gemini-2.5-pro",
			modelAction: "/gemini-2.5-pro",
			wantKey:     "gateway.gemini.model_action_path_invalid",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, tt.rawPath, nil)
			c.Request.Header.Set("Accept-Language", "en")
			c.Params = gin.Params{{Key: "modelAction", Value: tt.modelAction}}

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
			require.NotEmpty(t, payload.Error.Message)
			require.Len(t, payload.Error.Details, 1)
			require.Equal(t, googleRPCTypeErrorInfo, payload.Error.Details[0].Type)
			require.Equal(t, service.GatewayReasonRouteMismatch, payload.Error.Details[0].Reason)
		})
	}
}
