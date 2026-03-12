package modelregistry

func DefaultAntigravityModelMapping() map[string]string {
	return cloneStringMap(defaultAntigravityModelMapping)
}

func ModelCatalogExplicitAliases() map[string]string {
	return cloneStringMap(modelCatalogExplicitAliases)
}

func ModelCatalogCanonicalDefaults() map[string]string {
	return cloneStringMap(modelCatalogCanonicalDefaults)
}

func cloneStringMap(input map[string]string) map[string]string {
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

var defaultAntigravityModelMapping = map[string]string{
	"claude-opus-4-6-thinking":       "claude-opus-4-6-thinking",
	"claude-opus-4-6":                "claude-opus-4-6-thinking",
	"claude-opus-4-5-thinking":       "claude-opus-4-6-thinking",
	"claude-sonnet-4-6":              "claude-sonnet-4-6",
	"claude-sonnet-4-5":              "claude-sonnet-4-5",
	"claude-sonnet-4-5-thinking":     "claude-sonnet-4-5-thinking",
	"claude-opus-4-5-20251101":       "claude-opus-4-6-thinking",
	"claude-sonnet-4-5-20250929":     "claude-sonnet-4-5",
	"claude-haiku-4-5":               "claude-sonnet-4-5",
	"claude-haiku-4-5-20251001":      "claude-sonnet-4-5",
	"gemini-2.5-flash":               "gemini-2.5-flash",
	"gemini-2.5-flash-image":         "gemini-2.5-flash-image",
	"gemini-2.5-flash-image-preview": "gemini-2.5-flash-image",
	"gemini-2.5-flash-lite":          "gemini-2.5-flash-lite",
	"gemini-2.5-flash-thinking":      "gemini-2.5-flash-thinking",
	"gemini-2.5-pro":                 "gemini-2.5-pro",
	"gemini-3-flash":                 "gemini-3-flash",
	"gemini-3-pro-high":              "gemini-3-pro-high",
	"gemini-3-pro-low":               "gemini-3-pro-low",
	"gemini-3-flash-preview":         "gemini-3-flash",
	"gemini-3-pro-preview":           "gemini-3-pro-high",
	"gemini-3.1-pro-high":            "gemini-3.1-pro-high",
	"gemini-3.1-pro-low":             "gemini-3.1-pro-low",
	"gemini-3.1-pro-preview":         "gemini-3.1-pro-high",
	"gemini-3.1-flash-image":         "gemini-3.1-flash-image",
	"gemini-3.1-flash-image-preview": "gemini-3.1-flash-image",
	"gemini-3-pro-image":             "gemini-3.1-flash-image",
	"gemini-3-pro-image-preview":     "gemini-3.1-flash-image",
	"gpt-oss-120b-medium":            "gpt-oss-120b-medium",
	"tab_flash_lite_preview":         "tab_flash_lite_preview",
}

var modelCatalogExplicitAliases = map[string]string{
	"claude-opus-4-1":                "claude-opus-4.1",
	"claude-opus-4-1-20250805":       "claude-opus-4.1",
	"claude-opus-4-5":                "claude-opus-4.1",
	"claude-opus-4-5-20251101":       "claude-opus-4.1",
	"claude-opus-4.5-20251101":       "claude-opus-4.1",
	"claude-opus-4-5-thinking":       "claude-opus-4.1",
	"claude-opus-4.5-thinking":       "claude-opus-4.1",
	"claude-opus-4-6":                "claude-opus-4.1",
	"claude-opus-4-6-thinking":       "claude-opus-4.1",
	"claude-sonnet-4-5":              "claude-sonnet-4.5",
	"claude-sonnet-4-5-20250929":     "claude-sonnet-4.5",
	"claude-sonnet-4.5-20250929":     "claude-sonnet-4.5",
	"claude-sonnet-4-5-thinking":     "claude-sonnet-4.5",
	"claude-sonnet-4.5-thinking":     "claude-sonnet-4.5",
	"claude-sonnet-4-6":              "claude-sonnet-4.5",
	"claude-sonnet-4-6-thinking":     "claude-sonnet-4.5",
	"claude-haiku-4-5":               "claude-haiku-4.5",
	"claude-haiku-4-5-20251001":      "claude-haiku-4.5",
	"claude-haiku-4.5-20251001":      "claude-haiku-4.5",
	"gpt-5-4":                        "gpt-5.4",
	"gpt-5.4":                        "gpt-5.4",
	"gpt-5-4-2026-03-05":             "gpt-5.4",
	"gpt-5.4-2026-03-05":             "gpt-5.4",
	"gpt-5-4-chat-latest":            "gpt-5.4",
	"gpt-5.4-chat-latest":            "gpt-5.4",
	"gpt-5-4-pro":                    "gpt-5.4-pro",
	"gpt-5.4-pro":                    "gpt-5.4-pro",
	"gpt-5-3-codex":                  "gpt-5-codex",
	"gpt-5.3-codex":                  "gpt-5-codex",
	"gpt-5-2-codex":                  "gpt-5-codex",
	"gpt-5.2-codex":                  "gpt-5-codex",
	"gpt-5-1-codex":                  "gpt-5-codex",
	"gpt-5.1-codex":                  "gpt-5-codex",
	"gemini-2.5-flash-image-preview": "gemini-2.5-flash-image",
	"gemini-2.5-flash-thinking":      "gemini-2.5-flash",
	"gemini-3-flash-preview":         "gemini-3-flash",
	"gemini-3-pro-preview":           "gemini-3-pro",
	"gemini-3-pro-high":              "gemini-3-pro",
	"gemini-3-pro-low":               "gemini-3-pro",
	"gemini-3-pro-image-preview":     "gemini-3-pro-image",
	"gemini-3.1-pro-preview":         "gemini-3.1-pro",
	"gemini-3.1-pro-high":            "gemini-3.1-pro",
	"gemini-3.1-pro-low":             "gemini-3.1-pro",
	"gemini-3.1-flash-lite-preview":  "gemini-3.1-flash-lite",
	"gemini-3.1-flash-image-preview": "gemini-3.1-flash-image",
}

var modelCatalogCanonicalDefaults = map[string]string{
	"claude-opus-4.1":        "claude-opus-4-1-20250805",
	"claude-sonnet-4.5":      "claude-sonnet-4-5-20250929",
	"claude-haiku-4.5":       "claude-haiku-4-5-20251001",
	"gpt-5.4":                "gpt-5.4",
	"gpt-5.4-pro":            "gpt-5.4-pro",
	"gpt-5-mini":             "gpt-5-mini",
	"gpt-5-nano":             "gpt-5-nano",
	"gpt-5-codex":            "gpt-5-codex",
	"gemini-2.5-flash-image": "gemini-2.5-flash-image",
	"gemini-3-pro":           "gemini-3-pro-preview",
	"gemini-3-flash":         "gemini-3-flash-preview",
	"gemini-3-pro-image":     "gemini-3-pro-image-preview",
	"gemini-3.1-pro":         "gemini-3.1-pro-preview",
	"gemini-3.1-flash-lite":  "gemini-3.1-flash-lite-preview",
	"gemini-3.1-flash-image": "gemini-3.1-flash-image-preview",
}
