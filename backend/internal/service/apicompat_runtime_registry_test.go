package service

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
)

func TestCompatRuntimeRegistryEntriesExposePolicies(t *testing.T) {
	t.Parallel()

	entries := CompatRuntimeRegistryEntries()
	require.Len(t, entries, 3)

	entry, ok := LookupCompatRuntimeRegistryEntry(compatRuntimePolicyAnthropicMessagesToResponses)
	require.True(t, ok)
	require.Equal(t, "anthropic", entry.SourceProtocol)
	require.Equal(t, "responses", entry.TargetProtocol)
	require.Equal(t, apicompat.AnthropicMessagesToResponsesCompatPolicy().Name, entry.PolicyDefinition.Name)
}

func TestConvertAnthropicMessagesToResponsesRuntimeUsesRegistry(t *testing.T) {
	t.Parallel()

	req := &apicompat.AnthropicRequest{
		Model:     "gpt-5.4",
		MaxTokens: 64,
		System:    json.RawMessage(`"system"`),
		Messages: []apicompat.AnthropicMessage{
			{Role: "user", Content: json.RawMessage(`"hello"`)},
		},
	}

	responsesReq, entry, err := ConvertAnthropicMessagesToResponsesRuntime(req)
	require.NoError(t, err)
	require.NotNil(t, responsesReq)
	require.Equal(t, compatRuntimePolicyAnthropicMessagesToResponses, entry.PolicyID)
	require.Equal(t, req.Model, responsesReq.Model)
}

func TestConvertChatCompletionsToResponsesRuntimeUsesRegistry(t *testing.T) {
	t.Parallel()

	req := &apicompat.ChatCompletionsRequest{
		Model: "gpt-5.4",
		Messages: []apicompat.ChatMessage{
			{Role: "user", Content: json.RawMessage(`"hello"`)},
		},
	}

	responsesReq, entry, err := ConvertChatCompletionsToResponsesRuntime(req)
	require.NoError(t, err)
	require.NotNil(t, responsesReq)
	require.Equal(t, compatRuntimePolicyChatCompletionsToResponses, entry.PolicyID)
	require.True(t, responsesReq.Stream)
}

func TestConvertAnthropicMessagesToGeminiGenerateContentRuntimeUsesRegistry(t *testing.T) {
	t.Parallel()

	body := []byte(`{"model":"gemini-3-flash-preview","messages":[{"role":"user","content":"hello"}]}`)

	geminiReq, entry, err := ConvertAnthropicMessagesToGeminiGenerateContentRuntime(body)
	require.NoError(t, err)
	require.NotEmpty(t, geminiReq)
	require.Equal(t, compatRuntimePolicyAnthropicMessagesToGeminiGenContent, entry.PolicyID)
}

func TestRequireCompatRuntimeRegistryEntryReturnsCompatErrorWhenMissing(t *testing.T) {
	t.Parallel()

	_, err := requireCompatRuntimeRegistryEntry("missing_policy")
	require.Error(t, err)

	compatErr, ok := apicompat.AsCompatError(err)
	require.True(t, ok)
	require.Equal(t, apicompat.CompatReasonRuntimeRegistryMissing, compatErr.Reason)
	require.Equal(t, "compat.runtime.registry_missing", compatErr.MessageKey)
}
