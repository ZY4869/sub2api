package handler

import (
	"encoding/json"
	"strings"
)

func parseOpsTraceJSON(payload []byte) map[string]any {
	if len(payload) == 0 || !json.Valid(payload) {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil
	}
	return out
}

func parseOpsTraceResponseJSON(payload []byte) map[string]any {
	if parsed := parseOpsTraceJSON(payload); parsed != nil {
		return parsed
	}
	lines := strings.Split(string(payload), "\n")
	var last map[string]any
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		if parsed := parseOpsTraceJSON([]byte(data)); parsed != nil {
			last = parsed
		}
	}
	return last
}

func inferOpsTraceToolKinds(tool any) []string {
	payload, ok := tool.(map[string]any)
	if !ok || payload == nil {
		return nil
	}
	kinds := make([]string, 0, 4)
	if typeValue := strings.TrimSpace(stringValueFromMap(payload, "type")); typeValue != "" {
		kinds = append(kinds, typeValue)
	}
	for key, mapped := range map[string]string{
		"googleSearch":          "googleSearch",
		"googleSearchRetrieval": "googleSearch",
		"codeExecution":         "codeExecution",
		"googleMaps":            "googleMaps",
		"fileSearch":            "fileSearch",
		"urlContext":            "urlContext",
		"functionDeclarations":  "function",
	} {
		if _, exists := payload[key]; exists {
			kinds = append(kinds, mapped)
		}
	}
	return dedupeTraceStrings(kinds)
}

func dedupeTraceStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
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
	return out
}

func stringValueFromMap(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	return stringValueFromAny(payload[key])
}

func stringValueFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strings.TrimSpace(strings.TrimRight(strings.TrimRight(strconvFloat(typed), "0"), "."))
	default:
		return ""
	}
}

func intValueFromMap(payload map[string]any, key string) *int {
	if payload == nil {
		return nil
	}
	return firstNonNilInt(payload[key])
}

func firstNonNilInt(values ...any) *int {
	for _, value := range values {
		switch typed := value.(type) {
		case int:
			v := typed
			return &v
		case int32:
			v := int(typed)
			return &v
		case int64:
			v := int(typed)
			return &v
		case float64:
			v := int(typed)
			return &v
		case json.Number:
			if parsed, err := typed.Int64(); err == nil {
				v := int(parsed)
				return &v
			}
		}
	}
	return nil
}

func filterOpsTraceHeaders(headers map[string][]string, allowlist []string) map[string]string {
	if len(headers) == 0 || len(allowlist) == 0 {
		return nil
	}
	allowed := make(map[string]string, len(allowlist))
	for _, key := range allowlist {
		values, ok := headers[key]
		if !ok || len(values) == 0 {
			values = headers[strings.ToLower(key)]
		}
		if len(values) == 0 {
			continue
		}
		allowed[strings.ToLower(key)] = truncateString(strings.Join(values, ", "), 1024)
	}
	if len(allowed) == 0 {
		return nil
	}
	return allowed
}

func marshalOpsTraceHeaders(headers map[string]string) *string {
	if len(headers) == 0 {
		return nil
	}
	raw, err := json.Marshal(headers)
	if err != nil {
		return nil
	}
	value := string(raw)
	return &value
}

func serviceHashString(value string) uint64 {
	var out uint64
	for _, ch := range []byte(value) {
		out = out*131 + uint64(ch)
	}
	return out
}

var opsTraceRequestHeaderAllowlist = []string{
	"Content-Type",
	"Accept",
	"User-Agent",
	"anthropic-version",
	"anthropic-beta",
	"openai-beta",
	"x-request-id",
}

var opsTraceResponseHeaderAllowlist = []string{
	"Content-Type",
	"X-Request-Id",
	"x-goog-request-id",
	"X-Sub2api-CountTokens-Source",
	"x-ratelimit-limit-requests",
	"x-ratelimit-remaining-requests",
	"x-ratelimit-reset-requests",
}
