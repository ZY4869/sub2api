package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type availableModelsAdminService struct {
	*stubAdminService
	accounts map[int64]service.Account
}

func (s *availableModelsAdminService) GetAccount(_ context.Context, id int64) (*service.Account, error) {
	if account, ok := s.accounts[id]; ok {
		copy := account
		return &copy, nil
	}
	return s.stubAdminService.GetAccount(context.Background(), id)
}

func setupAvailableModelsRouter(adminSvc service.AdminService, registrySvc *service.ModelRegistryService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.SetModelRegistryService(registrySvc)
	router.GET("/api/v1/admin/accounts/:id/models", handler.GetAvailableModels)
	return router
}

func decodeAvailableModelsResponse(t *testing.T, rec *httptest.ResponseRecorder) []service.AvailableTestModel {
	t.Helper()

	var resp struct {
		Data []service.AvailableTestModel `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp.Data
}

func TestAccountHandlerGetAvailableModels_AppliesAccountLevelMappingsAndScopes(t *testing.T) {
	repo := service.NewModelRegistryService(newTestSettingRepo())
	_, err := repo.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:          "custom-shared-b",
		DisplayName: "Custom Shared B",
		Platforms:   []string{"custom-tests"},
		UIPriority:  2,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = repo.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:          "custom-shared-a",
		DisplayName: "Custom Shared A",
		Platforms:   []string{"custom-tests"},
		UIPriority:  1,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = repo.ActivateModels(context.Background(), []string{"custom-shared-a", "custom-shared-b"})
	require.NoError(t, err)

	adminSvc := &availableModelsAdminService{
		stubAdminService: newStubAdminService(),
		accounts: map[int64]service.Account{
			42: {
				ID:       42,
				Name:     "mapping-account",
				Platform: "custom-tests",
				Type:     service.AccountTypeOAuth,
				Status:   service.StatusActive,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"only-this-model": "custom-shared-a",
					},
				},
			},
			43: {
				ID:       43,
				Name:     "scope-account",
				Platform: "custom-tests",
				Type:     service.AccountTypeSetupToken,
				Status:   service.StatusActive,
				Extra: map[string]any{
					"model_scope_v2": map[string]any{
						"supported_models_by_provider": map[string]any{
							"custom-tests": []any{"custom-shared-b"},
						},
					},
				},
			},
		},
	}
	router := setupAvailableModelsRouter(adminSvc, repo)

	recA := httptest.NewRecorder()
	reqA := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/42/models", nil)
	router.ServeHTTP(recA, reqA)
	require.Equal(t, http.StatusOK, recA.Code)

	recB := httptest.NewRecorder()
	reqB := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/43/models", nil)
	router.ServeHTTP(recB, reqB)
	require.Equal(t, http.StatusOK, recB.Code)

	modelsA := decodeAvailableModelsResponse(t, recA)
	modelsB := decodeAvailableModelsResponse(t, recB)
	require.Len(t, modelsA, 1)
	require.Len(t, modelsB, 1)
	require.Equal(t, "only-this-model", modelsA[0].ID)
	require.Equal(t, "custom-shared-a", modelsA[0].TargetModelID)
	require.Equal(t, "custom-shared-b", modelsB[0].ID)
}

func TestAccountHandlerGetAvailableModels_DedupesCanonicalModelsAndSortsDeprecatedLast(t *testing.T) {
	registrySvc := service.NewModelRegistryService(newTestSettingRepo())
	_, err := registrySvc.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:          "family-stable",
		DisplayName: "Family Stable",
		Platforms:   []string{"custom-dedupe"},
		UIPriority:  1,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:          "family-old",
		DisplayName: "Family Old",
		Platforms:   []string{"custom-dedupe"},
		UIPriority:  2,
		ExposedIn:   []string{"test"},
		Status:      "deprecated",
		ReplacedBy:  "family-stable",
	})
	require.NoError(t, err)
	_, err = registrySvc.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:           "legacy-only",
		DisplayName:  "Legacy Only",
		Platforms:    []string{"custom-dedupe"},
		UIPriority:   3,
		ExposedIn:    []string{"test"},
		Status:       "deprecated",
		DeprecatedAt: "2026-01-01T00:00:00Z",
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"family-stable", "family-old", "legacy-only"})
	require.NoError(t, err)

	adminSvc := &availableModelsAdminService{
		stubAdminService: newStubAdminService(),
		accounts: map[int64]service.Account{
			44: {
				ID:       44,
				Name:     "dedupe-account",
				Platform: "custom-dedupe",
				Type:     service.AccountTypeOAuth,
				Status:   service.StatusActive,
			},
		},
	}
	router := setupAvailableModelsRouter(adminSvc, registrySvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/44/models", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	models := decodeAvailableModelsResponse(t, rec)
	require.Len(t, models, 2)
	require.Equal(t, "family-stable", models[0].ID)
	require.Equal(t, "family-stable", models[0].CanonicalID)
	require.Equal(t, "legacy-only", models[1].ID)
	require.Equal(t, "deprecated", models[1].Status)
}

func TestAccountHandlerGetAvailableModels_KiroFallsBackToBuiltinCatalog(t *testing.T) {
	registrySvc := service.NewModelRegistryService(newTestSettingRepo())
	account := service.Account{
		ID:       45,
		Name:     "kiro-oauth",
		Platform: service.PlatformKiro,
		Type:     service.AccountTypeOAuth,
		Status:   service.StatusActive,
	}
	adminSvc := &availableModelsAdminService{
		stubAdminService: newStubAdminService(),
		accounts: map[int64]service.Account{
			45: account,
		},
	}
	router := setupAvailableModelsRouter(adminSvc, registrySvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/45/models", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	models := decodeAvailableModelsResponse(t, rec)
	require.NotEmpty(t, models)
	require.Equal(t, service.BuildAvailableTestModels(context.Background(), &account, registrySvc), models)
}

func TestAccountHandlerGetAvailableModels_ReadPathUsesProjectedPolicyInsteadOfProbeSnapshot(t *testing.T) {
	registrySvc := service.NewModelRegistryService(newTestSettingRepo())
	_, err := registrySvc.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:          "saved-snapshot-model",
		DisplayName: "Saved Snapshot Model",
		Platforms:   []string{service.PlatformOpenAI},
		UIPriority:  1,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:          "live-registry-model",
		DisplayName: "Live Registry Model",
		Platforms:   []string{service.PlatformOpenAI},
		UIPriority:  2,
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"saved-snapshot-model", "live-registry-model"})
	require.NoError(t, err)

	adminSvc := &availableModelsAdminService{
		stubAdminService: newStubAdminService(),
		accounts: map[int64]service.Account{
			46: {
				ID:       46,
				Name:     "snapshot-account",
				Platform: service.PlatformOpenAI,
				Type:     service.AccountTypeAPIKey,
				Status:   service.StatusActive,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"friendly-live": "live-registry-model",
					},
				},
				Extra: map[string]any{
					"model_probe_snapshot": map[string]any{
						"models": []any{"saved-snapshot-model"},
					},
				},
			},
		},
	}
	router := setupAvailableModelsRouter(adminSvc, registrySvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/46/models", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	models := decodeAvailableModelsResponse(t, rec)
	require.Len(t, models, 1)
	require.Equal(t, "friendly-live", models[0].ID)
	require.Equal(t, "live-registry-model", models[0].TargetModelID)

	refreshRec := httptest.NewRecorder()
	refreshReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/46/models?refresh=true", nil)
	router.ServeHTTP(refreshRec, refreshReq)
	require.Equal(t, http.StatusOK, refreshRec.Code)

	refreshedModels := decodeAvailableModelsResponse(t, refreshRec)
	require.Len(t, refreshedModels, 1)
	require.Equal(t, "friendly-live", refreshedModels[0].ID)
	require.Equal(t, "live-registry-model", refreshedModels[0].TargetModelID)
}

func TestAccountHandlerGetAvailableModels_UsesSnapshotRegistryMetadata(t *testing.T) {
	registrySvc := service.NewModelRegistryService(newTestSettingRepo())
	_, err := registrySvc.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:           "snapshot-image-model",
		DisplayName:  "Snapshot Image Model",
		Platforms:    []string{service.PlatformOpenAI},
		Provider:     service.PlatformOpenAI,
		Modalities:   []string{"image"},
		Capabilities: []string{"image_generation"},
		UIPriority:   1,
		ExposedIn:    []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"snapshot-image-model"})
	require.NoError(t, err)

	adminSvc := &availableModelsAdminService{
		stubAdminService: newStubAdminService(),
		accounts: map[int64]service.Account{
			47: {
				ID:       47,
				Name:     "snapshot-metadata-account",
				Platform: service.PlatformOpenAI,
				Type:     service.AccountTypeAPIKey,
				Status:   service.StatusActive,
				Extra: map[string]any{
					"model_scope_v2": map[string]any{
						"policy_mode": "whitelist",
						"entries": []any{
							map[string]any{
								"display_model_id": "snapshot-image-model",
								"target_model_id":  "snapshot-image-model",
								"provider":         service.PlatformOpenAI,
								"visibility_mode":  service.AccountModelVisibilityModeDirect,
							},
						},
					},
					"model_probe_snapshot": map[string]any{
						"models": []any{"snapshot-image-model"},
					},
				},
			},
		},
	}
	router := setupAvailableModelsRouter(adminSvc, registrySvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/47/models", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	models := decodeAvailableModelsResponse(t, rec)
	require.Len(t, models, 1)
	require.Equal(t, "snapshot-image-model", models[0].ID)
	require.Equal(t, strings.ToLower(service.FormatModelCatalogDisplayName("snapshot-image-model")), strings.ToLower(models[0].DisplayName))
	require.Equal(t, "image", models[0].Mode)
	require.Equal(t, "verified", models[0].AvailabilityState)
}

func TestAccountHandlerGetAvailableModels_OpenAIProRuntimeQuotaHidesLimitedSide(t *testing.T) {
	registrySvc := service.NewModelRegistryService(newTestSettingRepo())
	adminSvc := &availableModelsAdminService{
		stubAdminService: newStubAdminService(),
		accounts: map[int64]service.Account{
			48: {
				ID:       48,
				Name:     "openai-pro-runtime-hide",
				Platform: service.PlatformOpenAI,
				Type:     service.AccountTypeOAuth,
				Status:   service.StatusActive,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					"model_scope_v2": map[string]any{
						"policy_mode": service.AccountModelPolicyModeWhitelist,
						"entries": []any{
							map[string]any{
								"display_model_id": "friendly-normal",
								"target_model_id":  "gpt-5.4",
								"provider":         service.PlatformOpenAI,
								"visibility_mode":  service.AccountModelVisibilityModeAlias,
							},
							map[string]any{
								"display_model_id": "friendly-spark",
								"target_model_id":  "gpt-5.3-codex-spark-high",
								"provider":         service.PlatformOpenAI,
								"visibility_mode":  service.AccountModelVisibilityModeAlias,
							},
						},
					},
					"model_rate_limits": map[string]any{
						"gpt-5.3-codex": map[string]any{
							"rate_limited_at":     time.Now().UTC().Format(time.RFC3339),
							"rate_limit_reset_at": time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339),
						},
					},
				},
			},
		},
	}
	router := setupAvailableModelsRouter(adminSvc, registrySvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/48/models", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	models := decodeAvailableModelsResponse(t, rec)
	require.Len(t, models, 1)
	require.Equal(t, "friendly-spark", models[0].ID)
	require.Equal(t, "gpt-5.3-codex-spark-high", models[0].TargetModelID)
}
