package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractGeminiPassthroughResourceName_OpenAICompat(t *testing.T) {
	require.Equal(t, "file_123", extractGeminiPassthroughResourceName(UpstreamResourceKindGeminiFile, "/v1beta/openai/files/file_123"))
	require.Equal(t, "batch_123", extractGeminiPassthroughResourceName(UpstreamResourceKindGeminiBatch, "/v1beta/openai/batches/batch_123/cancel?foo=bar"))
}

func TestExtractOpenAICompatObjectIDs(t *testing.T) {
	topLevel := extractOpenAICompatObjectIDs([]byte(`{"id":"file_123","object":"file"}`))
	require.Equal(t, []string{"file_123"}, topLevel)

	listPayload := []byte(`{"object":"list","data":[{"id":"file_123"},{"id":"file_456"},{"id":"file_123"}]}`)
	require.ElementsMatch(t, []string{"file_123", "file_456"}, extractOpenAICompatObjectIDs(listPayload))
}

func TestBuildGeminiPassthroughRequestUsesUpstreamPathOverride(t *testing.T) {
	svc := &GeminiMessagesCompatService{}
	account := &Account{
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "gemini-test-key",
			"base_url": "https://generativelanguage.googleapis.com",
		},
	}
	input := GeminiPublicPassthroughInput{
		GoogleBatchForwardInput: GoogleBatchForwardInput{
			Method:        http.MethodPost,
			Path:          "/v1beta/live/auth-token",
			RawQuery:      "alt=sse",
			Headers:       http.Header{"X-Test": []string{"1"}},
			Body:          []byte(`{}`),
			ContentLength: 2,
		},
		UpstreamPath: GeminiLiveAuthTokensPath,
	}

	req, proxyURL, fullURL, err := svc.buildGeminiPassthroughRequest(context.Background(), input, account)

	require.NoError(t, err)
	require.Equal(t, "", proxyURL)
	require.Equal(t, "https://generativelanguage.googleapis.com/v1alpha/authTokens?alt=sse", fullURL)
	require.Equal(t, fullURL, req.URL.String())
	require.Equal(t, "gemini-test-key", req.Header.Get("x-goog-api-key"))
	require.Equal(t, "1", req.Header.Get("X-Test"))
}
