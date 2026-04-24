package service

// IsOpenAINativeImageModelID returns true when the supplied model id refers to a
// native OpenAI image generation model (capability=image_generation).
//
// Tool-only image routing models (image_generation_tool) are intentionally NOT
// included here.
//
// This is a pure local check and must not trigger upstream probing.
func IsOpenAINativeImageModelID(modelID string) bool {
	return isOpenAIGPTImageProfileModelID(modelID)
}
