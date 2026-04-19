//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiV1BetaModels_UsesNativeSurfaceRecorder(t *testing.T) {
	fixture := newGeminiSurfaceFixture(t)
	fixture.nativeRecorder.response = geminiSurfaceHTTPResponse{
		statusCode: http.StatusOK,
		body:       `{"responseId":"native-resp-1","candidates":[{"content":{"role":"model","parts":[{"text":"hello from native"}]}}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":5,"totalTokenCount":8}}`,
	}

	c, recorder := fixture.newContext(
		http.MethodPost,
		"/v1beta/models/gemini-2.5-flash:generateContent",
		`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`,
		gin.Params{{Key: "modelAction", Value: "/gemini-2.5-flash:generateContent"}},
	)

	fixture.handler.GeminiV1BetaModels(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.GreaterOrEqual(t, fixture.accountRepo.listByGroupAndPlatformsCalls, 1)
	require.Equal(t, *fixture.apiKey.GroupID, fixture.accountRepo.lastListGroupID)
	require.Contains(t, fixture.accountRepo.lastListPlatforms, "gemini")

	fixture.requireOnlyRecorderHit(fixture.nativeRecorder)
	require.NotNil(t, fixture.nativeRecorder.lastReq)
	require.Equal(t, http.MethodPost, fixture.nativeRecorder.lastReq.Method)
	require.Equal(t, "/v1beta/models/gemini-2.5-flash:generateContent", fixture.nativeRecorder.lastReq.URL.Path)
	require.Equal(t, "gemini-test-key", fixture.nativeRecorder.lastReq.Header.Get("x-goog-api-key"))
	require.JSONEq(t, `{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`, string(fixture.nativeRecorder.lastBody))
	require.JSONEq(t, fixture.nativeRecorder.response.body, recorder.Body.String())
}

func TestGeminiV1BetaModels_FailoverExhaustedMapsGoogleError(t *testing.T) {
	fixture := newGeminiSurfaceFixture(t)
	fixture.handler.maxAccountSwitchesGemini = 0
	fixture.nativeRecorder.response = geminiSurfaceHTTPResponse{
		statusCode: http.StatusForbidden,
		body:       `{"error":{"message":"native upstream forbidden"}}`,
	}

	c, recorder := fixture.newContext(
		http.MethodPost,
		"/v1beta/models/gemini-2.5-flash:generateContent",
		`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`,
		gin.Params{{Key: "modelAction", Value: "/gemini-2.5-flash:generateContent"}},
	)

	fixture.handler.GeminiV1BetaModels(c)

	require.Equal(t, http.StatusBadGateway, recorder.Code)
	fixture.requireOnlyRecorderHit(fixture.nativeRecorder)

	var payload geminiSurfaceErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusBadGateway, payload.Error.Code)
	require.Equal(t, "INTERNAL", payload.Error.Status)
	require.True(t, strings.Contains(payload.Error.Message, "Upstream access forbidden"))
	require.Empty(t, payload.Error.Details)
}
