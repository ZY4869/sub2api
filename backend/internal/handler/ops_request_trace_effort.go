package handler

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type opsTraceRequestEffortFields struct {
	gatewayEffort         string
	openAIReasoningEffort string
	openAIAliasEffort     string
	anthropicEffort       string
	geminiEffort          string
	geminiNestedEffort    string
}

func resolveOpsTraceRequestEffort(ctx context.Context, payload map[string]any, protocolFamily string, result *service.ProtocolNormalizeResult) (service.GatewayEffortResolution, opsTraceRequestEffortFields) {
	fields := opsTraceRequestEffortFields{
		gatewayEffort:     strings.TrimSpace(stringValueFromMap(payload, "effortLevel")),
		openAIAliasEffort: strings.TrimSpace(stringValueFromMap(payload, "reasoning_effort")),
		geminiEffort:      strings.TrimSpace(stringValueFromMap(payload, "thinkingLevel")),
	}
	if reasoning, ok := payload["reasoning"].(map[string]any); ok && reasoning != nil {
		fields.openAIReasoningEffort = strings.TrimSpace(stringValueFromMap(reasoning, "effort"))
	}
	if outputConfig, ok := payload["output_config"].(map[string]any); ok && outputConfig != nil {
		fields.anthropicEffort = strings.TrimSpace(stringValueFromMap(outputConfig, "effort"))
	}
	if thinking, ok := payload["thinking"].(map[string]any); ok && thinking != nil {
		fields.geminiNestedEffort = firstNonEmptyString(
			stringValueFromMap(thinking, "thinkingLevel"),
			stringValueFromMap(thinking, "thinking_level"),
			stringValueFromMap(thinking, "level"),
		)
	}

	if fields.gatewayEffort != "" {
		result.GatewayEffortLevel = fields.gatewayEffort
	}
	applyOpsTraceClaudeMetadata(ctx, result)

	switch {
	case protocolFamily == "gemini":
		resolution := service.ResolveGeminiEffort(fields.anthropicEffort, fields.geminiNestedEffort, fields.openAIReasoningEffort, fields.openAIAliasEffort, "")
		if fields.geminiEffort != "" || fields.anthropicEffort != "" {
			resolution = service.ResolveGeminiEffort(fields.geminiEffort, firstNonEmptyString(fields.geminiNestedEffort, fields.anthropicEffort), fields.openAIReasoningEffort, fields.openAIAliasEffort, "")
		}
		return resolution, fields
	case protocolFamily == "anthropic":
		resolution := service.ResolveAnthropicEffort(firstNonEmptyString(fields.anthropicEffort, fields.openAIReasoningEffort, fields.openAIAliasEffort), fields.gatewayEffort)
		if fields.anthropicEffort == "" && (fields.openAIReasoningEffort != "" || fields.openAIAliasEffort != "") {
			resolution.Source = "mapped_reasoning_effort"
		}
		return resolution, fields
	case fields.anthropicEffort != "":
		return service.ResolveAnthropicEffortForOpenAI(fields.anthropicEffort, fields.gatewayEffort), fields
	case fields.openAIReasoningEffort != "":
		return service.ResolveOpenAIEffort(fields.openAIReasoningEffort, "", "reasoning.effort"), fields
	case fields.openAIAliasEffort != "":
		return service.ResolveOpenAIEffort(fields.openAIAliasEffort, "", "reasoning_effort"), fields
	case fields.geminiEffort != "" || fields.geminiNestedEffort != "":
		return service.ResolveGeminiEffort(fields.geminiEffort, fields.geminiNestedEffort, fields.openAIReasoningEffort, fields.openAIAliasEffort, ""), fields
	default:
		return service.GatewayEffortResolution{}, fields
	}
}

func applyOpsTraceClaudeMetadata(ctx context.Context, result *service.ProtocolNormalizeResult) {
	if rawModel, ok := service.ClaudeRequestedModelRawMetadataFromContext(ctx); ok {
		result.RequestedModelRaw = rawModel
		result.RequestedModel = rawModel
	}
	if normalizedModel, ok := service.ClaudeRequestedModelNormalizedMetadataFromContext(ctx); ok && strings.TrimSpace(normalizedModel) != "" {
		result.RequestedModelNormalized = normalizedModel
		result.UpstreamModel = normalizedModel
	}
	if requested, ok := service.ClaudeMillionContextRequestedMetadataFromContext(ctx); ok {
		result.MillionContextRequested = requested
		if requested {
			result.HasThinking = true
		}
	}
	if effective, ok := service.ClaudeMillionContextEffectiveMetadataFromContext(ctx); ok {
		result.MillionContextEffective = effective
		if effective {
			result.HasThinking = true
		}
	}
	if source, ok := service.ClaudeMillionContextSourceMetadataFromContext(ctx); ok {
		result.MillionContextSource = source
	}
	if betaToken, ok := service.ClaudeMillionContextBetaTokenMetadataFromContext(ctx); ok {
		result.MillionContextBetaToken = betaToken
	}
}
