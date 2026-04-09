package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func compatRuleByField(t *testing.T, policy CompatPolicy, field string) CompatFieldRule {
	t.Helper()
	for _, rule := range policy.FieldRules {
		if rule.Field == field {
			return rule
		}
	}
	t.Fatalf("missing compat rule for field %q in policy %q", field, policy.Name)
	return CompatFieldRule{}
}

func TestAnthropicMessagesToResponsesCompatPolicy(t *testing.T) {
	t.Parallel()

	policy := AnthropicMessagesToResponsesCompatPolicy()
	require.Equal(t, "anthropic_messages_to_responses", policy.Name)
	require.Equal(t, CompatFieldTranslate, compatRuleByField(t, policy, "messages").Strategy)
	require.Equal(t, "input", compatRuleByField(t, policy, "messages").TargetField)
	require.Equal(t, CompatFieldTranslate, compatRuleByField(t, policy, "max_tokens").Strategy)
	require.Equal(t, CompatFieldDelete, compatRuleByField(t, policy, "stop_sequences").Strategy)
}

func TestChatCompletionsToResponsesCompatPolicy(t *testing.T) {
	t.Parallel()

	policy := ChatCompletionsToResponsesCompatPolicy()
	require.Equal(t, "chat_completions_to_responses", policy.Name)
	require.Equal(t, CompatFieldTranslate, compatRuleByField(t, policy, "messages").Strategy)
	require.Equal(t, CompatFieldTranslate, compatRuleByField(t, policy, "function_call").Strategy)
	require.Equal(t, CompatFieldDelete, compatRuleByField(t, policy, "stream").Strategy)
	require.Equal(t, CompatFieldPassthrough, compatRuleByField(t, policy, "service_tier").Strategy)
}

func TestAnthropicMessagesToGeminiGenerateContentCompatPolicy(t *testing.T) {
	t.Parallel()

	policy := AnthropicMessagesToGeminiGenerateContentCompatPolicy()
	require.Equal(t, "anthropic_messages_to_gemini_generate_content", policy.Name)
	require.Equal(t, CompatFieldTranslate, compatRuleByField(t, policy, "messages").Strategy)
	require.Equal(t, CompatFieldTranslate, compatRuleByField(t, policy, "thinking").Strategy)
	require.Equal(t, CompatFieldReject, compatRuleByField(t, policy, "urlContext").Strategy)
}

func TestAnthropicToResponsesReturnsCompatErrorForInvalidSystem(t *testing.T) {
	t.Parallel()

	_, err := AnthropicToResponses(&AnthropicRequest{
		Model:     "gpt-5.4",
		MaxTokens: 128,
		System:    json.RawMessage(`123`),
		Messages: []AnthropicMessage{
			{Role: "user", Content: json.RawMessage(`"hello"`)},
		},
	})

	require.Error(t, err)
	compatErr, ok := AsCompatError(err)
	require.True(t, ok)
	require.Equal(t, CompatReasonAnthropicSystemInvalid, compatErr.Reason)
	require.Equal(t, "compat.anthropic.system_invalid", compatErr.MessageKey)
}

func TestChatCompletionsToResponsesReturnsCompatErrorForInvalidUserContent(t *testing.T) {
	t.Parallel()

	_, err := ChatCompletionsToResponses(&ChatCompletionsRequest{
		Model: "gpt-5.4",
		Messages: []ChatMessage{
			{Role: "user", Content: json.RawMessage(`123`)},
		},
	})

	require.Error(t, err)
	compatErr, ok := AsCompatError(err)
	require.True(t, ok)
	require.Equal(t, CompatReasonChatUserContentInvalid, compatErr.Reason)
	require.Equal(t, "compat.chat.user_content_invalid", compatErr.MessageKey)
}
