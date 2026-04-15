package service

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

const (
	geminiCountTokensSourceHeader = "X-Sub2api-CountTokens-Source"

	geminiDynamicThinkingBudget  = -1
	gemini25FlashThinkingBudget  = 24576
	gemini25ProThinkingBudgetCap = 32768
)

type geminiCountTokensSource string

const (
	geminiCountTokensSourceUpstream  geminiCountTokensSource = "upstream"
	geminiCountTokensSourceEstimated geminiCountTokensSource = "estimated"
)

type geminiModelFamily string

const (
	geminiModelFamilyUnknown     geminiModelFamily = "unknown"
	geminiModelFamily2x          geminiModelFamily = "gemini-2.x"
	geminiModelFamily25          geminiModelFamily = "gemini-2.5"
	geminiModelFamily3Flash      geminiModelFamily = "gemini-3-flash"
	geminiModelFamily31Pro       geminiModelFamily = "gemini-3.1-pro"
	geminiModelFamily31FlashLite geminiModelFamily = "gemini-3.1-flash-lite"
	geminiModelFamily3Image      geminiModelFamily = "gemini-3-image"
)

type geminiModelCapabilities struct {
	Family                           geminiModelFamily
	SupportsThinkingLevel            bool
	SupportsLegacyThinkingBudget     bool
	SupportsMinimalThinkingLevel     bool
	SupportsMediaResolution          bool
	SupportsServerSideToolCalls      bool
	SupportsURLContext               bool
	SupportsImageGenerationResponses bool
}

type geminiThinkingConfigResult struct {
	Config map[string]any
	Source string
}

type geminiToolSummary struct {
	HasFunctionDeclarations bool
	BuiltInKinds            []string
}

type geminiTransformOptions struct {
	AllowURLContext bool
}

type geminiResponseAnalysis struct {
	ResponseID         string
	ModelVersion       string
	FinishReason       string
	PromptBlockReason  string
	PromptBlockMessage string
	GroundingMetadata  map[string]any
	Parts              []map[string]any
	Usage              *ClaudeUsage
	HasCandidates      bool
}

func analyzeGeminiResponse(geminiResp map[string]any, rawData []byte) geminiResponseAnalysis {
	analysis := geminiResponseAnalysis{
		Usage: extractGeminiUsage(rawData),
	}
	if geminiResp == nil {
		return analysis
	}

	analysis.ResponseID = strings.TrimSpace(stringValueFromAny(geminiResp["responseId"]))
	analysis.ModelVersion = strings.TrimSpace(stringValueFromAny(geminiResp["modelVersion"]))

	if promptFeedback, ok := geminiResp["promptFeedback"].(map[string]any); ok && promptFeedback != nil {
		analysis.PromptBlockReason = strings.TrimSpace(stringValueFromAny(promptFeedback["blockReason"]))
		analysis.PromptBlockMessage = strings.TrimSpace(stringValueFromAny(promptFeedback["blockReasonMessage"]))
	}

	if candidates, ok := geminiResp["candidates"].([]any); ok && len(candidates) > 0 {
		analysis.HasCandidates = true
		if candidate, ok := candidates[0].(map[string]any); ok && candidate != nil {
			analysis.FinishReason = strings.TrimSpace(stringValueFromAny(candidate["finishReason"]))
			if groundingMetadata, ok := candidate["groundingMetadata"].(map[string]any); ok && len(groundingMetadata) > 0 {
				analysis.GroundingMetadata = groundingMetadata
			}
			if content, ok := candidate["content"].(map[string]any); ok && content != nil {
				if partsAny, ok := content["parts"].([]any); ok && len(partsAny) > 0 {
					analysis.Parts = make([]map[string]any, 0, len(partsAny))
					for _, part := range partsAny {
						if pm, ok := part.(map[string]any); ok && pm != nil {
							analysis.Parts = append(analysis.Parts, pm)
						}
					}
				}
			}
		}
	}

	return analysis
}

