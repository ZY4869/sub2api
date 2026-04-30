package dto

import (
	"encoding/json"
	"strings"
)

// CustomMenuItem represents a user-configured custom menu entry.
type CustomMenuItem struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	IconSVG    string `json:"icon_svg"`
	URL        string `json:"url"`
	Visibility string `json:"visibility"` // "user" or "admin"
	SortOrder  int    `json:"sort_order"`
}

// SystemSettings represents the admin settings API response payload.
type SystemSettings struct {
	RegistrationEnabled              bool     `json:"registration_enabled"`
	EmailVerifyEnabled               bool     `json:"email_verify_enabled"`
	RegistrationEmailSuffixWhitelist []string `json:"registration_email_suffix_whitelist"`
	PromoCodeEnabled                 bool     `json:"promo_code_enabled"`
	PasswordResetEnabled             bool     `json:"password_reset_enabled"`
	FrontendURL                      string   `json:"frontend_url"`
	InvitationCodeEnabled            bool     `json:"invitation_code_enabled"`
	TotpEnabled                      bool     `json:"totp_enabled"`                   // TOTP 双因素认证
	TotpEncryptionKeyConfigured      bool     `json:"totp_encryption_key_configured"` // TOTP 加密密钥是否已配置

	SMTPHost                   string `json:"smtp_host"`
	SMTPPort                   int    `json:"smtp_port"`
	SMTPUsername               string `json:"smtp_username"`
	SMTPPasswordConfigured     bool   `json:"smtp_password_configured"`
	SMTPFrom                   string `json:"smtp_from_email"`
	SMTPFromName               string `json:"smtp_from_name"`
	SMTPUseTLS                 bool   `json:"smtp_use_tls"`
	TelegramChatID             string `json:"telegram_chat_id"`
	TelegramBotTokenConfigured bool   `json:"telegram_bot_token_configured"`
	TelegramBotTokenMasked     string `json:"telegram_bot_token_masked"`

	TurnstileEnabled             bool   `json:"turnstile_enabled"`
	TurnstileSiteKey             string `json:"turnstile_site_key"`
	TurnstileSecretKeyConfigured bool   `json:"turnstile_secret_key_configured"`

	LinuxDoConnectEnabled                bool   `json:"linuxdo_connect_enabled"`
	LinuxDoConnectClientID               string `json:"linuxdo_connect_client_id"`
	LinuxDoConnectClientSecretConfigured bool   `json:"linuxdo_connect_client_secret_configured"`
	LinuxDoConnectRedirectURL            string `json:"linuxdo_connect_redirect_url"`

	SiteName                             string           `json:"site_name"`
	SiteLogo                             string           `json:"site_logo"`
	SiteSubtitle                         string           `json:"site_subtitle"`
	APIBaseURL                           string           `json:"api_base_url"`
	ContactInfo                          string           `json:"contact_info"`
	DocURL                               string           `json:"doc_url"`
	HomeContent                          string           `json:"home_content"`
	HideCcsImportButton                  bool             `json:"hide_ccs_import_button"`
	AvailableChannelsEnabled             bool             `json:"available_channels_enabled"`
	ChannelMonitorEnabled                bool             `json:"channel_monitor_enabled"`
	ChannelMonitorDefaultIntervalSeconds int              `json:"channel_monitor_default_interval_seconds"`
	PublicModelCatalogEnabled            bool             `json:"public_model_catalog_enabled"`
	PurchaseSubscriptionEnabled          bool             `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL              string           `json:"purchase_subscription_url"`
	CustomMenuItems                      []CustomMenuItem `json:"custom_menu_items"`

	AffiliateEnabled              bool    `json:"affiliate_enabled"`
	AffiliateTransferEnabled      bool    `json:"affiliate_transfer_enabled"`
	AffiliateRebateOnUsageEnabled bool    `json:"affiliate_rebate_on_usage_enabled"`
	AffiliateRebateOnTopupEnabled bool    `json:"affiliate_rebate_on_topup_enabled"`
	AffiliateRebateRate           float64 `json:"affiliate_rebate_rate"`
	AffiliateRebateFreezeHours    int     `json:"affiliate_rebate_freeze_hours"`
	AffiliateRebateDurationDays   int     `json:"affiliate_rebate_duration_days"`
	AffiliateRebatePerInviteeCap  float64 `json:"affiliate_rebate_per_invitee_cap"`
	AffiliateAffCodeLength        int     `json:"affiliate_aff_code_length"`

	DefaultConcurrency   int                          `json:"default_concurrency"`
	DefaultBalance       float64                      `json:"default_balance"`
	DefaultSubscriptions []DefaultSubscriptionSetting `json:"default_subscriptions"`

	// Model fallback configuration
	EnableModelFallback      bool   `json:"enable_model_fallback"`
	FallbackModelAnthropic   string `json:"fallback_model_anthropic"`
	FallbackModelOpenAI      string `json:"fallback_model_openai"`
	FallbackModelGemini      string `json:"fallback_model_gemini"`
	FallbackModelAntigravity string `json:"fallback_model_antigravity"`

	// Identity patch configuration (Claude -> Gemini)
	EnableIdentityPatch bool   `json:"enable_identity_patch"`
	IdentityPatchPrompt string `json:"identity_patch_prompt"`

	// Ops monitoring (vNext)
	OpsMonitoringEnabled         bool   `json:"ops_monitoring_enabled"`
	OpsRealtimeMonitoringEnabled bool   `json:"ops_realtime_monitoring_enabled"`
	OpsQueryModeDefault          string `json:"ops_query_mode_default"`
	OpsMetricsIntervalSeconds    int    `json:"ops_metrics_interval_seconds"`

	MinClaudeCodeVersion string `json:"min_claude_code_version"`
	MaxClaudeCodeVersion string `json:"max_claude_code_version"`

	// 分组隔离
	AllowUngroupedKeyScheduling bool `json:"allow_ungrouped_key_scheduling"`

	// Backend Mode
	BackendModeEnabled     bool `json:"backend_mode_enabled"`
	MaintenanceModeEnabled bool `json:"maintenance_mode_enabled"`

	OpenAIFastPolicySettings           *OpenAIFastPolicySettings `json:"openai_fast_policy_settings,omitempty"`
	EnableAnthropicCacheTTL1hInjection bool                      `json:"enable_anthropic_cache_ttl_1h_injection"`
}

type DefaultSubscriptionSetting struct {
	GroupID      int64 `json:"group_id"`
	ValidityDays int   `json:"validity_days"`
}

type PublicSettings struct {
	RegistrationEnabled              bool             `json:"registration_enabled"`
	EmailVerifyEnabled               bool             `json:"email_verify_enabled"`
	RegistrationEmailSuffixWhitelist []string         `json:"registration_email_suffix_whitelist"`
	PromoCodeEnabled                 bool             `json:"promo_code_enabled"`
	PasswordResetEnabled             bool             `json:"password_reset_enabled"`
	InvitationCodeEnabled            bool             `json:"invitation_code_enabled"`
	TotpEnabled                      bool             `json:"totp_enabled"` // TOTP 双因素认证
	TurnstileEnabled                 bool             `json:"turnstile_enabled"`
	TurnstileSiteKey                 string           `json:"turnstile_site_key"`
	SiteName                         string           `json:"site_name"`
	SiteLogo                         string           `json:"site_logo"`
	SiteSubtitle                     string           `json:"site_subtitle"`
	APIBaseURL                       string           `json:"api_base_url"`
	ContactInfo                      string           `json:"contact_info"`
	DocURL                           string           `json:"doc_url"`
	HomeContent                      string           `json:"home_content"`
	HideCcsImportButton              bool             `json:"hide_ccs_import_button"`
	AvailableChannelsEnabled         bool             `json:"available_channels_enabled"`
	ChannelMonitorEnabled            bool             `json:"channel_monitor_enabled"`
	PublicModelCatalogEnabled        bool             `json:"public_model_catalog_enabled"`
	AffiliateEnabled                 bool             `json:"affiliate_enabled"`
	PurchaseSubscriptionEnabled      bool             `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL          string           `json:"purchase_subscription_url"`
	CustomMenuItems                  []CustomMenuItem `json:"custom_menu_items"`
	LinuxDoOAuthEnabled              bool             `json:"linuxdo_oauth_enabled"`
	BackendModeEnabled               bool             `json:"backend_mode_enabled"`
	MaintenanceModeEnabled           bool             `json:"maintenance_mode_enabled"`
	Version                          string           `json:"version"`
}

