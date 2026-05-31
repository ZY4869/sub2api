package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func (s *SettingService) applyParsedPaymentSettings(result *SystemSettings, settings map[string]string) {
	paymentSettings := paymentSettingsFromRaw(settings)
	result.PaymentProviderAirwallexEnabled = paymentSettings.AirwallexEnabled
	result.PaymentProviderAirwallexEffective = IsPaymentPublicAirwallexEnabled(paymentSettings)
	result.AirwallexEnv = paymentSettings.AirwallexEnv
	result.AirwallexClientID = paymentSettings.AirwallexClientID
	result.AirwallexAPIKey = paymentSettings.AirwallexAPIKey
	result.AirwallexAPIKeyConfigured = paymentSettings.AirwallexAPIKeyConfigured
	result.AirwallexWebhookSecret = paymentSettings.AirwallexWebhookSecret
	result.AirwallexWebhookSecretConfigured = paymentSettings.AirwallexWebhookSecretConfigured
	result.PaymentMobileForceQRCodeEnabled = paymentSettings.MobileForceQRCodeEnabled
	result.PaymentAllowedCurrencies = paymentSettings.AllowedCurrencies
	result.PaymentDefaultCurrency = paymentSettings.DefaultCurrency
	result.PaymentMinTopupAmount = paymentSettings.MinTopupAmount
	result.PaymentMaxTopupAmount = paymentSettings.MaxTopupAmount
	result.PaymentSubscriptionPlans = paymentSettings.SubscriptionPlans
	result.BillingCurrencyConversionEnabled = settings[SettingKeyBillingCurrencyConversionEnabled] == "true"
	result.BillingCurrencyCNYToUSDRate = parseClampedFloatSetting(settings[SettingKeyBillingCurrencyCNYToUSDRate], 0.6, 0.00000001, 0)
	result.BillingCurrencyUSDToCNYRate = parseClampedFloatSetting(settings[SettingKeyBillingCurrencyUSDToCNYRate], 7, 0.00000001, 0)
}

func (s *SettingService) applyParsedOAuthSettings(result *SystemSettings, settings map[string]string) {
	result.AntigravityUserAgentVersion = strings.TrimSpace(settings[SettingKeyAntigravityUserAgentVersion])
	codexUAPolicy := NormalizeCodexOAuthUserAgentPolicy(settings[SettingKeyCodexOAuthUserAgentMode], settings[SettingKeyCodexOAuthUserAgentOverride])
	result.CodexOAuthUserAgentMode = codexUAPolicy.Mode
	result.CodexOAuthUserAgentOverride = codexUAPolicy.Override
	result.OpenAIAllowClaudeCodeCodexPlugin = settings[SettingKeyOpenAIAllowClaudeCodeCodexPlugin] == "true"
	result.SMTPPassword = settings[SettingKeySMTPPassword]
	result.TelegramBotToken = strings.TrimSpace(settings[SettingKeyTelegramBotToken])
	result.TelegramBotTokenConfigured = result.TelegramBotToken != ""
	result.TelegramBotTokenMasked = maskTelegramBotToken(result.TelegramBotToken)
	result.TurnstileSecretKey = settings[SettingKeyTurnstileSecretKey]

	linuxDoBase := config.LinuxDoConnectConfig{}
	if s.cfg != nil {
		linuxDoBase = s.cfg.LinuxDo
	}
	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		result.LinuxDoConnectEnabled = raw == "true"
	} else {
		result.LinuxDoConnectEnabled = linuxDoBase.Enabled
	}
	if v, ok := settings[SettingKeyLinuxDoConnectClientID]; ok && strings.TrimSpace(v) != "" {
		result.LinuxDoConnectClientID = strings.TrimSpace(v)
	} else {
		result.LinuxDoConnectClientID = linuxDoBase.ClientID
	}
	if v, ok := settings[SettingKeyLinuxDoConnectRedirectURL]; ok && strings.TrimSpace(v) != "" {
		result.LinuxDoConnectRedirectURL = strings.TrimSpace(v)
	} else {
		result.LinuxDoConnectRedirectURL = linuxDoBase.RedirectURL
	}
	result.LinuxDoConnectClientSecret = strings.TrimSpace(settings[SettingKeyLinuxDoConnectClientSecret])
	if result.LinuxDoConnectClientSecret == "" {
		result.LinuxDoConnectClientSecret = strings.TrimSpace(linuxDoBase.ClientSecret)
	}
	result.LinuxDoConnectClientSecretConfigured = result.LinuxDoConnectClientSecret != ""
	applyParsedSocialOAuthSettings(result, settings)
}

