package service

import "github.com/Wei-Shaw/sub2api/internal/domain"

// Status constants.
const (
	StatusActive   = domain.StatusActive
	StatusDisabled = domain.StatusDisabled
	StatusError    = domain.StatusError
	StatusUnused   = domain.StatusUnused
	StatusUsed     = domain.StatusUsed
	StatusExpired  = domain.StatusExpired
)

// Role constants.
const (
	RoleAdmin = domain.RoleAdmin
	RoleUser  = domain.RoleUser
)

// Platform constants.
const (
	PlatformAnthropic       = domain.PlatformAnthropic
	PlatformOpenAI          = domain.PlatformOpenAI
	PlatformGemini          = domain.PlatformGemini
	PlatformProtocolGateway = domain.PlatformProtocolGateway
	PlatformAntigravity     = domain.PlatformAntigravity
	PlatformKiro            = domain.PlatformKiro
	PlatformCopilot         = domain.PlatformCopilot
	PlatformGrok            = domain.PlatformGrok
)

// Account type constants.
const (
	AccountTypeOAuth      = domain.AccountTypeOAuth
	AccountTypeSetupToken = domain.AccountTypeSetupToken
	AccountTypeAPIKey     = domain.AccountTypeAPIKey
	AccountTypeUpstream   = domain.AccountTypeUpstream
	AccountTypeBedrock    = domain.AccountTypeBedrock
	AccountTypeSSO        = domain.AccountTypeSSO
)

// Redeem type constants.
const (
	RedeemTypeBalance      = domain.RedeemTypeBalance
	RedeemTypeConcurrency  = domain.RedeemTypeConcurrency
	RedeemTypeSubscription = domain.RedeemTypeSubscription
	RedeemTypeInvitation   = domain.RedeemTypeInvitation
)

// PromoCode status constants.
const (
	PromoCodeStatusActive   = domain.PromoCodeStatusActive
	PromoCodeStatusDisabled = domain.PromoCodeStatusDisabled
)

// Admin adjustment type constants.
const (
	AdjustmentTypeAdminBalance     = domain.AdjustmentTypeAdminBalance
	AdjustmentTypeAdminConcurrency = domain.AdjustmentTypeAdminConcurrency
)

// Group subscription type constants.
const (
	SubscriptionTypeStandard     = domain.SubscriptionTypeStandard
	SubscriptionTypeSubscription = domain.SubscriptionTypeSubscription
)

// Subscription status constants.
const (
	SubscriptionStatusActive    = domain.SubscriptionStatusActive
	SubscriptionStatusExpired   = domain.SubscriptionStatusExpired
	SubscriptionStatusSuspended = domain.SubscriptionStatusSuspended
)

const LinuxDoConnectSyntheticEmailDomain = "@linuxdo-connect.invalid"

