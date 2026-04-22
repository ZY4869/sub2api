//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
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
					"api_key":  "anthropic-key",
					"base_url": "https://anthropic.example.test",
					"model_mapping": map[string]any{
						"friendly-sonnet": "claude-sonnet-4-20250514",
					},
				},
				Extra: map[string]any{
					"model_probe_snapshot": map[string]any{
						"models":       []string{"claude-sonnet-4-20250514"},
						"updated_at":   "2026-04-01T10:00:00Z",
						"source":       "manual_probe",
						"probe_source": "upstream",
					},
				},
			},
		},
	}
	svc := &GatewayService{
		accountRepo: repo,
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
	require.False(t, ok)
	require.Nil(t, entry)

	aliasEntry, aliasOK, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformAnthropic, "friendly-sonnet")
	require.NoError(t, err)
	require.True(t, aliasOK)
	require.Equal(t, "friendly-sonnet", aliasEntry.PublicID)
	require.Equal(t, "claude-sonnet-4-20250514", aliasEntry.SourceID)

	got := svc.ResolveAPIKeySelectionModel(context.Background(), apiKey, PlatformAnthropic, "claude-sonnet-4-20250514")
	require.Equal(t, "claude-sonnet-4-20250514", got)
}

func TestGatewayService_GetAPIKeyPublicModels_VertexExpressUsesDefaultAliasPrefix(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "gemini-2.0-flash",
		DisplayName: "Gemini 2.0 Flash",
		Provider:    PlatformGemini,
		Platforms:   []string{PlatformGemini},
		ExposedIn:   []string{"runtime", "whitelist"},
		UIPriority:  1,
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"gemini-2.0-flash"})
	require.NoError(t, err)

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
		accountRepo:          repo,
		modelRegistryService: registrySvc,
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
				ModelPatterns: []string{
					"gemini-2.0-flash",
				},
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
	require.Equal(t, "gemini-2.0-flash", entries[0].AliasID)
	require.Equal(t, "gemini-2.0-flash", entries[0].SourceID)

	entry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "gemini-2.0-flash")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "gemini-2.0-flash", entry.PublicID)
	require.Equal(t, "gemini-2.0-flash", entry.AliasID)
	require.Equal(t, "gemini-2.0-flash", entry.SourceID)
	aliasEntry, aliasVisible, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, DefaultVertexPublicModelAlias("gemini-2.0-flash"))
	require.NoError(t, err)
	require.False(t, aliasVisible)
	require.Nil(t, aliasEntry)
	require.Equal(t, DefaultVertexPublicModelAlias("gemini-2.0-flash"), svc.ResolveAPIKeySelectionModel(context.Background(), apiKey, PlatformGemini, DefaultVertexPublicModelAlias("gemini-2.0-flash")))

	snapshot := protocolruntime.Snapshot()
	require.GreaterOrEqual(t, snapshot.PublicModelProjectionBySource[apiKeyPublicModelsSourcePolicyProjection], int64(2))
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
	require.Len(t, entries, 2)
	require.ElementsMatch(t, []string{"friendly-flash", "friendly-pro"}, []string{entries[0].PublicID, entries[1].PublicID})
	entriesByID := make(map[string]APIKeyPublicModelEntry, len(entries))
	for _, candidate := range entries {
		entriesByID[candidate.PublicID] = candidate
	}
	require.Equal(t, "friendly-flash", entriesByID["friendly-flash"].AliasID)
	require.Equal(t, "gemini-2.0-flash", entriesByID["friendly-flash"].SourceID)

	entry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "gemini-2.0-flash")
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, entry)
	proEntry, proVisible, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "friendly-pro")
	require.NoError(t, err)
	require.True(t, proVisible)
	require.Equal(t, "gemini-3.1-pro-preview", proEntry.SourceID)
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
	require.Equal(t, "friendly-gpt", entries[0].PublicID)
	require.Equal(t, "friendly-gpt", entries[0].AliasID)
	require.Equal(t, "gpt-4.1-mini", entries[0].SourceID)
	require.Equal(t, "friendly-gpt", entries[0].DisplayName)
	require.Nil(t, upstream.lastReq)
}

