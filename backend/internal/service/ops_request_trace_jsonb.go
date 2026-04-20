package service

import (
	"encoding/json"
	"strings"
)

const (
	opsTraceJSONBActionSanitized = "sanitized"
	opsTraceJSONBActionEnveloped = "enveloped"
)

type opsTraceJSONBNormalizationResult struct {
	Value  *string
	Action string
}

func normalizeOpsTraceJSONBPayload(value *string, source string, contentType string) opsTraceJSONBNormalizationResult {
	if value == nil {
		return opsTraceJSONBNormalizationResult{}
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" || trimmed == "null" {
		return opsTraceJSONBNormalizationResult{}
	}

	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
		sanitized, changed := sanitizeOpsTraceJSONBValue(parsed)
		raw, marshalErr := json.Marshal(sanitized)
		if marshalErr == nil {
			out := string(raw)
			action := ""
			if changed {
				action = opsTraceJSONBActionSanitized
			}
			return opsTraceJSONBNormalizationResult{Value: &out, Action: action}
		}
	}

	payload, _ := sanitizeOpsTraceJSONString(trimmed)
	if payload == "" {
		return opsTraceJSONBNormalizationResult{}
	}
	normalizedContentType := strings.TrimSpace(contentType)
	if normalizedContentType == "" {
		normalizedContentType = "application/json"
	}
	return opsTraceJSONBNormalizationResult{
		Value:  BuildOpsTracePayloadEnvelopeJSON(OpsTracePayloadStateCaptured, source, payload, normalizedContentType, false),
		Action: opsTraceJSONBActionEnveloped,
	}
}

func sanitizeOpsTraceJSONBValue(value any) (any, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		out := make(map[string]any, len(typed))
		changed := false
		for key, item := range typed {
			sanitizedKey, keyChanged := sanitizeOpsTraceJSONString(key)
			sanitizedItem, itemChanged := sanitizeOpsTraceJSONBValue(item)
			out[sanitizedKey] = sanitizedItem
			changed = changed || keyChanged || itemChanged
		}
		return out, changed
	case []any:
		out := make([]any, 0, len(typed))
		changed := false
		for _, item := range typed {
			sanitizedItem, itemChanged := sanitizeOpsTraceJSONBValue(item)
			out = append(out, sanitizedItem)
			changed = changed || itemChanged
		}
		return out, changed
	case string:
		sanitized, changed := sanitizeOpsTraceJSONString(typed)
		return sanitized, changed
	default:
		return value, false
	}
}

func sanitizeOpsTraceJSONString(value string) (string, bool) {
	withoutNUL := strings.ReplaceAll(value, "\u0000", "")
	sanitized := strings.ToValidUTF8(withoutNUL, "")
	return sanitized, sanitized != value
}
