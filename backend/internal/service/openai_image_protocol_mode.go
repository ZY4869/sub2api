package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	OpenAIImageProtocolModeNative = "native"
	OpenAIImageProtocolModeCompat = "compat"

	OpenAIGroupImageProtocolModeInherit = "inherit"

	openAIImageProtocolModeExtraKey        = "image_protocol_mode"
	openAIImageCompatAllowedExtraKey       = "image_compat_allowed"
	gatewayOpenAIImageProtocolModeExtraKey = "gateway_openai_image_protocol_mode"

	OpenAICompatImageTargetModel = "gpt-image-2"
	OpenAICompatImageHostModel   = "gpt-5.4-mini"
)

var recognizedOpenAIPaidPlans = map[string]struct{}{
	"plus":       {},
	"pro":        {},
	"team":       {},
	"business":   {},
	"enterprise": {},
	"edu":        {},
}

type openAIImageCompatAllowance struct {
	Allowed     bool
	PlanType    string
	UnknownPlan bool
}

func NormalizeOpenAIImageProtocolMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case OpenAIImageProtocolModeNative:
		return OpenAIImageProtocolModeNative
	case OpenAIImageProtocolModeCompat:
		return OpenAIImageProtocolModeCompat
	default:
		return ""
	}
}

func NormalizeOpenAIGroupImageProtocolMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case OpenAIGroupImageProtocolModeInherit:
		return OpenAIGroupImageProtocolModeInherit
	case OpenAIImageProtocolModeNative:
		return OpenAIImageProtocolModeNative
	case OpenAIImageProtocolModeCompat:
		return OpenAIImageProtocolModeCompat
	default:
		return ""
	}
}

func ResolveGatewayOpenAIImageProtocolMode(platform string, extra map[string]any) string {
	if !protocolGatewayAcceptsOpenAI(platform, extra) {
		return ""
	}
	if value, ok := extra[gatewayOpenAIImageProtocolModeExtraKey].(string); ok {
		if normalized := NormalizeOpenAIImageProtocolMode(value); normalized != "" {
			return normalized
		}
	}
	return OpenAIImageProtocolModeNative
}

func ResolveAccountImageProtocolMode(account *Account) string {
	if account == nil {
		return OpenAIImageProtocolModeNative
	}
	if IsProtocolGatewayAccount(account) {
		if mode := ResolveGatewayOpenAIImageProtocolMode(account.Platform, account.Extra); mode != "" {
			return mode
		}
		return OpenAIImageProtocolModeNative
	}
	if !account.IsOpenAI() {
		return OpenAIImageProtocolModeNative
	}
	if mode := NormalizeOpenAIImageProtocolMode(account.GetExtraString(openAIImageProtocolModeExtraKey)); mode != "" {
		return mode
	}
	if account.IsOpenAIOAuth() {
		if IsOpenAIImageCompatAllowed(account) {
			return OpenAIImageProtocolModeCompat
		}
		return OpenAIImageProtocolModeNative
	}
	return OpenAIImageProtocolModeNative
}

func ResolveEffectiveOpenAIImageProtocolMode(group *Group, account *Account) string {
	if group != nil {
		switch NormalizeOpenAIGroupImageProtocolMode(group.ImageProtocolMode) {
		case OpenAIImageProtocolModeNative:
			return OpenAIImageProtocolModeNative
		case OpenAIImageProtocolModeCompat:
			return OpenAIImageProtocolModeCompat
		}
	}
	return ResolveAccountImageProtocolMode(account)
}

func IsOpenAIImageCompatAllowed(account *Account) bool {
	if account == nil {
		return false
	}
	if IsProtocolGatewayAccount(account) {
		return protocolGatewayAcceptsOpenAI(account.Platform, account.Extra)
	}
	if !account.IsOpenAI() {
		return false
	}
	if !account.IsOpenAIOAuth() {
		return true
	}
	if account.Extra != nil {
		if _, exists := account.Extra[openAIImageCompatAllowedExtraKey]; exists {
			return parseExtraBool(account.Extra[openAIImageCompatAllowedExtraKey])
		}
	}
	return resolveOpenAIImageCompatAllowance(account.Credentials).Allowed
}

