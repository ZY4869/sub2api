package service

import "strings"

type ProtocolGatewayDescriptor struct {
	ID                  string
	DisplayName         string
	DefaultBaseURL      string
	APIKeyPlaceholder   string
	ModelImportStrategy string
	TestStrategy        string
	TargetGroupPlatform string
	RegistryRoute       string
}

var protocolGatewayDescriptors = map[string]ProtocolGatewayDescriptor{
	PlatformOpenAI: {
		ID:                  PlatformOpenAI,
		DisplayName:         "OpenAI",
		DefaultBaseURL:      "https://api.openai.com",
		APIKeyPlaceholder:   "sk-proj-...",
		ModelImportStrategy: "openai",
		TestStrategy:        "openai",
		TargetGroupPlatform: PlatformOpenAI,
		RegistryRoute:       "openai",
	},
	PlatformAnthropic: {
		ID:                  PlatformAnthropic,
		DisplayName:         "Anthropic",
		DefaultBaseURL:      "https://api.anthropic.com",
		APIKeyPlaceholder:   "sk-ant-...",
		ModelImportStrategy: "anthropic",
		TestStrategy:        "anthropic",
		TargetGroupPlatform: PlatformAnthropic,
		RegistryRoute:       "anthropic_apikey",
	},
	PlatformGemini: {
		ID:                  PlatformGemini,
		DisplayName:         "Gemini",
		DefaultBaseURL:      "https://generativelanguage.googleapis.com",
		APIKeyPlaceholder:   "AIza...",
		ModelImportStrategy: "gemini",
		TestStrategy:        "gemini",
		TargetGroupPlatform: PlatformGemini,
		RegistryRoute:       "gemini",
	},
}

func NormalizeGatewayProtocol(protocol string) string {
	normalized := strings.TrimSpace(strings.ToLower(protocol))
	if _, ok := protocolGatewayDescriptors[normalized]; ok {
		return normalized
	}
	return ""
}

func ProtocolGatewayDescriptorByID(id string) (ProtocolGatewayDescriptor, bool) {
	descriptor, ok := protocolGatewayDescriptors[NormalizeGatewayProtocol(id)]
	return descriptor, ok
}

func IsProtocolGatewayPlatform(platform string) bool {
	return strings.TrimSpace(strings.ToLower(platform)) == PlatformProtocolGateway
}

func IsProtocolGatewayAccount(account *Account) bool {
	return account != nil && IsProtocolGatewayPlatform(account.Platform)
}

func GetAccountGatewayProtocol(account *Account) string {
	if !IsProtocolGatewayAccount(account) {
		return ""
	}
	return NormalizeGatewayProtocol(account.GetExtraString("gateway_protocol"))
}

func ResolveAccountGatewayProtocol(platform string, extra map[string]any) string {
	if !IsProtocolGatewayPlatform(platform) || len(extra) == 0 {
		return ""
	}
	if value, ok := extra["gateway_protocol"].(string); ok {
		return NormalizeGatewayProtocol(value)
	}
	return ""
}

func EffectiveProtocol(account *Account) string {
	if account == nil {
		return ""
	}
	if protocol := GetAccountGatewayProtocol(account); protocol != "" {
		return protocol
	}
	return strings.TrimSpace(strings.ToLower(account.Platform))
}

func EffectiveProtocolFromValues(platform string, extra map[string]any) string {
	if protocol := ResolveAccountGatewayProtocol(platform, extra); protocol != "" {
		return protocol
	}
	return strings.TrimSpace(strings.ToLower(platform))
}

func RoutingPlatformForAccount(account *Account) string {
	if account == nil {
		return ""
	}
	if descriptor, ok := ProtocolGatewayDescriptorByID(GetAccountGatewayProtocol(account)); ok {
		return descriptor.TargetGroupPlatform
	}
	return strings.TrimSpace(strings.ToLower(account.Platform))
}

func RoutingPlatformFromValues(platform string, extra map[string]any) string {
	if descriptor, ok := ProtocolGatewayDescriptorByID(ResolveAccountGatewayProtocol(platform, extra)); ok {
		return descriptor.TargetGroupPlatform
	}
	return strings.TrimSpace(strings.ToLower(platform))
}

func MatchesGroupPlatform(account *Account, groupPlatform string) bool {
	if account == nil {
		return false
	}
	groupPlatform = strings.TrimSpace(strings.ToLower(groupPlatform))
	if groupPlatform == "" {
		return false
	}
	if IsProtocolGatewayAccount(account) {
		return RoutingPlatformForAccount(account) == groupPlatform
	}
	return strings.TrimSpace(strings.ToLower(account.Platform)) == groupPlatform
}

func QueryPlatformsForGroupPlatform(groupPlatform string, includeMixedAntigravity bool) []string {
	normalized := strings.TrimSpace(strings.ToLower(groupPlatform))
	if normalized == "" {
		return nil
	}
	platforms := []string{normalized}
	if normalized == PlatformOpenAI || normalized == PlatformAnthropic || normalized == PlatformGemini {
		platforms = append(platforms, PlatformProtocolGateway)
	}
	if includeMixedAntigravity && (normalized == PlatformAnthropic || normalized == PlatformGemini) {
		platforms = append(platforms, PlatformAntigravity)
	}
	return uniqueStrings(platforms)
}

func ProtocolGatewayRegistryRoute(account *Account) string {
	if account == nil {
		return "default"
	}
	if descriptor, ok := ProtocolGatewayDescriptorByID(GetAccountGatewayProtocol(account)); ok {
		return descriptor.RegistryRoute
	}
	return "default"
}

func DisplayAccountProtocolName(account *Account) string {
	if account == nil {
		return ""
	}
	if descriptor, ok := ProtocolGatewayDescriptorByID(GetAccountGatewayProtocol(account)); ok {
		return descriptor.DisplayName
	}
	return DisplayPlatformName(account.Platform)
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ToLower(value))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}
