package service

import (
	"strings"

	"github.com/tidwall/gjson"
)

const (
	effortSourceTopLevel       = "top_level_effortLevel"
	effortSourceAnthropicField = "output_config.effort"
	effortSourceOpenAIField    = "reasoning.effort"
	effortSourceOpenAIAlias    = "reasoning_effort"
	effortSourceGeminiField    = "thinkingLevel"
	effortSourceGeminiNested   = "thinking.thinkingLevel"
	effortSourceGeminiMapped   = "mapped_reasoning_effort"
)

type GatewayEffortResolution struct {
	Raw       *string
	Effective *string
	Source    string
}

func NormalizeGatewayEffortLevel(raw string) *string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return nil
	}
	value = strings.NewReplacer("-", "", "_", "", " ", "").Replace(value)
	switch value {
	case "low", "medium", "high", "xhigh", "max":
		return &value
	default:
		return nil
	}
}

func NormalizeClaudeOutputEffort(raw string) *string {
	return NormalizeGatewayEffortLevel(raw)
}

func normalizeOpenAIReasoningEffortRaw(raw string) *string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return nil
	}
	value = strings.NewReplacer("-", "", "_", "", " ", "").Replace(value)
	switch value {
	case "none", "minimal":
		normalized := "none"
		return &normalized
	case "low", "medium", "high", "xhigh", "max":
		return &value
	case "extrahigh":
		normalized := "xhigh"
		return &normalized
	default:
		return nil
	}
}

func NormalizeOpenAIReasoningEffortEffective(raw string) *string {
	normalized := normalizeOpenAIReasoningEffortRaw(raw)
	if normalized == nil {
		return nil
	}
	switch *normalized {
	case "max":
		effective := "xhigh"
		return &effective
	default:
		return normalized
	}
}

func NormalizeGeminiThinkingLevel(raw string) *string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return nil
	}
	value = strings.NewReplacer("-", "", "_", "", " ", "").Replace(value)
	switch value {
	case "minimal", "none":
		normalized := "MINIMAL"
		return &normalized
	case "low":
		normalized := "LOW"
		return &normalized
	case "medium":
		normalized := "MEDIUM"
		return &normalized
	case "high", "xhigh", "extrahigh", "max":
		normalized := "HIGH"
		return &normalized
	default:
		return nil
	}
}

func normalizeGeminiEffectiveEffortValue(raw string) *string {
	normalized := NormalizeGatewayEffortLevel(raw)
	if normalized == nil {
		return nil
	}
	switch *normalized {
	case "max", "xhigh", "high":
		value := "HIGH"
		return &value
	case "medium":
		value := "MEDIUM"
		return &value
	case "low":
		value := "LOW"
		return &value
	default:
		return nil
	}
}

func ResolveGeminiEffort(native string, nativeNested string, openAINative string, openAIAlias string, topLevel string) GatewayEffortResolution {
	if normalized := NormalizeGeminiThinkingLevel(native); normalized != nil {
		return GatewayEffortResolution{
			Raw:       normalized,
			Effective: normalized,
			Source:    effortSourceGeminiField,
		}
	}
	if normalized := NormalizeGeminiThinkingLevel(nativeNested); normalized != nil {
		return GatewayEffortResolution{
			Raw:       normalized,
			Effective: normalized,
			Source:    effortSourceGeminiNested,
		}
	}
	if normalized := normalizeOpenAIReasoningEffortRaw(openAINative); normalized != nil {
		return GatewayEffortResolution{
			Raw:       normalized,
			Effective: normalizeGeminiEffectiveEffortValue(*normalized),
			Source:    effortSourceOpenAIField,
		}
	}
	if normalized := normalizeOpenAIReasoningEffortRaw(openAIAlias); normalized != nil {
		return GatewayEffortResolution{
			Raw:       normalized,
			Effective: normalizeGeminiEffectiveEffortValue(*normalized),
			Source:    effortSourceOpenAIAlias,
		}
	}
	if normalized := NormalizeGatewayEffortLevel(topLevel); normalized != nil {
		return GatewayEffortResolution{
			Raw:       normalized,
			Effective: normalizeGeminiEffectiveEffortValue(*normalized),
			Source:    effortSourceTopLevel,
		}
	}
	return GatewayEffortResolution{}
}

func ResolveAnthropicEffort(native string, topLevel string) GatewayEffortResolution {
	if normalized := NormalizeClaudeOutputEffort(native); normalized != nil {
		return GatewayEffortResolution{
			Raw:       normalized,
			Effective: normalized,
			Source:    effortSourceAnthropicField,
		}
	}
	if normalized := NormalizeGatewayEffortLevel(topLevel); normalized != nil {
		return GatewayEffortResolution{
			Raw:       normalized,
			Effective: normalized,
			Source:    effortSourceTopLevel,
		}
	}
	return GatewayEffortResolution{}
}

func ResolveAnthropicEffortFromBody(body string) GatewayEffortResolution {
	return ResolveAnthropicEffort(
		strings.TrimSpace(gjson.Get(body, "output_config.effort").String()),
		strings.TrimSpace(gjson.Get(body, "effortLevel").String()),
	)
}

func ResolveOpenAIEffort(native string, topLevel string, source string) GatewayEffortResolution {
	if normalized := normalizeOpenAIReasoningEffortRaw(native); normalized != nil {
		return GatewayEffortResolution{
			Raw:       normalized,
			Effective: NormalizeOpenAIReasoningEffortEffective(*normalized),
			Source:    source,
		}
	}
	if normalized := NormalizeGatewayEffortLevel(topLevel); normalized != nil {
		return GatewayEffortResolution{
			Raw:       normalized,
			Effective: NormalizeOpenAIReasoningEffortEffective(*normalized),
			Source:    effortSourceTopLevel,
		}
	}
	return GatewayEffortResolution{}
}

func ResolveAnthropicEffortForOpenAI(native string, topLevel string) GatewayEffortResolution {
	resolution := ResolveAnthropicEffort(native, topLevel)
	if resolution.Raw == nil {
		return resolution
	}
	resolution.Effective = NormalizeOpenAIReasoningEffortEffective(*resolution.Raw)
	return resolution
}

func NormalizeGatewayEffortForUsage(legacy *string, raw *string, effective *string) (*string, *string, *string) {
	if legacy == nil && raw == nil && effective == nil {
		return nil, nil, nil
	}
	normalizedRaw := raw
	if normalizedRaw == nil && legacy != nil {
		normalizedRaw = legacy
	}
	if normalizedRaw == nil && effective != nil {
		normalizedRaw = effective
	}
	normalizedEffective := effective
	if normalizedEffective == nil && legacy != nil {
		normalizedEffective = legacy
	}
	if normalizedEffective == nil && normalizedRaw != nil {
		normalizedEffective = normalizedRaw
	}
	return normalizedEffective, normalizedRaw, normalizedEffective
}
