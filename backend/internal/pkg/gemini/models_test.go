package gemini

import "testing"

func TestDefaultModels_ContainsImageModels(t *testing.T) {
	t.Parallel()

	models := DefaultModels()
	byName := make(map[string]Model, len(models))
	for _, model := range models {
		byName[model.Name] = model
	}

	required := []string{
		"models/gemini-2.5-flash-image",
		"models/gemini-3.1-pro-preview-customtools",
		"models/gemini-3.1-flash-image-preview",
		"models/gemini-3-pro-image-preview",
	}

	for _, name := range required {
		model, ok := byName[name]
		if !ok {
			t.Fatalf("expected fallback model %q to exist", name)
		}
		if len(model.SupportedGenerationMethods) == 0 {
			t.Fatalf("expected fallback model %q to advertise generation methods", name)
		}
	}
}

func TestDefaultModels_ContainsGemini3PreviewFamilies(t *testing.T) {
	t.Parallel()

	byName := make(map[string]Model, len(DefaultModels()))
	for _, model := range DefaultModels() {
		byName[model.Name] = model
	}

	required := []string{
		"models/gemini-3-flash-preview",
		"models/gemini-3.1-pro-preview",
		"models/gemini-3.1-flash-lite-preview",
	}

	for _, name := range required {
		if _, ok := byName[name]; !ok {
			t.Fatalf("expected Gemini 3 fallback model %q to exist", name)
		}
	}
}

func TestDefaultModels_AdvertiseCountTokens(t *testing.T) {
	t.Parallel()

	for _, model := range DefaultModels() {
		found := false
		for _, method := range model.SupportedGenerationMethods {
			if method == "countTokens" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected fallback model %q to advertise countTokens", model.Name)
		}
	}
}
