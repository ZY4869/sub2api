package service

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
)

type OpsTracePayloadState string

const (
	OpsTracePayloadStateCaptured OpsTracePayloadState = "captured"
	OpsTracePayloadStateEmpty    OpsTracePayloadState = "empty"
	OpsTracePayloadStateRawOnly  OpsTracePayloadState = "raw_only"

	opsTracePayloadInlineBytesLimit = 64 * 1024
	opsTraceLargeStringBytesLimit   = 16 * 1024

	opsTraceNormalizedRequestPayloadKey = "ops_trace_normalized_request_payload"
	opsTraceUpstreamRequestPayloadKey   = "ops_trace_upstream_request_payload"
	opsTraceUpstreamResponsePayloadKey  = "ops_trace_upstream_response_payload"
	opsTraceGatewayResponsePayloadKey   = "ops_trace_gateway_response_payload"
	opsTraceToolTracePayloadKey         = "ops_trace_tool_trace_payload"
	opsTracePayloadPreviewLimitKey      = "ops_trace_payload_preview_limit"
)

type OpsTracePayloadEnvelope struct {
	State       OpsTracePayloadState `json:"state"`
	Source      string               `json:"source,omitempty"`
	Truncated   bool                 `json:"truncated,omitempty"`
	ContentType string               `json:"content_type,omitempty"`
	KeyFields   map[string]any       `json:"key_fields,omitempty"`
	Payload     any                  `json:"payload,omitempty"`
}

func SetOpsTraceNormalizedRequest(c *gin.Context, source string, payload any) {
	setOpsTracePayload(c, opsTraceNormalizedRequestPayloadKey, OpsTracePayloadStateCaptured, source, payload, "application/json", false)
}

func SetOpsTraceUpstreamRequest(c *gin.Context, source string, payload any, contentType string, truncated bool) {
	setOpsTracePayload(c, opsTraceUpstreamRequestPayloadKey, OpsTracePayloadStateCaptured, source, payload, contentType, truncated)
}

func SetOpsTraceUpstreamResponse(c *gin.Context, source string, payload any, contentType string, truncated bool) {
	setOpsTracePayload(c, opsTraceUpstreamResponsePayloadKey, OpsTracePayloadStateCaptured, source, payload, contentType, truncated)
}

func SetOpsTraceGatewayResponse(c *gin.Context, source string, payload any, contentType string, truncated bool) {
	setOpsTracePayload(c, opsTraceGatewayResponsePayloadKey, OpsTracePayloadStateCaptured, source, payload, contentType, truncated)
}

func SetOpsTraceToolTrace(c *gin.Context, source string, payload any) {
	setOpsTracePayload(c, opsTraceToolTracePayloadKey, OpsTracePayloadStateCaptured, source, payload, "application/json", false)
}

func SetOpsTracePayloadPreviewLimit(c *gin.Context, maxBytes int) {
	if c == nil || maxBytes <= 0 {
		return
	}
	c.Set(opsTracePayloadPreviewLimitKey, maxBytes)
}

func GetOpsTraceNormalizedRequestJSON(c *gin.Context) *string {
	return getOpsTracePayloadJSON(c, opsTraceNormalizedRequestPayloadKey)
}

func GetOpsTraceUpstreamRequestJSON(c *gin.Context) *string {
	return getOpsTracePayloadJSON(c, opsTraceUpstreamRequestPayloadKey)
}

func GetOpsTraceUpstreamResponseJSON(c *gin.Context) *string {
	return getOpsTracePayloadJSON(c, opsTraceUpstreamResponsePayloadKey)
}

func GetOpsTraceGatewayResponseJSON(c *gin.Context) *string {
	return getOpsTracePayloadJSON(c, opsTraceGatewayResponsePayloadKey)
}

func GetOpsTraceToolTraceJSON(c *gin.Context) *string {
	return getOpsTracePayloadJSON(c, opsTraceToolTracePayloadKey)
}

