package admin

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func parseChannelMonitorTemplateIDField(raw json.RawMessage) (*int64, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, true, nil
	}
	var id int64
	if err := json.Unmarshal(trimmed, &id); err != nil || id <= 0 {
		return nil, true, service.ErrChannelMonitorInvalidTemplateID
	}
	return &id, true, nil
}

func parseChannelMonitorHeaders(raw json.RawMessage) (map[string]string, bool, error) {
	value, present, err := parseChannelMonitorObject(raw, service.ErrChannelMonitorInvalidHeaders)
	if err != nil || !present {
		return nil, present, err
	}
	headers := make(map[string]string, len(value))
	for key, rawValue := range value {
		text, ok := rawValue.(string)
		if !ok {
			return nil, true, service.ErrChannelMonitorInvalidHeaders
		}
		headers[key] = text
	}
	return headers, true, nil
}

func parseChannelMonitorBodyOverride(raw json.RawMessage) (map[string]any, bool, error) {
	return parseChannelMonitorObject(raw, service.ErrChannelMonitorInvalidBodyOverride)
}

func parseChannelMonitorMode(raw json.RawMessage, invalidErr error) (string, bool, error) {
	if raw == nil {
		return "", false, nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", true, nil
	}
	var value string
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return "", true, invalidErr
	}
	return strings.TrimSpace(value), true, nil
}

func parseChannelMonitorObject(raw json.RawMessage, invalidErr error) (map[string]any, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return map[string]any{}, true, nil
	}
	var value map[string]any
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return nil, true, invalidErr
	}
	if value == nil {
		value = map[string]any{}
	}
	return value, true, nil
}