// Setting keys.
const (
	SettingKeyRegistrationEnabled              = "registration_enabled"
	SettingKeyEmailVerifyEnabled               = "email_verify_enabled"
	SettingKeyRegistrationEmailSuffixWhitelist = "registration_email_suffix_whitelist"
	SettingKeyPromoCodeEnabled                 = "promo_code_enabled"
	SettingKeyPasswordResetEnabled             = "password_reset_enabled"
	SettingKeyFrontendURL                      = "frontend_url"
	SettingKeyInvitationCodeEnabled            = "invitation_code_enabled"

	SettingKeySMTPHost     = "smtp_host"
	SettingKeySMTPPort     = "smtp_port"
	SettingKeySMTPUsername = "smtp_username"
	SettingKeySMTPPassword = "smtp_password"
	SettingKeySMTPFrom     = "smtp_from"
	SettingKeySMTPFromName = "smtp_from_name"
	SettingKeySMTPUseTLS   = "smtp_use_tls"

	SettingKeyTelegramChatID   = "telegram_chat_id"
	SettingKeyTelegramBotToken = "telegram_bot_token"

	SettingKeyTurnstileEnabled   = "turnstile_enabled"
	SettingKeyTurnstileSiteKey   = "turnstile_site_key"
	SettingKeyTurnstileSecretKey = "turnstile_secret_key"
	SettingKeyTotpEnabled        = "totp_enabled"

	SettingKeyLinuxDoConnectEnabled      = "linuxdo_connect_enabled"
	SettingKeyLinuxDoConnectClientID     = "linuxdo_connect_client_id"
	SettingKeyLinuxDoConnectClientSecret = "linuxdo_connect_client_secret"
	SettingKeyLinuxDoConnectRedirectURL  = "linuxdo_connect_redirect_url"

	SettingKeySiteName                    = "site_name"
	SettingKeySiteLogo                    = "site_logo"
	SettingKeySiteSubtitle                = "site_subtitle"
	SettingKeyAPIBaseURL                  = "api_base_url"
	SettingKeyContactInfo                 = "contact_info"
	SettingKeyDocURL                      = "doc_url"
	SettingKeyHomeContent                 = "home_content"
	SettingKeyHideCcsImportButton         = "hide_ccs_import_button"
	SettingKeyPurchaseSubscriptionEnabled = "purchase_subscription_enabled"
	SettingKeyPurchaseSubscriptionURL     = "purchase_subscription_url"
	SettingKeyCustomMenuItems             = "custom_menu_items"
	SettingKeyDefaultConcurrency          = "default_concurrency"
	SettingKeyDefaultBalance              = "default_balance"
	SettingKeyDefaultSubscriptions        = "default_subscriptions"
	SettingKeyAdminAPIKey                 = "admin_api_key"
	SettingKeyGeminiQuotaPolicy           = "gemini_quota_policy"

	SettingKeyEnableModelFallback                            = "enable_model_fallback"
	SettingKeyFallbackModelAnthropic                         = "fallback_model_anthropic"
	SettingKeyFallbackModelOpenAI                            = "fallback_model_openai"
	SettingKeyFallbackModelGemini                            = "fallback_model_gemini"
	SettingKeyFallbackModelAntigravity                       = "fallback_model_antigravity"
	SettingKeyModelCatalogEntries                            = "model_catalog_entries"
	SettingKeyModelPriceOverrides                            = "model_price_overrides"
	SettingKeyModelOfficialPriceOverrides                    = "model_official_price_overrides"
	SettingKeyModelPricingCurrencies                         = "model_pricing_currencies"
	SettingKeyBillingCenterRules                             = "billing_center_rules"
	SettingKeyBillingPricingCatalogSnapshot                  = "billing_pricing_catalog_snapshot"
	SettingKeyModelRegistryEntries                           = "model_registry_entries"
	SettingKeyModelRegistryHiddenModels                      = "model_registry_hidden_models"
	SettingKeyModelRegistryTombstones                        = "model_registry_tombstones"
	SettingKeyModelRegistryAvailableModels                   = "model_registry_available_models"
	SettingKeyModelRegistryAvailableModelsBootstrapV20260313 = "model_registry_available_models_bootstrap_v20260313"
	SettingKeyModelRegistryAvailableModelsBootstrapV20260317 = "model_registry_available_models_bootstrap_v20260317"
	SettingKeyModelRegistryAvailableModelsBootstrapV20260328 = "model_registry_available_models_bootstrap_v20260328"
	SettingKeyModelRegistryAvailableModelsBootstrapV20260416 = "model_registry_available_models_bootstrap_v20260416"
	SettingKeyModelRegistryAvailableModelsBootstrapV20260417 = "model_registry_available_models_bootstrap_v20260417"

	SettingKeyEnableIdentityPatch = "enable_identity_patch"
	SettingKeyIdentityPatchPrompt = "identity_patch_prompt"

	SettingKeyOpsMonitoringEnabled         = "ops_monitoring_enabled"
	SettingKeyOpsRealtimeMonitoringEnabled = "ops_realtime_monitoring_enabled"
	SettingKeyOpsQueryModeDefault          = "ops_query_mode_default"
	SettingKeyOpsEmailNotificationConfig   = "ops_email_notification_config"
	SettingKeyOpsAlertRuntimeSettings      = "ops_alert_runtime_settings"
	SettingKeyOpsMetricsIntervalSeconds    = "ops_metrics_interval_seconds"
	SettingKeyOpsAdvancedSettings          = "ops_advanced_settings"
	SettingKeyOpsRuntimeLogConfig          = "ops_runtime_log_config"

	SettingKeyOverloadCooldownSettings = "overload_cooldown_settings"
	SettingKeyStreamTimeoutSettings    = "stream_timeout_settings"
	SettingKeyRectifierSettings        = "rectifier_settings"
	SettingKeyBetaPolicySettings       = "beta_policy_settings"
	SettingKeyBlacklistRuleCandidates  = "blacklist_rule_candidates"

	SettingKeyGoogleBatchGCSProfiles = "google_batch_gcs_profiles"

	SettingKeyMinClaudeCodeVersion = "min_claude_code_version"
	SettingKeyMaxClaudeCodeVersion = "max_claude_code_version"

	SettingKeyAllowUngroupedKeyScheduling = "allow_ungrouped_key_scheduling"
	SettingKeyMultiGroupRoutingEnabled    = "multi_group_routing_enabled"
	SettingKeyBackendModeEnabled          = "backend_mode_enabled"
)

// AdminAPIKeyPrefix is the prefix for admin API keys.
const AdminAPIKeyPrefix = "admin-"
