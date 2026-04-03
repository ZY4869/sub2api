package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildKiroClaudePayload_NormalizesDeprecatedClaudeAliases(t *testing.T) {
	payload, err := buildKiroClaudePayload(
		[]byte(`{"model":"claude-sonnet-4-5-20250929","messages":[{"role":"user","content":"hi"}]}`),
		"claude-sonnet-4-5-20250929",
		"",
		kiroPrimaryOrigin,
		nil,
	)
	require.NoError(t, err)

	body := string(payload)
	require.Contains(t, body, `"modelId":"claude-sonnet-4.5"`)
	require.NotContains(t, body, `"modelId":"claude-sonnet-4-5-20250929"`)
}
