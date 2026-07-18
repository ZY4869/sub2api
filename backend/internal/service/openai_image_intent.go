package service

import (
	"strings"

	"github.com/tidwall/gjson"
)

// IsExplicitOpenAIResponsesImageIntent detects requests that explicitly need
// native Responses image-generation support. Passive image_gen namespaces are
// intentionally ignored because Codex may advertise them on ordinary text turns.
func IsExplicitOpenAIResponsesImageIntent(requestedModel string, body []byte) bool {
	if IsOpenAINativeImageModelID(requestedModel) {
		return true
	}
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return false
	}
	if IsOpenAINativeImageModelID(gjson.GetBytes(body, "model").String()) {
		return true
	}
	if _, ok := DetectOpenAIResponsesImageGenerationToolModel(body); ok {
		return true
	}
	return openAIResponsesToolChoiceSelectsImageGeneration(gjson.GetBytes(body, "tool_choice"))
}

func openAIResponsesToolChoiceSelectsImageGeneration(choice gjson.Result) bool {
	if !choice.Exists() {
		return false
	}
	if choice.Type == gjson.String {
		return strings.TrimSpace(choice.String()) == "image_generation"
	}
	if !choice.IsObject() {
		return false
	}
	if strings.TrimSpace(choice.Get("type").String()) == "image_generation" {
		return true
	}
	if tool := choice.Get("tool"); tool.IsObject() {
		return openAIResponsesToolChoiceSelectsImageGeneration(tool)
	}
	if fn := choice.Get("function"); fn.IsObject() {
		return strings.TrimSpace(fn.Get("name").String()) == "image_generation"
	}
	return false
}