func BuildOpsTracePayloadEnvelopeJSON(state OpsTracePayloadState, source string, payload any, contentType string, truncated bool) *string {
	return BuildOpsTracePayloadEnvelopeJSONWithKeyFields(state, source, payload, contentType, truncated, nil)
}

func BuildOpsTracePayloadEnvelopeJSONWithKeyFields(state OpsTracePayloadState, source string, payload any, contentType string, truncated bool, keyFields map[string]any) *string {
	state = normalizeOpsTracePayloadState(state)
	normalizedPayload := normalizeOpsTracePayloadEnvelopePayload(payload)
	if normalizedPayload == nil {
		state = OpsTracePayloadStateEmpty
	} else if compacted, compactedPayload := compactOpsTracePayloadValue(normalizedPayload); compactedPayload {
		normalizedPayload = compacted
		truncated = true
	}
	envelope := OpsTracePayloadEnvelope{
		State:       state,
		Source:      strings.TrimSpace(source),
		Truncated:   truncated,
		ContentType: strings.TrimSpace(contentType),
		KeyFields:   normalizeOpsTraceKeyFields(keyFields),
		Payload:     normalizedPayload,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return nil
	}
	if len(raw) > opsTracePayloadInlineBytesLimit {
		envelope.Payload = buildOpsTraceOmittedPayloadSummary(len(raw), opsTracePayloadInlineBytesLimit, "envelope_exceeds_preview_limit")
		envelope.Truncated = true
		raw, err = json.Marshal(envelope)
		if err != nil {
			return nil
		}
	}
	value := string(raw)
	return &value
}

func BuildOpsTracePayloadEnvelopeJSONFromBytes(payload []byte, maxBytes int, state OpsTracePayloadState, source string, contentType string) *string {
	return BuildOpsTracePayloadEnvelopeJSONFromBytesWithKeyFields(payload, maxBytes, state, source, contentType, nil)
}

func BuildOpsTracePayloadEnvelopeJSONFromBytesWithKeyFields(payload []byte, maxBytes int, state OpsTracePayloadState, source string, contentType string, keyFields map[string]any) *string {
	if len(payload) == 0 {
		return BuildOpsTracePayloadEnvelopeJSONWithKeyFields(OpsTracePayloadStateEmpty, source, nil, contentType, false, keyFields)
	}
	if maxBytes > 0 && len(payload) > maxBytes {
		return BuildOpsTracePayloadEnvelopeJSONWithKeyFields(
			state,
			source,
			buildOpsTraceOmittedPayloadSummary(len(payload), maxBytes, "payload_exceeds_preview_limit"),
			contentType,
			true,
			keyFields,
		)
	}

	sanitized, truncated, _ := sanitizeAndTrimRequestBody(payload, maxBytes)
	if strings.TrimSpace(sanitized) != "" {
		if parsed, ok := parseOpsTracePayloadValue(sanitized); ok {
			return BuildOpsTracePayloadEnvelopeJSONWithKeyFields(state, source, parsed, contentType, truncated, keyFields)
		}
	}

	text := strings.TrimSpace(string(payload))
	if text == "" {
		return BuildOpsTracePayloadEnvelopeJSONWithKeyFields(OpsTracePayloadStateEmpty, source, nil, contentType, truncated, keyFields)
	}
	if maxBytes > 0 && len(text) > maxBytes {
		text = truncateString(text, maxBytes)
		truncated = true
	}
	if contentType == "" {
		contentType = "text/plain"
	}
	return BuildOpsTracePayloadEnvelopeJSONWithKeyFields(state, source, text, contentType, truncated, keyFields)
}

