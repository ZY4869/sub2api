// Package gemini provides minimal fallback model metadata for Gemini native endpoints.
// It is used when upstream model listing is unavailable (e.g. OAuth token missing AI Studio scopes).
package gemini

import "github.com/Wei-Shaw/sub2api/internal/modelregistry"

type Model struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"displayName,omitempty"`
	Description                string   `json:"description,omitempty"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods,omitempty"`
}

type ModelsListResponse struct {
	Models []Model `json:"models"`
}

func DefaultModels() []Model {
	methods := []string{"generateContent", "streamGenerateContent"}
	entries := modelregistry.ModelsByPlatform(modelregistry.SeedModels(), "gemini", "runtime", "whitelist", "use_key")
	models := make([]Model, 0, len(entries))
	for _, entry := range entries {
		name := entry.ID
		if name == "" {
			continue
		}
		if len(name) < 7 || name[:7] != "models/" {
			name = "models/" + name
		}
		models = append(models, Model{Name: name, DisplayName: entry.DisplayName, SupportedGenerationMethods: methods})
	}
	return models
}

func FallbackModelsList() ModelsListResponse {
	return ModelsListResponse{Models: DefaultModels()}
}

func FallbackModel(model string) Model {
	methods := []string{"generateContent", "streamGenerateContent"}
	if model == "" {
		return Model{Name: "models/unknown", SupportedGenerationMethods: methods}
	}
	if len(model) >= 7 && model[:7] == "models/" {
		return Model{Name: model, SupportedGenerationMethods: methods}
	}
	return Model{Name: "models/" + model, SupportedGenerationMethods: methods}
}
