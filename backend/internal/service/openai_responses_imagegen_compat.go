package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	openAIResponsesImagegenPrefix = "$imagegen "

	OpenAIResponsesImagegenCompatSourceJSONShorthand  = "json_shorthand"
	OpenAIResponsesImagegenCompatSourceModelShorthand = "model_shorthand"
	OpenAIResponsesImagegenCompatSourceStructured     = "structured_json"
	OpenAIResponsesImagegenCompatSourceMultipart      = "multipart"

	openAIResponsesReferenceImageUploadLimitBytes      = 20 * 1024 * 1024
	openAIResponsesReferenceImageUploadTotalLimitBytes = 40 * 1024 * 1024
	openAIResponsesReferenceImageNormalizedLimitBytes  = 8 * 1024 * 1024
	openAIResponsesReferenceImageUploadLimitCount      = 4
	openAIResponsesReferenceImageMaxDimension          = 2048
)

var openAIResponsesImagegenToolOptionKeys = []string{
	"action",
	"size",
	"quality",
	"background",
	"output_format",
	"output_compression",
	"partial_images",
	"moderation",
	"input_image_mask",
}

var openAIResponsesImagegenAcceptedOptionKeys = []string{
	"action",
	"aspect_ratio",
	"image_size",
	"size",
	"quality",
	"background",
	"output_format",
	"output_compression",
	"partial_images",
	"moderation",
	"input_fidelity",
	"input_image_mask",
	"n",
}

type OpenAIResponsesCompatMetadata struct {
	Enabled                   bool
	Source                    string
	SourceGuess               string
	Rejected                  bool
	RejectCode                string
	ReferenceImageCount       int
	ReferenceImageBytesBefore int64
	ReferenceImageBytesAfter  int64
	ReferenceImagesNormalized bool
	ImageGenerationSize       string
}

type OpenAIResponsesCompatResult struct {
	Body        []byte
	ContentType string
	ParsedBody  map[string]any
	Metadata    OpenAIResponsesCompatMetadata
	TraceTool   map[string]any
}

type OpenAIResponsesCompatError struct {
	Status   int
	Type     string
	Code     string
	Message  string
	Metadata OpenAIResponsesCompatMetadata
}

func (e *OpenAIResponsesCompatError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NormalizeOpenAIResponsesImageGenCompat(body []byte, contentType string) (*OpenAIResponsesCompatResult, error) {
	result := &OpenAIResponsesCompatResult{
		Body:        body,
		ContentType: strings.TrimSpace(contentType),
	}

	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return nil, newOpenAIResponsesCompatErrorWithMetadata(
				http.StatusBadRequest,
				"invalid_request_error",
				"",
				"missing multipart boundary",
				OpenAIResponsesCompatMetadata{
					Rejected:    true,
					SourceGuess: OpenAIResponsesImagegenCompatSourceMultipart,
				},
			)
		}
		return normalizeOpenAIResponsesMultipartImagegenCompat(body, boundary)
	}

	if len(body) == 0 || !json.Valid(body) {
		return result, nil
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return result, nil
	}
	if reqBody == nil {
		return result, nil
	}

	normalized, compatErr := normalizeOpenAIResponsesJSONImagegenCompat(reqBody)
	if compatErr != nil {
		enrichOpenAIResponsesCompatRejectMetadata(
			compatErr,
			OpenAIResponsesCompatMetadata{
				Rejected:            true,
				SourceGuess:         guessOpenAIResponsesCompatJSONSource(reqBody),
				ReferenceImageCount: countOpenAIResponsesCompatReferenceImageCandidates(reqBody),
			},
		)
		return nil, compatErr
	}
	if normalized == nil {
		result.ParsedBody = reqBody
		return result, nil
	}

	encoded, err := json.Marshal(normalized.ParsedBody)
	if err != nil {
		return nil, fmt.Errorf("marshal responses compat body: %w", err)
	}
	normalized.Body = encoded
	normalized.ContentType = "application/json"
	return normalized, nil
}

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

