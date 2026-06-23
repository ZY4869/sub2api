package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAuthSessionCompatReturnsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAuthCompatRoutes(router, &handler.Handlers{
		Auth: &handler.AuthHandler{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.NotContains(t, strings.ToLower(recorder.Body.String()), "<html")

	var payload struct {
		User          any     `json:"user"`
		Expires       *string `json:"expires"`
		Authenticated bool    `json:"authenticated"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Nil(t, payload.User)
	require.Nil(t, payload.Expires)
	require.False(t, payload.Authenticated)
}
