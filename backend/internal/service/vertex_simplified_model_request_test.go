package service

import (
	"encoding/json"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSimplifiedVertexModelRequest_FriendlyBody(t *testing.T) {
	body := []byte(`{
		"messages":[{"role":"user","content":"hello"}],
		"system":"be concise",
		"temperature":0.2,
		"response_mime_type":"application/json",
		"response_schema":{"type":"object","properties":{"answer":{"type":"string"}}}
	}`)

	normalized, err := NormalizeSimplifiedVertexModelRequest("gemini-2.5-pro", "generateContent", body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(normalized, &payload))
	require.NotNil(t, payload["contents"])
	require.NotNil(t, payload["systemInstruction"])

	generationConfig, _ := payload["generationConfig"].(map[string]any)
	require.Equal(t, "application/json", generationConfig["responseMimeType"])
	require.NotNil(t, generationConfig["responseJsonSchema"])
}

func TestNormalizeSimplifiedVertexModelRequest_FriendlyBodyAcceptsToolsAndMetadata(t *testing.T) {
	body := []byte(`{
		"messages":[{"role":"user","content":"hello"}],
		"tools":[{"name":"lookup","description":"Lookup data","input_schema":{"type":"object","properties":{"id":{"type":"string"}}}}],
		"tool_choice":{"type":"tool","name":"lookup"},
		"safety_settings":[{"category":"HARM_CATEGORY_HATE_SPEECH","threshold":"BLOCK_ONLY_HIGH"}],
		"labels":{"source":"vertex-simplified"},
		"metadata":{"trace_id":"trace-1"}
	}`)

	normalized, err := NormalizeSimplifiedVertexModelRequest("gemini-2.5-pro", "generateContent", body)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(normalized, &payload))
	require.NotNil(t, payload["contents"])
	require.NotNil(t, payload["tools"])
	require.NotNil(t, payload["toolConfig"])
	require.NotNil(t, payload["safetySettings"])
	require.Equal(t, map[string]any{"source": "vertex-simplified"}, payload["labels"])
	require.Equal(t, map[string]any{"trace_id": "trace-1"}, payload["metadata"])
}

func TestNormalizeSimplifiedVertexModelRequest_RejectsMixedBody(t *testing.T) {
	_, err := NormalizeSimplifiedVertexModelRequest("gemini-2.5-pro", "generateContent", []byte(`{
		"messages":[{"role":"user","content":"hello"}],
		"contents":[{"role":"user","parts":[{"text":"hello"}]}]
	}`))
	appErr := infraerrors.FromError(err)
	require.NotNil(t, appErr)
	require.Equal(t, "VERTEX_SIMPLIFIED_BODY_MIXED", appErr.Reason)
}

func TestNormalizeSimplifiedVertexModelRequest_RejectsSafetyAliasMixedWithNativeBody(t *testing.T) {
	_, err := NormalizeSimplifiedVertexModelRequest("gemini-2.5-pro", "generateContent", []byte(`{
		"contents":[{"role":"user","parts":[{"text":"hello"}]}],
		"safety_settings":[{"category":"HARM_CATEGORY_HATE_SPEECH","threshold":"BLOCK_ONLY_HIGH"}]
	}`))
	appErr := infraerrors.FromError(err)
	require.NotNil(t, appErr)
	require.Equal(t, "VERTEX_SIMPLIFIED_BODY_MIXED", appErr.Reason)
}

func TestNormalizeSimplifiedVertexModelRequest_RejectsModelConflict(t *testing.T) {
	_, err := NormalizeSimplifiedVertexModelRequest("gemini-2.5-pro", "generateContent", []byte(`{
		"model":"gemini-2.5-flash",
		"messages":[{"role":"user","content":"hello"}]
	}`))
	appErr := infraerrors.FromError(err)
	require.NotNil(t, appErr)
	require.Equal(t, "VERTEX_SIMPLIFIED_MODEL_CONFLICT", appErr.Reason)
}

func TestNormalizeSimplifiedVertexModelRequest_AllowsNativeBody(t *testing.T) {
	normalized, err := NormalizeSimplifiedVertexModelRequest("gemini-2.5-pro", "countTokens", []byte(`{
		"model":"gemini-2.5-pro",
		"contents":[{"role":"user","parts":[{"text":"hello"}]}]
	}`))
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(normalized, &payload))
	require.NotContains(t, payload, "model")
	require.NotNil(t, payload["contents"])
}

func TestNormalizeSimplifiedVertexModelRequest_AllowsNativeBodyWithTools(t *testing.T) {
	normalized, err := NormalizeSimplifiedVertexModelRequest("gemini-2.5-pro", "generateContent", []byte(`{
		"contents":[{"role":"user","parts":[{"text":"hello"}]}],
		"tools":[{"functionDeclarations":[{"name":"lookup","parameters":{"type":"object"}}]}]
	}`))
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(normalized, &payload))
	require.NotNil(t, payload["contents"])
	require.NotNil(t, payload["tools"])
}
