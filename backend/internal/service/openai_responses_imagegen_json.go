package service

import (
	"net/http"
	"strings"
)

func normalizeOpenAIResponsesJSONImagegenCompat(reqBody map[string]any) (*OpenAIResponsesCompatResult, *OpenAIResponsesCompatError) {
	isModelShorthand := isOpenAIResponsesImagegenModelShorthand(reqBody)
	imageGeneration, hasImageGeneration, compatErr := parseResponsesCompatImageGenerationField(reqBody)
	if compatErr != nil {
		return nil, compatErr
	}
	referenceImages, hasReferenceImages, compatErr := parseResponsesCompatReferenceImagesField(reqBody)
	if compatErr != nil {
		return nil, compatErr
	}
	maskImage, hasMaskImage, compatErr := parseResponsesCompatImageMaskField(reqBody)
	if compatErr != nil {
		return nil, compatErr
	}
	hasCompatFields := hasImageGeneration || hasReferenceImages || hasMaskImage
	hasToolsField := hasExplicitResponsesCompatTools(reqBody)
	if hasToolsField && hasCompatFields && !isModelShorthand {
		return nil, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "imagegen_compat_conflict", "explicit tools cannot be combined with image_generation, reference_images, or mask fields")
	}
	if hasToolsField && !isModelShorthand {
		return nil, nil
	}
	if isModelShorthand {
		if compatErr := validateOpenAIResponsesImagegenModelShorthandToolChoice(reqBody); compatErr != nil {
			return nil, compatErr
		}
	}
	if hasMaskImage {
		if imageGeneration == nil {
			imageGeneration = map[string]any{}
		}
		if scalarString(imageGeneration["input_image_mask"]) == "" {
			imageGeneration["input_image_mask"] = maskImage
		}
	}

	inputValue, hasInput := reqBody["input"]
	if !hasInput {
		if isModelShorthand {
			return nil, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "input is required")
		}
		if hasCompatFields {
			return nil, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "imagegen_compat_requires_prefix", "image_generation, reference_images, or mask fields require an input starting with $imagegen ")
		}
		return nil, nil
	}

	triggered := false
	source := ""
	switch input := inputValue.(type) {
	case string:
		prompt := input
		if strings.HasPrefix(prompt, openAIResponsesImagegenPrefix) {
			prompt = strings.TrimPrefix(prompt, openAIResponsesImagegenPrefix)
			reqBody["input"] = buildResponsesCompatInputMessage(prompt, referenceImages)
			triggered = true
			if isModelShorthand {
				source = OpenAIResponsesImagegenCompatSourceModelShorthand
			} else {
				source = OpenAIResponsesImagegenCompatSourceJSONShorthand
			}
			break
		}
		if isModelShorthand {
			if len(referenceImages) > 0 {
				reqBody["input"] = buildResponsesCompatInputMessage(prompt, referenceImages)
			}
			triggered = true
			source = OpenAIResponsesImagegenCompatSourceModelShorthand
		}
	case []any:
		nextInput, changed := rewriteResponsesCompatInputItems(input, referenceImages)
		if changed {
			reqBody["input"] = nextInput
			triggered = true
			if isModelShorthand {
				source = OpenAIResponsesImagegenCompatSourceModelShorthand
			} else {
				source = OpenAIResponsesImagegenCompatSourceStructured
			}
			break
		}
		if isModelShorthand {
			if len(referenceImages) > 0 {
				nextInput, changed = appendResponsesCompatReferenceImagesToInputItems(input, referenceImages)
				if changed {
					reqBody["input"] = nextInput
				}
			}
			triggered = true
			source = OpenAIResponsesImagegenCompatSourceModelShorthand
		}
	default:
		if isModelShorthand {
			return nil, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "input must be a string or array")
		}
	}

	if !triggered {
		if hasCompatFields {
			return nil, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "imagegen_compat_requires_prefix", "image_generation, reference_images, or mask fields require an input starting with $imagegen ")
		}
		return nil, nil
	}

	targetModel := ""
	if isModelShorthand {
		targetModel = OpenAICompatImageTargetModel
	}
	if compatErr := normalizeResponsesCompatImageGenerationOptions(imageGeneration); compatErr != nil {
		return nil, compatErr
	}
	imageTool := buildResponsesCompatImageGenerationTool(imageGeneration)
	if targetModel != "" {
		imageTool["model"] = targetModel
	}

	if hasToolsField {
		tools, ok := reqBody["tools"].([]any)
		if !ok {
			return nil, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "tools must be an array")
		}
		found := false
		for idx, rawTool := range tools {
			toolMap, ok := rawTool.(map[string]any)
			if !ok || strings.TrimSpace(scalarString(toolMap["type"])) != "image_generation" {
				continue
			}
			found = true
			if compatErr := normalizeResponsesCompatImageGenerationOptions(toolMap); compatErr != nil {
				return nil, compatErr
			}
			if targetModel != "" {
				existingToolModel := scalarString(toolMap["model"])
				if existingToolModel != "" && !strings.EqualFold(existingToolModel, targetModel) {
					return nil, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "imagegen_compat_tool_model_conflict", "image_generation tool model must match request model")
				}
				if existingToolModel == "" {
					toolMap["model"] = targetModel
				}
			}
			for _, key := range openAIResponsesImagegenToolOptionKeys {
				if _, exists := toolMap[key]; exists {
					continue
				}
				if value, exists := imageGeneration[key]; exists && value != nil {
					toolMap[key] = value
				}
			}
			tools[idx] = toolMap
			break
		}
		if !found {
			tools = append(tools, imageTool)
		}
		reqBody["tools"] = tools
	} else {
		reqBody["tools"] = []any{imageTool}
	}
	reqBody["tool_choice"] = map[string]any{"type": "image_generation"}
	delete(reqBody, "image_generation")
	delete(reqBody, "reference_images")
	delete(reqBody, "mask")
	delete(reqBody, "input_image_mask")

	return &OpenAIResponsesCompatResult{
		ParsedBody: reqBody,
		TraceTool:  buildResponsesCompatTraceImageGenerationTool(imageGeneration),
		Metadata: OpenAIResponsesCompatMetadata{
			Enabled:             true,
			Source:              source,
			ReferenceImageCount: len(referenceImages),
			ImageGenerationSize: scalarString(imageGeneration["size"]),
		},
	}, nil
}