func normalizeOpenAIResponsesMultipartImagegenCompat(body []byte, boundary string) (*OpenAIResponsesCompatResult, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)

	fields := make(map[string]any)
	imageGeneration := make(map[string]any)
	referenceImages := make([]string, 0, 4)
	fileCount := 0
	metadata := OpenAIResponsesCompatMetadata{
		Enabled:     true,
		Source:      OpenAIResponsesImagegenCompatSourceMultipart,
		SourceGuess: OpenAIResponsesImagegenCompatSourceMultipart,
	}
	rejectError := func(status int, errType string, code string, message string) error {
		rejectMetadata := metadata
		rejectMetadata.Enabled = false
		rejectMetadata.Rejected = true
		rejectMetadata.RejectCode = strings.TrimSpace(code)
		rejectMetadata.ReferenceImagesNormalized = false
		rejectMetadata.ReferenceImageCount = maxOpenAIResponsesCompatReferenceImageCount(len(referenceImages), fileCount)
		return newOpenAIResponsesCompatErrorWithMetadata(status, errType, code, message, rejectMetadata)
	}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read responses multipart body: %w", err)
		}

		name := strings.TrimSpace(part.FormName())
		if name == "" {
			_ = part.Close()
			continue
		}

		if part.FileName() != "" {
			if name != "reference_image" && name != "mask" {
				_ = part.Close()
				return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "", fmt.Sprintf("unsupported multipart file field %q", name))
			}
			if name == "reference_image" {
				fileCount++
			}
			rawBytes, readErr := io.ReadAll(part)
			_ = part.Close()
			if readErr != nil {
				return nil, fmt.Errorf("read multipart reference image: %w", readErr)
			}
			if int64(len(rawBytes)) > openAIResponsesReferenceImageUploadLimitBytes {
				if name == "mask" {
					return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "input_image_mask_too_large", fmt.Sprintf("mask image %q exceeds the 20MB upload limit", part.FileName()))
				}
				return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "reference_image_too_large", fmt.Sprintf("reference image %q exceeds the 20MB upload limit", part.FileName()))
			}
			if name == "reference_image" && fileCount > openAIResponsesReferenceImageUploadLimitCount {
				return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "reference_image_count_exceeded", fmt.Sprintf("reference_image accepts at most %d files", openAIResponsesReferenceImageUploadLimitCount))
			}
			if name == "reference_image" {
				metadata.ReferenceImageBytesBefore += int64(len(rawBytes))
				if metadata.ReferenceImageBytesBefore > openAIResponsesReferenceImageUploadTotalLimitBytes {
					return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "reference_image_total_too_large", "total reference image upload size exceeds 40MB")
				}
			}
			dataURI, normalizedBytes, normalizeErr := normalizeResponsesCompatReferenceImage(rawBytes)
			if normalizeErr != nil {
				return nil, withOpenAIResponsesCompatRejectMetadata(normalizeErr, metadata, maxOpenAIResponsesCompatReferenceImageCount(len(referenceImages), fileCount))
			}
			if name == "mask" {
				imageGeneration["input_image_mask"] = dataURI
				continue
			}
			if len(referenceImages)+1 > openAIResponsesReferenceImageUploadLimitCount {
				return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "reference_image_count_exceeded", fmt.Sprintf("reference images accept at most %d items", openAIResponsesReferenceImageUploadLimitCount))
			}
			metadata.ReferenceImageBytesAfter += normalizedBytes
			metadata.ReferenceImagesNormalized = true
			referenceImages = append(referenceImages, dataURI)
			continue
		}

		valueBytes, readErr := io.ReadAll(part)
		_ = part.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read multipart field %q: %w", name, readErr)
		}
		value := strings.TrimSpace(string(valueBytes))

		switch name {
		case "image_generation":
			if value == "" {
				continue
			}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(value), &parsed); err != nil || parsed == nil {
				return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "", "image_generation must be a valid JSON object")
			}
			imageGeneration = mergeResponsesCompatToolOptions(imageGeneration, parsed)
		case "reference_image_url":
			if value == "" {
				continue
			}
			validated, err := validateResponsesCompatReferenceImageURL(value)
			if err != nil {
				return nil, withOpenAIResponsesCompatRejectMetadata(err, metadata, len(referenceImages))
			}
			if len(referenceImages)+1 > openAIResponsesReferenceImageUploadLimitCount {
				return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "reference_image_count_exceeded", fmt.Sprintf("reference images accept at most %d items", openAIResponsesReferenceImageUploadLimitCount))
			}
			referenceImages = append(referenceImages, validated)
		case "size", "image_size", "aspect_ratio", "quality", "background", "output_format", "output_compression", "partial_images", "action", "moderation", "input_fidelity", "n":
			if value == "" {
				continue
			}
			if _, exists := imageGeneration[name]; !exists {
				imageGeneration[name] = parseResponsesMultipartFieldValue(value)
			}
		case "mask", "mask_image_url", "input_image_mask":
			if value == "" {
				continue
			}
			if _, exists := imageGeneration["input_image_mask"]; exists {
				continue
			}
			validated, err := validateResponsesCompatReferenceImageURL(value)
			if err != nil {
				return nil, withOpenAIResponsesCompatRejectMetadata(err, metadata, len(referenceImages))
			}
			imageGeneration["input_image_mask"] = validated
		default:
			fields[name] = parseResponsesMultipartFieldValue(value)
		}
	}

	if hasExplicitResponsesCompatTools(fields) {
		return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "imagegen_compat_conflict", "explicit tools cannot be combined with multipart imagegen compatibility fields")
	}

	modelValue, ok := fields["model"]
	if !ok || strings.TrimSpace(scalarString(modelValue)) == "" {
		return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "", "model is required")
	}
	inputValue, ok := fields["input"]
	if !ok || strings.TrimSpace(scalarString(inputValue)) == "" {
		return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "", "input is required")
	}

	if streamRaw, exists := fields["stream"]; exists {
		if streamValue, ok := streamRaw.(bool); ok && streamValue {
			return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "multipart_stream_unsupported", "multipart responses image generation does not support stream=true")
		}
	}

	inputText := scalarString(inputValue)
	if !strings.HasPrefix(inputText, openAIResponsesImagegenPrefix) {
		return nil, rejectError(http.StatusBadRequest, "invalid_request_error", "imagegen_compat_requires_prefix", "multipart image generation requires input starting with $imagegen ")
	}
	fields["input"] = buildResponsesCompatInputMessage(strings.TrimPrefix(inputText, openAIResponsesImagegenPrefix), referenceImages)
	if compatErr := normalizeResponsesCompatImageGenerationOptions(imageGeneration); compatErr != nil {
		return nil, withOpenAIResponsesCompatRejectMetadata(compatErr, metadata, metadata.ReferenceImageCount)
	}
	fields["tools"] = []any{buildResponsesCompatImageGenerationTool(imageGeneration)}
	fields["tool_choice"] = map[string]any{"type": "image_generation"}

	metadata.ReferenceImageCount = len(referenceImages)
	metadata.ImageGenerationSize = scalarString(imageGeneration["size"])

	return &OpenAIResponsesCompatResult{
		Body:        mustMarshalResponsesCompatBody(fields),
		ContentType: "application/json",
		ParsedBody:  fields,
		Metadata:    metadata,
		TraceTool:   buildResponsesCompatTraceImageGenerationTool(imageGeneration),
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

func buildResponsesCompatImageGenerationTool(imageGeneration map[string]any) map[string]any {
	tool := map[string]any{"type": "image_generation"}
	for _, key := range openAIResponsesImagegenToolOptionKeys {
		if value, exists := imageGeneration[key]; exists && value != nil {
			tool[key] = value
		}
	}
	return tool
}

func mergeResponsesCompatToolOptions(base map[string]any, updates map[string]any) map[string]any {
	if base == nil {
		base = make(map[string]any)
	}
	for _, key := range openAIResponsesImagegenAcceptedOptionKeys {
		if value, exists := updates[key]; exists && value != nil {
			base[key] = value
		}
	}
	return base
}

func normalizeResponsesCompatImageGenerationOptions(imageGeneration map[string]any) *OpenAIResponsesCompatError {
	if imageGeneration == nil {
		return nil
	}

	if normalizedSize, _, errCode, errMessage := normalizeOpenAIImageSizeWithAspect(
		scalarString(imageGeneration["size"]),
		scalarString(imageGeneration["image_size"]),
		scalarString(imageGeneration["aspect_ratio"]),
	); errCode != "" {
		return newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", errCode, errMessage)
	} else if strings.TrimSpace(normalizedSize) != "" {
		imageGeneration["size"] = normalizedSize
	}
	delete(imageGeneration, "image_size")
	delete(imageGeneration, "aspect_ratio")

	if rawN, exists := imageGeneration["n"]; exists && rawN != nil {
		normalized, ok := coerceResponsesCompatInt(rawN)
		if !ok {
			return newOpenAIResponsesCompatError(
				http.StatusBadRequest,
				"invalid_request_error",
				"imagegen_compat_invalid_n",
				"image_generation.n must be an integer",
			)
		}
		if normalized < 1 {
			return newOpenAIResponsesCompatError(
				http.StatusBadRequest,
				"invalid_request_error",
				"imagegen_compat_invalid_n",
				"image_generation.n must be at least 1",
			)
		}
		if normalized != 1 {
			return newOpenAIResponsesCompatError(
				http.StatusBadRequest,
				"invalid_request_error",
				"image_n_not_supported",
				"image_generation.n is not supported; remove it or set it to 1",
			)
		}
		delete(imageGeneration, "n")
	}

	intOptions := []struct {
		key string
		min int
		max int
	}{
		{key: "output_compression", min: 0, max: 100},
		{key: "partial_images", min: 0, max: 3},
	}

	for _, option := range intOptions {
		raw, exists := imageGeneration[option.key]
		if !exists || raw == nil {
			continue
		}

		normalized, err := normalizeResponsesCompatIntOption(option.key, raw, option.min, option.max)
		if err != nil {
			return err
		}
		imageGeneration[option.key] = normalized
	}

	return nil
}

func normalizeResponsesCompatIntOption(key string, raw any, min int, max int) (int, *OpenAIResponsesCompatError) {
	value, ok := coerceResponsesCompatInt(raw)
	if !ok {
		return 0, newOpenAIResponsesCompatError(
			http.StatusBadRequest,
			"invalid_request_error",
			"imagegen_compat_invalid_"+key,
			fmt.Sprintf("image_generation.%s must be an integer", key),
		)
	}
	if value < min || value > max {
		return 0, newOpenAIResponsesCompatError(
			http.StatusBadRequest,
			"invalid_request_error",
			"imagegen_compat_invalid_"+key,
			fmt.Sprintf("image_generation.%s must be between %d and %d", key, min, max),
		)
	}
	return value, nil
}

func coerceResponsesCompatInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		if v != math.Trunc(v) {
			return 0, false
		}
		return int(v), true
	case float32:
		value := float64(v)
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0, false
		}
		if value != math.Trunc(value) {
			return 0, false
		}
		return int(value), true
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func buildResponsesCompatTraceImageGenerationTool(imageGeneration map[string]any) map[string]any {
	tool := map[string]any{"type": "image_generation"}
	for _, key := range openAIResponsesImagegenAcceptedOptionKeys {
		if value, exists := imageGeneration[key]; exists && value != nil {
			tool[key] = value
		}
	}
	return tool
}