type GoogleBatchGCSProfile struct {
	ProfileID                    string `json:"profile_id"`
	Name                         string `json:"name"`
	IsActive                     bool   `json:"is_active"`
	Enabled                      bool   `json:"enabled"`
	Bucket                       string `json:"bucket"`
	Prefix                       string `json:"prefix"`
	ProjectID                    string `json:"project_id"`
	ServiceAccountJSONConfigured bool   `json:"service_account_json_configured"`
	UpdatedAt                    string `json:"updated_at"`
}

type ListGoogleBatchGCSProfilesResponse struct {
	ActiveProfileID string                  `json:"active_profile_id"`
	Items           []GoogleBatchGCSProfile `json:"items"`
}

type GoogleBatchArchiveSettings struct {
	Enabled                bool   `json:"enabled"`
	PollMinIntervalSeconds int    `json:"poll_min_interval_seconds"`
	PollMaxIntervalSeconds int    `json:"poll_max_interval_seconds"`
	PollBackoffFactor      int    `json:"poll_backoff_factor"`
	PollJitterSeconds      int    `json:"poll_jitter_seconds"`
	PollMaxConcurrency     int    `json:"poll_max_concurrency"`
	PrefetchAfterHours     int    `json:"prefetch_after_hours"`
	DownloadTimeoutSeconds int    `json:"download_timeout_seconds"`
	CleanupIntervalMinutes int    `json:"cleanup_interval_minutes"`
	LocalStorageRoot       string `json:"local_storage_root"`
}

