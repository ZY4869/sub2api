package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type kiroPayload struct {
	ConversationState kiroConversationState `json:"conversationState"`
	ProfileARN        string                `json:"profileArn,omitempty"`
	InferenceConfig   *kiroInferenceConfig  `json:"inferenceConfig,omitempty"`
}

type kiroInferenceConfig struct {
	MaxTokens   int     `json:"maxTokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"topP,omitempty"`
}

type kiroConversationState struct {
	AgentTaskType   string               `json:"agentTaskType,omitempty"`
	ChatTriggerType string               `json:"chatTriggerType"`
	ConversationID  string               `json:"conversationId"`
	CurrentMessage  kiroCurrentMessage   `json:"currentMessage"`
	History         []kiroHistoryMessage `json:"history,omitempty"`
}

type kiroCurrentMessage struct {
	UserInputMessage kiroUserInputMessage `json:"userInputMessage"`
}

type kiroHistoryMessage struct {
	UserInputMessage         *kiroUserInputMessage         `json:"userInputMessage,omitempty"`
	AssistantResponseMessage *kiroAssistantResponseMessage `json:"assistantResponseMessage,omitempty"`
}

type kiroUserInputMessage struct {
	Content                 string                       `json:"content"`
	ModelID                 string                       `json:"modelId"`
	Origin                  string                       `json:"origin"`
	Images                  []kiroImage                  `json:"images,omitempty"`
	UserInputMessageContext *kiroUserInputMessageContext `json:"userInputMessageContext,omitempty"`
}

type kiroUserInputMessageContext struct {
	ToolResults []kiroToolResult  `json:"toolResults,omitempty"`
	Tools       []kiroToolWrapper `json:"tools,omitempty"`
}

type kiroAssistantResponseMessage struct {
	Content  string        `json:"content"`
	ToolUses []kiroToolUse `json:"toolUses,omitempty"`
}

type kiroToolResult struct {
	Content   []kiroTextContent `json:"content"`
	Status    string            `json:"status"`
	ToolUseID string            `json:"toolUseId"`
}

type kiroTextContent struct {
	Text string `json:"text"`
}

type kiroToolWrapper struct {
	ToolSpecification kiroToolSpecification `json:"toolSpecification"`
}

type kiroToolSpecification struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema kiroInputSchema `json:"inputSchema"`
}

type kiroInputSchema struct {
	JSON any `json:"json"`
}

type kiroImage struct {
	Format string          `json:"format"`
	Source kiroImageSource `json:"source"`
}

type kiroImageSource struct {
	Bytes string `json:"bytes"`
}

func buildKiroClaudePayload(body []byte, modelID, profileARN, origin string, headers http.Header) ([]byte, error) {
	modelID = normalizeKiroRuntimeModelID(modelID)

	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("parse claude request: %w", err)
	}

	systemText := extractKiroSystemText(req["system"])
	if kiroRequestWantsThinking(req, headers) {
		prefix := "<thinking_mode>enabled</thinking_mode>\n<max_thinking_length>16000</max_thinking_length>"
		if systemText == "" {
			systemText = prefix
		} else {
			systemText = prefix + "\n\n" + systemText
		}
	}

	messages, _ := req["messages"].([]any)
	history, currentUser, currentToolResults := buildKiroConversation(messages, modelID, origin)
	if currentUser == nil {
		currentUser = &kiroUserInputMessage{
			Content: "Continue.",
			ModelID: modelID,
			Origin:  origin,
		}
	}
	currentUser.Content = prependKiroSystemText(systemText, currentUser.Content, len(currentToolResults) > 0)
	if strings.TrimSpace(currentUser.Content) == "" {
		if len(currentToolResults) > 0 {
			currentUser.Content = "[Tool results attached]"
		} else {
			currentUser.Content = "Continue."
		}
	}

	tools := buildKiroTools(req["tools"])
	if len(tools) > 0 || len(currentToolResults) > 0 {
		currentUser.UserInputMessageContext = &kiroUserInputMessageContext{
			Tools:       tools,
			ToolResults: currentToolResults,
		}
	}

	conversationID := uuid.NewString()
	if metadata, ok := req["metadata"].(map[string]any); ok {
		if userID := strings.TrimSpace(stringValue(metadata["user_id"])); userID != "" {
			conversationID = userID
		}
	}

	payload := kiroPayload{
		ConversationState: kiroConversationState{
			AgentTaskType:   kiroAgentMode,
			ChatTriggerType: "MANUAL",
			ConversationID:  conversationID,
			CurrentMessage:  kiroCurrentMessage{UserInputMessage: *currentUser},
			History:         history,
		},
		ProfileARN: strings.TrimSpace(profileARN),
	}
	if maxTokens, ok := intValue(req["max_tokens"]); ok && maxTokens > 0 {
		payload.InferenceConfig = &kiroInferenceConfig{MaxTokens: maxTokens}
	}
	if temperature, ok := floatValue(req["temperature"]); ok {
		if payload.InferenceConfig == nil {
			payload.InferenceConfig = &kiroInferenceConfig{}
		}
		payload.InferenceConfig.Temperature = temperature
	}
	if topP, ok := floatValue(req["top_p"]); ok {
		if payload.InferenceConfig == nil {
			payload.InferenceConfig = &kiroInferenceConfig{}
		}
		payload.InferenceConfig.TopP = topP
	}

	return json.Marshal(payload)
}

