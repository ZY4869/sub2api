package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const (
	PublicImageRouteFamily               = "public_images"
	PublicImageToolRouteFamily           = "responses_image_tool"
	PublicImageToolRouteReason           = "responses_image_tool"
	PublicImageToolRouteReasonRejected   = "responses_image_tool_platform_rejected"
	PublicImageRouteReasonModelProvider  = "model_provider_match"
	PublicImageRouteReasonSingleProvider = "single_provider_fallback"
	PublicImageRouteReasonAmbiguousModel = "ambiguous_model_provider"
	PublicImageRouteReasonMissingModel   = "missing_model_provider"
	PublicImageRouteReasonToolOnlyModel  = "tool_only_model"
	PublicImageRouteReasonUnsupported    = "unsupported_image_action"
	PublicImageRouteReasonUnknownModel   = "unknown_image_model"
)

type PublicImageRouteDecision struct {
	Supported        bool
	StatusCode       int
	ErrorType        string
	ErrorCode        string
	ErrorMessage     string
	ImageRouteFamily string
	ImageAction      string
	ResolvedProvider string
	DisplayModelID   string
	TargetModelID    string
	UpstreamEndpoint string
	RequestFormat    string
	RouteReason      string
}

type publicImageModelSupport struct {
	Provider  string
	Entry     APIKeyPublicModelEntry
	Native    bool
	ToolOnly  bool
	CanHandle bool
}

