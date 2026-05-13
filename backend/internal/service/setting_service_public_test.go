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
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
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

func TestSettingService_AvailableChannelsAndChannelMonitor_RoundTripsAcrossPublicSettings(t *testing.T) {
	ctx := context.Background()
	repo := &settingPublicRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	initial, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.False(t, initial.AvailableChannelsEnabled)
	require.False(t, initial.ChannelMonitorEnabled)

	err = svc.UpdateSettings(ctx, &SystemSettings{
		AvailableChannelsEnabled: true,
		ChannelMonitorEnabled:    true,
	})
	require.NoError(t, err)
	require.Equal(t, "true", repo.values[SettingKeyAvailableChannelsEnabled])
	require.Equal(t, "true", repo.values[SettingKeyChannelMonitorEnabled])

	updated, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.True(t, updated.AvailableChannelsEnabled)
	require.True(t, updated.ChannelMonitorEnabled)

	injected, err := svc.GetPublicSettingsForInjection(ctx)
	require.NoError(t, err)
	raw, err := json.Marshal(injected)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, true, payload["available_channels_enabled"])
	require.Equal(t, true, payload["channel_monitor_enabled"])
}

func TestSettingService_LoginAgreement_RoundTripsAcrossPublicSettings(t *testing.T) {
	ctx := context.Background()
	repo := &settingPublicRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(ctx, &SystemSettings{
		LoginAgreementEnabled:   true,
		LoginAgreementMode:      "checkbox",
		LoginAgreementUpdatedAt: "2026-05-07",
		LoginAgreementDocuments: []LoginAgreementDocument{
			{ID: "terms", Title: "Terms", PageSlug: "terms"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "true", repo.values[SettingKeyLoginAgreementEnabled])
	require.NotContains(t, repo.values[SettingKeyLoginAgreementDocuments], "page_content")

	settings, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.True(t, settings.LoginAgreementEnabled)
	require.Equal(t, LoginAgreementModeCheckbox, settings.LoginAgreementMode)
	require.Equal(t, "2026-05-07", settings.LoginAgreementUpdatedAt)
	require.Equal(t, []LoginAgreementDocument{
		{ID: "terms", Title: "Terms", PageSlug: "terms"},
	}, settings.LoginAgreementDocuments)

	injected, err := svc.GetPublicSettingsForInjection(ctx)
	require.NoError(t, err)
	raw, err := json.Marshal(injected)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"login_agreement_enabled":true`)
	require.Contains(t, string(raw), `"page_slug":"terms"`)
}

func TestSettingService_PurchaseSubscriptionSettings_RoundTripAcrossPublicSettings(t *testing.T) {
	ctx := context.Background()
	repo := &settingPublicRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(ctx, &SystemSettings{
		PurchaseSubscriptionEnabled:             true,
		PurchaseSubscriptionURL:                 "https://pay.example.com/checkout",
		PurchaseSubscriptionProvider:            PurchaseSubscriptionProviderAirwallex,
		PurchaseSubscriptionSupportedCurrencies: []string{"usd", "cny", "usd"},
		PurchaseSubscriptionDefaultCurrency:     "usd",
		PurchaseSubscriptionDefaultCountryCode:  "us",
		PurchaseSubscriptionPaymentEnv:          PurchaseSubscriptionPaymentEnvSandbox,
		PurchaseSubscriptionExtraParams: map[string]string{
			"merchant_region": "global",
		},
	})
	require.NoError(t, err)

	settings, err := svc.GetPublicSettings(ctx)
	require.NoError(t, err)
	require.True(t, settings.PurchaseSubscriptionEnabled)
	require.Equal(t, PurchaseSubscriptionProviderAirwallex, settings.PurchaseSubscriptionProvider)
	require.Equal(t, []string{"USD", "CNY"}, settings.PurchaseSubscriptionSupportedCurrencies)
	require.Equal(t, "USD", settings.PurchaseSubscriptionDefaultCurrency)
	require.Equal(t, "US", settings.PurchaseSubscriptionDefaultCountryCode)
	require.Equal(t, PurchaseSubscriptionPaymentEnvSandbox, settings.PurchaseSubscriptionPaymentEnv)
	require.Equal(t, map[string]string{"merchant_region": "global"}, settings.PurchaseSubscriptionExtraParams)
}

func TestSettingService_UpdateSettings_PreservesContentModerationKeysWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	rawKeys, err := MarshalContentModerationAPIKeys([]ContentModerationAPIKey{
		{Key: "sk-first"},
		{Key: "sk-second"},
	})
	require.NoError(t, err)
	repo := &settingPublicRepoStub{values: map[string]string{
		SettingKeyContentModerationAPIKey:  "sk-first",
		SettingKeyContentModerationAPIKeys: rawKeys,
	}}
	svc := NewSettingService(repo, &config.Config{})

	err = svc.UpdateSettings(ctx, &SystemSettings{
		SiteName: "Updated",
	})
	require.NoError(t, err)

	require.Equal(t, rawKeys, repo.values[SettingKeyContentModerationAPIKeys])
	require.Equal(t, "sk-first", repo.values[SettingKeyContentModerationAPIKey])
}

func TestSettingService_UpdateSettings_ClearsContentModerationKeysWhenExplicitlyEmpty(t *testing.T) {
	ctx := context.Background()
	rawKeys, err := MarshalContentModerationAPIKeys([]ContentModerationAPIKey{{Key: "sk-first"}})
	require.NoError(t, err)
	repo := &settingPublicRepoStub{values: map[string]string{
		SettingKeyContentModerationAPIKey:  "sk-first",
		SettingKeyContentModerationAPIKeys: rawKeys,
	}}
	svc := NewSettingService(repo, &config.Config{})

	err = svc.UpdateSettings(ctx, &SystemSettings{
		ContentModerationAPIKeys: []ContentModerationAPIKey{},
	})
	require.NoError(t, err)

	require.Equal(t, "[]", repo.values[SettingKeyContentModerationAPIKeys])
	require.Equal(t, "", repo.values[SettingKeyContentModerationAPIKey])
}

func TestSettingService_GetPublicSettings_FiltersDraftMarkdownPages(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyCustomMenuItems: `[{"id":"published","label":"Published","visibility":"user","page_mode":"markdown","page_slug":"published","page_content":"# hidden","page_published":true},{"id":"draft","label":"Draft","visibility":"user","page_mode":"markdown","page_slug":"draft","page_content":"# hidden","page_published":false},{"id":"admin","label":"Admin","visibility":"admin","url":"https://admin.example.com","page_mode":"iframe"}]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)

	raw, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	payload, err := json.Marshal(raw)
	require.NoError(t, err)

	var injected map[string]any
	require.NoError(t, json.Unmarshal(payload, &injected))

	items, ok := injected["custom_menu_items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)

	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "published", item["id"])
	_, exists := item["page_content"]
	require.False(t, exists)
	require.NotEmpty(t, settings.CustomMenuItems)
}

func TestSettingService_GetCustomPageBySlug_PreservesBackslashes(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyCustomMenuItems: `[{"id":"guide","label":"Guide","visibility":"user","page_mode":"markdown","page_slug":"guide","page_content":"# Guide\r\nPath: C:\\temp\\file.txt","page_published":true}]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	page, err := svc.GetCustomPageBySlug(context.Background(), "guide")
	require.NoError(t, err)
	require.NotNil(t, page)
	require.Equal(t, "# Guide\nPath: C:\\temp\\file.txt", page.Content)
}
