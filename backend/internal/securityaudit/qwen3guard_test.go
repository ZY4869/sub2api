package securityaudit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQwen3GuardNormalizesTwoLineResponse(t *testing.T) {
	result, err := ParseQwen3Guard("Safety: Unsafe\nCategories: Jailbreak, PII", AllScannerIDs)

	require.NoError(t, err)
	require.Equal(t, EventCritical, result.Decision)
	require.Equal(t, ActionBlock, result.Action)
	require.Equal(t, []string{"pii", "jailbreak"}, result.MatchedScanners)
	require.Equal(t, []string{"pii", "jailbreak"}, result.Categories)
}

func TestParseQwen3GuardKeepsUnknownCategories(t *testing.T) {
	result, err := ParseQwen3Guard("Categories: cosmic badness\nSafety: Unsafe", []string{"jailbreak"})

	require.NoError(t, err)
	require.Equal(t, EventCritical, result.Decision)
	require.Empty(t, result.MatchedScanners)
	require.Len(t, result.UnknownCategories, 1)
	require.True(t, strings.HasPrefix(result.UnknownCategories[0], "unknown:"))
}

func TestParseQwen3GuardRejectsInvalidEnvelope(t *testing.T) {
	_, err := ParseQwen3Guard("Safety: Maybe\nCategories: none", AllScannerIDs)

	require.Error(t, err)
	require.Equal(t, ErrorCodeInvalidResponse, guardErrorCode(err))
}

func TestNormalizeBaseURLRemovesTrailingV1AndRejectsUnsafeParts(t *testing.T) {
	normalized, err := NormalizeBaseURL("HTTPS://guard.example.com/v1/")
	require.NoError(t, err)
	require.Equal(t, "https://guard.example.com", normalized)

	chatURL, err := ChatCompletionsURL("https://guard.example.com/v1")
	require.NoError(t, err)
	require.Equal(t, "https://guard.example.com/v1/chat/completions", chatURL)

	for _, raw := range []string{
		"https://token@guard.example.com",
		"https://guard.example.com?token=x",
		"https://guard.example.com/#secret",
		"file:///tmp/guard",
	} {
		_, err := NormalizeBaseURL(raw)
		require.Error(t, err, raw)
	}
}
