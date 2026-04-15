package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type geminiPublicModelsSettingRepoStub struct {
	values map[string]string
}

func newGeminiPublicModelsSettingRepoStub() *geminiPublicModelsSettingRepoStub {
	return &geminiPublicModelsSettingRepoStub{values: map[string]string{}}
}

func (s *geminiPublicModelsSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	value, err := s.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (s *geminiPublicModelsSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *geminiPublicModelsSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *geminiPublicModelsSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *geminiPublicModelsSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *geminiPublicModelsSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *geminiPublicModelsSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type geminiPublicModelsAccountRepoStub struct {
	service.AccountRepository
	accounts []service.Account
}

func (s *geminiPublicModelsAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, platforms []string) ([]service.Account, error) {
	platformSet := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		platformSet[platform] = struct{}{}
	}

	result := make([]service.Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		if !account.IsSchedulable() {
			continue
		}
		if _, ok := platformSet[account.Platform]; !ok {
			continue
		}
		if len(account.AccountGroups) > 0 {
			matchedGroup := false
			for _, binding := range account.AccountGroups {
				if binding.GroupID == groupID {
					matchedGroup = true
					break
				}
			}
			if !matchedGroup {
				continue
			}
		}
		result = append(result, account)
	}
	return result, nil
}

type geminiPublicModelsVertexCatalogStub struct {
	err error
}

func (s *geminiPublicModelsVertexCatalogStub) GetCatalog(_ context.Context, _ *service.Account, _ bool) (*service.VertexCatalogResult, error) {
	return nil, s.err
}

func newGeminiPublicModelsFallbackHandler(t *testing.T) (*GatewayHandler, *service.APIKey) {
	t.Helper()

	registryRepo := newGeminiPublicModelsSettingRepoStub()
	registrySvc := service.NewModelRegistryService(registryRepo)
	_, err := registrySvc.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:          "gemini-2.0-flash",
		DisplayName: "Gemini 2.0 Flash",
		Provider:    service.PlatformGemini,
		Platforms:   []string{service.PlatformGemini},
		ExposedIn:   []string{"runtime"},
		UIPriority:  1,
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"gemini-2.0-flash"})
	require.NoError(t, err)

	groupID := int64(3001)
	group := &service.Group{
		ID:       groupID,
		Name:     "gemini-public-fallback",
		Platform: service.PlatformGemini,
		Status:   service.StatusActive,
	}
	apiKey := &service.APIKey{
		ID:               4001,
		UserID:           5001,
		Key:              "sk-gemini-public-fallback",
		Status:           service.StatusActive,
		ModelDisplayMode: service.APIKeyModelDisplayModeSourceOnly,
		GroupID:          &groupID,
		Group:            group,
		GroupBindings: []service.APIKeyGroupBinding{
			{
				APIKeyID: 4001,
				GroupID:  groupID,
				Group:    group,
			},
		},
		User: &service.User{ID: 5001, Status: service.StatusActive},
	}

	accountRepo := &geminiPublicModelsAccountRepoStub{
		accounts: []service.Account{
			{
				ID:          6001,
				Name:        "vertex-express",
				Platform:    service.PlatformGemini,
				Type:        service.AccountTypeAPIKey,
				Status:      service.StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"api_key":            "vertex-express-key",
					"gemini_api_variant": service.GeminiAPIKeyVariantVertexExpress,
				},
				AccountGroups: []service.AccountGroup{{AccountID: 6001, GroupID: groupID}},
			},
		},
	}

	gatewaySvc := service.NewGatewayService(
		accountRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	gatewaySvc.SetModelRegistryService(registrySvc)
	gatewaySvc.SetVertexCatalogService(&geminiPublicModelsVertexCatalogStub{
		err: errors.New(`status 403 PERMISSION_DENIED: missing scope api.model.read`),
	})

	handler := &GatewayHandler{
		gatewayService: gatewaySvc,
	}
	handler.SetModelRegistryService(registrySvc)
	return handler, apiKey
}

func newGeminiPublicModelsContext(method, path string, apiKey *service.APIKey, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, path, nil)
	c.Params = params
	c.Set(string(servermiddleware.ContextKeyAPIKey), apiKey)
	return c, recorder
}

func TestGeminiV1BetaListModels_VertexCatalogFailureFallsBackToRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, apiKey := newGeminiPublicModelsFallbackHandler(t)
	c, recorder := newGeminiPublicModelsContext(http.MethodGet, "/v1beta/models", apiKey, nil)

	handler.GeminiV1BetaListModels(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var payload gemini.ModelsListResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.NotEmpty(t, payload.Models)

	var target *gemini.Model
	for i := range payload.Models {
		if payload.Models[i].Name == "models/gemini-2.0-flash" {
			target = &payload.Models[i]
			break
		}
	}
	require.NotNil(t, target)
	require.NotEmpty(t, target.DisplayName)
	require.True(t, strings.Contains(target.Description, "fallback entry"))
	require.Contains(t, target.SupportedGenerationMethods, "generateContent")
	require.Empty(t, payload.NextPageToken)
}

func TestGeminiV1BetaGetModel_VertexCatalogFailureFallsBackToRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, apiKey := newGeminiPublicModelsFallbackHandler(t)
	c, recorder := newGeminiPublicModelsContext(
		http.MethodGet,
		"/v1beta/models/gemini-2.0-flash",
		apiKey,
		gin.Params{{Key: "model", Value: "gemini-2.0-flash"}},
	)

	handler.GeminiV1BetaGetModel(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var payload gemini.Model
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "models/gemini-2.0-flash", payload.Name)
	require.NotEmpty(t, payload.DisplayName)
	require.Contains(t, strings.ToLower(payload.DisplayName), "gemini")
	require.Contains(t, strings.ToLower(payload.DisplayName), "flash")
	require.True(t, strings.Contains(payload.Description, "fallback entry"))
	require.Contains(t, payload.SupportedGenerationMethods, "generateContent")
}
