//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeminiV1BetaOpenAICompat_UsesCompatSurfaceRecorder(t *testing.T) {
	fixture := newGeminiSurfaceFixture(t)
	fixture.compatRecorder.response = geminiSurfaceHTTPResponse{
		statusCode: http.StatusOK,
		body:       `{"id":"chatcmpl_gemini","object":"chat.completion"}`,
	}

	c, recorder := fixture.newContext(
		http.MethodPost,
		"/v1beta/openai/chat/completions",
		`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hello"}]}`,
		nil,
	)

	fixture.handler.GeminiV1BetaOpenAICompat(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	fixture.requireOnlyRecorderHit(fixture.compatRecorder)
	require.NotNil(t, fixture.compatRecorder.lastReq)
	require.Equal(t, "/v1beta/openai/chat/completions", fixture.compatRecorder.lastReq.URL.Path)
	require.Equal(t, "gemini-test-key", fixture.compatRecorder.lastReq.Header.Get("x-goog-api-key"))
	require.JSONEq(t, `{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hello"}]}`, string(fixture.compatRecorder.lastBody))
}

func TestGeminiV1BetaOpenAICompat_FailureUsesCompatSurfaceRecorder(t *testing.T) {
	fixture := newGeminiSurfaceFixture(t)
	fixture.compatRecorder.response = geminiSurfaceHTTPResponse{
		statusCode: http.StatusInternalServerError,
		body:       `{"error":{"message":"compat upstream failed"}}`,
	}

	c, recorder := fixture.newContext(
		http.MethodPost,
		"/v1beta/openai/chat/completions",
		`{"model":"gemini-2.5-flash","messages":[{"role":"user","content":"hello"}]}`,
		nil,
	)

	fixture.handler.GeminiV1BetaOpenAICompat(c)

	assertGeminiPassthroughFailure(t, recorder, fixture.compatRecorder, fixture, "GEMINI_PASSTHROUGH_UPSTREAM_ERROR")
}

func TestGeminiV1BetaLive_AuthTokenUsesLiveSurfaceRecorder(t *testing.T) {
	fixture := newGeminiSurfaceFixture(t)
	fixture.liveRecorder.response = geminiSurfaceHTTPResponse{
		statusCode: http.StatusOK,
		body:       `{"name":"authTokens/test-token"}`,
	}

	c, recorder := fixture.newContext(
		http.MethodPost,
		"/v1beta/live/auth-token",
		`{"ttl":60}`,
		nil,
	)

	fixture.handler.GeminiV1BetaLive(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	fixture.requireOnlyRecorderHit(fixture.liveRecorder)
	require.NotNil(t, fixture.liveRecorder.lastReq)
	require.Equal(t, "/v1alpha/authTokens", fixture.liveRecorder.lastReq.URL.Path)
	require.Equal(t, "gemini-test-key", fixture.liveRecorder.lastReq.Header.Get("x-goog-api-key"))
	require.JSONEq(t, `{"ttl":60}`, string(fixture.liveRecorder.lastBody))
}

func TestGeminiV1BetaLive_AuthTokenFailureUsesLiveSurfaceRecorder(t *testing.T) {
	fixture := newGeminiSurfaceFixture(t)
	fixture.liveRecorder.response = geminiSurfaceHTTPResponse{
		statusCode: http.StatusInternalServerError,
		body:       `{"error":{"message":"live auth failed"}}`,
	}

	c, recorder := fixture.newContext(
		http.MethodPost,
		"/v1beta/live/auth-token",
		`{"ttl":60}`,
		nil,
	)

	fixture.handler.GeminiV1BetaLive(c)

	assertGeminiPassthroughFailure(t, recorder, fixture.liveRecorder, fixture, "GEMINI_PASSTHROUGH_UPSTREAM_ERROR")
}

func TestGeminiV1BetaInteractions_UsesInteractionsSurfaceRecorder(t *testing.T) {
	fixture := newGeminiSurfaceFixture(t)
	fixture.interactionsRecorder.response = geminiSurfaceHTTPResponse{
		statusCode: http.StatusOK,
		body:       `{"id":"interaction_123","state":"completed"}`,
	}

	c, recorder := fixture.newContext(
		http.MethodPost,
		"/v1beta/interactions",
		`{"model":"gemini-2.5-flash","input":{"text":"hello"}}`,
		nil,
	)

	fixture.handler.GeminiV1BetaInteractions(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	fixture.requireOnlyRecorderHit(fixture.interactionsRecorder)
	require.NotNil(t, fixture.interactionsRecorder.lastReq)
	require.Equal(t, "/v1beta/interactions", fixture.interactionsRecorder.lastReq.URL.Path)
	require.Equal(t, "gemini-test-key", fixture.interactionsRecorder.lastReq.Header.Get("x-goog-api-key"))
	require.JSONEq(t, `{"model":"gemini-2.5-flash","input":{"text":"hello"}}`, string(fixture.interactionsRecorder.lastBody))
}

func TestGeminiV1BetaInteractions_FailureUsesInteractionsSurfaceRecorder(t *testing.T) {
	fixture := newGeminiSurfaceFixture(t)
	fixture.interactionsRecorder.response = geminiSurfaceHTTPResponse{
		statusCode: http.StatusInternalServerError,
		body:       `{"error":{"message":"interactions upstream failed"}}`,
	}

	c, recorder := fixture.newContext(
		http.MethodPost,
		"/v1beta/interactions",
		`{"model":"gemini-2.5-flash","input":{"text":"hello"}}`,
		nil,
	)

	fixture.handler.GeminiV1BetaInteractions(c)

	assertGeminiPassthroughFailure(t, recorder, fixture.interactionsRecorder, fixture, "GEMINI_PASSTHROUGH_UPSTREAM_ERROR")
}

func assertGeminiPassthroughFailure(t *testing.T, recorder *httptest.ResponseRecorder, expected *geminiSurfaceHTTPUpstreamRecorder, fixture *geminiSurfaceFixture, wantReason string) {
	t.Helper()

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	fixture.requireOnlyRecorderHit(expected)

	var payload geminiSurfaceErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, http.StatusInternalServerError, payload.Error.Code)
	require.Equal(t, "INTERNAL", payload.Error.Status)
	require.Equal(t, "Request failed", payload.Error.Message)
	require.Len(t, payload.Error.Details, 1)
	require.Equal(t, googleRPCTypeErrorInfo, payload.Error.Details[0].Type)
	require.Equal(t, wantReason, payload.Error.Details[0].Reason)
}
