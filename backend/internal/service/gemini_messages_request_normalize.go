package service

import (
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
)

func normalizeGeminiRequestForAIStudio(body []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}

	modified := normalizeGeminiInvalidParameters(payload)
	tools, ok := payload["tools"].([]any)
	if ok && len(tools) > 0 {
		for _, rawTool := range tools {
			tool, ok := rawTool.(map[string]any)
			if !ok {
				continue
			}
			googleSearch, ok := tool["googleSearch"]
			if !ok {
				continue
			}
			if _, exists := tool["google_search"]; exists {
				continue
			}
			tool["google_search"] = googleSearch
			delete(tool, "googleSearch")
			modified = true
		}
	}

	if !modified {
		return body
	}

	normalized, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return normalized
}

func cleanGeminiNativeRequestParameters(body []byte) []byte {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	if !normalizeGeminiInvalidParameters(payload) {
		return body
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return normalized
}

func normalizeGeminiInvalidParameters(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	modified := false
	antigravity.DeepCleanUndefined(payload)

	for _, alias := range []struct {
		from string
		to   string
	}{
		{"cached_content", "cachedContent"},
		{"generation_config", "generationConfig"},
		{"system_instruction", "systemInstruction"},
		{"tool_config", "toolConfig"},
		{"safety_settings", "safetySettings"},
	} {
		if moveGeminiAlias(payload, alias.from, alias.to) {
			modified = true
		}
	}

	if generationConfig, ok := payload["generationConfig"].(map[string]any); ok {
		if normalizeGeminiGenerationConfigAliases(generationConfig) {
			modified = true
		}
		if pruneGeminiInvalidEmptyFields(generationConfig) {
			modified = true
		}
		if len(generationConfig) == 0 {
			delete(payload, "generationConfig")
			modified = true
		}
	}
	if toolConfig, ok := payload["toolConfig"].(map[string]any); ok {
		if normalizeGeminiToolConfigAliases(toolConfig) {
			modified = true
		}
		if pruneGeminiInvalidEmptyFields(toolConfig) {
			modified = true
		}
		if len(toolConfig) == 0 {
			delete(payload, "toolConfig")
			modified = true
		}
	}
	if systemInstruction, ok := payload["systemInstruction"].(map[string]any); ok {
		if pruneGeminiInvalidEmptyFields(systemInstruction) {
			modified = true
		}
		if len(systemInstruction) == 0 {
			delete(payload, "systemInstruction")
			modified = true
		}
	}
	if tools, ok := payload["tools"].([]any); ok {
		nextTools := make([]any, 0, len(tools))
		for _, rawTool := range tools {
			tool, ok := rawTool.(map[string]any)
			if !ok {
				nextTools = append(nextTools, rawTool)
				continue
			}
			if cleanGeminiFunctionDeclarationSchemas(tool) {
				modified = true
			}
			if pruneGeminiInvalidEmptyFields(tool) {
				modified = true
			}
			if len(tool) == 0 {
				modified = true
				continue
			}
			nextTools = append(nextTools, tool)
		}
		if len(nextTools) == 0 {
			delete(payload, "tools")
			modified = true
		} else if len(nextTools) != len(tools) {
			payload["tools"] = nextTools
			modified = true
		}
	}
	if pruneGeminiInvalidEmptyFields(payload) {
		modified = true
	}
	return modified
}

func normalizeGeminiGenerationConfigAliases(config map[string]any) bool {
	if config == nil {
		return false
	}
	modified := false
	for _, alias := range []struct {
		from string
		to   string
	}{
		{"max_output_tokens", "maxOutputTokens"},
		{"top_p", "topP"},
		{"top_k", "topK"},
		{"stop_sequences", "stopSequences"},
		{"response_mime_type", "responseMimeType"},
		{"response_json_schema", "responseJsonSchema"},
		{"response_modalities", "responseModalities"},
		{"thinking_config", "thinkingConfig"},
		{"media_resolution", "mediaResolution"},
		{"image_config", "imageConfig"},
	} {
		if moveGeminiAlias(config, alias.from, alias.to) {
			modified = true
		}
	}
	if imageConfig, ok := config["imageConfig"].(map[string]any); ok {
		if moveGeminiAlias(imageConfig, "image_size", "imageSize") {
			modified = true
		}
		if pruneGeminiInvalidEmptyFields(imageConfig) {
			modified = true
		}
		if len(imageConfig) == 0 {
			delete(config, "imageConfig")
			modified = true
		}
	}
	if thinkingConfig, ok := config["thinkingConfig"].(map[string]any); ok {
		for _, alias := range []struct {
			from string
			to   string
		}{
			{"include_thoughts", "includeThoughts"},
			{"thinking_budget", "thinkingBudget"},
			{"thinking_level", "thinkingLevel"},
		} {
			if moveGeminiAlias(thinkingConfig, alias.from, alias.to) {
				modified = true
			}
		}
		if pruneGeminiInvalidEmptyFields(thinkingConfig) {
			modified = true
		}
		if len(thinkingConfig) == 0 {
			delete(config, "thinkingConfig")
			modified = true
		}
	}
	return modified
}

func normalizeGeminiToolConfigAliases(config map[string]any) bool {
	if config == nil {
		return false
	}
	modified := moveGeminiAlias(config, "function_calling_config", "functionCallingConfig")
	if moveGeminiAlias(config, "include_server_side_tool_invocations", "includeServerSideToolInvocations") {
		modified = true
	}
	if functionConfig, ok := config["functionCallingConfig"].(map[string]any); ok {
		if moveGeminiAlias(functionConfig, "allowed_function_names", "allowedFunctionNames") {
			modified = true
		}
		if pruneGeminiInvalidEmptyFields(functionConfig) {
			modified = true
		}
		if len(functionConfig) == 0 {
			delete(config, "functionCallingConfig")
			modified = true
		}
	}
	return modified
}

func cleanGeminiFunctionDeclarationSchemas(tool map[string]any) bool {
	if tool == nil {
		return false
	}
	modified := false
	for _, key := range []string{"functionDeclarations", "function_declarations"} {
		funcs, ok := tool[key].([]any)
		if !ok || len(funcs) == 0 {
			continue
		}
		if key == "function_declarations" {
			if _, exists := tool["functionDeclarations"]; !exists {
				tool["functionDeclarations"] = funcs
			}
			delete(tool, "function_declarations")
			modified = true
		}
		for _, rawFunc := range funcs {
			funcMap, ok := rawFunc.(map[string]any)
			if !ok {
				continue
			}
			if params, ok := funcMap["parameters"].(map[string]any); ok {
				antigravity.DeepCleanUndefined(params)
				funcMap["parameters"] = antigravity.CleanJSONSchema(params)
				modified = true
			}
		}
	}
	return modified
}

func moveGeminiAlias(target map[string]any, from string, to string) bool {
	if target == nil {
		return false
	}
	value, exists := target[from]
	if !exists {
		return false
	}
	if _, hasCanonical := target[to]; !hasCanonical {
		target[to] = value
	}
	delete(target, from)
	return true
}

func pruneGeminiInvalidEmptyFields(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		modified := false
		for key, item := range typed {
			if isGeminiInvalidEmptyField(key, item) {
				delete(typed, key)
				modified = true
				continue
			}
			if pruneGeminiInvalidEmptyFields(item) {
				modified = true
			}
			if isGeminiInvalidEmptyField(key, typed[key]) {
				delete(typed, key)
				modified = true
			}
		}
		return modified
	case []any:
		modified := false
		for _, item := range typed {
			if pruneGeminiInvalidEmptyFields(item) {
				modified = true
			}
		}
		return modified
	default:
		return false
	}
}

func isGeminiInvalidEmptyField(key string, value any) bool {
	if isGeminiEmptyToolConfigKey(key) {
		return value == nil
	}
	return isGeminiInvalidEmptyValue(value)
}

func isGeminiEmptyToolConfigKey(key string) bool {
	switch key {
	case "googleSearch", "google_search", "codeExecution", "code_execution", "googleMaps", "google_maps", "fileSearch", "file_search", "urlContext", "url_context":
		return true
	default:
		return false
	}
}

func isGeminiInvalidEmptyValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case map[string]any:
		return len(typed) == 0
	case []any:
		return len(typed) == 0
	default:
		return false
	}
}
