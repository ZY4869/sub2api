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
	toolConversion, err := convertResponsesToolsToChat(req.Tools)
	if err != nil {
		return nil, err
	}
	toolChoice, err := responsesToolChoiceToChat(req.ToolChoice, toolConversion)
	if err != nil {
		return nil, err
	}
	out := &ChatCompletionsRequest{
		Model:             strings.TrimSpace(req.Model),
		Messages:          messages,
		Temperature:       req.Temperature,
		TopP:              req.TopP,
		Stream:            req.Stream,
		Tools:             toolConversion.chatTools,
		ToolChoice:        toolChoice,
		ParallelToolCalls: req.ParallelToolCalls,
		ServiceTier:       strings.TrimSpace(req.ServiceTier),
		ReasoningEffort:   responsesReasoningEffort(req.Reasoning),
	}
	responseFormat := req.ResponseFormat
	if req.Text != nil && len(req.Text.Format) > 0 {
		responseFormat = req.Text.Format
	}
	if len(responseFormat) > 0 {
		format, err := responsesTextFormatToChatResponseFormat(responseFormat)
		if err != nil {
			return nil, err
		}
		out.ResponseFormat = format
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

func responsesToolChoiceToChat(raw json.RawMessage, conversion *responsesToolConversion) (json.RawMessage, error) {
	if len(raw) == 0 || conversion == nil {
		return raw, nil
	}
	var shorthand string
	if err := json.Unmarshal(raw, &shorthand); err == nil {
		switch strings.TrimSpace(shorthand) {
		case "", "auto", "none", "required":
			return raw, nil
		default:
			if proxyName, ok := conversion.byChoice[responsesToolChoiceKey("", shorthand, "")]; ok {
				return json.RawMessage(mustMarshalJSON(map[string]any{
					"type":     "function",
					"function": map[string]string{"name": proxyName},
				})), nil
			}
			return nil, nil
		}
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("tool_choice must be a string or object")
	}
	toolType := jsonRawString(obj["type"])
	switch normalizeResponsesToolType(toolType) {
	case "", "auto", "none", "required", "function":
		return raw, nil
	}
	name := firstNonEmptyCompat(jsonRawString(obj["name"]), jsonRawString(obj["tool"]))
	namespace := jsonRawString(obj["namespace"])
	if proxyName, ok := conversion.byChoice[responsesToolChoiceKey(toolType, name, namespace)]; ok {
		return json.RawMessage(mustMarshalJSON(map[string]any{
			"type":     "function",
			"function": map[string]string{"name": proxyName},
		})), nil
	}
	if proxyName, ok := conversion.byChoice[responsesToolChoiceKey(toolType, toolType, "")]; ok {
		return json.RawMessage(mustMarshalJSON(map[string]any{
			"type":     "function",
			"function": map[string]string{"name": proxyName},
		})), nil
	}
	return nil, nil
}

func jsonRawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return strings.TrimSpace(value)
	}
	return strings.Trim(strings.TrimSpace(string(raw)), `"`)
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
