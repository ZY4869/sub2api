package service

import (
	"encoding/json"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type vertexSimplifiedModelBodyKind string

const (
	vertexSimplifiedModelBodyNative     vertexSimplifiedModelBodyKind = "native"
	vertexSimplifiedModelBodySimplified vertexSimplifiedModelBodyKind = "simplified"
)

var vertexSimplifiedNativeModelKeys = []string{
	"contents", "systemInstruction", "system_instruction", "generationConfig", "generation_config",
	"toolConfig", "tool_config", "cachedContent", "cached_content", "safetySettings",
}

var vertexSimplifiedFriendlyModelKeys = []string{
	"system", "messages", "temperature", "top_p", "top_k", "max_tokens", "stop", "tool_choice",
	"response_mime_type", "response_schema", "thinking", "safety_settings",
}

var vertexSimplifiedSharedModelKeys = []string{
	"tools", "labels", "metadata",
}

func NormalizeSimplifiedVertexModelRequest(modelName string, action string, body []byte) ([]byte, error) {
	trimmedBody := strings.TrimSpace(string(body))
	if trimmedBody == "" {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_EMPTY", "request body is empty")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmedBody), &payload); err != nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_INVALID", "invalid JSON body")
	}

	if bodyModel := strings.TrimSpace(stringValueFromAny(payload["model"])); bodyModel != "" && !vertexSimplifiedModelsMatch(modelName, bodyModel) {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_MODEL_CONFLICT", "path model does not match body model")
	}

	kind, err := detectVertexSimplifiedModelBodyKind(payload)
	if err != nil {
		return nil, err
	}
	switch kind {
	case vertexSimplifiedModelBodyNative:
		delete(payload, "model")
		return json.Marshal(payload)
	case vertexSimplifiedModelBodySimplified:
		if !vertexSimplifiedActionSupportsFriendlyBody(action) {
			return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_ACTION_UNSUPPORTED", "simplified body only supports generateContent, streamGenerateContent, and countTokens")
		}
		return buildFriendlyVertexModelBody(modelName, payload)
	default:
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_UNSUPPORTED", "unsupported Vertex request body")
	}
}

func detectVertexSimplifiedModelBodyKind(payload map[string]any) (vertexSimplifiedModelBodyKind, error) {
	hasNative := vertexSimplifiedHasAnyKey(payload, vertexSimplifiedNativeModelKeys)
	hasFriendly := vertexSimplifiedHasAnyKey(payload, vertexSimplifiedFriendlyModelKeys)
	if hasNative && hasFriendly {
		return "", infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_MIXED", "do not mix simplified fields with native Vertex fields")
	}
	switch {
	case hasNative:
		return vertexSimplifiedModelBodyNative, nil
	case hasFriendly:
		return vertexSimplifiedModelBodySimplified, nil
	case payload != nil && vertexSimplifiedHasAnyKey(payload, vertexSimplifiedSharedModelKeys):
		switch {
		case payload["contents"] != nil || payload["systemInstruction"] != nil || payload["system_instruction"] != nil:
			return vertexSimplifiedModelBodyNative, nil
		case payload["messages"] != nil || payload["system"] != nil:
			return vertexSimplifiedModelBodySimplified, nil
		}
		return "", infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_UNSUPPORTED", "request body must include either native Vertex fields or simplified messages fields")
	case payload != nil && payload["contents"] != nil:
		return vertexSimplifiedModelBodyNative, nil
	case payload != nil && payload["messages"] != nil:
		return vertexSimplifiedModelBodySimplified, nil
	default:
		return "", infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_UNSUPPORTED", "request body must include either native Vertex fields or simplified messages fields")
	}
}

func buildFriendlyVertexModelBody(modelName string, payload map[string]any) ([]byte, error) {
	if !vertexSimplifiedHasMessages(payload) {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_MESSAGES_REQUIRED", "simplified Vertex body requires messages")
	}

	prepared, _ := deepCloneGeminiValue(payload).(map[string]any)
	if prepared == nil {
		prepared = map[string]any{}
	}
	prepared["model"] = vertexSimplifiedCanonicalModelID(modelName)
	if stopValue, ok := prepared["stop"]; ok && prepared["stop_sequences"] == nil && prepared["stopSequences"] == nil {
		switch typed := stopValue.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				prepared["stop_sequences"] = []any{typed}
			}
		case []any:
			prepared["stop_sequences"] = typed
		}
	}

	rawPrepared, err := json.Marshal(prepared)
	if err != nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_INVALID", "invalid simplified Vertex body")
	}
	normalized, err := convertClaudeMessagesToGeminiGenerateContent(rawPrepared, geminiTransformOptions{AllowURLContext: false})
	if err != nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_INVALID", err.Error())
	}

	var out map[string]any
	if err := json.Unmarshal(normalized, &out); err != nil {
		return nil, infraerrors.BadRequest("VERTEX_SIMPLIFIED_BODY_INVALID", "invalid converted Vertex body")
	}
	if out == nil {
		out = map[string]any{}
	}
	applyFriendlyVertexStructuredOutput(out, payload)
	copyGeminiRequestFieldExact(out, payload, "labels")
	copyGeminiRequestFieldExact(out, payload, "metadata")
	return json.Marshal(out)
}

func applyFriendlyVertexStructuredOutput(out map[string]any, payload map[string]any) {
	if out == nil || payload == nil {
		return
	}
	generationConfig, _ := out["generationConfig"].(map[string]any)
	if generationConfig == nil {
		generationConfig = map[string]any{}
	}
	if mimeType := strings.TrimSpace(stringValueFromAny(payload["response_mime_type"])); mimeType != "" {
		setGeminiValueIfMissing(generationConfig, "responseMimeType", mimeType)
	}
	if schema := normalizeGeminiSchema(payload["response_schema"]); schema != nil {
		setGeminiValueIfMissing(generationConfig, "responseJsonSchema", schema)
		setGeminiValueIfMissing(generationConfig, "responseMimeType", "application/json")
	}
	if len(generationConfig) == 0 {
		delete(out, "generationConfig")
		return
	}
	out["generationConfig"] = generationConfig
}

func vertexSimplifiedHasMessages(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	items, ok := payload["messages"].([]any)
	return ok && len(items) > 0
}

func vertexSimplifiedHasAnyKey(payload map[string]any, keys []string) bool {
	if payload == nil {
		return false
	}
	for _, key := range keys {
		if value, ok := payload[key]; ok && value != nil {
			return true
		}
	}
	return false
}

func vertexSimplifiedActionSupportsFriendlyBody(action string) bool {
	switch strings.TrimSpace(action) {
	case "generateContent", "streamGenerateContent", "countTokens":
		return true
	default:
		return false
	}
}

func vertexSimplifiedCanonicalModelID(value string) string {
	modelID := strings.TrimSpace(value)
	modelID = strings.TrimPrefix(modelID, "publishers/google/models/")
	modelID = strings.TrimPrefix(modelID, "models/")
	return normalizeVertexUpstreamModelID(modelID)
}

func vertexSimplifiedModelsMatch(left string, right string) bool {
	return strings.EqualFold(vertexSimplifiedCanonicalModelID(left), vertexSimplifiedCanonicalModelID(right))
}
