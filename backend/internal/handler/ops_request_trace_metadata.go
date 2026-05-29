package handler

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func enrichOpsTraceRequestMetadata(ctx context.Context, payload map[string]any, result *service.ProtocolNormalizeResult) {
	if payload == nil || result == nil {
		return
	}

	if result.RequestedModel == "" {
		result.RequestedModel = strings.TrimSpace(stringValueFromMap(payload, "model"))
	}
	if stream, ok := payload["stream"].(bool); ok {
		result.Stream = stream
	}

	enrichOpsTraceRequestThinkingConfig(payload, result)

	protocolFamily := opsTraceProtocolFamily(firstNonEmptyString(result.ProtocolOut, result.ProtocolIn))
	effortResolution, effortFields := resolveOpsTraceRequestEffort(ctx, payload, protocolFamily, result)
	applyOpsTraceRequestEffortFields(effortFields, result)
	applyOpsTraceRequestEffortResolution(protocolFamily, effortFields, effortResolution, result)

	if mediaResolution := strings.TrimSpace(stringValueFromMap(payload, "media_resolution")); mediaResolution != "" && result.MediaResolution == "" {
		result.MediaResolution = mediaResolution
	}
	if mediaResolution := strings.TrimSpace(stringValueFromMap(payload, "mediaResolution")); mediaResolution != "" && result.MediaResolution == "" {
		result.MediaResolution = mediaResolution
	}

	toolKinds := make([]string, 0, 4)
	switch tools := payload["tools"].(type) {
	case []any:
		for _, tool := range tools {
			toolKinds = append(toolKinds, inferOpsTraceToolKinds(tool)...)
		}
	case []map[string]any:
		for _, tool := range tools {
			toolKinds = append(toolKinds, inferOpsTraceToolKinds(tool)...)
		}
	}
	toolKinds = dedupeTraceStrings(toolKinds)
	result.ToolKinds = toolKinds
	result.HasTools = len(toolKinds) > 0
}

func enrichOpsTraceRequestThinkingConfig(payload map[string]any, result *service.ProtocolNormalizeResult) {
	if generationConfig, ok := payload["generationConfig"].(map[string]any); ok && generationConfig != nil {
		if mediaResolution := strings.TrimSpace(stringValueFromMap(generationConfig, "mediaResolution")); mediaResolution != "" {
			result.MediaResolution = mediaResolution
		}
		if thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any); ok && thinkingConfig != nil {
			if thinkingLevel := strings.TrimSpace(stringValueFromMap(thinkingConfig, "thinkingLevel")); thinkingLevel != "" {
				result.HasThinking = true
				result.ThinkingSource = "thinking_level"
				result.ThinkingLevel = thinkingLevel
			}
			if thinkingBudget := intValueFromMap(thinkingConfig, "thinkingBudget"); thinkingBudget != nil {
				result.HasThinking = true
				if result.ThinkingSource == "" {
					result.ThinkingSource = "thinking_budget"
				}
				result.ThinkingBudget = thinkingBudget
			}
		}
	}

	if thinking, ok := payload["thinking"].(map[string]any); ok && thinking != nil {
		result.HasThinking = true
		if result.ThinkingSource == "" {
			result.ThinkingSource = "compat_thinking"
		}
		if level := strings.TrimSpace(stringValueFromMap(thinking, "type")); level != "" && result.ThinkingLevel == "" {
			result.ThinkingLevel = strings.ToUpper(level)
		}
		if budget := firstNonNilInt(thinking["budget_tokens"], thinking["budgetTokens"]); budget != nil && result.ThinkingBudget == nil {
			result.ThinkingBudget = budget
		}
	}
}
