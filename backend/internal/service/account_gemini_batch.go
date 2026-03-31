package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (a *Account) GetExtraBool(key string) bool {
	return a.getExtraBool(key)
}

func (a *Account) getExtraBool(key string) bool {
	if a == nil || a.Extra == nil {
		return false
	}
	value, ok := a.Extra[key]
	if !ok {
		return false
	}
	return parseExtraBool(value)
}

func parseExtraBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		default:
			return false
		}
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i != 0
		}
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	}
	return false
}

func (a *Account) GeminiBatchCapability() string {
	if a == nil || EffectiveProtocol(a) != PlatformGemini {
		return GeminiBatchCapabilityNone
	}
	switch {
	case a.Type == AccountTypeAPIKey && a.GeminiAPIKeyVariant() == GeminiAPIKeyVariantAIStudio:
		return GeminiBatchCapabilityAIStudio
	case a.Type == AccountTypeOAuth && a.IsGeminiVertexAI():
		return GeminiBatchCapabilityVertex
	default:
		return GeminiBatchCapabilityNone
	}
}

func (a *Account) IsGatewayBatchEnabled() bool {
	if a == nil {
		return false
	}
	return a.GetExtraBool(gatewayExtraBatchEnabledKey)
}

func SupportsProtocolGatewayGeminiBatch(account *Account) bool {
	if !IsProtocolGatewayAccount(account) || !account.IsGatewayBatchEnabled() {
		return false
	}
	for _, protocol := range GetAccountGatewayAcceptedProtocols(account) {
		if protocol == PlatformGemini {
			return true
		}
	}
	return false
}

func SupportsAIStudioBatch(account *Account) bool {
	if account == nil {
		return false
	}
	if SupportsProtocolGatewayGeminiBatch(account) {
		return true
	}
	return account.GeminiBatchCapability() == GeminiBatchCapabilityAIStudio
}

func SupportsVertexBatch(account *Account) bool {
	if account == nil {
		return false
	}
	if SupportsProtocolGatewayGeminiBatch(account) {
		return true
	}
	return account.GeminiBatchCapability() == GeminiBatchCapabilityVertex
}

func CanParticipateInAccountQuota(account *Account) bool {
	return account != nil
}

func ShouldExposeAccountQuota(account *Account) bool {
	if account == nil {
		return false
	}
	return CanParticipateInAccountQuota(account) || account.HasAnyQuotaLimit()
}

func formatGeminiBatchConflictMessage(resourceKind string) string {
	switch strings.TrimSpace(resourceKind) {
	case UpstreamResourceKindGeminiFile:
		return "Gemini files in the batch request were created by different accounts"
	case UpstreamResourceKindGeminiBatch:
		return "Gemini batch resource is bound to a different account"
	default:
		return fmt.Sprintf("resource %s is bound to a different account", resourceKind)
	}
}
