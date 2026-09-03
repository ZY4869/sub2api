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

func TestGrokBuildTextModelIDsDefaultOrder(t *testing.T) {
	models := GrokBuildTextModelIDs()

	require.Equal(t, DefaultGrokBuildTextModelID(), models[0])
	require.Equal(t, []string{
		GrokModelBuild46,
		GrokModelBuild45,
		GrokModelBuild43,
		GrokModelBuild01,
		GrokModelComposer25Fast,
		GrokModel420Reasoning,
		GrokModel420NonReasoning,
		GrokModel420MultiAgent,
	}, models)
}

func TestGrokLatestAliasesResolveToBuild46(t *testing.T) {
	require.Equal(t, GrokModelBuild46, DefaultGrokBuildTextModelID())
	require.Equal(t, GrokModelBuild46, NormalizeGrokPublicModelID("grok"))
	require.Equal(t, GrokModelBuild46, NormalizeGrokPublicModelID("grok-latest"))
	require.Equal(t, GrokModelBuild46, NormalizeGrokPublicModelID("grok-build-latest"))
	require.Equal(t, GrokModelBuild45, NormalizeGrokPublicModelID("grok-4.5-latest"))
}

func TestGrokSSOVisibleModelsDoNotUseBuildTextCatalog(t *testing.T) {
	models := GrokVisibleModelIDsForAccount(&Account{
		Platform: PlatformGrok,
		Type:     AccountTypeSSO,
		Extra:    map[string]any{"grok_tier": GrokTierBasic},
	})

	require.NotContains(t, models, GrokModelBuild45)
	require.Contains(t, models, GrokModelAuto)
}
