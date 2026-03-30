//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayService_ResolveAPIKeySelectionModel_SourceOnlyUsesAlias(t *testing.T) {
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          1,
				Name:        "anthropic-apikey",
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"friendly-sonnet": "claude-sonnet-4-20250514",
					},
				},
			},
		},
	}
	svc := &GatewayService{accountRepo: repo}
	apiKey := &APIKey{
		ID:               10,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 20,
				Group: &Group{
					ID:       20,
					Name:     "anthropic-group",
					Platform: PlatformAnthropic,
					Status:   StatusActive,
				},
			},
		},
	}

	entry, ok := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformAnthropic, "claude-sonnet-4-20250514")
	require.True(t, ok)
	require.Equal(t, "friendly-sonnet", entry.AliasID)
	require.Equal(t, "claude-sonnet-4-20250514", entry.PublicID)
	require.Equal(t, "claude-sonnet-4-20250514", entry.SourceID)

	got := svc.ResolveAPIKeySelectionModel(context.Background(), apiKey, PlatformAnthropic, "claude-sonnet-4-20250514")
	require.Equal(t, "friendly-sonnet", got)
}

func TestGatewayService_GetAPIKeyPublicModels_VertexExpressUsesDefaultAliasPrefix(t *testing.T) {
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          2,
				Name:        "vertex-express",
				Platform:    PlatformGemini,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":            "vertex-express-key",
					"gemini_api_variant": GeminiAPIKeyVariantVertexExpress,
				},
			},
		},
	}
	svc := &GatewayService{
		accountRepo: repo,
		vertexCatalogService: newTestVertexCatalogProvider(&VertexCatalogResult{
			CallableUnion: []VertexCatalogModel{
				{ID: "gemini-2.0-flash", DisplayName: "Gemini 2.0 Flash"},
			},
		}),
	}
	apiKey := &APIKey{
		ID:               11,
		ModelDisplayMode: APIKeyModelDisplayModeAliasOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 21,
				Group: &Group{
					ID:       21,
					Name:     "gemini-group",
					Platform: PlatformGemini,
					Status:   StatusActive,
				},
			},
		},
	}

	entries := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformGemini)
	require.Len(t, entries, 1)
	require.Equal(t, "gemini-2.0-flash", entries[0].PublicID)
	require.Equal(t, DefaultVertexPublicModelAlias("gemini-2.0-flash"), entries[0].AliasID)
	require.Equal(t, "gemini-2.0-flash", entries[0].SourceID)

	entry, ok := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "gemini-2.0-flash")
	require.True(t, ok)
	require.Equal(t, "gemini-2.0-flash", entry.PublicID)
	require.Equal(t, DefaultVertexPublicModelAlias("gemini-2.0-flash"), entry.AliasID)
	require.Equal(t, "gemini-2.0-flash", entry.SourceID)
	_, aliasVisible := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, DefaultVertexPublicModelAlias("gemini-2.0-flash"))
	require.False(t, aliasVisible)
	require.Equal(
		t,
		DefaultVertexPublicModelAlias("gemini-2.0-flash"),
		svc.ResolveAPIKeySelectionModel(context.Background(), apiKey, PlatformGemini, DefaultVertexPublicModelAlias("gemini-2.0-flash")),
	)
}

func TestGatewayService_GetAPIKeyPublicModels_VertexExpressSourceOnlyHidesVertexPrefix(t *testing.T) {
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          3,
				Name:        "vertex-express",
				Platform:    PlatformGemini,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":            "vertex-express-key",
					"gemini_api_variant": GeminiAPIKeyVariantVertexExpress,
					"model_mapping": map[string]any{
						"friendly-flash": "gemini-2.0-flash",
						"friendly-pro":   "gemini-3.1-pro-preview",
					},
				},
			},
		},
	}
	svc := &GatewayService{
		accountRepo: repo,
		vertexCatalogService: newTestVertexCatalogProvider(&VertexCatalogResult{
			CallableUnion: []VertexCatalogModel{
				{ID: "gemini-2.0-flash", DisplayName: "Gemini 2.0 Flash"},
			},
		}),
	}
	apiKey := &APIKey{
		ID:               12,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 22,
				Group: &Group{
					ID:       22,
					Name:     "gemini-group",
					Platform: PlatformGemini,
					Status:   StatusActive,
				},
			},
		},
	}

	entries := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformGemini)
	require.Len(t, entries, 1)
	require.Equal(t, "gemini-2.0-flash", entries[0].PublicID)
	require.Equal(t, "friendly-flash", entries[0].AliasID)
	require.Equal(t, "gemini-2.0-flash", entries[0].SourceID)

	entry, ok := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "gemini-2.0-flash")
	require.True(t, ok)
	require.Equal(t, "gemini-2.0-flash", entry.PublicID)
	require.Equal(t, "friendly-flash", entry.AliasID)
	require.Equal(t, "gemini-2.0-flash", entry.SourceID)
	_, missing := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "gemini-3.1-pro-preview")
	require.False(t, missing)
}