func applyParsedSocialOAuthSettings(result *SystemSettings, settings map[string]string) {
	result.GitHubOAuthEnabled = settings[SettingKeyGitHubOAuthEnabled] == "true"
	result.GitHubOAuthClientID = strings.TrimSpace(settings[SettingKeyGitHubOAuthClientID])
	result.GitHubOAuthRedirectURL = strings.TrimSpace(settings[SettingKeyGitHubOAuthRedirectURL])
	result.GitHubOAuthClientSecret = strings.TrimSpace(settings[SettingKeyGitHubOAuthClientSecret])
	result.GitHubOAuthClientSecretConfigured = result.GitHubOAuthClientSecret != ""
	result.GoogleOAuthEnabled = settings[SettingKeyGoogleOAuthEnabled] == "true"
	result.GoogleOAuthClientID = strings.TrimSpace(settings[SettingKeyGoogleOAuthClientID])
	result.GoogleOAuthRedirectURL = strings.TrimSpace(settings[SettingKeyGoogleOAuthRedirectURL])
	result.GoogleOAuthClientSecret = strings.TrimSpace(settings[SettingKeyGoogleOAuthClientSecret])
	result.GoogleOAuthClientSecretConfigured = result.GoogleOAuthClientSecret != ""
	result.DingTalkOAuthEnabled = settings[SettingKeyDingTalkOAuthEnabled] == "true"
	result.DingTalkOAuthClientID = strings.TrimSpace(settings[SettingKeyDingTalkOAuthClientID])
	result.DingTalkOAuthRedirectURL = strings.TrimSpace(settings[SettingKeyDingTalkOAuthRedirectURL])
	result.DingTalkOAuthClientSecret = strings.TrimSpace(settings[SettingKeyDingTalkOAuthClientSecret])
	result.DingTalkOAuthClientSecretConfigured = result.DingTalkOAuthClientSecret != ""
}

func (s *SettingService) applyParsedContentModerationSettings(result *SystemSettings, settings map[string]string) {
	result.ContentModerationEnabled = settings[SettingKeyContentModerationEnabled] == "true"
	result.ContentModerationProvider = s.getStringOrDefault(settings, SettingKeyContentModerationProvider, "openai")
	result.ContentModerationBaseURL = strings.TrimSpace(settings[SettingKeyContentModerationBaseURL])
	result.ContentModerationAPIKey = strings.TrimSpace(settings[SettingKeyContentModerationAPIKey])
	result.ContentModerationAPIKeys = NormalizeContentModerationAPIKeys(result.ContentModerationAPIKey, settings[SettingKeyContentModerationAPIKeys])
	result.ContentModerationAPIKeyStatuses = ContentModerationAPIKeyStatuses(result.ContentModerationAPIKeys, time.Time{})
	result.ContentModerationAPIKeyConfigured = len(result.ContentModerationAPIKeys) > 0
	result.ContentModerationModel = strings.TrimSpace(settings[SettingKeyContentModerationModel])
	result.ContentModerationTimeoutMs = parseSettingInt(settings[SettingKeyContentModerationTimeoutMs], 1500)
	result.ContentModerationDedupeWindowSeconds = parseSettingInt(settings[SettingKeyContentModerationDedupeWindowSeconds], 300)
	result.ContentModerationFailOpen = settings[SettingKeyContentModerationFailOpen] != "false"
	result.ContentModerationKeywordBlockEnabled = settings[SettingKeyContentModerationKeywordBlockEnabled] == "true"
	result.ContentModerationKeywords = NormalizeContentModerationKeywords(settings[SettingKeyContentModerationKeywords])
	result.ContentModerationModelFilter = NormalizeContentModerationModelFilter(settings[SettingKeyContentModerationModelFilter])
	result.ContentModerationCategoryThresholds = NormalizeContentModerationCategoryThresholds(settings[SettingKeyContentModerationCategoryThresholds])
}

