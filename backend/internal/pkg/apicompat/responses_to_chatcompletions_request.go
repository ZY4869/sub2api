package apicompat

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ResponsesToChatCompletionsRequest converts a Responses request into the
// closest Chat Completions request supported by the local native-chat bridge.
func ResponsesToChatCompletionsRequest(req *ResponsesRequest) (*ChatCompletionsRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("responses request is required")
	}
	messages, err := responsesInputToChatMessages(req.Input)
	if err != nil {
		return nil, err
	}
	out := &ChatCompletionsRequest{
		Model:           strings.TrimSpace(req.Model),
		Messages:        messages,
		Temperature:     req.Temperature,
		TopP:            req.TopP,
		Stream:          req.Stream,
		Tools:           responsesToolsToChatTools(req.Tools),
		ToolChoice:      req.ToolChoice,
		ServiceTier:     strings.TrimSpace(req.ServiceTier),
		ReasoningEffort: responsesReasoningEffort(req.Reasoning),
	}
	if req.MaxOutputTokens != nil {
		out.MaxCompletionTokens = req.MaxOutputTokens
	}
	if req.Stream {
		out.StreamOptions = &ChatStreamOptions{IncludeUsage: true}
	}
	return out, nil
}

func responsesInputToChatMessages(raw json.RawMessage) ([]ChatMessage, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("input is required")
	}
	var inputString string
	if err := json.Unmarshal(raw, &inputString); err == nil {
		if strings.TrimSpace(inputString) == "" {
			return nil, fmt.Errorf("input is required")
		}
		return []ChatMessage{{
			Role:    "user",
			Content: json.RawMessage(mustMarshalJSON(inputString)),
		}}, nil
	}
	var items []ResponsesInputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("input must be a string or an array")
	}
	messages := make([]ChatMessage, 0, len(items))
	pendingToolCalls := map[string]ChatToolCall{}
	for _, item := range items {
		switch strings.TrimSpace(item.Type) {
		case "":
			msg, ok := responsesMessageItemToChatMessage(item)
			if ok {
				messages = append(messages, msg)
			}
		case "message":
			msg, ok := responsesMessageItemToChatMessage(item)
			if ok {
				messages = append(messages, msg)
			}
		case "function_call":
			callID := strings.TrimSpace(item.CallID)
			if callID == "" {
				callID = strings.TrimSpace(item.ID)
			}
			call := ChatToolCall{
				ID:   callID,
				Type: "function",
				Function: ChatFunctionCall{
					Name:      strings.TrimSpace(item.Name),
					Arguments: firstNonEmptyJSONText(item.Arguments, "{}"),
				},
			}
			pendingToolCalls[callID] = call
			messages = append(messages, ChatMessage{
				Role:      "assistant",
				Content:   json.RawMessage(`null`),
				ToolCalls: []ChatToolCall{call},
			})
		case "function_call_output":
			callID := strings.TrimSpace(item.CallID)
			messages = append(messages, ChatMessage{
				Role:       "tool",
				Content:    json.RawMessage(mustMarshalJSON(item.Output)),
				ToolCallID: callID,
			})
			delete(pendingToolCalls, callID)
		}
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("input did not contain any chat-compatible messages")
	}
	return messages, nil
}

func responsesMessageItemToChatMessage(item ResponsesInputItem) (ChatMessage, bool) {
	role := strings.TrimSpace(item.Role)
	if role == "" {
		role = "user"
	}
	switch role {
	case "system", "developer":
		role = "system"
	case "user", "assistant":
	default:
		return ChatMessage{}, false
	}
	return ChatMessage{
		Role:    role,
		Content: responsesContentToChatContent(item.Content, role),
	}, true
}

func responsesContentToChatContent(raw json.RawMessage, role string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`""`)
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return json.RawMessage(mustMarshalJSON(text))
	}
	var parts []ResponsesContentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return raw
	}
	if role == "assistant" {
		var out strings.Builder
		for _, part := range parts {
			if strings.TrimSpace(part.Text) != "" {
				_, _ = out.WriteString(part.Text)
			}
		}
		return json.RawMessage(mustMarshalJSON(out.String()))
	}
	chatParts := make([]ChatContentPart, 0, len(parts))
	for _, part := range parts {
		switch strings.TrimSpace(part.Type) {
		case "input_text", "output_text", "text":
			chatParts = append(chatParts, ChatContentPart{Type: "text", Text: part.Text})
		case "input_image", "output_image":
			if strings.TrimSpace(part.ImageURL) == "" {
				continue
			}
			chatParts = append(chatParts, ChatContentPart{
				Type:     "image_url",
				ImageURL: &ChatImageURL{URL: strings.TrimSpace(part.ImageURL)},
			})
		}
	}
	if len(chatParts) == 0 {
		return json.RawMessage(`""`)
	}
	return json.RawMessage(mustMarshalJSON(chatParts))
}

func responsesToolsToChatTools(tools []ResponsesTool) []ChatTool {
	if len(tools) == 0 {
		return nil
	}
	out := make([]ChatTool, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Type) != "function" {
			continue
		}
		out = append(out, ChatTool{
			Type: "function",
			Function: &ChatFunction{
				Name:        strings.TrimSpace(tool.Name),
				Description: strings.TrimSpace(tool.Description),
				Parameters:  tool.Parameters,
				Strict:      tool.Strict,
			},
		})
	}
	return out
}

func responsesReasoningEffort(reasoning *ResponsesReasoning) string {
	if reasoning == nil {
		return ""
	}
	switch strings.TrimSpace(strings.ToLower(reasoning.Effort)) {
	case "low", "medium", "high":
		return strings.TrimSpace(strings.ToLower(reasoning.Effort))
	default:
		return ""
	}
}

func firstNonEmptyJSONText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mustMarshalJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return []byte(`null`)
	}
	return raw
}
