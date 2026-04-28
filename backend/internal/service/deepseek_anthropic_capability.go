package service

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

var deepSeekUnsupportedAnthropicContentBlocks = map[string]struct{}{
	"code_execution_tool_result": {},
	"container_upload":           {},
	"document":                   {},
	"image":                      {},
	"mcp_tool_result":            {},
	"mcp_tool_use":               {},
	"redacted_thinking":          {},
	"search_result":              {},
	"server_tool_use":            {},
	"web_search_tool_result":     {},
}

func ValidateDeepSeekAnthropicMessagesBody(body []byte) error {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return nil
	}

	firstUnsupported := ""
	gjson.GetBytes(body, "messages").ForEach(func(_, message gjson.Result) bool {
		content := message.Get("content")
		if !content.IsArray() {
			return true
		}
		content.ForEach(func(_, block gjson.Result) bool {
			blockType := strings.ToLower(strings.TrimSpace(block.Get("type").String()))
			if _, unsupported := deepSeekUnsupportedAnthropicContentBlocks[blockType]; unsupported {
				firstUnsupported = blockType
				return false
			}
			return true
		})
		return firstUnsupported == ""
	})
	if firstUnsupported == "" {
		return nil
	}
	return fmt.Errorf("DeepSeek Anthropic API does not support %q content blocks", firstUnsupported)
}
