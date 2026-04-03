package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertClaudeMessagesToGeminiGenerateContent_Gemini3ThinkingLevelAndMediaResolution(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gemini-3-flash-preview",
		"reasoning_effort":"high",
		"media_resolution":"medium",
		"messages":[{"role":"user","content":"hello"}]
	}`)

	out, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(out, &payload))

	generationConfig, _ := payload["generationConfig"].(map[string]any)
	require.Equal(t, "MEDIUM", generationConfig["mediaResolution"])

	thinkingConfig, _ := generationConfig["thinkingConfig"].(map[string]any)
	require.Equal(t, true, thinkingConfig["includeThoughts"])
	require.Equal(t, "HIGH", thinkingConfig["thinkingLevel"])
}

func TestConvertClaudeMessagesToGeminiGenerateContent_Gemini3ThinkingConflict(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gemini-3-flash-preview",
		"reasoning_effort":"low",
		"thinking":{"type":"enabled","budget_tokens":2048},
		"messages":[{"role":"user","content":"hello"}]
	}`)

	_, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "thinkingLevel and thinkingBudget")
}

func TestConvertClaudeMessagesToGeminiGenerateContent_Gemini3MinimalThinkingRestricted(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gemini-3.1-pro-preview",
		"reasoning_effort":"none",
		"messages":[{"role":"user","content":"hello"}]
	}`)

	_, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MINIMAL")
}

func TestConvertClaudeMessagesToGeminiGenerateContent_Gemini3LegacyBudgetCompatibility(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gemini-3.1-flash-lite-preview",
		"thinking":{"type":"enabled","budgetTokens":2048},
		"messages":[{"role":"user","content":"hello"}]
	}`)

	out, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(out, &payload))

	generationConfig, _ := payload["generationConfig"].(map[string]any)
	thinkingConfig, _ := generationConfig["thinkingConfig"].(map[string]any)
	require.Equal(t, true, thinkingConfig["includeThoughts"])
	require.Equal(t, float64(2048), thinkingConfig["thinkingBudget"])
}

func TestConvertClaudeMessagesToGeminiGenerateContent_BuiltInToolsAndServerSideInvocations(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gemini-3.1-pro-preview",
		"messages":[{"role":"user","content":"hello"}],
		"tools":[
			{"name":"get_weather","description":"Get weather","input_schema":{"type":"object","properties":{"city":{"type":"string"}}}},
			{"type":"google_search"},
			{"type":"code_execution"},
			{"type":"google_maps"},
			{"type":"file_search"}
		],
		"tool_config":{"include_server_side_tool_invocations":true}
	}`)

	out, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(out, &payload))

	tools, _ := payload["tools"].([]any)
	require.Len(t, tools, 5)

	serialized := string(out)
	for _, key := range []string{"googleSearch", "codeExecution", "googleMaps", "fileSearch"} {
		require.Contains(t, serialized, `"`+key+`"`)
	}

	toolConfig, _ := payload["toolConfig"].(map[string]any)
	require.Equal(t, true, toolConfig["includeServerSideToolInvocations"])
	functionCallingConfig, _ := toolConfig["functionCallingConfig"].(map[string]any)
	require.Equal(t, "VALIDATED", functionCallingConfig["mode"])
}

func TestConvertClaudeMessagesToGeminiGenerateContent_URLContextRejectedForVertex(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gemini-3-flash-preview",
		"messages":[{"role":"user","content":"hello"}],
		"tools":[{"type":"url_context"}]
	}`)

	_, err := convertClaudeMessagesToGeminiGenerateContent(body, geminiTransformOptions{AllowURLContext: false})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "urlContext") || strings.Contains(err.Error(), "URL Context"))
}