type GeminiRateCatalog struct {
	EffectiveDate              string                       `json:"effective_date"`
	RemainingQuotaAPISupported bool                         `json:"remaining_quota_api_supported"`
	AIStudioTiers              []GeminiRateCatalogTier      `json:"ai_studio_tiers"`
	BatchLimits                GeminiRateCatalogBatchLimits `json:"batch_limits"`
	Links                      []GeminiRateCatalogLink      `json:"links"`
	Notes                      []string                     `json:"notes"`
}

type GeminiRateCatalogTier struct {
	TierID         string                      `json:"tier_id"`
	DisplayName    string                      `json:"display_name"`
	Qualification  string                      `json:"qualification"`
	BillingTierCap string                      `json:"billing_tier_cap"`
	ModelFamilies  []GeminiRateCatalogModelRow `json:"model_families"`
}

type GeminiRateCatalogModelRow struct {
	ModelFamily string `json:"model_family"`
	DisplayName string `json:"display_name"`
	RPM         int64  `json:"rpm"`
	TPM         int64  `json:"tpm"`
	RPD         int64  `json:"rpd"`
	Notes       string `json:"notes,omitempty"`
}

type GeminiRateCatalogBatchLimits struct {
	ConcurrentBatchRequests int64                        `json:"concurrent_batch_requests"`
	InputFileSizeLimitBytes int64                        `json:"input_file_size_limit_bytes"`
	FileStorageLimitBytes   int64                        `json:"file_storage_limit_bytes"`
	ByTier                  []GeminiRateCatalogBatchTier `json:"by_tier"`
}

type GeminiRateCatalogBatchTier struct {
	TierID  string                      `json:"tier_id"`
	Entries []GeminiRateCatalogBatchRow `json:"entries"`
}

type GeminiRateCatalogBatchRow struct {
	ModelFamily    string `json:"model_family"`
	DisplayName    string `json:"display_name"`
	EnqueuedTokens int64  `json:"enqueued_tokens"`
}

type GeminiRateCatalogLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// OverloadCooldownSettings 529过载冷却配置 DTO
type OverloadCooldownSettings struct {
	Enabled         bool `json:"enabled"`
	CooldownMinutes int  `json:"cooldown_minutes"`
}

// StreamTimeoutSettings 流超时处理配置 DTO
type StreamTimeoutSettings struct {
	Enabled                bool   `json:"enabled"`
	Action                 string `json:"action"`
	TempUnschedMinutes     int    `json:"temp_unsched_minutes"`
	ThresholdCount         int    `json:"threshold_count"`
	ThresholdWindowMinutes int    `json:"threshold_window_minutes"`
}

// RectifierSettings 请求整流器配置 DTO
type RectifierSettings struct {
	Enabled                  bool `json:"enabled"`
	ThinkingSignatureEnabled bool `json:"thinking_signature_enabled"`
	ThinkingBudgetEnabled    bool `json:"thinking_budget_enabled"`
}

// BetaPolicyRule Beta 策略规则 DTO
type BetaPolicyRule struct {
	BetaToken    string `json:"beta_token"`
	Action       string `json:"action"`
	Scope        string `json:"scope"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// BetaPolicySettings Beta 策略配置 DTO
type BetaPolicySettings struct {
	Rules []BetaPolicyRule `json:"rules"`
}

type OpenAIFastPolicyRule struct {
	ServiceTier    string   `json:"service_tier"`
	Action         string   `json:"action"`
	Scope          string   `json:"scope"`
	ModelWhitelist []string `json:"model_whitelist,omitempty"`
	FallbackAction string   `json:"fallback_action,omitempty"`
}

type OpenAIFastPolicySettings struct {
	Rules []OpenAIFastPolicyRule `json:"rules"`
}

// ParseCustomMenuItems parses a JSON string into a slice of CustomMenuItem.
// Returns empty slice on empty/invalid input.
func ParseCustomMenuItems(raw string) []CustomMenuItem {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []CustomMenuItem{}
	}
	var items []CustomMenuItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []CustomMenuItem{}
	}
	return items
}

// ParseUserVisibleMenuItems parses custom menu items and filters out admin-only entries.
func ParseUserVisibleMenuItems(raw string) []CustomMenuItem {
	items := ParseCustomMenuItems(raw)
	filtered := make([]CustomMenuItem, 0, len(items))
	for _, item := range items {
		if item.Visibility != "admin" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
