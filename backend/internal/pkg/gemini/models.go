// Package gemini provides fallback model metadata for Gemini native endpoints.
// It is used when upstream model listing is unavailable or when the local
// gateway only has registry/public-entry information.
package gemini

import "strings"

type Model struct {
	Name                       string   `json:"name"`
	BaseModelID                string   `json:"baseModelId,omitempty"`
	Version                    string   `json:"version,omitempty"`
	DisplayName                string   `json:"displayName,omitempty"`
	Description                string   `json:"description,omitempty"`
	InputTokenLimit            int      `json:"inputTokenLimit,omitempty"`
	OutputTokenLimit           int      `json:"outputTokenLimit,omitempty"`
	Temperature                float64  `json:"temperature,omitempty"`
	MaxTemperature             float64  `json:"maxTemperature,omitempty"`
	TopP                       float64  `json:"topP,omitempty"`
	TopK                       int      `json:"topK,omitempty"`
	Thinking                   bool     `json:"thinking,omitempty"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods,omitempty"`
}

type ModelsListResponse struct {
	Models        []Model `json:"models"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
}

var defaultFallbackModels = []Model{
	{Name: "models/gemini-2.0-flash", DisplayName: "Gemini 2.0 Flash"},
	{Name: "models/gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash"},
	{Name: "models/gemini-2.5-flash-image", DisplayName: "Gemini 2.5 Flash Image"},
	{Name: "models/gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro"},
	{Name: "models/gemini-3-flash-preview", DisplayName: "Gemini 3 Flash Preview"},
	{Name: "models/gemini-3.1-pro-preview", DisplayName: "Gemini 3.1 Pro Preview"},
	{Name: "models/gemini-3.1-pro-preview-customtools", DisplayName: "Gemini 3.1 Pro Preview Customtools"},
	{Name: "models/gemini-3.1-flash-lite-preview", DisplayName: "Gemini 3.1 Flash Lite Preview"},
	{Name: "models/gemini-3.1-flash-image-preview", DisplayName: "Gemini 3.1 Flash Image Preview"},
	{Name: "models/gemini-3-pro-image-preview", DisplayName: "Gemini 3 Pro Image Preview"},
}

func DefaultModels() []Model {
	models := make([]Model, 0, len(defaultFallbackModels))
	for _, model := range defaultFallbackModels {
		models = append(models, BuildModel(model.Name, model.DisplayName, model.Description, nil))
	}
	return models
}

func FallbackModelsList() ModelsListResponse {
	return ModelsListResponse{Models: DefaultModels()}
}

func FallbackModel(model string) Model {
	return BuildModel(model, "", "", nil)
}

func ProjectMinimalModel(model string, displayName string, description string, methods []string) Model {
	name := normalizeModelName(model)
	if strings.TrimSpace(displayName) == "" {
		displayName = strings.TrimPrefix(name, "models/")
	}
	projected := Model{
		Name:        name,
		DisplayName: displayName,
	}
	if strings.TrimSpace(description) != "" {
		projected.Description = description
	}
	if len(methods) > 0 {
		projected.SupportedGenerationMethods = append([]string(nil), methods...)
	}
	return projected
}

func BuildModel(model string, displayName string, description string, methods []string) Model {
	name := normalizeModelName(model)
	modelID := strings.TrimPrefix(name, "models/")
	baseModelID, version := splitBaseModelVersion(modelID)
	if strings.TrimSpace(displayName) == "" {
		displayName = fallbackDisplayName(modelID)
	}
	if strings.TrimSpace(description) == "" && strings.TrimSpace(displayName) != "" {
		description = displayName + " model metadata generated from local fallback registry."
	}
	if len(methods) == 0 {
		methods = SupportedGenerationMethodsForModel(modelID)
	}
	inputTokenLimit, outputTokenLimit := fallbackTokenLimits(modelID)
	return Model{
		Name:                       name,
		BaseModelID:                baseModelID,
		Version:                    version,
		DisplayName:                displayName,
		Description:                description,
		InputTokenLimit:            inputTokenLimit,
		OutputTokenLimit:           outputTokenLimit,
		Temperature:                1,
		MaxTemperature:             2,
		TopP:                       0.95,
		TopK:                       64,
		Thinking:                   supportsThinking(modelID),
		SupportedGenerationMethods: append([]string(nil), methods...),
	}
}

func normalizeModelName(model string) string {
	trimmed := strings.TrimSpace(model)
	if trimmed == "" {
		return "models/unknown"
	}
	if strings.HasPrefix(trimmed, "models/") {
		return trimmed
	}
	return "models/" + trimmed
}

func splitBaseModelVersion(modelID string) (string, string) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return "", ""
	}
	if idx := strings.Index(modelID, "-preview"); idx > 0 {
		return modelID[:idx], strings.TrimPrefix(modelID[idx:], "-")
	}
	if idx := strings.Index(modelID, "-exp"); idx > 0 {
		return modelID[:idx], strings.TrimPrefix(modelID[idx:], "-")
	}
	lastDash := strings.LastIndex(modelID, "-")
	if lastDash > 0 {
		suffix := modelID[lastDash+1:]
		if isNumericVersionSuffix(suffix) || isDateVersionSuffix(suffix) {
			return modelID[:lastDash], suffix
		}
	}
	return modelID, ""
}

func isNumericVersionSuffix(value string) bool {
	if len(value) == 0 {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func isDateVersionSuffix(value string) bool {
	if len(value) != len("2026-04-14") {
		return false
	}
	for i, ch := range value {
		if i == 4 || i == 7 {
			if ch != '-' {
				return false
			}
			continue
		}
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func fallbackDisplayName(modelID string) string {
	if trimmed := strings.TrimSpace(modelID); trimmed != "" {
		return trimmed
	}
	return "Unknown Gemini Model"
}

func fallbackTokenLimits(modelID string) (int, int) {
	normalized := strings.ToLower(strings.TrimSpace(modelID))
	switch {
	case strings.Contains(normalized, "image"), strings.Contains(normalized, "video"), strings.Contains(normalized, "tts"):
		return 32768, 8192
	case strings.Contains(normalized, "lite"):
		return 1048576, 32768
	default:
		return 1048576, 65536
	}
}

func supportsThinking(modelID string) bool {
	normalized := strings.ToLower(strings.TrimSpace(modelID))
	switch {
	case normalized == "",
		strings.Contains(normalized, "image"),
		strings.Contains(normalized, "video"),
		strings.Contains(normalized, "tts"),
		strings.Contains(normalized, "embedding"):
		return false
	default:
		return strings.Contains(normalized, "gemini")
	}
}
