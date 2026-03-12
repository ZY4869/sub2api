package service

import (
	"encoding/json"
	"errors"
	"strings"
)

func convertClaudeMessagesToGeminiGenerateContent(body []byte) ([]byte, error) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	toolUseIDToName := make(map[string]string)
	systemText := extractClaudeSystemText(req["system"])
	contents, err := convertClaudeMessagesToGeminiContents(req["messages"], toolUseIDToName)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any)
	if systemText != "" {
		out["systemInstruction"] = map[string]any{"parts": []any{map[string]any{"text": systemText}}}
	}
	out["contents"] = contents
	if tools := convertClaudeToolsToGeminiTools(req["tools"]); tools != nil {
		out["tools"] = tools
	}
	generationConfig := convertClaudeGenerationConfig(req)
	if generationConfig != nil {
		out["generationConfig"] = generationConfig
	}
	stripGeminiFunctionIDs(out)
	return json.Marshal(out)
}
func stripGeminiFunctionIDs(req map[string]any) {
	contents, ok := req["contents"].([]any)
	if !ok {
		return
	}
	for _, c := range contents {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		contentParts, ok := cm["parts"].([]any)
		if !ok {
			continue
		}
		for _, p := range contentParts {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if fc, ok := pm["functionCall"].(map[string]any); ok && fc != nil {
				delete(fc, "id")
			}
			if fr, ok := pm["functionResponse"].(map[string]any); ok && fr != nil {
				delete(fr, "id")
			}
		}
	}
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
func convertClaudeMessagesToGeminiContents(messages any, toolUseIDToName map[string]string) ([]any, error) {
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
				case "tool_use":
					id, _ := bm["id"].(string)
					name, _ := bm["name"].(string)
					if strings.TrimSpace(id) != "" && strings.TrimSpace(name) != "" {
						toolUseIDToName[id] = name
					}
					signature, _ := bm["signature"].(string)
					signature = strings.TrimSpace(signature)
					if signature == "" {
						signature = geminiDummyThoughtSignature
					}
					parts = append(parts, map[string]any{"thoughtSignature": signature, "functionCall": map[string]any{"name": name, "args": bm["input"]}})
				case "tool_result":
					toolUseID, _ := bm["tool_use_id"].(string)
					name := toolUseIDToName[toolUseID]
					if name == "" {
						name = "tool"
					}
					parts = append(parts, map[string]any{"functionResponse": map[string]any{"name": name, "response": map[string]any{"content": extractClaudeContentText(bm["content"])}}})
				case "image":
					if src, ok := bm["source"].(map[string]any); ok {
						if srcType, _ := src["type"].(string); srcType == "base64" {
							mediaType, _ := src["media_type"].(string)
							data, _ := src["data"].(string)
							if mediaType != "" && data != "" {
								parts = append(parts, map[string]any{"inlineData": map[string]any{"mimeType": mediaType, "data": data}})
							}
						}
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
func convertClaudeToolsToGeminiTools(tools any) []any {
	arr, ok := tools.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	funcDecls := make([]any, 0, len(arr))
	for _, t := range arr {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		var name, desc string
		var params any
		toolType, _ := tm["type"].(string)
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
		cleanedParams := cleanToolSchema(params)
		funcDecls = append(funcDecls, map[string]any{"name": name, "description": desc, "parameters": cleanedParams})
	}
	if len(funcDecls) == 0 {
		return nil
	}
	return []any{map[string]any{"functionDeclarations": funcDecls}}
}
func cleanToolSchema(schema any) any {
	if schema == nil {
		return nil
	}
	switch v := schema.(type) {
	case map[string]any:
		cleaned := make(map[string]any)
		for key, value := range v {
			if key == "$schema" || key == "$id" || key == "$ref" || key == "additionalProperties" || key == "minLength" || key == "maxLength" || key == "minItems" || key == "maxItems" {
				continue
			}
			cleaned[key] = cleanToolSchema(value)
		}
		if typeVal, ok := cleaned["type"].(string); ok {
			cleaned["type"] = strings.ToUpper(typeVal)
		}
		return cleaned
	case []any:
		cleaned := make([]any, len(v))
		for i, item := range v {
			cleaned[i] = cleanToolSchema(item)
		}
		return cleaned
	default:
		return v
	}
}
func convertClaudeGenerationConfig(req map[string]any) map[string]any {
	out := make(map[string]any)
	if mt, ok := asInt(req["max_tokens"]); ok && mt > 0 {
		out["maxOutputTokens"] = mt
	}
	if temp, ok := req["temperature"].(float64); ok {
		out["temperature"] = temp
	}
	if topP, ok := req["top_p"].(float64); ok {
		out["topP"] = topP
	}
	if stopSeq, ok := req["stop_sequences"].([]any); ok && len(stopSeq) > 0 {
		out["stopSequences"] = stopSeq
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
