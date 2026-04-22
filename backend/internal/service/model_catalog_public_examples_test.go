package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectPublicModelCatalogExampleSpec_UsesResponsesForImageGenerationTool(t *testing.T) {
	spec := selectPublicModelCatalogExampleSpec(PublicModelCatalogItem{
		Model:            "gpt-5.4-mini",
		Provider:         PlatformOpenAI,
		RequestProtocols: []string{PlatformOpenAI},
		Mode:             "chat",
	}, "image_generation_tool")

	require.Equal(t, "image-generation-tool", spec.OverrideID)
	require.Equal(t, "openai-native", spec.PageID)
	require.Equal(t, PlatformOpenAI, spec.Protocol)
}

func TestSelectPublicModelCatalogExampleSpec_UsesNativeImageExampleForGeminiImageModel(t *testing.T) {
	spec := selectPublicModelCatalogExampleSpec(PublicModelCatalogItem{
		Model:            "gemini-2.5-flash-image",
		Provider:         PlatformGemini,
		RequestProtocols: []string{PlatformGemini},
		Mode:             "image",
	}, "image_generation")

	require.Equal(t, "image-generation", spec.OverrideID)
	require.Equal(t, PlatformGemini, spec.Protocol)
}
