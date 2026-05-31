package service

import "net/http"

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
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformDeepSeek, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: EndpointMessages, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformAntigravity, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformAnthropic, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointMessages, RequestFormat: "/v1/messages/count_tokens", Action: ProtocolCapabilityActionCountTokens, SourceProtocol: PlatformAnthropic, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityReject},
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
			{Method: http.MethodPost, Pattern: "/openrouter/v1/chat/completions"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformDeepSeek, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointChatCompletions, RequestFormat: EndpointChatCompletions, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenRouter, Mode: ProtocolCapabilityNativePassthrough},
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
		CanonicalEndpoint: EndpointEmbeddings,
		SourceProtocol:    PlatformOpenAI,
		HandlerFamily:     "openai_embeddings",
		NormalizePrefixes: []string{"/openai"},
		Routes: []PublicEndpointRoute{
			{Method: http.MethodPost, Pattern: "/v1/embeddings"},
			{Method: http.MethodPost, Pattern: "/embeddings"},
		},
		Capabilities: []PublicProtocolCapability{
			{InboundEndpoint: EndpointEmbeddings, RequestFormat: EndpointEmbeddings, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityNativePassthrough},
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
			{InboundEndpoint: EndpointResponses, RequestFormat: EndpointResponses, Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformGrok, Mode: ProtocolCapabilityNativePassthrough},
			{InboundEndpoint: EndpointResponses, RequestFormat: EndpointResponses, Action: ProtocolCapabilityActionWebSocket, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenAI, Mode: ProtocolCapabilityNativePassthrough},
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
			{Method: http.MethodGet, Pattern: "/openrouter/v1/models", RegisteredHandlerFamily: "gateway_models"},
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
			{InboundEndpoint: EndpointGeminiModels, RequestFormat: "/v1/models", Action: ProtocolCapabilityActionDefault, SourceProtocol: PlatformOpenAI, RuntimePlatform: PlatformOpenRouter, Mode: ProtocolCapabilityNativePassthrough},
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

func flattenPublicProtocolCapabilities(entries []PublicEndpointRegistryEntry) []PublicProtocolCapability {
	capabilities := make([]PublicProtocolCapability, 0)
	for _, entry := range entries {
		capabilities = append(capabilities, entry.Capabilities...)
	}
	return capabilities
}
