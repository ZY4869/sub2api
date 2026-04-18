package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

// ──────────────────────────────────────────────────────────
// NormalizeInboundEndpoint
// ──────────────────────────────────────────────────────────

func TestNormalizeInboundEndpoint(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		// Direct canonical paths.
		{"/v1/messages", EndpointMessages},
		{"/v1/chat/completions", EndpointChatCompletions},
		{"/v1/responses", EndpointResponses},
		{"/v1/videos", EndpointVideosCreate},
		{"/v1/videos/generations", EndpointVideosCreate},
		{"/v1/models/gemini-2.5-pro:generateContent", EndpointGeminiModels},
		{"/v1alpha/authTokens", EndpointGeminiLiveAuthTokens},
		{"/v1beta/models", EndpointGeminiModels},
		{"/v1beta/models/gemini-2.5-pro:batchGenerateContent", EndpointGeminiBatches},
		{"/v1beta/models/gemini-2.5-pro:embedContent", EndpointGeminiEmbeddings},
		{"/v1beta/cachedContents", EndpointGeminiCachedContents},
		{"/v1beta/fileSearchStores", EndpointGeminiFileSearchStores},
		{"/v1beta/fileSearchStores/default-store:importFile", EndpointGeminiFileSearchStores},
		{"/v1beta/fileSearchStores/default/documents/doc-123", EndpointGeminiDocuments},
		{"/v1beta/fileSearchStores/default/operations/op-123", EndpointGeminiOperations},
		{"/v1beta/fileSearchStores/default/upload/operations/op-123", EndpointGeminiUploadOperations},
		{"/v1beta/documents", EndpointGeminiDocuments},
		{"/v1beta/operations/sample", EndpointGeminiOperations},
		{"/v1beta/corpora/default-corpus", EndpointGeminiCorpora},
		{"/v1beta/corpora/default-corpus/operations/op-123", EndpointGeminiCorporaOperations},
		{"/v1beta/corpora/default-corpus/permissions/perm-123", EndpointGeminiCorporaPermissions},
		{"/v1beta/dynamic/session-123:generateContent", EndpointGeminiDynamic},
		{"/v1beta/generatedFiles", EndpointGeminiGeneratedFiles},
		{"/v1beta/generatedFiles/file-123/operations/op-123", EndpointGeminiGeneratedFilesOperations},
		{"/v1beta/models/gemini-2.5-pro/operations", EndpointGeminiModelOperations},
		{"/v1beta/models/gemini-2.5-pro/operations/op-123", EndpointGeminiModelOperations},
		{"/v1beta/models/gemini-2.5-pro:generateAnswer", EndpointGeminiModels},
		{"/v1beta/tunedModels/tuned-123", EndpointGeminiTunedModels},
		{"/v1beta/tunedModels/tuned-123/permissions/perm-123", EndpointGeminiTunedModelsPermissions},
		{"/v1beta/tunedModels/tuned-123/operations", EndpointGeminiTunedModelsOperations},
		{"/v1beta/tunedModels/tuned-123/operations/op-123", EndpointGeminiTunedModelsOperations},
		{"/v1beta/tunedModels/tuned-123:asyncBatchEmbedContent", EndpointGeminiTunedModels},
		{"/v1beta/tunedModels/tuned-123:generateContent", EndpointGeminiTunedModels},
		{"/v1beta/openai/chat/completions", EndpointGeminiOpenAICompat},
		{"/v1beta/openai/files", EndpointGeminiOpenAICompat},
		{"/v1beta/openai/files/file_123", EndpointGeminiOpenAICompat},
		{"/v1beta/openai/batches/batch_123", EndpointGeminiOpenAICompat},
		{"/v1beta/interactions/sample", EndpointGeminiInteractions},
		{"/v1beta/live/sample", EndpointGeminiLive},
		{"/v1beta/embeddings", EndpointGeminiEmbeddings},
		{"/v1/vertex/models/gemini-2.5-pro:generateContent", EndpointVertexSyncModels},
		{"/v1/vertex/batchPredictionJobs", EndpointVertexBatchJobs},
		{"/v1/vertex/batchPredictionJobs/job-1", EndpointVertexBatchJobs},
		{"/vertex-batch/jobs/job-1:cancel", EndpointVertexBatchJobs},

		// Prefixed paths (antigravity, openai).
		{"/antigravity/v1/messages", EndpointMessages},
		{"/openai/v1/responses", EndpointResponses},
		{"/openai/v1/responses/compact", EndpointResponses},
		{"/grok/v1/videos", EndpointVideosCreate},
		{"/antigravity/v1beta/models/gemini:generateContent", EndpointGeminiModels},

		// Gin route patterns with wildcards.
		{"/v1beta/models/*modelAction", EndpointGeminiModels},
		{"/v1/responses/*subpath", EndpointResponses},

		// Unknown path is returned as-is.
		{"/v1/embeddings", "/v1/embeddings"},
		{"", ""},
		{"  /v1/messages  ", EndpointMessages},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			require.Equal(t, tt.want, NormalizeInboundEndpoint(tt.path))
		})
	}
}

