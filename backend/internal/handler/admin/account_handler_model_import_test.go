package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDefaultAvailableModels_PrefersTestExposure(t *testing.T) {
	h := &AccountHandler{
		modelRegistryService: service.NewModelRegistryService(newTestSettingRepo()),
	}

	models := h.defaultAvailableModels(context.Background(), &service.Account{Platform: service.PlatformOpenAI})
	ids := make([]string, 0, len(models))
	for _, model := range models {
		ids = append(ids, model.ID)
	}

	require.Contains(t, ids, "gpt-5.4")
	require.NotContains(t, ids, "gpt-5-codex")
}

type handlerModelImportSettingRepoStub struct {
	values map[string]string
}

func (s *handlerModelImportSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	value, ok := s.values[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (s *handlerModelImportSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *handlerModelImportSettingRepoStub) Set(ctx context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *handlerModelImportSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *handlerModelImportSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *handlerModelImportSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *handlerModelImportSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

type handlerModelImportHTTPUpstream struct{}

func (u *handlerModelImportHTTPUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	return nil, errors.New("unexpected Do call")
}

func (u *handlerModelImportHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, tlsProfile *service.TLSFingerprintProfile) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       ioNopCloser(`{"data":[{"id":"gpt-5.4"},{"id":"gpt-4.1-mini"}]}`),
		Request:    req,
	}, nil
}

func ioNopCloser(body string) *stringReadCloser {
	return &stringReadCloser{Reader: strings.NewReader(body)}
}

type stringReadCloser struct {
	*strings.Reader
}

func (r *stringReadCloser) Close() error { return nil }

func TestImportModels_OpenAIOAuthUpdatesKnownModelsSnapshot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:       901,
			Name:     "openai-oauth",
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
			Credentials: map[string]any{
				"access_token": "test-token",
			},
			Extra: map[string]any{
				"existing": "value",
			},
		},
	}

	repo := &handlerModelImportSettingRepoStub{values: make(map[string]string)}
	modelRegistryService := service.NewModelRegistryService(repo)
	importSvc := service.NewAccountModelImportService(nil, nil, &handlerModelImportHTTPUpstream{}, nil)
	importSvc.SetModelRegistryService(modelRegistryService)

	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.SetAccountModelImportService(importSvc)

	router := gin.New()
	router.POST("/api/v1/admin/accounts/:id/import-models", handler.ImportModels)

	body, err := json.Marshal(ImportAccountModelsRequest{Trigger: "manual"})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/901/import-models", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.updatedAccountIDs, 1)
	require.Equal(t, int64(901), adminSvc.updatedAccountIDs[0])
	require.NotNil(t, adminSvc.updatedAccounts[0].Extra)
	require.Equal(t, "value", adminSvc.updatedAccounts[0].Extra["existing"])
	require.Equal(t, service.OpenAIKnownModelsSourceImportModels, adminSvc.updatedAccounts[0].Extra["openai_known_models_source"])
	require.Equal(t, []string{"gpt-5.4", "gpt-4.1-mini"}, adminSvc.updatedAccounts[0].Extra["openai_known_models"])
	require.NotEmpty(t, adminSvc.updatedAccounts[0].Extra["openai_known_models_updated_at"])
	snapshot, ok := adminSvc.updatedAccounts[0].Extra["model_probe_snapshot"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, []string{"gpt-5.4", "gpt-4.1-mini"}, snapshot["models"])
	require.Equal(t, service.AccountModelProbeSnapshotSourceImportModels, snapshot["source"])
	require.Equal(t, "upstream", snapshot["probe_source"])
}

type handlerMixedProbeHTTPUpstream struct{}

func (u *handlerMixedProbeHTTPUpstream) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return u.DoWithTLS(req, proxyURL, accountID, accountConcurrency, nil)
}

func (u *handlerMixedProbeHTTPUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, tlsProfile *service.TLSFingerprintProfile) (*http.Response, error) {
	switch {
	case strings.HasSuffix(req.URL.Path, "/v1/models"):
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(`{"data":[{"id":"gpt-5.4"},{"id":"shared-model"}]}`),
			Request:    req,
		}, nil
	case strings.HasSuffix(req.URL.Path, "/v1beta/models"):
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioNopCloser(`{"models":[{"name":"models/gemini-2.5-pro"},{"name":"models/shared-model"}]}`),
			Request:    req,
		}, nil
	default:
		return nil, errors.New("unexpected path: " + req.URL.Path)
	}
}

func TestProbeProtocolGatewayModels_MixedReturnsSourceProtocolAndRegistryState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()

	repo := &handlerModelImportSettingRepoStub{values: make(map[string]string)}
	modelRegistryService := service.NewModelRegistryService(repo)
	_, err := modelRegistryService.UpsertEntry(context.Background(), service.UpsertModelRegistryEntryInput{
		ID:          "gpt-5.4",
		Provider:    service.PlatformOpenAI,
		DisplayName: "GPT-5.4",
		ExposedIn:   []string{"runtime", "test", "whitelist"},
	})
	require.NoError(t, err)

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	upstream := &handlerMixedProbeHTTPUpstream{}
	geminiCompatService := service.NewGeminiMessagesCompatService(nil, nil, nil, nil, nil, nil, upstream, nil, cfg)
	importSvc := service.NewAccountModelImportService(nil, geminiCompatService, upstream, nil)
	importSvc.SetModelRegistryService(modelRegistryService)

	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	handler.SetAccountModelImportService(importSvc)
	handler.SetModelRegistryService(modelRegistryService)

	router := gin.New()
	router.POST("/api/v1/admin/accounts/protocol-gateway/probe-models", handler.ProbeProtocolGatewayModels)

	body, err := json.Marshal(map[string]any{
		"gateway_protocol":   "mixed",
		"accepted_protocols": []string{"openai", "gemini"},
		"base_url":           "http://gateway.local.test",
		"api_key":            "gateway-key",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/protocol-gateway/probe-models", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ProbeSource string `json:"probe_source"`
			Models      []struct {
				ID             string `json:"id"`
				RegistryState  string `json:"registry_state"`
				RegistryModel  string `json:"registry_model_id"`
				SourceProtocol string `json:"source_protocol"`
			} `json:"models"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "upstream", resp.Data.ProbeSource)
	require.Len(t, resp.Data.Models, 3)

	modelsByID := make(map[string]struct {
		RegistryState  string
		RegistryModel  string
		SourceProtocol string
	}, len(resp.Data.Models))
	for _, model := range resp.Data.Models {
		modelsByID[model.ID] = struct {
			RegistryState  string
			RegistryModel  string
			SourceProtocol string
		}{
			RegistryState:  model.RegistryState,
			RegistryModel:  model.RegistryModel,
			SourceProtocol: model.SourceProtocol,
		}
	}

	require.Equal(t, "existing", modelsByID["gpt-5.4"].RegistryState)
	require.Equal(t, "gpt-5.4", modelsByID["gpt-5.4"].RegistryModel)
	require.Equal(t, service.PlatformOpenAI, modelsByID["gpt-5.4"].SourceProtocol)
	require.Equal(t, service.PlatformGemini, modelsByID["gemini-2.5-pro"].SourceProtocol)
	require.Equal(t, service.PlatformOpenAI, modelsByID["shared-model"].SourceProtocol)
}
