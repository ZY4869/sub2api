package service

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
)

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

func hasExplicitResponsesCompatTools(reqBody map[string]any) bool {
	if reqBody == nil {
		return false
	}
	raw, exists := reqBody["tools"]
	return exists && raw != nil
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
