package admin

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type channelMonitorHandlerRepoStub struct {
	service.ChannelMonitorRepository
	createErr error
}

func (s *channelMonitorHandlerRepoStub) Create(_ context.Context, monitor *service.ChannelMonitor) (*service.ChannelMonitor, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	clone := *monitor
	clone.ID = 88
	clone.CreatedAt = time.Now().UTC()
	clone.UpdatedAt = clone.CreatedAt
	return &clone, nil
}

type channelMonitorHandlerEncryptorStub struct{}

func (channelMonitorHandlerEncryptorStub) Encrypt(plaintext string) (string, error) {
	return "encrypted:" + plaintext, nil
}

func (channelMonitorHandlerEncryptorStub) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

type channelMonitorHandlerSettingRepoStub struct {
	values map[string]string
}

func (s *channelMonitorHandlerSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *channelMonitorHandlerSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *channelMonitorHandlerSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *channelMonitorHandlerSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *channelMonitorHandlerSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *channelMonitorHandlerSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}

func (s *channelMonitorHandlerSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestChannelMonitorHandler_Create_ReturnsSanitizedMonitor(t *testing.T) {
	router := newChannelMonitorHandlerTestRouter(&channelMonitorHandlerRepoStub{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/channel-monitors", bytes.NewBufferString(`{
		"name":"OpenAI health",
		"provider":"openai",
		"endpoint":"https://api.openai.example/v1/responses",
		"api_key":"sk-secret",
		"enabled":true,
		"primary_model_id":"gpt-5.4"
	}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), "sk-secret")
	require.Contains(t, rec.Body.String(), `"api_key_configured":true`)
}

func TestChannelMonitorHandler_Create_EnabledWithoutKeyReturnsVisibleError(t *testing.T) {
	router := newChannelMonitorHandlerTestRouter(&channelMonitorHandlerRepoStub{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/channel-monitors", bytes.NewBufferString(`{
		"name":"OpenAI health",
		"provider":"openai",
		"endpoint":"https://api.openai.example/v1/responses",
		"api_key":"   ",
		"enabled":true,
		"primary_model_id":"gpt-5.4"
	}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "CHANNEL_MONITOR_API_KEY_REQUIRED")
}

func TestChannelMonitorHandler_Create_LogsCreateFailure(t *testing.T) {
	router := newChannelMonitorHandlerTestRouter(&channelMonitorHandlerRepoStub{
		createErr: errors.New("db down"),
	})
	core, logs := observer.New(zap.WarnLevel)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/channel-monitors", bytes.NewBufferString(`{
		"name":"OpenAI health",
		"provider":"openai",
		"endpoint":"https://api.openai.example/v1/responses",
		"api_key":"sk-secret",
		"enabled":true,
		"primary_model_id":"gpt-5.4"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(logger.IntoContext(req.Context(), zap.New(core)))
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, 1, logs.FilterMessage("channel_monitor_create_failed").Len())
	require.NotContains(t, rec.Body.String(), "sk-secret")
}

func newChannelMonitorHandlerTestRouter(repo service.ChannelMonitorRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	settingRepo := &channelMonitorHandlerSettingRepoStub{values: map[string]string{
		service.SettingKeyChannelMonitorDefaultIntervalSeconds: "60",
	}}
	monitorSvc := service.NewChannelMonitorService(
		repo,
		nil,
		nil,
		service.NewSettingService(settingRepo, &config.Config{}),
		channelMonitorHandlerEncryptorStub{},
		&config.Config{},
	)
	router := gin.New()
	router.POST("/admin/channel-monitors", NewChannelMonitorHandler(monitorSvc).Create)
	return router
}