func setOpsTracePayload(c *gin.Context, key string, state OpsTracePayloadState, source string, payload any, contentType string, truncated bool) {
	if c == nil || strings.TrimSpace(key) == "" {
		return
	}
	previewLimit := resolveOpsTracePayloadPreviewLimit(c)
	keyFields := ExtractOpsTraceKeyFieldsFromPayload(payload)
	switch typed := payload.(type) {
	case []byte:
		keyFields = ExtractOpsTraceKeyFieldsFromBytes(typed)
		if value := BuildOpsTracePayloadEnvelopeJSONFromBytesWithKeyFields(typed, previewLimit, state, source, contentType, keyFields); value != nil {
			c.Set(key, *value)
		}
		return
	case string:
		if len(typed) > previewLimit {
			payload = buildOpsTraceOmittedPayloadSummary(len(typed), previewLimit, "payload_exceeds_preview_limit")
			truncated = true
		}
	}
	if value := BuildOpsTracePayloadEnvelopeJSONWithKeyFields(state, source, payload, contentType, truncated, keyFields); value != nil {
		c.Set(key, *value)
	}
}

func resolveOpsTracePayloadPreviewLimit(c *gin.Context) int {
	if c == nil {
		return opsTracePayloadInlineBytesLimit
	}
	value, ok := c.Get(opsTracePayloadPreviewLimitKey)
	if !ok {
		return opsTracePayloadInlineBytesLimit
	}
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	}
	return opsTracePayloadInlineBytesLimit
}

func getOpsTracePayloadJSON(c *gin.Context, key string) *string {
	if c == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	value, ok := c.Get(key)
	if !ok {
		return nil
	}
	switch cast := value.(type) {
	case string:
		trimmed := strings.TrimSpace(cast)
		if trimmed == "" {
			return nil
		}
		return &trimmed
	case *string:
		if cast == nil {
			return nil
		}
		trimmed := strings.TrimSpace(*cast)
		if trimmed == "" {
			return nil
		}
		return &trimmed
	default:
		return nil
	}
}

func normalizeOpsTracePayloadState(state OpsTracePayloadState) OpsTracePayloadState {
	switch state {
	case OpsTracePayloadStateCaptured, OpsTracePayloadStateEmpty, OpsTracePayloadStateRawOnly:
		return state
	default:
		return OpsTracePayloadStateCaptured
	}
}

func normalizeOpsTracePayloadEnvelopePayload(payload any) any {
	switch value := payload.(type) {
	case nil:
		return nil
	case []byte:
		parsed, ok := parseOpsTracePayloadValue(string(value))
		if ok {
			return parsed
		}
		return nil
	case string:
		parsed, ok := parseOpsTracePayloadValue(value)
		if ok {
			return parsed
		}
		return nil
	default:
		return payload
	}
}

func compactOpsTracePayloadValue(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		changed := false
		for key, item := range typed {
			compacted, itemChanged := compactOpsTracePayloadValue(item)
			out[key] = compacted
			changed = changed || itemChanged
		}
		return out, changed
	case []any:
		out := make([]any, 0, len(typed))
		changed := false
		for _, item := range typed {
			compacted, itemChanged := compactOpsTracePayloadValue(item)
			out = append(out, compacted)
			changed = changed || itemChanged
		}
		return out, changed
	case string:
		if len(typed) > opsTraceLargeStringBytesLimit {
			return buildOpsTraceOmittedPayloadSummary(len(typed), opsTraceLargeStringBytesLimit, "large_string_omitted"), true
		}
	}
	return value, false
}

func buildOpsTraceOmittedPayloadSummary(bytesLen int, limit int, reason string) map[string]any {
	if bytesLen < 0 {
		bytesLen = 0
	}
	if limit < 0 {
		limit = 0
	}
	return map[string]any{
		"omitted":             true,
		"reason":              strings.TrimSpace(reason),
		"bytes":               bytesLen,
		"preview_limit_bytes": limit,
	}
}

func normalizeOpsTraceKeyFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		switch typed := value.(type) {
		case nil, bool, float64, int, int64, string:
			out[trimmedKey] = typed
		case json.Number:
			out[trimmedKey] = typed.String()
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseOpsTracePayloadValue(raw string) (any, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, false
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return parsed, true
	}
	return raw, true
}
