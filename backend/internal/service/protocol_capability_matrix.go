package service

import (
	"net/http"
	"regexp"
	"strings"
)

const (
	EndpointMessages                  = "/v1/messages"
	EndpointChatCompletions           = "/v1/chat/completions"
	EndpointResponses                 = "/v1/responses"
	EndpointImagesGen                 = "/v1/images/generations"
	EndpointImagesEdits               = "/v1/images/edits"
	EndpointVideosCreate              = "/v1/videos"
	EndpointVideosGen                 = "/v1/videos/generations"
	EndpointVideosStatus              = "/v1/videos/:request_id"
	EndpointGeminiModels              = "/v1beta/models"
	EndpointGeminiFiles               = "/v1beta/files"
	EndpointGeminiFilesUp             = "/upload/v1beta/files"
	EndpointGeminiFilesDownload       = "/download/v1beta/files"
	EndpointGeminiBatches             = "/v1beta/batches"
	EndpointGeminiCachedContents      = "/v1beta/cachedContents"
	EndpointGeminiFileSearchStores    = "/v1beta/fileSearchStores"
	EndpointGeminiDocuments           = "/v1beta/documents"
	EndpointGeminiOperations          = "/v1beta/operations"
	EndpointGeminiEmbeddings          = "/v1beta/embeddings"
	EndpointGeminiInteractions        = "/v1beta/interactions"
	EndpointGeminiLive                = "/v1beta/live"
	EndpointGeminiOpenAICompat        = "/v1beta/openai"
	EndpointGoogleBatchArchiveBatches = "/google/batch/archive/v1beta/batches"
	EndpointGoogleBatchArchiveFiles   = "/google/batch/archive/v1beta/files"
	EndpointVertexSyncModels          = "/v1/projects/:project/locations/:location/publishers/google/models"
	EndpointVertexBatchJobs           = "/v1/projects/:project/locations/:location/batchPredictionJobs"
)

type ProtocolCapabilityMode string

const (
	ProtocolCapabilityNativePassthrough ProtocolCapabilityMode = "native_passthrough"
	ProtocolCapabilityCompatTranslate   ProtocolCapabilityMode = "compat_translate"
	ProtocolCapabilityReject            ProtocolCapabilityMode = "reject"
)

type PublicProtocolCapability struct {
	InboundEndpoint string
	RequestFormat   string
	Action          string
	SourceProtocol  string
	RuntimePlatform string
	Mode            ProtocolCapabilityMode
}

type PublicEndpointRoute struct {
	Method                  string
	Pattern                 string
	RegisteredHandlerFamily string
}

type PublicEndpointRegistryEntry struct {
	CanonicalEndpoint string
	SourceProtocol    string
	HandlerFamily     string
	Routes            []PublicEndpointRoute
	NormalizePrefixes []string
	Capabilities      []PublicProtocolCapability
}

type ProtocolCapabilityDecision struct {
	Supported            bool                   `json:"supported"`
	Mode                 ProtocolCapabilityMode `json:"mode,omitempty"`
	Reason               string                 `json:"reason,omitempty"`
	MessageKey           string                 `json:"message_key,omitempty"`
	RequestFormat        string                 `json:"request_format,omitempty"`
	StatusCode           int                    `json:"status_code,omitempty"`
	InternalMismatchKind string                 `json:"internal_mismatch_kind,omitempty"`
}

const (
	GatewayReasonRouteMismatch             = "GATEWAY_ROUTE_MISMATCH"
	GatewayReasonUnsupportedAction         = "GATEWAY_UNSUPPORTED_ACTION"
	GatewayReasonPublicEndpointUnsupported = "GATEWAY_PUBLIC_ENDPOINT_UNSUPPORTED"
)

const (
	ProtocolCapabilityActionDefault               = ""
	ProtocolCapabilityActionCountTokens           = "count_tokens"
	ProtocolCapabilityActionWebSocket             = "websocket"
	ProtocolCapabilityActionGenerateContent       = "generateContent"
	ProtocolCapabilityActionStreamGenerateContent = "streamGenerateContent"
	ProtocolCapabilityActionBatchGenerateContent  = "batchGenerateContent"
	ProtocolCapabilityActionGeminiCountTokens     = "countTokens"
	ProtocolCapabilityActionGeminiEmbedContent    = "embedContent"
	ProtocolCapabilityActionGeminiBatchEmbeddings = "batchEmbedContents"
	ProtocolCapabilityActionGeminiAsyncEmbedding  = "asyncBatchEmbedContent"
)