func parseResponsesCompatImageGenerationField(reqBody map[string]any) (map[string]any, bool, *OpenAIResponsesCompatError) {
	raw, exists := reqBody["image_generation"]
	if !exists || raw == nil {
		return nil, false, nil
	}
	parsed, ok := raw.(map[string]any)
	if !ok || parsed == nil {
		return nil, false, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "image_generation must be a JSON object")
	}
	return mergeResponsesCompatToolOptions(nil, parsed), true, nil
}

func parseResponsesCompatReferenceImagesField(reqBody map[string]any) ([]string, bool, *OpenAIResponsesCompatError) {
	raw, exists := reqBody["reference_images"]
	if !exists || raw == nil {
		return nil, false, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, false, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference_images must be an array of {image_url} objects")
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return nil, false, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference_images must be an array of {image_url} objects")
		}
		imageURL := scalarString(itemMap["image_url"])
		validated, err := validateResponsesCompatReferenceImageURL(imageURL)
		if err != nil {
			return nil, false, err
		}
		result = append(result, validated)
	}
	return result, true, nil
}

func parseResponsesCompatImageMaskField(reqBody map[string]any) (string, bool, *OpenAIResponsesCompatError) {
	for _, key := range []string{"input_image_mask", "mask"} {
		raw, exists := reqBody[key]
		if !exists || raw == nil {
			continue
		}
		value, err := parseResponsesCompatSingleImageSource(raw)
		if err != nil {
			return "", false, err
		}
		return value, true, nil
	}
	return "", false, nil
}

func buildResponsesCompatInputMessage(prompt string, referenceImages []string) []any {
	content := make([]any, 0, 1+len(referenceImages))
	content = append(content, map[string]any{
		"type": "input_text",
		"text": prompt,
	})
	for _, imageURL := range referenceImages {
		content = append(content, map[string]any{
			"type":      "input_image",
			"image_url": imageURL,
		})
	}
	return []any{
		map[string]any{
			"type":    "message",
			"role":    "user",
			"content": content,
		},
	}
}

