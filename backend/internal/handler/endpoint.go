package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ──────────────────────────────────────────────────────────
// Canonical inbound / upstream endpoint paths.
// All normalization and derivation reference this single set
// of constants — add new paths HERE when a new API surface
// is introduced.
// ──────────────────────────────────────────────────────────

const (
	EndpointMessages               = service.EndpointMessages
	EndpointChatCompletions        = service.EndpointChatCompletions
	EndpointResponses              = service.EndpointResponses
	EndpointImagesGen              = service.EndpointImagesGen
	EndpointImagesEdits            = service.EndpointImagesEdits
	EndpointVideosCreate           = service.EndpointVideosCreate
	EndpointVideosGen              = service.EndpointVideosGen
	EndpointVideosStatus           = service.EndpointVideosStatus
	EndpointGeminiModels           = service.EndpointGeminiModels
	EndpointGeminiFiles            = service.EndpointGeminiFiles
	EndpointGeminiFilesUp          = service.EndpointGeminiFilesUp
	EndpointGeminiBatches          = service.EndpointGeminiBatches
	EndpointGeminiCachedContents   = service.EndpointGeminiCachedContents
	EndpointGeminiFileSearchStores = service.EndpointGeminiFileSearchStores
	EndpointGeminiDocuments        = service.EndpointGeminiDocuments
	EndpointGeminiOperations       = service.EndpointGeminiOperations
	EndpointGeminiEmbeddings       = service.EndpointGeminiEmbeddings
	EndpointGeminiInteractions     = service.EndpointGeminiInteractions
	EndpointGeminiLive             = service.EndpointGeminiLive
	EndpointGeminiOpenAICompat     = service.EndpointGeminiOpenAICompat
	EndpointVertexSyncModels       = service.EndpointVertexSyncModels
	EndpointVertexBatchJobs        = service.EndpointVertexBatchJobs
)

// gin.Context keys used by the middleware and helpers below.
const (
	ctxKeyInboundEndpoint = "_gateway_inbound_endpoint"
)

// ──────────────────────────────────────────────────────────
// Normalization functions
// ──────────────────────────────────────────────────────────

// NormalizeInboundEndpoint maps a raw request path (which may carry
// prefixes like /antigravity and /openai) to its canonical form.
//
//	"/antigravity/v1/messages"   → "/v1/messages"
//	"/v1/chat/completions"       → "/v1/chat/completions"
//	"/openai/v1/responses/foo"   → "/v1/responses"
//	"/v1beta/models/gemini:gen"  → "/v1beta/models"
func NormalizeInboundEndpoint(path string) string {
	return service.NormalizeInboundEndpoint(path)
}

// DeriveUpstreamEndpoint determines the upstream endpoint from the
// account platform and the normalized inbound endpoint.
//
// Platform-specific rules:
//   - OpenAI always forwards to /v1/responses (with optional subpath
//     such as /v1/responses/compact preserved from the raw URL).
//   - Anthropic  → /v1/messages
//   - Gemini     → /v1beta/models
//   - Antigravity routes may target either Claude or Gemini, so the
//     inbound endpoint is used to distinguish.
func DeriveUpstreamEndpoint(inbound, rawRequestPath, platform string) string {
	inbound = strings.TrimSpace(inbound)

	switch platform {
	case service.PlatformOpenAI, service.PlatformCopilot:
		// OpenAI forwards everything to the Responses API.
		// Preserve subresource suffix (e.g. /v1/responses/compact).
		if suffix := responsesSubpathSuffix(rawRequestPath); suffix != "" {
			return EndpointResponses + suffix
		}
		return EndpointResponses

	case service.PlatformAnthropic, service.PlatformKiro:
		return EndpointMessages

	case service.PlatformGemini:
		switch inbound {
		case EndpointGeminiFiles,
			EndpointGeminiFilesUp,
			EndpointGeminiBatches,
			EndpointGeminiCachedContents,
			EndpointGeminiFileSearchStores,
			EndpointGeminiDocuments,
			EndpointGeminiOperations,
			EndpointGeminiEmbeddings,
			EndpointGeminiInteractions,
			EndpointGeminiLive,
			EndpointGeminiOpenAICompat,
			EndpointVertexSyncModels,
			EndpointVertexBatchJobs:
			return inbound
		default:
			return EndpointGeminiModels
		}

	case service.PlatformGrok:
		if suffix := responsesSubpathSuffix(rawRequestPath); suffix != "" {
			return EndpointResponses + suffix
		}
		if strings.Contains(rawRequestPath, "/videos/") && !strings.Contains(rawRequestPath, EndpointVideosGen) {
			return normalizeVideoStatusPath(rawRequestPath)
		}
		switch inbound {
		case EndpointChatCompletions, EndpointResponses, EndpointImagesGen, EndpointImagesEdits:
			return inbound
		case EndpointVideosCreate, EndpointVideosGen:
			return EndpointVideosGen
		case EndpointVideosStatus:
			return normalizeVideoStatusPath(rawRequestPath)
		}
		return inbound

	case service.PlatformAntigravity:
		// Antigravity accounts serve both Claude and Gemini.
		if inbound == EndpointGeminiModels || inbound == EndpointVertexSyncModels {
			return EndpointGeminiModels
		}
		return EndpointMessages
	}

	// Unknown platform — fall back to inbound.
	return inbound
}

