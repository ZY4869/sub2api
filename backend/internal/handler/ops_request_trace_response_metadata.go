package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func enrichOpsTraceResponseMetadata(payload map[string]any, protocolOut string, result *service.ProtocolNormalizeResult, usage *opsTraceUsage) {
	if payload == nil || result == nil || usage == nil {
		return
	}

	switch opsTraceProtocolFamily(protocolOut) {
	case "gemini":
		enrichOpsTraceGeminiResponseMetadata(payload, result, usage)
	default:
		enrichOpsTraceOpenAIAnthropicResponseMetadata(payload, result, usage)
	}
}

func enrichOpsTraceGeminiResponseMetadata(payload map[string]any, result *service.ProtocolNormalizeResult, usage *opsTraceUsage) {
	if responseID := strings.TrimSpace(stringValueFromMap(payload, "responseId")); responseID != "" {
		result.UpstreamRequestID = responseID
	}
	if modelVersion := strings.TrimSpace(stringValueFromMap(payload, "modelVersion")); modelVersion != "" {
		result.ActualUpstreamModel = modelVersion
	}
	if promptFeedback, ok := payload["promptFeedback"].(map[string]any); ok && promptFeedback != nil {
		result.PromptBlockReason = strings.TrimSpace(stringValueFromMap(promptFeedback, "blockReason"))
	}
	if candidates, ok := payload["candidates"].([]any); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]any); ok && candidate != nil {
			result.FinishReason = strings.TrimSpace(stringValueFromMap(candidate, "finishReason"))
			if content, ok := candidate["content"].(map[string]any); ok && content != nil {
				if parts, ok := content["parts"].([]any); ok {
					for _, part := range parts {
						if pm, ok := part.(map[string]any); ok && pm != nil {
							if functionCall, ok := pm["functionCall"].(map[string]any); ok && functionCall != nil {
								result.HasTools = true
								result.ToolKinds = dedupeTraceStrings(append(result.ToolKinds, "function"))
							}
						}
					}
				}
			}
		}
	}
	if usageMetadata, ok := payload["usageMetadata"].(map[string]any); ok && usageMetadata != nil {
		if promptTokens := intValueFromMap(usageMetadata, "promptTokenCount"); promptTokens != nil {
			usage.inputTokens = *promptTokens
		}
		if completionTokens := intValueFromMap(usageMetadata, "candidatesTokenCount"); completionTokens != nil {
			usage.outputTokens = *completionTokens
		}
		if totalTokens := intValueFromMap(usageMetadata, "totalTokenCount"); totalTokens != nil {
			usage.totalTokens = *totalTokens
		}
	}
}

func enrichOpsTraceOpenAIAnthropicResponseMetadata(payload map[string]any, result *service.ProtocolNormalizeResult, usage *opsTraceUsage) {
	if model := strings.TrimSpace(stringValueFromMap(payload, "model")); model != "" {
		result.ActualUpstreamModel = model
	}
	if responseID := strings.TrimSpace(stringValueFromMap(payload, "id")); responseID != "" {
		result.UpstreamRequestID = responseID
	}
	if stopReason := strings.TrimSpace(stringValueFromMap(payload, "stop_reason")); stopReason != "" {
		result.FinishReason = stopReason
	}
	if usageMap, ok := payload["usage"].(map[string]any); ok && usageMap != nil {
		if inputTokens := firstNonNilInt(usageMap["input_tokens"], usageMap["prompt_tokens"]); inputTokens != nil {
			usage.inputTokens = *inputTokens
		}
		if outputTokens := firstNonNilInt(usageMap["output_tokens"], usageMap["completion_tokens"]); outputTokens != nil {
			usage.outputTokens = *outputTokens
		}
		if totalTokens := firstNonNilInt(usageMap["total_tokens"], usageMap["totalTokens"]); totalTokens != nil {
			usage.totalTokens = *totalTokens
		}
	}
	if choices, ok := payload["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok && choice != nil {
			if finishReason := strings.TrimSpace(stringValueFromMap(choice, "finish_reason")); finishReason != "" && result.FinishReason == "" {
				result.FinishReason = finishReason
			}
		}
	}
}
