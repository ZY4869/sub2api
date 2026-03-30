package service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForwardAIStudioGET_VertexServiceAccountUsesCallableUnionForModels(t *testing.T) {
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusNotFound,
		body:       `{"error":"should not call upstream"}`,
	}
	svc := newTestGeminiCompatService(upstream)
	svc.SetVertexCatalogService(newTestVertexCatalogProvider(&VertexCatalogResult{
		CallableUnion: []VertexCatalogModel{
			{ID: "gemini-3.1-pro-preview", DisplayName: "Gemini 3.1 Pro Preview"},
			{ID: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash"},
		},
	}))
	account := newTestVertexServiceAccountAccount("global")
	account.ID = 401

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
	require.Equal(t, []string{"models/gemini-2.5-flash", "models/gemini-3.1-pro-preview"}, []string{
		payload.Models[0].Name,
		payload.Models[1].Name,
	})
}

func TestForwardAIStudioGET_VertexServiceAccountModelDetailRequiresCallableRealID(t *testing.T) {
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusNotFound,
		body:       `{"error":"should not call upstream"}`,
	}
	svc := newTestGeminiCompatService(upstream)
	svc.SetVertexCatalogService(newTestVertexCatalogProvider(&VertexCatalogResult{
		CallableUnion: []VertexCatalogModel{
			{ID: "gemini-3.1-flash-image-preview", DisplayName: "Gemini 3.1 Flash Image Preview"},
		},
	}))
	account := newTestVertexServiceAccountAccount("us-central1")
	account.ID = 402

	successResult, err := svc.ForwardAIStudioGET(context.Background(), account, "/v1beta/models/gemini-3.1-flash-image-preview")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, successResult.StatusCode)

	notFoundResult, err := svc.ForwardAIStudioGET(context.Background(), account, "/v1beta/models/gemini-3.1-flash-image")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, notFoundResult.StatusCode)
	require.Nil(t, upstream.lastReq)
}