func TestGatewayService_GetAPIKeyPublicModels_ExplicitAliasHidesSourceRowAndSourceLookup(t *testing.T) {
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          40,
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
					},
				},
				Extra: map[string]any{
					"model_probe_snapshot": map[string]any{
						"models":       []string{"gpt-4.1-mini"},
						"updated_at":   "2026-04-01T10:00:00Z",
						"source":       "manual_probe",
						"probe_source": "upstream",
					},
				},
			},
		},
	}
	svc := &GatewayService{accountRepo: repo}
	apiKey := &APIKey{
		ID:               140,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 240,
				Group: &Group{
					ID:       240,
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
	require.Equal(t, "friendly-gpt", entries[0].PublicID)
	require.Equal(t, "gpt-4.1-mini", entries[0].SourceID)

	sourceEntry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformOpenAI, "gpt-4.1-mini")
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, sourceEntry)

	aliasEntry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformOpenAI, "friendly-gpt")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "friendly-gpt", aliasEntry.PublicID)

	require.Equal(t, "gpt-4.1-mini", svc.ResolveAPIKeySelectionModel(context.Background(), apiKey, PlatformOpenAI, "gpt-4.1-mini"))
}

func TestGatewayService_GetAPIKeyPublicModels_OpenAIGroupIncludesProtocolGatewayAccounts(t *testing.T) {
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          41,
				Name:        "openai-gateway",
				Platform:    PlatformProtocolGateway,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":  "gateway-key",
					"base_url": "https://gateway.example.test",
					"model_mapping": map[string]any{
						"grok-auto": "grok-auto",
					},
				},
				Extra: map[string]any{
					"gateway_protocol": "openai",
					"model_probe_snapshot": map[string]any{
						"models":       []string{"grok-auto"},
						"updated_at":   "2026-04-01T10:00:00Z",
						"source":       "manual_probe",
						"probe_source": "protocol_gateway",
					},
				},
			},
		},
	}
	svc := &GatewayService{
		accountRepo: repo,
	}
	apiKey := &APIKey{
		ID:               141,
		ModelDisplayMode: APIKeyModelDisplayModeAliasOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 241,
				Group: &Group{
					ID:       241,
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
	require.Equal(t, "grok-auto", entries[0].PublicID)

	entry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformOpenAI, "grok-auto")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "grok-auto", entry.PublicID)
}

func TestGatewayService_GetAPIKeyPublicModels_LiveProbeFailureFallsBackToRegistry(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

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
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "registry-openai-fallback",
		DisplayName: "Registry OpenAI Fallback",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		ExposedIn:   []string{"runtime"},
		UIPriority:  1,
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"registry-openai-fallback"})
	require.NoError(t, err)
	svc := &GatewayService{
		accountRepo:               repo,
		accountModelImportService: NewAccountModelImportService(nil, nil, upstream, nil),
		modelRegistryService:      registrySvc,
	}
	apiKey := &APIKey{
		ID:               14,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 24,
				ModelPatterns: []string{
					"registry-openai-*",
				},
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
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "registry-openai-fallback", entries[0].PublicID)
	require.Nil(t, upstream.lastReq)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.PublicModelProjectionBySource[apiKeyPublicModelsSourcePolicyProjection])
}

func TestGatewayService_GetAPIKeyPublicModels_UsesSavedSnapshotBeforeLiveProbe(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "gpt-4.1-mini",
		DisplayName: "GPT-4.1 Mini",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		ExposedIn:   []string{"runtime", "whitelist"},
		UIPriority:  1,
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"gpt-4.1-mini"})
	require.NoError(t, err)

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          6,
				Name:        "openai-apikey",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":  "sk-test",
					"base_url": "https://openai.example.test",
				},
				Extra: map[string]any{
					"model_probe_snapshot": map[string]any{
						"models":       []string{"gpt-4.1-mini"},
						"updated_at":   "2026-04-01T10:00:00Z",
						"source":       "manual_probe",
						"probe_source": "upstream",
					},
				},
			},
		},
	}
	upstream := &accountModelImportHTTPUpstreamStub{
		statusCode: http.StatusForbidden,
		body:       `{"error":"should not be called"}`,
	}
	svc := &GatewayService{
		accountRepo:               repo,
		accountModelImportService: NewAccountModelImportService(nil, nil, upstream, nil),
		modelRegistryService:      registrySvc,
	}
	apiKey := &APIKey{
		ID:               15,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 25,
				ModelPatterns: []string{
					"gpt-4.1-mini",
				},
				Group: &Group{
					ID:       25,
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
	require.Nil(t, upstream.lastReq)
	require.Len(t, repo.updateExtraCalls, 0)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.PublicModelProjectionBySource[apiKeyPublicModelsSourcePolicyProjection])
}