func parseResponsesCompatSingleImageSource(raw any) (string, *OpenAIResponsesCompatError) {
	switch typed := raw.(type) {
	case string:
		return validateResponsesCompatReferenceImageURL(typed)
	case map[string]any:
		for _, key := range []string{"image_url", "input_image_mask", "mask"} {
			if value := scalarString(typed[key]); value != "" {
				return validateResponsesCompatReferenceImageURL(value)
			}
		}
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "mask must be a valid image_url object")
	default:
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "mask must be a string or {image_url} object")
	}
}

func validateResponsesCompatReferenceImageURL(raw string) (string, *OpenAIResponsesCompatError) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference image_url must not be empty")
	}
	lowerValue := strings.ToLower(value)
	if strings.HasPrefix(lowerValue, "data:image/") {
		if strings.Contains(lowerValue, ",") {
			return value, nil
		}
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference image data URI is invalid")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.Host == "" {
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference image_url must be an http(s) URL or data URI")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return value, nil
	default:
		return "", newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "reference image_url must be an http(s) URL or data URI")
	}
}

func normalizeResponsesCompatReferenceImage(rawBytes []byte) (string, int64, *OpenAIResponsesCompatError) {
	mediaType := strings.ToLower(strings.TrimSpace(http.DetectContentType(rawBytes)))
	switch mediaType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return "", 0, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "unsupported_reference_image_type", "reference_image only supports JPEG, PNG, or WebP uploads")
	}

	img, _, err := image.Decode(bytes.NewReader(rawBytes))
	if err != nil {
		return "", 0, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "unsupported_reference_image_type", "reference_image could not be decoded as JPEG, PNG, or WebP")
	}

	bounds := img.Bounds()
	targetWidth, targetHeight := fitResponsesCompatImageSize(bounds.Dx(), bounds.Dy(), openAIResponsesReferenceImageMaxDimension)
	resized := img
	if targetWidth != bounds.Dx() || targetHeight != bounds.Dy() {
		if imageHasTransparentPixels(img) {
			dst := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)
			resized = dst
		} else {
			dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Src, nil)
			resized = dst
		}
	}

	var (
		buffer     bytes.Buffer
		outputType string
		encodeErr  error
		hasAlpha   = imageHasTransparentPixels(resized)
	)
	if hasAlpha {
		outputType = "image/png"
		encodeErr = png.Encode(&buffer, resized)
	} else {
		outputType = "image/jpeg"
		encodeErr = jpeg.Encode(&buffer, resized, &jpeg.Options{Quality: 82})
	}
	if encodeErr != nil {
		return "", 0, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "", "failed to normalize reference_image")
	}
	if int64(buffer.Len()) > openAIResponsesReferenceImageNormalizedLimitBytes {
		return "", 0, newOpenAIResponsesCompatError(http.StatusBadRequest, "invalid_request_error", "reference_image_too_large_after_normalization", "reference_image is still larger than 8MB after normalization")
	}

	encoded := base64.StdEncoding.EncodeToString(buffer.Bytes())
	return "data:" + outputType + ";base64," + encoded, int64(buffer.Len()), nil
}

