package modelregistry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveToCanonicalIDVariants(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "canonical sonnet",
			input:    "claude-sonnet-4.5",
			expected: "claude-sonnet-4.5",
		},
		{
			name:     "dated sonnet protocol",
			input:    "claude-sonnet-4-5-20250929",
			expected: "claude-sonnet-4.5",
		},
		{
			name:     "dated sonnet dotted alias",
			input:    "claude-sonnet-4.5-20250929",
			expected: "claude-sonnet-4.5",
		},
		{
			name:     "gemini models prefix",
			input:    "models/gemini-2.5-pro",
			expected: "gemini-2.5-pro",
		},
		{
			name:     "gemini publishers path",
			input:    "/publishers/google/models/gemini-2.5-pro",
			expected: "gemini-2.5-pro",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, ok := ResolveToCanonicalID(test.input)
			require.True(t, ok)
			require.Equal(t, test.expected, actual)
		})
	}
}

func TestResolveToProtocolIDRouteSpecific(t *testing.T) {
	oauthModel, ok := ResolveToProtocolID("claude-sonnet-4.5", "anthropic_oauth")
	require.True(t, ok)
	require.Equal(t, "claude-sonnet-4-5-20250929", oauthModel)

	apiKeyModel, ok := ResolveToProtocolID("claude-sonnet-4.5", "anthropic_apikey")
	require.True(t, ok)
	require.Equal(t, "claude-sonnet-4.5", apiKeyModel)

	geminiModel, ok := ResolveToProtocolID("models/gemini-2.5-pro", "gemini")
	require.True(t, ok)
	require.Equal(t, "gemini-2.5-pro", geminiModel)
}

func TestExplainSeedResolutionReportsDeprecatedReplacement(t *testing.T) {
	resolution, ok := ExplainSeedResolution("gpt-5.3-codex")
	require.True(t, ok)
	require.NotNil(t, resolution)
	require.Equal(t, "gpt-5.3-codex", resolution.CanonicalID)
	require.Equal(t, "gpt-5-codex", resolution.EffectiveID)
	require.True(t, resolution.Deprecated)
	require.NotNil(t, resolution.ReplacementEntry)
	require.Equal(t, "gpt-5-codex", resolution.ReplacementEntry.ID)
}
