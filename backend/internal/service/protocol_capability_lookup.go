package service

import (
	"net/http"
	"strings"
)

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
