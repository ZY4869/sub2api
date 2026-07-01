package service

import (
	"net/http"
	"sort"
	"strings"
)

func normalizePublicModelProtocolEndpoints(
	existing []PublicModelProtocolEndpoint,
	legacyProtocols []string,
	source publicModelCatalogMetadataSource,
) []PublicModelProtocolEndpoint {
	endpoints := make([]PublicModelProtocolEndpoint, 0, len(existing)+len(legacyProtocols)*2)
	for _, endpoint := range existing {
		if normalized, ok := normalizePublicModelProtocolEndpoint(endpoint, source); ok {
			endpoints = append(endpoints, normalized)
		}
	}
	for _, protocol := range legacyProtocols {
		endpoints = append(endpoints, defaultPublicModelProtocolEndpoints(protocol, source)...)
		endpoints = append(endpoints, publicModelProtocolCapabilityMatrixEndpoints(protocol, source)...)
	}
	return dedupePublicModelProtocolEndpoints(endpoints)
}

func defaultPublicModelProtocolEndpoints(protocol string, source publicModelCatalogMetadataSource) []PublicModelProtocolEndpoint {
	protocol = publicModelCatalogProtocolFamily(protocol)
	switch protocol {
	case PlatformOpenAI:
		return []PublicModelProtocolEndpoint{
			newPublicModelProtocolEndpoint("openai.chat.completions", PlatformOpenAI, EndpointChatCompletions, http.MethodPost, PublicModelSupportSupported, source),
			newPublicModelProtocolEndpoint("openai.embeddings", PlatformOpenAI, EndpointEmbeddings, http.MethodPost, PublicModelSupportPartial, source),
			newPublicModelProtocolEndpoint("openai.responses", PlatformOpenAI, EndpointResponses, http.MethodPost, PublicModelSupportPartial, source),
			newPublicModelProtocolEndpoint("openai.images.generations", PlatformOpenAI, EndpointImagesGen, http.MethodPost, PublicModelSupportPartial, source),
		}
	case PlatformAnthropic:
		return []PublicModelProtocolEndpoint{
			newPublicModelProtocolEndpoint("anthropic.messages", PlatformAnthropic, EndpointMessages, http.MethodPost, PublicModelSupportSupported, source),
		}
	case PlatformGemini:
		return []PublicModelProtocolEndpoint{
			newPublicModelProtocolEndpoint("gemini.generateContent", PlatformGemini, "/v1beta/models/{model}:generateContent", http.MethodPost, PublicModelSupportSupported, source),
			newPublicModelProtocolEndpoint("gemini.countTokens", PlatformGemini, "/v1beta/models/{model}:countTokens", http.MethodPost, PublicModelSupportPartial, source),
			newPublicModelProtocolEndpoint("gemini.images.generations", PlatformGemini, "/v1beta/openai/images/generations", http.MethodPost, PublicModelSupportPartial, source),
		}
	case PlatformGrok:
		return []PublicModelProtocolEndpoint{
			newPublicModelProtocolEndpoint("grok.chat.completions", PlatformGrok, "/grok/v1/chat/completions", http.MethodPost, PublicModelSupportSupported, source),
			newPublicModelProtocolEndpoint("grok.messages", PlatformGrok, "/grok/v1/messages", http.MethodPost, PublicModelSupportPartial, source),
			newPublicModelProtocolEndpoint("grok.messages.countTokens", PlatformGrok, "/grok/v1/messages/count_tokens", http.MethodPost, PublicModelSupportPartial, source),
			newPublicModelProtocolEndpoint("grok.responses", PlatformGrok, "/grok/v1/responses", http.MethodPost, PublicModelSupportPartial, source),
			newPublicModelProtocolEndpoint("grok.images.generations", PlatformGrok, "/grok/v1/images/generations", http.MethodPost, PublicModelSupportPartial, source),
		}
	default:
		return nil
	}
}

