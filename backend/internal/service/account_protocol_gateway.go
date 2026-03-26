package service

import "strings"

const (
	GatewayProtocolOpenAI    = PlatformOpenAI
	GatewayProtocolAnthropic = PlatformAnthropic
	GatewayProtocolGemini    = PlatformGemini
	GatewayProtocolMixed     = "mixed"

	GatewayClientProfileCodex     = "codex"
	GatewayClientProfileGeminiCLI = "gemini_cli"

	gatewayExtraProtocolKey          = "gateway_protocol"
	gatewayExtraAcceptedProtocolsKey = "gateway_accepted_protocols"
	gatewayExtraClientProfilesKey    = "gateway_client_profiles"
	gatewayExtraClientRoutesKey      = "gateway_client_routes"
)

type ProtocolGatewayDescriptor struct {
	ID                  string
	DisplayName         string
	RequestFormats      []string
	DefaultBaseURL      string
	APIKeyPlaceholder   string
	ModelImportStrategy string
	TestStrategy        string
	TargetGroupPlatform string
	RegistryRoute       string
}

type GatewayClientRoute struct {
	Protocol      string
	MatchType     string
	MatchValue    string
	ClientProfile string
}

var protocolGatewayDescriptors = map[string]ProtocolGatewayDescriptor{
	PlatformOpenAI: {
		ID:                  PlatformOpenAI,
		DisplayName:         "OpenAI",
		RequestFormats:      []string{"/v1/chat/completions", "/v1/responses"},
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
		RequestFormats:      []string{"/v1/messages"},
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
		RequestFormats:      []string{"/v1beta/models/{model}:generateContent"},
		DefaultBaseURL:      "https://generativelanguage.googleapis.com",
		APIKeyPlaceholder:   "AIza...",
		ModelImportStrategy: "gemini",
		TestStrategy:        "gemini",
		TargetGroupPlatform: PlatformGemini,
		RegistryRoute:       "gemini",
	},
	GatewayProtocolMixed: {
		ID:                  GatewayProtocolMixed,
		DisplayName:         "Mixed",
		RequestFormats:      []string{"/v1/chat/completions", "/v1/responses", "/v1/messages", "/v1beta/models/{model}:generateContent"},
		DefaultBaseURL:      "",
		APIKeyPlaceholder:   "gateway-key-...",
		ModelImportStrategy: GatewayProtocolMixed,
		TestStrategy:        GatewayProtocolMixed,
		TargetGroupPlatform: "",
		RegistryRoute:       "default",
	},
}

var gatewayBaseProtocols = []string{PlatformOpenAI, PlatformAnthropic, PlatformGemini}

