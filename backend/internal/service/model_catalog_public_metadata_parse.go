package service

import (
	"encoding/json"
	"strings"
)

func normalizePublicModelContextWindowAny(raw any) PublicModelContextWindow {
	value, ok := raw.(map[string]any)
	if !ok {
		return PublicModelContextWindow{}
	}
	return normalizePublicModelContextWindow(PublicModelContextWindow{
		Tokens:        int64ValueFromAny(value["tokens"]),
		Source:        stringValueFromAny(value["source"]),
		Verified:      parseBoolAny(value["verified"]),
		LastCheckedAt: stringValueFromAny(value["last_checked_at"]),
		LimitKind:     stringValueFromAny(value["limit_kind"]),
		Notes:         normalizeStringSliceAny(value["notes"], strings.TrimSpace),
	}, 0, publicModelCatalogMetadataSource{})
}

func normalizePublicModelCapabilityMatrixAny(raw any) []PublicModelCapabilityMatrixEntry {
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	entries := make([]PublicModelCapabilityMatrixEntry, 0, len(values))
	for _, item := range values {
		value, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entries = append(entries, PublicModelCapabilityMatrixEntry{
			Capability:    stringValueFromAny(value["capability"]),
			Protocol:      stringValueFromAny(value["protocol"]),
			Endpoint:      stringValueFromAny(value["endpoint"]),
			Support:       stringValueFromAny(value["support"]),
			Mode:          stringValueFromAny(value["mode"]),
			Source:        stringValueFromAny(value["source"]),
			Verified:      parseBoolAny(value["verified"]),
			LastCheckedAt: stringValueFromAny(value["last_checked_at"]),
			Limitations:   normalizeStringSliceAny(value["limitations"], strings.TrimSpace),
		})
	}
	return dedupePublicModelCapabilityMatrix(entries)
}

func normalizePublicModelProtocolEndpointsAny(raw any) []PublicModelProtocolEndpoint {
	values, ok := raw.([]any)
	if !ok {
		return nil
	}
	endpoints := make([]PublicModelProtocolEndpoint, 0, len(values))
	for _, item := range values {
		value, ok := item.(map[string]any)
		if !ok {
			continue
		}
		endpoints = append(endpoints, PublicModelProtocolEndpoint{
			Key:           stringValueFromAny(value["key"]),
			Protocol:      stringValueFromAny(value["protocol"]),
			Endpoint:      stringValueFromAny(value["endpoint"]),
			Method:        stringValueFromAny(value["method"]),
			Support:       stringValueFromAny(value["support"]),
			Source:        stringValueFromAny(value["source"]),
			Verified:      parseBoolAny(value["verified"]),
			LastCheckedAt: stringValueFromAny(value["last_checked_at"]),
			Limitations:   normalizeStringSliceAny(value["limitations"], strings.TrimSpace),
		})
	}
	return dedupePublicModelProtocolEndpoints(endpoints)
}

func int64ValueFromAny(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		out, _ := v.Int64()
		return out
	default:
		return 0
	}
}
