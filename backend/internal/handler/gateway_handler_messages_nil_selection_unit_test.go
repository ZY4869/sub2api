//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandlerSelectionAccountOrFail_NilSelectionReturnsControlled502(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &GatewayHandler{}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	account, ok := h.selectionAccountOrFail(c, nil, false)
	require.False(t, ok)
	require.Nil(t, account)
	require.Equal(t, http.StatusBadGateway, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	errorPayload, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "api_error", errorPayload["type"])
	require.Equal(t, "No available accounts", errorPayload["message"])
}

func TestGatewayHandlerSelectionAccountOrFail_NilAccountReturnsControlled502(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &GatewayHandler{}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	account, ok := h.selectionAccountOrFail(c, &service.AccountSelectionResult{}, false)
	require.False(t, ok)
	require.Nil(t, account)
	require.Equal(t, http.StatusBadGateway, rec.Code)
}
