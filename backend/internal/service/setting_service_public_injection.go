package service

import (
	"context"
	"encoding/json"
	"strings"
)

func (s *SettingService) GetPublicSettingsForInjection(ctx context.Context) (any, error) {
	settings, err := s.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &struct {
		RegistrationEnabled              bool                      `json:"registration_enabled"`
		EmailVerifyEnabled               bool                      `json:"email_verify_enabled"`
		RegistrationEmailSuffixWhitelist []string                  `json:"registration_email_suffix_whitelist"`
		PromoCodeEnabled                 bool                      `json:"promo_code_enabled"`
		PasswordResetEnabled             bool                      `json:"password_reset_enabled"`
		InvitationCodeEnabled            bool                      `json:"invitation_code_enabled"`
		TotpEnabled                      bool                      `json:"totp_enabled"`
		TurnstileEnabled                 bool                      `json:"turnstile_enabled"`
		TurnstileSiteKey                 string                    `json:"turnstile_site_key,omitempty"`
		SiteName                         string                    `json:"site_name"`
		SiteLogo                         string                    `json:"site_logo,omitempty"`
		SiteSubtitle                     string                    `json:"site_subtitle,omitempty"`
		VisualPresetDefault              string                    `json:"visual_preset_default"`
		AccountAiryWhiteSurfaceEnabled   bool                      `json:"account_airy_white_surface_enabled"`
		APIBaseURL                       string                    `json:"api_base_url,omitempty"`
		ContactInfo                      string                    `json:"contact_info,omitempty"`
		DocURL                           string                    `json:"doc_url,omitempty"`
		HomeContent                      string                    `json:"home_content,omitempty"`
		HideCcsImportButton              bool                      `json:"hide_ccs_import_button"`
		AvailableChannelsEnabled         bool                      `json:"available_channels_enabled"`
		ChannelMonitorEnabled            bool                      `json:"channel_monitor_enabled"`
		PublicModelCatalogEnabled        bool                      `json:"public_model_catalog_enabled"`
		PurchaseSubscriptionEnabled      bool                      `json:"purchase_subscription_enabled"`
		PurchaseSubscriptionURL          string                    `json:"purchase_subscription_url,omitempty"`
		PaymentProviderAirwallexEnabled  bool                      `json:"payment_provider_airwallex_enabled"`
		PaymentMobileForceQRCodeEnabled  bool                      `json:"payment_mobile_force_qrcode_enabled"`
		PaymentAllowedCurrencies         []string                  `json:"payment_allowed_currencies"`
		PaymentDefaultCurrency           string                    `json:"payment_default_currency"`
		PaymentMinTopupAmount            float64                   `json:"payment_min_topup_amount"`
		PaymentMaxTopupAmount            float64                   `json:"payment_max_topup_amount"`
		PaymentSubscriptionPlans         []PaymentSubscriptionPlan `json:"payment_subscription_plans"`
		CustomMenuItems                  json.RawMessage           `json:"custom_menu_items"`
		LoginAgreementEnabled            bool                      `json:"login_agreement_enabled"`
		LoginAgreementMode               string                    `json:"login_agreement_mode"`
		LoginAgreementUpdatedAt          string                    `json:"login_agreement_updated_at,omitempty"`
		LoginAgreementDocuments          any                       `json:"login_agreement_documents"`
		LinuxDoOAuthEnabled              bool                      `json:"linuxdo_oauth_enabled"`
		GitHubOAuthEnabled               bool                      `json:"github_oauth_enabled"`
		GoogleOAuthEnabled               bool                      `json:"google_oauth_enabled"`
		DingTalkOAuthEnabled             bool                      `json:"dingtalk_oauth_enabled"`
		BackendModeEnabled               bool                      `json:"backend_mode_enabled"`
		MaintenanceModeEnabled           bool                      `json:"maintenance_mode_enabled"`
		Version                          string                    `json:"version,omitempty"`
	}{
		RegistrationEnabled:              settings.RegistrationEnabled,
		EmailVerifyEnabled:               settings.EmailVerifyEnabled,
		RegistrationEmailSuffixWhitelist: settings.RegistrationEmailSuffixWhitelist,
		PromoCodeEnabled:                 settings.PromoCodeEnabled,
		PasswordResetEnabled:             settings.PasswordResetEnabled,
		InvitationCodeEnabled:            settings.InvitationCodeEnabled,
		TotpEnabled:                      settings.TotpEnabled,
		TurnstileEnabled:                 settings.TurnstileEnabled,
		TurnstileSiteKey:                 settings.TurnstileSiteKey,
		SiteName:                         settings.SiteName,
		SiteLogo:                         settings.SiteLogo,
		SiteSubtitle:                     settings.SiteSubtitle,
		VisualPresetDefault:              settings.VisualPresetDefault,
		AccountAiryWhiteSurfaceEnabled:   settings.AccountAiryWhiteSurfaceEnabled,
		APIBaseURL:                       settings.APIBaseURL,
		ContactInfo:                      settings.ContactInfo,
		DocURL:                           settings.DocURL,
		HomeContent:                      settings.HomeContent,
		HideCcsImportButton:              settings.HideCcsImportButton,
		AvailableChannelsEnabled:         settings.AvailableChannelsEnabled,
		ChannelMonitorEnabled:            settings.ChannelMonitorEnabled,
		PublicModelCatalogEnabled:        settings.PublicModelCatalogEnabled,
		PurchaseSubscriptionEnabled:      settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:          settings.PurchaseSubscriptionURL,
		PaymentProviderAirwallexEnabled:  settings.PaymentProviderAirwallexEnabled,
		PaymentMobileForceQRCodeEnabled:  settings.PaymentMobileForceQRCodeEnabled,
		PaymentAllowedCurrencies:         settings.PaymentAllowedCurrencies,
		PaymentDefaultCurrency:           settings.PaymentDefaultCurrency,
		PaymentMinTopupAmount:            settings.PaymentMinTopupAmount,
		PaymentMaxTopupAmount:            settings.PaymentMaxTopupAmount,
		PaymentSubscriptionPlans:         settings.PaymentSubscriptionPlans,
		CustomMenuItems:                  filterUserVisibleMenuItems(settings.CustomMenuItems),
		LoginAgreementEnabled:            settings.LoginAgreementEnabled,
		LoginAgreementMode:               settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:          settings.LoginAgreementUpdatedAt,
		LoginAgreementDocuments:          settings.LoginAgreementDocuments,
		LinuxDoOAuthEnabled:              settings.LinuxDoOAuthEnabled,
		GitHubOAuthEnabled:               settings.GitHubOAuthEnabled,
		GoogleOAuthEnabled:               settings.GoogleOAuthEnabled,
		DingTalkOAuthEnabled:             settings.DingTalkOAuthEnabled,
		BackendModeEnabled:               settings.BackendModeEnabled,
		MaintenanceModeEnabled:           settings.MaintenanceModeEnabled,
		Version:                          s.version,
	}, nil
}

func filterUserVisibleMenuItems(raw string) json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return json.RawMessage("[]")
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return json.RawMessage("[]")
	}
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		visibility, _ := item["visibility"].(string)
		if visibility == "admin" {
			continue
		}
		if !isVisiblePublishedCustomPageItem(item) {
			continue
		}
		delete(item, "page_content")
		filtered = append(filtered, item)
	}
	if len(filtered) == 0 {
		return json.RawMessage("[]")
	}
	result, err := json.Marshal(filtered)
	if err != nil {
		return json.RawMessage("[]")
	}
	return result
}

func isVisiblePublishedCustomPageItem(item map[string]any) bool {
	pageMode, _ := item["page_mode"].(string)
	if !strings.EqualFold(strings.TrimSpace(pageMode), "markdown") {
		return true
	}

	published, ok := item["page_published"].(bool)
	return ok && published
}