var publicEndpointRegistry = []PublicEndpointRegistryEntry{
	{
		CanonicalEndpoint: EndpointMessages,
		SourceProtocol:    PlatformAnthropic,
		HandlerFamily:     "anthropic_messages",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/messages"},
			{Method: http.MethodPost, Pattern: "/v1/messages/count_tokens"},
			{Method: http.MethodPost, Pattern: "/antigravity/v1/messages"},
			{Method: http.MethodPost, Pattern: "/antigravity/v1/messages/count_tokens"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformAnthropic, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityCompatTranslate},
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityCompatTranslate},
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformAnthropic, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityReject},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityReject},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityReject},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityReject},
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityReject},
		},
	},
	{
		CanonicalEndpoint: EndpointChatCompletions,
		SourceProtocol:    PlatformOpenAI,
		HandlerFamily:     "openai_chat_completions",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/chat/completions"},
			{Method: http.MethodPost, Pattern: "/chat/completions"},
			{Method: http.MethodPost, Pattern: "/grok/v1/chat/completions"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointResponses,
		SourceProtocol:    PlatformOpenAI,
		HandlerFamily:     "openai_responses",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/responses"},
			{Method: http.MethodPost, Pattern: "/v1/responses/*subpath"},
			{Method: http.MethodGet, Pattern: "/v1/responses"},
			{Method: http.MethodGet, Pattern: "/v1/responses/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1/responses/*subpath"},
			{Method: http.MethodPost, Pattern: "/responses"},
			{Method: http.MethodPost, Pattern: "/responses/*subpath"},
			{Method: http.MethodGet, Pattern: "/responses"},
			{Method: http.MethodGet, Pattern: "/responses/*subpath"},
			{Method: http.MethodDelete, Pattern: "/responses/*subpath"},
			{Method: http.MethodPost, Pattern: "/grok/v1/responses"},
			{Method: http.MethodPost, Pattern: "/grok/v1/responses/*subpath"},
			{Method: http.MethodGet, Pattern: "/grok/v1/responses/*subpath"},
			{Method: http.MethodDelete, Pattern: "/grok/v1/responses/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointResponses, RequestFormat: EndpointResponses, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointResponses, RequestFormat: EndpointResponses, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointResponses, RequestFormat: EndpointResponses, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointResponses, RequestFormat: EndpointResponses, Action: ProtocolCapabilityActionWebSocket, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointResponses, RequestFormat: EndpointResponses, Action: ProtocolCapabilityActionWebSocket, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointResponses, RequestFormat: EndpointResponses, Action: ProtocolCapabilityActionWebSocket, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityReject},
		},
	},
	{
		CanonicalEndpoint: EndpointImagesGen,
		SourceProtocol:    PlatformOpenAI,
		HandlerFamily:     "grok_images_generation",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/images/generations"},
			{Method: http.MethodPost, Pattern: "/images/generations"},
			{Method: http.MethodPost, Pattern: "/grok/v1/images/generations"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointImagesGen, RequestFormat: EndpointImagesGen, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointImagesEdits,
		SourceProtocol:    PlatformOpenAI,
		HandlerFamily:     "grok_images_edits",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/images/edits"},
			{Method: http.MethodPost, Pattern: "/images/edits"},
			{Method: http.MethodPost, Pattern: "/grok/v1/images/edits"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointImagesEdits, RequestFormat: EndpointImagesEdits, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointVideosCreate,
		SourceProtocol:    PlatformOpenAI,
		HandlerFamily:     "grok_videos_generation",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/videos"},
			{Method: http.MethodPost, Pattern: "/v1/videos/generations"},
			{Method: http.MethodPost, Pattern: "/videos"},
			{Method: http.MethodPost, Pattern: "/videos/generations"},
			{Method: http.MethodPost, Pattern: "/grok/v1/videos"},
			{Method: http.MethodPost, Pattern: "/grok/v1/videos/generations"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointVideosCreate, RequestFormat: EndpointVideosCreate, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointVideosCreate, RequestFormat: EndpointVideosGen, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointVideosStatus,
		SourceProtocol:    PlatformOpenAI,
		HandlerFamily:     "grok_videos_status",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1/videos/:request_id"},
			{Method: http.MethodGet, Pattern: "/videos/:request_id"},
			{Method: http.MethodGet, Pattern: "/grok/v1/videos/:request_id"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointVideosStatus, RequestFormat: EndpointVideosStatus, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiModels,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_models",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/models"},
			{Method: http.MethodGet, Pattern: "/v1beta/models/:model"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/*modelAction"},
			{Method: http.MethodGet, Pattern: "/antigravity/v1beta/models"},
			{Method: http.MethodGet, Pattern: "/antigravity/v1beta/models/:model"},
			{Method: http.MethodPost, Pattern: "/antigravity/v1beta/models/*modelAction"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:generateContent", Action: ProtocolCapabilityActionGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:generateContent", Action: ProtocolCapabilityActionGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityCompatTranslate},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:streamGenerateContent", Action: ProtocolCapabilityActionStreamGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:streamGenerateContent", Action: ProtocolCapabilityActionStreamGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityCompatTranslate},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:countTokens", Action: ProtocolCapabilityActionGeminiCountTokens, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:countTokens", Action: ProtocolCapabilityActionGeminiCountTokens, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityCompatTranslate},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiFiles,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_files",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/files"},
			{Method: http.MethodPost, Pattern: "/v1beta/files"},
			{Method: http.MethodPost, Pattern: "/v1beta/files:action"},
			{Method: http.MethodGet, Pattern: "/v1beta/files/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/files/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiFiles, RequestFormat: EndpointGeminiFiles, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiFilesUp,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_files_upload",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/upload/v1beta/files"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiFilesUp, RequestFormat: EndpointGeminiFilesUp, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiFilesDownload,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_files_download",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/download/v1beta/files/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiFilesDownload, RequestFormat: EndpointGeminiFilesDownload, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiBatches,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_batches",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/batches"},
			{Method: http.MethodGet, Pattern: "/v1beta/batches/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1beta/batches/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/batches/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/{model}:batchGenerateContent", RegisteredHandlerFamily: "gemini_models"},
			{Method: http.MethodPost, Pattern: "/antigravity/v1beta/models/{model}:batchGenerateContent", RegisteredHandlerFamily: "gemini_models"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiBatches, RequestFormat: "/v1beta/models/{model}:batchGenerateContent", Action: ProtocolCapabilityActionBatchGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiBatches, RequestFormat: "/v1beta/batches/{batch}", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiBatches, RequestFormat: "/v1beta/models/{model}:batchGenerateContent", Action: ProtocolCapabilityActionBatchGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityReject},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiCachedContents,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_cached_contents",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/cachedContents"},
			{Method: http.MethodPost, Pattern: "/v1beta/cachedContents"},
			{Method: http.MethodGet, Pattern: "/v1beta/cachedContents/*subpath"},
			{Method: http.MethodPatch, Pattern: "/v1beta/cachedContents/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/cachedContents/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiCachedContents, RequestFormat: EndpointGeminiCachedContents, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiFileSearchStores,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_file_search_stores",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/fileSearchStores"},
			{Method: http.MethodPost, Pattern: "/v1beta/fileSearchStores"},
			{Method: http.MethodGet, Pattern: "/v1beta/fileSearchStores/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1beta/fileSearchStores/*subpath"},
			{Method: http.MethodPatch, Pattern: "/v1beta/fileSearchStores/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/fileSearchStores/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiFileSearchStores, RequestFormat: EndpointGeminiFileSearchStores, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiDocuments,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_documents",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/documents"},
			{Method: http.MethodPost, Pattern: "/v1beta/documents"},
			{Method: http.MethodGet, Pattern: "/v1beta/documents/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1beta/documents/*subpath"},
			{Method: http.MethodPatch, Pattern: "/v1beta/documents/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/documents/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiDocuments, RequestFormat: EndpointGeminiDocuments, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiOperations,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_operations",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/operations"},
			{Method: http.MethodGet, Pattern: "/v1beta/operations/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/operations/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiOperations, RequestFormat: EndpointGeminiOperations, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiEmbeddings,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_embeddings",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1beta/embeddings"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/{model}:embedContent", RegisteredHandlerFamily: "gemini_models"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/{model}:batchEmbedContents", RegisteredHandlerFamily: "gemini_models"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/{model}:asyncBatchEmbedContent", RegisteredHandlerFamily: "gemini_models"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiEmbeddings, RequestFormat: EndpointGeminiEmbeddings, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiEmbeddings, RequestFormat: "/v1beta/models/{model}:embedContent", Action: ProtocolCapabilityActionGeminiEmbedContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiEmbeddings, RequestFormat: "/v1beta/models/{model}:batchEmbedContents", Action: ProtocolCapabilityActionGeminiBatchEmbeddings, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiEmbeddings, RequestFormat: "/v1beta/models/{model}:asyncBatchEmbedContent", Action: ProtocolCapabilityActionGeminiAsyncEmbedding, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiInteractions,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_interactions",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1beta/interactions"},
			{Method: http.MethodGet, Pattern: "/v1beta/interactions/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1beta/interactions/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/interactions/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiInteractions, RequestFormat: EndpointGeminiInteractions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiLive,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_live",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/live"},
			{Method: http.MethodGet, Pattern: "/v1beta/live/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1beta/live"},
			{Method: http.MethodPost, Pattern: "/v1beta/live/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiLive, RequestFormat: EndpointGeminiLive, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiLive, RequestFormat: EndpointGeminiLive, Action: ProtocolCapabilityActionWebSocket, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiOpenAICompat,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_openai_compat",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/openai/models"},
			{Method: http.MethodGet, Pattern: "/v1beta/openai/models/*subpath"},
			{Method: http.MethodGet, Pattern: "/v1beta/openai/files"},
			{Method: http.MethodPost, Pattern: "/v1beta/openai/files"},
			{Method: http.MethodGet, Pattern: "/v1beta/openai/files/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/openai/files/*subpath"},
			{Method: http.MethodGet, Pattern: "/v1beta/openai/batches"},
			{Method: http.MethodPost, Pattern: "/v1beta/openai/batches"},
			{Method: http.MethodGet, Pattern: "/v1beta/openai/batches/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1beta/openai/batches/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1beta/openai/chat/completions"},
			{Method: http.MethodPost, Pattern: "/v1beta/openai/embeddings"},
			{Method: http.MethodPost, Pattern: "/v1beta/openai/images/generations"},
			{Method: http.MethodPost, Pattern: "/v1beta/openai/videos"},
			{Method: http.MethodGet, Pattern: "/v1beta/openai/videos/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiOpenAICompat, RequestFormat: EndpointGeminiOpenAICompat, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGoogleBatchArchiveBatches,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "google_batch_archive_batches",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/google/batch/archive/v1beta/batches/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGoogleBatchArchiveBatches, RequestFormat: EndpointGoogleBatchArchiveBatches, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGoogleBatchArchiveFiles,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "google_batch_archive_files",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/google/batch/archive/v1beta/files/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGoogleBatchArchiveFiles, RequestFormat: EndpointGoogleBatchArchiveFiles, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointVertexSyncModels,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "vertex_models",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/projects/:project/locations/:location/publishers/google/models/*modelAction"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointVertexSyncModels, RequestFormat: EndpointVertexSyncModels, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointVertexBatchJobs,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "vertex_batch_prediction_jobs",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1/projects/:project/locations/:location/batchPredictionJobs"},
			{Method: http.MethodPost, Pattern: "/v1/projects/:project/locations/:location/batchPredictionJobs"},
			{Method: http.MethodGet, Pattern: "/v1/projects/:project/locations/:location/batchPredictionJobs/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1/projects/:project/locations/:location/batchPredictionJobs/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1/projects/:project/locations/:location/batchPredictionJobs/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointVertexBatchJobs, RequestFormat: EndpointVertexBatchJobs, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
}

var publicProtocolCapabilityMatrix = flattenPublicProtocolCapabilities(publicEndpointRegistry)

func PublicEndpointRegistryEntries() []PublicEndpointRegistryEntry {
	return append([]PublicEndpointRegistryEntry(nil), publicEndpointRegistry...)
}

func NormalizeInboundEndpoint(path string) string {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return ""
	}
	bestScore := -1
	bestEndpoint := ""
	for _, entry := range publicEndpointRegistry {
		for _, candidate := range publicEndpointNormalizationCandidates(normalizedPath, entry) {
			for _, route := range entry.Routes {
				if !matchPublicEndpointRoute(candidate, route.Pattern) {
					continue
				}
				score := publicEndpointRouteSpecificity(route.Pattern)
				if score > bestScore {
					bestScore = score
					bestEndpoint = entry.CanonicalEndpoint
				}
			}
		}
	}
	if bestEndpoint != "" {
		return bestEndpoint
	}
	return normalizedPath
}

func ProtocolGatewayRequestFormats(protocol string) []string {
	normalized := NormalizeGatewayProtocol(protocol)
	if normalized == "" {
		return nil
	}
	if normalized == GatewayProtocolMixed {
		formats := make([]string, 0)
		for _, baseProtocol := range gatewayBaseProtocols {
			formats = append(formats, ProtocolGatewayRequestFormats(baseProtocol)...)
		}
		return uniqueTrimmedStringsPreserveCase(formats)
	}

	formats := make([]string, 0)
	seen := make(map[string]struct{})
	for _, capability := range publicProtocolCapabilityMatrix {
		if capability.SourceProtocol != normalized {
			continue
		}
		if NormalizeProtocolCapabilityAction(capability.Action) == ProtocolCapabilityActionWebSocket {
			continue
		}
		if capability.RequestFormat == "" {
			continue
		}
		if _, ok := seen[capability.RequestFormat]; ok {
			continue
		}
		seen[capability.RequestFormat] = struct{}{}
		formats = append(formats, capability.RequestFormat)
	}
	return formats
}

func NormalizeProtocolCapabilityAction(action string) string {
	switch strings.TrimSpace(action) {
	case ProtocolCapabilityActionDefault:
		return ProtocolCapabilityActionDefault
	case ProtocolCapabilityActionCountTokens:
		return ProtocolCapabilityActionCountTokens
	case ProtocolCapabilityActionWebSocket:
		return ProtocolCapabilityActionWebSocket
	case ProtocolCapabilityActionGenerateContent:
		return ProtocolCapabilityActionGenerateContent
	case ProtocolCapabilityActionStreamGenerateContent:
		return ProtocolCapabilityActionStreamGenerateContent
	case ProtocolCapabilityActionBatchGenerateContent:
		return ProtocolCapabilityActionBatchGenerateContent
	case ProtocolCapabilityActionGeminiCountTokens:
		return ProtocolCapabilityActionGeminiCountTokens
	case ProtocolCapabilityActionGeminiEmbedContent:
		return ProtocolCapabilityActionGeminiEmbedContent
	case ProtocolCapabilityActionGeminiBatchEmbeddings:
		return ProtocolCapabilityActionGeminiBatchEmbeddings
	case ProtocolCapabilityActionGeminiAsyncEmbedding:
		return ProtocolCapabilityActionGeminiAsyncEmbedding
	default:
		return strings.TrimSpace(action)
	}
}

func GeminiActionEndpoint(action string) string {
	switch NormalizeProtocolCapabilityAction(action) {
	case ProtocolCapabilityActionBatchGenerateContent:
		return EndpointGeminiBatches
	case ProtocolCapabilityActionGeminiEmbedContent, ProtocolCapabilityActionGeminiBatchEmbeddings, ProtocolCapabilityActionGeminiAsyncEmbedding:
		return EndpointGeminiEmbeddings
	default:
		return EndpointGeminiModels
	}
}

func PublicEndpointRequestFormatForAction(inboundEndpoint string, action string) string {
	inboundEndpoint = NormalizeInboundEndpoint(inboundEndpoint)
	action = NormalizeProtocolCapabilityAction(action)
	fallback := inboundEndpoint
	for _, capability := range publicProtocolCapabilityMatrix {
		if capability.InboundEndpoint != inboundEndpoint {
			continue
		}
		if fallback == "" && capability.RequestFormat != "" {
			fallback = capability.RequestFormat
		}
		if NormalizeProtocolCapabilityAction(capability.Action) != action {
			continue
		}
		if capability.RequestFormat != "" {
			return capability.RequestFormat
		}
	}
	return fallback
}

func LookupProtocolCapability(runtimePlatform string, inboundEndpoint string) (ProtocolCapabilityMode, bool) {
	return LookupProtocolCapabilityForAction(runtimePlatform, inboundEndpoint, ProtocolCapabilityActionDefault)
}

func LookupProtocolCapabilityForAction(runtimePlatform string, inboundEndpoint string, action string) (ProtocolCapabilityMode, bool) {
	mode, endpointKnown, actionKnown := resolveProtocolCapability(runtimePlatform, inboundEndpoint, action)
	if !endpointKnown || !actionKnown {
		return "", false
	}
	return mode, true
}

func PublicEndpointUnsupportedDecision(inboundEndpoint string, action string) ProtocolCapabilityDecision {
	return ProtocolCapabilityDecision{
		Supported:            false,
		Reason:               GatewayReasonPublicEndpointUnsupported,
		MessageKey:           "gateway.public_endpoint.unsupported_platform",
		RequestFormat:        PublicEndpointRequestFormatForAction(inboundEndpoint, action),
		StatusCode:           http.StatusNotFound,
		InternalMismatchKind: "unsupported_platform",
	}
}

func UnsupportedActionDecision(inboundEndpoint string, action string) ProtocolCapabilityDecision {
	return ProtocolCapabilityDecision{
		Supported:            false,
		Reason:               GatewayReasonUnsupportedAction,
		MessageKey:           "gateway.public_endpoint.unsupported_action",
		RequestFormat:        PublicEndpointRequestFormatForAction(GeminiActionEndpoint(action), action),
		StatusCode:           http.StatusBadRequest,
		InternalMismatchKind: "unsupported_action",
	}
}

func GrokMessagesUnsupportedDecision() ProtocolCapabilityDecision {
	return ProtocolCapabilityDecision{
		Supported:            false,
		Reason:               GatewayReasonRouteMismatch,
		MessageKey:           "gateway.grok.messages_unsupported",
		RequestFormat:        EndpointMessages,
		StatusCode:           http.StatusBadRequest,
		InternalMismatchKind: "grok_messages_unsupported",
	}
}

func GrokAliasReservedDecision(inboundEndpoint string) ProtocolCapabilityDecision {
	return ProtocolCapabilityDecision{
		Supported:            false,
		Reason:               GatewayReasonRouteMismatch,
		MessageKey:           "gateway.grok.alias_reserved",
		RequestFormat:        PublicEndpointRequestFormatForAction(inboundEndpoint, ProtocolCapabilityActionDefault),
		StatusCode:           http.StatusNotFound,
		InternalMismatchKind: "grok_alias_reserved",
	}
}

func DecideProtocolCapability(runtimePlatform string, inboundEndpoint string, action string) ProtocolCapabilityDecision {
	mode, endpointKnown, actionKnown := resolveProtocolCapability(runtimePlatform, inboundEndpoint, action)
	switch {
	case !endpointKnown:
		return ProtocolCapabilityDecision{
			Supported:            false,
			Reason:               GatewayReasonRouteMismatch,
			MessageKey:           "gateway.public_endpoint.unsupported_platform",
			RequestFormat:        strings.TrimSpace(inboundEndpoint),
			StatusCode:           http.StatusNotFound,
			InternalMismatchKind: "unknown_public_endpoint",
		}
	case !actionKnown:
		return UnsupportedActionDecision(inboundEndpoint, action)
	case mode == ProtocolCapabilityReject:
		return PublicEndpointUnsupportedDecision(inboundEndpoint, action)
	default:
		return ProtocolCapabilityDecision{
			Supported:     true,
			Mode:          mode,
			RequestFormat: PublicEndpointRequestFormatForAction(inboundEndpoint, action),
			StatusCode:    http.StatusOK,
		}
	}
}

func resolveProtocolCapability(runtimePlatform string, inboundEndpoint string, action string) (ProtocolCapabilityMode, bool, bool) {
	runtimePlatform = strings.TrimSpace(strings.ToLower(runtimePlatform))
	inboundEndpoint = NormalizeInboundEndpoint(inboundEndpoint)
	action = NormalizeProtocolCapabilityAction(action)

	endpointKnown := false
	actionKnown := false
	for _, capability := range publicProtocolCapabilityMatrix {
		if capability.InboundEndpoint != inboundEndpoint {
			continue
		}
		endpointKnown = true
		if NormalizeProtocolCapabilityAction(capability.Action) != action {
			continue
		}
		actionKnown = true
		if capability.RuntimePlatform == runtimePlatform {
			return capability.Mode, true, true
		}
	}
	if endpointKnown && actionKnown {
		return ProtocolCapabilityReject, true, true
	}
	return "", endpointKnown, actionKnown
}

func flattenPublicProtocolCapabilities(entries []PublicEndpointRegistryEntry) []PublicProtocolCapability {
	capabilities := make([]PublicProtocolCapability, 0)
	for _, entry := range entries {
		capabilities = append(capabilities, entry.Capabilities...)
	}
	return capabilities
}

func matchPublicEndpointRoute(path string, pattern string) bool {
	path = normalizePublicRouteValue(path)
	pattern = normalizePublicRouteValue(pattern)
	if path == pattern {
		return true
	}

	expression := buildPublicRouteExpression(pattern)
	matched, _ := regexp.MatchString(expression, path)
	return matched
}

func buildPublicRouteExpression(pattern string) string {
	segments := splitPublicRouteSegments(pattern)
	if len(segments) == 0 {
		return `^/?$`
	}

	var expression strings.Builder
	writeBuilderString(&expression, "^")
	for _, segment := range segments {
		writeBuilderString(&expression, "/")
		switch {
		case strings.HasPrefix(segment, "*"):
			writeBuilderString(&expression, ".+")
			writeBuilderString(&expression, "$")
			return expression.String()
		case strings.HasPrefix(segment, ":"):
			writeBuilderString(&expression, `[^/]+`)
		default:
			segmentExpression := buildPublicRouteSegmentExpression(segment)
			segmentExpression = strings.TrimPrefix(segmentExpression, "^")
			segmentExpression = strings.TrimSuffix(segmentExpression, "$")
			writeBuilderString(&expression, segmentExpression)
		}
	}
	writeBuilderString(&expression, "$")
	return expression.String()
}

func normalizePublicRouteValue(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return ""
	}
	if len(normalized) > 1 {
		normalized = strings.TrimRight(normalized, "/")
	}
	return normalized
}

func publicEndpointNormalizationCandidates(path string, entry PublicEndpointRegistryEntry) []string {
	candidates := []string{path}
	for _, prefix := range entry.NormalizePrefixes {
		trimmedPrefix := strings.TrimSpace(prefix)
		if trimmedPrefix == "" {
			continue
		}
		if trimmed, ok := strings.CutPrefix(path, trimmedPrefix); ok && strings.HasPrefix(trimmed, "/") {
			candidates = append(candidates, trimmed)
		}
	}
	return uniqueTrimmedStringsPreserveCase(candidates)
}

func uniqueTrimmedStringsPreserveCase(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func splitPublicRouteSegments(value string) []string {
	normalized := strings.Trim(normalizePublicRouteValue(value), "/")
	if normalized == "" {
		return nil
	}
	return strings.Split(normalized, "/")
}

func publicEndpointRouteSpecificity(pattern string) int {
	score := 0
	for _, segment := range splitPublicRouteSegments(pattern) {
		switch {
		case strings.HasPrefix(segment, "*"):
			score += 1
		case strings.HasPrefix(segment, ":"):
			score += 2
		case strings.Contains(segment, "{"):
			score += 3
		default:
			score += 4
		}
	}
	return score
}

func buildPublicRouteSegmentExpression(pattern string) string {
	var expression strings.Builder
	writeBuilderString(&expression, "^")
	for index := 0; index < len(pattern); {
		switch pattern[index] {
		case '{':
			if end := strings.IndexByte(pattern[index:], '}'); end >= 0 {
				writeBuilderString(&expression, `[^/]+`)
				index += end + 1
				continue
			}
		case ':':
			if index > 0 && pattern[index-1] == '}' {
				break
			}
			if index+1 < len(pattern) && isRouteParamStart(pattern[index+1]) {
				end := index + 2
				for end < len(pattern) && isRouteParamContinue(pattern[end]) {
					end++
				}
				writeBuilderString(&expression, `[^/]+`)
				index = end
				continue
			}
		}
		writeBuilderString(&expression, regexp.QuoteMeta(string(pattern[index])))
		index++
	}
	writeBuilderString(&expression, "$")
	return expression.String()
}

func writeBuilderString(builder *strings.Builder, value string) {
	_, _ = builder.WriteString(value)
}

func isRouteParamStart(ch byte) bool {
	return ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isRouteParamContinue(ch byte) bool {
	return isRouteParamStart(ch) || (ch >= '0' && ch <= '9')
}
