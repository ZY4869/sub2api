package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"log"
	"regexp"
	"strings"
	"time"
)

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
var menuItemIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func generateMenuItemID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate menu item ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

type SettingHandler struct {
	settingService   *service.SettingService
	emailService     *service.EmailService
	emailTemplates   *service.EmailTemplateService
	telegramNotifier *service.TelegramNotifierService
	turnstileService *service.TurnstileService
	opsService       *service.OpsService
}

func NewSettingHandler(settingService *service.SettingService, emailService *service.EmailService, telegramNotifier *service.TelegramNotifierService, turnstileService *service.TurnstileService, opsService *service.OpsService) *SettingHandler {
	return &SettingHandler{settingService: settingService, emailService: emailService, telegramNotifier: telegramNotifier, turnstileService: turnstileService, opsService: opsService}
}

func (h *SettingHandler) SetEmailTemplateService(templateService *service.EmailTemplateService) {
	if h == nil {
		return
	}
	h.emailTemplates = templateService
}
func (h *SettingHandler) auditSettingsUpdate(c *gin.Context, before *service.SystemSettings, after *service.SystemSettings, req UpdateSettingsRequest) {
	if before == nil || after == nil {
		return
	}
	changed := diffSettings(before, after, req)
	if len(changed) == 0 {
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	role, _ := middleware.GetUserRoleFromContext(c)
	log.Printf("AUDIT: settings updated at=%s user_id=%d role=%s changed=%v", time.Now().UTC().Format(time.RFC3339), subject.UserID, role, changed)
}
func diffSettings(before *service.SystemSettings, after *service.SystemSettings, req UpdateSettingsRequest) []string {
	changed := make([]string, 0, 20)
	if before.RegistrationEnabled != after.RegistrationEnabled {
		changed = append(changed, "registration_enabled")
	}
	if before.EmailVerifyEnabled != after.EmailVerifyEnabled {
		changed = append(changed, "email_verify_enabled")
	}
	if !equalStringSlice(before.RegistrationEmailSuffixWhitelist, after.RegistrationEmailSuffixWhitelist) {
		changed = append(changed, "registration_email_suffix_whitelist")
	}
	if before.PasswordResetEnabled != after.PasswordResetEnabled {
		changed = append(changed, "password_reset_enabled")
	}
	if before.FrontendURL != after.FrontendURL {
		changed = append(changed, "frontend_url")
	}
	if before.TotpEnabled != after.TotpEnabled {
		changed = append(changed, "totp_enabled")
	}
	if before.SMTPHost != after.SMTPHost {
		changed = append(changed, "smtp_host")
	}
	if before.SMTPPort != after.SMTPPort {
		changed = append(changed, "smtp_port")
	}
	if before.SMTPUsername != after.SMTPUsername {
		changed = append(changed, "smtp_username")
	}
	if req.SMTPPassword != "" {
		changed = append(changed, "smtp_password")
	}
	if before.SMTPFrom != after.SMTPFrom {
		changed = append(changed, "smtp_from_email")
	}
	if before.SMTPFromName != after.SMTPFromName {
		changed = append(changed, "smtp_from_name")
	}
	if before.SMTPUseTLS != after.SMTPUseTLS {
		changed = append(changed, "smtp_use_tls")
	}
	if before.TelegramChatID != after.TelegramChatID {
		changed = append(changed, "telegram_chat_id")
	}
	if strings.TrimSpace(req.TelegramBotToken) != "" {
		changed = append(changed, "telegram_bot_token")
	}
	if before.TurnstileEnabled != after.TurnstileEnabled {
		changed = append(changed, "turnstile_enabled")
	}
	if before.TurnstileSiteKey != after.TurnstileSiteKey {
		changed = append(changed, "turnstile_site_key")
	}
	if req.TurnstileSecretKey != "" {
		changed = append(changed, "turnstile_secret_key")
	}
	if before.LinuxDoConnectEnabled != after.LinuxDoConnectEnabled {
		changed = append(changed, "linuxdo_connect_enabled")
	}
	if before.LinuxDoConnectClientID != after.LinuxDoConnectClientID {
		changed = append(changed, "linuxdo_connect_client_id")
	}
	if req.LinuxDoConnectClientSecret != "" {
		changed = append(changed, "linuxdo_connect_client_secret")
	}
	if before.LinuxDoConnectRedirectURL != after.LinuxDoConnectRedirectURL {
		changed = append(changed, "linuxdo_connect_redirect_url")
	}
	if before.GitHubOAuthEnabled != after.GitHubOAuthEnabled {
		changed = append(changed, "github_oauth_enabled")
	}
	if before.GitHubOAuthClientID != after.GitHubOAuthClientID {
		changed = append(changed, "github_oauth_client_id")
	}
	if req.GitHubOAuthClientSecret != "" {
		changed = append(changed, "github_oauth_client_secret")
	}
	if before.GitHubOAuthRedirectURL != after.GitHubOAuthRedirectURL {
		changed = append(changed, "github_oauth_redirect_url")
	}
	if before.GoogleOAuthEnabled != after.GoogleOAuthEnabled {
		changed = append(changed, "google_oauth_enabled")
	}
	if before.GoogleOAuthClientID != after.GoogleOAuthClientID {
		changed = append(changed, "google_oauth_client_id")
	}
	if req.GoogleOAuthClientSecret != "" {
		changed = append(changed, "google_oauth_client_secret")
	}
	if before.GoogleOAuthRedirectURL != after.GoogleOAuthRedirectURL {
		changed = append(changed, "google_oauth_redirect_url")
	}
	if before.DingTalkOAuthEnabled != after.DingTalkOAuthEnabled {
		changed = append(changed, "dingtalk_oauth_enabled")
	}
	if before.DingTalkOAuthClientID != after.DingTalkOAuthClientID {
		changed = append(changed, "dingtalk_oauth_client_id")
	}
	if req.DingTalkOAuthClientSecret != "" {
		changed = append(changed, "dingtalk_oauth_client_secret")
	}
	if before.DingTalkOAuthRedirectURL != after.DingTalkOAuthRedirectURL {
		changed = append(changed, "dingtalk_oauth_redirect_url")
	}
	if before.ContentModerationEnabled != after.ContentModerationEnabled {
		changed = append(changed, "content_moderation_enabled")
	}
	if before.ContentModerationProvider != after.ContentModerationProvider {
		changed = append(changed, "content_moderation_provider")
	}
	if before.ContentModerationBaseURL != after.ContentModerationBaseURL {
		changed = append(changed, "content_moderation_base_url")
	}
	if req.ContentModerationAPIKey != "" {
		changed = append(changed, "content_moderation_api_key")
	}
	if len(req.ContentModerationAPIKeys) > 0 || len(req.DeleteContentModerationAPIKeyHashes) > 0 || strings.TrimSpace(req.ContentModerationAPIKeysMode) != "" {
		changed = append(changed, "content_moderation_api_keys")
	}
	if before.ContentModerationModel != after.ContentModerationModel {
		changed = append(changed, "content_moderation_model")
	}
	if before.ContentModerationTimeoutMs != after.ContentModerationTimeoutMs {
		changed = append(changed, "content_moderation_timeout_ms")
	}
	if before.ContentModerationDedupeWindowSeconds != after.ContentModerationDedupeWindowSeconds {
		changed = append(changed, "content_moderation_dedupe_window_seconds")
	}
	if before.ContentModerationFailOpen != after.ContentModerationFailOpen {
		changed = append(changed, "content_moderation_fail_open")
	}
	if before.ContentModerationKeywordBlockEnabled != after.ContentModerationKeywordBlockEnabled {
		changed = append(changed, "content_moderation_keyword_block_enabled")
	}
	if !equalStringSlice(before.ContentModerationKeywords, after.ContentModerationKeywords) {
		changed = append(changed, "content_moderation_keywords")
	}
	if !equalFloatMap(before.ContentModerationCategoryThresholds, after.ContentModerationCategoryThresholds) {
		changed = append(changed, "content_moderation_category_thresholds")
	}
	if before.ContentModerationCyberPolicyEnabled != after.ContentModerationCyberPolicyEnabled {
		changed = append(changed, "content_moderation_cyber_policy_enabled")
	}
	if !equalContentModerationCyberCategories(before.ContentModerationCyberCategories, after.ContentModerationCyberCategories) {
		changed = append(changed, "content_moderation_cyber_categories")
	}
	if before.SiteName != after.SiteName {
		changed = append(changed, "site_name")
	}
	if before.SiteLogo != after.SiteLogo {
		changed = append(changed, "site_logo")
	}
	if before.SiteSubtitle != after.SiteSubtitle {
		changed = append(changed, "site_subtitle")
	}
	if before.AccountAiryWhiteSurfaceEnabled != after.AccountAiryWhiteSurfaceEnabled {
		changed = append(changed, "account_airy_white_surface_enabled")
	}
	if before.APIBaseURL != after.APIBaseURL {
		changed = append(changed, "api_base_url")
	}
	if before.ContactInfo != after.ContactInfo {
		changed = append(changed, "contact_info")
	}
	if before.DocURL != after.DocURL {
		changed = append(changed, "doc_url")
	}
	if before.HomeContent != after.HomeContent {
		changed = append(changed, "home_content")
	}
	if before.HideCcsImportButton != after.HideCcsImportButton {
		changed = append(changed, "hide_ccs_import_button")
	}
	if before.AvailableChannelsEnabled != after.AvailableChannelsEnabled {
		changed = append(changed, "available_channels_enabled")
	}
	if before.ChannelMonitorEnabled != after.ChannelMonitorEnabled {
		changed = append(changed, "channel_monitor_enabled")
	}
	if before.ChannelMonitorDefaultIntervalSeconds != after.ChannelMonitorDefaultIntervalSeconds {
		changed = append(changed, "channel_monitor_default_interval_seconds")
	}
	if before.PublicModelCatalogEnabled != after.PublicModelCatalogEnabled {
		changed = append(changed, "public_model_catalog_enabled")
	}
	if before.DefaultConcurrency != after.DefaultConcurrency {
		changed = append(changed, "default_concurrency")
	}
	if before.DefaultBalance != after.DefaultBalance {
		changed = append(changed, "default_balance")
	}
	if !equalDefaultSubscriptions(before.DefaultSubscriptions, after.DefaultSubscriptions) {
		changed = append(changed, "default_subscriptions")
	}
	if before.EnableModelFallback != after.EnableModelFallback {
		changed = append(changed, "enable_model_fallback")
	}
	if before.FallbackModelAnthropic != after.FallbackModelAnthropic {
		changed = append(changed, "fallback_model_anthropic")
	}
	if before.FallbackModelOpenAI != after.FallbackModelOpenAI {
		changed = append(changed, "fallback_model_openai")
	}
	if before.FallbackModelGemini != after.FallbackModelGemini {
		changed = append(changed, "fallback_model_gemini")
	}
	if before.FallbackModelAntigravity != after.FallbackModelAntigravity {
		changed = append(changed, "fallback_model_antigravity")
	}
	if before.EnableIdentityPatch != after.EnableIdentityPatch {
		changed = append(changed, "enable_identity_patch")
	}
	if before.IdentityPatchPrompt != after.IdentityPatchPrompt {
		changed = append(changed, "identity_patch_prompt")
	}
	if before.ClaudeOAuthSystemPromptBlocksEnabled != after.ClaudeOAuthSystemPromptBlocksEnabled {
		changed = append(changed, "claude_oauth_system_prompt_blocks_enabled")
	}
	if before.ClaudeOAuthSystemPromptBlocks != after.ClaudeOAuthSystemPromptBlocks {
		changed = append(changed, "claude_oauth_system_prompt_blocks")
	}
	if before.OpsMonitoringEnabled != after.OpsMonitoringEnabled {
		changed = append(changed, "ops_monitoring_enabled")
	}
	if before.OpsRealtimeMonitoringEnabled != after.OpsRealtimeMonitoringEnabled {
		changed = append(changed, "ops_realtime_monitoring_enabled")
	}
	if before.OpsQueryModeDefault != after.OpsQueryModeDefault {
		changed = append(changed, "ops_query_mode_default")
	}
	if before.OpsMetricsIntervalSeconds != after.OpsMetricsIntervalSeconds {
		changed = append(changed, "ops_metrics_interval_seconds")
	}
	if before.MinClaudeCodeVersion != after.MinClaudeCodeVersion {
		changed = append(changed, "min_claude_code_version")
	}
	if before.MaxClaudeCodeVersion != after.MaxClaudeCodeVersion {
		changed = append(changed, "max_claude_code_version")
	}
	if before.AllowUngroupedKeyScheduling != after.AllowUngroupedKeyScheduling {
		changed = append(changed, "allow_ungrouped_key_scheduling")
	}
	if before.BackendModeEnabled != after.BackendModeEnabled {
		changed = append(changed, "backend_mode_enabled")
	}
	if before.MaintenanceModeEnabled != after.MaintenanceModeEnabled {
		changed = append(changed, "maintenance_mode_enabled")
	}
	if before.PurchaseSubscriptionEnabled != after.PurchaseSubscriptionEnabled {
		changed = append(changed, "purchase_subscription_enabled")
	}
	if before.PurchaseSubscriptionURL != after.PurchaseSubscriptionURL {
		changed = append(changed, "purchase_subscription_url")
	}
	if before.PaymentProviderAirwallexEnabled != after.PaymentProviderAirwallexEnabled {
		changed = append(changed, "payment_provider_airwallex_enabled")
	}
	if before.PaymentMobileForceQRCodeEnabled != after.PaymentMobileForceQRCodeEnabled {
		changed = append(changed, "payment_mobile_force_qrcode_enabled")
	}
	if before.AirwallexEnv != after.AirwallexEnv {
		changed = append(changed, "airwallex_env")
	}
	if before.AirwallexClientID != after.AirwallexClientID {
		changed = append(changed, "airwallex_client_id")
	}
	if req.AirwallexAPIKey != nil && strings.TrimSpace(*req.AirwallexAPIKey) != "" {
		changed = append(changed, "airwallex_api_key")
	}
	if req.AirwallexWebhookSecret != nil && strings.TrimSpace(*req.AirwallexWebhookSecret) != "" {
		changed = append(changed, "airwallex_webhook_secret")
	}
	if !equalStringSlice(before.PaymentAllowedCurrencies, after.PaymentAllowedCurrencies) {
		changed = append(changed, "payment_allowed_currencies")
	}
	if before.PaymentDefaultCurrency != after.PaymentDefaultCurrency {
		changed = append(changed, "payment_default_currency")
	}
	if before.PaymentMinTopupAmount != after.PaymentMinTopupAmount {
		changed = append(changed, "payment_min_topup_amount")
	}
	if before.PaymentMaxTopupAmount != after.PaymentMaxTopupAmount {
		changed = append(changed, "payment_max_topup_amount")
	}
	if !equalPaymentSubscriptionPlans(before.PaymentSubscriptionPlans, after.PaymentSubscriptionPlans) {
		changed = append(changed, "payment_subscription_plans")
	}
	if before.BillingCurrencyConversionEnabled != after.BillingCurrencyConversionEnabled {
		changed = append(changed, "billing_currency_conversion_enabled")
	}
	if before.BillingCurrencyCNYToUSDRate != after.BillingCurrencyCNYToUSDRate {
		changed = append(changed, "billing_currency_cny_to_usd_rate")
	}
	if before.BillingCurrencyUSDToCNYRate != after.BillingCurrencyUSDToCNYRate {
		changed = append(changed, "billing_currency_usd_to_cny_rate")
	}
	if before.AntigravityUserAgentVersion != after.AntigravityUserAgentVersion {
		changed = append(changed, "antigravity_user_agent_version")
	}
	if before.CodexOAuthUserAgentMode != after.CodexOAuthUserAgentMode {
		changed = append(changed, "codex_oauth_user_agent_mode")
	}
	if before.CodexOAuthUserAgentOverride != after.CodexOAuthUserAgentOverride {
		changed = append(changed, "codex_oauth_user_agent_override")
	}
	if before.OpenAIAllowClaudeCodeCodexPlugin != after.OpenAIAllowClaudeCodeCodexPlugin {
		changed = append(changed, "openai_allow_claude_code_codex_plugin")
	}
	if !equalStringSlice(before.OpenAIAllowedCodexClients, after.OpenAIAllowedCodexClients) {
		changed = append(changed, "openai_allowed_codex_clients")
	}
	if before.LoginAgreementEnabled != after.LoginAgreementEnabled {
		changed = append(changed, "login_agreement_enabled")
	}
	if before.LoginAgreementMode != after.LoginAgreementMode {
		changed = append(changed, "login_agreement_mode")
	}
	if before.LoginAgreementUpdatedAt != after.LoginAgreementUpdatedAt {
		changed = append(changed, "login_agreement_updated_at")
	}
	if !equalLoginAgreementDocuments(before.LoginAgreementDocuments, after.LoginAgreementDocuments) {
		changed = append(changed, "login_agreement_documents")
	}
	if before.AffiliateEnabled != after.AffiliateEnabled {
		changed = append(changed, "affiliate_enabled")
	}
	if before.AffiliateTransferEnabled != after.AffiliateTransferEnabled {
		changed = append(changed, "affiliate_transfer_enabled")
	}
	if before.AffiliateRebateOnUsageEnabled != after.AffiliateRebateOnUsageEnabled {
		changed = append(changed, "affiliate_rebate_on_usage_enabled")
	}
	if before.AffiliateRebateOnTopupEnabled != after.AffiliateRebateOnTopupEnabled {
		changed = append(changed, "affiliate_rebate_on_topup_enabled")
	}
	if before.AffiliateRebateRate != after.AffiliateRebateRate {
		changed = append(changed, "affiliate_rebate_rate")
	}
	if before.AffiliateRebateFreezeHours != after.AffiliateRebateFreezeHours {
		changed = append(changed, "affiliate_rebate_freeze_hours")
	}
	if before.AffiliateRebateDurationDays != after.AffiliateRebateDurationDays {
		changed = append(changed, "affiliate_rebate_duration_days")
	}
	if before.AffiliateRebatePerInviteeCap != after.AffiliateRebatePerInviteeCap {
		changed = append(changed, "affiliate_rebate_per_invitee_cap")
	}
	if before.AffiliateAffCodeLength != after.AffiliateAffCodeLength {
		changed = append(changed, "affiliate_aff_code_length")
	}
	if before.CustomMenuItems != after.CustomMenuItems {
		changed = append(changed, "custom_menu_items")
	}
	return changed
}

func buildSystemSettingsDTO(settingService *service.SettingService, settings *service.SystemSettings, customMenuItems []dto.CustomMenuItem) dto.SystemSettings {
	defaultSubscriptions := make([]dto.DefaultSubscriptionSetting, 0, len(settings.DefaultSubscriptions))
	for _, sub := range settings.DefaultSubscriptions {
		defaultSubscriptions = append(defaultSubscriptions, dto.DefaultSubscriptionSetting{
			GroupID:      sub.GroupID,
			ValidityDays: sub.ValidityDays,
		})
	}

	var openAIFastPolicy *dto.OpenAIFastPolicySettings
	if settings.OpenAIFastPolicySettings != nil {
		rules := make([]dto.OpenAIFastPolicyRule, 0, len(settings.OpenAIFastPolicySettings.Rules))
		for _, r := range settings.OpenAIFastPolicySettings.Rules {
			rules = append(rules, dto.OpenAIFastPolicyRule{
				ServiceTier:    r.ServiceTier,
				Action:         r.Action,
				Scope:          r.Scope,
				ModelWhitelist: append([]string(nil), r.ModelWhitelist...),
				FallbackAction: r.FallbackAction,
			})
		}
		openAIFastPolicy = &dto.OpenAIFastPolicySettings{Rules: rules}
	}

	return dto.SystemSettings{
		RegistrationEnabled:                  settings.RegistrationEnabled,
		EmailVerifyEnabled:                   settings.EmailVerifyEnabled,
		RegistrationEmailSuffixWhitelist:     settings.RegistrationEmailSuffixWhitelist,
		PromoCodeEnabled:                     settings.PromoCodeEnabled,
		PasswordResetEnabled:                 settings.PasswordResetEnabled,
		FrontendURL:                          settings.FrontendURL,
		InvitationCodeEnabled:                settings.InvitationCodeEnabled,
		TotpEnabled:                          settings.TotpEnabled,
		TotpEncryptionKeyConfigured:          settingService.IsTotpEncryptionKeyConfigured(),
		SMTPHost:                             settings.SMTPHost,
		SMTPPort:                             settings.SMTPPort,
		SMTPUsername:                         settings.SMTPUsername,
		SMTPPasswordConfigured:               settings.SMTPPasswordConfigured,
		SMTPFrom:                             settings.SMTPFrom,
		SMTPFromName:                         settings.SMTPFromName,
		SMTPUseTLS:                           settings.SMTPUseTLS,
		TelegramChatID:                       settings.TelegramChatID,
		TelegramBotTokenConfigured:           settings.TelegramBotTokenConfigured,
		TelegramBotTokenMasked:               settings.TelegramBotTokenMasked,
		TurnstileEnabled:                     settings.TurnstileEnabled,
		TurnstileSiteKey:                     settings.TurnstileSiteKey,
		TurnstileSecretKeyConfigured:         settings.TurnstileSecretKeyConfigured,
		LinuxDoConnectEnabled:                settings.LinuxDoConnectEnabled,
		LinuxDoConnectClientID:               settings.LinuxDoConnectClientID,
		LinuxDoConnectClientSecretConfigured: settings.LinuxDoConnectClientSecretConfigured,
		LinuxDoConnectRedirectURL:            settings.LinuxDoConnectRedirectURL,
		GitHubOAuthEnabled:                   settings.GitHubOAuthEnabled,
		GitHubOAuthClientID:                  settings.GitHubOAuthClientID,
		GitHubOAuthClientSecretConfigured:    settings.GitHubOAuthClientSecretConfigured,
		GitHubOAuthRedirectURL:               settings.GitHubOAuthRedirectURL,
		GoogleOAuthEnabled:                   settings.GoogleOAuthEnabled,
		GoogleOAuthClientID:                  settings.GoogleOAuthClientID,
		GoogleOAuthClientSecretConfigured:    settings.GoogleOAuthClientSecretConfigured,
		GoogleOAuthRedirectURL:               settings.GoogleOAuthRedirectURL,
		DingTalkOAuthEnabled:                 settings.DingTalkOAuthEnabled,
		DingTalkOAuthClientID:                settings.DingTalkOAuthClientID,
		DingTalkOAuthClientSecretConfigured:  settings.DingTalkOAuthClientSecretConfigured,
		DingTalkOAuthRedirectURL:             settings.DingTalkOAuthRedirectURL,
		ContentModerationEnabled:             settings.ContentModerationEnabled,
		ContentModerationProvider:            settings.ContentModerationProvider,
		ContentModerationBaseURL:             settings.ContentModerationBaseURL,
		ContentModerationAPIKeyConfigured:    settings.ContentModerationAPIKeyConfigured,
		ContentModerationAPIKeyStatuses:      buildContentModerationAPIKeyStatusDTOs(settings.ContentModerationAPIKeyStatuses),
		ContentModerationModel:               settings.ContentModerationModel,
		ContentModerationTimeoutMs:           settings.ContentModerationTimeoutMs,
		ContentModerationDedupeWindowSeconds: settings.ContentModerationDedupeWindowSeconds,
		ContentModerationFailOpen:            settings.ContentModerationFailOpen,
		ContentModerationKeywordBlockEnabled: settings.ContentModerationKeywordBlockEnabled,
		ContentModerationKeywords:            settings.ContentModerationKeywords,
		ContentModerationModelFilter:         settings.ContentModerationModelFilter,
		ContentModerationCategoryThresholds:  settings.ContentModerationCategoryThresholds,
		ContentModerationCyberPolicyEnabled:  settings.ContentModerationCyberPolicyEnabled,
		ContentModerationCyberCategories:     settings.ContentModerationCyberCategories,
		SiteName:                             settings.SiteName,
		SiteLogo:                             settings.SiteLogo,
		SiteSubtitle:                         settings.SiteSubtitle,
		VisualPresetDefault:                  settings.VisualPresetDefault,
		AccountAiryWhiteSurfaceEnabled:       settings.AccountAiryWhiteSurfaceEnabled,
		APIBaseURL:                           settings.APIBaseURL,
		ContactInfo:                          settings.ContactInfo,
		DocURL:                               settings.DocURL,
		HomeContent:                          settings.HomeContent,
		HideCcsImportButton:                  settings.HideCcsImportButton,
		AvailableChannelsEnabled:             settings.AvailableChannelsEnabled,
		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,
		PublicModelCatalogEnabled:            settings.PublicModelCatalogEnabled,
		PurchaseSubscriptionEnabled:          settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:              settings.PurchaseSubscriptionURL,
		PaymentProviderAirwallexEnabled:      settings.PaymentProviderAirwallexEnabled,
		PaymentProviderAirwallexEffective:    settings.PaymentProviderAirwallexEffective,
		AirwallexEnv:                         settings.AirwallexEnv,
		AirwallexClientID:                    settings.AirwallexClientID,
		AirwallexAPIKeyConfigured:            settings.AirwallexAPIKeyConfigured,
		AirwallexWebhookSecretConfigured:     settings.AirwallexWebhookSecretConfigured,
		PaymentMobileForceQRCodeEnabled:      settings.PaymentMobileForceQRCodeEnabled,
		PaymentAllowedCurrencies:             settings.PaymentAllowedCurrencies,
		PaymentDefaultCurrency:               settings.PaymentDefaultCurrency,
		PaymentMinTopupAmount:                settings.PaymentMinTopupAmount,
		PaymentMaxTopupAmount:                settings.PaymentMaxTopupAmount,
		PaymentSubscriptionPlans:             buildPaymentSubscriptionPlanDTOs(settings.PaymentSubscriptionPlans),
		BillingCurrencyConversionEnabled:     settings.BillingCurrencyConversionEnabled,
		BillingCurrencyCNYToUSDRate:          settings.BillingCurrencyCNYToUSDRate,
		BillingCurrencyUSDToCNYRate:          settings.BillingCurrencyUSDToCNYRate,
		AntigravityUserAgentVersion:          settings.AntigravityUserAgentVersion,
		CodexOAuthUserAgentMode:              settings.CodexOAuthUserAgentMode,
		CodexOAuthUserAgentOverride:          settings.CodexOAuthUserAgentOverride,
		OpenAIAllowClaudeCodeCodexPlugin:     settings.OpenAIAllowClaudeCodeCodexPlugin,
		OpenAIAllowedCodexClients:            cloneStringSliceForJSON(settings.OpenAIAllowedCodexClients),
		CustomMenuItems:                      customMenuItems,
		LoginAgreementEnabled:                settings.LoginAgreementEnabled,
		LoginAgreementMode:                   settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:              settings.LoginAgreementUpdatedAt,
		LoginAgreementDocuments:              buildLoginAgreementDocumentDTOs(settings.LoginAgreementDocuments),
		AffiliateEnabled:                     settings.AffiliateEnabled,
		AffiliateTransferEnabled:             settings.AffiliateTransferEnabled,
		AffiliateRebateOnUsageEnabled:        settings.AffiliateRebateOnUsageEnabled,
		AffiliateRebateOnTopupEnabled:        settings.AffiliateRebateOnTopupEnabled,
		AffiliateRebateRate:                  settings.AffiliateRebateRate,
		AffiliateRebateFreezeHours:           settings.AffiliateRebateFreezeHours,
		AffiliateRebateDurationDays:          settings.AffiliateRebateDurationDays,
		AffiliateRebatePerInviteeCap:         settings.AffiliateRebatePerInviteeCap,
		AffiliateAffCodeLength:               settings.AffiliateAffCodeLength,
		DefaultConcurrency:                   settings.DefaultConcurrency,
		DefaultBalance:                       settings.DefaultBalance,
		DefaultSubscriptions:                 defaultSubscriptions,
		EnableModelFallback:                  settings.EnableModelFallback,
		FallbackModelAnthropic:               settings.FallbackModelAnthropic,
		FallbackModelOpenAI:                  settings.FallbackModelOpenAI,
		FallbackModelGemini:                  settings.FallbackModelGemini,
		FallbackModelAntigravity:             settings.FallbackModelAntigravity,
		EnableIdentityPatch:                  settings.EnableIdentityPatch,
		IdentityPatchPrompt:                  settings.IdentityPatchPrompt,
		ClaudeOAuthSystemPromptBlocksEnabled: settings.ClaudeOAuthSystemPromptBlocksEnabled,
		ClaudeOAuthSystemPromptBlocks:        settings.ClaudeOAuthSystemPromptBlocks,
		OpsMonitoringEnabled:                 settings.OpsMonitoringEnabled,
		OpsRealtimeMonitoringEnabled:         settings.OpsRealtimeMonitoringEnabled,
		OpsQueryModeDefault:                  settings.OpsQueryModeDefault,
		OpsMetricsIntervalSeconds:            settings.OpsMetricsIntervalSeconds,
		MinClaudeCodeVersion:                 settings.MinClaudeCodeVersion,
		MaxClaudeCodeVersion:                 settings.MaxClaudeCodeVersion,
		AllowUngroupedKeyScheduling:          settings.AllowUngroupedKeyScheduling,
		BackendModeEnabled:                   settings.BackendModeEnabled,
		MaintenanceModeEnabled:               settings.MaintenanceModeEnabled,
		AdminComplianceEnabled:               settings.AdminComplianceEnabled,
		OpenAIFastPolicySettings:             openAIFastPolicy,
		EnableAnthropicCacheTTL1hInjection:   settings.EnableAnthropicCacheTTL1hInjection,
	}
}
func normalizeDefaultSubscriptions(input []dto.DefaultSubscriptionSetting) []dto.DefaultSubscriptionSetting {
	if len(input) == 0 {
		return nil
	}
	normalized := make([]dto.DefaultSubscriptionSetting, 0, len(input))
	for _, item := range input {
		if item.GroupID <= 0 || item.ValidityDays <= 0 {
			continue
		}
		if item.ValidityDays > service.MaxValidityDays {
			item.ValidityDays = service.MaxValidityDays
		}
		normalized = append(normalized, item)
	}
	return normalized
}
func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func cloneStringSliceForJSON(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}

func equalFloatMap(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func equalContentModerationCyberCategories(a, b []service.ContentModerationCyberCategory) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID || !equalStringSlice(a[i].Keywords, b[i].Keywords) {
			return false
		}
	}
	return true
}

