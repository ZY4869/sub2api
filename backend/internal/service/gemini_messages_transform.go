package service

import (
	"encoding/json"
	"errors"
	"strings"
)

func convertClaudeMessagesToGeminiGenerateContent(body []byte, options ...geminiTransformOptions) ([]byte, error) {
	transformOptions := resolveGeminiTransformOptions(options)
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	toolUseIDToName := make(map[string]string)
	systemText := extractClaudeSystemText(req["system"])
	contents, err := convertClaudeMessagesToGeminiContents(req["messages"], toolUseIDToName, true)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any)
	if systemText != "" {
		out["systemInstruction"] = map[string]any{"parts": []any{map[string]any{"text": systemText}}}
	}
	out["contents"] = contents
	tools, toolSummary, err := convertClaudeToolsToGeminiTools(req["tools"], transformOptions)
	if err != nil {
		return nil, err
	}
	if tools != nil {
		out["tools"] = tools
	}
	if toolConfig := buildGeminiToolConfig(req, toolSummary); toolConfig != nil {
		out["toolConfig"] = toolConfig
	}
	generationConfig, err := convertClaudeGenerationConfig(req)
	if err != nil {
		return nil, err
	}
	if generationConfig != nil {
		out["generationConfig"] = generationConfig
	}
	return json.Marshal(out)
}

