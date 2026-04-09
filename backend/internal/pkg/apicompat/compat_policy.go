package apicompat

import "errors"

type CompatFieldStrategy string

const (
	CompatFieldPassthrough CompatFieldStrategy = "passthrough"
	CompatFieldTranslate   CompatFieldStrategy = "translate"
	CompatFieldDelete      CompatFieldStrategy = "delete"
	CompatFieldReject      CompatFieldStrategy = "reject"
)

type CompatFieldRule struct {
	Field       string
	Strategy    CompatFieldStrategy
	TargetField string
	Notes       string
}

type CompatPolicy struct {
	Name           string
	SourceProtocol string
	TargetProtocol string
	FieldRules     []CompatFieldRule
}

type CompatError struct {
	Reason     string
	MessageKey string
	Message    string
	StatusCode int
	Err        error
}

func (e *CompatError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Reason
}

func (e *CompatError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewCompatError(reason string, messageKey string, message string) *CompatError {
	return &CompatError{
		Reason:     reason,
		MessageKey: messageKey,
		Message:    message,
		StatusCode: 400,
	}
}

func WrapCompatError(err error, reason string, messageKey string, message string) error {
	if err == nil {
		return nil
	}
	var compatErr *CompatError
	if errors.As(err, &compatErr) {
		return err
	}
	return &CompatError{
		Reason:     reason,
		MessageKey: messageKey,
		Message:    message,
		StatusCode: 400,
		Err:        err,
	}
}

func AsCompatError(err error) (*CompatError, bool) {
	var compatErr *CompatError
	if !errors.As(err, &compatErr) {
		return nil, false
	}
	return compatErr, true
}

const (
	CompatReasonAnthropicSystemInvalid           = "COMPAT_ANTHROPIC_SYSTEM_INVALID"
	CompatReasonAnthropicToolChoiceInvalid       = "COMPAT_ANTHROPIC_TOOL_CHOICE_INVALID"
	CompatReasonAnthropicMessageContentInvalid   = "COMPAT_ANTHROPIC_MESSAGE_CONTENT_INVALID"
	CompatReasonChatUserContentInvalid           = "COMPAT_CHAT_USER_CONTENT_INVALID"
	CompatReasonChatStringContentInvalid         = "COMPAT_CHAT_STRING_CONTENT_INVALID"
	CompatReasonChatFunctionCallInvalid          = "COMPAT_CHAT_FUNCTION_CALL_INVALID"
	CompatReasonGeminiMessagesInvalid            = "COMPAT_GEMINI_MESSAGES_INVALID"
	CompatReasonGeminiURLContextUnsupported      = "COMPAT_GEMINI_URL_CONTEXT_UNSUPPORTED"
	CompatReasonGeminiThinkingConflict           = "COMPAT_GEMINI_THINKING_CONFLICT"
	CompatReasonGeminiThinkingLevelUnsupported   = "COMPAT_GEMINI_THINKING_LEVEL_UNSUPPORTED"
	CompatReasonGeminiMinimalThinkingUnsupported = "COMPAT_GEMINI_MINIMAL_THINKING_UNSUPPORTED"
	CompatReasonGeminiReasoningNoneUnsupported   = "COMPAT_GEMINI_REASONING_NONE_UNSUPPORTED"
	CompatReasonGeminiMediaResolutionInvalid     = "COMPAT_GEMINI_MEDIA_RESOLUTION_INVALID"
	CompatReasonGeminiMediaResolutionUnsupported = "COMPAT_GEMINI_MEDIA_RESOLUTION_UNSUPPORTED"
	CompatReasonRuntimeRegistryMissing           = "COMPAT_RUNTIME_REGISTRY_MISSING"
	CompatReasonUpstreamRequestFailed            = "COMPAT_UPSTREAM_REQUEST_FAILED"
	CompatReasonUpstreamTerminalEventMissing     = "COMPAT_UPSTREAM_TERMINAL_EVENT_MISSING"
)

var anthropicMessagesToResponsesCompatPolicy = CompatPolicy{
	Name:           "anthropic_messages_to_responses",
	SourceProtocol: "anthropic",
	TargetProtocol: "responses",
	FieldRules: []CompatFieldRule{
		{Field: "model", Strategy: CompatFieldPassthrough},
		{Field: "temperature", Strategy: CompatFieldPassthrough},
		{Field: "top_p", Strategy: CompatFieldPassthrough},
		{Field: "stream", Strategy: CompatFieldPassthrough},
		{Field: "messages", Strategy: CompatFieldTranslate, TargetField: "input"},
		{Field: "system", Strategy: CompatFieldTranslate, TargetField: "input"},
		{Field: "max_tokens", Strategy: CompatFieldTranslate, TargetField: "max_output_tokens"},
		{Field: "tools", Strategy: CompatFieldTranslate, TargetField: "tools"},
		{Field: "tool_choice", Strategy: CompatFieldTranslate, TargetField: "tool_choice"},
		{Field: "output_config.effort", Strategy: CompatFieldTranslate, TargetField: "reasoning.effort"},
		{Field: "thinking", Strategy: CompatFieldDelete, Notes: "thinking blocks are translated through messages and not forwarded as a top-level request field"},
		{Field: "stop_sequences", Strategy: CompatFieldDelete},
	},
}

var chatCompletionsToResponsesCompatPolicy = CompatPolicy{
	Name:           "chat_completions_to_responses",
	SourceProtocol: "openai_chat_completions",
	TargetProtocol: "responses",
	FieldRules: []CompatFieldRule{
		{Field: "model", Strategy: CompatFieldPassthrough},
		{Field: "temperature", Strategy: CompatFieldPassthrough},
		{Field: "top_p", Strategy: CompatFieldPassthrough},
		{Field: "service_tier", Strategy: CompatFieldPassthrough},
		{Field: "messages", Strategy: CompatFieldTranslate, TargetField: "input"},
		{Field: "max_tokens", Strategy: CompatFieldTranslate, TargetField: "max_output_tokens"},
		{Field: "max_completion_tokens", Strategy: CompatFieldTranslate, TargetField: "max_output_tokens"},
		{Field: "tools", Strategy: CompatFieldTranslate, TargetField: "tools"},
		{Field: "functions", Strategy: CompatFieldTranslate, TargetField: "tools"},
		{Field: "tool_choice", Strategy: CompatFieldTranslate, TargetField: "tool_choice"},
		{Field: "function_call", Strategy: CompatFieldTranslate, TargetField: "tool_choice"},
		{Field: "reasoning_effort", Strategy: CompatFieldTranslate, TargetField: "reasoning.effort"},
		{Field: "stream", Strategy: CompatFieldDelete, Notes: "compat forwarding always streams upstream"},
		{Field: "stream_options", Strategy: CompatFieldDelete, Notes: "stream options stay on the downstream chat-completions response path"},
		{Field: "stop", Strategy: CompatFieldDelete},
	},
}

var anthropicMessagesToGeminiGenerateContentCompatPolicy = CompatPolicy{
	Name:           "anthropic_messages_to_gemini_generate_content",
	SourceProtocol: "anthropic",
	TargetProtocol: "gemini_generate_content",
	FieldRules: []CompatFieldRule{
		{Field: "messages", Strategy: CompatFieldTranslate, TargetField: "contents"},
		{Field: "system", Strategy: CompatFieldTranslate, TargetField: "systemInstruction"},
		{Field: "tools", Strategy: CompatFieldTranslate, TargetField: "tools"},
		{Field: "tool_choice", Strategy: CompatFieldTranslate, TargetField: "toolConfig.functionCallingConfig"},
		{Field: "max_tokens", Strategy: CompatFieldTranslate, TargetField: "generationConfig.maxOutputTokens"},
		{Field: "temperature", Strategy: CompatFieldTranslate, TargetField: "generationConfig.temperature"},
		{Field: "top_p", Strategy: CompatFieldTranslate, TargetField: "generationConfig.topP"},
		{Field: "top_k", Strategy: CompatFieldTranslate, TargetField: "generationConfig.topK"},
		{Field: "stop_sequences", Strategy: CompatFieldTranslate, TargetField: "generationConfig.stopSequences"},
		{Field: "thinking", Strategy: CompatFieldTranslate, TargetField: "generationConfig.thinkingConfig"},
		{Field: "reasoning_effort", Strategy: CompatFieldTranslate, TargetField: "generationConfig.thinkingConfig"},
		{Field: "response_format", Strategy: CompatFieldTranslate, TargetField: "generationConfig.responseMimeType"},
		{Field: "media_resolution", Strategy: CompatFieldTranslate, TargetField: "generationConfig.mediaResolution"},
		{Field: "urlContext", Strategy: CompatFieldReject, Notes: "rejected when the selected Gemini runtime does not support URL Context"},
	},
}

func AnthropicMessagesToResponsesCompatPolicy() CompatPolicy {
	return anthropicMessagesToResponsesCompatPolicy
}

func ChatCompletionsToResponsesCompatPolicy() CompatPolicy {
	return chatCompletionsToResponsesCompatPolicy
}

func AnthropicMessagesToGeminiGenerateContentCompatPolicy() CompatPolicy {
	return anthropicMessagesToGeminiGenerateContentCompatPolicy
}
