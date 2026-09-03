package service

import (
	"encoding/json"
	"strings"
)

type OpenAIEndpointCapability string

const (
	OpenAIEndpointCapabilityChatCompletions OpenAIEndpointCapability = "chat_completions"
	OpenAIEndpointCapabilityEmbeddings      OpenAIEndpointCapability = "embeddings"
	OpenAIEndpointCapabilityAlphaSearch     OpenAIEndpointCapability = "alpha_search"
	OpenAIEndpointCapabilityResponses       OpenAIEndpointCapability = "responses"
	OpenAIEndpointCapabilityGrokMedia       OpenAIEndpointCapability = "grok_media_generation"

	openAIEndpointCapabilitiesCredentialKey = "openai_capabilities"
	openAIResponsesSupportedExtraKey        = "openai_responses_supported"
	GrokMediaEligibleExtraKey               = "grok_media_eligible"
)

func NormalizeOpenAIEndpointCapability(value string) OpenAIEndpointCapability {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case string(OpenAIEndpointCapabilityChatCompletions), "chat", "chat.completions", "chat/completions":
		return OpenAIEndpointCapabilityChatCompletions
	case string(OpenAIEndpointCapabilityEmbeddings), "embedding", "openai.embeddings":
		return OpenAIEndpointCapabilityEmbeddings
	case string(OpenAIEndpointCapabilityAlphaSearch), "search", "web_search", "alpha.search", "alpha/search":
		return OpenAIEndpointCapabilityAlphaSearch
	case string(OpenAIEndpointCapabilityResponses), "response", "openai.responses", "responses_api", "responses/api":
		return OpenAIEndpointCapabilityResponses
	case string(OpenAIEndpointCapabilityGrokMedia), "grok_media", "grok.media", "media_generation":
		return OpenAIEndpointCapabilityGrokMedia
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
	if capability == OpenAIEndpointCapabilityGrokMedia {
		eligible, _ := account.GrokMediaGenerationEligibility()
		return eligible
	}
	resolved := ResolveProtocolGatewayInboundAccount(account, PlatformOpenAI)
	if resolved == nil {
		return false
	}
	if !supportsOpenAIEndpointCapabilityByAccountKind(resolved, capability) {
		return false
	}
	if capability == OpenAIEndpointCapabilityAlphaSearch && !openAIAlphaSearchAccountEnabled(resolved) {
		return false
	}
	configured := resolved.GetOpenAIEndpointCapabilities()
	if len(configured) == 0 {
		return true
	}
	if capability == OpenAIEndpointCapabilityResponses {
		return openAIEndpointCapabilityConfigured(configured, OpenAIEndpointCapabilityResponses) ||
			openAIEndpointCapabilityConfigured(configured, OpenAIEndpointCapabilityChatCompletions)
	}
	if capability == OpenAIEndpointCapabilityAlphaSearch {
		// Legacy installations used chat_completions as the broad OpenAI
		// capability marker before alpha/search had its own entry.
		return openAIEndpointCapabilityConfigured(configured, OpenAIEndpointCapabilityAlphaSearch) ||
			openAIEndpointCapabilityConfigured(configured, OpenAIEndpointCapabilityChatCompletions)
	}
	return openAIEndpointCapabilityConfigured(configured, capability)
}

func openAIAlphaSearchAccountEnabled(account *Account) bool {
	if account == nil || (!account.IsOpenAIOAuth() && !account.IsOpenAIApiKey()) {
		return false
	}
	if account.Extra == nil {
		return true
	}
	raw, ok := account.Extra["openai_alpha_search_enabled"]
	if !ok || raw == nil {
		return true
	}
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		n := strings.ToLower(strings.TrimSpace(value))
		return n != "false" && n != "0" && n != "no"
	default:
		return true
	}
}

func supportsOpenAIEndpointCapabilityByAccountKind(account *Account, capability OpenAIEndpointCapability) bool {
	if account == nil {
		return false
	}
	switch capability {
	case OpenAIEndpointCapabilityChatCompletions:
		return account.IsOpenAITextCompatible() || account.IsGrok()
	case OpenAIEndpointCapabilityEmbeddings:
		return account.IsOpenAIApiKey()
	case OpenAIEndpointCapabilityAlphaSearch:
		return account.IsOpenAIApiKey() || account.IsOpenAIOAuth()
	case OpenAIEndpointCapabilityResponses:
		if !account.IsOpenAITextCompatible() && !account.IsGrok() {
			return false
		}
		return !isOpenAIAPIKeyResponsesExplicitlyUnsupported(account)
	default:
		return false
	}
}

func openAIEndpointCapabilityConfigured(configured []OpenAIEndpointCapability, capability OpenAIEndpointCapability) bool {
	for _, allowed := range configured {
		if allowed == capability {
			return true
		}
	}
	return false
}

func isOpenAIAPIKeyResponsesExplicitlyUnsupported(account *Account) bool {
	if account == nil || !account.IsOpenAIApiKey() || account.Extra == nil {
		return false
	}
	raw, exists := account.Extra[openAIResponsesSupportedExtraKey]
	if !exists {
		return false
	}
	switch typed := raw.(type) {
	case bool:
		return !typed
	case string:
		normalized := strings.TrimSpace(strings.ToLower(typed))
		return normalized == "false" || normalized == "0" || normalized == "no" || normalized == "unsupported"
	default:
		return false
	}
}

func (a *Account) GrokMediaGenerationEligibility() (bool, string) {
	if a == nil || !a.IsGrok() {
		return false, "not_grok"
	}
	if !a.IsGrokOAuth() {
		return true, "non_oauth"
	}
	if override, ok := grokMediaEligibilityOverride(a.Extra); ok {
		if override {
			return true, "override_enabled"
		}
		return false, "override_disabled"
	}
	billing, err := grokBillingSnapshotFromExtra(a.Extra)
	if err != nil || billing == nil {
		return true, "billing_unobserved"
	}
	if billing.StatusCode == 403 {
		return false, "billing_forbidden"
	}
	return true, "eligible"
}

func grokMediaEligibilityOverride(extra map[string]any) (bool, bool) {
	if len(extra) == 0 {
		return false, false
	}
	value, exists := extra[GrokMediaEligibleExtraKey]
	if !exists || value == nil {
		return false, false
	}
	enabled, ok := value.(bool)
	return enabled, ok
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
