package service

import (
	"strconv"
	"strings"
)

func buildEncryptedTracePayload(key string, payload []byte, maxBytes int) ([]byte, *int, bool, error) {
	if len(payload) == 0 {
		return nil, nil, false, nil
	}
	trimmed, size, truncated := trimTraceRawPayload(payload, maxBytes)
	ciphertext, err := encryptOpsRequestTracePayload(key, trimmed)
	return ciphertext, size, truncated, err
}

func trimTraceRawPayload(payload []byte, maxBytes int) ([]byte, *int, bool) {
	if len(payload) == 0 {
		return nil, nil, false
	}
	size := len(payload)
	sizePtr := &size
	if maxBytes > 0 && len(payload) > maxBytes {
		copied := append([]byte(nil), payload[:maxBytes]...)
		return copied, sizePtr, true
	}
	return append([]byte(nil), payload...), sizePtr, false
}

func sanitizeTracePayloadForStorage(payload []byte, maxBytes int, contentType string) *string {
	if len(payload) == 0 {
		return nil
	}
	keyFields := ExtractOpsTraceKeyFieldsFromBytes(payload)
	if sanitized, truncated, _ := sanitizeAndTrimRequestBody(payload, maxBytes); strings.TrimSpace(sanitized) != "" {
		if strings.TrimSpace(contentType) == "" {
			contentType = "application/json"
		}
		if parsed, ok := parseOpsTracePayloadValue(sanitized); ok {
			return BuildOpsTracePayloadEnvelopeJSONWithKeyFields(
				OpsTracePayloadStateCaptured,
				"legacy_capture_fallback",
				parsed,
				contentType,
				truncated,
				keyFields,
			)
		}
	}
	return BuildOpsTracePayloadEnvelopeJSONFromBytesWithKeyFields(
		payload,
		maxBytes,
		OpsTracePayloadStateCaptured,
		"legacy_capture_fallback",
		contentType,
		keyFields,
	)
}

func buildOpsRequestTraceSearchText(input *OpsInsertRequestTraceInput) string {
	if input == nil {
		return ""
	}
	parts := []string{
		input.RequestID,
		input.ClientRequestID,
		input.UpstreamRequestID,
		input.Platform,
		input.ProtocolIn,
		input.ProtocolOut,
		input.Channel,
		input.RoutePath,
		input.RequestType,
		input.RequestedModel,
		input.UpstreamModel,
		input.ActualUpstreamModel,
		input.Status,
		input.FinishReason,
		input.PromptBlockReason,
		input.CaptureReason,
		input.ThinkingSource,
		input.ThinkingLevel,
		input.MediaResolution,
		input.CountTokensSource,
		strings.Join(input.ToolKinds, " "),
	}
	for _, raw := range []*string{
		input.InboundRequestJSON,
		input.NormalizedRequestJSON,
		input.UpstreamRequestJSON,
		input.UpstreamResponseJSON,
		input.GatewayResponseJSON,
		input.ToolTraceJSON,
	} {
		parts = append(parts, buildOpsTraceKeyFieldSearchTerms(raw)...)
	}
	value := strings.Join(dedupeNonEmptyStrings(parts), " ")
	if len(value) > opsRequestTraceSearchTextLimit {
		value = truncateString(value, opsRequestTraceSearchTextLimit)
	}
	return value
}

func buildOpsTraceKeyFieldSearchTerms(raw *string) []string {
	if raw == nil {
		return nil
	}
	fields := ExtractOpsTraceKeyFieldsFromEnvelopeJSON(*raw)
	if len(fields) == 0 {
		return nil
	}
	terms := make([]string, 0, len(fields)*2)
	for key, value := range fields {
		terms = append(terms, key)
		switch typed := value.(type) {
		case string:
			terms = append(terms, typed)
		case bool:
			if typed {
				terms = append(terms, "true")
			} else {
				terms = append(terms, "false")
			}
		case float64:
			terms = append(terms, formatFloatForSearch(typed))
		case int:
			terms = append(terms, strconv.Itoa(typed))
		case int64:
			terms = append(terms, strconv.FormatInt(typed, 10))
		}
	}
	return terms
}

func formatFloatForSearch(value float64) string {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	return strings.TrimSpace(text)
}

func dedupeNonEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}
