package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ForceOpenAIResponsesImageToolModel(body []byte, targetModel string) ([]byte, error) {
	return rewriteOpenAIResponsesImageToolModel(body, targetModel, true)
}

func RewriteOpenAIResponsesImageToolModel(body []byte, targetModel string) ([]byte, error) {
	return rewriteOpenAIResponsesImageToolModel(body, targetModel, false)
}

func rewriteOpenAIResponsesImageToolModel(body []byte, targetModel string, stripInputFidelity bool) ([]byte, error) {
	if len(body) == 0 || !json.Valid(body) || strings.TrimSpace(targetModel) == "" {
		return body, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		return nil, fmt.Errorf("invalid json body")
	}
	tools, ok := payload["tools"].([]any)
	if !ok || len(tools) == 0 {
		return body, nil
	}
	modified := false
	for _, raw := range tools {
		tool, ok := raw.(map[string]any)
		if !ok || strings.TrimSpace(stringValueFromAny(tool["type"])) != "image_generation" {
			continue
		}
		if stringValueFromAny(tool["model"]) != targetModel {
			tool["model"] = targetModel
			modified = true
		}
		if stripInputFidelity {
			if _, exists := tool["input_fidelity"]; !exists {
				continue
			}
			delete(tool, "input_fidelity")
			modified = true
		}
	}
	if !modified {
		return body, nil
	}
	return json.Marshal(payload)
}

func NormalizeOpenAIResponsesImageToolRequest(body []byte) (*NormalizedImageRequest, error) {
	if len(body) == 0 || !json.Valid(body) {
		return nil, fmt.Errorf("invalid json body")
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		return nil, fmt.Errorf("invalid json body")
	}
	tool := firstOpenAIResponsesImageTool(body)
	if tool == nil {
		return nil, fmt.Errorf("image_generation tool is required")
	}
	req := &NormalizedImageRequest{
		Operation:      ResolveOpenAIResponsesImageToolAction(body),
		DisplayModelID: stringValueFromAny(payload["model"]),
		TargetModelID:  stringValueFromAny(tool["model"]),
		Prompt:         extractOpenAIResponsesImageToolPrompt(payload["input"]),
		Images:         extractOpenAIResponsesImageToolImages(payload["input"]),
		Mask:           strings.TrimSpace(stringValueFromAny(tool["input_image_mask"])),
		Size:           stringValueFromAny(tool["size"]),
		Quality:        stringValueFromAny(tool["quality"]),
		Background:     stringValueFromAny(tool["background"]),
		OutputFormat:   stringValueFromAny(tool["output_format"]),
		Moderation:     stringValueFromAny(tool["moderation"]),
		InputFidelity:  stringValueFromAny(tool["input_fidelity"]),
		Stream:         parseExtraBool(payload["stream"]),
	}
	req.OutputCompression = optionalIntFromAny(tool["output_compression"])
	req.PartialImages = optionalIntFromAny(tool["partial_images"])
	req.N = optionalIntFromAny(tool["n"])
	return req, nil
}

func ResolveOpenAIResponsesImageToolAction(body []byte) string {
	tool := firstOpenAIResponsesImageTool(body)
	if tool == nil {
		return ""
	}
	switch strings.TrimSpace(strings.ToLower(stringValueFromAny(tool["action"]))) {
	case "edit":
		return "edit"
	case "generate":
		return "generate"
	}
	if stringValueFromAny(tool["input_image_mask"]) != "" {
		return "edit"
	}
	return "generate"
}

func ResolveOpenAIResponsesImageToolSizeTier(body []byte) string {
	tool := firstOpenAIResponsesImageTool(body)
	if tool == nil {
		return ""
	}
	return ResolveOpenAIImageSizeTier(stringValueFromAny(tool["size"]))
}

func firstOpenAIResponsesImageTool(body []byte) map[string]any {
	if len(body) == 0 || !json.Valid(body) {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		return nil
	}
	tools, ok := payload["tools"].([]any)
	if !ok {
		return nil
	}
	for _, raw := range tools {
		tool, ok := raw.(map[string]any)
		if ok && strings.TrimSpace(stringValueFromAny(tool["type"])) == "image_generation" {
			return tool
		}
	}
	return nil
}

func extractOpenAIResponsesImageToolPrompt(raw any) string {
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		for _, item := range typed {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if prompt := extractOpenAIResponsesImageToolPrompt(itemMap["content"]); prompt != "" {
				return prompt
			}
			itemType := strings.TrimSpace(stringValueFromAny(itemMap["type"]))
			if itemType == "input_text" {
				return strings.TrimSpace(stringValueFromAny(itemMap["text"]))
			}
		}
	case map[string]any:
		return extractOpenAIResponsesImageToolPrompt(typed["content"])
	}
	return ""
}

func extractOpenAIResponsesImageToolImages(raw any) []string {
	switch typed := raw.(type) {
	case []any:
		result := make([]string, 0, 4)
		for _, item := range typed {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			itemType := strings.TrimSpace(stringValueFromAny(itemMap["type"]))
			if itemType == "input_image" {
				if imageURL := strings.TrimSpace(stringValueFromAny(itemMap["image_url"])); imageURL != "" {
					result = append(result, imageURL)
				}
				continue
			}
			result = append(result, extractOpenAIResponsesImageToolImages(itemMap["content"])...)
		}
		return result
	case map[string]any:
		return extractOpenAIResponsesImageToolImages(typed["content"])
	default:
		return nil
	}
}
