package apicompat

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

func normalizeResponsesToolType(value string) string {
	switch strings.TrimSpace(value) {
	case "custom_tool":
		return "custom"
	case "tool_search_preview":
		return "tool_search"
	default:
		return strings.TrimSpace(value)
	}
}

func flattenedResponsesToolName(namespace, name string) string {
	namespace = strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if namespace == "" {
		return name
	}
	if name == "" {
		return namespace
	}
	return namespace + "." + name
}

func responsesToolChatName(proxy ResponsesToolProxy) string {
	seed := proxy.Type + "\x00" + proxy.Namespace + "\x00" + proxy.Name
	sum := sha1.Sum([]byte(seed))
	kind := sanitizeChatFunctionName(proxy.Type)
	if kind == "" {
		kind = "tool"
	}
	return responsesToolProxyPrefix + kind + "_" + hex.EncodeToString(sum[:])[:12]
}

func sanitizeChatFunctionName(value string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func responsesToolDescription(tool ResponsesTool, proxy ResponsesToolProxy) string {
	desc := strings.TrimSpace(tool.Description)
	if proxy.Type == "function" && proxy.Namespace == "" {
		return desc
	}
	name := proxy.OutputName()
	if desc == "" {
		return fmt.Sprintf("Responses %s tool proxy for %s.", proxy.Type, name)
	}
	return fmt.Sprintf("%s\n\nResponses %s tool proxy for %s.", desc, proxy.Type, name)
}

func responsesToolParameters(tool ResponsesTool) json.RawMessage {
	for _, raw := range []json.RawMessage{tool.Parameters, tool.Params, tool.InputSchema} {
		if len(raw) > 0 && strings.TrimSpace(string(raw)) != "" {
			return raw
		}
	}
	switch normalizeResponsesToolType(tool.Type) {
	case "tool_search":
		return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"additionalProperties":true}`)
	case "custom":
		return json.RawMessage(`{"type":"object","properties":{"input":{"type":"string"}},"additionalProperties":true}`)
	default:
		return json.RawMessage(`{"type":"object","additionalProperties":true}`)
	}
}

func responsesToolChoiceKey(toolType, name, namespace string) string {
	return normalizeResponsesToolType(toolType) + "\x00" + strings.TrimSpace(namespace) + "\x00" + strings.TrimSpace(name)
}

func firstNonEmptyCompat(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