func TestGatewayService_GetAPIKeyPublicModels_WithoutRestrictionsReturnsCompleteDefaultLibraryProjection(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	for _, entry := range []UpsertModelRegistryEntryInput{
		{
			ID:          "registry-openai-alpha",
			DisplayName: "Registry OpenAI Alpha",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime", "whitelist"},
			UIPriority:  1,
		},
		{
			ID:          "registry-openai-beta",
			DisplayName: "Registry OpenAI Beta",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime", "whitelist"},
			UIPriority:  2,
		},
	} {
		_, err := registrySvc.UpsertEntry(context.Background(), entry)
		require.NoError(t, err)
	}
	_, err := registrySvc.ActivateModels(context.Background(), []string{"registry-openai-alpha", "registry-openai-beta"})
	require.NoError(t, err)

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          60,
				Name:        "openai-apikey",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":  "sk-test",
					"base_url": "https://openai.example.test",
				},
				Extra: map[string]any{
					"model_probe_snapshot": map[string]any{
						"models":       []string{"gpt-4.1-mini", "gpt-4o"},
						"updated_at":   "2026-04-01T10:00:00Z",
						"source":       "manual_probe",
						"probe_source": "upstream",
					},
				},
			},
		},
	}
	svc := &GatewayService{
		accountRepo:          repo,
		modelRegistryService: registrySvc,
	}
	apiKey := &APIKey{
		ID:               160,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 260,
				ModelPatterns: []string{
					"registry-openai-*",
				},
				Group: &Group{
					ID:       260,
					Name:     "openai-group",
					Platform: PlatformOpenAI,
					Status:   StatusActive,
				},
			},
		},
	}

	entries, err := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformOpenAI)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.ElementsMatch(t, []string{"registry-openai-alpha", "registry-openai-beta"}, []string{entries[0].PublicID, entries[1].PublicID})
}

func TestGatewayService_GetAPIKeyPublicModels_ReadPathDoesNotBackfillSnapshot(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	for _, entry := range []UpsertModelRegistryEntryInput{
		{
			ID:          "registry-openai-alpha",
			DisplayName: "Registry OpenAI Alpha",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime", "whitelist"},
			UIPriority:  1,
		},
		{
			ID:          "registry-openai-beta",
			DisplayName: "Registry OpenAI Beta",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime", "whitelist"},
			UIPriority:  2,
		},
	} {
		_, err := registrySvc.UpsertEntry(context.Background(), entry)
		require.NoError(t, err)
	}
	_, err := registrySvc.ActivateModels(context.Background(), []string{"registry-openai-alpha", "registry-openai-beta"})
	require.NoError(t, err)

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          7,
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
		body: `{"data":[{"id":"gpt-4.1-mini"},{"id":"gpt-4o"}]}`,
	}
	svc := &GatewayService{
		accountRepo:               repo,
		accountModelImportService: NewAccountModelImportService(nil, nil, upstream, nil),
		modelRegistryService:      registrySvc,
	}
	apiKey := &APIKey{
		ID:               16,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 26,
				ModelPatterns: []string{
					"registry-openai-*",
				},
				Group: &Group{
					ID:       26,
					Name:     "openai-group",
					Platform: PlatformOpenAI,
					Status:   StatusActive,
				},
			},
		},
	}

	entries, err := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformOpenAI)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.ElementsMatch(t, []string{"registry-openai-alpha", "registry-openai-beta"}, []string{entries[0].PublicID, entries[1].PublicID})
	require.Nil(t, upstream.lastReq)
	require.Len(t, repo.updateExtraCalls, 0)
}

