package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func applyOpsTraceRequestEffortFields(fields opsTraceRequestEffortFields, result *service.ProtocolNormalizeResult) {
	if fields.anthropicEffort != "" {
		result.ProtocolEffortField = "output_config.effort"
		result.ProtocolEffortValue = fields.anthropicEffort
	} else if fields.openAIReasoningEffort != "" {
		result.ProtocolEffortField = "reasoning.effort"
		result.ProtocolEffortValue = fields.openAIReasoningEffort
	} else if fields.openAIAliasEffort != "" {
		result.ProtocolEffortField = "reasoning_effort"
		result.ProtocolEffortValue = fields.openAIAliasEffort
	} else if fields.geminiEffort != "" {
		result.ProtocolEffortField = "thinkingLevel"
		result.ProtocolEffortValue = fields.geminiEffort
	} else if fields.geminiNestedEffort != "" {
		result.ProtocolEffortField = "thinking.thinkingLevel"
		result.ProtocolEffortValue = fields.geminiNestedEffort
	}
}

func applyOpsTraceRequestEffortResolution(protocolFamily string, fields opsTraceRequestEffortFields, effortResolution service.GatewayEffortResolution, result *service.ProtocolNormalizeResult) {
	if effortResolution.Raw != nil {
		result.HasThinking = true
		result.ReasoningEffortRaw = *effortResolution.Raw
	}
	if effortResolution.Effective == nil {
		return
	}
	result.HasThinking = true
	result.ReasoningEffortEffective = *effortResolution.Effective
	result.ThinkingSource = effortResolution.Source
	levelValue := firstNonEmptyString(result.ReasoningEffortRaw, result.ReasoningEffortEffective)
	if protocolFamily == "gemini" && fields.geminiEffort == "" && fields.geminiNestedEffort == "" {
		levelValue = result.ReasoningEffortEffective
	}
	switch strings.ToLower(levelValue) {
	case "low":
		result.ThinkingLevel = "LOW"
	case "medium":
		result.ThinkingLevel = "MEDIUM"
	case "high":
		result.ThinkingLevel = "HIGH"
	case "xhigh":
		result.ThinkingLevel = "XHIGH"
	case "max":
		result.ThinkingLevel = "MAX"
	case "none", "minimal":
		result.ThinkingLevel = "MINIMAL"
	}
}
