package geminicli

import geminipkg "github.com/Wei-Shaw/sub2api/internal/pkg/gemini"

// Model represents a selectable Gemini model for UI/testing purposes.
// Keep JSON fields consistent with existing frontend expectations.
type Model struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

// DefaultModels is the curated Gemini model list used by the admin UI "test account" flow.
var DefaultModels = defaultModels()

func defaultModels() []Model {
	fallbackModels := geminipkg.DefaultModels()
	models := make([]Model, 0, len(fallbackModels))
	for _, model := range fallbackModels {
		id := model.Name
		if len(id) >= len("models/") && id[:len("models/")] == "models/" {
			id = id[len("models/"):]
		}
		models = append(models, Model{
			ID:          id,
			Type:        "model",
			DisplayName: model.DisplayName,
			CreatedAt:   "",
		})
	}
	return models
}

// DefaultTestModel is the default model to preselect in test flows.
const DefaultTestModel = "gemini-2.5-flash"
