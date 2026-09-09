package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayEffortResolutionDefaultSuite(t *testing.T) {
	t.Run("openai max becomes xhigh", func(t *testing.T) {
		resolution := ResolveOpenAIEffort("max", "", effortSourceOpenAIAlias)

		require.NotNil(t, resolution.Raw)
		require.NotNil(t, resolution.Effective)
		require.Equal(t, "max", *resolution.Raw)
		require.Equal(t, "xhigh", *resolution.Effective)
		require.Equal(t, effortSourceOpenAIAlias, resolution.Source)
	})

	t.Run("openai gpt 5.6 max remains max", func(t *testing.T) {
		resolution := ResolveOpenAIEffortForModels("max", "", effortSourceOpenAIAlias, "gpt-5.6-sol")

		require.NotNil(t, resolution.Raw)
		require.NotNil(t, resolution.Effective)
		require.Equal(t, "max", *resolution.Raw)
		require.Equal(t, "max", *resolution.Effective)
		require.Equal(t, effortSourceOpenAIAlias, resolution.Source)
	})

	t.Run("gpt 6 astra does not receive unsupported none reasoning", func(t *testing.T) {
		resolution := ResolveOpenAIEffortForModels("none", "", effortSourceOpenAIAlias, "gpt-6-astra")

		require.NotNil(t, resolution.Raw)
		require.NotNil(t, resolution.Effective)
		require.Equal(t, "none", *resolution.Raw)
		require.Equal(t, "low", *resolution.Effective)
	})

	t.Run("anthropic output_config wins over top-level fallback", func(t *testing.T) {
		resolution := ResolveAnthropicEffort("high", "max")

		require.NotNil(t, resolution.Raw)
		require.NotNil(t, resolution.Effective)
		require.Equal(t, "high", *resolution.Raw)
		require.Equal(t, "high", *resolution.Effective)
		require.Equal(t, effortSourceAnthropicField, resolution.Source)
	})

	t.Run("gemini reasoning effort max maps to high thinking", func(t *testing.T) {
		resolution := ResolveGeminiEffort("", "", "", "max", "")

		require.NotNil(t, resolution.Raw)
		require.NotNil(t, resolution.Effective)
		require.Equal(t, "max", *resolution.Raw)
		require.Equal(t, "HIGH", *resolution.Effective)
		require.Equal(t, effortSourceOpenAIAlias, resolution.Source)
	})
}

func TestRectifyThinkingBudgetKeepsAdaptiveRequestsUnchanged(t *testing.T) {
	body := []byte(`{"model":"minimax-m1","thinking":{"type":"adaptive"},"max_tokens":1}`)

	got, changed := RectifyThinkingBudget(body)

	require.False(t, changed)
	require.JSONEq(t, string(body), string(got))
}
