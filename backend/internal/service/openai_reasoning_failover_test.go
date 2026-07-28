package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIReasoningFailover_TrimsForeignEncryptedReasoningFromCanonicalCopy(t *testing.T) {
	originalReasoning := map[string]any{
		"type":              "reasoning",
		"encrypted_content": "foreign-provider-ciphertext",
		"summary":           []any{map[string]any{"type": "summary_text", "text": "keep"}},
	}
	canonical := map[string]any{
		"model":                "gpt-5.3-codex",
		"previous_response_id": "resp_foreign",
		"input": []any{
			originalReasoning,
			map[string]any{"type": "input_text", "text": "hello"},
		},
	}

	canonicalInput, ok := canonical["input"].([]any)
	require.True(t, ok)

	retry := map[string]any{
		"model":                canonical["model"],
		"previous_response_id": canonical["previous_response_id"],
		"input":                append([]any(nil), canonicalInput...),
	}
	changed := trimOpenAIEncryptedReasoningItems(retry)
	if !HasFunctionCallOutput(retry) {
		delete(retry, "previous_response_id")
	}

	require.True(t, changed)
	require.NotContains(t, retry, "previous_response_id")
	retryInput, ok := retry["input"].([]any)
	require.True(t, ok)
	require.Len(t, retryInput, 2)
	retryReasoning, ok := retryInput[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "reasoning", retryReasoning["type"])
	require.NotContains(t, retryReasoning, "encrypted_content")
	require.Contains(t, retryReasoning, "summary")

	require.Equal(t, "foreign-provider-ciphertext", originalReasoning["encrypted_content"], "canonical source request must stay reusable for other failover branches")
	require.Equal(t, "resp_foreign", canonical["previous_response_id"])
}
