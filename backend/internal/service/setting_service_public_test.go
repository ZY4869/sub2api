//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingPublicRepoStub struct {
	values map[string]string
}

func (s *settingPublicRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingPublicRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *settingPublicRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *settingPublicRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingPublicRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *settingPublicRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingPublicRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetPublicSettings_ExposesRegistrationEmailSuffixWhitelist(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyRegistrationEnabled:              "true",
			SettingKeyEmailVerifyEnabled:               "true",
			SettingKeyRegistrationEmailSuffixWhitelist: `["@EXAMPLE.com"," @foo.bar ","@invalid_domain",""]`,
			SettingKeyPublicModelCatalogEnabled:        "false",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar"}, settings.RegistrationEmailSuffixWhitelist)
	require.False(t, settings.PublicModelCatalogEnabled)
}

func TestSettingService_IsPublicModelCatalogEnabled_DefaultsTrue(t *testing.T) {
	repo := &settingPublicRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	require.True(t, svc.IsPublicModelCatalogEnabled(context.Background()))
}

func TestSettingService_PublicModelCatalogEnabled_RoundTripsAcrossPublicSettings(t *testing.T) {
	ctx := context.Background()
	repo := &settingPublicRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	initial, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.True(t, initial.PublicModelCatalogEnabled)
	require.True(t, svc.IsPublicModelCatalogEnabled(ctx))

	err = svc.UpdateSettings(ctx, &SystemSettings{
		PublicModelCatalogEnabled: false,
	})
	require.NoError(t, err)
	require.Equal(t, "false", repo.values[SettingKeyPublicModelCatalogEnabled])

	updated, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.False(t, updated.PublicModelCatalogEnabled)
	require.False(t, svc.IsPublicModelCatalogEnabled(ctx))

	injected, err := svc.GetPublicSettingsForInjection(ctx)
	require.NoError(t, err)
	raw, err := json.Marshal(injected)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, false, payload["public_model_catalog_enabled"])

	err = svc.UpdateSettings(ctx, &SystemSettings{
		PublicModelCatalogEnabled: true,
	})
	require.NoError(t, err)
	require.Equal(t, "true", repo.values[SettingKeyPublicModelCatalogEnabled])

	restored, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.True(t, restored.PublicModelCatalogEnabled)
	require.True(t, svc.IsPublicModelCatalogEnabled(ctx))
}

func TestSettingService_MaintenanceMode_RoundTripsAcrossPublicSettings(t *testing.T) {
	ctx := context.Background()
	repo := &settingPublicRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	initial, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.False(t, initial.MaintenanceModeEnabled)

	err = svc.UpdateSettings(ctx, &SystemSettings{
		MaintenanceModeEnabled: true,
	})
	require.NoError(t, err)
	require.Equal(t, "true", repo.values[SettingKeyMaintenanceModeEnabled])

	updated, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.True(t, updated.MaintenanceModeEnabled)
	require.True(t, svc.IsMaintenanceModeEnabled(ctx))

	injected, err := svc.GetPublicSettingsForInjection(ctx)
	require.NoError(t, err)
	raw, err := json.Marshal(injected)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, true, payload["maintenance_mode_enabled"])
}