func TestGatewayService_GetAPIKeyPublicModels_RestrictedProjectionStillRespectsScopeAndChannel(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	for _, entry := range []UpsertModelRegistryEntryInput{
		{
			ID:          "registry-openai-alpha",
			DisplayName: "Registry OpenAI Alpha",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime"},
			UIPriority:  1,
		},
		{
			ID:          "registry-openai-beta",
			DisplayName: "Registry OpenAI Beta",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime"},
			UIPriority:  2,
		},
	} {
		_, err := registrySvc.UpsertEntry(context.Background(), entry)
		require.NoError(t, err)
	}
	_, err := registrySvc.ActivateModels(context.Background(), []string{"registry-openai-alpha", "registry-openai-beta"})
	require.NoError(t, err)

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          8,
				Name:        "openai-scoped",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":  "sk-test",
					"base_url": "https://openai.example.test",
				},
				Extra: map[string]any{
					"model_scope_v2": map[string]any{
						"supported_models_by_provider": map[string]any{
							PlatformOpenAI: []any{"registry-openai-beta"},
						},
					},
				},
			},
		},
	}
	svc := &GatewayService{
		accountRepo:          repo,
		modelRegistryService: registrySvc,
		channelService: &ChannelService{repo: &apiKeyPublicModelsChannelRepoStub{
			channel: &model.Channel{
				ID:             1,
				Name:           "restricted",
				Status:         model.ChannelStatusActive,
				RestrictModels: true,
				ModelPricing: []model.ChannelModelPricing{
					{
						ID:          1,
						Platform:    PlatformOpenAI,
						Models:      []string{"registry-openai-beta"},
						BillingMode: model.ChannelBillingModeToken,
					},
				},
			},
		}},
	}
	apiKey := &APIKey{
		ID:               18,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 28,
				ModelPatterns: []string{
					"registry-openai-*",
				},
				Group: &Group{
					ID:       28,
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
	require.Equal(t, "registry-openai-beta", entries[0].PublicID)
	require.NotEmpty(t, entries[0].DisplayName)

	snapshot := protocolruntime.Snapshot()
	require.GreaterOrEqual(t, snapshot.PublicModelProjectionBySource[apiKeyPublicModelsSourcePolicyProjection], int64(1))
}

func TestGatewayService_GetAPIKeyPublicModels_ManualModelsDoNotExpandProjection(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	for _, entry := range []UpsertModelRegistryEntryInput{
		{
			ID:          "registry-openai-alpha",
			DisplayName: "Registry OpenAI Alpha",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime"},
			UIPriority:  1,
		},
		{
			ID:          "registry-openai-beta",
			DisplayName: "Registry OpenAI Beta",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime"},
			UIPriority:  2,
		},
	} {
		_, err := registrySvc.UpsertEntry(context.Background(), entry)
		require.NoError(t, err)
	}
	_, err := registrySvc.ActivateModels(context.Background(), []string{"registry-openai-alpha", "registry-openai-beta"})
	require.NoError(t, err)

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          9,
				Name:        "openai-manual-and-mapping",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":  "sk-test",
					"base_url": "https://openai.example.test",
					"model_mapping": map[string]any{
						"friendly-beta": "registry-openai-beta",
					},
				},
				Extra: map[string]any{
					"manual_models": []any{
						map[string]any{"model_id": "registry-openai-alpha"},
					},
				},
			},
		},
	}
	svc := &GatewayService{
		accountRepo:          repo,
		modelRegistryService: registrySvc,
	}
	apiKey := &APIKey{
		ID:               19,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 29,
				ModelPatterns: []string{
					"registry-openai-*",
				},
				Group: &Group{
					ID:       29,
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
	require.Equal(t, "friendly-beta", entries[0].PublicID)
}

func TestGatewayService_GetAPIKeyPublicModels_RestrictedScopePrefersAliasForWhitelistedTarget(t *testing.T) {
	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	for _, entry := range []UpsertModelRegistryEntryInput{
		{
			ID:          "registry-openai-beta",
			DisplayName: "Registry OpenAI Beta",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime"},
			UIPriority:  1,
		},
		{
			ID:          "registry-openai-gamma",
			DisplayName: "Registry OpenAI Gamma",
			Provider:    PlatformOpenAI,
			Platforms:   []string{PlatformOpenAI},
			ExposedIn:   []string{"runtime"},
			UIPriority:  2,
		},
	} {
		_, err := registrySvc.UpsertEntry(context.Background(), entry)
		require.NoError(t, err)
	}
	_, err := registrySvc.ActivateModels(context.Background(), []string{"registry-openai-beta", "registry-openai-gamma"})
	require.NoError(t, err)

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          61,
				Name:        "openai-scoped-alias",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":  "sk-test",
					"base_url": "https://openai.example.test",
					"model_mapping": map[string]any{
						"friendly-beta": "registry-openai-beta",
					},
				},
				Extra: map[string]any{
					"model_scope_v2": map[string]any{
						"supported_models_by_provider": map[string]any{
							PlatformOpenAI: []any{"registry-openai-beta"},
						},
					},
				},
			},
		},
	}
	svc := &GatewayService{
		accountRepo:          repo,
		modelRegistryService: registrySvc,
	}
	apiKey := &APIKey{
		ID:               161,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 261,
				ModelPatterns: []string{
					"registry-openai-*",
				},
				Group: &Group{
					ID:       261,
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
	require.Equal(t, "friendly-beta", entries[0].PublicID)
	require.Equal(t, "registry-openai-beta", entries[0].SourceID)

	sourceEntry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformOpenAI, "registry-openai-beta")
	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, sourceEntry)

	aliasEntry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformOpenAI, "friendly-beta")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "friendly-beta", aliasEntry.PublicID)
}

