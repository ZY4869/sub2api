package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectPublicModelCatalogExampleSpec_UsesResponsesForImageGenerationTool(t *testing.T) {
	spec, ok := selectPublicModelCatalogExampleSpec(PublicModelCatalogItem{
		Model:            "gpt-5.4-mini",
		Provider:         PlatformOpenAI,
		RequestProtocols: []string{PlatformOpenAI},
		Mode:             "chat",
	}, "image_generation_tool")

	require.True(t, ok)
	require.Equal(t, "image-generation-tool", spec.OverrideID)
	require.Equal(t, "openai-native", spec.PageID)
	require.Equal(t, PlatformOpenAI, spec.Protocol)
}

func TestSelectPublicModelCatalogExampleSpec_UsesNativeImageExampleForGeminiImageModel(t *testing.T) {
	spec, ok := selectPublicModelCatalogExampleSpec(PublicModelCatalogItem{
		Model:            "gemini-2.5-flash-image",
		Provider:         PlatformGemini,
		RequestProtocols: []string{PlatformGemini},
		Mode:             "image",
	}, "image_generation")

	require.True(t, ok)
	require.Equal(t, "image-generation", spec.OverrideID)
	require.Equal(t, PlatformGemini, spec.Protocol)
}

func TestSelectPublicModelCatalogExampleSpec_UsesEmbeddingsOverride(t *testing.T) {
	spec, ok := selectPublicModelCatalogExampleSpec(PublicModelCatalogItem{
		Model:            "text-embedding-3-small",
		Provider:         PlatformOpenAI,
		RequestProtocols: []string{PlatformOpenAI},
		Mode:             "embedding",
	}, "")

	require.True(t, ok)
	require.Equal(t, "embeddings", spec.OverrideID)
	require.Equal(t, PlatformOpenAI, spec.Protocol)
	require.Equal(t, "openai.embeddings", spec.EndpointKey)
}

func TestSelectPublicModelCatalogExampleSpec_ReturnsFalseWithoutSupportedEndpoint(t *testing.T) {
	_, ok := selectPublicModelCatalogExampleSpec(PublicModelCatalogItem{
		Model: "blocked-model",
		ProtocolEndpoints: []PublicModelProtocolEndpoint{{
			Key:      "openai.responses",
			Protocol: PlatformOpenAI,
			Support:  PublicModelSupportUnsupported,
		}},
	}, "")

	require.False(t, ok)
}

func TestSelectPublicModelCatalogExampleSpec_RejectsEndpointWhenCapabilityUnsupported(t *testing.T) {
	_, ok := selectPublicModelCatalogExampleSpec(PublicModelCatalogItem{
		Model: "gpt-5.4-image",
		Mode:  "image",
		ProtocolEndpoints: []PublicModelProtocolEndpoint{{
			Key:      "openai.images.generations",
			Protocol: PlatformOpenAI,
			Support:  PublicModelSupportSupported,
		}},
		CapabilityMatrix: []PublicModelCapabilityMatrixEntry{{
			Capability: "image_generation",
			Protocol:   PlatformOpenAI,
			Endpoint:   "openai.images.generations",
			Support:    PublicModelSupportUnsupported,
			Source:     PublicModelCapabilitySourceRuntimeObserved,
			Verified:   true,
		}},
	}, "image_generation")

	require.False(t, ok)
}
