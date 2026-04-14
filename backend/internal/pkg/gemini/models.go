// Package gemini provides minimal fallback model metadata for Gemini native endpoints.
// It is used when upstream model listing is unavailable (e.g. OAuth token missing AI Studio scopes).
package gemini

type Model struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"displayName,omitempty"`
	Description                string   `json:"description,omitempty"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods,omitempty"`
}

type ModelsListResponse struct {
	Models []Model `json:"models"`
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

func defaultSupportedGenerationMethods() []string {
	return []string{"generateContent", "streamGenerateContent", "countTokens"}
}

func DefaultModels() []Model {
	methods := defaultSupportedGenerationMethods()
	models := make([]Model, 0, len(defaultFallbackModels))
	for _, model := range defaultFallbackModels {
		clone := model
		clone.SupportedGenerationMethods = append([]string(nil), methods...)
		models = append(models, clone)
	}
	return models
}

func FallbackModelsList() ModelsListResponse {
	return ModelsListResponse{Models: DefaultModels()}
}

func FallbackModel(model string) Model {
	methods := defaultSupportedGenerationMethods()
	if model == "" {
		return Model{Name: "models/unknown", SupportedGenerationMethods: methods}
	}
	if len(model) >= 7 && model[:7] == "models/" {
		return Model{Name: model, SupportedGenerationMethods: methods}
	}
	return Model{Name: "models/" + model, SupportedGenerationMethods: methods}
}
