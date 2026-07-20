package apicompat

import (
	"encoding/json"
	"sort"
	"strings"
)

func ChatCompletionsToResponsesResponse(chat *ChatCompletionsResponse, fallbackModel string) *ResponsesResponse {
	return ChatCompletionsToResponsesResponseWithToolProxies(chat, fallbackModel, nil)
}

func ChatCompletionsToResponsesResponseWithToolProxies(chat *ChatCompletionsResponse, fallbackModel string, proxies map[string]ResponsesToolProxy) *ResponsesResponse {
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
		Output: chatChoicesToResponsesOutput(chat.Choices, proxies),
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
			events = append(events, state.messageAddedEvent())
		}
		if choice.Delta.Content != nil && *choice.Delta.Content != "" {
			if !state.MessageAdded {
				events = append(events, state.messageAddedEvent())
			}
			if !state.ContentPartAdded {
				events = append(events, state.contentPartAddedEvent())
			}
			_, _ = state.Text.WriteString(*choice.Delta.Content)
			events = append(events, ResponsesStreamEvent{
				Type:         "response.output_text.delta",
				OutputIndex:  0,
				ContentIndex: 0,
				ItemID:       state.messageID(),
				Delta:        *choice.Delta.Content,
			})
		}
		for _, call := range choice.Delta.ToolCalls {
			if call.Index == nil {
				continue
			}
			idx := *call.Index
			if state.ToolCalls == nil {
				state.ToolCalls = map[int]*ChatToResponsesToolCallState{}
			}
			callState := state.ToolCalls[idx]
			if callState == nil && strings.TrimSpace(call.Function.Name) != "" {
				proxy := ResponsesToolProxy{}
				if state.ToolProxies != nil {
					proxy = state.ToolProxies[strings.TrimSpace(call.Function.Name)]
				}
				itemType := "function_call"
				itemName := strings.TrimSpace(call.Function.Name)
				if proxy.ProxyName != "" {
					itemType = proxy.OutputType()
					itemName = proxy.OutputName()
				}
				outputIndex := idx + 1
				item := &ResponsesOutput{
					Type:      itemType,
					CallID:    strings.TrimSpace(call.ID),
					Name:      itemName,
					Namespace: strings.TrimSpace(proxy.Namespace),
					Status:    "in_progress",
				}
				callState = &ChatToResponsesToolCallState{OutputIndex: outputIndex, Item: item}
				state.ToolCalls[idx] = callState
				if state.ToolCallIndexes != nil {
					state.ToolCallIndexes[idx] = true
				}
				events = append(events, ResponsesStreamEvent{
					Type:        "response.output_item.added",
					OutputIndex: outputIndex,
					Item:        cloneResponsesOutput(item),
				})
			}
			if callState != nil && strings.TrimSpace(call.ID) != "" && callState.Item.CallID == "" {
				callState.Item.CallID = strings.TrimSpace(call.ID)
			}
			if callState != nil && strings.TrimSpace(call.Function.Arguments) != "" {
				_, _ = callState.Arguments.WriteString(call.Function.Arguments)
				events = append(events, ResponsesStreamEvent{
					Type:        "response.function_call_arguments.delta",
					OutputIndex: callState.OutputIndex,
					CallID:      callState.Item.CallID,
					Name:        callState.Item.Name,
					Delta:       call.Function.Arguments,
				})
			}
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

func FinalizeChatCompletionsToResponsesStreamEvents(state *ChatToResponsesStreamState) []ResponsesStreamEvent {
	if state == nil {
		state = &ChatToResponsesStreamState{}
	}
	var events []ResponsesStreamEvent
	if state.ContentPartAdded && !state.OutputTextDoneSent {
		state.OutputTextDoneSent = true
		events = append(events, ResponsesStreamEvent{
			Type:         "response.output_text.done",
			OutputIndex:  0,
			ContentIndex: 0,
			ItemID:       state.messageID(),
			Text:         state.Text.String(),
		})
	}
	if state.ContentPartAdded && !state.ContentPartDoneSent {
		state.ContentPartDoneSent = true
		events = append(events, ResponsesStreamEvent{
			Type:         "response.content_part.done",
			OutputIndex:  0,
			ContentIndex: 0,
			ItemID:       state.messageID(),
			Part:         &ResponsesContentPart{Type: "output_text", Text: state.Text.String()},
		})
	}
	if state.MessageAdded && !state.MessageDoneSent {
		state.MessageDoneSent = true
		events = append(events, ResponsesStreamEvent{
			Type:        "response.output_item.done",
			OutputIndex: 0,
			Item:        state.messageOutput("completed"),
		})
	}
	for _, idx := range state.sortedToolCallIndexes() {
		call := state.ToolCalls[idx]
		if call == nil || call.Item == nil {
			continue
		}
		if !call.ArgumentsDoneSent {
			call.ArgumentsDoneSent = true
			call.Item.Arguments = call.Arguments.String()
			events = append(events, ResponsesStreamEvent{
				Type:        "response.function_call_arguments.done",
				OutputIndex: call.OutputIndex,
				CallID:      call.Item.CallID,
				Name:        call.Item.Name,
				Arguments:   call.Item.Arguments,
			})
		}
		if !call.ItemDoneSent {
			call.ItemDoneSent = true
			item := cloneResponsesOutput(call.Item)
			item.Status = "completed"
			item.Arguments = call.Arguments.String()
			events = append(events, ResponsesStreamEvent{
				Type:        "response.output_item.done",
				OutputIndex: call.OutputIndex,
				Item:        item,
			})
		}
	}
	events = append(events, FinalizeChatCompletionsToResponsesStream(state))
	return events
}

type ChatToResponsesStreamState struct {
	ID                  string
	Model               string
	CreatedSent         bool
	MessageAdded        bool
	ContentPartAdded    bool
	OutputTextDoneSent  bool
	ContentPartDoneSent bool
	MessageDoneSent     bool
	Finished            bool
	Text                strings.Builder
	Usage               *ResponsesUsage
	ToolProxies         map[string]ResponsesToolProxy

	ToolCallIndexes map[int]bool
	ToolCalls       map[int]*ChatToResponsesToolCallState
}

type ChatToResponsesToolCallState struct {
	OutputIndex       int
	Item              *ResponsesOutput
	Arguments         strings.Builder
	ArgumentsDoneSent bool
	ItemDoneSent      bool
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

func (s *ChatToResponsesStreamState) messageAddedEvent() ResponsesStreamEvent {
	s.MessageAdded = true
	return ResponsesStreamEvent{
		Type:        "response.output_item.added",
		OutputIndex: 0,
		Item: &ResponsesOutput{
			Type:   "message",
			ID:     s.messageID(),
			Role:   "assistant",
			Status: "in_progress",
		},
	}
}

func (s *ChatToResponsesStreamState) contentPartAddedEvent() ResponsesStreamEvent {
	s.ContentPartAdded = true
	return ResponsesStreamEvent{
		Type:         "response.content_part.added",
		OutputIndex:  0,
		ContentIndex: 0,
		ItemID:       s.messageID(),
		Part:         &ResponsesContentPart{Type: "output_text", Text: ""},
	}
}

func (s *ChatToResponsesStreamState) output() []ResponsesOutput {
	out := []ResponsesOutput{*s.messageOutput("completed")}
	for _, idx := range s.sortedToolCallIndexes() {
		call := s.ToolCalls[idx]
		if call == nil || call.Item == nil {
			continue
		}
		item := cloneResponsesOutput(call.Item)
		item.Status = "completed"
		item.Arguments = call.Arguments.String()
		out = append(out, *item)
	}
	return out
}

func (s *ChatToResponsesStreamState) messageOutput(status string) *ResponsesOutput {
	return &ResponsesOutput{
		Type:   "message",
		ID:     s.messageID(),
		Role:   "assistant",
		Status: status,
		Content: []ResponsesContentPart{{
			Type: "output_text",
			Text: s.Text.String(),
		}},
	}
}

func (s *ChatToResponsesStreamState) sortedToolCallIndexes() []int {
	if len(s.ToolCalls) == 0 {
		return nil
	}
	indexes := make([]int, 0, len(s.ToolCalls))
	for idx := range s.ToolCalls {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	return indexes
}

func cloneResponsesOutput(in *ResponsesOutput) *ResponsesOutput {
	if in == nil {
		return nil
	}
	out := *in
	if len(in.Content) > 0 {
		out.Content = append([]ResponsesContentPart(nil), in.Content...)
	}
	if len(in.Summary) > 0 {
		out.Summary = append([]ResponsesSummary(nil), in.Summary...)
	}
	if in.Action != nil {
		action := *in.Action
		out.Action = &action
	}
	return &out
}

func chatChoicesToResponsesOutput(choices []ChatChoice, proxies map[string]ResponsesToolProxy) []ResponsesOutput {
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
			output = append(output, responsesOutputFromChatToolCall(call, proxies))
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
				_, _ = out.WriteString(part.Text)
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
