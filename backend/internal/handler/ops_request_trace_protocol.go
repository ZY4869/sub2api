package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func normalizeOpsTraceProtocolValue(value string) string {
	normalized := strings.TrimSpace(service.NormalizeInboundEndpoint(value))
	if normalized != "" && len(normalized) <= 50 {
		return normalized
	}
	return ""
}

func opsTraceProtocolFamily(value string) string {
	switch normalized := strings.ToLower(normalizeOpsTraceProtocolValue(value)); normalized {
	case "", "unknown":
		return ""
	case "openai", "anthropic", "claude", "gemini", "vertex":
		if normalized == "claude" {
			return "anthropic"
		}
		if normalized == "vertex" {
			return "gemini"
		}
		return normalized
	case EndpointMessages:
		return "anthropic"
	case EndpointChatCompletions, EndpointEmbeddings, EndpointResponses, EndpointImagesGen, EndpointImagesEdits, EndpointVideosCreate, EndpointVideosGen, EndpointVideosStatus:
		return "openai"
	case EndpointGeminiModels,
		EndpointGeminiFiles,
		EndpointGeminiFilesUp,
		EndpointGeminiBatches,
		EndpointGeminiCachedContents,
		EndpointGeminiFileSearchStores,
		EndpointGeminiDocuments,
		EndpointGeminiOperations,
		EndpointGeminiUploadOperations,
		EndpointGeminiEmbeddings,
		EndpointGeminiInteractions,
		EndpointGeminiCorpora,
		EndpointGeminiCorporaOperations,
		EndpointGeminiCorporaPermissions,
		EndpointGeminiDynamic,
		EndpointGeminiGeneratedFiles,
		EndpointGeminiGeneratedFilesOperations,
		EndpointGeminiModelOperations,
		EndpointGeminiTunedModels,
		EndpointGeminiTunedModelsPermissions,
		EndpointGeminiTunedModelsOperations,
		EndpointGeminiLive,
		EndpointGeminiLiveAuthTokens,
		EndpointGeminiOpenAICompat,
		EndpointVertexSyncModels,
		EndpointVertexBatchJobs:
		return "gemini"
	default:
		switch {
		case strings.HasPrefix(normalized, "/v1/messages"):
			return "anthropic"
		case strings.HasPrefix(normalized, "/v1/chat/completions"),
			strings.HasPrefix(normalized, "/v1/embeddings"),
			strings.HasPrefix(normalized, "/v1/responses"),
			strings.HasPrefix(normalized, "/v1/images/"),
			strings.HasPrefix(normalized, "/v1/videos"):
			return "openai"
		case strings.HasPrefix(normalized, "/v1beta/"),
			strings.HasPrefix(normalized, "/v1alpha/"),
			strings.HasPrefix(normalized, "/upload/v1beta/"),
			strings.HasPrefix(normalized, "/download/v1beta/"),
			strings.HasPrefix(normalized, "/google/batch/archive/"),
			strings.HasPrefix(normalized, "/v1/projects/"):
			return "gemini"
		default:
			return ""
		}
	}
}
