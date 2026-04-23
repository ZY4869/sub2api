package service

import "strings"

type accountTestRuntimeMeta struct {
	Mode            AccountTestMode
	RuntimePlatform string
	SourceProtocol  string
	SimulatedClient string
	InboundEndpoint string
	CompatPath      string
	TargetProvider  string
	TargetModelID   string
	ResolvedModelID string
}

func buildAccountTestRuntimeMeta(
	account *Account,
	mode AccountTestMode,
	sourceProtocol string,
	targetProvider string,
	targetModelID string,
	resolvedModelID string,
	simulatedClient string,
) accountTestRuntimeMeta {
	runtimePlatform := strings.TrimSpace(RoutingPlatformForAccount(account))
	normalizedSourceProtocol := inferAccountTestSourceProtocol(runtimePlatform, sourceProtocol, resolvedModelID)
	inboundEndpoint, action := accountTestCapabilityLookupTarget(account, normalizedSourceProtocol, resolvedModelID)

	return accountTestRuntimeMeta{
		Mode:            mode,
		RuntimePlatform: runtimePlatform,
		SourceProtocol:  normalizedSourceProtocol,
		SimulatedClient: strings.TrimSpace(simulatedClient),
		InboundEndpoint: inboundEndpoint,
		CompatPath:      accountTestCompatPath(runtimePlatform, normalizedSourceProtocol, inboundEndpoint, action),
		TargetProvider:  NormalizeModelProvider(targetProvider),
		TargetModelID:   strings.TrimSpace(targetModelID),
		ResolvedModelID: strings.TrimSpace(resolvedModelID),
	}
}

func inferAccountTestSourceProtocol(runtimePlatform string, sourceProtocol string, resolvedModelID string) string {
	if normalized := normalizeTestSourceProtocol(sourceProtocol); normalized != "" {
		return normalized
	}

	switch strings.TrimSpace(strings.ToLower(runtimePlatform)) {
	case PlatformOpenAI, PlatformGrok, PlatformCopilot:
		return PlatformOpenAI
	case PlatformAnthropic, PlatformKiro:
		return PlatformAnthropic
	case PlatformGemini:
		return PlatformGemini
	case PlatformAntigravity:
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(resolvedModelID)), "gemini-") {
			return PlatformGemini
		}
		return PlatformAnthropic
	default:
		return ""
	}
}

func accountTestCapabilityLookupTarget(account *Account, sourceProtocol string, resolvedModelID string) (string, string) {
	switch normalizeTestSourceProtocol(sourceProtocol) {
	case PlatformOpenAI:
		if isOpenAIGPTImageProfileModelID(resolvedModelID) {
			return EndpointImagesGen, ProtocolCapabilityActionDefault
		}
		return ResolveOpenAITextRequestFormatForAccount(account, ""), ProtocolCapabilityActionDefault
	case PlatformAnthropic:
		return EndpointMessages, ProtocolCapabilityActionDefault
	case PlatformGemini:
		return EndpointGeminiModels, ProtocolCapabilityActionStreamGenerateContent
	default:
		return "", ""
	}
}

func accountTestCompatPath(runtimePlatform string, sourceProtocol string, inboundEndpoint string, action string) string {
	runtimePlatform = strings.TrimSpace(strings.ToLower(runtimePlatform))
	sourceProtocol = normalizeTestSourceProtocol(sourceProtocol)
	inboundEndpoint = strings.TrimSpace(inboundEndpoint)
	if runtimePlatform == "" || sourceProtocol == "" || inboundEndpoint == "" {
		return ""
	}

	mode, ok := LookupProtocolCapabilityForAction(runtimePlatform, inboundEndpoint, action)
	if !ok {
		return sourceProtocol + "->" + runtimePlatform + ":unknown"
	}
	return sourceProtocol + "->" + runtimePlatform + ":" + string(mode)
}
