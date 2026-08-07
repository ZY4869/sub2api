package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type passkeySettingRepoStub struct {
	values map[string]string
}

func (s *passkeySettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *passkeySettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", ErrSettingNotFound
}

func (s *passkeySettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *passkeySettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *passkeySettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *passkeySettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *passkeySettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestPasskeySettingsRequireWebAuthnConfigAndStoredSetting(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name            string
		webAuthnEnabled bool
		storedValue     string
		wantEnabled     bool
	}{
		{name: "config disabled keeps public disabled even when setting is true", webAuthnEnabled: false, storedValue: "true", wantEnabled: false},
		{name: "config enabled but setting missing is disabled by default", webAuthnEnabled: true, storedValue: "", wantEnabled: false},
		{name: "config enabled but setting false is disabled", webAuthnEnabled: true, storedValue: "false", wantEnabled: false},
		{name: "config enabled and setting true exposes passkey", webAuthnEnabled: true, storedValue: "true", wantEnabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{}
			if tt.storedValue != "" {
				values[SettingKeyPasskeyEnabled] = tt.storedValue
			}
			repo := &passkeySettingRepoStub{values: values}
			svc := NewSettingService(repo, &config.Config{WebAuthn: config.WebAuthnConfig{Enabled: tt.webAuthnEnabled}})

			publicSettings, err := svc.GetPublicSettings(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.wantEnabled, publicSettings.PasskeyEnabled)
			require.Equal(t, tt.wantEnabled, svc.IsPasskeyEnabled(ctx))
		})
	}
}

func TestPasskeySettingsInjectionIncludesEffectiveFlag(t *testing.T) {
	ctx := context.Background()
	repo := &passkeySettingRepoStub{values: map[string]string{SettingKeyPasskeyEnabled: "true"}}
	svc := NewSettingService(repo, &config.Config{WebAuthn: config.WebAuthnConfig{Enabled: true}})

	injected, err := svc.GetPublicSettingsForInjection(ctx)
	require.NoError(t, err)
	raw, err := json.Marshal(injected)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, true, payload["passkey_enabled"])
}
