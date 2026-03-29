package service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForwardAIStudioGET_VertexServiceAccountUsesLocalCatalogForModels(t *testing.T) {
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusNotFound,
		body:       `{"error":"should not call upstream"}`,
	}
	svc := newTestGeminiCompatService(upstream)
	account := &Account{
		ID:       401,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"oauth_type":        "vertex_ai",
			"vertex_project_id": "vertex-project",
			"vertex_location":   "global",
		},
	}

	result, err := svc.ForwardAIStudioGET(context.Background(), account, "/v1beta/models")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Nil(t, upstream.lastReq)

	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	require.NoError(t, json.Unmarshal(result.Body, &payload))
	require.NotEmpty(t, payload.Models)
	require.Equal(t, "models/"+GeminiVertexCatalogModelIDs()[0], payload.Models[0].Name)
}

func TestForwardAIStudioGET_VertexServiceAccountUsesLocalCatalogForModelDetail(t *testing.T) {
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusNotFound,
		body:       `{"error":"should not call upstream"}`,
	}
	svc := newTestGeminiCompatService(upstream)
	account := &Account{
		ID:       402,
		Platform: PlatformGemini,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"oauth_type":        "vertex_ai",
			"vertex_project_id": "vertex-project",
			"vertex_location":   "us-central1",
		},
	}

	modelID := GeminiVertexCatalogModelIDs()[0]
	result, err := svc.ForwardAIStudioGET(context.Background(), account, "/v1beta/models/"+modelID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Nil(t, upstream.lastReq)

	var payload struct {
		Name string `json:"name"`
	}
	require.NoError(t, json.Unmarshal(result.Body, &payload))
	require.Equal(t, "models/"+modelID, payload.Name)
}