func TestGatewayService_GetAPIKeyPublicModels_OpenAIProRuntimeQuotaHidesLimitedSide(t *testing.T) {
	groupID := int64(2701)
	group := &Group{
		ID:       groupID,
		Name:     "openai-pro-runtime-hide",
		Platform: PlatformOpenAI,
		Status:   StatusActive,
	}
	apiKey := &APIKey{
		ID:               2701,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: groupID,
				Group:   group,
			},
		},
	}

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          2702,
				Name:        "openai-pro-blocked-normal",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					"model_scope_v2": map[string]any{
						"policy_mode": AccountModelPolicyModeWhitelist,
						"entries": []any{
							map[string]any{
								"display_model_id": "gpt-5.4",
								"target_model_id":  "gpt-5.4",
								"provider":         PlatformOpenAI,
								"visibility_mode":  AccountModelVisibilityModeDirect,
							},
							map[string]any{
								"display_model_id": "friendly-spark",
								"target_model_id":  "gpt-5.3-codex-spark-high",
								"provider":         PlatformOpenAI,
								"visibility_mode":  AccountModelVisibilityModeAlias,
							},
						},
					},
					modelRateLimitsKey: map[string]any{
						openAICodexScopeNormal: newModelRateLimitEntry(time.Now().Add(10 * time.Minute)),
					},
				},
				AccountGroups: []AccountGroup{{AccountID: 2702, GroupID: groupID}},
			},
		},
	}

	svc := &GatewayService{accountRepo: repo}
	entries, err := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformOpenAI)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "friendly-spark", entries[0].PublicID)

	hiddenEntry, hiddenOK, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformOpenAI, "gpt-5.4")
	require.NoError(t, err)
	require.False(t, hiddenOK)
	require.Nil(t, hiddenEntry)

	sparkEntry, sparkOK, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformOpenAI, "friendly-spark")
	require.NoError(t, err)
	require.True(t, sparkOK)
	require.Equal(t, "friendly-spark", sparkEntry.PublicID)
}

