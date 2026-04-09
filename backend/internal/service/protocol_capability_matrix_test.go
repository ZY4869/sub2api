package service

import (
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
	require.Equal(t, EndpointResponses, PublicEndpointRequestFormatForAction(EndpointResponses, ProtocolCapabilityActionWebSocket))
}

func TestNormalizeInboundEndpoint_DerivesOpenAIAliasFromRegistry(t *testing.T) {
	require.Equal(t, EndpointResponses, NormalizeInboundEndpoint("/openai/v1/responses"))
	require.Equal(t, EndpointResponses, NormalizeInboundEndpoint("/openai/v1/responses/compact"))
	require.Equal(t, EndpointChatCompletions, NormalizeInboundEndpoint("/openai/v1/chat/completions"))
}
