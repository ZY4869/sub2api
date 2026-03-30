//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
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
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"data":[{"id":"claude-sonnet-4-20250514"}]}`,
	}
	svc := &GatewayService{
		accountRepo:               repo,
		accountModelImportService: NewAccountModelImportService(nil, nil, upstream, nil),
	}
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

	entry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformAnthropic, "claude-sonnet-4-20250514")
	require.NoError(t, err)
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

	entries, err := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformGemini)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "gemini-2.0-flash", entries[0].PublicID)
	require.Equal(t, DefaultVertexPublicModelAlias("gemini-2.0-flash"), entries[0].AliasID)
	require.Equal(t, "gemini-2.0-flash", entries[0].SourceID)

	entry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "gemini-2.0-flash")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "gemini-2.0-flash", entry.PublicID)
	require.Equal(t, DefaultVertexPublicModelAlias("gemini-2.0-flash"), entry.AliasID)
	require.Equal(t, "gemini-2.0-flash", entry.SourceID)
	_, aliasVisible, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, DefaultVertexPublicModelAlias("gemini-2.0-flash"))
	require.NoError(t, err)
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

	entries, err := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformGemini)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "gemini-2.0-flash", entries[0].PublicID)
	require.Equal(t, "friendly-flash", entries[0].AliasID)
	require.Equal(t, "gemini-2.0-flash", entries[0].SourceID)

	entry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "gemini-2.0-flash")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "gemini-2.0-flash", entry.PublicID)
	require.Equal(t, "friendly-flash", entry.AliasID)
	require.Equal(t, "gemini-2.0-flash", entry.SourceID)
	_, missing, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "gemini-3.1-pro-preview")
	require.NoError(t, err)
	require.False(t, missing)
}

func TestGatewayService_GetAPIKeyPublicModels_OpenAIUsesUpstreamProjection(t *testing.T) {
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          4,
				Name:        "openai-apikey",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":  "sk-test",
					"base_url": "https://openai.example.test",
					"model_mapping": map[string]any{
						"friendly-gpt": "gpt-4.1-mini",
						"hidden-gpt":   "gpt-5",
					},
				},
			},
		},
	}
	upstream := &accountModelImportHTTPUpstreamStub{
		body: `{"data":[{"id":"gpt-4.1-mini"},{"id":"gpt-4o"}]}`,
	}
	svc := &GatewayService{
		accountRepo:               repo,
		accountModelImportService: NewAccountModelImportService(nil, nil, upstream, nil),
	}
	apiKey := &APIKey{
		ID:               13,
		ModelDisplayMode: APIKeyModelDisplayModeAliasOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID:       23,
				ModelPatterns: []string{"friendly-*"},
				Group: &Group{
					ID:       23,
					Name:     "openai-group",
					Platform: PlatformOpenAI,
					Status:   StatusActive,
				},
			},
		},
	}

	entries, err := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformOpenAI)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "gpt-4.1-mini", entries[0].PublicID)
	require.Equal(t, "friendly-gpt", entries[0].AliasID)
	require.Equal(t, "gpt-4.1-mini", entries[0].SourceID)
	require.Equal(t, "Gpt 4.1 Mini", entries[0].DisplayName)
	require.Equal(t, "https://openai.example.test/v1/models", upstream.lastReq.URL.String())
}

func TestGatewayService_GetAPIKeyPublicModels_PropagatesUpstreamError(t *testing.T) {
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          5,
				Name:        "openai-apikey",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":  "sk-test",
					"base_url": "https://openai.example.test",
				},
			},
		},
	}
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusForbidden,
		body:       `{"error":"You have insufficient permissions for this operation. Missing scopes: api.model.read."}`,
	}
	svc := &GatewayService{
		accountRepo:               repo,
		accountModelImportService: NewAccountModelImportService(nil, nil, upstream, nil),
	}
	apiKey := &APIKey{
		ID:               14,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 24,
				Group: &Group{
					ID:       24,
					Name:     "openai-group",
					Platform: PlatformOpenAI,
					Status:   StatusActive,
				},
			},
		},
	}

	entries, err := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformOpenAI)
	require.Nil(t, entries)
	require.Error(t, err)

	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(http.StatusForbidden), appErr.Code)
	require.Equal(t, accountModelImportReasonKindOpenAIScopeInsufficient, appErr.Metadata["reason_kind"])
	require.Equal(t, accountModelImportHintKeyOpenAIModelRead, appErr.Metadata["hint_key"])
}
