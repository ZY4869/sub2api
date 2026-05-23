package service

import (
	"encoding/json"
	"net/http"
	"strings"
)

func sanitizeOpenAIEmptyThinkingBlocksInJSON(body []byte) ([]byte, bool, error) {
	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return body, false, err
	}
	if !sanitizeOpenAIEmptyThinkingBlocks(reqBody) {
		return body, false, nil
	}
	nextBody, err := json.Marshal(reqBody)
	if err != nil {
		return body, false, err
	}
	return nextBody, true, nil
}

func sanitizeOpenAIEmptyThinkingBlocks(reqBody map[string]any) bool {
	if len(reqBody) == 0 {
		return false
	}
	_, changed, _ := sanitizeOpenAIThinkingValue(reqBody)
	return changed
}

func sanitizeOpenAIThinkingValue(value any) (any, bool, bool) {
	switch v := value.(type) {
	case map[string]any:
		if next, changed, keep := sanitizeOpenAIThinkingBlock(v); changed || !keep {
			return next, changed, keep
		}
		changed := false
		for key, child := range v {
			nextChild, childChanged, keep := sanitizeOpenAIThinkingValue(child)
			if !childChanged && keep {
				continue
			}
			changed = true
			if !keep {
				delete(v, key)
				continue
			}
			v[key] = nextChild
		}
		return v, changed, true
	case []any:
		filtered := v[:0]
		changed := false
		for _, item := range v {
			nextItem, itemChanged, keep := sanitizeOpenAIThinkingValue(item)
			if itemChanged {
				changed = true
			}
			if !keep {
				changed = true
				continue
			}
			filtered = append(filtered, nextItem)
		}
		return filtered, changed, true
	default:
		return value, false, true
	}
}

func sanitizeOpenAIThinkingBlock(block map[string]any) (any, bool, bool) {
	blockType := strings.ToLower(strings.TrimSpace(stringValueFromAny(block["type"])))
	switch blockType {
	case "thinking":
		if strings.TrimSpace(stringValueFromAny(block["thinking"])) != "" {
			return block, false, true
		}
		if text := strings.TrimSpace(stringValueFromAny(block["text"])); text != "" {
			block["thinking"] = text
			delete(block, "text")
			return block, true, true
		}
		return nil, true, false
	case "redacted_thinking":
		if strings.TrimSpace(stringValueFromAny(block["data"])) != "" {
			return block, false, true
		}
		return nil, true, false
	default:
		return block, false, true
	}
}

func isOpenAIEmptyThinkingBlockError(statusCode int, upstreamMsg string, upstreamBody []byte) bool {
	if statusCode != 0 && statusCode != http.StatusBadRequest {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(upstreamMsg))
	if msg == "" {
		msg = strings.ToLower(strings.TrimSpace(extractUpstreamErrorMessage(upstreamBody)))
	}
	if msg == "" || !strings.Contains(msg, "thinking") {
		return false
	}
	return strings.Contains(msg, "must contain thinking") ||
		strings.Contains(msg, "thinking block must contain") ||
		strings.Contains(msg, "empty thinking")
}
