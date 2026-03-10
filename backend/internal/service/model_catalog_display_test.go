package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatModelCatalogDisplayNameAndIconKey(t *testing.T) {
	require.Equal(t, "Claude-3-5-haiku", FormatModelCatalogDisplayName("claude-3-5-haiku-20241022"))
	require.Equal(t, "GPT-4o-mini", FormatModelCatalogDisplayName("gpt-4o-mini-2026-03-05"))
	require.Equal(t, "Gemini-2.5-pro-preview", FormatModelCatalogDisplayName("gemini-2.5-pro-preview-2026-03-05"))

	require.Equal(t, "claude", InferModelCatalogIconKey("claude-3-5-haiku-20241022"))
	require.Equal(t, "chatgpt", InferModelCatalogIconKey("gpt-4o-mini"))
	require.Equal(t, "chatgpt", InferModelCatalogIconKey("o3-mini"))
	require.Equal(t, "gemini", InferModelCatalogIconKey("gemini-2.5-flash"))
}