func equalDefaultSubscriptions(a, b []service.DefaultSubscriptionSetting) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].GroupID != b[i].GroupID || a[i].ValidityDays != b[i].ValidityDays {
			return false
		}
	}
	return true
}

func equalPaymentSubscriptionPlans(a, b []service.PaymentSubscriptionPlan) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].PlanID != b[i].PlanID ||
			a[i].Name != b[i].Name ||
			a[i].GroupID != b[i].GroupID ||
			a[i].ValidityDays != b[i].ValidityDays ||
			a[i].Enabled != b[i].Enabled ||
			len(a[i].PricesByCurrency) != len(b[i].PricesByCurrency) {
			return false
		}
		for currency, price := range a[i].PricesByCurrency {
			if b[i].PricesByCurrency[currency] != price {
				return false
			}
		}
	}
	return true
}

func buildPaymentSubscriptionPlanDTOs(items []service.PaymentSubscriptionPlan) []dto.PaymentSubscriptionPlan {
	out := make([]dto.PaymentSubscriptionPlan, 0, len(items))
	for _, item := range items {
		out = append(out, dto.PaymentSubscriptionPlan{
			PlanID:           item.PlanID,
			Name:             item.Name,
			GroupID:          item.GroupID,
			ValidityDays:     item.ValidityDays,
			PricesByCurrency: item.PricesByCurrency,
			Enabled:          item.Enabled,
		})
	}
	return out
}

