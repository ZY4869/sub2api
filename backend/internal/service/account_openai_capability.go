package service

import (
	"encoding/json"
	"strings"
)

type OpenAIEndpointCapability string

const (
	OpenAIEndpointCapabilityChatCompletions OpenAIEndpointCapability = "chat_completions"
	OpenAIEndpointCapabilityEmbeddings      OpenAIEndpointCapability = "embeddings"

	openAIEndpointCapabilitiesCredentialKey = "openai_capabilities"
)

func NormalizeOpenAIEndpointCapability(value string) OpenAIEndpointCapability {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case string(OpenAIEndpointCapabilityChatCompletions), "chat", "chat.completions", "chat/completions":
		return OpenAIEndpointCapabilityChatCompletions
	case string(OpenAIEndpointCapabilityEmbeddings), "embedding", "openai.embeddings":
		return OpenAIEndpointCapabilityEmbeddings
	default:
		return ""
	}
}

func (a *Account) GetOpenAIEndpointCapabilities() []OpenAIEndpointCapability {
	if a == nil || a.Credentials == nil {
		return nil
	}
	raw, ok := a.Credentials[openAIEndpointCapabilitiesCredentialKey]
	if !ok {
		return nil
	}
	values := parseOpenAIEndpointCapabilityValues(raw)
	if len(values) == 0 {
		return nil
	}
	seen := make(map[OpenAIEndpointCapability]struct{}, len(values))
	result := make([]OpenAIEndpointCapability, 0, len(values))
	for _, value := range values {
		capability := NormalizeOpenAIEndpointCapability(value)
		if capability == "" {
			continue
		}
		if _, exists := seen[capability]; exists {
			continue
		}
		seen[capability] = struct{}{}
		result = append(result, capability)
	}
	return result
}

func SupportsOpenAIEndpointCapability(account *Account, capability OpenAIEndpointCapability) bool {
	if capability == "" {
		return true
	}
	resolved := ResolveProtocolGatewayInboundAccount(account, PlatformOpenAI)
	if resolved == nil {
		return false
	}
	if !supportsOpenAIEndpointCapabilityByAccountKind(resolved, capability) {
		return false
	}
	configured := resolved.GetOpenAIEndpointCapabilities()
	if len(configured) == 0 {
		return true
	}
	for _, allowed := range configured {
		if allowed == capability {
			return true
		}
	}
	return false
}

func supportsOpenAIEndpointCapabilityByAccountKind(account *Account, capability OpenAIEndpointCapability) bool {
	if account == nil {
		return false
	}
	switch capability {
	case OpenAIEndpointCapabilityChatCompletions:
		return account.IsOpenAITextCompatible()
	case OpenAIEndpointCapabilityEmbeddings:
		return account.IsOpenAIApiKey()
	default:
		return false
	}
}

func parseOpenAIEndpointCapabilityValues(raw any) []string {
	switch typed := raw.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				result = append(result, value)
			}
		}
		return result
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		if strings.HasPrefix(trimmed, "[") {
			var values []string
			if err := json.Unmarshal([]byte(trimmed), &values); err == nil {
				return values
			}
		}
		return strings.FieldsFunc(trimmed, func(r rune) bool {
			return r == ',' || r == ';' || r == '|' || r == ' '
		})
	default:
		return nil
	}
}
