package apicompat

import (
	"encoding/json"
	"strings"
)

func responsesOutputFromChatToolCall(call ChatToolCall, proxies map[string]ResponsesToolProxy) ResponsesOutput {
	proxy, ok := proxies[strings.TrimSpace(call.Function.Name)]
	if !ok {
		return ResponsesOutput{
			Type:      "function_call",
			CallID:    strings.TrimSpace(call.ID),
			Name:      strings.TrimSpace(call.Function.Name),
			Arguments: firstNonEmptyCompat(call.Function.Arguments, "{}"),
		}
	}

	out := ResponsesOutput{
		Type:      proxy.OutputType(),
		CallID:    strings.TrimSpace(call.ID),
		Name:      proxy.OutputName(),
		Namespace: strings.TrimSpace(proxy.Namespace),
		Arguments: firstNonEmptyCompat(call.Function.Arguments, "{}"),
	}
	if proxy.Type == "tool_search" {
		out.Action = &WebSearchAction{Type: "search", Query: extractToolSearchQuery(out.Arguments)}
	}
	return out
}

func extractToolSearchQuery(arguments string) string {
	var obj map[string]any
	if err := json.Unmarshal([]byte(arguments), &obj); err != nil {
		return ""
	}
	for _, key := range []string{"query", "q", "search_query"} {
		if value, ok := obj[key].(string); ok {
			return value
		}
	}
	return ""
}