func publicModelProtocolCapabilityMatrixEndpoints(protocol string, source publicModelCatalogMetadataSource) []PublicModelProtocolEndpoint {
	protocol = publicModelCatalogProtocolFamily(protocol)
	if protocol == "" {
		return nil
	}
	endpoints := make([]PublicModelProtocolEndpoint, 0)
	for _, capability := range publicProtocolCapabilityMatrix {
		sourceProtocol := publicModelCatalogProtocolFamily(capability.SourceProtocol)
		runtimePlatform := publicModelCatalogProtocolFamily(capability.RuntimePlatform)
		if protocol != sourceProtocol && protocol != runtimePlatform {
			continue
		}
		if capability.Mode == ProtocolCapabilityReject {
			continue
		}
		key := publicModelEndpointKeyForCapability(protocol, capability)
		if key == "" {
			continue
		}
		endpoints = append(endpoints, PublicModelProtocolEndpoint{
			Key:           key,
			Protocol:      protocol,
			Endpoint:      firstNonEmptyTrimmed(capability.RequestFormat, capability.InboundEndpoint),
			Method:        http.MethodPost,
			Support:       PublicModelSupportPartial,
			Source:        firstNonEmptyTrimmed(source.CapabilitySource, PublicModelCapabilitySourceInferred),
			Verified:      false,
			LastCheckedAt: source.LastCheckedAt,
		})
	}
	return endpoints
}

func publicModelEndpointKeyForCapability(targetProtocol string, capability PublicProtocolCapability) string {
	protocol := publicModelCatalogProtocolFamily(firstNonEmptyTrimmed(targetProtocol, capability.SourceProtocol, capability.RuntimePlatform))
	if protocol == "" {
		return ""
	}
	action := NormalizeProtocolCapabilityAction(capability.Action)
	switch protocol {
	case PlatformOpenAI:
		switch NormalizeInboundEndpoint(capability.InboundEndpoint) {
		case EndpointChatCompletions:
			return "openai.chat.completions"
		case EndpointEmbeddings:
			return "openai.embeddings"
		case EndpointResponses:
			return "openai.responses"
		case EndpointImagesGen:
			return "openai.images.generations"
		case EndpointImagesEdits:
			return "openai.images.edits"
		case EndpointVideosCreate, EndpointVideosGen:
			return "openai.videos.generations"
		}
	case PlatformAnthropic:
		if NormalizeInboundEndpoint(capability.InboundEndpoint) == EndpointMessages {
			if action == ProtocolCapabilityActionCountTokens {
				return "anthropic.messages.countTokens"
			}
			return "anthropic.messages"
		}
	case PlatformGemini:
		switch NormalizeInboundEndpoint(capability.InboundEndpoint) {
		case EndpointImagesGen:
			return "gemini.images.generations"
		case EndpointImagesEdits:
			return "gemini.images.edits"
		}
		switch action {
		case ProtocolCapabilityActionGeminiCountTokens:
			return "gemini.countTokens"
		case ProtocolCapabilityActionGeminiEmbedContent, ProtocolCapabilityActionGeminiBatchEmbeddings, ProtocolCapabilityActionGeminiAsyncEmbedding:
			return "gemini.embedContent"
		case ProtocolCapabilityActionGenerateContent, ProtocolCapabilityActionStreamGenerateContent:
			return "gemini.generateContent"
		}
		switch NormalizeInboundEndpoint(capability.InboundEndpoint) {
		case EndpointGeminiModels:
			return "gemini.generateContent"
		case EndpointGeminiEmbeddings:
			return "gemini.embedContent"
		case EndpointVertexSyncModels:
			if action == ProtocolCapabilityActionGeminiCountTokens {
				return "vertex.countTokens"
			}
			return "vertex.generateContent"
		}
	case PlatformGrok:
		if NormalizeInboundEndpoint(capability.InboundEndpoint) == EndpointMessages {
			if action == ProtocolCapabilityActionCountTokens {
				return "grok.messages.countTokens"
			}
			return "grok.messages"
		}
		switch NormalizeInboundEndpoint(capability.InboundEndpoint) {
		case EndpointChatCompletions:
			return "grok.chat.completions"
		case EndpointResponses:
			return "grok.responses"
		case EndpointImagesGen:
			return "grok.images.generations"
		case EndpointImagesEdits:
			return "grok.images.edits"
		case EndpointVideosCreate, EndpointVideosGen:
			return "grok.videos.generations"
		}
	}
	return ""
}

