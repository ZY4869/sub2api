package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPaginateGeminiPublicModels(t *testing.T) {
	t.Parallel()

	entries := []service.APIKeyPublicModelEntry{
		{PublicID: "gemini-2.5-pro"},
		{PublicID: "gemini-2.5-flash"},
		{PublicID: "gemini-2.5-flash-lite"},
		{PublicID: "gemini-2.5-flash-image"},
	}

	page, nextPageToken, err := paginateGeminiPublicModels(entries, "2", "1")
	require.NoError(t, err)
	require.Len(t, page, 2)
	require.Equal(t, "gemini-2.5-flash", page[0].PublicID)
	require.Equal(t, "gemini-2.5-flash-lite", page[1].PublicID)
	require.Equal(t, encodeGeminiFallbackPageToken(3), nextPageToken)
}

func TestPaginateGeminiPublicModels_InvalidPagination(t *testing.T) {
	t.Parallel()

	_, _, err := paginateGeminiPublicModels([]service.APIKeyPublicModelEntry{{PublicID: "gemini-2.5-pro"}}, "bad", "")
	require.Error(t, err)

	_, _, err = paginateGeminiPublicModels([]service.APIKeyPublicModelEntry{{PublicID: "gemini-2.5-pro"}}, "1", "bad")
	require.Error(t, err)
}

func TestParseGeminiModelsPageToken_SupportsOpaqueAndLegacyTokens(t *testing.T) {
	t.Parallel()

	offset, err := parseGeminiModelsPageToken(encodeGeminiFallbackPageToken(7))
	require.NoError(t, err)
	require.Equal(t, 7, offset)

	offset, err = parseGeminiModelsPageToken("5")
	require.NoError(t, err)
	require.Equal(t, 5, offset)
}

func TestAPIKeyPublicEntriesToGeminiModels_IncludesNextPageToken(t *testing.T) {
	t.Parallel()

	nextPageToken := encodeGeminiFallbackPageToken(2)
	resp := apiKeyPublicEntriesToGeminiModels([]service.APIKeyPublicModelEntry{{PublicID: "gemini-2.5-pro", Platform: service.PlatformGemini}}, nextPageToken)
	require.Len(t, resp.Models, 1)
	require.Equal(t, nextPageToken, resp.NextPageToken)
	require.Equal(t, "models/gemini-2.5-pro", resp.Models[0].Name)
	require.Equal(t, "gemini-2.5-pro", resp.Models[0].DisplayName)
	require.NotEmpty(t, resp.Models[0].Description)
	require.Empty(t, resp.Models[0].BaseModelID)
	require.Empty(t, resp.Models[0].Version)
	require.Zero(t, resp.Models[0].InputTokenLimit)
	require.Zero(t, resp.Models[0].OutputTokenLimit)
	require.Empty(t, resp.Models[0].SupportedGenerationMethods)
}

func TestGatewayModels_VertexCatalogFailureFallsBackToRegistry(t *testing.T) {
	t.Parallel()

	handler, apiKey := newGeminiPublicModelsFallbackHandler(t)
	c, recorder := newGeminiPublicModelsContext(http.MethodGet, "/v1/models", apiKey, nil)

	handler.Models(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var payload struct {
		Object string         `json:"object"`
		Data   []claude.Model `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "list", payload.Object)
	require.NotEmpty(t, payload.Data)

	found := false
	for _, model := range payload.Data {
		if model.ID != "gemini-2.0-flash" {
			continue
		}
		found = true
		require.NotEmpty(t, model.DisplayName)
	}
	require.True(t, found)
}

func TestGatewayModels_ExplicitAliasOnlySurfaceDoesNotLeakSourceModel(t *testing.T) {
	t.Parallel()

	handler, apiKey := newGeminiPublicModelsAliasHandler(t)
	c, recorder := newGeminiPublicModelsContext(http.MethodGet, "/v1/models", apiKey, nil)

	handler.Models(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var payload struct {
		Object string         `json:"object"`
		Data   []claude.Model `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "list", payload.Object)
	require.Len(t, payload.Data, 1)
	require.Equal(t, "friendly-flash", payload.Data[0].ID)
	require.Equal(t, "friendly-flash", payload.Data[0].DisplayName)
	require.NotContains(t, recorder.Body.String(), "gemini-2.0-flash")
}