func rewriteResponsesCompatInputItems(items []any, referenceImages []string) ([]any, bool) {
	for index, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		itemType := strings.TrimSpace(scalarString(itemMap["type"]))
		role := strings.TrimSpace(scalarString(itemMap["role"]))
		switch {
		case role == "user" || (itemType == "message" && role == "user"):
			content, ok := itemMap["content"].([]any)
			if !ok {
				continue
			}
			nextContent, changed := rewriteResponsesCompatInputTextSlice(content, referenceImages)
			if !changed {
				continue
			}
			itemMap["content"] = nextContent
			nextItems := append([]any(nil), items...)
			nextItems[index] = itemMap
			return nextItems, true
		case itemType == "input_text":
			text := scalarString(itemMap["text"])
			if !strings.HasPrefix(text, openAIResponsesImagegenPrefix) {
				continue
			}
			itemMap["text"] = strings.TrimPrefix(text, openAIResponsesImagegenPrefix)
			nextItems := make([]any, 0, len(items)+len(referenceImages))
			nextItems = append(nextItems, items[:index]...)
			nextItems = append(nextItems, itemMap)
			for _, imageURL := range referenceImages {
				nextItems = append(nextItems, map[string]any{
					"type":      "input_image",
					"image_url": imageURL,
				})
			}
			nextItems = append(nextItems, items[index+1:]...)
			return nextItems, true
		}
	}
	return items, false
}

func rewriteResponsesCompatInputTextSlice(parts []any, referenceImages []string) ([]any, bool) {
	for index, part := range parts {
		partMap, ok := part.(map[string]any)
		if !ok || strings.TrimSpace(scalarString(partMap["type"])) != "input_text" {
			continue
		}
		text := scalarString(partMap["text"])
		if !strings.HasPrefix(text, openAIResponsesImagegenPrefix) {
			continue
		}
		partMap["text"] = strings.TrimPrefix(text, openAIResponsesImagegenPrefix)
		nextParts := make([]any, 0, len(parts)+len(referenceImages))
		nextParts = append(nextParts, parts[:index]...)
		nextParts = append(nextParts, partMap)
		for _, imageURL := range referenceImages {
			nextParts = append(nextParts, map[string]any{
				"type":      "input_image",
				"image_url": imageURL,
			})
		}
		nextParts = append(nextParts, parts[index+1:]...)
		return nextParts, true
	}
	return parts, false
}

func guessOpenAIResponsesCompatJSONSource(reqBody map[string]any) string {
	if reqBody == nil {
		return OpenAIResponsesImagegenCompatSourceJSONShorthand
	}
	switch reqBody["input"].(type) {
	case []any:
		return OpenAIResponsesImagegenCompatSourceStructured
	default:
		return OpenAIResponsesImagegenCompatSourceJSONShorthand
	}
}

func isOpenAIResponsesImagegenModelShorthand(reqBody map[string]any) bool {
	if reqBody == nil {
		return false
	}
	return strings.EqualFold(scalarString(reqBody["model"]), OpenAICompatImageTargetModel)
}

func validateOpenAIResponsesImagegenModelShorthandToolChoice(reqBody map[string]any) *OpenAIResponsesCompatError {
	if reqBody == nil {
		return nil
	}
	raw, exists := reqBody["tool_choice"]
	if !exists || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case string:
		choice := strings.TrimSpace(typed)
		if choice == "" || strings.EqualFold(choice, "image_generation") {
			return nil
		}
	case map[string]any:
		if len(typed) == 0 || strings.EqualFold(scalarString(typed["type"]), "image_generation") {
			return nil
		}
	default:
	}
	return newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "imagegen_compat_tool_choice_conflict", "tool_choice must be image_generation when model is gpt-image-2")
}

func appendResponsesCompatReferenceImagesToInputItems(items []any, referenceImages []string) ([]any, bool) {
	if len(referenceImages) == 0 {
		return items, false
	}
	for index, item := range items {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role := strings.TrimSpace(scalarString(itemMap["role"]))
		if role != "user" {
			continue
		}
		content, ok := itemMap["content"].([]any)
		if !ok {
			continue
		}
		nextContent := append([]any(nil), content...)
		for _, imageURL := range referenceImages {
			nextContent = append(nextContent, map[string]any{
				"type":      "input_image",
				"image_url": imageURL,
			})
		}
		itemMap["content"] = nextContent
		nextItems := append([]any(nil), items...)
		nextItems[index] = itemMap
		return nextItems, true
	}
	nextItems := append([]any(nil), items...)
	for _, imageURL := range referenceImages {
		nextItems = append(nextItems, map[string]any{
			"type":      "input_image",
			"image_url": imageURL,
		})
	}
	return nextItems, true
}

func countOpenAIResponsesCompatReferenceImageCandidates(reqBody map[string]any) int {
	if reqBody == nil {
		return 0
	}
	raw, ok := reqBody["reference_images"]
	if !ok || raw == nil {
		return 0
	}
	items, ok := raw.([]any)
	if !ok {
		return 0
	}
	return len(items)
}
