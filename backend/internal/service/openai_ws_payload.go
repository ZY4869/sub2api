package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"sort"
	"strings"
)

func openAIWSEventMayContainModel(eventType string) bool {
	switch eventType {
	case "response.created", "response.in_progress", "response.completed", "response.done", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
		return true
	default:
		trimmed := strings.TrimSpace(eventType)
		if trimmed == eventType {
			return false
		}
		switch trimmed {
		case "response.created", "response.in_progress", "response.completed", "response.done", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
			return true
		default:
			return false
		}
	}
}
func openAIWSEventMayContainToolCalls(eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return false
	}
	if strings.Contains(eventType, "function_call") || strings.Contains(eventType, "tool_call") {
		return true
	}
	switch eventType {
	case "response.output_item.added", "response.output_item.done", "response.completed", "response.done":
		return true
	default:
		return false
	}
}
func openAIWSEventShouldParseUsage(eventType string) bool {
	return eventType == "response.completed" || strings.TrimSpace(eventType) == "response.completed"
}
func parseOpenAIWSEventEnvelope(message []byte) (eventType string, responseID string, response gjson.Result) {
	if len(message) == 0 {
		return "", "", gjson.Result{}
	}
	values := gjson.GetManyBytes(message, "type", "response.id", "id", "response")
	eventType = strings.TrimSpace(values[0].String())
	if id := strings.TrimSpace(values[1].String()); id != "" {
		responseID = id
	} else {
		responseID = strings.TrimSpace(values[2].String())
	}
	return eventType, responseID, values[3]
}
func openAIWSMessageLikelyContainsToolCalls(message []byte) bool {
	if len(message) == 0 {
		return false
	}
	return bytes.Contains(message, []byte(`"tool_calls"`)) || bytes.Contains(message, []byte(`"tool_call"`)) || bytes.Contains(message, []byte(`"function_call"`))
}
func parseOpenAIWSResponseUsageFromCompletedEvent(message []byte, usage *OpenAIUsage) {
	if usage == nil || len(message) == 0 {
		return
	}
	values := gjson.GetManyBytes(message, "response.usage.input_tokens", "response.usage.output_tokens", "response.usage.input_tokens_details.cached_tokens")
	usage.InputTokens = int(values[0].Int())
	usage.OutputTokens = int(values[1].Int())
	usage.CacheReadInputTokens = int(values[2].Int())
}
func parseOpenAIWSErrorEventFields(message []byte) (code string, errType string, errMessage string) {
	if len(message) == 0 {
		return "", "", ""
	}
	values := gjson.GetManyBytes(message, "error.code", "error.type", "error.message")
	return strings.TrimSpace(values[0].String()), strings.TrimSpace(values[1].String()), strings.TrimSpace(values[2].String())
}
func summarizeOpenAIWSErrorEventFieldsFromRaw(codeRaw, errTypeRaw, errMessageRaw string) (code string, errType string, errMessage string) {
	code = truncateOpenAIWSLogValue(codeRaw, openAIWSLogValueMaxLen)
	errType = truncateOpenAIWSLogValue(errTypeRaw, openAIWSLogValueMaxLen)
	errMessage = truncateOpenAIWSLogValue(errMessageRaw, openAIWSLogValueMaxLen)
	return code, errType, errMessage
}
func summarizeOpenAIWSErrorEventFields(message []byte) (code string, errType string, errMessage string) {
	if len(message) == 0 {
		return "-", "-", "-"
	}
	return summarizeOpenAIWSErrorEventFieldsFromRaw(parseOpenAIWSErrorEventFields(message))
}
func summarizeOpenAIWSPayloadKeySizes(payload map[string]any, topN int) string {
	if len(payload) == 0 {
		return "-"
	}
	type keySize struct {
		Key  string
		Size int
	}
	sizes := make([]keySize, 0, len(payload))
	for key, value := range payload {
		size := estimateOpenAIWSPayloadValueSize(value, openAIWSPayloadSizeEstimateDepth)
		sizes = append(sizes, keySize{Key: key, Size: size})
	}
	sort.Slice(sizes, func(i, j int) bool {
		if sizes[i].Size == sizes[j].Size {
			return sizes[i].Key < sizes[j].Key
		}
		return sizes[i].Size > sizes[j].Size
	})
	if topN <= 0 || topN > len(sizes) {
		topN = len(sizes)
	}
	parts := make([]string, 0, topN)
	for idx := 0; idx < topN; idx++ {
		item := sizes[idx]
		parts = append(parts, fmt.Sprintf("%s:%d", item.Key, item.Size))
	}
	return strings.Join(parts, ",")
}
func estimateOpenAIWSPayloadValueSize(value any, depth int) int {
	if depth <= 0 {
		return -1
	}
	switch v := value.(type) {
	case nil:
		return 0
	case string:
		return len(v)
	case []byte:
		return len(v)
	case bool:
		return 1
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return 8
	case float32, float64:
		return 8
	case map[string]any:
		if len(v) == 0 {
			return 2
		}
		total := 2
		count := 0
		for key, item := range v {
			count++
			if count > openAIWSPayloadSizeEstimateMaxItems {
				return -1
			}
			itemSize := estimateOpenAIWSPayloadValueSize(item, depth-1)
			if itemSize < 0 {
				return -1
			}
			total += len(key) + itemSize + 3
			if total > openAIWSPayloadSizeEstimateMaxBytes {
				return -1
			}
		}
		return total
	case []any:
		if len(v) == 0 {
			return 2
		}
		total := 2
		limit := len(v)
		if limit > openAIWSPayloadSizeEstimateMaxItems {
			return -1
		}
		for i := 0; i < limit; i++ {
			itemSize := estimateOpenAIWSPayloadValueSize(v[i], depth-1)
			if itemSize < 0 {
				return -1
			}
			total += itemSize + 1
			if total > openAIWSPayloadSizeEstimateMaxBytes {
				return -1
			}
		}
		return total
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return -1
		}
		if len(raw) > openAIWSPayloadSizeEstimateMaxBytes {
			return -1
		}
		return len(raw)
	}
}
func openAIWSPayloadString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return ""
	}
}
func openAIWSPayloadStringFromRaw(payload []byte, key string) string {
	if len(payload) == 0 || strings.TrimSpace(key) == "" {
		return ""
	}
	return strings.TrimSpace(gjson.GetBytes(payload, key).String())
}
func openAIWSPayloadBoolFromRaw(payload []byte, key string, defaultValue bool) bool {
	if len(payload) == 0 || strings.TrimSpace(key) == "" {
		return defaultValue
	}
	value := gjson.GetBytes(payload, key)
	if !value.Exists() {
		return defaultValue
	}
	if value.Type != gjson.True && value.Type != gjson.False {
		return defaultValue
	}
	return value.Bool()
}
func openAIWSSessionHashesFromID(sessionID string) (string, string) {
	return deriveOpenAISessionHashes(sessionID)
}
func extractOpenAIWSImageURL(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if raw, ok := v["url"].(string); ok {
			return strings.TrimSpace(raw)
		}
	}
	return ""
}
func summarizeOpenAIWSInput(input any) string {
	items, ok := input.([]any)
	if !ok || len(items) == 0 {
		return "-"
	}
	itemCount := len(items)
	textChars := 0
	imageDataURLs := 0
	imageDataURLChars := 0
	imageRemoteURLs := 0
	handleContentItem := func(contentItem map[string]any) {
		contentType, _ := contentItem["type"].(string)
		switch strings.TrimSpace(contentType) {
		case "input_text", "output_text", "text":
			if text, ok := contentItem["text"].(string); ok {
				textChars += len(text)
			}
		case "input_image":
			imageURL := extractOpenAIWSImageURL(contentItem["image_url"])
			if imageURL == "" {
				return
			}
			if strings.HasPrefix(strings.ToLower(imageURL), "data:image/") {
				imageDataURLs++
				imageDataURLChars += len(imageURL)
				return
			}
			imageRemoteURLs++
		}
	}
	handleInputItem := func(inputItem map[string]any) {
		if content, ok := inputItem["content"].([]any); ok {
			for _, rawContent := range content {
				contentItem, ok := rawContent.(map[string]any)
				if !ok {
					continue
				}
				handleContentItem(contentItem)
			}
			return
		}
		itemType, _ := inputItem["type"].(string)
		switch strings.TrimSpace(itemType) {
		case "input_text", "output_text", "text":
			if text, ok := inputItem["text"].(string); ok {
				textChars += len(text)
			}
		case "input_image":
			imageURL := extractOpenAIWSImageURL(inputItem["image_url"])
			if imageURL == "" {
				return
			}
			if strings.HasPrefix(strings.ToLower(imageURL), "data:image/") {
				imageDataURLs++
				imageDataURLChars += len(imageURL)
				return
			}
			imageRemoteURLs++
		}
	}
	for _, rawItem := range items {
		inputItem, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		handleInputItem(inputItem)
	}
	return fmt.Sprintf("items=%d,text_chars=%d,image_data_urls=%d,image_data_url_chars=%d,image_remote_urls=%d", itemCount, textChars, imageDataURLs, imageDataURLChars, imageRemoteURLs)
}
func dropOpenAIWSPayloadKey(payload map[string]any, key string, removed *[]string) {
	if len(payload) == 0 || strings.TrimSpace(key) == "" {
		return
	}
	if _, exists := payload[key]; !exists {
		return
	}
	delete(payload, key)
	*removed = append(*removed, key)
}
func applyOpenAIWSRetryPayloadStrategy(payload map[string]any, attempt int) (strategy string, removedKeys []string) {
	if len(payload) == 0 {
		return "empty", nil
	}
	if attempt <= 1 {
		return "full", nil
	}
	removed := make([]string, 0, 2)
	if attempt >= 2 {
		dropOpenAIWSPayloadKey(payload, "include", &removed)
	}
	if len(removed) == 0 {
		return "full", nil
	}
	sort.Strings(removed)
	return "trim_optional_fields", removed
}
func payloadAsJSON(payload map[string]any) string {
	return string(payloadAsJSONBytes(payload))
}
func payloadAsJSONBytes(payload map[string]any) []byte {
	if len(payload) == 0 {
		return []byte("{}")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return []byte("{}")
	}
	return body
}
func isOpenAIWSTerminalEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "response.completed", "response.done", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
		return true
	default:
		return false
	}
}
func isOpenAIWSTokenEvent(eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return false
	}
	switch eventType {
	case "response.created", "response.in_progress", "response.output_item.added", "response.output_item.done":
		return false
	}
	if strings.Contains(eventType, ".delta") {
		return true
	}
	if strings.HasPrefix(eventType, "response.output_text") {
		return true
	}
	if strings.HasPrefix(eventType, "response.output") {
		return true
	}
	return eventType == "response.completed" || eventType == "response.done"
}
func replaceOpenAIWSMessageModel(message []byte, fromModel, toModel string) []byte {
	if len(message) == 0 {
		return message
	}
	if strings.TrimSpace(fromModel) == "" || strings.TrimSpace(toModel) == "" || fromModel == toModel {
		return message
	}
	if !bytes.Contains(message, []byte(`"model"`)) || !bytes.Contains(message, []byte(fromModel)) {
		return message
	}
	modelValues := gjson.GetManyBytes(message, "model", "response.model")
	replaceModel := modelValues[0].Exists() && modelValues[0].Str == fromModel
	replaceResponseModel := modelValues[1].Exists() && modelValues[1].Str == fromModel
	if !replaceModel && !replaceResponseModel {
		return message
	}
	updated := message
	if replaceModel {
		if next, err := sjson.SetBytes(updated, "model", toModel); err == nil {
			updated = next
		}
	}
	if replaceResponseModel {
		if next, err := sjson.SetBytes(updated, "response.model", toModel); err == nil {
			updated = next
		}
	}
	return updated
}
