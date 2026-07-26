package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestEnrichOpsTraceRequestThinkingConfig_DisabledNotMarkedAsThinking(t *testing.T) {
	result := &service.ProtocolNormalizeResult{}
	enrichOpsTraceRequestThinkingConfig(map[string]any{"thinking": map[string]any{"type": "disabled"}}, result)

	require.False(t, result.HasThinking, "显式关闭思考的请求不应打 Thinking 标记")
	require.Equal(t, "compat_thinking", result.ThinkingSource, "来源仍应记录便于排查")
	require.Equal(t, "DISABLED", result.ThinkingLevel)
}

func TestEnrichOpsTraceRequestThinkingConfig_EnabledMarked(t *testing.T) {
	result := &service.ProtocolNormalizeResult{}
	enrichOpsTraceRequestThinkingConfig(map[string]any{"thinking": map[string]any{"type": "enabled", "budget_tokens": 1024}}, result)

	require.True(t, result.HasThinking)
	require.Equal(t, "ENABLED", result.ThinkingLevel)
	require.NotNil(t, result.ThinkingBudget)
	require.Equal(t, 1024, *result.ThinkingBudget)
}
