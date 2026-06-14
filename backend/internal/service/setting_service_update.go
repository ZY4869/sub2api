package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"strconv"
	"strings"
	"time"
)

func (s *SettingService) UpdateSettings(ctx context.Context, settings *SystemSettings) error {
	if err := s.validateDefaultSubscriptionGroups(ctx, settings.DefaultSubscriptions); err != nil {
		return err
	}
	normalizedWhitelist, err := NormalizeRegistrationEmailSuffixWhitelist(settings.RegistrationEmailSuffixWhitelist)
	if err != nil {
		return infraerrors.BadRequest("INVALID_REGISTRATION_EMAIL_SUFFIX_WHITELIST", err.Error())
	}
	if normalizedWhitelist == nil {
		normalizedWhitelist = []string{}
	}
	settings.RegistrationEmailSuffixWhitelist = normalizedWhitelist
	updates := make(map[string]string)
	updates[SettingKeyRegistrationEnabled] = strconv.FormatBool(settings.RegistrationEnabled)
	updates[SettingKeyEmailVerifyEnabled] = strconv.FormatBool(settings.EmailVerifyEnabled)
	registrationEmailSuffixWhitelistJSON, err := json.Marshal(settings.RegistrationEmailSuffixWhitelist)
	if err != nil {
		return fmt.Errorf("marshal registration email suffix whitelist: %w", err)
	}
	updates[SettingKeyRegistrationEmailSuffixWhitelist] = string(registrationEmailSuffixWhitelistJSON)
	updates[SettingKeyPromoCodeEnabled] = strconv.FormatBool(settings.PromoCodeEnabled)
	updates[SettingKeyPasswordResetEnabled] = strconv.FormatBool(settings.PasswordResetEnabled)
	updates[SettingKeyFrontendURL] = strings.TrimSpace(settings.FrontendURL)
	updates[SettingKeyInvitationCodeEnabled] = strconv.FormatBool(settings.InvitationCodeEnabled)
	updates[SettingKeyTotpEnabled] = strconv.FormatBool(settings.TotpEnabled)
	updates[SettingKeySMTPHost] = settings.SMTPHost
	updates[SettingKeySMTPPort] = strconv.Itoa(settings.SMTPPort)
	updates[SettingKeySMTPUsername] = settings.SMTPUsername
	if settings.SMTPPassword != "" {
		updates[SettingKeySMTPPassword] = settings.SMTPPassword
	}
	updates[SettingKeySMTPFrom] = settings.SMTPFrom
	updates[SettingKeySMTPFromName] = settings.SMTPFromName
	updates[SettingKeySMTPUseTLS] = strconv.FormatBool(settings.SMTPUseTLS)
	updates[SettingKeyTelegramChatID] = strings.TrimSpace(settings.TelegramChatID)
	if strings.TrimSpace(settings.TelegramBotToken) != "" {
		updates[SettingKeyTelegramBotToken] = strings.TrimSpace(settings.TelegramBotToken)
	}
	updates[SettingKeyTurnstileEnabled] = strconv.FormatBool(settings.TurnstileEnabled)
	updates[SettingKeyTurnstileSiteKey] = settings.TurnstileSiteKey
	if settings.TurnstileSecretKey != "" {
		updates[SettingKeyTurnstileSecretKey] = settings.TurnstileSecretKey
	}
	updates[SettingKeyLinuxDoConnectEnabled] = strconv.FormatBool(settings.LinuxDoConnectEnabled)
	updates[SettingKeyLinuxDoConnectClientID] = settings.LinuxDoConnectClientID
	updates[SettingKeyLinuxDoConnectRedirectURL] = settings.LinuxDoConnectRedirectURL
	if settings.LinuxDoConnectClientSecret != "" {
		updates[SettingKeyLinuxDoConnectClientSecret] = settings.LinuxDoConnectClientSecret
	}
	updates[SettingKeyGitHubOAuthEnabled] = strconv.FormatBool(settings.GitHubOAuthEnabled)
	updates[SettingKeyGitHubOAuthClientID] = settings.GitHubOAuthClientID
	updates[SettingKeyGitHubOAuthRedirectURL] = settings.GitHubOAuthRedirectURL
	if settings.GitHubOAuthClientSecret != "" {
		updates[SettingKeyGitHubOAuthClientSecret] = settings.GitHubOAuthClientSecret
	}
	updates[SettingKeyGoogleOAuthEnabled] = strconv.FormatBool(settings.GoogleOAuthEnabled)
	updates[SettingKeyGoogleOAuthClientID] = settings.GoogleOAuthClientID
	updates[SettingKeyGoogleOAuthRedirectURL] = settings.GoogleOAuthRedirectURL
	if settings.GoogleOAuthClientSecret != "" {
		updates[SettingKeyGoogleOAuthClientSecret] = settings.GoogleOAuthClientSecret
	}
	updates[SettingKeyDingTalkOAuthEnabled] = strconv.FormatBool(settings.DingTalkOAuthEnabled)
	updates[SettingKeyDingTalkOAuthClientID] = strings.TrimSpace(settings.DingTalkOAuthClientID)
	updates[SettingKeyDingTalkOAuthRedirectURL] = strings.TrimSpace(settings.DingTalkOAuthRedirectURL)
	if strings.TrimSpace(settings.DingTalkOAuthClientSecret) != "" {
		updates[SettingKeyDingTalkOAuthClientSecret] = strings.TrimSpace(settings.DingTalkOAuthClientSecret)
	}
	updates[SettingKeyContentModerationEnabled] = strconv.FormatBool(settings.ContentModerationEnabled)
	updates[SettingKeyContentModerationProvider] = strings.TrimSpace(settings.ContentModerationProvider)
	updates[SettingKeyContentModerationBaseURL] = strings.TrimSpace(settings.ContentModerationBaseURL)
	if settings.ContentModerationAPIKeys != nil {
		apiKeysJSON, err := MarshalContentModerationAPIKeys(settings.ContentModerationAPIKeys)
		if err != nil {
			return fmt.Errorf("marshal content moderation api keys: %w", err)
		}
		updates[SettingKeyContentModerationAPIKeys] = apiKeysJSON
		if len(settings.ContentModerationAPIKeys) > 0 {
			updates[SettingKeyContentModerationAPIKey] = settings.ContentModerationAPIKeys[0].Key
		} else {
			updates[SettingKeyContentModerationAPIKey] = ""
		}
	} else if strings.TrimSpace(settings.ContentModerationAPIKey) != "" {
		key := strings.TrimSpace(settings.ContentModerationAPIKey)
		apiKeysJSON, err := MarshalContentModerationAPIKeys([]ContentModerationAPIKey{{Key: key}})
		if err != nil {
			return fmt.Errorf("marshal content moderation api key: %w", err)
		}
		updates[SettingKeyContentModerationAPIKey] = key
		updates[SettingKeyContentModerationAPIKeys] = apiKeysJSON
	}
	updates[SettingKeyContentModerationModel] = strings.TrimSpace(settings.ContentModerationModel)
	updates[SettingKeyContentModerationTimeoutMs] = strconv.Itoa(settings.ContentModerationTimeoutMs)
	updates[SettingKeyContentModerationDedupeWindowSeconds] = strconv.Itoa(settings.ContentModerationDedupeWindowSeconds)
	updates[SettingKeyContentModerationFailOpen] = strconv.FormatBool(settings.ContentModerationFailOpen)
	updates[SettingKeyContentModerationKeywordBlockEnabled] = strconv.FormatBool(settings.ContentModerationKeywordBlockEnabled)
	moderationKeywordsJSON, err := MarshalContentModerationKeywords(settings.ContentModerationKeywords)
	if err != nil {
		return fmt.Errorf("marshal content moderation keywords: %w", err)
	}
	updates[SettingKeyContentModerationKeywords] = moderationKeywordsJSON
	moderationModelFilterJSON, err := MarshalContentModerationModelFilter(settings.ContentModerationModelFilter)
	if err != nil {
		return fmt.Errorf("marshal content moderation model filter: %w", err)
	}
	updates[SettingKeyContentModerationModelFilter] = moderationModelFilterJSON
	moderationThresholdsJSON, err := MarshalContentModerationCategoryThresholds(settings.ContentModerationCategoryThresholds)
	if err != nil {
		return fmt.Errorf("marshal content moderation category thresholds: %w", err)
	}
	updates[SettingKeyContentModerationCategoryThresholds] = moderationThresholdsJSON
	updates[SettingKeySiteName] = settings.SiteName
	updates[SettingKeySiteLogo] = settings.SiteLogo
	updates[SettingKeySiteSubtitle] = settings.SiteSubtitle
	updates[SettingKeyVisualPresetDefault] = NormalizeVisualPreset(settings.VisualPresetDefault)
	updates[SettingKeyAccountAiryWhiteSurfaceEnabled] = strconv.FormatBool(settings.AccountAiryWhiteSurfaceEnabled)
	updates[SettingKeyAPIBaseURL] = settings.APIBaseURL
	updates[SettingKeyContactInfo] = settings.ContactInfo
	updates[SettingKeyDocURL] = settings.DocURL
	updates[SettingKeyHomeContent] = settings.HomeContent
	updates[SettingKeyHideCcsImportButton] = strconv.FormatBool(settings.HideCcsImportButton)
	updates[SettingKeyAvailableChannelsEnabled] = strconv.FormatBool(settings.AvailableChannelsEnabled)
	updates[SettingKeyChannelMonitorEnabled] = strconv.FormatBool(settings.ChannelMonitorEnabled)
	monitorInterval := settings.ChannelMonitorDefaultIntervalSeconds
	if monitorInterval <= 0 {
		monitorInterval = 60
	}
	if monitorInterval < 15 {
		monitorInterval = 15
	}
	if monitorInterval > 3600 {
		monitorInterval = 3600
	}
	updates[SettingKeyChannelMonitorDefaultIntervalSeconds] = strconv.Itoa(monitorInterval)
	updates[SettingKeyPublicModelCatalogEnabled] = strconv.FormatBool(settings.PublicModelCatalogEnabled)
	updates[SettingKeyPurchaseSubscriptionEnabled] = strconv.FormatBool(settings.PurchaseSubscriptionEnabled)
	updates[SettingKeyPurchaseSubscriptionURL] = strings.TrimSpace(settings.PurchaseSubscriptionURL)
	updates[SettingKeyPaymentProviderAirwallexEnabled] = strconv.FormatBool(settings.PaymentProviderAirwallexEnabled)
	updates[SettingKeyAirwallexEnv] = NormalizeAirwallexEnv(settings.AirwallexEnv)
	updates[SettingKeyAirwallexClientID] = strings.TrimSpace(settings.AirwallexClientID)
	if strings.TrimSpace(settings.AirwallexAPIKey) != "" {
		updates[SettingKeyAirwallexAPIKey] = strings.TrimSpace(settings.AirwallexAPIKey)
	}
	if strings.TrimSpace(settings.AirwallexWebhookSecret) != "" {
		updates[SettingKeyAirwallexWebhookSecret] = strings.TrimSpace(settings.AirwallexWebhookSecret)
	}
	updates[SettingKeyPaymentMobileForceQRCodeEnabled] = strconv.FormatBool(settings.PaymentMobileForceQRCodeEnabled)
	paymentCurrenciesJSON, err := json.Marshal(NormalizePaymentAllowedCurrencies(settings.PaymentAllowedCurrencies))
	if err != nil {
		return fmt.Errorf("marshal payment allowed currencies: %w", err)
	}
	updates[SettingKeyPaymentAllowedCurrencies] = string(paymentCurrenciesJSON)
	defaultCurrency := NormalizePaymentCurrency(settings.PaymentDefaultCurrency)
	if defaultCurrency == "" || !PaymentCurrencyAllowed(defaultCurrency, settings.PaymentAllowedCurrencies) {
		defaultCurrency = NormalizePaymentAllowedCurrencies(settings.PaymentAllowedCurrencies)[0]
	}
	updates[SettingKeyPaymentDefaultCurrency] = defaultCurrency
	if settings.PaymentMinTopupAmount <= 0 {
		settings.PaymentMinTopupAmount = DefaultPaymentSettings().MinTopupAmount
	}
	if settings.PaymentMaxTopupAmount < settings.PaymentMinTopupAmount {
		settings.PaymentMaxTopupAmount = settings.PaymentMinTopupAmount
	}
	updates[SettingKeyPaymentMinTopupAmount] = strconv.FormatFloat(settings.PaymentMinTopupAmount, 'f', 8, 64)
	updates[SettingKeyPaymentMaxTopupAmount] = strconv.FormatFloat(settings.PaymentMaxTopupAmount, 'f', 8, 64)
	updates[SettingKeyPaymentSubscriptionPlans] = MarshalPaymentSubscriptionPlans(settings.PaymentSubscriptionPlans)
	conversion := settings.BillingCurrencyConversionSettings()
	updates[SettingKeyBillingCurrencyConversionEnabled] = strconv.FormatBool(conversion.Enabled)
	updates[SettingKeyBillingCurrencyCNYToUSDRate] = strconv.FormatFloat(conversion.CNYToUSDRate, 'f', 8, 64)
	updates[SettingKeyBillingCurrencyUSDToCNYRate] = strconv.FormatFloat(conversion.USDToCNYRate, 'f', 8, 64)
	antigravityVersion, ok := NormalizeAntigravityUserAgentVersion(settings.AntigravityUserAgentVersion)
	if !ok {
		return infraerrors.BadRequest("ANTIGRAVITY_USER_AGENT_VERSION_INVALID", "antigravity user-agent version must match major.minor.patch[-suffix]")
	}
	updates[SettingKeyAntigravityUserAgentVersion] = antigravityVersion
	codexUAPolicy := NormalizeCodexOAuthUserAgentPolicy(settings.CodexOAuthUserAgentMode, settings.CodexOAuthUserAgentOverride)
	updates[SettingKeyCodexOAuthUserAgentMode] = codexUAPolicy.Mode
	updates[SettingKeyCodexOAuthUserAgentOverride] = codexUAPolicy.Override
	updates[SettingKeyOpenAIAllowClaudeCodeCodexPlugin] = strconv.FormatBool(settings.OpenAIAllowClaudeCodeCodexPlugin)
	allowedCodexClientsJSON, err := MarshalOpenAIAllowedCodexClients(settings.OpenAIAllowedCodexClients)
	if err != nil {
		return err
	}
	updates[SettingKeyOpenAIAllowedCodexClients] = allowedCodexClientsJSON
	updates[SettingKeyCustomMenuItems] = settings.CustomMenuItems
	updates[SettingKeyLoginAgreementEnabled] = strconv.FormatBool(settings.LoginAgreementEnabled)
	updates[SettingKeyLoginAgreementMode] = NormalizeLoginAgreementMode(settings.LoginAgreementMode)
	updates[SettingKeyLoginAgreementUpdatedAt] = strings.TrimSpace(settings.LoginAgreementUpdatedAt)
	loginAgreementDocumentsJSON, err := MarshalLoginAgreementDocuments(settings.LoginAgreementDocuments)
	if err != nil {
		return fmt.Errorf("marshal login agreement documents: %w", err)
	}
	updates[SettingKeyLoginAgreementDocuments] = loginAgreementDocumentsJSON
	updates[SettingKeyDefaultConcurrency] = strconv.Itoa(settings.DefaultConcurrency)
	updates[SettingKeyDefaultBalance] = strconv.FormatFloat(settings.DefaultBalance, 'f', 8, 64)
	defaultSubsJSON, err := json.Marshal(settings.DefaultSubscriptions)
	if err != nil {
		return fmt.Errorf("marshal default subscriptions: %w", err)
	}
	updates[SettingKeyDefaultSubscriptions] = string(defaultSubsJSON)
	updates[SettingKeyEnableModelFallback] = strconv.FormatBool(settings.EnableModelFallback)
	updates[SettingKeyFallbackModelAnthropic] = settings.FallbackModelAnthropic
	updates[SettingKeyFallbackModelOpenAI] = settings.FallbackModelOpenAI
	updates[SettingKeyFallbackModelGemini] = settings.FallbackModelGemini
	updates[SettingKeyFallbackModelAntigravity] = settings.FallbackModelAntigravity
	updates[SettingKeyEnableIdentityPatch] = strconv.FormatBool(settings.EnableIdentityPatch)
	updates[SettingKeyIdentityPatchPrompt] = settings.IdentityPatchPrompt
	updates[SettingKeyOpsMonitoringEnabled] = strconv.FormatBool(settings.OpsMonitoringEnabled)
	updates[SettingKeyOpsRealtimeMonitoringEnabled] = strconv.FormatBool(settings.OpsRealtimeMonitoringEnabled)
	updates[SettingKeyOpsQueryModeDefault] = string(ParseOpsQueryMode(settings.OpsQueryModeDefault))
	if settings.OpsMetricsIntervalSeconds > 0 {
		updates[SettingKeyOpsMetricsIntervalSeconds] = strconv.Itoa(settings.OpsMetricsIntervalSeconds)
	}

	fastPolicy := settings.OpenAIFastPolicySettings
	if fastPolicy == nil {
		fastPolicy = DefaultOpenAIFastPolicySettings()
	}
	fastPolicy = NormalizeOpenAIFastPolicySettings(fastPolicy)
	fastPolicyJSON, err := json.Marshal(fastPolicy)
	if err != nil {
		return fmt.Errorf("marshal openai fast policy settings: %w", err)
	}
	updates[SettingKeyOpenAIFastPolicySettings] = string(fastPolicyJSON)
	updates[SettingKeyEnableAnthropicCacheTTL1hInjection] = strconv.FormatBool(settings.EnableAnthropicCacheTTL1hInjection)

	updates[SettingKeyMinClaudeCodeVersion] = settings.MinClaudeCodeVersion
	updates[SettingKeyMaxClaudeCodeVersion] = settings.MaxClaudeCodeVersion
	updates[SettingKeyAllowUngroupedKeyScheduling] = strconv.FormatBool(settings.AllowUngroupedKeyScheduling)
	updates[SettingKeyBackendModeEnabled] = strconv.FormatBool(settings.BackendModeEnabled)
	updates[SettingKeyMaintenanceModeEnabled] = strconv.FormatBool(settings.MaintenanceModeEnabled)
	updates[SettingKeyAdminComplianceEnabled] = strconv.FormatBool(settings.AdminComplianceEnabled)

	updates[SettingKeyAffiliateEnabled] = strconv.FormatBool(settings.AffiliateEnabled)
	updates[SettingKeyAffiliateTransferEnabled] = strconv.FormatBool(settings.AffiliateTransferEnabled)
	updates[SettingKeyAffiliateRebateOnUsageEnabled] = strconv.FormatBool(settings.AffiliateRebateOnUsageEnabled)
	updates[SettingKeyAffiliateRebateOnTopupEnabled] = strconv.FormatBool(settings.AffiliateRebateOnTopupEnabled)
	updates[SettingKeyAffiliateRebateRate] = strconv.FormatFloat(settings.AffiliateRebateRate, 'f', 4, 64)
	updates[SettingKeyAffiliateRebateFreezeHours] = strconv.Itoa(settings.AffiliateRebateFreezeHours)
	updates[SettingKeyAffiliateRebateDurationDays] = strconv.Itoa(settings.AffiliateRebateDurationDays)
	updates[SettingKeyAffiliateRebatePerInviteeCap] = strconv.FormatFloat(settings.AffiliateRebatePerInviteeCap, 'f', 8, 64)
	updates[SettingKeyAffiliateAffCodeLength] = strconv.Itoa(settings.AffiliateAffCodeLength)
	err = s.settingRepo.SetMultiple(ctx, updates)
	if err == nil {
		versionBoundsSF.Forget("version_bounds")
		versionBoundsCache.Store(&cachedVersionBounds{
			min:       settings.MinClaudeCodeVersion,
			max:       settings.MaxClaudeCodeVersion,
			expiresAt: time.Now().Add(versionBoundsCacheTTL).UnixNano(),
		})
		backendModeSF.Forget("backend_mode_enabled")
		backendModeCache.Store(&cachedBackendMode{value: settings.BackendModeEnabled, expiresAt: time.Now().Add(backendModeCacheTTL).UnixNano()})
		maintenanceModeSF.Forget("maintenance_mode_enabled")
		maintenanceModeCache.Store(&cachedMaintenanceMode{value: settings.MaintenanceModeEnabled, expiresAt: time.Now().Add(maintenanceModeCacheTTL).UnixNano()})
		s.notifyUpdateCallbacks()
	}
	return err
}
func (s *SettingService) validateDefaultSubscriptionGroups(ctx context.Context, items []DefaultSubscriptionSetting) error {
	if len(items) == 0 {
		return nil
	}
	checked := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.GroupID <= 0 {
			continue
		}
		if _, ok := checked[item.GroupID]; ok {
			return ErrDefaultSubGroupDuplicate.WithMetadata(map[string]string{"group_id": strconv.FormatInt(item.GroupID, 10)})
		}
		checked[item.GroupID] = struct{}{}
		if s.defaultSubGroupReader == nil {
			continue
		}
		group, err := s.defaultSubGroupReader.GetByID(ctx, item.GroupID)
		if err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				return ErrDefaultSubGroupInvalid.WithMetadata(map[string]string{"group_id": strconv.FormatInt(item.GroupID, 10)})
			}
			return fmt.Errorf("get default subscription group %d: %w", item.GroupID, err)
		}
		if !group.IsSubscriptionType() {
			return ErrDefaultSubGroupInvalid.WithMetadata(map[string]string{"group_id": strconv.FormatInt(item.GroupID, 10)})
		}
	}
	return nil
}
