package apicompat

import (
	"encoding/json"
	"fmt"
)

// AnthropicToChatCompletionsRequest converts Anthropic Messages directly into
// the Chat Completions shape used by force-chat OpenAI protocol accounts. It
// reuses the local Responses representation as the semantic bridge without
// routing the request through the upstream Responses endpoint.
func AnthropicToChatCompletionsRequest(req *AnthropicRequest) (*ChatCompletionsRequest, map[string]ResponsesToolProxy, error) {
	if req == nil {
		return nil, nil, fmt.Errorf("anthropic request is required")
	}
	responsesReq, err := AnthropicToResponses(req)
	if err != nil {
		return nil, nil, err
	}
	chatReq, err := ResponsesToChatCompletionsRequest(responsesReq)
	if err != nil {
		return nil, nil, err
	}
	if len(req.StopSeqs) > 0 {
		stop, err := json.Marshal(req.StopSeqs)
		if err != nil {
			return nil, nil, err
		}
		chatReq.Stop = stop
	}
	toolProxies, err := ResponsesToolProxyMap(responsesReq.Tools)
	if err != nil {
		return nil, nil, err
	}
	return chatReq, toolProxies, nil
}

func ChatCompletionsResponseToAnthropic(chat *ChatCompletionsResponse, model string, toolProxies map[string]ResponsesToolProxy) *AnthropicResponse {
	return ResponsesToAnthropic(ChatCompletionsToResponsesResponseWithToolProxies(chat, model, toolProxies), model)
}

type ChatCompletionsToAnthropicStreamState struct {
	responses *ChatToResponsesStreamState
	anthropic *ResponsesEventToAnthropicState
}

func NewChatCompletionsToAnthropicStreamState(model string, toolProxies map[string]ResponsesToolProxy) *ChatCompletionsToAnthropicStreamState {
	return &ChatCompletionsToAnthropicStreamState{
		responses: &ChatToResponsesStreamState{
			Model:           model,
			ToolProxies:     toolProxies,
			ToolCallIndexes: map[int]bool{},
		},
		anthropic: NewResponsesEventToAnthropicState(),
	}
}

func ChatCompletionsChunkToAnthropicEvents(chunk *ChatCompletionsChunk, state *ChatCompletionsToAnthropicStreamState) []AnthropicStreamEvent {
	if state == nil {
		state = NewChatCompletionsToAnthropicStreamState("", nil)
	}
	var out []AnthropicStreamEvent
	for _, event := range ChatCompletionsChunkToResponsesEvents(chunk, state.responses) {
		out = append(out, ResponsesEventToAnthropicEvents(&event, state.anthropic)...)
	}
	return out
}

func FinalizeChatCompletionsAnthropicStream(state *ChatCompletionsToAnthropicStreamState) []AnthropicStreamEvent {
	if state == nil {
		state = NewChatCompletionsToAnthropicStreamState("", nil)
	}
	final := FinalizeChatCompletionsToResponsesStream(state.responses)
	events := ResponsesEventToAnthropicEvents(&final, state.anthropic)
	events = append(events, FinalizeResponsesAnthropicStream(state.anthropic)...)
	return events
}
