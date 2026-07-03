package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrokImageAliasesNormalizeToPublicModels(t *testing.T) {
	require.Equal(t, GrokModelImagineFast, NormalizeGrokPublicModelID("grok-imagine-image-fast"))
	require.Equal(t, GrokModelImagine, NormalizeGrokPublicModelID("grok-imagine-image"))
	require.Equal(t, GrokModelImagineEdit, NormalizeGrokPublicModelID("grok-imagine-image-edit"))

	require.True(t, GrokIsImageModel(GrokModelImagineFast))
	require.True(t, GrokIsImageModel("grok-imagine-image-fast"))
	require.True(t, GrokIsImageEditModel("grok-imagine-image-edit"))
}

func TestGrokAPIKeyResolvedUpstreamModelUsesImageAliases(t *testing.T) {
	require.Equal(t, "grok-imagine-image-fast", GrokAPIKeyResolvedUpstreamModel(GrokModelImagineFast))
	require.Equal(t, "grok-imagine-image", GrokAPIKeyResolvedUpstreamModel(GrokModelImagine))
	require.Equal(t, "grok-imagine-image-edit", GrokAPIKeyResolvedUpstreamModel(GrokModelImagineEdit))
	require.Equal(t, "grok-imagine-image-fast", GrokAPIKeyResolvedUpstreamModel("grok-imagine-image-fast"))
}

func TestGrokDefaultVisibleModelsIncludeMediaModels(t *testing.T) {
	models := GrokVisibleModelIDsForAccount(&Account{
		Platform: PlatformGrok,
		Type:     AccountTypeSSO,
		Extra:    map[string]any{"grok_tier": GrokTierBasic},
	})

	require.Contains(t, models, GrokModelImagineFast)
	require.Contains(t, models, GrokModelImagine)
	require.Contains(t, models, GrokModelImagineEdit)
	require.Contains(t, models, GrokModelImagineVideo)
}