func NormalizeOpenAIAccountImageExtra(platform string, accountType string, credentials map[string]any, extra map[string]any) map[string]any {
	nextExtra := cloneStringAnyMap(extra)
	if len(nextExtra) == 0 {
		nextExtra = map[string]any{}
	}

	normalizedPlatform := CanonicalizePlatformValue(platform)
	switch {
	case IsProtocolGatewayPlatform(normalizedPlatform):
		if !protocolGatewayAcceptsOpenAI(normalizedPlatform, nextExtra) {
			delete(nextExtra, gatewayOpenAIImageProtocolModeExtraKey)
			return emptyMapToNil(nextExtra)
		}
		mode := NormalizeOpenAIImageProtocolMode(stringAny(nextExtra[gatewayOpenAIImageProtocolModeExtraKey]))
		if mode == "" {
			mode = OpenAIImageProtocolModeNative
		}
		nextExtra[gatewayOpenAIImageProtocolModeExtraKey] = mode
		return emptyMapToNil(nextExtra)
	case IsOpenAIFamily(normalizedPlatform):
		mode := NormalizeOpenAIImageProtocolMode(stringAny(nextExtra[openAIImageProtocolModeExtraKey]))
		if accountType == AccountTypeOAuth {
			allowance := resolveOpenAIImageCompatAllowance(credentials)
			if !allowance.Allowed && mode == OpenAIImageProtocolModeCompat {
				mode = OpenAIImageProtocolModeNative
			}
			if mode == "" {
				if allowance.Allowed {
					mode = OpenAIImageProtocolModeCompat
				} else {
					mode = OpenAIImageProtocolModeNative
				}
			}
			nextExtra[openAIImageCompatAllowedExtraKey] = allowance.Allowed
			if allowance.UnknownPlan {
				logger.L().Warn(
					"service.openai_image_protocol_mode.oauth_plan_unknown",
					zap.String("plan_type", allowance.PlanType),
					zap.Bool("compat_allowed", allowance.Allowed),
				)
			}
		} else {
			delete(nextExtra, openAIImageCompatAllowedExtraKey)
			if mode == "" {
				mode = OpenAIImageProtocolModeNative
			}
		}
		if mode == "" {
			delete(nextExtra, openAIImageProtocolModeExtraKey)
		} else {
			nextExtra[openAIImageProtocolModeExtraKey] = mode
		}
		delete(nextExtra, gatewayOpenAIImageProtocolModeExtraKey)
		return emptyMapToNil(nextExtra)
	default:
		delete(nextExtra, openAIImageProtocolModeExtraKey)
		delete(nextExtra, openAIImageCompatAllowedExtraKey)
		delete(nextExtra, gatewayOpenAIImageProtocolModeExtraKey)
		return emptyMapToNil(nextExtra)
	}
}

func resolveOpenAIImageCompatAllowance(credentials map[string]any) openAIImageCompatAllowance {
	planType := strings.TrimSpace(strings.ToLower(normalizeOpenAIPlanType(stringValueFromAny(credentialsMapValue(credentials, "plan_type")))))
	switch planType {
	case "free":
		return openAIImageCompatAllowance{Allowed: false, PlanType: planType}
	case "":
		return openAIImageCompatAllowance{Allowed: true}
	default:
		if _, ok := recognizedOpenAIPaidPlans[planType]; ok {
			return openAIImageCompatAllowance{Allowed: true, PlanType: planType}
		}
		return openAIImageCompatAllowance{Allowed: true, PlanType: planType, UnknownPlan: true}
	}
}

func protocolGatewayAcceptsOpenAI(platform string, extra map[string]any) bool {
	if !IsProtocolGatewayPlatform(platform) {
		return false
	}
	protocol := ResolveAccountGatewayProtocol(platform, extra)
	if protocol == "" {
		return false
	}
	for _, accepted := range NormalizeGatewayAcceptedProtocols(protocol, extra) {
		if accepted == PlatformOpenAI {
			return true
		}
	}
	return false
}

func credentialsMapValue(credentials map[string]any, key string) any {
	if len(credentials) == 0 {
		return nil
	}
	return credentials[key]
}

func emptyMapToNil(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	return value
}