func newPublicModelProtocolEndpoint(key, protocol, endpoint, method, support string, source publicModelCatalogMetadataSource) PublicModelProtocolEndpoint {
	return PublicModelProtocolEndpoint{
		Key:           key,
		Protocol:      protocol,
		Endpoint:      endpoint,
		Method:        method,
		Support:       support,
		Source:        firstNonEmptyTrimmed(source.CapabilitySource, PublicModelCapabilitySourceInferred),
		Verified:      publicModelCatalogMetadataSourceVerified(source),
		LastCheckedAt: source.LastCheckedAt,
	}
}

func normalizePublicModelProtocolEndpoint(endpoint PublicModelProtocolEndpoint, source publicModelCatalogMetadataSource) (PublicModelProtocolEndpoint, bool) {
	endpoint.Key = strings.TrimSpace(endpoint.Key)
	endpoint.Protocol = publicModelCatalogProtocolFamily(firstNonEmptyTrimmed(endpoint.Protocol, endpoint.Key))
	endpoint.Endpoint = strings.TrimSpace(endpoint.Endpoint)
	if endpoint.Key == "" && endpoint.Protocol != "" && endpoint.Endpoint != "" {
		endpoint.Key = endpoint.Protocol + "." + strings.Trim(endpoint.Endpoint, "/")
	}
	if endpoint.Key == "" || endpoint.Protocol == "" {
		return PublicModelProtocolEndpoint{}, false
	}
	endpoint.Method = firstNonEmptyTrimmed(endpoint.Method, http.MethodPost)
	endpoint.Support = normalizePublicModelSupport(endpoint.Support)
	endpoint.Source = firstNonEmptyTrimmed(endpoint.Source, source.CapabilitySource, PublicModelCapabilitySourceInferred)
	endpoint.Verified = endpoint.Verified || publicModelCatalogMetadataSourceVerified(source)
	endpoint.LastCheckedAt = firstNonEmptyTrimmed(endpoint.LastCheckedAt, source.LastCheckedAt)
	endpoint.Limitations = uniqueTrimmedStringsPreserveCase(endpoint.Limitations)
	return endpoint, true
}

func dedupePublicModelProtocolEndpoints(endpoints []PublicModelProtocolEndpoint) []PublicModelProtocolEndpoint {
	if len(endpoints) == 0 {
		return nil
	}
	byKey := map[string]PublicModelProtocolEndpoint{}
	for _, endpoint := range endpoints {
		normalized, ok := normalizePublicModelProtocolEndpoint(endpoint, publicModelCatalogMetadataSource{})
		if !ok {
			continue
		}
		existing, exists := byKey[normalized.Key]
		if !exists || publicModelProtocolEndpointPreferred(normalized, existing) {
			byKey[normalized.Key] = normalized
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]PublicModelProtocolEndpoint, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result
}

func publicModelProtocolEndpointPreferred(left PublicModelProtocolEndpoint, right PublicModelProtocolEndpoint) bool {
	return publicModelMetadataEntryPreferred(
		left.Source,
		left.Verified,
		left.Support,
		left.LastCheckedAt,
		right.Source,
		right.Verified,
		right.Support,
		right.LastCheckedAt,
	)
}

func publicModelRequestProtocolsFromEndpoints(endpoints []PublicModelProtocolEndpoint, fallback []string) []string {
	values := make([]string, 0, len(endpoints)+len(fallback))
	for _, endpoint := range endpoints {
		if publicModelSupportAllowsSummary(endpoint.Support) {
			values = append(values, endpoint.Protocol)
		}
	}
	if len(values) == 0 {
		values = append(values, fallback...)
	}
	return sortPublicModelProtocols(values)
}

func sortPublicModelProtocols(values []string) []string {
	items := uniqueTrimmedStringsPreserveCase(values)
	sort.SliceStable(items, func(i, j int) bool {
		leftOrder, leftOK := publicModelCatalogProtocolOrder[items[i]]
		rightOrder, rightOK := publicModelCatalogProtocolOrder[items[j]]
		switch {
		case leftOK && rightOK && leftOrder != rightOrder:
			return leftOrder < rightOrder
		case leftOK != rightOK:
			return leftOK
		default:
			return items[i] < items[j]
		}
	})
	return items
}
