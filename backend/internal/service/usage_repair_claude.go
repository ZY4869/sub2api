package service

import (
	"strings"

	"github.com/tidwall/gjson"
)

func buildClaudeUsageRepairPatch(candidate ClaudeUsageRepairCandidate) UsageRepairTaskPatch {
	patch := UsageRepairTaskPatch{}

	if stringPtrBlank(candidate.InboundEndpoint) {
		if inbound := normalizeRepairEndpoint(candidate.TraceRoutePath); inbound != "" {
			patch.InboundEndpoint = &inbound
		}
	}

	if stringPtrBlank(candidate.UpstreamEndpoint) {
		if upstream := normalizeRepairEndpoint(candidate.TraceUpstreamPath); upstream != "" {
			patch.UpstreamEndpoint = &upstream
		} else if fallback := deriveClaudeRepairUpstreamEndpoint(candidate); fallback != "" {
			patch.UpstreamEndpoint = &fallback
		}
	}

	if candidate.ThinkingEnabled == nil && candidate.TraceHasThinking != nil {
		value := *candidate.TraceHasThinking
		patch.ThinkingEnabled = &value
	}

	if stringPtrBlank(candidate.ReasoningEffort) {
		if effort := extractClaudeOutputEffort(candidate.TraceInboundJSON); effort != nil {
			patch.ReasoningEffort = effort
		} else if effort := extractClaudeOutputEffort(candidate.TraceNormalizedJSON); effort != nil {
			patch.ReasoningEffort = effort
		}
	}

	return patch
}

func extractClaudeOutputEffort(payload string) *string {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil
	}
	return NormalizeClaudeOutputEffort(gjson.Get(payload, "output_config.effort").String())
}

func deriveClaudeRepairUpstreamEndpoint(candidate ClaudeUsageRepairCandidate) string {
	if routePath := normalizeRepairEndpoint(candidate.TraceRoutePath); routePath == EndpointMessages {
		return EndpointMessages
	}
	if routePath := normalizeRepairEndpoint(stringPtrValue(candidate.InboundEndpoint)); routePath == EndpointMessages {
		return EndpointMessages
	}
	models := []string{candidate.UpstreamModel, candidate.RequestedModel, candidate.Model}
	for _, model := range models {
		if strings.Contains(strings.ToLower(strings.TrimSpace(model)), "claude") {
			return EndpointMessages
		}
	}
	return ""
}

func normalizeRepairEndpoint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "/") {
		return "/" + value
	}
	return value
}

func stringPtrBlank(value *string) bool {
	return strings.TrimSpace(stringPtrValue(value)) == ""
}
