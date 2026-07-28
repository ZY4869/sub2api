package service

import (
	"context"
	"net/http"
	"strings"
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

func TestGeminiPassthroughAntigravityForceRequiresAPIKeyBaseURL(t *testing.T) {
	input := GeminiPublicPassthroughInput{
		ForcedPlatform:        PlatformAntigravity,
		RequiresAPIKeyAccount: true,
	}

	require.False(t, geminiPassthroughEligibleAccount(&Account{
		Platform: PlatformAntigravity,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "oauth-token",
			"base_url":     "https://antigravity.example.test/antigravity",
		},
	}, input), "OpenAI-compatible native passthrough must not select Antigravity OAuth accounts")

	require.False(t, geminiPassthroughEligibleAccount(&Account{
		Platform: PlatformAntigravity,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "antigravity-key",
		},
	}, input), "Antigravity OpenAI-compatible passthrough needs an explicit Antigravity base_url")

	require.True(t, geminiPassthroughEligibleAccount(&Account{
		Platform: PlatformAntigravity,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "antigravity-key",
			"base_url": "https://antigravity.example.test/antigravity",
		},
	}, input))
}

func TestBuildGeminiPassthroughRequestAntigravityStripsLocalPrefix(t *testing.T) {
	svc := &GeminiMessagesCompatService{}
	account := &Account{
		Type:     AccountTypeAPIKey,
		Platform: PlatformAntigravity,
		Credentials: map[string]any{
			"api_key":  "antigravity-test-key",
			"base_url": "https://antigravity.example.test/antigravity",
		},
	}
	input := GeminiPublicPassthroughInput{
		GoogleBatchForwardInput: GoogleBatchForwardInput{
			Method:        http.MethodPost,
			Path:          "/antigravity/v1beta/openai/chat/completions",
			Headers:       http.Header{"X-Test": []string{"1"}},
			Body:          []byte(`{"model":"gemini-3.6-flash","messages":[]}`),
			ContentLength: int64(len(`{"model":"gemini-3.6-flash","messages":[]}`)),
		},
		ForcedPlatform:        PlatformAntigravity,
		RequiresAPIKeyAccount: true,
	}

	req, _, fullURL, err := svc.buildGeminiPassthroughRequest(context.Background(), input, account)

	require.NoError(t, err)
	require.Equal(t, "https://antigravity.example.test/antigravity/v1beta/openai/chat/completions", fullURL)
	require.Equal(t, fullURL, req.URL.String())
	require.Equal(t, "antigravity-test-key", req.Header.Get("x-goog-api-key"))
	require.Equal(t, 1, strings.Count(req.URL.Path, "/antigravity"))
}

func TestOpenAICompatUsageOnlyNonStreamResponseGuard(t *testing.T) {
	input := GeminiPublicPassthroughInput{
		GoogleBatchForwardInput: GoogleBatchForwardInput{
			Path: "/antigravity/v1beta/openai/chat/completions",
			Body: []byte(`{"model":"gemini-3.6-flash","stream":false}`),
		},
	}

	require.True(t, isOpenAICompatUsageOnlyNonStreamResponse(input, []byte(`{"usage":{"prompt_tokens":1,"completion_tokens":0}}`)))
	require.False(t, isOpenAICompatUsageOnlyNonStreamResponse(input, []byte(`{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":1}}`)))

	input.Body = []byte(`{"model":"gemini-3.6-flash","stream":true}`)
	require.False(t, isOpenAICompatUsageOnlyNonStreamResponse(input, []byte(`{"usage":{"prompt_tokens":1}}`)))
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

func TestExtractGeminiPassthroughResourceName_NewGeminiResources(t *testing.T) {
	require.Equal(t, "fileSearchStores/default-store", extractGeminiPassthroughResourceName(UpstreamResourceKindGeminiFileSearchStore, "/v1beta/fileSearchStores/default-store:importFile"))
	require.Equal(t, "corpora/sample-corpus/operations/op-1", extractGeminiPassthroughResourceName(UpstreamResourceKindGeminiCorpusOperation, "/v1beta/corpora/sample-corpus/operations/op-1"))
	require.Equal(t, "corpora/sample-corpus/permissions/perm-1", extractGeminiPassthroughResourceName(UpstreamResourceKindGeminiCorpusPermission, "/v1beta/corpora/sample-corpus/permissions/perm-1"))
	require.Equal(t, "generatedFiles/file-1/operations/op-1", extractGeminiPassthroughResourceName(UpstreamResourceKindGeminiGeneratedFileOperation, "/v1beta/generatedFiles/file-1/operations/op-1"))
	require.Equal(t, "models/gemini-2.5-pro/operations/op-9", extractGeminiPassthroughResourceName(UpstreamResourceKindGeminiModelOperation, "/v1beta/models/gemini-2.5-pro/operations/op-9"))
	require.Equal(t, "tunedModels/tuned-1/permissions/perm-1", extractGeminiPassthroughResourceName(UpstreamResourceKindGeminiTunedModelPermission, "/v1beta/tunedModels/tuned-1/permissions/perm-1"))
	require.Equal(t, "tunedModels/tuned-1/operations/op-9", extractGeminiPassthroughResourceName(UpstreamResourceKindGeminiTunedModelOperation, "/v1beta/tunedModels/tuned-1/operations/op-9"))
}

func TestExtractGeminiPassthroughCreatedResourceNames_NewGeminiChildKinds(t *testing.T) {
	require.Equal(t, []string{"generatedFiles/file-1/operations/op-1"}, extractGeminiPassthroughCreatedResourceNames(UpstreamResourceKindGeminiGeneratedFileOperation, []byte(`{"name":"generatedFiles/file-1/operations/op-1"}`)))
	require.Equal(t, []string{"tunedModels/tuned-1/permissions/perm-1"}, extractGeminiPassthroughCreatedResourceNames(UpstreamResourceKindGeminiTunedModelPermission, []byte(`{"name":"tunedModels/tuned-1/permissions/perm-1"}`)))
	require.Equal(t, []string{"tunedModels/tuned-1/operations/op-9"}, extractGeminiPassthroughCreatedResourceNames(UpstreamResourceKindGeminiTunedModelOperation, []byte(`{"name":"tunedModels/tuned-1/operations/op-9"}`)))
	require.Equal(t, []string{"tunedModels/tuned-1/operations/op-async-1"}, extractGeminiPassthroughCreatedResourceNames(UpstreamResourceKindGeminiTunedModelOperation, []byte(`{"name":"tunedModels/tuned-1/operations/op-async-1"}`)))
}
