package service

import (
	"strconv"
	"strings"
)

func (s *SettingService) parseSettings(settings map[string]string) *SystemSettings {
	emailVerifyEnabled := settings[SettingKeyEmailVerifyEnabled] == "true"
	result := &SystemSettings{
		RegistrationEnabled:              settings[SettingKeyRegistrationEnabled] == "true",
		EmailVerifyEnabled:               emailVerifyEnabled,
		RegistrationEmailSuffixWhitelist: ParseRegistrationEmailSuffixWhitelist(settings[SettingKeyRegistrationEmailSuffixWhitelist]),
		PromoCodeEnabled:                 settings[SettingKeyPromoCodeEnabled] != "false",
		PasswordResetEnabled:             emailVerifyEnabled && settings[SettingKeyPasswordResetEnabled] == "true",
		FrontendURL:                      strings.TrimSpace(settings[SettingKeyFrontendURL]),
		InvitationCodeEnabled:            settings[SettingKeyInvitationCodeEnabled] == "true",
		TotpEnabled:                      settings[SettingKeyTotpEnabled] == "true",
		SMTPHost:                         settings[SettingKeySMTPHost],
		SMTPUsername:                     settings[SettingKeySMTPUsername],
		SMTPFrom:                         settings[SettingKeySMTPFrom],
		SMTPFromName:                     settings[SettingKeySMTPFromName],
		SMTPUseTLS:                       settings[SettingKeySMTPUseTLS] == "true",
		SMTPPasswordConfigured:           settings[SettingKeySMTPPassword] != "",
		TelegramChatID:                   strings.TrimSpace(settings[SettingKeyTelegramChatID]),
		TurnstileEnabled:                 settings[SettingKeyTurnstileEnabled] == "true",
		TurnstileSiteKey:                 settings[SettingKeyTurnstileSiteKey],
		TurnstileSecretKeyConfigured:     settings[SettingKeyTurnstileSecretKey] != "",
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
		PurchaseSubscriptionEnabled:      settings[SettingKeyPurchaseSubscriptionEnabled] == "true",
		PurchaseSubscriptionURL:          strings.TrimSpace(settings[SettingKeyPurchaseSubscriptionURL]),
		CustomMenuItems:                  settings[SettingKeyCustomMenuItems],
		LoginAgreementEnabled:            settings[SettingKeyLoginAgreementEnabled] == "true",
		LoginAgreementMode:               NormalizeLoginAgreementMode(settings[SettingKeyLoginAgreementMode]),
		LoginAgreementUpdatedAt:          strings.TrimSpace(settings[SettingKeyLoginAgreementUpdatedAt]),
		LoginAgreementDocuments:          ParseLoginAgreementDocuments(settings[SettingKeyLoginAgreementDocuments]),
		BackendModeEnabled:               settings[SettingKeyBackendModeEnabled] == "true",
		MaintenanceModeEnabled:           settings[SettingKeyMaintenanceModeEnabled] == "true",
	}
	if port, err := strconv.Atoi(settings[SettingKeySMTPPort]); err == nil {
		result.SMTPPort = port
	} else {
		result.SMTPPort = 587
	}
	if concurrency, err := strconv.Atoi(settings[SettingKeyDefaultConcurrency]); err == nil {
		result.DefaultConcurrency = concurrency
	} else {
		result.DefaultConcurrency = s.cfg.Default.UserConcurrency
	}
	if balance, err := strconv.ParseFloat(settings[SettingKeyDefaultBalance], 64); err == nil {
		result.DefaultBalance = balance
	} else {
		result.DefaultBalance = s.cfg.Default.UserBalance
	}
	result.DefaultSubscriptions = parseDefaultSubscriptions(settings[SettingKeyDefaultSubscriptions])
	s.applyParsedPaymentSettings(result, settings)
	s.applyParsedOAuthSettings(result, settings)
	s.applyParsedContentModerationSettings(result, settings)
	s.applyParsedOpsRuntimeSettings(result, settings)
	applyParsedAffiliateSettings(result, settings)
	return result
}