// ──────────────────────────────────────────────────────────
// DeriveUpstreamEndpoint
// ──────────────────────────────────────────────────────────

func TestDeriveUpstreamEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		inbound  string
		rawPath  string
		platform string
		want     string
	}{
		// Anthropic.
		{"anthropic messages", EndpointMessages, "/v1/messages", service.PlatformAnthropic, EndpointMessages},

		// Gemini.
		{"gemini models", EndpointGeminiModels, "/v1beta/models/gemini:gen", service.PlatformGemini, EndpointGeminiModels},
		{"gemini cached contents", EndpointGeminiCachedContents, "/v1beta/cachedContents/sample", service.PlatformGemini, EndpointGeminiCachedContents},
		{"gemini operations", EndpointGeminiOperations, "/v1beta/operations/sample", service.PlatformGemini, EndpointGeminiOperations},
		{"gemini upload operations", EndpointGeminiUploadOperations, "/v1beta/fileSearchStores/default/upload/operations/op-123", service.PlatformGemini, EndpointGeminiUploadOperations},
		{"gemini corpora", EndpointGeminiCorpora, "/v1beta/corpora/default-corpus", service.PlatformGemini, EndpointGeminiCorpora},
		{"gemini corpora operations", EndpointGeminiCorporaOperations, "/v1beta/corpora/default-corpus/operations/op-123", service.PlatformGemini, EndpointGeminiCorporaOperations},
		{"gemini corpora permissions", EndpointGeminiCorporaPermissions, "/v1beta/corpora/default-corpus/permissions/perm-123", service.PlatformGemini, EndpointGeminiCorporaPermissions},
		{"gemini dynamic", EndpointGeminiDynamic, "/v1beta/dynamic/session-123:generateContent", service.PlatformGemini, EndpointGeminiDynamic},
		{"gemini generated files", EndpointGeminiGeneratedFiles, "/v1beta/generatedFiles", service.PlatformGemini, EndpointGeminiGeneratedFiles},
		{"gemini generated file operations", EndpointGeminiGeneratedFilesOperations, "/v1beta/generatedFiles/file-123/operations/op-123", service.PlatformGemini, EndpointGeminiGeneratedFilesOperations},
		{"gemini model operations", EndpointGeminiModelOperations, "/v1beta/models/gemini-2.5-pro/operations/op-123", service.PlatformGemini, EndpointGeminiModelOperations},
		{"gemini tuned models", EndpointGeminiTunedModels, "/v1beta/tunedModels/tuned-123", service.PlatformGemini, EndpointGeminiTunedModels},
		{"gemini tuned model permissions", EndpointGeminiTunedModelsPermissions, "/v1beta/tunedModels/tuned-123/permissions/perm-123", service.PlatformGemini, EndpointGeminiTunedModelsPermissions},
		{"gemini tuned model operations", EndpointGeminiTunedModelsOperations, "/v1beta/tunedModels/tuned-123/operations/op-123", service.PlatformGemini, EndpointGeminiTunedModelsOperations},
		{"gemini live auth tokens", EndpointGeminiLiveAuthTokens, "/v1alpha/authTokens", service.PlatformGemini, EndpointGeminiLiveAuthTokens},
		{"gemini openai compat", EndpointGeminiOpenAICompat, "/v1beta/openai/chat/completions", service.PlatformGemini, EndpointGeminiOpenAICompat},
		{"gemini openai compat files", EndpointGeminiOpenAICompat, "/v1beta/openai/files/file_123", service.PlatformGemini, EndpointGeminiOpenAICompat},
		{"gemini openai compat batches", EndpointGeminiOpenAICompat, "/v1beta/openai/batches/batch_123", service.PlatformGemini, EndpointGeminiOpenAICompat},
		{"gemini live", EndpointGeminiLive, "/v1beta/live/sample", service.PlatformGemini, EndpointGeminiLive},

		// Grok videos.
		{"grok videos create canonical", EndpointVideosCreate, "/v1/videos", service.PlatformGrok, EndpointVideosGen},
		{"grok videos create alias", EndpointVideosCreate, "/v1/videos/generations", service.PlatformGrok, EndpointVideosGen},

		// OpenAI — always /v1/responses.
		{"openai responses root", EndpointResponses, "/v1/responses", service.PlatformOpenAI, EndpointResponses},
		{"openai responses compact", EndpointResponses, "/openai/v1/responses/compact", service.PlatformOpenAI, "/v1/responses/compact"},
		{"openai responses nested", EndpointResponses, "/openai/v1/responses/compact/detail", service.PlatformOpenAI, "/v1/responses/compact/detail"},
		{"openai from messages", EndpointMessages, "/v1/messages", service.PlatformOpenAI, EndpointResponses},
		{"openai from completions", EndpointChatCompletions, "/v1/chat/completions", service.PlatformOpenAI, EndpointResponses},

		// Antigravity — uses inbound to pick Claude vs Gemini upstream.
		{"antigravity claude", EndpointMessages, "/antigravity/v1/messages", service.PlatformAntigravity, EndpointMessages},
		{"antigravity gemini", EndpointGeminiModels, "/antigravity/v1beta/models", service.PlatformAntigravity, EndpointGeminiModels},

		// Unknown platform — passthrough.
		{"unknown platform", "/v1/embeddings", "/v1/embeddings", "unknown", "/v1/embeddings"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, DeriveUpstreamEndpoint(tt.inbound, tt.rawPath, tt.platform))
		})
	}
}

