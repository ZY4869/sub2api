// Package openai provides helpers and types for OpenAI API integration.
package openai

import (
	_ "embed"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

// Model represents an OpenAI model
type Model struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Created     int64  `json:"created"`
	OwnedBy     string `json:"owned_by"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
}

// DefaultModels OpenAI models list.
var DefaultModels = defaultModelsFromSeed()

func defaultModelsFromSeed() []Model {
	entries := modelregistry.ModelsByPlatform(modelregistry.SeedModels(), "openai", "runtime", "whitelist", "use_key")
	models := make([]Model, 0, len(entries))
	for _, entry := range entries {
		displayName := entry.DisplayName
		if displayName == "" {
			displayName = entry.ID
		}
		models = append(models, Model{ID: entry.ID, Object: "model", OwnedBy: "openai", Type: "model", DisplayName: displayName})
	}
	return models
}

// DefaultModelIDs returns the default model ID list
func DefaultModelIDs() []string {
	ids := make([]string, len(DefaultModels))
	for i, m := range DefaultModels {
		ids[i] = m.ID
	}
	return ids
}

// DefaultTestModel default model for testing OpenAI accounts
const DefaultTestModel = "gpt-5.1-codex"

// DefaultInstructions default instructions for non-Codex CLI requests
// Content loaded from instructions.txt at compile time
//
//go:embed instructions.txt
var DefaultInstructions string
