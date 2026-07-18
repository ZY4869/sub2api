package securityaudit

import (
	"sort"
	"strings"
)

func extractMediaPrompts(root map[string]any) []string {
	result, seen := []string{}, map[string]struct{}{}
	var walk func(any, string)
	walk = func(value any, key string) {
		switch typed := value.(type) {
		case map[string]any:
			walkMediaObject(typed, walk)
		case []any:
			for _, item := range typed {
				walk(item, key)
			}
		case string:
			maybeAppendMediaPrompt(&result, seen, key, typed)
		}
	}
	walk(root, "")
	return result
}

func walkMediaObject(object map[string]any, walk func(any, string)) {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		walk(object[key], key)
	}
}

func maybeAppendMediaPrompt(result *[]string, seen map[string]struct{}, key string, value string) {
	text := strings.TrimSpace(value)
	if !isMediaPromptKey(key) || looksLikeMediaPayload(text) || text == "" {
		return
	}
	if _, ok := seen[text]; ok {
		return
	}
	seen[text] = struct{}{}
	*result = append(*result, text)
}

func isMediaPromptKey(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	switch normalized {
	case "prompt", "inputprompt", "textprompt", "description", "query", "lyrics", "negativeprompt", "positiveprompt", "input":
		return true
	default:
		return false
	}
}

func looksLikeMediaPayload(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(lower, "data:image/") || strings.HasPrefix(lower, "data:video/") ||
		strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return true
	}
	return looksLikeBase64MediaBlob(value)
}

func looksLikeBase64MediaBlob(value string) bool {
	if len(value) < 256 {
		return false
	}
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' {
			continue
		}
		return false
	}
	return true
}
