package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtocolGatewayRequestFormatsMatchDescriptors(t *testing.T) {
	for protocol, descriptor := range protocolGatewayDescriptors {
		require.ElementsMatch(t, ProtocolGatewayRequestFormats(protocol), descriptor.RequestFormats, protocol)
	}
}

func TestProtocolCapabilityMatrixNormalizesRequestFormats(t *testing.T) {
	for _, capability := range publicProtocolCapabilityMatrix {
		require.Equal(t, capability.InboundEndpoint, NormalizeInboundEndpoint(capability.RequestFormat), capability.RequestFormat)
	}
}

func TestLookupProtocolCapability(t *testing.T) {
	tests := []struct {
		name            string
		runtimePlatform string
		inboundEndpoint string
		action          string
		wantMode        ProtocolCapabilityMode
		wantOK          bool
	}{
		{name: "anthropic native messages", runtimePlatform: PlatformAnthropic, inboundEndpoint: EndpointMessages, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "openai compat messages", runtimePlatform: PlatformOpenAI, inboundEndpoint: EndpointMessages, wantMode: ProtocolCapabilityCompatTranslate, wantOK: true},
		{name: "grok rejects messages", runtimePlatform: PlatformGrok, inboundEndpoint: EndpointMessages, wantMode: ProtocolCapabilityReject, wantOK: true},
		{name: "anthropic count tokens native", runtimePlatform: PlatformAnthropic, inboundEndpoint: EndpointMessages, action: ProtocolCapabilityActionCountTokens, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "antigravity count tokens rejected", runtimePlatform: PlatformAntigravity, inboundEndpoint: EndpointMessages, action: ProtocolCapabilityActionCountTokens, wantMode: ProtocolCapabilityReject, wantOK: true},
		{name: "grok websocket responses rejected", runtimePlatform: PlatformGrok, inboundEndpoint: EndpointResponses, action: ProtocolCapabilityActionWebSocket, wantMode: ProtocolCapabilityReject, wantOK: true},
		{name: "gemini batch alias uses batch capability", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/models/gemini-2.5-pro:batchGenerateContent", action: ProtocolCapabilityActionBatchGenerateContent, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini v1 generate content supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1/models/gemini-2.5-pro:generateContent", action: ProtocolCapabilityActionGenerateContent, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini embed action uses embeddings capability", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/models/gemini-2.5-pro:embedContent", action: ProtocolCapabilityActionGeminiEmbedContent, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini live websocket supported", runtimePlatform: PlatformGemini, inboundEndpoint: EndpointGeminiLive, action: ProtocolCapabilityActionWebSocket, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini auth tokens supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1alpha/authTokens", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini openai compat supported", runtimePlatform: PlatformGemini, inboundEndpoint: EndpointGeminiOpenAICompat, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini corpora supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/corpora/default-corpus", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini corpora operations supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/corpora/default-corpus/operations/op-1", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini corpora permissions supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/corpora/default-corpus/permissions/perm-1", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini dynamic generate supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/dynamic/configs-demo:generateContent", action: ProtocolCapabilityActionGenerateContent, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini dynamic stream supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/dynamic/configs-demo:streamGenerateContent", action: ProtocolCapabilityActionStreamGenerateContent, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini file search import supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/fileSearchStores/default-store:importFile", action: ProtocolCapabilityActionImportFile, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini generate answer supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/models/gemini-2.5-pro:generateAnswer", action: ProtocolCapabilityActionGenerateAnswer, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini generated files supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/generatedFiles", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini generated file operations supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/generatedFiles/file-1/operations/op-1", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini model operations supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/models/gemini-2.5-pro/operations/op-1", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini tuned models supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/tunedModels/tuned-1", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini tuned model generate supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/tunedModels/tuned-1:generateContent", action: ProtocolCapabilityActionGenerateContent, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini tuned model batch supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/tunedModels/tuned-1:batchGenerateContent", action: ProtocolCapabilityActionBatchGenerateContent, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini tuned model async embedding supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/tunedModels/tuned-1:asyncBatchEmbedContent", action: ProtocolCapabilityActionGeminiAsyncEmbedding, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini tuned model transfer supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/tunedModels/tuned-1:transferOwnership", action: ProtocolCapabilityActionTransferOwnership, wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini tuned model permissions supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/tunedModels/tuned-1/permissions/perm-1", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "gemini tuned model operations supported", runtimePlatform: PlatformGemini, inboundEndpoint: "/v1beta/tunedModels/tuned-1/operations/op-1", wantMode: ProtocolCapabilityNativePassthrough, wantOK: true},
		{name: "antigravity batch alias rejected", runtimePlatform: PlatformAntigravity, inboundEndpoint: "/antigravity/v1beta/models/gemini-2.5-pro:batchGenerateContent", action: ProtocolCapabilityActionBatchGenerateContent, wantMode: ProtocolCapabilityReject, wantOK: true},
		{name: "openai images fall back to reject", runtimePlatform: PlatformOpenAI, inboundEndpoint: EndpointImagesGen, wantMode: ProtocolCapabilityReject, wantOK: true},
		{name: "unknown endpoint stays unknown", runtimePlatform: PlatformOpenAI, inboundEndpoint: "/v1/embeddings", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMode, ok := LookupProtocolCapabilityForAction(tt.runtimePlatform, tt.inboundEndpoint, tt.action)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantMode, gotMode)
		})
	}
}

