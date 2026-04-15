package service

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
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
	compatErr, ok := apicompat.AsCompatError(err)
	require.True(t, ok)
	require.Equal(t, apicompat.CompatReasonGeminiThinkingConflict, compatErr.Reason)
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
	compatErr, ok := apicompat.AsCompatError(err)
	require.True(t, ok)
	require.Equal(t, apicompat.CompatReasonGeminiMinimalThinkingUnsupported, compatErr.Reason)
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
	compatErr, ok := apicompat.AsCompatError(err)
	require.True(t, ok)
	require.Equal(t, apicompat.CompatReasonGeminiURLContextUnsupported, compatErr.Reason)
}

func TestConvertClaudeMessagesToGeminiGenerateContent_VersionedWebSearchTool(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gemini-3.1-pro-preview",
		"messages":[{"role":"user","content":"hello"}],
		"tools":[
			{"name":"get_weather","description":"Get weather","input_schema":{"type":"object"}},
			{"type":"web_search_20250305"}
		]
	}`)

	out, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(out, &payload))

	tools, _ := payload["tools"].([]any)
	require.Len(t, tools, 2)

	functionTool, ok := tools[0].(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, functionTool["functionDeclarations"])

	searchTool, ok := tools[1].(map[string]any)
	require.True(t, ok)
	_, hasGoogleSearch := searchTool["googleSearch"]
	require.True(t, hasGoogleSearch)
}

func TestConvertClaudeMessagesToGeminiGenerateContent_PreservesGeminiOfficialFields(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gemini-2.5-pro",
		"system":"anthropic system",
		"messages":[{"role":"user","content":"hello"}],
		"service_tier":"flex",
		"cachedContent":"cachedContents/cache-1",
		"safetySettings":[{"category":"HARM_CATEGORY_HATE_SPEECH","threshold":"BLOCK_ONLY_HIGH"}],
		"systemInstruction":{"parts":[{"text":"existing system"}]},
		"generationConfig":{"candidateCount":2,"topP":0.9,"responseModalities":["TEXT"]},
		"toolConfig":{"functionCallingConfig":{"mode":"ANY","allowedFunctionNames":["keep_me"]}},
		"tool_choice":{"type":"auto"},
		"max_tokens":256,
		"top_p":0.3
	}`)

	out, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(out, &payload))

	require.Equal(t, "flex", payload["service_tier"])
	require.Equal(t, "cachedContents/cache-1", payload["cachedContent"])

	safetySettings, ok := payload["safetySettings"].([]any)
	require.True(t, ok)
	require.Len(t, safetySettings, 1)

	systemInstruction, ok := payload["systemInstruction"].(map[string]any)
	require.True(t, ok)
	parts, ok := systemInstruction["parts"].([]any)
	require.True(t, ok)
	texts := make([]string, 0, len(parts))
	for _, rawPart := range parts {
		part, ok := rawPart.(map[string]any)
		require.True(t, ok)
		if text, ok := part["text"].(string); ok {
			texts = append(texts, text)
		}
	}
	require.Contains(t, texts, "existing system")
	require.Contains(t, texts, "anthropic system")

	generationConfig, ok := payload["generationConfig"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), generationConfig["candidateCount"])
	require.Equal(t, 0.9, generationConfig["topP"])
	require.Equal(t, float64(256), generationConfig["maxOutputTokens"])

	toolConfig, ok := payload["toolConfig"].(map[string]any)
	require.True(t, ok)
	functionCallingConfig, ok := toolConfig["functionCallingConfig"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "ANY", functionCallingConfig["mode"])
	require.Equal(t, []any{"keep_me"}, functionCallingConfig["allowedFunctionNames"])
}

func TestConvertClaudeMessagesToGeminiGenerateContent_PreservesExistingGeminiFunctionDeclarationTools(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"model":"gemini-2.5-pro",
		"messages":[{"role":"user","content":"hello"}],
		"tools":[
			{"functionDeclarations":[{"name":"existing_tool","description":"Keep me","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}]},
			{"name":"new_tool","description":"New tool","input_schema":{"type":"object","properties":{"zip":{"type":"string"}}}}
		]
	}`)

	out, err := convertClaudeMessagesToGeminiGenerateContent(body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(out, &payload))

	tools, ok := payload["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 2)

	existingTool, ok := tools[0].(map[string]any)
	require.True(t, ok)
	existingDecls, ok := existingTool["functionDeclarations"].([]any)
	require.True(t, ok)
	require.Len(t, existingDecls, 1)
	existingDecl, ok := existingDecls[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "existing_tool", existingDecl["name"])

	convertedTool, ok := tools[1].(map[string]any)
	require.True(t, ok)
	convertedDecls, ok := convertedTool["functionDeclarations"].([]any)
	require.True(t, ok)
	require.Len(t, convertedDecls, 1)
	convertedDecl, ok := convertedDecls[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "new_tool", convertedDecl["name"])
}