func normalizeVideoStatusPath(rawPath string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(rawPath), "/")
	if trimmed == "" {
		return EndpointVideosStatus
	}
	idx := strings.LastIndex(trimmed, "/videos/")
	if idx < 0 {
		return EndpointVideosStatus
	}
	requestID := strings.TrimSpace(trimmed[idx+len("/videos/"):])
	if requestID == "" {
		return EndpointVideosStatus
	}
	return "/v1/videos/" + requestID
}

// responsesSubpathSuffix extracts the part after "/responses" in a raw
// request path, e.g. "/openai/v1/responses/compact" → "/compact".
// Returns "" when there is no meaningful suffix.
func responsesSubpathSuffix(rawPath string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(rawPath), "/")
	idx := strings.LastIndex(trimmed, "/responses")
	if idx < 0 {
		return ""
	}
	suffix := trimmed[idx+len("/responses"):]
	if suffix == "" || suffix == "/" {
		return ""
	}
	if !strings.HasPrefix(suffix, "/") {
		return ""
	}
	return suffix
}

// ──────────────────────────────────────────────────────────
// Middleware
// ──────────────────────────────────────────────────────────

// InboundEndpointMiddleware normalizes the request path and stores the
// canonical inbound endpoint in gin.Context so that every handler in
// the chain can read it via GetInboundEndpoint.
//
// Apply this middleware to all gateway route groups.
func InboundEndpointMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" && c.Request != nil && c.Request.URL != nil {
			path = c.Request.URL.Path
		}
		c.Set(ctxKeyInboundEndpoint, NormalizeInboundEndpoint(path))
		c.Next()
	}
}

// ──────────────────────────────────────────────────────────
// Context helpers — used by handlers before building
// RecordUsageInput / RecordUsageLongContextInput.
// ──────────────────────────────────────────────────────────

// GetInboundEndpoint returns the canonical inbound endpoint stored by
// InboundEndpointMiddleware. If the middleware did not run (e.g. in
// tests), it falls back to normalizing c.FullPath() on the fly.
func GetInboundEndpoint(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyInboundEndpoint); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	// Fallback: normalize on the fly.
	path := ""
	if c != nil {
		path = c.FullPath()
		if path == "" && c.Request != nil && c.Request.URL != nil {
			path = c.Request.URL.Path
		}
	}
	return NormalizeInboundEndpoint(path)
}

// GetUpstreamEndpoint derives the upstream endpoint from the context
// and the account platform. Handlers call this after scheduling an
// account, passing account.Platform.
func GetUpstreamEndpoint(c *gin.Context, platform string) string {
	inbound := GetInboundEndpoint(c)
	rawPath := ""
	if c != nil && c.Request != nil && c.Request.URL != nil {
		rawPath = c.Request.URL.Path
	}
	return DeriveUpstreamEndpoint(inbound, rawPath, platform)
}