var gatewayClientProfileCompatibility = map[string]map[string]struct{}{
	GatewayClientProfileCodex: {
		PlatformOpenAI: {},
	},
	GatewayClientProfileGeminiCLI: {
		PlatformGemini: {},
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

func NormalizeGatewayClientProfile(profile string) string {
	normalized := strings.TrimSpace(strings.ToLower(profile))
	if _, ok := gatewayClientProfileCompatibility[normalized]; ok {
		return normalized
	}
	return ""
}

func NormalizeGatewayClientRouteMatchType(matchType string) string {
	switch strings.TrimSpace(strings.ToLower(matchType)) {
	case "exact":
		return "exact"
	case "prefix":
		return "prefix"
	default:
		return ""
	}
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
	return NormalizeGatewayProtocol(account.GetExtraString(gatewayExtraProtocolKey))
}

func ResolveAccountGatewayProtocol(platform string, extra map[string]any) string {
	if !IsProtocolGatewayPlatform(platform) || len(extra) == 0 {
		return ""
	}
	if value, ok := extra[gatewayExtraProtocolKey].(string); ok {
		return NormalizeGatewayProtocol(value)
	}
	return ""
}

func gatewayAcceptedProtocolsForProtocol(protocol string) []string {
	switch NormalizeGatewayProtocol(protocol) {
	case PlatformOpenAI:
		return []string{PlatformOpenAI}
	case PlatformAnthropic:
		return []string{PlatformAnthropic}
	case PlatformGemini:
		return []string{PlatformGemini}
	case GatewayProtocolMixed:
		return append([]string(nil), gatewayBaseProtocols...)
	default:
		return nil
	}
}

func parseStringSlice(raw any) []string {
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
	default:
		return nil
	}
}

func NormalizeGatewayAcceptedProtocols(protocol string, extra map[string]any) []string {
	normalizedProtocol := NormalizeGatewayProtocol(protocol)
	if normalizedProtocol == "" {
		return nil
	}
	if normalizedProtocol != GatewayProtocolMixed {
		return gatewayAcceptedProtocolsForProtocol(normalizedProtocol)
	}
	raw := parseStringSlice(extra[gatewayExtraAcceptedProtocolsKey])
	if len(raw) == 0 {
		return gatewayAcceptedProtocolsForProtocol(normalizedProtocol)
	}
	accepted := make([]string, 0, len(raw))
	for _, value := range raw {
		if normalized := NormalizeGatewayProtocol(value); normalized == PlatformOpenAI || normalized == PlatformAnthropic || normalized == PlatformGemini {
			accepted = append(accepted, normalized)
		}
	}
	accepted = uniqueStrings(accepted)
	if len(accepted) == 0 {
		return gatewayAcceptedProtocolsForProtocol(normalizedProtocol)
	}
	return accepted
}

func GetAccountGatewayAcceptedProtocols(account *Account) []string {
	if !IsProtocolGatewayAccount(account) {
		return nil
	}
	return NormalizeGatewayAcceptedProtocols(GetAccountGatewayProtocol(account), account.Extra)
}

func ResolveAccountGatewayAcceptedProtocols(platform string, extra map[string]any) []string {
	return NormalizeGatewayAcceptedProtocols(ResolveAccountGatewayProtocol(platform, extra), extra)
}

func NormalizeGatewayClientProfiles(protocol string, extra map[string]any) []string {
	acceptedProtocols := NormalizeGatewayAcceptedProtocols(protocol, extra)
	if len(acceptedProtocols) == 0 || len(extra) == 0 {
		return nil
	}
	rawProfiles := parseStringSlice(extra[gatewayExtraClientProfilesKey])
	if len(rawProfiles) == 0 {
		return nil
	}
	acceptedSet := make(map[string]struct{}, len(acceptedProtocols))
	for _, value := range acceptedProtocols {
		acceptedSet[value] = struct{}{}
	}
	profiles := make([]string, 0, len(rawProfiles))
	for _, rawProfile := range rawProfiles {
		profile := NormalizeGatewayClientProfile(rawProfile)
		if profile == "" {
			continue
		}
		if !gatewayClientProfileMatchesAnyAcceptedProtocol(profile, acceptedSet) {
			continue
		}
		profiles = append(profiles, profile)
	}
	return uniqueStrings(profiles)
}

func GetAccountGatewayClientProfiles(account *Account) []string {
	if !IsProtocolGatewayAccount(account) {
		return nil
	}
	return NormalizeGatewayClientProfiles(GetAccountGatewayProtocol(account), account.Extra)
}

func ResolveAccountGatewayClientProfiles(platform string, extra map[string]any) []string {
	return NormalizeGatewayClientProfiles(ResolveAccountGatewayProtocol(platform, extra), extra)
}

func NormalizeGatewayClientRoutes(protocol string, extra map[string]any) []GatewayClientRoute {
	acceptedProtocols := NormalizeGatewayAcceptedProtocols(protocol, extra)
	if len(acceptedProtocols) == 0 || len(extra) == 0 {
		return nil
	}
	rawRoutes, ok := extra[gatewayExtraClientRoutesKey].([]any)
	if !ok || len(rawRoutes) == 0 {
		return nil
	}
	acceptedSet := make(map[string]struct{}, len(acceptedProtocols))
	for _, value := range acceptedProtocols {
		acceptedSet[value] = struct{}{}
	}
	routes := make([]GatewayClientRoute, 0, len(rawRoutes))
	for _, rawRoute := range rawRoutes {
		routeMap, ok := rawRoute.(map[string]any)
		if !ok {
			continue
		}
		route := GatewayClientRoute{
			Protocol:      NormalizeGatewayProtocol(stringAny(routeMap["protocol"])),
			MatchType:     NormalizeGatewayClientRouteMatchType(stringAny(routeMap["match_type"])),
			MatchValue:    strings.TrimSpace(stringAny(routeMap["match_value"])),
			ClientProfile: NormalizeGatewayClientProfile(stringAny(routeMap["client_profile"])),
		}
		if route.Protocol == "" || route.Protocol == GatewayProtocolMixed {
			continue
		}
		if _, ok := acceptedSet[route.Protocol]; !ok {
			continue
		}
		if route.MatchType == "" || route.MatchValue == "" || route.ClientProfile == "" {
			continue
		}
		if !GatewayClientProfileSupportsProtocol(route.ClientProfile, route.Protocol) {
			continue
		}
		routes = append(routes, route)
	}
	return routes
}

func GetAccountGatewayClientRoutes(account *Account) []GatewayClientRoute {
	if !IsProtocolGatewayAccount(account) {
		return nil
	}
	return NormalizeGatewayClientRoutes(GetAccountGatewayProtocol(account), account.Extra)
}

func ResolveAccountGatewayClientRoutes(platform string, extra map[string]any) []GatewayClientRoute {
	return NormalizeGatewayClientRoutes(ResolveAccountGatewayProtocol(platform, extra), extra)
}

func GatewayClientProfileSupportsProtocol(profile string, protocol string) bool {
	profile = NormalizeGatewayClientProfile(profile)
	protocol = NormalizeGatewayProtocol(protocol)
	if profile == "" || protocol == "" || protocol == GatewayProtocolMixed {
		return false
	}
	allowedProtocols, ok := gatewayClientProfileCompatibility[profile]
	if !ok {
		return false
	}
	_, ok = allowedProtocols[protocol]
	return ok
}

func gatewayClientProfileMatchesAnyAcceptedProtocol(profile string, acceptedProtocols map[string]struct{}) bool {
	if len(acceptedProtocols) == 0 {
		return false
	}
	for protocol := range acceptedProtocols {
		if GatewayClientProfileSupportsProtocol(profile, protocol) {
			return true
		}
	}
	return false
}

func MatchGatewayClientRoute(account *Account, protocol string, model string) *GatewayClientRoute {
	if !IsProtocolGatewayAccount(account) {
		return nil
	}
	protocol = NormalizeGatewayProtocol(protocol)
	model = strings.TrimSpace(model)
	if protocol == "" || protocol == GatewayProtocolMixed || model == "" {
		return nil
	}
	routes := GetAccountGatewayClientRoutes(account)
	if len(routes) == 0 {
		return nil
	}
	var best *GatewayClientRoute
	for i := range routes {
		route := routes[i]
		if route.Protocol != protocol {
			continue
		}
		switch route.MatchType {
		case "exact":
			if model != route.MatchValue {
				continue
			}
		case "prefix":
			if !strings.HasPrefix(model, route.MatchValue) {
				continue
			}
		default:
			continue
		}
		if best == nil || len(route.MatchValue) > len(best.MatchValue) {
			copied := route
			best = &copied
		}
	}
	return best
}

func EffectiveProtocol(account *Account) string {
	if account == nil {
		return ""
	}
	if protocol := GetAccountGatewayProtocol(account); protocol != "" && protocol != GatewayProtocolMixed {
		return protocol
	}
	return strings.TrimSpace(strings.ToLower(account.Platform))
}

func EffectiveProtocolFromValues(platform string, extra map[string]any) string {
	if protocol := ResolveAccountGatewayProtocol(platform, extra); protocol != "" && protocol != GatewayProtocolMixed {
		return protocol
	}
	return strings.TrimSpace(strings.ToLower(platform))
}

func RoutingPlatformsForAccount(account *Account) []string {
	if account == nil {
		return nil
	}
	return RoutingPlatformsFromValues(account.Platform, account.Extra)
}

func RoutingPlatformsFromValues(platform string, extra map[string]any) []string {
	if !IsProtocolGatewayPlatform(platform) {
		normalized := strings.TrimSpace(strings.ToLower(platform))
		if normalized == "" {
			return nil
		}
		return []string{normalized}
	}
	acceptedProtocols := ResolveAccountGatewayAcceptedProtocols(platform, extra)
	if len(acceptedProtocols) == 0 {
		return nil
	}
	platforms := make([]string, 0, len(acceptedProtocols))
	for _, protocol := range acceptedProtocols {
		if descriptor, ok := ProtocolGatewayDescriptorByID(protocol); ok && descriptor.TargetGroupPlatform != "" {
			platforms = append(platforms, descriptor.TargetGroupPlatform)
		}
	}
	return uniqueStrings(platforms)
}

func RoutingPlatformForAccount(account *Account) string {
	platforms := RoutingPlatformsForAccount(account)
	if len(platforms) == 0 {
		return ""
	}
	return platforms[0]
}

func RoutingPlatformFromValues(platform string, extra map[string]any) string {
	platforms := RoutingPlatformsFromValues(platform, extra)
	if len(platforms) == 0 {
		return ""
	}
	return platforms[0]
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
		for _, platform := range RoutingPlatformsForAccount(account) {
			if platform == groupPlatform {
				return true
			}
		}
		return false
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
	protocol := GetAccountGatewayProtocol(account)
	if protocol == GatewayProtocolMixed {
		return "default"
	}
	if descriptor, ok := ProtocolGatewayDescriptorByID(protocol); ok {
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

func NormalizeProtocolGatewayExtra(platform string, extra map[string]any, gatewayProtocol string, fallback string) map[string]any {
	if !IsProtocolGatewayPlatform(platform) {
		return cloneProtocolGatewayExtraMap(extra)
	}
	nextExtra := cloneProtocolGatewayExtraMap(extra)
	if nextExtra == nil {
		nextExtra = map[string]any{}
	}
	protocol := NormalizeGatewayProtocol(gatewayProtocol)
	if protocol == "" {
		protocol = NormalizeGatewayProtocol(fallback)
	}
	if protocol == "" {
		delete(nextExtra, gatewayExtraProtocolKey)
		delete(nextExtra, gatewayExtraAcceptedProtocolsKey)
		delete(nextExtra, gatewayExtraClientProfilesKey)
		delete(nextExtra, gatewayExtraClientRoutesKey)
		return nextExtra
	}
	nextExtra[gatewayExtraProtocolKey] = protocol
	acceptedProtocols := NormalizeGatewayAcceptedProtocols(protocol, nextExtra)
	if len(acceptedProtocols) > 0 {
		nextExtra[gatewayExtraAcceptedProtocolsKey] = acceptedProtocols
	}
	if profiles := NormalizeGatewayClientProfiles(protocol, nextExtra); len(profiles) > 0 {
		nextExtra[gatewayExtraClientProfilesKey] = profiles
	} else {
		delete(nextExtra, gatewayExtraClientProfilesKey)
	}
	if routes := NormalizeGatewayClientRoutes(protocol, nextExtra); len(routes) > 0 {
		items := make([]map[string]any, 0, len(routes))
		for _, route := range routes {
			items = append(items, map[string]any{
				"protocol":       route.Protocol,
				"match_type":     route.MatchType,
				"match_value":    route.MatchValue,
				"client_profile": route.ClientProfile,
			})
		}
		nextExtra[gatewayExtraClientRoutesKey] = items
	} else {
		delete(nextExtra, gatewayExtraClientRoutesKey)
	}
	return nextExtra
}

func ResolveProtocolGatewayInboundAccount(account *Account, protocol string) *Account {
	if !IsProtocolGatewayAccount(account) {
		return account
	}
	protocol = NormalizeGatewayProtocol(protocol)
	if protocol == "" || protocol == GatewayProtocolMixed {
		return account
	}
	acceptedProtocols := GetAccountGatewayAcceptedProtocols(account)
	acceptedSet := make(map[string]struct{}, len(acceptedProtocols))
	for _, value := range acceptedProtocols {
		acceptedSet[value] = struct{}{}
	}
	if _, ok := acceptedSet[protocol]; !ok {
		return account
	}
	cloned := *account
	cloned.Extra = NormalizeProtocolGatewayExtra(account.Platform, account.Extra, protocol, GetAccountGatewayProtocol(account))
	return &cloned
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

func cloneProtocolGatewayExtraMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func stringAny(value any) string {
	if raw, ok := value.(string); ok {
		return raw
	}
	return ""
}