func TestPublicEndpointRequestFormatForAction(t *testing.T) {
	require.Equal(t, "/v1/messages/count_tokens", PublicEndpointRequestFormatForAction(EndpointMessages, ProtocolCapabilityActionCountTokens))
	require.Equal(t, "/v1beta/models/{model}:batchGenerateContent", PublicEndpointRequestFormatForAction(EndpointGeminiBatches, ProtocolCapabilityActionBatchGenerateContent))
	require.Equal(t, "/v1/models/{model}:embedContent", PublicEndpointRequestFormatForAction(EndpointGeminiEmbeddings, ProtocolCapabilityActionGeminiEmbedContent))
	require.Equal(t, "/v1beta/fileSearchStores/{store}:importFile", PublicEndpointRequestFormatForAction(EndpointGeminiFileSearchStores, ProtocolCapabilityActionImportFile))
	require.Equal(t, "/v1beta/models/{model}:generateAnswer", PublicEndpointRequestFormatForAction(EndpointGeminiModels, ProtocolCapabilityActionGenerateAnswer))
	require.Equal(t, "/v1beta/dynamic/{dynamic}:generateContent", PublicEndpointRequestFormatForAction(EndpointGeminiDynamic, ProtocolCapabilityActionGenerateContent))
	require.Equal(t, "/v1beta/dynamic/{dynamic}:streamGenerateContent", PublicEndpointRequestFormatForAction(EndpointGeminiDynamic, ProtocolCapabilityActionStreamGenerateContent))
	require.Equal(t, "/v1beta/tunedModels/{tunedModel}:generateContent", PublicEndpointRequestFormatForAction(EndpointGeminiTunedModels, ProtocolCapabilityActionGenerateContent))
	require.Equal(t, "/v1beta/tunedModels/{tunedModel}:batchGenerateContent", PublicEndpointRequestFormatForAction(EndpointGeminiTunedModels, ProtocolCapabilityActionBatchGenerateContent))
	require.Equal(t, "/v1beta/tunedModels/{tunedModel}:asyncBatchEmbedContent", PublicEndpointRequestFormatForAction(EndpointGeminiTunedModels, ProtocolCapabilityActionGeminiAsyncEmbedding))
	require.Equal(t, "/v1beta/tunedModels/{tunedModel}:transferOwnership", PublicEndpointRequestFormatForAction(EndpointGeminiTunedModels, ProtocolCapabilityActionTransferOwnership))
	require.Equal(t, "/v1alpha/authTokens", PublicEndpointRequestFormatForAction(EndpointGeminiLiveAuthTokens, ProtocolCapabilityActionDefault))
	require.Equal(t, EndpointResponses, PublicEndpointRequestFormatForAction(EndpointResponses, ProtocolCapabilityActionWebSocket))
}

