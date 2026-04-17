package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type publicSettingsRepoStub struct {
	values map[string]string
}

func (s *publicSettingsRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *publicSettingsRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return value, nil
}

func (s *publicSettingsRepoStub) Set(ctx context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *publicSettingsRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = s.values[key]
	}
	return result, nil
}

func (s *publicSettingsRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *publicSettingsRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *publicSettingsRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingHandlerGetPublicSettings_DoesNotExposeAPIDocsOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &publicSettingsRepoStub{
		values: map[string]string{
			service.SettingKeySiteName:           "Sub2API",
			service.SettingKeyDocURL:             "https://docs.example.com",
			service.SettingKeyAPIDocsMarkdown:    "# 运行时覆盖文档\n",
			service.SettingKeyBackendModeEnabled: "false",
		},
	}
	settingService := service.NewSettingService(repo, &config.Config{})
	handler := NewSettingHandler(settingService, "v-test")

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/settings/public", nil)

	handler.GetPublicSettings(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "api_docs_markdown")

	var body response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)

	data, ok := body.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://docs.example.com", data["doc_url"])
	_, exists := data["api_docs_markdown"]
	require.False(t, exists)
}