func TestDeriveUpstreamEndpointForAccount_UsesProtocolGatewayOpenAIPreference(t *testing.T) {
	account := &service.Account{
		Platform: service.PlatformProtocolGateway,
		Type:     service.AccountTypeAPIKey,
		Extra: map[string]any{
			"gateway_protocol":              service.GatewayProtocolOpenAI,
			"gateway_openai_request_format": service.GatewayOpenAIRequestFormatChatCompletions,
		},
	}

	require.Equal(t,
		EndpointChatCompletions,
		DeriveUpstreamEndpointForAccount(account, EndpointChatCompletions, "/v1/chat/completions"),
	)
	require.Equal(t,
		EndpointChatCompletions,
		DeriveUpstreamEndpointForAccount(account, EndpointResponses, "/v1/responses"),
	)
}

// ──────────────────────────────────────────────────────────
// responsesSubpathSuffix
// ──────────────────────────────────────────────────────────

func TestResponsesSubpathSuffix(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"/v1/responses", ""},
		{"/v1/responses/", ""},
		{"/v1/responses/compact", "/compact"},
		{"/openai/v1/responses/compact/detail", "/compact/detail"},
		{"/v1/messages", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			require.Equal(t, tt.want, responsesSubpathSuffix(tt.raw))
		})
	}
}

// ──────────────────────────────────────────────────────────
// InboundEndpointMiddleware + context helpers
// ──────────────────────────────────────────────────────────

func TestInboundEndpointMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(InboundEndpointMiddleware())

	var captured string
	router.POST("/v1/messages", func(c *gin.Context) {
		captured = GetInboundEndpoint(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, EndpointMessages, captured)
}

func TestGetInboundEndpoint_FallbackWithoutMiddleware(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/antigravity/v1/messages", nil)

	// Middleware did not run — fallback to normalizing c.Request.URL.Path.
	got := GetInboundEndpoint(c)
	require.Equal(t, EndpointMessages, got)
}

func TestGetUpstreamEndpoint_FullFlow(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses/compact", nil)

	// Simulate middleware.
	c.Set(ctxKeyInboundEndpoint, NormalizeInboundEndpoint(c.Request.URL.Path))

	got := GetUpstreamEndpoint(c, service.PlatformOpenAI)
	require.Equal(t, "/v1/responses/compact", got)
}

func TestGetUpstreamEndpointForAccount_UsesAccountPreference(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(ctxKeyInboundEndpoint, NormalizeInboundEndpoint(c.Request.URL.Path))

	account := &service.Account{
		Platform: service.PlatformProtocolGateway,
		Type:     service.AccountTypeAPIKey,
		Extra: map[string]any{
			"gateway_protocol":              service.GatewayProtocolOpenAI,
			"gateway_openai_request_format": service.GatewayOpenAIRequestFormatChatCompletions,
		},
	}

	require.Equal(t, EndpointChatCompletions, GetUpstreamEndpointForAccount(c, account))
}
