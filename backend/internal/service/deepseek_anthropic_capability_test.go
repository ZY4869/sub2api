package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDeepSeekAnthropicMessagesBodyRejectsUnsupportedBlocks(t *testing.T) {
	unsupportedBlocks := []string{
		"image",
		"document",
		"search_result",
		"redacted_thinking",
		"server_tool_use",
		"web_search_tool_result",
		"code_execution_tool_result",
		"mcp_tool_use",
		"mcp_tool_result",
		"container_upload",
	}

	for _, blockType := range unsupportedBlocks {
		t.Run(blockType, func(t *testing.T) {
			body := []byte(`{"messages":[{"role":"user","content":[{"type":"` + blockType + `"}]}]}`)

			err := ValidateDeepSeekAnthropicMessagesBody(body)

			require.Error(t, err)
			require.Contains(t, err.Error(), blockType)
		})
	}
}

func TestValidateDeepSeekAnthropicMessagesBodyAllowsSupportedTextAndToolBlocks(t *testing.T) {
	body := []byte(`{
		"messages": [
			{"role": "assistant", "content": [{"type": "text", "text": "ok"}, {"type": "tool_use", "id": "toolu_1", "name": "lookup", "input": {}}]},
			{"role": "user", "content": [{"type": "tool_result", "tool_use_id": "toolu_1", "content": "done"}]}
		]
	}`)

	require.NoError(t, ValidateDeepSeekAnthropicMessagesBody(body))
}