func (s *GatewayService) ResolvePublicImageRoute(
	ctx context.Context,
	apiKey *APIKey,
	inboundEndpoint string,
	requestedModel string,
) (PublicImageRouteDecision, error) {
	inboundEndpoint = NormalizeInboundEndpoint(inboundEndpoint)
	requestedModel = strings.TrimSpace(requestedModel)

	decision := PublicImageRouteDecision{
		StatusCode:       http.StatusBadRequest,
		ErrorType:        "invalid_request_error",
		ErrorCode:        GatewayReasonPublicEndpointUnsupported,
		ImageRouteFamily: PublicImageRouteFamily,
		ImageAction:      publicImageActionForEndpoint(inboundEndpoint),
		DisplayModelID:   requestedModel,
		RequestFormat:    PublicEndpointRequestFormatForAction(inboundEndpoint, ProtocolCapabilityActionDefault),
	}
	if decision.RequestFormat == "" {
		decision.RequestFormat = inboundEndpoint
	}

	switch inboundEndpoint {
	case EndpointImagesGen, EndpointImagesEdits:
	default:
		decision.ErrorCode = GatewayReasonRouteMismatch
		decision.ErrorMessage = "Unsupported public image endpoint"
		decision.RouteReason = GatewayReasonRouteMismatch
		return decision, nil
	}

	if apiKey == nil {
		decision.StatusCode = http.StatusUnauthorized
		decision.ErrorType = "authentication_error"
		decision.ErrorCode = "invalid_api_key"
		decision.ErrorMessage = "Invalid API key"
		decision.RouteReason = "missing_api_key"
		return decision, nil
	}

	if requestedModel == "" {
		fallbackProvider := resolveSinglePublicImageProvider(apiKey, inboundEndpoint)
		if fallbackProvider == "" {
			decision.ErrorMessage = "model is required when the API key can route image requests to multiple providers"
			decision.RouteReason = PublicImageRouteReasonMissingModel
			return decision, nil
		}
		decision.Supported = true
		decision.ResolvedProvider = fallbackProvider
		decision.UpstreamEndpoint = publicImageUpstreamEndpoint(fallbackProvider, inboundEndpoint)
		decision.RouteReason = PublicImageRouteReasonSingleProvider
		return decision, nil
	}

	supports, err := s.resolvePublicImageModelSupports(ctx, apiKey, requestedModel, inboundEndpoint)
	if err != nil {
		return PublicImageRouteDecision{}, err
	}

	handleable := make(map[string]publicImageModelSupport)
	toolOnly := make(map[string]publicImageModelSupport)
	unsupported := make(map[string]publicImageModelSupport)
	knownProviders := make(map[string]publicImageModelSupport)
	for _, support := range supports {
		if _, exists := knownProviders[support.Provider]; !exists {
			knownProviders[support.Provider] = support
		}
		switch {
		case support.Native && support.CanHandle:
			handleable[support.Provider] = support
		case support.ToolOnly:
			toolOnly[support.Provider] = support
		case support.Native && !support.CanHandle:
			unsupported[support.Provider] = support
		}
	}

	switch len(handleable) {
	case 1:
		for provider, support := range handleable {
			decision.Supported = true
			decision.ResolvedProvider = provider
			decision.DisplayModelID = requestedModel
			decision.TargetModelID = strings.TrimSpace(support.Entry.SourceID)
			decision.UpstreamEndpoint = publicImageUpstreamEndpoint(provider, inboundEndpoint)
			decision.RouteReason = PublicImageRouteReasonModelProvider
			return decision, nil
		}
	case 0:
	default:
		if fallbackProvider := resolveSinglePublicImageProvider(apiKey, inboundEndpoint); fallbackProvider != "" {
			if support, ok := handleable[fallbackProvider]; ok {
				decision.Supported = true
				decision.ResolvedProvider = fallbackProvider
				decision.DisplayModelID = requestedModel
				decision.TargetModelID = strings.TrimSpace(support.Entry.SourceID)
				decision.UpstreamEndpoint = publicImageUpstreamEndpoint(fallbackProvider, inboundEndpoint)
				decision.RouteReason = PublicImageRouteReasonSingleProvider
				return decision, nil
			}
		}
		decision.RouteReason = PublicImageRouteReasonAmbiguousModel
		decision.ErrorMessage = fmt.Sprintf("model %q matches multiple image providers; use a provider-specific image route or narrow the model policy", requestedModel)
		return decision, nil
	}

	if len(toolOnly) > 0 {
		decision.RouteReason = PublicImageRouteReasonToolOnlyModel
		decision.ErrorMessage = fmt.Sprintf("model %q only supports image generation via /v1/responses tools:[{type:\"image_generation\"}]", requestedModel)
		return decision, nil
	}

	if len(unsupported) == 1 {
		for provider, support := range unsupported {
			decision.StatusCode = http.StatusBadRequest
			decision.ErrorType = "invalid_request_error"
			decision.ErrorCode = GatewayReasonUnsupportedAction
			decision.RouteReason = PublicImageRouteReasonUnsupported
			decision.ResolvedProvider = provider
			decision.DisplayModelID = requestedModel
			decision.TargetModelID = strings.TrimSpace(support.Entry.SourceID)
			switch provider {
			case PlatformGemini:
				decision.ErrorMessage = fmt.Sprintf("provider %q does not support %s on the public image endpoint", provider, decision.RequestFormat)
			default:
				decision.ErrorMessage = fmt.Sprintf("model %q is not available for %s on provider %q", requestedModel, decision.RequestFormat, provider)
			}
			return decision, nil
		}
	}

	if len(knownProviders) > 0 {
		decision.RouteReason = PublicImageRouteReasonUnsupported
		decision.ErrorCode = GatewayReasonPublicEndpointUnsupported
		decision.ErrorMessage = fmt.Sprintf("model %q is not available on the native image endpoints", requestedModel)
		return decision, nil
	}

	decision.RouteReason = PublicImageRouteReasonUnknownModel
	decision.ErrorMessage = fmt.Sprintf("model %q is not available for the public image endpoints", requestedModel)
	return decision, nil
}

