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
			name:     "opus 4.6 stays independent",
			input:    "claude-opus-4-6",
			expected: "claude-opus-4-6",
		},
		{
			name:     "sonnet 4.6 stays independent",
			input:    "claude-sonnet-4-6",
			expected: "claude-sonnet-4-6",
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

func TestResolveToPricingIDUsesIntendedSharedOrExactTargets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "gemini 3 flash keeps shared preview pricing id",
			input:    "gemini-3-flash",
			expected: "gemini-3-flash-preview",
		},
		{
			name:     "gemini 3.1 flash image keeps its own pricing id",
			input:    "gemini-3.1-flash-image",
			expected: "gemini-3.1-flash-image",
		},
		{
			name:     "gpt 5.4 pro dated snapshot shares base pricing id",
			input:    "gpt-5.4-pro-2026-03-05",
			expected: "gpt-5.4-pro",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, ok := ResolveToPricingID(test.input)
			require.True(t, ok)
			require.Equal(t, test.expected, actual)
		})
	}
}

func TestExplainSeedResolutionReportsDeprecatedReplacement(t *testing.T) {
	resolution, ok := ExplainSeedResolution("claude-haiku-4-5-20251001")
	require.True(t, ok)
	require.NotNil(t, resolution)
	require.Equal(t, "claude-haiku-4-5-20251001", resolution.CanonicalID)
	require.Equal(t, "claude-haiku-4.5", resolution.EffectiveID)
	require.True(t, resolution.Deprecated)
	require.NotNil(t, resolution.ReplacementEntry)
	require.Equal(t, "claude-haiku-4.5", resolution.ReplacementEntry.ID)
}
