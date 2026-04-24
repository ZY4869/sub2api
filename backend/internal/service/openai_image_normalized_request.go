package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
)

const openAIImageUploadMaxCount = 16

type NormalizedImageRequest struct {
	Operation         string
	DisplayModelID    string
	TargetModelID     string
	Prompt            string
	Images            []string
	Mask              string
	Size              string
	Quality           string
	Background        string
	OutputFormat      string
	OutputCompression *int
	PartialImages     *int
	N                 *int
	Moderation        string
	InputFidelity     string
	Stream            bool
}

func NormalizeOpenAIImageRequest(body []byte, contentType string, operation string) (*NormalizedImageRequest, error) {
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}
	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/form-data") {
		boundary := strings.TrimSpace(params["boundary"])
		if boundary == "" {
			return nil, fmt.Errorf("missing multipart boundary")
		}
		return normalizeOpenAIImageRequestMultipart(body, boundary, operation)
	}
	return normalizeOpenAIImageRequestJSON(body, operation)
}

func normalizeOpenAIImageRequestJSON(body []byte, operation string) (*NormalizedImageRequest, error) {
	var payload map[string]any
	if len(body) == 0 || !json.Valid(body) {
		return nil, fmt.Errorf("invalid json body")
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		return nil, fmt.Errorf("invalid json body")
	}
	rawImageSize := stringValueFromAny(payload["image_size"])
	rawAspectRatio := stringValueFromAny(payload["aspect_ratio"])
	req := &NormalizedImageRequest{
		Operation:      strings.TrimSpace(operation),
		DisplayModelID: stringValueFromAny(payload["model"]),
		Prompt:         stringValueFromAny(payload["prompt"]),
		Size:           stringValueFromAny(payload["size"]),
		Quality:        stringValueFromAny(payload["quality"]),
		Background:     stringValueFromAny(payload["background"]),
		OutputFormat:   stringValueFromAny(payload["output_format"]),
		Moderation:     stringValueFromAny(payload["moderation"]),
		InputFidelity:  stringValueFromAny(payload["input_fidelity"]),
		Stream:         parseExtraBool(payload["stream"]),
	}
	req.OutputCompression = optionalIntFromAny(payload["output_compression"])
	req.PartialImages = optionalIntFromAny(payload["partial_images"])
	req.N = optionalIntFromAny(payload["n"])
	req.Images = append(req.Images, collectJSONImageSources(payload["images"])...)
	if len(req.Images) == 0 {
		req.Images = append(req.Images, collectJSONImageSources(payload["image"])...)
	}
	req.Mask = firstJSONImageSource(payload["mask"])

	if normalizedSize, _, errCode, errMessage := normalizeOpenAIImageSizeWithAspect(req.Size, rawImageSize, rawAspectRatio); errCode != "" {
		return nil, newOpenAIImageRequestError(errCode, errMessage)
	} else if strings.TrimSpace(normalizedSize) != "" {
		req.Size = normalizedSize
	}
	if req.DisplayModelID == "" {
		req.DisplayModelID = OpenAICompatImageTargetModel
	}
	if req.Operation == "edits" && len(req.Images) == 0 {
		return nil, fmt.Errorf("images is required")
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	if len(req.Images) > openAIImageUploadMaxCount {
		return nil, fmt.Errorf("images accepts at most %d items", openAIImageUploadMaxCount)
	}
	return req, nil
}

func normalizeOpenAIImageRequestMultipart(body []byte, boundary string, operation string) (*NormalizedImageRequest, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	req := &NormalizedImageRequest{Operation: strings.TrimSpace(operation)}
	var (
		rawImageSize   string
		rawAspectRatio string
	)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read multipart image request: %w", err)
		}
		name := strings.TrimSpace(part.FormName())
		if name == "" {
			_ = part.Close()
			continue
		}
		raw, readErr := io.ReadAll(part)
		_ = part.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read multipart field %q: %w", name, readErr)
		}
		if part.FileName() != "" {
			dataURL, dataErr := encodeOpenAIImageDataURL(raw)
			if dataErr != nil {
				return nil, dataErr
			}
			switch name {
			case "mask":
				req.Mask = dataURL
			case "image", "image[]", "images", "images[]", "reference_image":
				req.Images = append(req.Images, dataURL)
			default:
				return nil, fmt.Errorf("unsupported multipart file field %q", name)
			}
			continue
		}
		value := strings.TrimSpace(string(raw))
		switch name {
		case "model":
			req.DisplayModelID = value
		case "prompt":
			req.Prompt = value
		case "size":
			req.Size = value
		case "image_size":
			rawImageSize = value
		case "aspect_ratio":
			rawAspectRatio = value
		case "quality":
			req.Quality = value
		case "background":
			req.Background = value
		case "output_format":
			req.OutputFormat = value
		case "moderation":
			req.Moderation = value
		case "input_fidelity":
			req.InputFidelity = value
		case "output_compression":
			req.OutputCompression = optionalIntFromAny(value)
		case "partial_images":
			req.PartialImages = optionalIntFromAny(value)
		case "n":
			req.N = optionalIntFromAny(value)
		case "stream":
			req.Stream = parseExtraBool(value)
		case "mask_image_url":
			req.Mask = firstJSONImageSource(value)
		case "image_url", "images_url", "reference_image_url", "image", "image[]", "images", "images[]":
			if source := firstJSONImageSource(value); source != "" {
				req.Images = append(req.Images, source)
			}
		}
	}
	if normalizedSize, _, errCode, errMessage := normalizeOpenAIImageSizeWithAspect(req.Size, rawImageSize, rawAspectRatio); errCode != "" {
		return nil, newOpenAIImageRequestError(errCode, errMessage)
	} else if strings.TrimSpace(normalizedSize) != "" {
		req.Size = normalizedSize
	}
	if req.DisplayModelID == "" {
		req.DisplayModelID = OpenAICompatImageTargetModel
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	if req.Operation == "edits" && len(req.Images) == 0 {
		return nil, fmt.Errorf("images is required")
	}
	if len(req.Images) > openAIImageUploadMaxCount {
		return nil, fmt.Errorf("images accepts at most %d items", openAIImageUploadMaxCount)
	}
	return req, nil
}

