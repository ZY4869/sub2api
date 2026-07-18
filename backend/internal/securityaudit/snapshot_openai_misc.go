package securityaudit

func extractOpenAICompletions(root map[string]any) []promptSegment {
	return userSegments(promptTexts(firstPresent(root, "prompt", "suffix")))
}

func extractOpenAIEmbeddings(root map[string]any) []promptSegment {
	return userSegments(promptTexts(firstPresent(root, "input", "inputs")))
}

func extractOpenAIAlphaSearch(root map[string]any) []promptSegment {
	return userSegments(promptTexts(firstPresent(root, "query", "q", "search_query", "input")))
}

func promptTexts(value any) []string {
	switch typed := value.(type) {
	case string:
		if looksLikeMediaPayload(typed) {
			return nil
		}
		return []string{typed}
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, promptTexts(item)...)
		}
		return out
	case map[string]any:
		return promptTexts(firstPresent(typed, "text", "content", "input", "query"))
	default:
		return nil
	}
}