func (a geminiResponseAnalysis) promptBlocked() bool {
	return strings.TrimSpace(a.PromptBlockReason) != ""
}

func (a geminiResponseAnalysis) hasRenderableParts() bool {
	for _, part := range a.Parts {
		if strings.TrimSpace(stringValueFromAny(part["text"])) != "" {
			return true
		}
		if inlineData, ok := part["inlineData"].(map[string]any); ok && inlineData != nil {
			if strings.TrimSpace(stringValueFromAny(inlineData["data"])) != "" {
				return true
			}
		}
		if functionCall, ok := part["functionCall"].(map[string]any); ok && functionCall != nil {
			return true
		}
	}
	return false
}

func buildGeminiBlockedMessage(analysis geminiResponseAnalysis) string {
	if msg := strings.TrimSpace(analysis.PromptBlockMessage); msg != "" {
		return msg
	}
	if reason := strings.TrimSpace(analysis.PromptBlockReason); reason != "" {
		return fmt.Sprintf("Gemini blocked the prompt: %s", reason)
	}
	return "Gemini blocked the prompt"
}

func buildGeminiNoCandidateMessage(analysis geminiResponseAnalysis) string {
	if analysis.promptBlocked() {
		return buildGeminiBlockedMessage(analysis)
	}
	if reason := strings.TrimSpace(analysis.FinishReason); reason != "" {
		return fmt.Sprintf("Gemini returned no candidate content (finishReason=%s)", reason)
	}
	return "Gemini returned no candidate content"
}

func buildGeminiToolUseID(functionCall map[string]any, sequence int) string {
	if functionCall != nil {
		if id := strings.TrimSpace(stringValueFromAny(functionCall["id"])); id != "" {
			return id
		}
	}
	if sequence < 1 {
		sequence = 1
	}
	return fmt.Sprintf("toolu_%04d", sequence)
}

func resolveGeminiTransformOptions(options []geminiTransformOptions) geminiTransformOptions {
	resolved := geminiTransformOptions{AllowURLContext: true}
	if len(options) == 0 {
		return resolved
	}
	resolved = options[0]
	return resolved
}

