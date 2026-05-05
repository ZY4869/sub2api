package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

const claudeMillionContextSuffix = "[1m]"

type ClaudeRequestCapability struct {
	RequestedModelRaw        string
	RequestedModelNormalized string
	GatewayEffortLevel       string
	MillionContextRequested  bool
	MillionContextSupported  bool
	MillionContextEffective  bool
	MillionContextSource     string
	MillionContextBetaToken  string
}

func ParseClaudeRequestCapability(model string, effortLevel string) ClaudeRequestCapability {
	rawModel := strings.TrimSpace(model)
	normalizedModel, requested, source := StripClaudeMillionContextSuffix(rawModel)
	if normalizedModel == "" {
		normalizedModel = rawModel
	}
	if canonicalID, ok := modelregistry.ResolveToCanonicalID(normalizedModel); ok && canonicalID != "" {
		normalizedModel = canonicalID
	}

	var normalizedEffort string
	if effort := NormalizeGatewayEffortLevel(effortLevel); effort != nil {
		normalizedEffort = *effort
	}

	supported := SupportsClaudeMillionContextModel(normalizedModel)
	return ClaudeRequestCapability{
		RequestedModelRaw:        rawModel,
		RequestedModelNormalized: normalizedModel,
		GatewayEffortLevel:       normalizedEffort,
		MillionContextRequested:  requested,
		MillionContextSupported:  supported,
		MillionContextSource:     source,
		MillionContextBetaToken:  claude.BetaContext1M,
	}
}

func StripClaudeMillionContextSuffix(model string) (normalized string, requested bool, source string) {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return "", false, ""
	}
	if len(trimmed) < len(claudeMillionContextSuffix) {
		return trimmed, false, ""
	}
	lower := strings.ToLower(trimmed)
	if !strings.HasSuffix(lower, claudeMillionContextSuffix) {
		return trimmed, false, ""
	}
	normalized = strings.TrimSpace(trimmed[:len(trimmed)-len(claudeMillionContextSuffix)])
	if normalized == "" {
		return trimmed, false, ""
	}
	return normalized, true, "model_suffix_[1m]"
}

func NormalizeRequestedModelForClaudeCapability(model string) string {
	normalized, _, _ := StripClaudeMillionContextSuffix(model)
	if normalized == "" {
		return strings.TrimSpace(model)
	}
	return normalized
}

func SupportsClaudeMillionContextModel(model string) bool {
	normalized := NormalizeRequestedModelForClaudeCapability(model)
	if normalized == "" {
		return false
	}
	for _, candidate := range modelregistry.AlternateVersionVariants(normalized) {
		switch {
		case strings.HasPrefix(candidate, "claude-"),
			strings.Contains(candidate, ".claude-"),
			strings.HasPrefix(candidate, "deepseek-v4-flash"),
			strings.HasPrefix(candidate, "deepseek-v4-pro"),
			strings.Contains(candidate, ".deepseek-v4-flash"),
			strings.Contains(candidate, ".deepseek-v4-pro"):
			return true
		}
	}
	return false
}

func ShouldApplyClaudeGatewayEffortForPlatform(runtimePlatform string) bool {
	return IsAnthropicFamily(runtimePlatform)
}

func ShouldApplyClaudeCapabilitiesForAccount(account *Account) bool {
	if account == nil {
		return false
	}
	return ShouldApplyClaudeGatewayEffortForPlatform(EffectiveProtocol(account))
}

func ApplyClaudeCapabilityRuntime(capability ClaudeRequestCapability, runtimePlatform string) ClaudeRequestCapability {
	capability.MillionContextEffective = capability.MillionContextRequested &&
		capability.MillionContextSupported &&
		ShouldApplyClaudeGatewayEffortForPlatform(runtimePlatform)
	return capability
}

func ResolveClaudeRequestCapabilityForRuntime(model string, effortLevel string, runtimePlatform string) ClaudeRequestCapability {
	capability := ParseClaudeRequestCapability(model, effortLevel)
	return ApplyClaudeCapabilityRuntime(capability, runtimePlatform)
}

func RecordClaudeCapabilityMetadataForRuntime(ctx context.Context, model string, effortLevel string, runtimePlatform string) ClaudeRequestCapability {
	capability := ResolveClaudeRequestCapabilityForRuntime(model, effortLevel, runtimePlatform)
	RecordClaudeCapabilityMetadata(ctx, capability)
	return capability
}

func RecordClaudeCapabilityMetadataRequestedOnly(ctx context.Context, model string, effortLevel string) ClaudeRequestCapability {
	capability := ParseClaudeRequestCapability(model, effortLevel)
	RecordClaudeCapabilityMetadata(ctx, capability)
	return capability
}

func ApplyClaudeCapabilityBetaHeader(existingHeader string, capability ClaudeRequestCapability, required ...string) string {
	requiredBetas := make([]string, 0, len(required)+1)
	requiredBetas = append(requiredBetas, required...)
	if capability.MillionContextEffective && strings.TrimSpace(capability.MillionContextBetaToken) != "" {
		requiredBetas = append(requiredBetas, capability.MillionContextBetaToken)
	}
	return mergeAnthropicBeta(requiredBetas, existingHeader)
}

func ApplyClaudeCapabilityBetaHeaderDropping(existingHeader string, drop map[string]struct{}, capability ClaudeRequestCapability, required ...string) string {
	requiredBetas := make([]string, 0, len(required)+1)
	requiredBetas = append(requiredBetas, required...)
	if capability.MillionContextEffective && strings.TrimSpace(capability.MillionContextBetaToken) != "" {
		requiredBetas = append(requiredBetas, capability.MillionContextBetaToken)
	}
	return mergeAnthropicBetaDropping(requiredBetas, existingHeader, drop)
}

func ApplyClaudeCapabilityToHeader(req *http.Request, capability ClaudeRequestCapability, required ...string) {
	if req == nil {
		return
	}
	req.Header.Set("anthropic-beta", ApplyClaudeCapabilityBetaHeader(req.Header.Get("anthropic-beta"), capability, required...))
}

func ApplyClaudeCapabilityToHeaderDropping(req *http.Request, drop map[string]struct{}, capability ClaudeRequestCapability, required ...string) {
	if req == nil {
		return
	}
	req.Header.Set("anthropic-beta", ApplyClaudeCapabilityBetaHeaderDropping(req.Header.Get("anthropic-beta"), drop, capability, required...))
}

func RecordClaudeCapabilityMetadata(ctx context.Context, capability ClaudeRequestCapability) {
	if ctx == nil {
		return
	}
	SetClaudeRequestedModelRawMetadata(ctx, capability.RequestedModelRaw)
	SetClaudeRequestedModelNormalizedMetadata(ctx, capability.RequestedModelNormalized)
	SetClaudeMillionContextRequestedMetadata(ctx, capability.MillionContextRequested)
	SetClaudeMillionContextEffectiveMetadata(ctx, capability.MillionContextEffective)
	SetClaudeMillionContextSourceMetadata(ctx, capability.MillionContextSource)
	if capability.MillionContextEffective {
		SetClaudeMillionContextBetaTokenMetadata(ctx, capability.MillionContextBetaToken)
		return
	}
	SetClaudeMillionContextBetaTokenMetadata(ctx, "")
}
