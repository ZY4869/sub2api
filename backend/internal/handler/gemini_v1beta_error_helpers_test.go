package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type geminiGoogleErrorPayload struct {
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

func TestGoogleErrorFromServiceErrorUsesKnownReasonMappings(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1beta/files", nil)
	c.Request.Header.Set("Accept-Language", "en")

	googleErrorFromServiceError(c, infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted"))

	var payload geminiGoogleErrorPayload
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, http.StatusServiceUnavailable, payload.Error.Code)
	require.Equal(t, "All accounts in the selected group have been exhausted", payload.Error.Message)
	require.Len(t, payload.Error.Details, 1)
	require.Equal(t, googleRPCTypeErrorInfo, payload.Error.Details[0].Type)
	require.Equal(t, "GROUP_EXHAUSTED", payload.Error.Details[0].Reason)
}

func TestGoogleErrorFromServiceErrorUsesGenericFallbackForUnknownReason(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1beta/files", nil)
	c.Request.Header.Set("Accept-Language", "en")

	googleErrorFromServiceError(c, infraerrors.BadRequest("UNMAPPED_REASON", "sensitive upstream detail"))

	var payload geminiGoogleErrorPayload
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "Request failed", payload.Error.Message)
	require.NotContains(t, payload.Error.Message, "sensitive upstream detail")
	require.Len(t, payload.Error.Details, 1)
	require.Equal(t, "UNMAPPED_REASON", payload.Error.Details[0].Reason)
}

func TestGoogleErrorFromServiceErrorDoesNotExposeRawGenericErrors(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1beta/files", nil)
	c.Request.Header.Set("Accept-Language", "en")

	googleErrorFromServiceError(c, errors.New("raw internal detail"))

	var payload geminiGoogleErrorPayload
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, "Request failed", payload.Error.Message)
	require.Empty(t, payload.Error.Details)
}

func TestGoogleErrorBodyTooLargeUsesLocalizedKey(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-pro:generateContent", nil)
	c.Request.Header.Set("Accept-Language", "en")

	googleErrorBodyTooLarge(c, 1024)

	var payload geminiGoogleErrorPayload
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.Equal(t, "Request body too large, limit is 1024B", payload.Error.Message)
}

func TestGoogleErrorPendingRequestsUsesStableMessage(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-pro:generateContent", nil)
	c.Request.Header.Set("Accept-Language", "en")

	googleErrorPendingRequests(c)

	var payload geminiGoogleErrorPayload
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "Too many pending requests, please retry later", payload.Error.Message)
}

func TestGoogleNoAvailableAccountsErrorUsesKnownReasonMapping(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1beta/models/gemini-2.5-pro:generateContent", nil)
	c.Request.Header.Set("Accept-Language", "en")

	googleNoAvailableAccountsError(c, infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted"))

	var payload geminiGoogleErrorPayload
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, "All accounts in the selected group have been exhausted", payload.Error.Message)
	require.Len(t, payload.Error.Details, 1)
	require.Equal(t, "GROUP_EXHAUSTED", payload.Error.Details[0].Reason)
}

func TestGoogleNoAvailableAccountsErrorDoesNotExposeRawDetail(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1beta/models/gemini-2.5-pro:generateContent", nil)
	c.Request.Header.Set("Accept-Language", "en")

	googleNoAvailableAccountsError(c, errors.New("sensitive upstream selection detail"))

	var payload geminiGoogleErrorPayload
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Equal(t, "No available Gemini accounts", payload.Error.Message)
	require.NotContains(t, payload.Error.Message, "sensitive upstream selection detail")
	require.Empty(t, payload.Error.Details)
}