func BuildOpenAIImageCompatResponsesBody(req *NormalizedImageRequest, topLevelModel string, targetModel string) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("normalized image request is required")
	}
	if targetModel == "" {
		targetModel = OpenAICompatImageTargetModel
	}
	if topLevelModel == "" {
		topLevelModel = targetModel
	}
	tool := map[string]any{
		"type":   "image_generation",
		"model":  targetModel,
		"action": compatImageToolAction(req.Operation),
	}
	setToolString(tool, "size", req.Size)
	setToolString(tool, "quality", req.Quality)
	setToolString(tool, "background", req.Background)
	setToolString(tool, "output_format", req.OutputFormat)
	setToolString(tool, "moderation", req.Moderation)
	if req.OutputCompression != nil {
		tool["output_compression"] = *req.OutputCompression
	}
	if req.PartialImages != nil {
		tool["partial_images"] = *req.PartialImages
	}
	if req.N != nil {
		n := *req.N
		if n < 1 {
			return nil, newOpenAIImageRequestError("image_n_invalid", "n must be at least 1")
		}
		if n > 1 {
			return nil, newOpenAIImageRequestError("image_n_not_supported", "compat image generation does not support n>1; call the endpoint multiple times instead")
		}
	}
	if req.Mask != "" {
		tool["input_image_mask"] = req.Mask
	}
	body := map[string]any{
		"model":       topLevelModel,
		"input":       buildResponsesCompatInputMessage(req.Prompt, req.Images),
		"tools":       []any{tool},
		"tool_choice": map[string]any{"type": "image_generation"},
	}
	if req.Stream {
		body["stream"] = true
	}
	return json.Marshal(body)
}

func collectJSONImageSources(raw any) []string {
	switch typed := raw.(type) {
	case nil:
		return nil
	case string:
		if source := firstJSONImageSource(typed); source != "" {
			return []string{source}
		}
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if source := firstJSONImageSource(item); source != "" {
				result = append(result, source)
			}
		}
		return result
	case map[string]any:
		if source := firstJSONImageSource(typed); source != "" {
			return []string{source}
		}
	}
	return nil
}

func firstJSONImageSource(raw any) string {
	switch typed := raw.(type) {
	case nil:
		return ""
	case string:
		value, _ := validateResponsesCompatReferenceImageURL(typed)
		return value
	case map[string]any:
		value, _ := validateResponsesCompatReferenceImageURL(stringValueFromAny(typed["image_url"]))
		return value
	default:
		return ""
	}
}

func compatImageToolAction(operation string) string {
	if strings.TrimSpace(strings.ToLower(operation)) == "edits" {
		return "edit"
	}
	return "generate"
}

func optionalIntFromAny(value any) *int {
	switch typed := value.(type) {
	case nil:
		return nil
	case int:
		out := typed
		return &out
	case float64:
		out := int(typed)
		return &out
	case string:
		if trimmed := strings.TrimSpace(typed); trimmed != "" {
			out := ParseExtraInt(trimmed)
			return &out
		}
	}
	return nil
}

func encodeOpenAIImageDataURL(raw []byte) (string, error) {
	contentType := strings.ToLower(strings.TrimSpace(http.DetectContentType(raw)))
	switch contentType {
	case "image/png", "image/jpeg", "image/webp":
	default:
		return "", fmt.Errorf("uploaded image only supports PNG, JPEG, or WebP")
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func setToolString(target map[string]any, key string, value string) {
	if target == nil {
		return
	}
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		target[key] = trimmed
	}
}
