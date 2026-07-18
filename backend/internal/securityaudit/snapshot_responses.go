package securityaudit

import "strings"

func extractResponses(root map[string]any) []promptSegment {
	if root == nil {
		return nil
	}
	if frameType := stringValue(root["type"]); frameType != "" && frameType != "response.create" {
		return nil
	}
	if response, ok := root["response"].(map[string]any); ok {
		return extractResponsesValue(response["input"])
	}
	return extractResponsesValue(root["input"])
}

func extractResponsesValue(value any) []promptSegment {
	switch typed := value.(type) {
	case string:
		return []promptSegment{{text: typed, user: true}}
	case []any:
		return extractResponsesItems(typed)
	case map[string]any:
		role := strings.ToLower(stringValue(typed["role"]))
		return roleSegments(contentTexts(firstPresent(typed, "content", "text")), role)
	default:
		return nil
	}
}

func extractResponsesItems(items []any) []promptSegment {
	out := make([]promptSegment, 0, len(items))
	for _, item := range items {
		switch entry := item.(type) {
		case string:
			out = append(out, promptSegment{text: entry, user: true})
		case map[string]any:
			role := strings.ToLower(stringValue(entry["role"]))
			for _, text := range contentTexts(firstPresent(entry, "content", "text")) {
				out = append(out, promptSegment{text: text, user: role == "" || role == "user"})
			}
		}
	}
	return out
}
