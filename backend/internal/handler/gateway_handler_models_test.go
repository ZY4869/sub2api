package handler

import (
	"testing"

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
