package service

import (
	"context"
	"strings"
)

func (s *GatewayService) isModelSupportedByAccountWithContext(ctx context.Context, account *Account, requestedModel string) bool {
	if account.Platform == PlatformAntigravity {
		if strings.TrimSpace(requestedModel) == "" {
			return true
		}
		mapped := mapAntigravityModel(account, requestedModel)
		if mapped == "" {
			return false
		}
		if enabled, ok := ThinkingEnabledFromContext(ctx); ok {
			finalModel := applyThinkingModelSuffix(mapped, enabled)
			if finalModel == mapped {
				return true
			}
			return account.IsModelSupported(finalModel)
		}
		return true
	}
	return s.isModelSupportedByAccount(account, requestedModel)
}
func (s *GatewayService) isModelSupportedByAccount(account *Account, requestedModel string) bool {
	if account != nil && account.Type == AccountTypeBedrock {
		if strings.TrimSpace(requestedModel) == "" {
			return true
		}
		_, ok := ResolveBedrockModelID(account, requestedModel)
		return ok
	}
	if account.Platform == PlatformAntigravity {
		if strings.TrimSpace(requestedModel) == "" {
			return true
		}
		return mapAntigravityModel(account, requestedModel) != ""
	}
	canonicalModel := s.resolveCanonicalRequestModel(context.Background(), requestedModel)
	if canonicalModel == "" {
		return account.IsModelSupported(requestedModel)
	}
	if account.IsModelSupported(canonicalModel) {
		return true
	}
	return account.IsModelSupported(s.resolveUpstreamModelID(context.Background(), account, canonicalModel))
}
