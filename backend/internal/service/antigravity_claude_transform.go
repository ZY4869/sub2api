package service

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"strings"
)

func stripThinkingFromClaudeRequest(req *antigravity.ClaudeRequest) (bool, error) {
	if req == nil {
		return false, nil
	}
	changed := false
	if req.Thinking != nil {
		req.Thinking = nil
		changed = true
	}
	for i := range req.Messages {
		raw := req.Messages[i].Content
		if len(raw) == 0 {
			continue
		}
		var str string
		if json.Unmarshal(raw, &str) == nil {
			continue
		}
		var blocks []map[string]any
		if err := json.Unmarshal(raw, &blocks); err != nil {
			continue
		}
		filtered := make([]map[string]any, 0, len(blocks))
		modifiedAny := false
		for _, block := range blocks {
			t, _ := block["type"].(string)
			switch t {
			case "thinking":
				thinkingText, _ := block["thinking"].(string)
				if thinkingText != "" {
					filtered = append(filtered, map[string]any{"type": "text", "text": thinkingText})
				}
				modifiedAny = true
			case "redacted_thinking":
				modifiedAny = true
			case "":
				if thinkingText, hasThinking := block["thinking"].(string); hasThinking {
					if thinkingText != "" {
						filtered = append(filtered, map[string]any{"type": "text", "text": thinkingText})
					}
					modifiedAny = true
				} else {
					filtered = append(filtered, block)
				}
			default:
				filtered = append(filtered, block)
			}
		}
		if !modifiedAny {
			continue
		}
		if len(filtered) == 0 {
			filtered = append(filtered, map[string]any{"type": "text", "text": "(content removed)"})
		}
		newRaw, err := json.Marshal(filtered)
		if err != nil {
			return changed, err
		}
		req.Messages[i].Content = newRaw
		changed = true
	}
	return changed, nil
}
func stripSignatureSensitiveBlocksFromClaudeRequest(req *antigravity.ClaudeRequest) (bool, error) {
	if req == nil {
		return false, nil
	}
	changed := false
	if req.Thinking != nil {
		req.Thinking = nil
		changed = true
	}
	for i := range req.Messages {
		raw := req.Messages[i].Content
		if len(raw) == 0 {
			continue
		}
		var str string
		if json.Unmarshal(raw, &str) == nil {
			continue
		}
		var blocks []map[string]any
		if err := json.Unmarshal(raw, &blocks); err != nil {
			continue
		}
		filtered := make([]map[string]any, 0, len(blocks))
		modifiedAny := false
		for _, block := range blocks {
			t, _ := block["type"].(string)
			switch t {
			case "thinking":
				thinkingText, _ := block["thinking"].(string)
				if thinkingText != "" {
					filtered = append(filtered, map[string]any{"type": "text", "text": thinkingText})
				}
				modifiedAny = true
			case "redacted_thinking":
				modifiedAny = true
			case "tool_use":
				name, _ := block["name"].(string)
				id, _ := block["id"].(string)
				input := block["input"]
				inputJSON, _ := json.Marshal(input)
				text := "(tool_use)"
				if name != "" {
					text += " name=" + name
				}
				if id != "" {
					text += " id=" + id
				}
				if len(inputJSON) > 0 && string(inputJSON) != "null" {
					text += " input=" + string(inputJSON)
				}
				filtered = append(filtered, map[string]any{"type": "text", "text": text})
				modifiedAny = true
			case "tool_result":
				toolUseID, _ := block["tool_use_id"].(string)
				isError, _ := block["is_error"].(bool)
				content := block["content"]
				contentJSON, _ := json.Marshal(content)
				text := "(tool_result)"
				if toolUseID != "" {
					text += " tool_use_id=" + toolUseID
				}
				if isError {
					text += " is_error=true"
				}
				if len(contentJSON) > 0 && string(contentJSON) != "null" {
					text += "\n" + string(contentJSON)
				}
				filtered = append(filtered, map[string]any{"type": "text", "text": text})
				modifiedAny = true
			case "":
				if thinkingText, hasThinking := block["thinking"].(string); hasThinking {
					if thinkingText != "" {
						filtered = append(filtered, map[string]any{"type": "text", "text": thinkingText})
					}
					modifiedAny = true
				} else {
					filtered = append(filtered, block)
				}
			default:
				filtered = append(filtered, block)
			}
		}
		if !modifiedAny {
			continue
		}
		if len(filtered) == 0 {
			filtered = append(filtered, map[string]any{"type": "text", "text": "(content removed)"})
		}
		newRaw, err := json.Marshal(filtered)
		if err != nil {
			return changed, err
		}
		req.Messages[i].Content = newRaw
		changed = true
	}
	return changed, nil
}
func (s *AntigravityGatewayService) extractImageSize(body []byte) string {
	var req antigravity.GeminiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return "2K"
	}
	if req.GenerationConfig != nil && req.GenerationConfig.ImageConfig != nil {
		size := strings.ToUpper(strings.TrimSpace(req.GenerationConfig.ImageConfig.ImageSize))
		if size == "1K" || size == "2K" || size == "4K" {
			return size
		}
	}
	return "2K"
}
func isImageGenerationModel(model string) bool {
	modelLower := strings.ToLower(model)
	modelLower = strings.TrimPrefix(modelLower, "models/")
	return modelLower == "gemini-3.1-flash-image" || modelLower == "gemini-3.1-flash-image-preview" || strings.HasPrefix(modelLower, "gemini-3.1-flash-image-") || modelLower == "gemini-3-pro-image" || modelLower == "gemini-3-pro-image-preview" || strings.HasPrefix(modelLower, "gemini-3-pro-image-") || modelLower == "gemini-2.5-flash-image" || modelLower == "gemini-2.5-flash-image-preview" || strings.HasPrefix(modelLower, "gemini-2.5-flash-image-")
}
func cleanGeminiRequest(body []byte) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	modified := false
	if tools, ok := payload["tools"].([]any); ok && len(tools) > 0 {
		for _, t := range tools {
			toolMap, ok := t.(map[string]any)
			if !ok {
				continue
			}
			var funcs []any
			if f, ok := toolMap["functionDeclarations"].([]any); ok {
				funcs = f
			} else if f, ok := toolMap["function_declarations"].([]any); ok {
				funcs = f
			}
			if len(funcs) == 0 {
				continue
			}
			for _, f := range funcs {
				funcMap, ok := f.(map[string]any)
				if !ok {
					continue
				}
				if params, ok := funcMap["parameters"].(map[string]any); ok {
					antigravity.DeepCleanUndefined(params)
					cleaned := antigravity.CleanJSONSchema(params)
					funcMap["parameters"] = cleaned
					modified = true
				}
			}
		}
	}
	if !modified {
		return body, nil
	}
	return json.Marshal(payload)
}
func filterEmptyPartsFromGeminiRequest(body []byte) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	contents, ok := payload["contents"].([]any)
	if !ok || len(contents) == 0 {
		return body, nil
	}
	filtered := make([]any, 0, len(contents))
	modified := false
	for _, c := range contents {
		contentMap, ok := c.(map[string]any)
		if !ok {
			filtered = append(filtered, c)
			continue
		}
		parts, hasParts := contentMap["parts"]
		if !hasParts {
			filtered = append(filtered, c)
			continue
		}
		partsSlice, ok := parts.([]any)
		if !ok {
			filtered = append(filtered, c)
			continue
		}
		if len(partsSlice) == 0 {
			modified = true
			continue
		}
		filtered = append(filtered, c)
	}
	if !modified {
		return body, nil
	}
	payload["contents"] = filtered
	return json.Marshal(payload)
}
