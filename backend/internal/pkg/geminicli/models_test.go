package geminicli

import "testing"

func TestDefaultModels_ContainsImageModels(t *testing.T) {
	t.Parallel()

	byID := make(map[string]Model, len(DefaultModels))
	for _, model := range DefaultModels {
		byID[model.ID] = model
	}

	required := []string{
		"gemini-2.5-flash-image",
		"gemini-3.1-flash-image-preview",
		"gemini-3-pro-image-preview",
	}

	for _, id := range required {
		if _, ok := byID[id]; !ok {
			t.Fatalf("expected curated Gemini model %q to exist", id)
		}
	}
}

func TestDefaultModels_ContainsGemini3PreviewFamilies(t *testing.T) {
	t.Parallel()

	byID := make(map[string]Model, len(DefaultModels))
	for _, model := range DefaultModels {
		byID[model.ID] = model
	}

	required := []string{
		"gemini-3-flash-preview",
		"gemini-3.1-pro-preview",
		"gemini-3.1-flash-lite-preview",
	}

	for _, id := range required {
		if _, ok := byID[id]; !ok {
			t.Fatalf("expected curated Gemini model %q to exist", id)
		}
	}
}

func TestDefaultTestModel_UsesGemini25Flash(t *testing.T) {
	t.Parallel()

	if DefaultTestModel != "gemini-2.5-flash" {
		t.Fatalf("expected Gemini CLI default test model to be gemini-2.5-flash, got %q", DefaultTestModel)
	}
}