func equalLoginAgreementDocuments(a, b []service.LoginAgreementDocument) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Title != b[i].Title || a[i].PageSlug != b[i].PageSlug {
			return false
		}
	}
	return true
}

func buildContentModerationAPIKeyStatusDTOs(items []service.ContentModerationAPIKeyStatus) []dto.ContentModerationAPIKeyStatus {
	out := make([]dto.ContentModerationAPIKeyStatus, 0, len(items))
	for _, item := range items {
		out = append(out, dto.ContentModerationAPIKeyStatus{
			Hash:        item.Hash,
			Masked:      item.Masked,
			FrozenUntil: item.FrozenUntil,
			LastError:   item.LastError,
		})
	}
	return out
}

func buildLoginAgreementDocumentDTOs(items []service.LoginAgreementDocument) []dto.LoginAgreementDocument {
	out := make([]dto.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		out = append(out, dto.LoginAgreementDocument{
			ID:       item.ID,
			Title:    item.Title,
			PageSlug: item.PageSlug,
		})
	}
	return out
}

func (h *SettingHandler) SendTestEmail(c *gin.Context) {
	var req SendTestEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.SMTPPort <= 0 {
		req.SMTPPort = 587
	}
	password := req.SMTPPassword
	if password == "" {
		savedConfig, err := h.emailService.GetSMTPConfig(c.Request.Context())
		if err == nil && savedConfig != nil {
			password = savedConfig.Password
		}
	}
	config := &service.SMTPConfig{Host: req.SMTPHost, Port: req.SMTPPort, Username: req.SMTPUsername, Password: password, From: req.SMTPFrom, FromName: req.SMTPFromName, UseTLS: req.SMTPUseTLS}
	siteName := h.settingService.GetSiteName(c.Request.Context())
	subject := "[" + siteName + "] Test Email"
	body := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background-color: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; }
        .content { padding: 40px 30px; text-align: center; }
        .success { color: #10b981; font-size: 48px; margin-bottom: 20px; }
        .footer { background-color: #f8f9fa; padding: 20px; text-align: center; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>` + siteName + `</h1>
        </div>
        <div class="content">
            <div class="success">✓</div>
            <h2>Email Configuration Successful!</h2>
            <p>This is a test email to verify your SMTP settings are working correctly.</p>
        </div>
        <div class="footer">
            <p>This is an automated test message.</p>
        </div>
    </div>
</body>
</html>
`
	if err := h.emailService.SendEmailWithConfig(config, req.Email, subject, body); err != nil {
		response.BadRequest(c, "Failed to send test email: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Test email sent successfully"})
}
func (h *SettingHandler) DeleteAdminAPIKey(c *gin.Context) {
	if err := h.settingService.DeleteAdminAPIKey(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Admin API key deleted"})
}
func (h *SettingHandler) GetStreamTimeoutSettings(c *gin.Context) {
	settings, err := h.settingService.GetStreamTimeoutSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.StreamTimeoutSettings{Enabled: settings.Enabled, Action: settings.Action, TempUnschedMinutes: settings.TempUnschedMinutes, ThresholdCount: settings.ThresholdCount, ThresholdWindowMinutes: settings.ThresholdWindowMinutes})
}
func (h *SettingHandler) UpdateBetaPolicySettings(c *gin.Context) {
	var req UpdateBetaPolicySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	rules := make([]service.BetaPolicyRule, len(req.Rules))
	for i, r := range req.Rules {
		rules[i] = service.BetaPolicyRule(r)
	}
	settings := &service.BetaPolicySettings{Rules: rules}
	if err := h.settingService.SetBetaPolicySettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updated, err := h.settingService.GetBetaPolicySettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	outRules := make([]dto.BetaPolicyRule, len(updated.Rules))
	for i, r := range updated.Rules {
		outRules[i] = dto.BetaPolicyRule(r)
	}
	response.Success(c, dto.BetaPolicySettings{Rules: outRules})
}
func (h *SettingHandler) UpdateStreamTimeoutSettings(c *gin.Context) {
	var req UpdateStreamTimeoutSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settings := &service.StreamTimeoutSettings{Enabled: req.Enabled, Action: req.Action, TempUnschedMinutes: req.TempUnschedMinutes, ThresholdCount: req.ThresholdCount, ThresholdWindowMinutes: req.ThresholdWindowMinutes}
	if err := h.settingService.SetStreamTimeoutSettings(c.Request.Context(), settings); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updatedSettings, err := h.settingService.GetStreamTimeoutSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.StreamTimeoutSettings{Enabled: updatedSettings.Enabled, Action: updatedSettings.Action, TempUnschedMinutes: updatedSettings.TempUnschedMinutes, ThresholdCount: updatedSettings.ThresholdCount, ThresholdWindowMinutes: updatedSettings.ThresholdWindowMinutes})
}