func fitResponsesCompatImageSize(width int, height int, maxDimension int) (int, int) {
	if width <= 0 || height <= 0 || maxDimension <= 0 {
		return width, height
	}
	if width <= maxDimension && height <= maxDimension {
		return width, height
	}
	if width >= height {
		nextWidth := maxDimension
		nextHeight := int(float64(height) * (float64(maxDimension) / float64(width)))
		if nextHeight < 1 {
			nextHeight = 1
		}
		return nextWidth, nextHeight
	}
	nextHeight := maxDimension
	nextWidth := int(float64(width) * (float64(maxDimension) / float64(height)))
	if nextWidth < 1 {
		nextWidth = 1
	}
	return nextWidth, nextHeight
}

func imageHasTransparentPixels(img image.Image) bool {
	if img == nil {
		return false
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha != 0xffff {
				return true
			}
		}
	}
	return false
}

func hasExplicitResponsesCompatTools(reqBody map[string]any) bool {
	if reqBody == nil {
		return false
	}
	raw, exists := reqBody["tools"]
	return exists && raw != nil
}

func parseResponsesMultipartFieldValue(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
		return parsed
	}
	return raw
}

func scalarString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	case float64:
		return strings.TrimSpace(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", v), "0"), "."))
	case float32:
		return strings.TrimSpace(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", v), "0"), "."))
	case int:
		return fmt.Sprintf("%d", v)
	case int8:
		return fmt.Sprintf("%d", v)
	case int16:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case uint:
		return fmt.Sprintf("%d", v)
	case uint8:
		return fmt.Sprintf("%d", v)
	case uint16:
		return fmt.Sprintf("%d", v)
	case uint32:
		return fmt.Sprintf("%d", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func mustMarshalResponsesCompatBody(reqBody map[string]any) []byte {
	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return []byte(`{}`)
	}
	return encoded
}

func newOpenAIResponsesCompatError(status int, errType string, code string, message string) *OpenAIResponsesCompatError {
	return newOpenAIResponsesCompatErrorWithMetadata(status, errType, code, message, OpenAIResponsesCompatMetadata{})
}

func newOpenAIResponsesCompatErrorWithMetadata(status int, errType string, code string, message string, metadata OpenAIResponsesCompatMetadata) *OpenAIResponsesCompatError {
	if metadata.Rejected {
		metadata.Enabled = false
		if strings.TrimSpace(metadata.RejectCode) == "" {
			metadata.RejectCode = strings.TrimSpace(code)
		}
	}
	return &OpenAIResponsesCompatError{
		Status:   status,
		Type:     strings.TrimSpace(errType),
		Code:     strings.TrimSpace(code),
		Message:  strings.TrimSpace(message),
		Metadata: metadata,
	}
}

func enrichOpenAIResponsesCompatRejectMetadata(target *OpenAIResponsesCompatError, metadata OpenAIResponsesCompatMetadata) {
	if target == nil {
		return
	}
	target.Metadata = mergeOpenAIResponsesCompatMetadata(target.Metadata, metadata)
	if target.Metadata.Rejected && strings.TrimSpace(target.Metadata.RejectCode) == "" {
		target.Metadata.RejectCode = strings.TrimSpace(target.Code)
	}
}

func withOpenAIResponsesCompatRejectMetadata(err error, metadata OpenAIResponsesCompatMetadata, referenceImageCount int) error {
	var compatErr *OpenAIResponsesCompatError
	if !errors.As(err, &compatErr) {
		return err
	}
	metadata.Rejected = true
	metadata.Enabled = false
	metadata.ReferenceImagesNormalized = false
	if referenceImageCount > metadata.ReferenceImageCount {
		metadata.ReferenceImageCount = referenceImageCount
	}
	enrichOpenAIResponsesCompatRejectMetadata(compatErr, metadata)
	return compatErr
}

func mergeOpenAIResponsesCompatMetadata(current OpenAIResponsesCompatMetadata, updates OpenAIResponsesCompatMetadata) OpenAIResponsesCompatMetadata {
	if current.Source == "" {
		current.Source = strings.TrimSpace(updates.Source)
	}
	if current.SourceGuess == "" {
		current.SourceGuess = strings.TrimSpace(updates.SourceGuess)
	}
	if updates.Rejected {
		current.Rejected = true
	}
	if current.RejectCode == "" {
		current.RejectCode = strings.TrimSpace(updates.RejectCode)
	}
	if updates.ReferenceImageCount > current.ReferenceImageCount {
		current.ReferenceImageCount = updates.ReferenceImageCount
	}
	if updates.ReferenceImageBytesBefore > current.ReferenceImageBytesBefore {
		current.ReferenceImageBytesBefore = updates.ReferenceImageBytesBefore
	}
	if updates.ReferenceImageBytesAfter > current.ReferenceImageBytesAfter {
		current.ReferenceImageBytesAfter = updates.ReferenceImageBytesAfter
	}
	if updates.ReferenceImagesNormalized {
		current.ReferenceImagesNormalized = true
	}
	if current.ImageGenerationSize == "" {
		current.ImageGenerationSize = strings.TrimSpace(updates.ImageGenerationSize)
	}
	return current
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

func maxOpenAIResponsesCompatReferenceImageCount(values ...int) int {
	maxValue := 0
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}