func TestGatewayService_GetAPIKeyPublicModels_OpenAIProRuntimeQuotaKeepsModelWhenAnotherAccountCanServe(t *testing.T) {
	groupID := int64(2801)
	group := &Group{
		ID:       groupID,
		Name:     "openai-pro-runtime-multi",
		Platform: PlatformOpenAI,
		Status:   StatusActive,
	}
	apiKey := &APIKey{
		ID:               2801,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: groupID,
				Group:   group,
			},
		},
	}

	baseExtra := map[string]any{
		"model_scope_v2": map[string]any{
			"policy_mode": AccountModelPolicyModeWhitelist,
			"entries": []any{
				map[string]any{
					"display_model_id": "gpt-5.4",
					"target_model_id":  "gpt-5.4",
					"provider":         PlatformOpenAI,
					"visibility_mode":  AccountModelVisibilityModeDirect,
				},
			},
		},
	}
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          2802,
				Name:        "openai-pro-blocked-normal",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					"model_scope_v2": baseExtra["model_scope_v2"],
					modelRateLimitsKey: map[string]any{
						openAICodexScopeNormal: newModelRateLimitEntry(time.Now().Add(10 * time.Minute)),
					},
				},
				AccountGroups: []AccountGroup{{AccountID: 2802, GroupID: groupID}},
			},
			{
				ID:          2803,
				Name:        "openai-pro-available",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra:         baseExtra,
				AccountGroups: []AccountGroup{{AccountID: 2803, GroupID: groupID}},
			},
		},
	}

	svc := &GatewayService{accountRepo: repo}
	entries, err := svc.GetAPIKeyPublicModels(context.Background(), apiKey, PlatformOpenAI)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "gpt-5.4", entries[0].PublicID)

	entry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformOpenAI, "gpt-5.4")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "gpt-5.4", entry.PublicID)
}

func TestGatewayService_FindAPIKeyPublicModel_VertexCatalogFailureFallsBackToRegistry(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	registrySvc := NewModelRegistryService(newAccountModelImportSettingRepoStub())
	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "gemini-2.0-flash",
		DisplayName: "Gemini 2.0 Flash",
		Provider:    PlatformGemini,
		Platforms:   []string{PlatformGemini},
		ExposedIn:   []string{"runtime"},
		UIPriority:  1,
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"gemini-2.0-flash"})
	require.NoError(t, err)

	vertexProvider := newTestVertexCatalogProvider(nil)
	vertexProvider.err = errors.New(`status 403 PERMISSION_DENIED: missing scope api.model.read`)

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          30,
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
		accountRepo:          repo,
		modelRegistryService: registrySvc,
		vertexCatalogService: vertexProvider,
	}
	apiKey := &APIKey{
		ID:               30,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 30,
				Group: &Group{
					ID:       30,
					Name:     "gemini-group",
					Platform: PlatformGemini,
					Status:   StatusActive,
				},
			},
		},
	}

	entry, ok, err := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformGemini, "gemini-2.0-flash")
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, entry)
	require.Equal(t, "gemini-2.0-flash", entry.PublicID)

	account, unique, err := svc.ResolveGeminiPublicModelMetadataAccount(context.Background(), apiKey, PlatformGemini, "gemini-2.0-flash")
	require.NoError(t, err)
	require.True(t, unique)
	require.NotNil(t, account)
	require.Equal(t, int64(30), account.ID)

	snapshot := protocolruntime.Snapshot()
	require.GreaterOrEqual(t, snapshot.PublicModelProjectionBySource[apiKeyPublicModelsSourcePolicyProjection], int64(1))
}

type apiKeyPublicModelsChannelRepoStub struct {
	channel *model.Channel
}

func (s *apiKeyPublicModelsChannelRepoStub) List(_ context.Context, _ pagination.PaginationParams, _ ChannelListFilters) ([]*model.Channel, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *apiKeyPublicModelsChannelRepoStub) GetByID(_ context.Context, _ int64) (*model.Channel, error) {
	return s.channel, nil
}

func (s *apiKeyPublicModelsChannelRepoStub) GetActiveByGroupID(_ context.Context, _ int64) (*model.Channel, error) {
	return s.channel, nil
}

func (s *apiKeyPublicModelsChannelRepoStub) Create(_ context.Context, channel *model.Channel) (*model.Channel, error) {
	s.channel = channel
	return channel, nil
}

func (s *apiKeyPublicModelsChannelRepoStub) Update(_ context.Context, channel *model.Channel) (*model.Channel, error) {
	s.channel = channel
	return channel, nil
}

func (s *apiKeyPublicModelsChannelRepoStub) Delete(_ context.Context, _ int64) error {
	s.channel = nil
	return nil
}
