package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type passkeyHandlerSettingRepoStub struct {
	values map[string]string
}

func (s *passkeyHandlerSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *passkeyHandlerSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *passkeyHandlerSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *passkeyHandlerSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *passkeyHandlerSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *passkeyHandlerSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *passkeyHandlerSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func newDisabledPasskeyHandler() *PasskeyHandler {
	cfg := &config.Config{WebAuthn: config.WebAuthnConfig{Enabled: true}}
	settings := service.NewSettingService(&passkeyHandlerSettingRepoStub{
		values: map[string]string{service.SettingKeyPasskeyEnabled: "false"},
	}, cfg)
	passkeys := service.NewPasskeyService(cfg, nil, nil, nil)
	return NewPasskeyHandler(nil, passkeys, settings)
}

func TestPasskeyHandlerDisabledGateReturnsEmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/user/passkeys", nil)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42, Concurrency: 1})

	newDisabledPasskeyHandler().List(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Code int                         `json:"code"`
		Data []passkeyCredentialResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 0, payload.Code)
	require.Empty(t, payload.Data)
}

func TestPasskeyHandlerDisabledGateRejectsBeginLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/passkey/login/begin", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	newDisabledPasskeyHandler().BeginLogin(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "PASSKEYS_DISABLED")
}
