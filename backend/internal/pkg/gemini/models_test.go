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
		if model.InputTokenLimit <= 0 || model.OutputTokenLimit <= 0 {
			t.Fatalf("expected fallback model %q to include token limits", name)
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

func TestBuildModel_SplitsPreviewVersion(t *testing.T) {
	t.Parallel()

	model := BuildModel("gemini-3.1-pro-preview-customtools", "Gemini 3.1 Pro Preview Customtools", "", nil)

	if model.BaseModelID != "gemini-3.1-pro" {
		t.Fatalf("expected baseModelId gemini-3.1-pro, got %q", model.BaseModelID)
	}
	if model.Version != "preview-customtools" {
		t.Fatalf("expected version preview-customtools, got %q", model.Version)
	}
	if !model.Thinking {
		t.Fatalf("expected preview text model to advertise thinking support")
	}
}

func TestSupportedGenerationMethodsForModel_IsDynamic(t *testing.T) {
	t.Parallel()

	imageMethods := SupportedGenerationMethodsForModel("gemini-2.5-flash-image")
	if len(imageMethods) != 2 || imageMethods[0] != "generateContent" || imageMethods[1] != "countTokens" {
		t.Fatalf("unexpected image methods: %#v", imageMethods)
	}

	embeddingMethods := SupportedGenerationMethodsForModel("gemini-embedding-001")
	if len(embeddingMethods) != 2 || embeddingMethods[0] != "embedContent" || embeddingMethods[1] != "countTokens" {
		t.Fatalf("unexpected embedding methods: %#v", embeddingMethods)
	}
}