func buildKiroConversation(messages []any, modelID, origin string) ([]kiroHistoryMessage, *kiroUserInputMessage, []kiroToolResult) {
	var history []kiroHistoryMessage
	var currentUser *kiroUserInputMessage
	var currentToolResults []kiroToolResult

	for idx, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role := strings.TrimSpace(stringValue(msg["role"]))
		switch role {
		case "user":
			userMsg, toolResults := buildKiroUserMessage(msg["content"], modelID, origin)
			if idx == len(messages)-1 {
				currentUser = &userMsg
				currentToolResults = toolResults
				continue
			}
			copyMsg := userMsg
			if strings.TrimSpace(copyMsg.Content) == "" {
				copyMsg.Content = "[History message]"
			}
			if len(toolResults) > 0 {
				copyMsg.UserInputMessageContext = &kiroUserInputMessageContext{ToolResults: toolResults}
			}
			history = append(history, kiroHistoryMessage{UserInputMessage: &copyMsg})
		case "assistant":
			assistantMsg := buildKiroAssistantMessage(msg["content"])
			history = append(history, kiroHistoryMessage{AssistantResponseMessage: &assistantMsg})
		}
	}

	return history, currentUser, currentToolResults
}

func buildKiroUserMessage(content any, modelID, origin string) (kiroUserInputMessage, []kiroToolResult) {
	msg := kiroUserInputMessage{
		ModelID: modelID,
		Origin:  origin,
	}
	var toolResults []kiroToolResult

	switch value := content.(type) {
	case string:
		msg.Content = value
	case []any:
		var textParts []string
		for _, partRaw := range value {
			part, ok := partRaw.(map[string]any)
			if !ok {
				continue
			}
			switch strings.TrimSpace(stringValue(part["type"])) {
			case "text":
				textParts = append(textParts, stringValue(part["text"]))
			case "image":
				image := buildKiroImage(part)
				if image != nil {
					msg.Images = append(msg.Images, *image)
				}
			case "tool_result":
				toolUseID := strings.TrimSpace(stringValue(part["tool_use_id"]))
				if toolUseID == "" {
					continue
				}
				status := "success"
				if boolValue(part["is_error"]) {
					status = "error"
				}
				toolResults = append(toolResults, kiroToolResult{
					ToolUseID: toolUseID,
					Status:    status,
					Content:   []kiroTextContent{{Text: collectKiroTextContent(part["content"])}},
				})
			}
		}
		msg.Content = strings.Join(textParts, "")
	default:
		msg.Content = stringValue(content)
	}

	return msg, toolResults
}

