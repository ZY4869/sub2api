package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

func (s *GatewayService) resolveCanonicalRequestModel(ctx context.Context, requestedModel string) string {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		return ""
	}
	if s != nil && s.modelRegistryService != nil {
		if resolved, ok, err := s.modelRegistryService.ResolveModel(ctx, requestedModel); err == nil && ok && resolved != "" {
			return resolved
		}
	}
	if resolved, ok := modelregistry.ResolveToCanonicalID(requestedModel); ok {
		return resolved
	}
	return normalizeRegistryID(requestedModel)
}

func (s *GatewayService) resolveUpstreamModelID(ctx context.Context, account *Account, requestedModel string) string {
	requestedModel = s.resolveCanonicalRequestModel(ctx, requestedModel)
	if requestedModel == "" || account == nil {
		return requestedModel
	}
	route := registryRouteForAccount(account)
	if s != nil && s.modelRegistryService != nil {
		if resolved, ok, err := s.modelRegistryService.ResolveProtocolModel(ctx, requestedModel, route); err == nil && ok && resolved != "" {
			return resolved
		}
	}
	if resolved, ok := modelregistry.ResolveToProtocolID(requestedModel, route); ok {
		return resolved
	}
	return requestedModel
}

func registryRouteForAccount(account *Account) string {
	if account == nil {
		return "default"
	}
	if IsProtocolGatewayAccount(account) {
		return ProtocolGatewayRegistryRoute(account)
	}
	switch account.Platform {
	case PlatformAnthropic:
		if account.Type == AccountTypeAPIKey {
			return "anthropic_apikey"
		}
		return "anthropic_oauth"
	case PlatformOpenAI:
		return "openai"
	case PlatformGemini:
		return "gemini"
	case PlatformAntigravity:
		return "antigravity"
	case PlatformSora:
		return "sora"
	default:
		return "default"
	}
}
