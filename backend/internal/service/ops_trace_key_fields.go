package service

import (
	"encoding/json"
	"strings"
)

var opsTraceKeyFieldPaths = []string{
	"model",
	"reasoning_effort",
	"output_config.effort",
	"thinking.type",
	"thinking.budget_tokens",
	"service_tier",
	"max_tokens",
	"max_output_tokens",
	"max_completion_tokens",
	"cache_read_input_tokens",
	"cache_creation_input_tokens",
	"cache_creation.ephemeral_5m_input_tokens",
	"cache_creation.ephemeral_1h_input_tokens",
	"prompt_cache_hit_tokens",
	"prompt_cache_miss_tokens",
}

func ExtractOpsTraceKeyFieldsFromPayload(payload any) OpsTraceKeyFields {
	normalized := normalizeOpsTracePayloadEnvelopePayload(payload)
	return extractOpsTraceKeyFieldsFromValue(normalized)
}

func ExtractOpsTraceKeyFieldsFromBytes(payload []byte) OpsTraceKeyFields {
	if len(payload) == 0 {
		return nil
	}
	var parsed any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil
	}
	return extractOpsTraceKeyFieldsFromValue(redactSensitiveJSON(parsed))
}

func extractOpsTraceKeyFieldsFromValue(value any) OpsTraceKeyFields {
	root, ok := value.(map[string]any)
	if !ok || len(root) == 0 {
		return nil
	}
	fields := make(OpsTraceKeyFields)
	for _, path := range opsTraceKeyFieldPaths {
		if picked, exists := pickOpsTracePathValue(root, path); exists {
			fields[path] = picked
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func pickOpsTracePathValue(root map[string]any, path string) (any, bool) {
	current := any(root)
	for _, segment := range strings.Split(path, ".") {
		node, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, exists := node[segment]
		if !exists {
			return nil, false
		}
		current = next
	}
	switch typed := current.(type) {
	case nil, bool, string, float64:
		return typed, true
	case int:
		return typed, true
	case int64:
		return typed, true
	default:
		return nil, false
	}
}

func ExtractOpsTraceKeyFieldsFromEnvelopeJSON(raw string) OpsTraceKeyFields {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil
	}
	value, _ := parsed["key_fields"].(map[string]any)
	if len(value) == 0 {
		return nil
	}
	return normalizeOpsTraceKeyFields(value)
}