func buildGeminiToolConfig(req map[string]any, summary geminiToolSummary) map[string]any {
	if req == nil {
		return nil
	}

	out := normalizeGeminiToolConfigMap(firstNonNil(req["toolConfig"], req["tool_config"]))
	if out == nil {
		out = make(map[string]any)
	}
	if toolChoiceConfig := buildGeminiFunctionCallingConfig(firstNonNil(req["tool_choice"], req["toolChoice"])); len(toolChoiceConfig) > 0 {
		if existing, ok := out["functionCallingConfig"].(map[string]any); ok && existing != nil {
			out["functionCallingConfig"] = mergeGeminiMapsKeepExisting(existing, toolChoiceConfig)
		} else {
			out["functionCallingConfig"] = toolChoiceConfig
		}
	}

	if includeServerSideToolInvocations, ok := extractGeminiIncludeServerSideToolInvocations(req); ok {
		if _, exists := out["includeServerSideToolInvocations"]; !exists {
			out["includeServerSideToolInvocations"] = includeServerSideToolInvocations
		}
		if includeServerSideToolInvocations && summary.HasFunctionDeclarations {
			functionCallingConfig, _ := out["functionCallingConfig"].(map[string]any)
			if len(functionCallingConfig) == 0 {
				functionCallingConfig = map[string]any{}
			}
			mode := normalizeGeminiFunctionCallingMode(stringValueFromAny(functionCallingConfig["mode"]))
			if mode == "" || mode == "AUTO" {
				functionCallingConfig["mode"] = "VALIDATED"
			}
			out["functionCallingConfig"] = functionCallingConfig
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func buildGeminiFunctionCallingConfig(toolChoice any) map[string]any {
	switch value := toolChoice.(type) {
	case string:
		mode := normalizeGeminiFunctionCallingMode(value)
		if mode == "" {
			return nil
		}
		return map[string]any{"mode": mode}
	case map[string]any:
		out := make(map[string]any)
		mode := normalizeGeminiFunctionCallingMode(firstNonEmptyGeminiString(
			stringValueFromAny(value["mode"]),
			stringValueFromAny(value["type"]),
		))
		if allowedNames := normalizeGeminiAllowedFunctionNames(value["allowedFunctionNames"]); len(allowedNames) > 0 {
			out["allowedFunctionNames"] = allowedNames
		}
		if _, ok := out["allowedFunctionNames"]; !ok {
			if allowedNames := normalizeGeminiAllowedFunctionNames(value["allowed_function_names"]); len(allowedNames) > 0 {
				out["allowedFunctionNames"] = allowedNames
			}
		}
		if name := strings.TrimSpace(extractGeminiNamedToolChoice(value)); name != "" {
			mode = "ANY"
			out["allowedFunctionNames"] = []string{name}
		}
		if mode == "" && len(out) == 0 {
			return nil
		}
		if mode == "" {
			mode = "AUTO"
		}
		out["mode"] = mode
		return out
	default:
		return nil
	}
}

func normalizeGeminiFunctionCallingConfigMap(raw any) map[string]any {
	config, ok := raw.(map[string]any)
	if !ok || config == nil {
		return nil
	}
	functionCallingConfig, _ := firstNonNil(config["functionCallingConfig"], config["function_calling_config"]).(map[string]any)
	if functionCallingConfig == nil {
		return nil
	}
	return buildGeminiFunctionCallingConfig(functionCallingConfig)
}

func normalizeGeminiToolConfigMap(raw any) map[string]any {
	config, ok := deepCloneGeminiValue(raw).(map[string]any)
	if !ok || config == nil {
		return nil
	}
	if sourceConfig := normalizeGeminiFunctionCallingConfigMap(raw); len(sourceConfig) > 0 {
		if existing, ok := config["functionCallingConfig"].(map[string]any); ok && existing != nil {
			config["functionCallingConfig"] = mergeGeminiMapsKeepExisting(existing, sourceConfig)
		} else {
			config["functionCallingConfig"] = sourceConfig
		}
		delete(config, "function_calling_config")
	}
	if include, ok := extractGeminiIncludeServerSideToolInvocations(map[string]any{"toolConfig": config}); ok {
		if _, exists := config["includeServerSideToolInvocations"]; !exists {
			config["includeServerSideToolInvocations"] = include
		}
		delete(config, "include_server_side_tool_invocations")
	}
	if len(config) == 0 {
		return nil
	}
	return config
}

func mergeGeminiMapsKeepExisting(existing map[string]any, defaults map[string]any) map[string]any {
	if existing == nil {
		if defaults == nil {
			return nil
		}
		cloned, _ := deepCloneGeminiValue(defaults).(map[string]any)
		return cloned
	}
	for key, value := range defaults {
		current, exists := existing[key]
		if !exists {
			existing[key] = deepCloneGeminiValue(value)
			continue
		}
		currentMap, currentOK := current.(map[string]any)
		defaultMap, defaultOK := value.(map[string]any)
		if currentOK && defaultOK {
			existing[key] = mergeGeminiMapsKeepExisting(currentMap, defaultMap)
		}
	}
	return existing
}

func normalizeGeminiAllowedFunctionNames(value any) []string {
	switch arr := value.(type) {
	case []string:
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if trimmed := strings.TrimSpace(stringValueFromAny(item)); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return nil
	}
}

func extractGeminiNamedToolChoice(value map[string]any) string {
	if value == nil {
		return ""
	}
	if name := strings.TrimSpace(stringValueFromAny(value["name"])); name != "" {
		return name
	}
	if tool, ok := value["tool"].(map[string]any); ok && tool != nil {
		if name := strings.TrimSpace(stringValueFromAny(tool["name"])); name != "" {
			return name
		}
	}
	if function, ok := value["function"].(map[string]any); ok && function != nil {
		if name := strings.TrimSpace(stringValueFromAny(function["name"])); name != "" {
			return name
		}
	}
	return ""
}

func normalizeGeminiFunctionCallingMode(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "AUTO":
		return "AUTO"
	case "NONE":
		return "NONE"
	case "ANY", "REQUIRED", "TOOL", "FUNCTION":
		return "ANY"
	case "VALIDATED":
		return "VALIDATED"
	default:
		return ""
	}
}

func buildGeminiThinkingConfig(req map[string]any, model string) (geminiThinkingConfigResult, error) {
	if req == nil {
		return geminiThinkingConfigResult{}, nil
	}

	capabilities := detectGeminiModelCapabilities(model)
	thinking, _ := req["thinking"].(map[string]any)
	thinkingType := strings.ToLower(strings.TrimSpace(stringValueFromAny(thinking["type"])))
	reasoningEffort := normalizeOpenAIReasoningEffort(stringValueFromAny(req["reasoning_effort"]))
	explicitLevel, hasLevel := extractGeminiThinkingLevel(req, thinking)
	explicitBudget, hasBudget := extractGeminiThinkingBudget(req, thinking)
	hasThinkingInstruction := thinkingType == "enabled" || thinkingType == "adaptive" || hasLevel || hasBudget || reasoningEffort != ""

	if capabilities.SupportsThinkingLevel {
		mappedLevel := mapGeminiReasoningEffortToThinkingLevel(reasoningEffort)
		if (hasLevel || mappedLevel != "") && hasBudget {
			return geminiThinkingConfigResult{}, apicompat.NewCompatError(
				apicompat.CompatReasonGeminiThinkingConflict,
				"compat.gemini.thinking_conflict",
				"thinkingLevel and thinkingBudget cannot be used together for Gemini 3 models",
			)
		}

		finalLevel := explicitLevel
		source := ""
		if hasLevel {
			source = "level"
		}
		if finalLevel == "" && mappedLevel != "" {
			finalLevel = mappedLevel
			source = "mapped_reasoning_effort"
		}
		if finalLevel != "" {
			if finalLevel == "MINIMAL" && !capabilities.SupportsMinimalThinkingLevel {
				return geminiThinkingConfigResult{}, apicompat.NewCompatError(
					apicompat.CompatReasonGeminiMinimalThinkingUnsupported,
					"compat.gemini.minimal_thinking_unsupported",
					"thinkingLevel MINIMAL is only supported by gemini-3-flash-preview and gemini-3.1-flash-lite-preview",
				)
			}
			return geminiThinkingConfigResult{
				Config: map[string]any{
					"includeThoughts": true,
					"thinkingLevel":   finalLevel,
				},
				Source: source,
			}, nil
		}

		if hasBudget {
			if normalizedBudget, ok := normalizeGeminiThinkingBudget(model, explicitBudget, true, false); ok {
				return geminiThinkingConfigResult{
					Config: map[string]any{
						"includeThoughts": true,
						"thinkingBudget":  normalizedBudget,
					},
					Source: "legacy_budget",
				}, nil
			}
		}

		if hasThinkingInstruction {
			return geminiThinkingConfigResult{
				Config: map[string]any{"includeThoughts": true},
				Source: "thinking_type",
			}, nil
		}
		return geminiThinkingConfigResult{}, nil
	}

	if hasLevel {
		return geminiThinkingConfigResult{}, apicompat.NewCompatError(
			apicompat.CompatReasonGeminiThinkingLevelUnsupported,
			"compat.gemini.thinking_level_unsupported",
			"thinkingLevel is only supported by Gemini 3 thinking models",
		)
	}

	if reasoningEffort == "none" {
		return geminiThinkingConfigResult{}, apicompat.NewCompatError(
			apicompat.CompatReasonGeminiReasoningNoneUnsupported,
			"compat.gemini.reasoning_none_unsupported",
			"reasoning_effort=none is only supported by gemini-3-flash-preview and gemini-3.1-flash-lite-preview",
		)
	}

	if hasBudget || thinkingType == "adaptive" {
		if normalizedBudget, ok := normalizeGeminiThinkingBudget(model, explicitBudget, hasBudget, thinkingType == "adaptive"); ok {
			return geminiThinkingConfigResult{
				Config: map[string]any{
					"includeThoughts": true,
					"thinkingBudget":  normalizedBudget,
				},
				Source: "budget",
			}, nil
		}
	}

	if hasThinkingInstruction {
		return geminiThinkingConfigResult{
			Config: map[string]any{"includeThoughts": true},
			Source: "thinking_type",
		}, nil
	}

	return geminiThinkingConfigResult{}, nil
}

func normalizeGeminiThinkingBudget(model string, rawBudget int, hasBudget bool, adaptive bool) (int, bool) {
	budget := rawBudget
	if !hasBudget || budget <= 0 || adaptive {
		budget = geminiDynamicThinkingBudget
	}
	if budget <= 0 {
		return budget, true
	}

	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(normalizedModel, "gemini-2.5-flash"):
		if budget > gemini25FlashThinkingBudget {
			budget = gemini25FlashThinkingBudget
		}
	case strings.Contains(normalizedModel, "gemini-2.5-pro"):
		if budget > gemini25ProThinkingBudgetCap {
			budget = gemini25ProThinkingBudgetCap
		}
	}
	return budget, true
}

func detectGeminiModelCapabilities(model string) geminiModelCapabilities {
	normalized := normalizeGeminiModelID(model)
	capabilities := geminiModelCapabilities{}

	switch {
	case strings.HasPrefix(normalized, "gemini-3.1-flash-image"), strings.HasPrefix(normalized, "gemini-3-pro-image"):
		capabilities.Family = geminiModelFamily3Image
		capabilities.SupportsMediaResolution = true
		capabilities.SupportsImageGenerationResponses = true
	case strings.HasPrefix(normalized, "gemini-3.1-flash-lite"):
		capabilities.Family = geminiModelFamily31FlashLite
		capabilities.SupportsThinkingLevel = true
		capabilities.SupportsLegacyThinkingBudget = true
		capabilities.SupportsMinimalThinkingLevel = strings.HasPrefix(normalized, "gemini-3.1-flash-lite-preview")
		capabilities.SupportsMediaResolution = true
		capabilities.SupportsServerSideToolCalls = true
		capabilities.SupportsURLContext = true
	case strings.HasPrefix(normalized, "gemini-3.1-pro"):
		capabilities.Family = geminiModelFamily31Pro
		capabilities.SupportsThinkingLevel = true
		capabilities.SupportsLegacyThinkingBudget = true
		capabilities.SupportsMediaResolution = true
		capabilities.SupportsServerSideToolCalls = true
		capabilities.SupportsURLContext = true
	case strings.HasPrefix(normalized, "gemini-3-flash"):
		capabilities.Family = geminiModelFamily3Flash
		capabilities.SupportsThinkingLevel = true
		capabilities.SupportsLegacyThinkingBudget = true
		capabilities.SupportsMinimalThinkingLevel = strings.HasPrefix(normalized, "gemini-3-flash-preview")
		capabilities.SupportsMediaResolution = true
		capabilities.SupportsServerSideToolCalls = true
		capabilities.SupportsURLContext = true
	case strings.HasPrefix(normalized, "gemini-2.5"):
		capabilities.Family = geminiModelFamily25
		capabilities.SupportsLegacyThinkingBudget = true
	case strings.HasPrefix(normalized, "gemini-2"):
		capabilities.Family = geminiModelFamily2x
	}

	return capabilities
}

func normalizeGeminiModelID(model string) string {
	normalized := normalizeRegistryID(model)
	normalized = strings.TrimPrefix(normalized, "publishers/google/models/")
	if replacement, ok := vertexLegacyUpstreamModelAliases[normalized]; ok {
		return replacement
	}
	return normalized
}

func extractGeminiThinkingLevel(req map[string]any, thinking map[string]any) (string, bool) {
	for _, value := range []string{
		stringValueFromAny(req["thinkingLevel"]),
		stringValueFromAny(req["thinking_level"]),
		stringValueFromAny(thinking["thinkingLevel"]),
		stringValueFromAny(thinking["thinking_level"]),
		stringValueFromAny(thinking["level"]),
	} {
		if normalized, ok := normalizeGeminiThinkingLevel(value); ok {
			return normalized, true
		}
	}
	return "", false
}

func extractGeminiThinkingBudget(req map[string]any, thinking map[string]any) (int, bool) {
	for _, value := range []any{
		req["thinkingBudget"],
		req["budget_tokens"],
		req["budgetTokens"],
		thinking["thinkingBudget"],
		thinking["budget_tokens"],
		thinking["budgetTokens"],
	} {
		if budget, ok := asInt(value); ok {
			return budget, true
		}
	}
	return 0, false
}

func normalizeGeminiThinkingLevel(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", false
	}
	value = strings.NewReplacer("-", "", "_", "", " ", "").Replace(value)
	switch value {
	case "minimal", "none":
		return "MINIMAL", true
	case "low":
		return "LOW", true
	case "medium":
		return "MEDIUM", true
	case "high", "xhigh", "extrahigh":
		return "HIGH", true
	default:
		return "", false
	}
}