func extractClaudeSystemText(system any) string {
	switch v := system.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		var parts []string
		for _, p := range v {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := pm["type"].(string); t != "text" {
				continue
			}
			if text, ok := pm["text"].(string); ok && strings.TrimSpace(text) != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func convertClaudeMessagesToGeminiContents(messages any, toolUseIDToName map[string]string, allowDummyThought bool) ([]any, error) {
	arr, ok := messages.([]any)
	if !ok {
		return nil, errors.New("messages must be an array")
	}
	out := make([]any, 0, len(arr))
	for _, m := range arr {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		role, _ := mm["role"].(string)
		role = strings.ToLower(strings.TrimSpace(role))
		gRole := "user"
		if role == "assistant" {
			gRole = "model"
		}
		parts := make([]any, 0)
		switch content := mm["content"].(type) {
		case string:
			parts = append(parts, map[string]any{"text": content})
		case []any:
			singleBlock := len(content) == 1
			for _, block := range content {
				bm, ok := block.(map[string]any)
				if !ok {
					continue
				}
				bt, _ := bm["type"].(string)
				switch bt {
				case "text":
					if text, ok := bm["text"].(string); ok {
						if singleBlock || strings.TrimSpace(text) != "" {
							parts = append(parts, map[string]any{"text": text})
						}
					}
				case "thinking":
					thinkingText, _ := bm["thinking"].(string)
					if strings.TrimSpace(thinkingText) == "" {
						continue
					}
					part := map[string]any{
						"text":    thinkingText,
						"thought": true,
					}
					signature, _ := bm["signature"].(string)
					signature = strings.TrimSpace(signature)
					if signature == "" && allowDummyThought {
						signature = geminiDummyThoughtSignature
					}
					if signature != "" {
						part["thoughtSignature"] = signature
					}
					parts = append(parts, part)
				case "tool_use":
					id, _ := bm["id"].(string)
					name, _ := bm["name"].(string)
					if strings.TrimSpace(id) != "" && strings.TrimSpace(name) != "" {
						toolUseIDToName[id] = name
					}
					signature, _ := bm["signature"].(string)
					signature = strings.TrimSpace(signature)
					if signature == "" && allowDummyThought {
						signature = geminiDummyThoughtSignature
					}
					functionCall := map[string]any{
						"name": name,
						"args": bm["input"],
					}
					if strings.TrimSpace(id) != "" {
						functionCall["id"] = id
					}
					part := map[string]any{
						"functionCall": functionCall,
					}
					if signature != "" {
						part["thoughtSignature"] = signature
					}
					parts = append(parts, part)
				case "tool_result":
					toolUseID, _ := bm["tool_use_id"].(string)
					name := toolUseIDToName[toolUseID]
					if name == "" {
						name = "tool"
					}
					response := map[string]any{
						"content": extractClaudeToolResultPayload(bm["content"]),
					}
					if isError, ok := bm["is_error"].(bool); ok && isError {
						response["isError"] = true
					}
					functionResponse := map[string]any{
						"name":     name,
						"response": response,
					}
					if strings.TrimSpace(toolUseID) != "" {
						functionResponse["id"] = toolUseID
					}
					parts = append(parts, map[string]any{"functionResponse": functionResponse})
				case "image":
					if imagePart := convertClaudeSourceToGeminiPart(bm["source"]); imagePart != nil {
						parts = append(parts, imagePart)
					}
				case "document", "file":
					if filePart := convertClaudeSourceToGeminiPart(firstNonNil(bm["source"], bm["file"])); filePart != nil {
						parts = append(parts, filePart)
					}
				default:
					if b, err := json.Marshal(bm); err == nil {
						parts = append(parts, map[string]any{"text": string(b)})
					}
				}
			}
		default:
		}
		out = append(out, map[string]any{"role": gRole, "parts": parts})
	}
	return out, nil
}

func extractClaudeContentText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		var sb strings.Builder
		for _, part := range t {
			pm, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if pm["type"] == "text" {
				if text, ok := pm["text"].(string); ok {
					_, _ = sb.WriteString(text)
				}
			}
		}
		return sb.String()
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func extractClaudeToolResultPayload(v any) any {
	switch content := v.(type) {
	case string:
		return content
	case []any:
		items := make([]map[string]any, 0, len(content))
		for _, rawPart := range content {
			part, ok := rawPart.(map[string]any)
			if !ok || part == nil {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(stringValueFromAny(part["type"]))) {
			case "text":
				text := stringValueFromAny(part["text"])
				if strings.TrimSpace(text) != "" {
					items = append(items, map[string]any{"type": "text", "text": text})
				}
			case "image", "document", "file":
				if converted := convertClaudeSourceToToolResultItem(firstNonNil(part["source"], part["file"])); converted != nil {
					items = append(items, converted)
				}
			default:
				items = append(items, map[string]any{"type": "json", "value": deepCloneGeminiValue(part)})
			}
		}
		if len(items) == 1 && items[0]["type"] == "text" {
			return items[0]["text"]
		}
		if len(items) > 0 {
			return map[string]any{"items": items}
		}
		return extractClaudeContentText(v)
	default:
		return v
	}
}

func convertClaudeSourceToGeminiPart(raw any) map[string]any {
	source, ok := raw.(map[string]any)
	if !ok || source == nil {
		return nil
	}
	sourceType := strings.ToLower(strings.TrimSpace(stringValueFromAny(source["type"])))
	mimeType := firstNonEmptyString(
		stringValueFromAny(source["media_type"]),
		stringValueFromAny(source["mime_type"]),
		stringValueFromAny(source["mimeType"]),
	)
	switch sourceType {
	case "base64":
		data := strings.TrimSpace(stringValueFromAny(source["data"]))
		if mimeType == "" || data == "" {
			return nil
		}
		return map[string]any{"inlineData": map[string]any{"mimeType": mimeType, "data": data}}
	case "url", "file", "uri":
		fileURI := firstNonEmptyString(
			stringValueFromAny(source["url"]),
			stringValueFromAny(source["uri"]),
			stringValueFromAny(source["file_uri"]),
			stringValueFromAny(source["fileUri"]),
		)
		if fileURI == "" {
			return nil
		}
		fileData := map[string]any{"fileUri": fileURI}
		if mimeType != "" {
			fileData["mimeType"] = mimeType
		}
		return map[string]any{"fileData": fileData}
	default:
		return nil
	}
}

func convertClaudeSourceToToolResultItem(raw any) map[string]any {
	source, ok := raw.(map[string]any)
	if !ok || source == nil {
		return nil
	}
	sourceType := strings.ToLower(strings.TrimSpace(stringValueFromAny(source["type"])))
	switch sourceType {
	case "base64":
		data := strings.TrimSpace(stringValueFromAny(source["data"]))
		mimeType := firstNonEmptyString(
			stringValueFromAny(source["media_type"]),
			stringValueFromAny(source["mime_type"]),
			stringValueFromAny(source["mimeType"]),
		)
		if data == "" {
			return nil
		}
		return map[string]any{"type": "image", "mimeType": mimeType, "data": data}
	case "url", "file", "uri":
		fileURI := firstNonEmptyString(
			stringValueFromAny(source["url"]),
			stringValueFromAny(source["uri"]),
			stringValueFromAny(source["file_uri"]),
			stringValueFromAny(source["fileUri"]),
		)
		if fileURI == "" {
			return nil
		}
		out := map[string]any{"type": "file", "fileUri": fileURI}
		if mimeType := firstNonEmptyString(
			stringValueFromAny(source["media_type"]),
			stringValueFromAny(source["mime_type"]),
			stringValueFromAny(source["mimeType"]),
		); mimeType != "" {
			out["mimeType"] = mimeType
		}
		return out
	default:
		return nil
	}
}

func convertClaudeToolsToGeminiTools(tools any, options geminiTransformOptions) ([]any, geminiToolSummary, error) {
	arr, ok := tools.([]any)
	if !ok || len(arr) == 0 {
		return nil, geminiToolSummary{}, nil
	}
	funcDecls := make([]any, 0, len(arr))
	builtInToolConfigs := make(map[string]map[string]any)
	builtInToolOrder := make([]string, 0, 4)
	for _, t := range arr {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		if builtInKind := extractGeminiBuiltInToolKind(tm); builtInKind != "" {
			if builtInKind == "urlContext" && !options.AllowURLContext {
				return nil, geminiToolSummary{}, errors.New("urlContext is only available on Gemini API accounts, not Vertex Gemini channels")
			}
			if _, exists := builtInToolConfigs[builtInKind]; !exists {
				builtInToolOrder = append(builtInToolOrder, builtInKind)
			}
			builtInToolConfigs[builtInKind] = mergeStringAnyMap(
				builtInToolConfigs[builtInKind],
				buildGeminiBuiltInToolConfig(tm, builtInKind),
			)
			continue
		}
		toolType := strings.ToLower(strings.TrimSpace(stringValueFromAny(tm["type"])))
		var name, desc string
		var params any
		if toolType == "custom" {
			custom, ok := tm["custom"].(map[string]any)
			if !ok {
				continue
			}
			name, _ = tm["name"].(string)
			desc, _ = custom["description"].(string)
			params = custom["input_schema"]
		} else {
			name, _ = tm["name"].(string)
			desc, _ = tm["description"].(string)
			params = tm["input_schema"]
		}
		if name == "" {
			continue
		}
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		cleanedParams := normalizeGeminiSchema(params)
		if cleanedParams == nil {
			cleanedParams = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		funcDecls = append(funcDecls, map[string]any{"name": name, "description": desc, "parameters": cleanedParams})
	}
	if len(funcDecls) == 0 && len(builtInToolOrder) == 0 {
		return nil, geminiToolSummary{}, nil
	}
	toolsOut := make([]any, 0, 1+len(builtInToolOrder))
	if len(funcDecls) > 0 {
		toolsOut = append(toolsOut, map[string]any{"functionDeclarations": funcDecls})
	}
	for _, kind := range builtInToolOrder {
		config := builtInToolConfigs[kind]
		if len(config) == 0 {
			config = map[string]any{}
		}
		toolsOut = append(toolsOut, map[string]any{kind: config})
	}
	return toolsOut, geminiToolSummary{
		HasFunctionDeclarations: len(funcDecls) > 0,
		BuiltInKinds:            append([]string(nil), builtInToolOrder...),
	}, nil
}

func convertClaudeGenerationConfig(req map[string]any) (map[string]any, error) {
	out := make(map[string]any)
	if mt, ok := asInt(firstNonNil(req["max_tokens"], req["maxTokens"])); ok && mt > 0 {
		out["maxOutputTokens"] = mt
	}
	if temp, ok := req["temperature"].(float64); ok {
		out["temperature"] = temp
	}
	if topP, ok := firstNonNil(req["top_p"], req["topP"]).(float64); ok {
		out["topP"] = topP
	}
	if topK, ok := asInt(firstNonNil(req["top_k"], req["topK"])); ok && topK > 0 {
		out["topK"] = topK
	}
	thinkingConfig, err := buildGeminiThinkingConfig(req, stringValueFromAny(req["model"]))
	if err != nil {
		return nil, err
	}
	if thinkingConfig.Config != nil {
		out["thinkingConfig"] = thinkingConfig.Config
	}
	if mediaResolution, ok, err := extractGeminiMediaResolution(req, stringValueFromAny(req["model"])); err != nil {
		return nil, err
	} else if ok {
		out["mediaResolution"] = mediaResolution
	}
	if stopSeq, ok := firstNonNil(req["stop_sequences"], req["stopSequences"]).([]any); ok && len(stopSeq) > 0 {
		out["stopSequences"] = normalizeClaudeStringList(stopSeq)
	}
	copyGeminiStructuredOutputConfig(req, out)
	if len(out) == 1 {
		if _, ok := out["responseMimeType"]; ok {
			return out, nil
		}
		if _, ok := out["responseJsonSchema"]; ok {
			return out, nil
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func normalizeClaudeStringList(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(stringValueFromAny(value)); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func extractGeminiBuiltInToolKind(tool map[string]any) string {
	if tool == nil {
		return ""
	}
	for _, key := range []string{"googleSearch", "codeExecution", "googleMaps", "fileSearch", "urlContext"} {
		if _, exists := tool[key]; exists {
			return key
		}
	}
	for _, raw := range []string{
		stringValueFromAny(tool["type"]),
		stringValueFromAny(tool["name"]),
	} {
		if kind := normalizeGeminiBuiltInToolKind(raw); kind != "" {
			return kind
		}
	}
	return ""
}

func buildGeminiBuiltInToolConfig(tool map[string]any, kind string) map[string]any {
	if tool == nil || kind == "" {
		return nil
	}
	for _, value := range []any{
		tool[kind],
		tool[camelizeGeminiFieldName(kind)],
		tool[strings.ToLower(kind)],
		tool["config"],
		tool["options"],
	} {
		if config, ok := deepCloneGeminiValue(value).(map[string]any); ok && config != nil {
			return config
		}
	}

	out := make(map[string]any)
	for key, value := range tool {
		switch key {
		case "type", "name", "description", "input_schema", "inputSchema", "custom", kind, strings.ToLower(kind):
			continue
		}
		if strings.TrimSpace(key) == "" {
			continue
		}
		out[camelizeGeminiFieldName(key)] = deepCloneGeminiValue(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
