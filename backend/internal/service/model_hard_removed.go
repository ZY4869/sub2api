package service

import "strings"

var runtimeModelHardRemoveExceptions = map[string]struct{}{
	"deepseek-v4-flash": {},
	"deepseek-v4-pro":   {},
}

func isRuntimeHardRemoveException(modelID string) bool {
	normalized := normalizeRegistryID(modelID)
	if normalized == "" {
		return false
	}
	_, ok := runtimeModelHardRemoveExceptions[normalized]
	return ok
}

func filterRuntimeHardRemoveExceptions(modelIDs []string) []string {
	normalizedIDs := normalizeStringList(modelIDs, normalizeRegistryID)
	if len(normalizedIDs) == 0 {
		return []string{}
	}
	filtered := make([]string, 0, len(normalizedIDs))
	for _, modelID := range normalizedIDs {
		if isRuntimeHardRemoveException(modelID) {
			continue
		}
		filtered = append(filtered, modelID)
	}
	return filtered
}

var explicitHardRemovedInputModelIDs = map[string]struct{}{
	"claude-haiku-4-5":               {},
	"claude-haiku-4-5-20251001":      {},
	"claude-opus-4-5":                {},
	"claude-opus-4-5-20251101":       {},
	"claude-opus-4-5-thinking":       {},
	"claude-sonnet-4-5":              {},
	"claude-sonnet-4-5-20250929":     {},
	"claude-sonnet-4-5-thinking":     {},
	"deepseek-chat":                  {},
	"deepseek-reasoner":              {},
	"gemini-2.5-flash-image-preview": {},
	"gemini-3-pro-preview":           {},
	"grok-2":                         {},
	"grok-2-image":                   {},
	"grok-2-vision":                  {},
	"grok-3-beta":                    {},
	"grok-3-fast-beta":               {},
	"grok-3-mini-beta":               {},
	"grok-4":                         {},
	"grok-4-0709":                    {},
	"grok-beta":                      {},
	"grok-imagine-image":             {},
	"grok-imagine-video":             {},
	"grok-vision-beta":               {},
	"unknown":                        {},
}

func init() {
	for _, modelID := range pricingPatchHardRemovedModelIDs20260506 {
		explicitHardRemovedInputModelIDs[modelID] = struct{}{}
		explicitHardRemovedRegistryModelIDsSet[modelID] = struct{}{}
	}
}

func isHardRemovedModelID(modelID string) bool {
	normalized := normalizeRegistryID(modelID)
	if normalized == "" {
		return false
	}
	if isRuntimeHardRemoveException(normalized) {
		return false
	}
	if _, ok := explicitHardRemovedInputModelIDs[normalized]; ok {
		return true
	}
	if strings.HasPrefix(normalized, "gpt-5.1") {
		return true
	}
	if strings.HasPrefix(normalized, "gpt-5-codex") {
		return true
	}
	if strings.HasPrefix(normalized, "gpt-5.2-codex") {
		return true
	}
	if strings.HasPrefix(normalized, "gpt-5.3-codex") && !strings.HasPrefix(normalized, "gpt-5.3-codex-spark") {
		return true
	}
	switch normalized {
	case "codex-mini-latest":
		return true
	default:
		return false
	}
}

var explicitHardRemovedRegistryModelIDsSet = map[string]struct{}{
	"claude-haiku-4-5":               {},
	"claude-haiku-4-5-20251001":      {},
	"claude-opus-4-5":                {},
	"claude-opus-4-5-20251101":       {},
	"claude-opus-4-5-thinking":       {},
	"claude-sonnet-4-5":              {},
	"claude-sonnet-4-5-20250929":     {},
	"claude-sonnet-4-5-thinking":     {},
	"deepseek-chat":                  {},
	"deepseek-reasoner":              {},
	"gemini-2.5-flash-image-preview": {},
	"gemini-3-pro-preview":           {},
	"grok-2":                         {},
	"grok-2-image":                   {},
	"grok-2-vision":                  {},
	"grok-3-beta":                    {},
	"grok-3-fast-beta":               {},
	"grok-3-mini-beta":               {},
	"grok-4":                         {},
	"grok-4-0709":                    {},
	"grok-beta":                      {},
	"grok-imagine-image":             {},
	"grok-imagine-video":             {},
	"grok-vision-beta":               {},
	"unknown":                        {},
}

func explicitHardRemovedRegistryModelIDs() []string {
	items := make([]string, 0, len(explicitHardRemovedRegistryModelIDsSet))
	for modelID := range explicitHardRemovedRegistryModelIDsSet {
		items = append(items, modelID)
	}
	return filterRuntimeHardRemoveExceptions(items)
}
