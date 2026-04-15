package gemini

import "strings"

type SupportedGenerationMethodsOptions struct {
	Modalities   []string
	Capabilities []string
}

func SupportedGenerationMethodsForModel(model string) []string {
	return SupportedGenerationMethodsForModelWithOptions(model, SupportedGenerationMethodsOptions{})
}

func SupportedGenerationMethodsForModelWithOptions(model string, opts SupportedGenerationMethodsOptions) []string {
	normalized := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(model, "models/")))
	if normalized == "" {
		return nil
	}

	if isGeminiEmbeddingModel(normalized, opts) {
		return []string{"embedContent", "countTokens"}
	}
	if isGeminiMediaGenerationModel(normalized, opts) {
		return []string{"generateContent", "countTokens"}
	}
	return []string{"generateContent", "streamGenerateContent", "countTokens", "batchGenerateContent"}
}

func isGeminiEmbeddingModel(model string, opts SupportedGenerationMethodsOptions) bool {
	if strings.Contains(model, "embedding") {
		return true
	}
	return containsGeminiMethodHint(opts.Capabilities, "embedding") ||
		containsGeminiMethodHint(opts.Modalities, "embedding")
}

func isGeminiMediaGenerationModel(model string, opts SupportedGenerationMethodsOptions) bool {
	for _, token := range []string{"image", "video", "tts"} {
		if strings.Contains(model, token) ||
			containsGeminiMethodHint(opts.Capabilities, token) ||
			containsGeminiMethodHint(opts.Modalities, token) {
			return true
		}
	}
	return false
}

func containsGeminiMethodHint(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return false
	}
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}