func TestNormalizeInboundEndpoint_DerivesOpenAIAliasFromRegistry(t *testing.T) {
	require.Equal(t, EndpointGeminiModels, NormalizeInboundEndpoint("/v1/models/gemini-2.5-pro:generateContent"))
	require.Equal(t, EndpointGeminiLiveAuthTokens, NormalizeInboundEndpoint("/v1alpha/authTokens"))
	require.Equal(t, EndpointGeminiBatches, NormalizeInboundEndpoint("/v1beta/models/gemini-2.5-pro:batchGenerateContent"))
	require.Equal(t, EndpointGeminiEmbeddings, NormalizeInboundEndpoint("/v1beta/models/gemini-2.5-pro:embedContent"))
	require.Equal(t, EndpointGeminiFileSearchStores, NormalizeInboundEndpoint("/v1beta/fileSearchStores/default-store:importFile"))
	require.Equal(t, EndpointGeminiDocuments, NormalizeInboundEndpoint("/v1beta/fileSearchStores/default/documents/doc_123"))
	require.Equal(t, EndpointGeminiOperations, NormalizeInboundEndpoint("/v1beta/fileSearchStores/default/operations/op_123"))
	require.Equal(t, EndpointGeminiUploadOperations, NormalizeInboundEndpoint("/v1beta/fileSearchStores/default/upload/operations/op_123"))
	require.Equal(t, EndpointGeminiCorpora, NormalizeInboundEndpoint("/v1beta/corpora/default-corpus"))
	require.Equal(t, EndpointGeminiCorporaOperations, NormalizeInboundEndpoint("/v1beta/corpora/default-corpus/operations/op_123"))
	require.Equal(t, EndpointGeminiCorporaPermissions, NormalizeInboundEndpoint("/v1beta/corpora/default-corpus/permissions/perm_123"))
	require.Equal(t, EndpointGeminiDynamic, NormalizeInboundEndpoint("/v1beta/dynamic/configs-demo:generateContent"))
	require.Equal(t, EndpointGeminiGeneratedFiles, NormalizeInboundEndpoint("/v1beta/generatedFiles"))
	require.Equal(t, EndpointGeminiGeneratedFilesOperations, NormalizeInboundEndpoint("/v1beta/generatedFiles/file_123/operations/op_123"))
	require.Equal(t, EndpointGeminiModelOperations, NormalizeInboundEndpoint("/v1beta/models/gemini-2.5-pro/operations"))
	require.Equal(t, EndpointGeminiModelOperations, NormalizeInboundEndpoint("/v1beta/models/gemini-2.5-pro/operations/op_123"))
	require.Equal(t, EndpointGeminiModels, NormalizeInboundEndpoint("/v1beta/models/gemini-2.5-pro:generateAnswer"))
	require.Equal(t, EndpointGeminiTunedModels, NormalizeInboundEndpoint("/v1beta/tunedModels/tuned_123"))
	require.Equal(t, EndpointGeminiTunedModelsPermissions, NormalizeInboundEndpoint("/v1beta/tunedModels/tuned_123/permissions/perm_123"))
	require.Equal(t, EndpointGeminiTunedModelsOperations, NormalizeInboundEndpoint("/v1beta/tunedModels/tuned_123/operations"))
	require.Equal(t, EndpointGeminiTunedModelsOperations, NormalizeInboundEndpoint("/v1beta/tunedModels/tuned_123/operations/op_123"))
	require.Equal(t, EndpointGeminiTunedModels, NormalizeInboundEndpoint("/v1beta/tunedModels/tuned_123:asyncBatchEmbedContent"))
	require.Equal(t, EndpointGeminiTunedModels, NormalizeInboundEndpoint("/v1beta/tunedModels/tuned_123:streamGenerateContent"))
	require.Equal(t, EndpointResponses, NormalizeInboundEndpoint("/openai/v1/responses"))
	require.Equal(t, EndpointResponses, NormalizeInboundEndpoint("/openai/v1/responses/compact"))
	require.Equal(t, EndpointChatCompletions, NormalizeInboundEndpoint("/openai/v1/chat/completions"))
	require.Equal(t, EndpointGeminiOpenAICompat, NormalizeInboundEndpoint("/v1beta/openai/chat/completions"))
	require.Equal(t, EndpointGeminiOpenAICompat, NormalizeInboundEndpoint("/v1beta/openai/files"))
	require.Equal(t, EndpointGeminiOpenAICompat, NormalizeInboundEndpoint("/v1beta/openai/files/file_123"))
	require.Equal(t, EndpointGeminiOpenAICompat, NormalizeInboundEndpoint("/v1beta/openai/batches/batch_123"))
	require.Equal(t, EndpointGeminiInteractions, NormalizeInboundEndpoint("/v1beta/interactions/sample"))
}

func TestDecideProtocolCapability_GeminiDeprecatedModelActionsRemainUnsupported(t *testing.T) {
	deprecatedActions := []string{
		"generateText",
		"generateMessage",
		"countTextTokens",
		"countMessageTokens",
		"embedText",
		"batchEmbedText",
		"predict",
		"predictLongRunning",
	}

	for _, action := range deprecatedActions {
		t.Run(action, func(t *testing.T) {
			decision := DecideProtocolCapability(PlatformGemini, "/v1beta/models/gemini-2.5-pro:"+action, action)
			require.False(t, decision.Supported)
			require.Equal(t, GatewayReasonUnsupportedAction, decision.Reason)
			require.Equal(t, "gateway.public_endpoint.unsupported_action", decision.MessageKey)
			require.Equal(t, http.StatusBadRequest, decision.StatusCode)
			require.Equal(t, "unsupported_action", decision.InternalMismatchKind)
		})
	}
}
