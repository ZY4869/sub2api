package securityaudit

import "strings"

func extractGeminiRoot(root map[string]any) []promptSegment {
	if root == nil {
		return nil
	}
	out := extractGeminiSystemInstruction(firstPresent(root, "systemInstruction", "system_instruction"))
	out = append(out, extractGeminiContents(root["contents"])...)
	out = append(out, extractGeminiContents(root["content"])...)
	out = append(out, extractGeminiInstances(root["instances"])...)
	return out
}

func extractGeminiSystemInstruction(value any) []promptSegment {
	switch typed := value.(type) {
	case string:
		return roleSegments([]string{typed}, "system")
	case map[string]any:
		return roleSegments(contentTexts(typed["parts"]), "system")
	case []any:
		segments := extractGeminiContents(typed)
		for i := range segments {
			segments[i].user = false
		}
		return segments
	default:
		return nil
	}
}

func extractGeminiContents(value any) []promptSegment {
	var contents []any
	switch typed := value.(type) {
	case []any:
		contents = typed
	case map[string]any:
		contents = []any{typed}
	default:
		return nil
	}
	return geminiContentSegments(contents)
}

func geminiContentSegments(contents []any) []promptSegment {
	out := make([]promptSegment, 0, len(contents))
	for _, item := range contents {
		content, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role := strings.ToLower(stringValue(content["role"]))
		for _, text := range contentTexts(content["parts"]) {
			out = append(out, promptSegment{text: text, user: role == "" || role == "user"})
		}
	}
	return out
}

func extractGeminiInstances(value any) []promptSegment {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]promptSegment, 0, len(items))
	for _, item := range items {
		instance, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if prompt := stringValue(instance["prompt"]); prompt != "" {
			out = append(out, promptSegment{text: prompt, user: true})
		}
	}
	return out
}
