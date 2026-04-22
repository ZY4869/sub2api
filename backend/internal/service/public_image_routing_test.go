//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayService_ResolvePublicImageRoute_OpenAINativeImageModel(t *testing.T) {
	svc, apiKey := newPublicImageRoutingGatewayServiceForTest(t, []publicImageRoutingProviderConfig{
		{platform: PlatformOpenAI, groupID: 101, models: []string{"gpt-image-2"}},
	}, "gpt-image-2")

	decision, err := svc.ResolvePublicImageRoute(context.Background(), apiKey, EndpointImagesGen, "gpt-image-2")
	require.NoError(t, err)
	require.True(t, decision.Supported)
	require.Equal(t, PlatformOpenAI, decision.ResolvedProvider)
	require.Equal(t, "gpt-image-2", decision.TargetModelID)
	require.Equal(t, EndpointImagesGen, decision.UpstreamEndpoint)
	require.Equal(t, PublicImageRouteReasonModelProvider, decision.RouteReason)
}

func TestGatewayService_ResolvePublicImageRoute_ToolOnlyModelRejectsNativeImages(t *testing.T) {
	svc, apiKey := newPublicImageRoutingGatewayServiceForTest(t, []publicImageRoutingProviderConfig{
		{platform: PlatformOpenAI, groupID: 102, models: []string{"gpt-5.4-mini"}},
	}, "gpt-5.4-mini")

	decision, err := svc.ResolvePublicImageRoute(context.Background(), apiKey, EndpointImagesGen, "gpt-5.4-mini")
	require.NoError(t, err)
	require.False(t, decision.Supported)
	require.Equal(t, PublicImageRouteReasonToolOnlyModel, decision.RouteReason)
	require.Contains(t, decision.ErrorMessage, "/v1/responses")
}

func TestGatewayService_ResolvePublicImageRoute_GeminiEditsRejected(t *testing.T) {
	svc, apiKey := newPublicImageRoutingGatewayServiceForTest(t, []publicImageRoutingProviderConfig{
		{platform: PlatformGemini, groupID: 103, models: []string{"gemini-2.5-flash-image"}},
	}, "gemini-2.5-flash-image")

	decision, err := svc.ResolvePublicImageRoute(context.Background(), apiKey, EndpointImagesEdits, "gemini-2.5-flash-image")
	require.NoError(t, err)
	require.False(t, decision.Supported)
	require.Equal(t, PlatformGemini, decision.ResolvedProvider)
	require.Equal(t, GatewayReasonUnsupportedAction, decision.ErrorCode)
	require.Equal(t, PublicImageRouteReasonUnsupported, decision.RouteReason)
}

func TestGatewayService_ResolvePublicImageRoute_SingleProviderFallbackWithoutModel(t *testing.T) {
	svc, apiKey := newPublicImageRoutingGatewayServiceForTest(t, []publicImageRoutingProviderConfig{
		{platform: PlatformOpenAI, groupID: 104, models: []string{"gpt-image-2"}},
	}, "gpt-image-2")

	decision, err := svc.ResolvePublicImageRoute(context.Background(), apiKey, EndpointImagesGen, "")
	require.NoError(t, err)
	require.True(t, decision.Supported)
	require.Equal(t, PlatformOpenAI, decision.ResolvedProvider)
	require.Equal(t, EndpointImagesGen, decision.UpstreamEndpoint)
	require.Equal(t, PublicImageRouteReasonSingleProvider, decision.RouteReason)
}

type publicImageRoutingProviderConfig struct {
	platform string
	groupID  int64
	models   []string
}

func newPublicImageRoutingGatewayServiceForTest(
	t *testing.T,
	providers []publicImageRoutingProviderConfig,
	activateModels ...string,
) (*GatewayService, *APIKey) {
	t.Helper()

	repo := newAccountModelImportSettingRepoStub()
	registrySvc := NewModelRegistryService(repo)
	if len(activateModels) > 0 {
		_, err := registrySvc.ActivateModels(context.Background(), activateModels)
		require.NoError(t, err)
	}

	apiKey := &APIKey{
		ID:               9001,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings:    make([]APIKeyGroupBinding, 0, len(providers)),
	}
	accounts := make([]Account, 0, len(providers))
	for index, provider := range providers {
		group := &Group{
			ID:       provider.groupID,
			Name:     provider.platform + "-group",
			Platform: provider.platform,
			Status:   StatusActive,
		}
		apiKey.GroupBindings = append(apiKey.GroupBindings, APIKeyGroupBinding{
			GroupID: provider.groupID,
			Group:   group,
		})

		supportedModels := make([]any, 0, len(provider.models))
		for _, modelID := range provider.models {
			supportedModels = append(supportedModels, modelID)
		}
		accountID := int64(7000 + index + 1)
		accounts = append(accounts, Account{
			ID:          accountID,
			Name:        provider.platform + "-account",
			Platform:    provider.platform,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"api_key":  "sk-test",
				"base_url": "https://example.test",
			},
			Extra: map[string]any{
				"model_scope_v2": map[string]any{
					"supported_models_by_provider": map[string]any{
						normalizePublicImageProvider(provider.platform): supportedModels,
					},
				},
			},
			AccountGroups: []AccountGroup{{AccountID: accountID, GroupID: provider.groupID}},
		})
	}

	return &GatewayService{
		accountRepo:          &mockAccountRepoForPlatform{accounts: accounts},
		modelRegistryService: registrySvc,
	}, apiKey
}