func (s *GatewayService) resolvePublicImageModelSupports(
	ctx context.Context,
	apiKey *APIKey,
	requestedModel string,
	inboundEndpoint string,
) ([]publicImageModelSupport, error) {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		return nil, nil
	}

	lookupPlatforms := []struct {
		LookupPlatform string
		Provider       string
	}{
		{LookupPlatform: PlatformOpenAI, Provider: PlatformOpenAI},
		{LookupPlatform: PlatformCopilot, Provider: PlatformOpenAI},
		{LookupPlatform: PlatformGrok, Provider: PlatformGrok},
		{LookupPlatform: PlatformGemini, Provider: PlatformGemini},
	}

	seenProviders := make(map[string]struct{}, len(lookupPlatforms))
	supports := make([]publicImageModelSupport, 0, len(lookupPlatforms))
	for _, item := range lookupPlatforms {
		entry, ok, err := s.FindAPIKeyPublicModel(ctx, apiKey, item.LookupPlatform, requestedModel)
		if err != nil {
			return nil, err
		}
		if !ok || entry == nil {
			continue
		}
		if _, exists := seenProviders[item.Provider]; exists {
			continue
		}
		native, toolOnly := s.resolvePublicImageCapability(ctx, entry)
		supports = append(supports, publicImageModelSupport{
			Provider:  item.Provider,
			Entry:     *entry,
			Native:    native,
			ToolOnly:  toolOnly && !native,
			CanHandle: native && publicImageProviderSupportsAction(item.Provider, inboundEndpoint),
		})
		seenProviders[item.Provider] = struct{}{}
	}
	return supports, nil
}

func (s *GatewayService) resolvePublicImageCapability(ctx context.Context, entry *APIKeyPublicModelEntry) (bool, bool) {
	if entry == nil {
		return false, false
	}
	lookupID := strings.TrimSpace(firstNonEmptyString(entry.SourceID, entry.PublicID, entry.AliasID))
	if lookupID != "" && s != nil && s.modelRegistryService != nil {
		if detail, err := s.modelRegistryService.GetDetail(ctx, lookupID); err == nil && detail != nil {
			native := containsAnyRegistryValue(detail.Capabilities, "image_generation")
			toolOnly := containsAnyRegistryValue(detail.Capabilities, "image_generation_tool")
			if !native && !toolOnly {
				native = containsAnyRegistryValue(detail.Modalities, "image") || inferModelMode(detail.ID, "") == "image"
			}
			return native, toolOnly
		}
	}
	return inferModelMode(lookupID, "") == "image", false
}

func resolveSinglePublicImageProvider(apiKey *APIKey, inboundEndpoint string) string {
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(bindings))
	providers := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		if binding.Group == nil || !binding.Group.IsActive() {
			continue
		}
		provider := normalizePublicImageProvider(binding.Group.Platform)
		if provider == "" || !publicImageProviderSupportsAction(provider, inboundEndpoint) {
			continue
		}
		if _, exists := seen[provider]; exists {
			continue
		}
		seen[provider] = struct{}{}
		providers = append(providers, provider)
	}
	if len(providers) != 1 {
		return ""
	}
	return providers[0]
}

func publicImageActionForEndpoint(inboundEndpoint string) string {
	switch NormalizeInboundEndpoint(inboundEndpoint) {
	case EndpointImagesEdits:
		return "edits"
	default:
		return "generations"
	}
}

func normalizePublicImageProvider(platform string) string {
	normalized := strings.TrimSpace(strings.ToLower(platform))
	switch {
	case IsOpenAIFamily(normalized):
		return PlatformOpenAI
	case normalized == PlatformGrok:
		return PlatformGrok
	case normalized == PlatformGemini:
		return PlatformGemini
	default:
		return ""
	}
}

func publicImageProviderSupportsAction(provider string, inboundEndpoint string) bool {
	decision := DecideProtocolCapability(provider, inboundEndpoint, ProtocolCapabilityActionDefault)
	return decision.Supported && decision.Mode == ProtocolCapabilityNativePassthrough
}

func publicImageUpstreamEndpoint(provider string, inboundEndpoint string) string {
	switch provider {
	case PlatformGemini:
		if NormalizeInboundEndpoint(inboundEndpoint) == EndpointImagesGen {
			return "/v1beta/openai/images/generations"
		}
		return ""
	case PlatformOpenAI, PlatformGrok:
		return NormalizeInboundEndpoint(inboundEndpoint)
	default:
		return ""
	}
}
