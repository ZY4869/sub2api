package service

import (
	"net/http"
	"regexp"
	"strings"
)

const (
	EndpointMessages                       = "/v1/messages"
	EndpointChatCompletions                = "/v1/chat/completions"
	EndpointCompletions                    = "/v1/completions"
	EndpointResponses                      = "/v1/responses"
	EndpointImagesGen                      = "/v1/images/generations"
	EndpointImagesEdits                    = "/v1/images/edits"
	EndpointVideosCreate                   = "/v1/videos"
	EndpointVideosGen                      = "/v1/videos/generations"
	EndpointVideosStatus                   = "/v1/videos/:request_id"
	EndpointGeminiModels                   = "/v1beta/models"
	EndpointGeminiFiles                    = "/v1beta/files"
	EndpointGeminiFilesUp                  = "/upload/v1beta/files"
	EndpointGeminiFilesDownload            = "/download/v1beta/files"
	EndpointGeminiBatches                  = "/v1beta/batches"
	EndpointGeminiCachedContents           = "/v1beta/cachedContents"
	EndpointGeminiFileSearchStores         = "/v1beta/fileSearchStores"
	EndpointGeminiDocuments                = "/v1beta/documents"
	EndpointGeminiOperations               = "/v1beta/operations"
	EndpointGeminiUploadOperations         = "/v1beta/fileSearchStores/upload/operations"
	EndpointGeminiEmbeddings               = "/v1beta/embeddings"
	EndpointGeminiInteractions             = "/v1beta/interactions"
	EndpointGeminiCorpora                  = "/v1beta/corpora"
	EndpointGeminiCorporaOperations        = "/v1beta/corpora/operations"
	EndpointGeminiCorporaPermissions       = "/v1beta/corpora/permissions"
	EndpointGeminiDynamic                  = "/v1beta/dynamic"
	EndpointGeminiGeneratedFiles           = "/v1beta/generatedFiles"
	EndpointGeminiGeneratedFilesOperations = "/v1beta/generatedFiles/operations"
	EndpointGeminiModelOperations          = "/v1beta/models/operations"
	EndpointGeminiTunedModels              = "/v1beta/tunedModels"
	EndpointGeminiTunedModelsPermissions   = "/v1beta/tunedModels/permissions"
	EndpointGeminiTunedModelsOperations    = "/v1beta/tunedModels/operations"
	EndpointGeminiLive                     = "/v1beta/live"
	EndpointGeminiLiveAuthTokens           = "/v1alpha/authTokens"
	EndpointGeminiOpenAICompat             = "/v1beta/openai"
	EndpointGoogleBatchArchiveBatches      = "/google/batch/archive/v1beta/batches"
	EndpointGoogleBatchArchiveFiles        = "/google/batch/archive/v1beta/files"
	EndpointVertexSyncModels               = "/v1/projects/:project/locations/:location/publishers/google/models"
	EndpointVertexBatchJobs                = "/v1/projects/:project/locations/:location/batchPredictionJobs"
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
	ProtocolCapabilityActionGenerateAnswer        = "generateAnswer"
	ProtocolCapabilityActionStreamGenerateContent = "streamGenerateContent"
	ProtocolCapabilityActionBatchGenerateContent  = "batchGenerateContent"
	ProtocolCapabilityActionImportFile            = "importFile"
	ProtocolCapabilityActionTransferOwnership     = "transferOwnership"
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
			{Method: http.MethodPost, Pattern: "/deepseek/v1/messages"},
			{Method: http.MethodPost, Pattern: "/deepseek/v1/messages/count_tokens"},
			{Method: http.MethodPost, Pattern: "/antigravity/v1/messages"},
			{Method: http.MethodPost, Pattern: "/antigravity/v1/messages/count_tokens"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformAnthropic, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityCompatTranslate},
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityCompatTranslate},
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformDeepSeek, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformAnthropic, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityReject},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityReject},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformDeepSeek, Mode: ProtocolCapabilityReject},
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
			{Method: http.MethodPost, Pattern: "/deepseek/v1/chat/completions"},
			{Method: http.MethodPost, Pattern: "/grok/v1/chat/completions"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformDeepSeek, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointCompletions,
		SourceProtocol:    PlatformOpenAI,
		HandlerFamily:     "openai_completions",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/completions"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointCompletions, RequestFormat: EndpointCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformDeepSeek, Mode: ProtocolCapabilityNativePassthrough},
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
		HandlerFamily:     "public_images_generation",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/images/generations"},
			{Method: http.MethodPost, Pattern: "/images/generations"},
			{Method: http.MethodPost, Pattern: "/grok/v1/images/generations", RegisteredHandlerFamily: "grok_images_generation"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointImagesGen, RequestFormat: EndpointImagesGen, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointImagesGen, RequestFormat: EndpointImagesGen, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointImagesGen, RequestFormat: EndpointImagesGen, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointImagesGen, RequestFormat: EndpointImagesGen, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointImagesEdits,
		SourceProtocol:    PlatformOpenAI,
		HandlerFamily:     "public_images_edits",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/images/edits"},
			{Method: http.MethodPost, Pattern: "/images/edits"},
			{Method: http.MethodPost, Pattern: "/grok/v1/images/edits", RegisteredHandlerFamily: "grok_images_edits"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointImagesEdits, RequestFormat: EndpointImagesEdits, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointImagesEdits, RequestFormat: EndpointImagesEdits, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformCopilot, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointImagesEdits, RequestFormat: EndpointImagesEdits, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointImagesEdits, RequestFormat: EndpointImagesEdits, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityReject},
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
			{Method: http.MethodGet, Pattern: "/v1/models", RegisteredHandlerFamily: "gateway_v1_models_list"},
			{Method: http.MethodGet, Pattern: "/v1/models/:model", RegisteredHandlerFamily: "gateway_v1_models_get"},
			{Method: http.MethodPost, Pattern: "/v1/models/*modelAction", RegisteredHandlerFamily: "gateway_v1_models_action"},
			{Method: http.MethodGet, Pattern: "/deepseek/v1/models", RegisteredHandlerFamily: "gateway_models"},
			{Method: http.MethodGet, Pattern: "/v1beta/models"},
			{Method: http.MethodGet, Pattern: "/v1beta/models/{model}", RegisteredHandlerFamily: "gemini_models_get"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/{model}:generateAnswer"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/*modelAction"},
			{Method: http.MethodGet, Pattern: "/antigravity/v1beta/models"},
			{Method: http.MethodGet, Pattern: "/antigravity/v1beta/models/:model"},
			{Method: http.MethodPost, Pattern: "/antigravity/v1beta/models/*modelAction"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1/models", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1/models", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformDeepSeek, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:generateContent", Action: ProtocolCapabilityActionGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:generateContent", Action: ProtocolCapabilityActionGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityCompatTranslate},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1/models/{model}:generateContent", Action: ProtocolCapabilityActionGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:generateAnswer", Action: ProtocolCapabilityActionGenerateAnswer, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:streamGenerateContent", Action: ProtocolCapabilityActionStreamGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:streamGenerateContent", Action: ProtocolCapabilityActionStreamGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityCompatTranslate},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1/models/{model}:streamGenerateContent", Action: ProtocolCapabilityActionStreamGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:countTokens", Action: ProtocolCapabilityActionGeminiCountTokens, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1beta/models/{model}:countTokens", Action: ProtocolCapabilityActionGeminiCountTokens, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityCompatTranslate},
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1/models/{model}:countTokens", Action: ProtocolCapabilityActionGeminiCountTokens, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
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
			{Method: http.MethodPatch, Pattern: "/v1beta/batches/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/batches/*subpath"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/{model}:batchGenerateContent", RegisteredHandlerFamily: "gemini_models"},
			{Method: http.MethodPost, Pattern: "/antigravity/v1beta/models/{model}:batchGenerateContent", RegisteredHandlerFamily: "gemini_models"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiBatches, RequestFormat: "/v1beta/models/{model}:batchGenerateContent", Action: ProtocolCapabilityActionBatchGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiBatches, RequestFormat: "/v1beta/batches/{batch}", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiBatches, RequestFormat: "/v1beta/batches/{batch}:updateGenerateContentBatch", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiBatches, RequestFormat: "/v1beta/batches/{batch}:updateEmbedContentBatch", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
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
			{Method: http.MethodGet, Pattern: "/v1beta/fileSearchStores/{store}"},
			{Method: http.MethodDelete, Pattern: "/v1beta/fileSearchStores/{store}"},
			{Method: http.MethodPost, Pattern: "/v1beta/fileSearchStores/{store}:importFile"},
			{Method: http.MethodPost, Pattern: "/v1beta/fileSearchStores/{store}:uploadToFileSearchStore"},
			{Method: http.MethodPost, Pattern: "/upload/v1beta/fileSearchStores/{store}:uploadToFileSearchStore"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiFileSearchStores, RequestFormat: EndpointGeminiFileSearchStores, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiFileSearchStores, RequestFormat: "/v1beta/fileSearchStores/{store}:importFile", Action: ProtocolCapabilityActionImportFile, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiDocuments,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_documents",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/fileSearchStores/{store}/documents", RegisteredHandlerFamily: "gemini_file_search_stores"},
			{Method: http.MethodGet, Pattern: "/v1beta/fileSearchStores/{store}/documents/{document}", RegisteredHandlerFamily: "gemini_file_search_stores"},
			{Method: http.MethodDelete, Pattern: "/v1beta/fileSearchStores/{store}/documents/{document}", RegisteredHandlerFamily: "gemini_file_search_stores"},
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
			{Method: http.MethodGet, Pattern: "/v1beta/fileSearchStores/{store}/operations/{operation}", RegisteredHandlerFamily: "gemini_file_search_stores"},
			{Method: http.MethodGet, Pattern: "/v1beta/operations"},
			{Method: http.MethodGet, Pattern: "/v1beta/operations/*subpath"},
			{Method: http.MethodDelete, Pattern: "/v1beta/operations/*subpath"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiOperations, RequestFormat: EndpointGeminiOperations, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiUploadOperations,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_upload_operations",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/fileSearchStores/{store}/upload/operations/{operation}", RegisteredHandlerFamily: "gemini_file_search_stores"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiUploadOperations, RequestFormat: "/v1beta/fileSearchStores/{store}/upload/operations/{operation}", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiEmbeddings,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_embeddings",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1beta/embeddings"},
			{Method: http.MethodPost, Pattern: "/v1/models/{model}:embedContent", RegisteredHandlerFamily: "gateway_v1_models_action"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/{model}:embedContent", RegisteredHandlerFamily: "gemini_models"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/{model}:batchEmbedContents", RegisteredHandlerFamily: "gemini_models"},
			{Method: http.MethodPost, Pattern: "/v1beta/models/{model}:asyncBatchEmbedContent", RegisteredHandlerFamily: "gemini_models"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiEmbeddings, RequestFormat: EndpointGeminiEmbeddings, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiEmbeddings, RequestFormat: "/v1/models/{model}:embedContent", Action: ProtocolCapabilityActionGeminiEmbedContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
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
		CanonicalEndpoint: EndpointGeminiCorpora,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_corpora",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/corpora"},
			{Method: http.MethodPost, Pattern: "/v1beta/corpora"},
			{Method: http.MethodGet, Pattern: "/v1beta/corpora/{corpus}"},
			{Method: http.MethodDelete, Pattern: "/v1beta/corpora/{corpus}"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiCorpora, RequestFormat: EndpointGeminiCorpora, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiCorporaOperations,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_corpora_operations",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/corpora/{corpus}/operations/{operation}", RegisteredHandlerFamily: "gemini_corpora_operations"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiCorporaOperations, RequestFormat: "/v1beta/corpora/{corpus}/operations/{operation}", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiCorporaPermissions,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_corpora_permissions",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/corpora/{corpus}/permissions", RegisteredHandlerFamily: "gemini_corpora_permissions"},
			{Method: http.MethodPost, Pattern: "/v1beta/corpora/{corpus}/permissions", RegisteredHandlerFamily: "gemini_corpora_permissions"},
			{Method: http.MethodGet, Pattern: "/v1beta/corpora/{corpus}/permissions/{permission}", RegisteredHandlerFamily: "gemini_corpora_permissions"},
			{Method: http.MethodPatch, Pattern: "/v1beta/corpora/{corpus}/permissions/{permission}", RegisteredHandlerFamily: "gemini_corpora_permissions"},
			{Method: http.MethodDelete, Pattern: "/v1beta/corpora/{corpus}/permissions/{permission}", RegisteredHandlerFamily: "gemini_corpora_permissions"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiCorporaPermissions, RequestFormat: "/v1beta/corpora/{corpus}/permissions/{permission}", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiDynamic,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_dynamic",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1beta/dynamic/{dynamic}:generateContent"},
			{Method: http.MethodPost, Pattern: "/v1beta/dynamic/{dynamic}:streamGenerateContent"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiDynamic, RequestFormat: "/v1beta/dynamic/{dynamic}:generateContent", Action: ProtocolCapabilityActionGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiDynamic, RequestFormat: "/v1beta/dynamic/{dynamic}:streamGenerateContent", Action: ProtocolCapabilityActionStreamGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiGeneratedFiles,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_generated_files",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/generatedFiles"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiGeneratedFiles, RequestFormat: EndpointGeminiGeneratedFiles, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiGeneratedFilesOperations,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_generated_files_operations",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/generatedFiles/{generatedFile}/operations/{operation}", RegisteredHandlerFamily: "gemini_generated_files_operations"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiGeneratedFilesOperations, RequestFormat: "/v1beta/generatedFiles/{generatedFile}/operations/{operation}", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiModelOperations,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_model_operations",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/models/{model}/operations", RegisteredHandlerFamily: "gemini_model_operations"},
			{Method: http.MethodGet, Pattern: "/v1beta/models/{model}/operations/{operation}", RegisteredHandlerFamily: "gemini_model_operations"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiModelOperations, RequestFormat: "/v1beta/models/{model}/operations/{operation}", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiTunedModels,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_tuned_models",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/tunedModels"},
			{Method: http.MethodPost, Pattern: "/v1beta/tunedModels"},
			{Method: http.MethodGet, Pattern: "/v1beta/tunedModels/{tunedModel}"},
			{Method: http.MethodPatch, Pattern: "/v1beta/tunedModels/{tunedModel}"},
			{Method: http.MethodDelete, Pattern: "/v1beta/tunedModels/{tunedModel}"},
			{Method: http.MethodPost, Pattern: "/v1beta/tunedModels/{tunedModel}:generateContent"},
			{Method: http.MethodPost, Pattern: "/v1beta/tunedModels/{tunedModel}:streamGenerateContent"},
			{Method: http.MethodPost, Pattern: "/v1beta/tunedModels/{tunedModel}:batchGenerateContent"},
			{Method: http.MethodPost, Pattern: "/v1beta/tunedModels/{tunedModel}:asyncBatchEmbedContent"},
			{Method: http.MethodPost, Pattern: "/v1beta/tunedModels/{tunedModel}:transferOwnership"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiTunedModels, RequestFormat: EndpointGeminiTunedModels, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiTunedModels, RequestFormat: "/v1beta/tunedModels/{tunedModel}:generateContent", Action: ProtocolCapabilityActionGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiTunedModels, RequestFormat: "/v1beta/tunedModels/{tunedModel}:streamGenerateContent", Action: ProtocolCapabilityActionStreamGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiTunedModels, RequestFormat: "/v1beta/tunedModels/{tunedModel}:batchGenerateContent", Action: ProtocolCapabilityActionBatchGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiTunedModels, RequestFormat: "/v1beta/tunedModels/{tunedModel}:asyncBatchEmbedContent", Action: ProtocolCapabilityActionGeminiAsyncEmbedding, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointGeminiTunedModels, RequestFormat: "/v1beta/tunedModels/{tunedModel}:transferOwnership", Action: ProtocolCapabilityActionTransferOwnership, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiTunedModelsPermissions,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_tuned_models_permissions",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/tunedModels/{tunedModel}/permissions", RegisteredHandlerFamily: "gemini_tuned_models_permissions"},
			{Method: http.MethodPost, Pattern: "/v1beta/tunedModels/{tunedModel}/permissions", RegisteredHandlerFamily: "gemini_tuned_models_permissions"},
			{Method: http.MethodGet, Pattern: "/v1beta/tunedModels/{tunedModel}/permissions/{permission}", RegisteredHandlerFamily: "gemini_tuned_models_permissions"},
			{Method: http.MethodPatch, Pattern: "/v1beta/tunedModels/{tunedModel}/permissions/{permission}", RegisteredHandlerFamily: "gemini_tuned_models_permissions"},
			{Method: http.MethodDelete, Pattern: "/v1beta/tunedModels/{tunedModel}/permissions/{permission}", RegisteredHandlerFamily: "gemini_tuned_models_permissions"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiTunedModelsPermissions, RequestFormat: "/v1beta/tunedModels/{tunedModel}/permissions/{permission}", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
		},
	},
	{
		CanonicalEndpoint: EndpointGeminiTunedModelsOperations,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_tuned_models_operations",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodGet, Pattern: "/v1beta/tunedModels/{tunedModel}/operations", RegisteredHandlerFamily: "gemini_tuned_models_operations"},
			{Method: http.MethodGet, Pattern: "/v1beta/tunedModels/{tunedModel}/operations/{operation}", RegisteredHandlerFamily: "gemini_tuned_models_operations"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiTunedModelsOperations, RequestFormat: "/v1beta/tunedModels/{tunedModel}/operations/{operation}", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
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
		CanonicalEndpoint: EndpointGeminiLiveAuthTokens,
		SourceProtocol:    PlatformGemini,
		HandlerFamily:     "gemini_live_auth_tokens",
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1alpha/authTokens", RegisteredHandlerFamily: "gemini_live_auth_tokens"},
			{Method: http.MethodPost, Pattern: "/v1beta/live/auth-token", RegisteredHandlerFamily: "gemini_live"},
			{Method: http.MethodPost, Pattern: "/v1beta/live/auth-tokens", RegisteredHandlerFamily: "gemini_live"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointGeminiLiveAuthTokens, RequestFormat: "/v1alpha/authTokens", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
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
			{Method: http.MethodPost, Pattern: "/v1/vertex/models/*modelAction", RegisteredHandlerFamily: "vertex_models_simplified"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointVertexSyncModels, RequestFormat: EndpointVertexSyncModels, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointVertexSyncModels, RequestFormat: "/v1/projects/{project}/locations/{location}/publishers/google/models/{model}:generateContent", Action: ProtocolCapabilityActionGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointVertexSyncModels, RequestFormat: "/v1/projects/{project}/locations/{location}/publishers/google/models/{model}:streamGenerateContent", Action: ProtocolCapabilityActionStreamGenerateContent, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointVertexSyncModels, RequestFormat: "/v1/projects/{project}/locations/{location}/publishers/google/models/{model}:countTokens", Action: ProtocolCapabilityActionGeminiCountTokens, SourceProtocol: PlatformGemini, RuntimePlatform: PlatformGemini, Mode: ProtocolCapabilityNativePassthrough},
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
			{Method: http.MethodGet, Pattern: "/v1/vertex/batchPredictionJobs", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
			{Method: http.MethodPost, Pattern: "/v1/vertex/batchPredictionJobs", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
			{Method: http.MethodGet, Pattern: "/v1/vertex/batchPredictionJobs/*subpath", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
			{Method: http.MethodPost, Pattern: "/v1/vertex/batchPredictionJobs/*subpath", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
			{Method: http.MethodDelete, Pattern: "/v1/vertex/batchPredictionJobs/*subpath", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
			{Method: http.MethodGet, Pattern: "/vertex-batch/jobs", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
			{Method: http.MethodPost, Pattern: "/vertex-batch/jobs", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
			{Method: http.MethodGet, Pattern: "/vertex-batch/jobs/*subpath", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
			{Method: http.MethodPost, Pattern: "/vertex-batch/jobs/*subpath", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
			{Method: http.MethodDelete, Pattern: "/vertex-batch/jobs/*subpath", RegisteredHandlerFamily: "vertex_batch_prediction_jobs_simplified"},
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
	case ProtocolCapabilityActionGenerateAnswer:
		return ProtocolCapabilityActionGenerateAnswer
	case ProtocolCapabilityActionStreamGenerateContent:
		return ProtocolCapabilityActionStreamGenerateContent
	case ProtocolCapabilityActionBatchGenerateContent:
		return ProtocolCapabilityActionBatchGenerateContent
	case ProtocolCapabilityActionImportFile:
		return ProtocolCapabilityActionImportFile
	case ProtocolCapabilityActionTransferOwnership:
		return ProtocolCapabilityActionTransferOwnership
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
		literalCount, paramCount, wildcard := publicEndpointRouteSegmentWeights(segment)
		score += 2
		score += literalCount * 10
		score += paramCount * 5
		if wildcard {
			score++
		}
	}
	return score
}

func publicEndpointRouteSegmentWeights(segment string) (literalCount int, paramCount int, wildcard bool) {
	for index := 0; index < len(segment); {
		switch segment[index] {
		case '*':
			if index == 0 {
				return literalCount, paramCount, true
			}
		case '{':
			if end := strings.IndexByte(segment[index:], '}'); end >= 0 {
				paramCount++
				index += end + 1
				continue
			}
		case ':':
			if index > 0 && segment[index-1] == '}' {
				literalCount++
				index++
				continue
			}
			if index+1 < len(segment) && isRouteParamStart(segment[index+1]) {
				paramCount++
				end := index + 2
				for end < len(segment) && isRouteParamContinue(segment[end]) {
					end++
				}
				index = end
				continue
			}
		}
		literalCount++
		index++
	}
	return literalCount, paramCount, false
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
