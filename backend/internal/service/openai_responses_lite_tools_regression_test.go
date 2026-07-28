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
	additional := reqBody["input"].([]any)[1].(map[string]any)["tools"].([]any)
	require.Len(t, additional, 1)
	require.Equal(t, "collaboration", additional[0].(map[string]any)["name"])
}
