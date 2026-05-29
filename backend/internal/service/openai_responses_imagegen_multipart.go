package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

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
