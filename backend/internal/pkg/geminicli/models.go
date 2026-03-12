package geminicli

import "github.com/Wei-Shaw/sub2api/internal/modelregistry"

// Model represents a selectable Gemini model for UI/testing purposes.
// Keep JSON fields consistent with existing frontend expectations.
type Model struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

// DefaultModels is the curated Gemini model list used by the admin UI "test account" flow.
var DefaultModels = defaultModelsFromSeed()

func defaultModelsFromSeed() []Model {
	entries := modelregistry.ModelsByPlatform(modelregistry.SeedModels(), "gemini", "runtime", "whitelist", "use_key")
	models := make([]Model, 0, len(entries))
	for _, entry := range entries {
		displayName := entry.DisplayName
		if displayName == "" {
			displayName = entry.ID
		}
		models = append(models, Model{ID: entry.ID, Type: "model", DisplayName: displayName, CreatedAt: ""})
	}
	return models
}

// DefaultTestModel is the default model to preselect in test flows.
const DefaultTestModel = "gemini-2.0-flash"
