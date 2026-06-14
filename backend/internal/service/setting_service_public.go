package service

import (
	"context"
	"fmt"
	"strings"
)

func (s *SettingService) GetAllSettings(ctx context.Context) (*SystemSettings, error) {
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}
	return s.parseSettings(settings), nil
}

func (s *SettingService) IsPublicModelCatalogEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyPublicModelCatalogEnabled)
	if err != nil {
		return true
	}
	return !isFalseSettingValue(value)
}

func (s *SettingService) GetPublicSettings(ctx context.Context) (*PublicSettings, error) {
	keys := []string{
		SettingKeyRegistrationEnabled,
		SettingKeyEmailVerifyEnabled,
		SettingKeyRegistrationEmailSuffixWhitelist,
		SettingKeyPromoCodeEnabled,
		SettingKeyPasswordResetEnabled,
		SettingKeyInvitationCodeEnabled,
		SettingKeyTotpEnabled,
		SettingKeyTurnstileEnabled,
		SettingKeyTurnstileSiteKey,
		SettingKeySiteName,
		SettingKeySiteLogo,
		SettingKeySiteSubtitle,
		SettingKeyVisualPresetDefault,
		SettingKeyAccountAiryWhiteSurfaceEnabled,
		SettingKeyAPIBaseURL,
		SettingKeyContactInfo,
		SettingKeyDocURL,
		SettingKeyHomeContent,
		SettingKeyHideCcsImportButton,
		SettingKeyAvailableChannelsEnabled,
		SettingKeyChannelMonitorEnabled,
		SettingKeyPublicModelCatalogEnabled,
		SettingKeyAffiliateEnabled,
		SettingKeyPurchaseSubscriptionEnabled,
		SettingKeyPurchaseSubscriptionURL,
		SettingKeyPaymentProviderAirwallexEnabled,
		SettingKeyAirwallexClientID,
		SettingKeyAirwallexAPIKey,
		SettingKeyPaymentMobileForceQRCodeEnabled,
		SettingKeyPaymentAllowedCurrencies,
		SettingKeyPaymentDefaultCurrency,
		SettingKeyPaymentMinTopupAmount,
		SettingKeyPaymentMaxTopupAmount,
		SettingKeyPaymentSubscriptionPlans,
		SettingKeyCustomMenuItems,
		SettingKeyLoginAgreementEnabled,
		SettingKeyLoginAgreementMode,
		SettingKeyLoginAgreementUpdatedAt,
		SettingKeyLoginAgreementDocuments,
		SettingKeyLinuxDoConnectEnabled,
		SettingKeyGitHubOAuthEnabled,
		SettingKeyGoogleOAuthEnabled,
		SettingKeyDingTalkOAuthEnabled,
		SettingKeyBackendModeEnabled,
		SettingKeyMaintenanceModeEnabled,
		SettingKeyAdminComplianceEnabled,
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("get public settings: %w", err)
	}
	linuxDoEnabled := false
	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		linuxDoEnabled = raw == "true"
	} else {
		linuxDoEnabled = s.cfg != nil && s.cfg.LinuxDo.Enabled
	}
	githubEnabled := settings[SettingKeyGitHubOAuthEnabled] == "true"
	googleEnabled := settings[SettingKeyGoogleOAuthEnabled] == "true"
	dingtalkEnabled := settings[SettingKeyDingTalkOAuthEnabled] == "true"
	emailVerifyEnabled := settings[SettingKeyEmailVerifyEnabled] == "true"
	passwordResetEnabled := emailVerifyEnabled && settings[SettingKeyPasswordResetEnabled] == "true"
	registrationEmailSuffixWhitelist := ParseRegistrationEmailSuffixWhitelist(settings[SettingKeyRegistrationEmailSuffixWhitelist])
	paymentSettings := paymentSettingsFromRaw(settings)
	return &PublicSettings{
		RegistrationEnabled:              settings[SettingKeyRegistrationEnabled] == "true",
		EmailVerifyEnabled:               emailVerifyEnabled,
		RegistrationEmailSuffixWhitelist: registrationEmailSuffixWhitelist,
		PromoCodeEnabled:                 settings[SettingKeyPromoCodeEnabled] != "false",
		PasswordResetEnabled:             passwordResetEnabled,
		InvitationCodeEnabled:            settings[SettingKeyInvitationCodeEnabled] == "true",
		TotpEnabled:                      settings[SettingKeyTotpEnabled] == "true",
		TurnstileEnabled:                 settings[SettingKeyTurnstileEnabled] == "true",
		TurnstileSiteKey:                 settings[SettingKeyTurnstileSiteKey],
		SiteName:                         s.getStringOrDefault(settings, SettingKeySiteName, "Sub2API"),
		SiteLogo:                         settings[SettingKeySiteLogo],
		SiteSubtitle:                     s.getStringOrDefault(settings, SettingKeySiteSubtitle, "Subscription to API Conversion Platform"),
		VisualPresetDefault:              NormalizeVisualPreset(settings[SettingKeyVisualPresetDefault]),
		AccountAiryWhiteSurfaceEnabled:   settings[SettingKeyAccountAiryWhiteSurfaceEnabled] == "true",
		APIBaseURL:                       settings[SettingKeyAPIBaseURL],
		ContactInfo:                      settings[SettingKeyContactInfo],
		DocURL:                           settings[SettingKeyDocURL],
		HomeContent:                      settings[SettingKeyHomeContent],
		HideCcsImportButton:              settings[SettingKeyHideCcsImportButton] == "true",
		AvailableChannelsEnabled:         settings[SettingKeyAvailableChannelsEnabled] == "true",
		ChannelMonitorEnabled:            settings[SettingKeyChannelMonitorEnabled] == "true",
		PublicModelCatalogEnabled:        !isFalseSettingValue(settings[SettingKeyPublicModelCatalogEnabled]),
		AffiliateEnabled:                 settings[SettingKeyAffiliateEnabled] == "true",
		PurchaseSubscriptionEnabled:      settings[SettingKeyPurchaseSubscriptionEnabled] == "true",
		PurchaseSubscriptionURL:          strings.TrimSpace(settings[SettingKeyPurchaseSubscriptionURL]),
		PaymentProviderAirwallexEnabled:  IsPaymentPublicAirwallexEnabled(paymentSettings),
		PaymentMobileForceQRCodeEnabled:  paymentSettings.MobileForceQRCodeEnabled,
		PaymentAllowedCurrencies:         paymentSettings.AllowedCurrencies,
		PaymentDefaultCurrency:           paymentSettings.DefaultCurrency,
		PaymentMinTopupAmount:            paymentSettings.MinTopupAmount,
		PaymentMaxTopupAmount:            paymentSettings.MaxTopupAmount,
		PaymentSubscriptionPlans:         paymentSettings.SubscriptionPlans,
		CustomMenuItems:                  settings[SettingKeyCustomMenuItems],
		LoginAgreementEnabled:            settings[SettingKeyLoginAgreementEnabled] == "true",
		LoginAgreementMode:               NormalizeLoginAgreementMode(settings[SettingKeyLoginAgreementMode]),
		LoginAgreementUpdatedAt:          strings.TrimSpace(settings[SettingKeyLoginAgreementUpdatedAt]),
		LoginAgreementDocuments:          ParseLoginAgreementDocuments(settings[SettingKeyLoginAgreementDocuments]),
		LinuxDoOAuthEnabled:              linuxDoEnabled,
		GitHubOAuthEnabled:               githubEnabled,
		GoogleOAuthEnabled:               googleEnabled,
		DingTalkOAuthEnabled:             dingtalkEnabled,
		BackendModeEnabled:               settings[SettingKeyBackendModeEnabled] == "true",
		MaintenanceModeEnabled:           settings[SettingKeyMaintenanceModeEnabled] == "true",
		AdminComplianceEnabled:           settings[SettingKeyAdminComplianceEnabled] == "true",
	}, nil
}

func (s *SettingService) SetOnUpdateCallback(callback func()) {
	s.addOnUpdateCallback(callback)
}

func (s *SettingService) SetOnS3UpdateCallback(callback func()) {
	s.onS3Update = callback
}

func (s *SettingService) SetVersion(version string) {
	s.version = version
}