func mapGeminiReasoningEffortToThinkingLevel(reasoningEffort string) string {
	switch normalizeOpenAIReasoningEffort(reasoningEffort) {
	case "none":
		return "MINIMAL"
	case "low":
		return "LOW"
	case "medium":
		return "MEDIUM"
	case "high", "xhigh":
		return "HIGH"
	default:
		return ""
	}
}

func extractGeminiMediaResolution(req map[string]any, model string) (string, bool, error) {
	if req == nil {
		return "", false, nil
	}
	raw := firstNonEmptyGeminiString(
		stringValueFromAny(req["mediaResolution"]),
		stringValueFromAny(req["media_resolution"]),
	)
	if raw == "" {
		return "", false, nil
	}
	normalized, ok := normalizeGeminiMediaResolution(raw)
	if !ok {
		return "", false, apicompat.NewCompatError(
			apicompat.CompatReasonGeminiMediaResolutionInvalid,
			"compat.gemini.media_resolution_invalid",
			"mediaResolution only supports LOW, MEDIUM, or HIGH on the current Gemini route",
		)
	}
	if !detectGeminiModelCapabilities(model).SupportsMediaResolution {
		return "", false, apicompat.NewCompatError(
			apicompat.CompatReasonGeminiMediaResolutionUnsupported,
			"compat.gemini.media_resolution_unsupported",
			"mediaResolution is only supported by Gemini 3 models",
		)
	}
	return normalized, true, nil
}

