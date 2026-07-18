package securityaudit

import "strings"

func extractMessages(value any, includeNonUser bool) []promptSegment {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]promptSegment, 0, len(items))
	for _, item := range items {
		message, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role := strings.ToLower(stringValue(message["role"]))
		if role != "user" && !includeNonUser {
			continue
		}
		for _, text := range contentTexts(message["content"]) {
			out = append(out, promptSegment{text: text, user: role == "" || role == "user"})
		}
	}
	return out
}

func extractSystem(value any) []promptSegment {
	return roleSegments(contentTexts(value), "system")
}

func contentTexts(value any) []string {
	switch typed := value.(type) {
	case string:
		return []string{typed}
	case []any:
		return partTexts(typed)
	case map[string]any:
		if text := stringValue(typed["text"]); text != "" {
			return []string{text}
		}
		return contentTexts(typed["parts"])
	default:
		return nil
	}
}

func partTexts(parts []any) []string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if text := partText(part); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func partText(value any) string {
	object, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	typeName := strings.ToLower(stringValue(object["type"]))
	if typeName != "" && typeName != "text" && typeName != "input_text" {
		return ""
	}
	return stringValue(object["text"])
}

func roleSegments(texts []string, role string) []promptSegment {
	out := make([]promptSegment, 0, len(texts))
	for _, text := range texts {
		out = append(out, promptSegment{text: text, user: role == "" || role == "user"})
	}
	return out
}

func userSegments(texts []string) []promptSegment { return roleSegments(texts, "user") }
