package apicompat

import (
	"encoding/json"
	"strings"
)

func ChatCompletionsToResponsesResponse(chat *ChatCompletionsResponse, fallbackModel string) *ResponsesResponse {
	if chat == nil {
		return nil
	}
	model := strings.TrimSpace(chat.Model)
	if model == "" {
		model = strings.TrimSpace(fallbackModel)
	}
	resp := &ResponsesResponse{
		ID:     strings.TrimSpace(chat.ID),
		Object: "response",
		Model:  model,
		Status: "completed",
		Output: chatChoicesToResponsesOutput(chat.Choices),
	}
	if chat.Usage != nil {
		resp.Usage = chatUsageToResponsesUsage(chat.Usage)
	}
	return resp
}

func ChatCompletionsChunkToResponsesEvents(chunk *ChatCompletionsChunk, state *ChatToResponsesStreamState) []ResponsesStreamEvent {
	if chunk == nil {
		return nil
	}
	if state == nil {
		state = &ChatToResponsesStreamState{}
	}
	if strings.TrimSpace(chunk.ID) != "" {
		state.ID = strings.TrimSpace(chunk.ID)
	}
	if strings.TrimSpace(chunk.Model) != "" {
		state.Model = strings.TrimSpace(chunk.Model)
	}
	if chunk.Usage != nil {
		state.Usage = chatUsageToResponsesUsage(chunk.Usage)
	}
	var events []ResponsesStreamEvent
	if !state.CreatedSent {
		state.CreatedSent = true
		events = append(events, ResponsesStreamEvent{
			Type: "response.created",
			Response: &ResponsesResponse{
				ID:     state.responseID(),
				Object: "response",
				Model:  state.Model,
				Status: "in_progress",
			},
		})
	}
	for _, choice := range chunk.Choices {
		if choice.Delta.Role != "" && !state.MessageAdded {
			state.MessageAdded = true
			events = append(events, ResponsesStreamEvent{
				Type:        "response.output_item.added",
				OutputIndex: 0,
				Item: &ResponsesOutput{
					Type:   "message",
					ID:     state.messageID(),
					Role:   "assistant",
					Status: "in_progress",
				},
			})
		}
		if choice.Delta.Content != nil && *choice.Delta.Content != "" {
			state.MessageAdded = true
			_, _ = state.Text.WriteString(*choice.Delta.Content)
			events = append(events, ResponsesStreamEvent{
				Type:        "response.output_text.delta",
				OutputIndex: 0,
				Delta:       *choice.Delta.Content,
			})
		}
		if choice.FinishReason != nil {
			state.Finished = true
		}
	}
	return events
}

func FinalizeChatCompletionsToResponsesStream(state *ChatToResponsesStreamState) ResponsesStreamEvent {
	if state == nil {
		state = &ChatToResponsesStreamState{}
	}
	return ResponsesStreamEvent{
		Type: "response.completed",
		Response: &ResponsesResponse{
			ID:     state.responseID(),
			Object: "response",
			Model:  state.Model,
			Status: "completed",
			Output: state.output(),
			Usage:  state.Usage,
		},
	}
}

type ChatToResponsesStreamState struct {
	ID           string
	Model        string
	CreatedSent  bool
	MessageAdded bool
	Finished     bool
	Text         strings.Builder
	Usage        *ResponsesUsage
}

func (s *ChatToResponsesStreamState) responseID() string {
	if strings.TrimSpace(s.ID) != "" {
		return strings.TrimSpace(s.ID)
	}
	return "resp_" + strings.TrimPrefix(generateChatCmplID(), "chatcmpl-")
}

func (s *ChatToResponsesStreamState) messageID() string {
	if strings.TrimSpace(s.ID) != "" {
		return "msg_" + strings.TrimPrefix(strings.TrimSpace(s.ID), "chatcmpl-")
	}
	return "msg_" + strings.TrimPrefix(generateChatCmplID(), "chatcmpl-")
}

func (s *ChatToResponsesStreamState) output() []ResponsesOutput {
	return []ResponsesOutput{{
		Type:   "message",
		ID:     s.messageID(),
		Role:   "assistant",
		Status: "completed",
		Content: []ResponsesContentPart{{
			Type: "output_text",
			Text: s.Text.String(),
		}},
	}}
}

func chatChoicesToResponsesOutput(choices []ChatChoice) []ResponsesOutput {
	output := make([]ResponsesOutput, 0, len(choices))
	for _, choice := range choices {
		msg := choice.Message
		contentText := chatMessageContentAsText(msg.Content)
		parts := []ResponsesContentPart{{Type: "output_text", Text: contentText}}
		output = append(output, ResponsesOutput{
			Type:      "message",
			Role:      firstNonEmptyStringCompat(msg.Role, "assistant"),
			Content:   parts,
			Status:    "completed",
			CallID:    "",
			Name:      "",
			Arguments: "",
		})
		for _, call := range msg.ToolCalls {
			output = append(output, ResponsesOutput{
				Type:      "function_call",
				CallID:    strings.TrimSpace(call.ID),
				Name:      strings.TrimSpace(call.Function.Name),
				Arguments: strings.TrimSpace(call.Function.Arguments),
			})
		}
	}
	if len(output) == 0 {
		return []ResponsesOutput{{
			Type:    "message",
			Role:    "assistant",
			Status:  "completed",
			Content: []ResponsesContentPart{{Type: "output_text", Text: ""}},
		}}
	}
	return output
}

func chatMessageContentAsText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var parts []ChatContentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		var out strings.Builder
		for _, part := range parts {
			if part.Type == "text" {
				out.WriteString(part.Text)
			}
		}
		return out.String()
	}
	return strings.TrimSpace(string(raw))
}

func chatUsageToResponsesUsage(usage *ChatUsage) *ResponsesUsage {
	if usage == nil {
		return nil
	}
	out := &ResponsesUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
	}
	if out.TotalTokens == 0 {
		out.TotalTokens = out.InputTokens + out.OutputTokens
	}
	if usage.PromptTokensDetails != nil && usage.PromptTokensDetails.CachedTokens > 0 {
		out.InputTokensDetails = &ResponsesInputTokensDetails{CachedTokens: usage.PromptTokensDetails.CachedTokens}
	}
	return out
}

func firstNonEmptyStringCompat(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