func (s *SettingService) applyParsedOpsRuntimeSettings(result *SystemSettings, settings map[string]string) {
	result.EnableModelFallback = settings[SettingKeyEnableModelFallback] == "true"
	result.FallbackModelAnthropic = s.getStringOrDefault(settings, SettingKeyFallbackModelAnthropic, "claude-3-5-sonnet-20241022")
	result.FallbackModelOpenAI = s.getStringOrDefault(settings, SettingKeyFallbackModelOpenAI, "gpt-4o")
	result.FallbackModelGemini = s.getStringOrDefault(settings, SettingKeyFallbackModelGemini, "gemini-2.5-pro")
	result.FallbackModelAntigravity = s.getStringOrDefault(settings, SettingKeyFallbackModelAntigravity, "gemini-2.5-pro")
	if v, ok := settings[SettingKeyEnableIdentityPatch]; ok && v != "" {
		result.EnableIdentityPatch = v == "true"
	} else {
		result.EnableIdentityPatch = true
	}
	result.IdentityPatchPrompt = settings[SettingKeyIdentityPatchPrompt]
	result.OpsMonitoringEnabled = !isFalseSettingValue(settings[SettingKeyOpsMonitoringEnabled])
	result.OpsRealtimeMonitoringEnabled = !isFalseSettingValue(settings[SettingKeyOpsRealtimeMonitoringEnabled])
	result.OpsQueryModeDefault = string(ParseOpsQueryMode(settings[SettingKeyOpsQueryModeDefault]))
	result.OpsMetricsIntervalSeconds = parseClampedIntSetting(settings[SettingKeyOpsMetricsIntervalSeconds], 60, 60, 3600)
	result.OpenAIFastPolicySettings = DefaultOpenAIFastPolicySettings()
	if raw := strings.TrimSpace(settings[SettingKeyOpenAIFastPolicySettings]); raw != "" {
		result.OpenAIFastPolicySettings = ParseOpenAIFastPolicySettings(raw)
	}
	result.EnableAnthropicCacheTTL1hInjection = strings.TrimSpace(settings[SettingKeyEnableAnthropicCacheTTL1hInjection]) == "true"
	result.ChannelMonitorDefaultIntervalSeconds = parseClampedIntSetting(settings[SettingKeyChannelMonitorDefaultIntervalSeconds], 60, 15, 3600)
	result.MinClaudeCodeVersion = settings[SettingKeyMinClaudeCodeVersion]
	result.MaxClaudeCodeVersion = settings[SettingKeyMaxClaudeCodeVersion]
	result.AllowUngroupedKeyScheduling = settings[SettingKeyAllowUngroupedKeyScheduling] == "true"
	result.BackendModeEnabled = settings[SettingKeyBackendModeEnabled] == "true"
	result.MaintenanceModeEnabled = settings[SettingKeyMaintenanceModeEnabled] == "true"
}

func applyParsedAffiliateSettings(result *SystemSettings, settings map[string]string) {
	result.AffiliateEnabled = settings[SettingKeyAffiliateEnabled] == "true"
	result.AffiliateTransferEnabled = !isFalseSettingValue(settings[SettingKeyAffiliateTransferEnabled])
	result.AffiliateRebateOnUsageEnabled = !isFalseSettingValue(settings[SettingKeyAffiliateRebateOnUsageEnabled])
	result.AffiliateRebateOnTopupEnabled = !isFalseSettingValue(settings[SettingKeyAffiliateRebateOnTopupEnabled])
	result.AffiliateRebateRate = parseClampedFloatSetting(settings[SettingKeyAffiliateRebateRate], 20.0, 0, 100)
	result.AffiliateRebateFreezeHours = parseClampedIntSetting(settings[SettingKeyAffiliateRebateFreezeHours], 0, 0, 720)
	result.AffiliateRebateDurationDays = parseClampedIntSetting(settings[SettingKeyAffiliateRebateDurationDays], 0, 0, 3650)
	result.AffiliateRebatePerInviteeCap = parseClampedFloatSetting(settings[SettingKeyAffiliateRebatePerInviteeCap], 0, 0, 0)
	result.AffiliateAffCodeLength = parseClampedIntSetting(settings[SettingKeyAffiliateAffCodeLength], 10, 6, 32)
}

func parseClampedIntSetting(raw string, fallback int, minValue int, maxValue int) int {
	value := fallback
	if parsed, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
		value = parsed
	}
	if value < minValue {
		return minValue
	}
	if maxValue > minValue && value > maxValue {
		return maxValue
	}
	return value
}

func parseClampedFloatSetting(raw string, fallback float64, minValue float64, maxValue float64) float64 {
	value := fallback
	if parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
		value = parsed
	}
	if value < minValue {
		return minValue
	}
	if maxValue > minValue && value > maxValue {
		return maxValue
	}
	return value
}
