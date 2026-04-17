package service

import (
	"strings"

	"github.com/tidwall/gjson"
)

// ParseExplicitThinkingEnabledValue extracts an explicitly provided thinking.type
// flag from the inbound request body. It returns nil when the caller did not
// explicitly provide a supported thinking mode.
func ParseExplicitThinkingEnabledValue(body []byte) *bool {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return nil
	}

	switch strings.TrimSpace(strings.ToLower(gjson.GetBytes(body, "thinking.type").String())) {
	case "enabled", "adaptive":
		value := true
		return &value
	case "disabled":
		value := false
		return &value
	default:
		return nil
	}
}