func normalizeGeminiMediaResolution(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", false
	}
	value = strings.NewReplacer("-", "", "_", "", " ", "").Replace(value)
	switch value {
	case "low", "mediaresolutionlow":
		return "LOW", true
	case "medium", "mediaresolutionmedium":
		return "MEDIUM", true
	case "high", "mediaresolutionhigh":
		return "HIGH", true
	default:
		return "", false
	}
}

func extractGeminiIncludeServerSideToolInvocations(req map[string]any) (bool, bool) {
	if req == nil {
		return false, false
	}
	for _, value := range []any{
		req["includeServerSideToolInvocations"],
		req["include_server_side_tool_invocations"],
	} {
		if include, ok := coerceGeminiBool(value); ok {
			return include, true
		}
	}
	toolConfig, _ := firstNonNil(req["toolConfig"], req["tool_config"]).(map[string]any)
	if toolConfig == nil {
		return false, false
	}
	for _, value := range []any{
		toolConfig["includeServerSideToolInvocations"],
		toolConfig["include_server_side_tool_invocations"],
	} {
		if include, ok := coerceGeminiBool(value); ok {
			return include, true
		}
	}
	return false, false
}

func coerceGeminiBool(raw any) (bool, bool) {
	switch value := raw.(type) {
	case bool:
		return value, true
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	case float64:
		return value != 0, true
	case int:
		return value != 0, true
	case int64:
		return value != 0, true
	}
	return false, false
}

