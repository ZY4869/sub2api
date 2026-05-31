package service

// Protocol capability constants and public DTO types live here; registry, lookup, and route helpers are split by responsibility.

const (
	EndpointMessages                       = "/v1/messages"
	EndpointChatCompletions                = "/v1/chat/completions"
	EndpointCompletions                    = "/v1/completions"
	EndpointEmbeddings                     = "/v1/embeddings"
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