func buildKiroAssistantMessage(content any) kiroAssistantResponseMessage {
	msg := kiroAssistantResponseMessage{}
	switch value := content.(type) {
	case string:
		msg.Content = value
	case []any:
		var textParts []string
		for _, partRaw := range value {
			part, ok := partRaw.(map[string]any)
			if !ok {
				continue
			}
			switch strings.TrimSpace(stringValue(part["type"])) {
			case "text":
				textParts = append(textParts, stringValue(part["text"]))
			case "thinking":
				thinking := stringValue(part["thinking"])
				if strings.TrimSpace(thinking) != "" {
					textParts = append(textParts, kiroThinkingStartTag+thinking+kiroThinkingEndTag)
				}
			case "tool_use":
				id := strings.TrimSpace(stringValue(part["id"]))
				name := strings.TrimSpace(stringValue(part["name"]))
				if id == "" || name == "" {
					continue
				}
				msg.ToolUses = append(msg.ToolUses, kiroToolUse{
					ID:    id,
					Name:  name,
					Input: mapValue(part["input"]),
				})
			}
		}
		msg.Content = strings.Join(textParts, "")
	default:
		msg.Content = stringValue(content)
	}
	if strings.TrimSpace(msg.Content) == "" && len(msg.ToolUses) > 0 {
		msg.Content = "[Assistant used tools]"
	}
	return msg
}

func buildKiroTools(raw any) []kiroToolWrapper {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	tools := make([]kiroToolWrapper, 0, len(items))
	for _, itemRaw := range items {
		item, ok := itemRaw.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(stringValue(item["name"]))
		if name == "" {
			continue
		}
		description := strings.TrimSpace(stringValue(item["description"]))
		if description == "" {
			description = "Tool: " + name
		}
		inputSchema := item["input_schema"]
		if inputSchema == nil {
			inputSchema = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		tools = append(tools, kiroToolWrapper{
			ToolSpecification: kiroToolSpecification{
				Name:        name,
				Description: description,
				InputSchema: kiroInputSchema{JSON: inputSchema},
			},
		})
	}
	return tools
}

func buildKiroImage(part map[string]any) *kiroImage {
	source, ok := part["source"].(map[string]any)
	if !ok {
		return nil
	}
	if strings.TrimSpace(stringValue(source["type"])) != "base64" {
		return nil
	}
	data := strings.TrimSpace(stringValue(source["data"]))
	if data == "" {
		return nil
	}
	mediaType := strings.TrimSpace(stringValue(source["media_type"]))
	format := "png"
	if slash := strings.LastIndex(mediaType, "/"); slash >= 0 && slash+1 < len(mediaType) {
		format = mediaType[slash+1:]
	}
	return &kiroImage{
		Format: format,
		Source: kiroImageSource{Bytes: data},
	}
}

func extractKiroSystemText(raw any) string {
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value)
	case []any:
		var parts []string
		for _, itemRaw := range value {
			if item, ok := itemRaw.(map[string]any); ok {
				if strings.TrimSpace(stringValue(item["type"])) == "text" {
					parts = append(parts, stringValue(item["text"]))
				}
			} else if text, ok := itemRaw.(string); ok {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, ""))
	default:
		return strings.TrimSpace(stringValue(raw))
	}
}

func prependKiroSystemText(systemText, userText string, hasToolResults bool) string {
	systemText = strings.TrimSpace(systemText)
	userText = strings.TrimSpace(userText)
	if systemText == "" {
		return userText
	}
	if userText == "" && hasToolResults {
		return systemText
	}
	if userText == "" {
		return systemText
	}
	return systemText + "\n\n" + userText
}

func kiroRequestWantsThinking(req map[string]any, headers http.Header) bool {
	if headers != nil && strings.Contains(strings.ToLower(headers.Get("anthropic-beta")), "interleaved-thinking") {
		return true
	}
	if thinking, ok := req["thinking"].(map[string]any); ok {
		return strings.EqualFold(strings.TrimSpace(stringValue(thinking["type"])), "enabled")
	}
	model := strings.ToLower(strings.TrimSpace(stringValue(req["model"])))
	return strings.Contains(model, "thinking") || strings.Contains(model, "reason")
}

func collectKiroTextContent(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case []any:
		var parts []string
		for _, itemRaw := range value {
			if item, ok := itemRaw.(map[string]any); ok {
				if strings.TrimSpace(stringValue(item["type"])) == "text" {
					parts = append(parts, stringValue(item["text"]))
					continue
				}
				if text := strings.TrimSpace(stringValue(item["text"])); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "")
	default:
		return stringValue(raw)
	}
}

func stringValue(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", raw)
	}
}

func intValue(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

func floatValue(raw any) (float64, bool) {
	switch value := raw.(type) {
	case float64:
		return value, true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	default:
		return 0, false
	}
}

func boolValue(raw any) bool {
	value, _ := raw.(bool)
	return value
}

func mapValue(raw any) map[string]any {
	value, _ := raw.(map[string]any)
	return value
}
