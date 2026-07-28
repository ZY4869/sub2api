package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponsesAnthropicCompat_AdditionalToolsLiftRestoreDeduplicatesNamespace_Default(t *testing.T) {
	reqBody := map[string]any{
		"tools": []any{
			map[string]any{"type": "namespace", "name": "collaboration", "tools": []any{
				map[string]any{"type": "function", "name": "spawn_agent"},
			}},
		},
		"input": []any{
			map[string]any{"type": "message", "role": "user", "content": "hello"},
			map[string]any{"type": "additional_tools", "role": "developer", "tools": []any{
				map[string]any{"type": "namespace", "name": "collaboration", "tools": []any{
					map[string]any{"type": "function", "name": "spawn_agent"},
				}},
			}},
		},
	}

	changed, err := normalizeOpenAIResponsesLiteTools(reqBody)

	require.NoError(t, err)
	require.True(t, changed)
	require.NotContains(t, reqBody, "tools")

	input, ok := reqBody["input"].([]any)
	require.True(t, ok)
	require.Len(t, input, 2)
	additionalToolsMessage, ok := input[1].(map[string]any)
	require.True(t, ok)
	additional, ok := additionalToolsMessage["tools"].([]any)
	require.True(t, ok)
	require.Len(t, additional, 1)
	namespace, ok := additional[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "collaboration", namespace["name"])
}