func normalizeGeminiBuiltInToolKind(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	value = strings.NewReplacer("-", "", "_", "", " ", "").Replace(value)
	switch {
	case strings.HasPrefix(value, "googlesearch"), strings.HasPrefix(value, "websearch"), value == "google":
		return "googleSearch"
	case strings.HasPrefix(value, "codeexecution"):
		return "codeExecution"
	case strings.HasPrefix(value, "googlemaps"), value == "maps":
		return "googleMaps"
	case strings.HasPrefix(value, "filesearch"):
		return "fileSearch"
	case strings.HasPrefix(value, "urlcontext"):
		return "urlContext"
	default:
		return ""
	}
}

func camelizeGeminiFieldName(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if !strings.ContainsAny(key, "_- ") {
		return key
	}
	parts := strings.FieldsFunc(key, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})
	if len(parts) == 0 {
		return key
	}
	out := strings.ToLower(parts[0])
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		out += strings.ToUpper(lower[:1]) + lower[1:]
	}
	return out
}

func copyGeminiStructuredOutputConfig(req map[string]any, generationConfig map[string]any) {
	if req == nil || generationConfig == nil {
		return
	}
	switch value := req["response_format"].(type) {
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "json", "json_object", "application/json":
			setGeminiValueIfMissing(generationConfig, "responseMimeType", "application/json")
		case "text", "text/plain":
			setGeminiValueIfMissing(generationConfig, "responseMimeType", "text/plain")
		}
	case map[string]any:
		if mimeType := strings.TrimSpace(stringValueFromAny(value["mime_type"])); mimeType != "" {
			setGeminiValueIfMissing(generationConfig, "responseMimeType", mimeType)
		}
		if mimeType := strings.TrimSpace(stringValueFromAny(value["mimeType"])); mimeType != "" {
			setGeminiValueIfMissing(generationConfig, "responseMimeType", mimeType)
		}
		switch strings.ToLower(strings.TrimSpace(stringValueFromAny(value["type"]))) {
		case "json", "json_object", "json_schema":
			setGeminiValueIfMissing(generationConfig, "responseMimeType", "application/json")
		case "text":
			setGeminiValueIfMissing(generationConfig, "responseMimeType", "text/plain")
		}

		schemaSource := value
		if jsonSchema, ok := value["json_schema"].(map[string]any); ok && jsonSchema != nil {
			schemaSource = jsonSchema
		}
		schema := firstNonNil(
			schemaSource["schema"],
			value["schema"],
			value["responseJsonSchema"],
			value["response_json_schema"],
		)
		if cleanedSchema := normalizeGeminiSchema(schema); cleanedSchema != nil {
			setGeminiValueIfMissing(generationConfig, "responseJsonSchema", cleanedSchema)
			setGeminiValueIfMissing(generationConfig, "responseMimeType", "application/json")
		}
	}
}

func setGeminiValueIfMissing(target map[string]any, key string, value any) {
	if target == nil || strings.TrimSpace(key) == "" {
		return
	}
	if _, exists := target[key]; exists {
		return
	}
	target[key] = deepCloneGeminiValue(value)
}

func normalizeGeminiSchema(schema any) any {
	cloned, ok := deepCloneGeminiValue(schema).(map[string]any)
	if !ok || cloned == nil {
		return nil
	}
	return antigravity.CleanJSONSchema(cloned)
}

func deepCloneGeminiValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = deepCloneGeminiValue(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			out[index] = deepCloneGeminiValue(item)
		}
		return out
	default:
		return value
	}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmptyGeminiString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
