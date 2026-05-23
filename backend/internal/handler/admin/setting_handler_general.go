package admin

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (h *SettingHandler) GetSettings(c *gin.Context) {
	settings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	opsEnabled := h.opsService != nil && h.opsService.IsMonitoringEnabled(c.Request.Context())
	payload := buildSystemSettingsDTO(h.settingService, settings, dto.ParseCustomMenuItems(settings.CustomMenuItems))
	payload.OpsMonitoringEnabled = opsEnabled && settings.OpsMonitoringEnabled
	response.Success(c, payload)
}

type UpdateSettingsRequest struct {
	RegistrationEnabled                  bool                                  `json:"registration_enabled"`
	EmailVerifyEnabled                   bool                                  `json:"email_verify_enabled"`
	RegistrationEmailSuffixWhitelist     []string                              `json:"registration_email_suffix_whitelist"`
	PromoCodeEnabled                     bool                                  `json:"promo_code_enabled"`
	PasswordResetEnabled                 bool                                  `json:"password_reset_enabled"`
	FrontendURL                          string                                `json:"frontend_url"`
	InvitationCodeEnabled                bool                                  `json:"invitation_code_enabled"`
	TotpEnabled                          bool                                  `json:"totp_enabled"`
	SMTPHost                             string                                `json:"smtp_host"`
	SMTPPort                             int                                   `json:"smtp_port"`
	SMTPUsername                         string                                `json:"smtp_username"`
	SMTPPassword                         string                                `json:"smtp_password"`
	SMTPFrom                             string                                `json:"smtp_from_email"`
	SMTPFromName                         string                                `json:"smtp_from_name"`
	SMTPUseTLS                           bool                                  `json:"smtp_use_tls"`
	TelegramChatID                       string                                `json:"telegram_chat_id"`
	TelegramBotToken                     string                                `json:"telegram_bot_token"`
	TurnstileEnabled                     bool                                  `json:"turnstile_enabled"`
	TurnstileSiteKey                     string                                `json:"turnstile_site_key"`
	TurnstileSecretKey                   string                                `json:"turnstile_secret_key"`
	LinuxDoConnectEnabled                bool                                  `json:"linuxdo_connect_enabled"`
	LinuxDoConnectClientID               string                                `json:"linuxdo_connect_client_id"`
	LinuxDoConnectClientSecret           string                                `json:"linuxdo_connect_client_secret"`
	LinuxDoConnectRedirectURL            string                                `json:"linuxdo_connect_redirect_url"`
	GitHubOAuthEnabled                   bool                                  `json:"github_oauth_enabled"`
	GitHubOAuthClientID                  string                                `json:"github_oauth_client_id"`
	GitHubOAuthClientSecret              string                                `json:"github_oauth_client_secret"`
	GitHubOAuthRedirectURL               string                                `json:"github_oauth_redirect_url"`
	GoogleOAuthEnabled                   bool                                  `json:"google_oauth_enabled"`
	GoogleOAuthClientID                  string                                `json:"google_oauth_client_id"`
	GoogleOAuthClientSecret              string                                `json:"google_oauth_client_secret"`
	GoogleOAuthRedirectURL               string                                `json:"google_oauth_redirect_url"`
	DingTalkOAuthEnabled                 bool                                  `json:"dingtalk_oauth_enabled"`
	DingTalkOAuthClientID                string                                `json:"dingtalk_oauth_client_id"`
	DingTalkOAuthClientSecret            string                                `json:"dingtalk_oauth_client_secret"`
	DingTalkOAuthRedirectURL             string                                `json:"dingtalk_oauth_redirect_url"`
	ContentModerationEnabled             bool                                  `json:"content_moderation_enabled"`
	ContentModerationProvider            string                                `json:"content_moderation_provider"`
	ContentModerationBaseURL             string                                `json:"content_moderation_base_url"`
	ContentModerationAPIKey              string                                `json:"content_moderation_api_key"`
	ContentModerationAPIKeys             []string                              `json:"content_moderation_api_keys"`
	ContentModerationAPIKeysMode         string                                `json:"content_moderation_api_keys_mode"`
	DeleteContentModerationAPIKeyHashes  []string                              `json:"delete_content_moderation_api_key_hashes"`
	ContentModerationModel               string                                `json:"content_moderation_model"`
	ContentModerationTimeoutMs           int                                   `json:"content_moderation_timeout_ms"`
	ContentModerationDedupeWindowSeconds int                                   `json:"content_moderation_dedupe_window_seconds"`
	ContentModerationFailOpen            *bool                                 `json:"content_moderation_fail_open"`
	ContentModerationKeywordBlockEnabled *bool                                 `json:"content_moderation_keyword_block_enabled"`
	ContentModerationKeywords            []string                              `json:"content_moderation_keywords"`
	ContentModerationModelFilter         *service.ContentModerationModelFilter `json:"content_moderation_model_filter"`
	SiteName                             string                                `json:"site_name"`
	SiteLogo                             string                                `json:"site_logo"`
	SiteSubtitle                         string                                `json:"site_subtitle"`
	VisualPresetDefault                  string                                `json:"visual_preset_default"`
	AccountAiryWhiteSurfaceEnabled       bool                                  `json:"account_airy_white_surface_enabled"`
	APIBaseURL                           string                                `json:"api_base_url"`
	ContactInfo                          string                                `json:"contact_info"`
	DocURL                               string                                `json:"doc_url"`
	HomeContent                          string                                `json:"home_content"`
	HideCcsImportButton                  bool                                  `json:"hide_ccs_import_button"`
	AvailableChannelsEnabled             *bool                                 `json:"available_channels_enabled"`
	ChannelMonitorEnabled                *bool                                 `json:"channel_monitor_enabled"`
	ChannelMonitorDefaultIntervalSeconds *int                                  `json:"channel_monitor_default_interval_seconds"`
	PublicModelCatalogEnabled            bool                                  `json:"public_model_catalog_enabled"`
	PurchaseSubscriptionEnabled          *bool                                 `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL              *string                               `json:"purchase_subscription_url"`
	PaymentProviderAirwallexEnabled      *bool                                 `json:"payment_provider_airwallex_enabled"`
	AirwallexEnv                         *string                               `json:"airwallex_env"`
	AirwallexClientID                    *string                               `json:"airwallex_client_id"`
	AirwallexAPIKey                      *string                               `json:"airwallex_api_key"`
	AirwallexWebhookSecret               *string                               `json:"airwallex_webhook_secret"`
	PaymentMobileForceQRCodeEnabled      *bool                                 `json:"payment_mobile_force_qrcode_enabled"`
	PaymentAllowedCurrencies             *[]string                             `json:"payment_allowed_currencies"`
	PaymentDefaultCurrency               *string                               `json:"payment_default_currency"`
	PaymentMinTopupAmount                *float64                              `json:"payment_min_topup_amount"`
	PaymentMaxTopupAmount                *float64                              `json:"payment_max_topup_amount"`
	PaymentSubscriptionPlans             *[]dto.PaymentSubscriptionPlan        `json:"payment_subscription_plans"`
	AntigravityUserAgentVersion          *string                               `json:"antigravity_user_agent_version"`
	CodexOAuthUserAgentMode              *string                               `json:"codex_oauth_user_agent_mode"`
	CodexOAuthUserAgentOverride          *string                               `json:"codex_oauth_user_agent_override"`
	CustomMenuItems                      *[]dto.CustomMenuItem                 `json:"custom_menu_items"`
	LoginAgreementEnabled                *bool                                 `json:"login_agreement_enabled"`
	LoginAgreementMode                   *string                               `json:"login_agreement_mode"`
	LoginAgreementUpdatedAt              *string                               `json:"login_agreement_updated_at"`
	LoginAgreementDocuments              *[]dto.LoginAgreementDocument         `json:"login_agreement_documents"`
	DefaultConcurrency                   int                                   `json:"default_concurrency"`
	DefaultBalance                       float64                               `json:"default_balance"`
	DefaultSubscriptions                 []dto.DefaultSubscriptionSetting      `json:"default_subscriptions"`
	EnableModelFallback                  bool                                  `json:"enable_model_fallback"`
	FallbackModelAnthropic               string                                `json:"fallback_model_anthropic"`
	FallbackModelOpenAI                  string                                `json:"fallback_model_openai"`
	FallbackModelGemini                  string                                `json:"fallback_model_gemini"`
	FallbackModelAntigravity             string                                `json:"fallback_model_antigravity"`
	EnableIdentityPatch                  bool                                  `json:"enable_identity_patch"`
	IdentityPatchPrompt                  string                                `json:"identity_patch_prompt"`
	OpsMonitoringEnabled                 *bool                                 `json:"ops_monitoring_enabled"`
	OpsRealtimeMonitoringEnabled         *bool                                 `json:"ops_realtime_monitoring_enabled"`
	OpsQueryModeDefault                  *string                               `json:"ops_query_mode_default"`
	OpsMetricsIntervalSeconds            *int                                  `json:"ops_metrics_interval_seconds"`
	MinClaudeCodeVersion                 string                                `json:"min_claude_code_version"`
	MaxClaudeCodeVersion                 string                                `json:"max_claude_code_version"`
	AllowUngroupedKeyScheduling          bool                                  `json:"allow_ungrouped_key_scheduling"`
	BackendModeEnabled                   bool                                  `json:"backend_mode_enabled"`
	MaintenanceModeEnabled               bool                                  `json:"maintenance_mode_enabled"`

	AffiliateEnabled              *bool    `json:"affiliate_enabled"`
	AffiliateTransferEnabled      *bool    `json:"affiliate_transfer_enabled"`
	AffiliateRebateOnUsageEnabled *bool    `json:"affiliate_rebate_on_usage_enabled"`
	AffiliateRebateOnTopupEnabled *bool    `json:"affiliate_rebate_on_topup_enabled"`
	AffiliateRebateRate           *float64 `json:"affiliate_rebate_rate"`
	AffiliateRebateFreezeHours    *int     `json:"affiliate_rebate_freeze_hours"`
	AffiliateRebateDurationDays   *int     `json:"affiliate_rebate_duration_days"`
	AffiliateRebatePerInviteeCap  *float64 `json:"affiliate_rebate_per_invitee_cap"`
	AffiliateAffCodeLength        *int     `json:"affiliate_aff_code_length"`

	OpenAIFastPolicySettings           *dto.OpenAIFastPolicySettings `json:"openai_fast_policy_settings"`
	EnableAnthropicCacheTTL1hInjection *bool                         `json:"enable_anthropic_cache_ttl_1h_injection"`
}

func (h *SettingHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	previousSettings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.DefaultConcurrency < 1 {
		req.DefaultConcurrency = 1
	}
	if req.DefaultBalance < 0 {
		req.DefaultBalance = 0
	}
	if req.SMTPPort <= 0 {
		req.SMTPPort = 587
	}
	req.DefaultSubscriptions = normalizeDefaultSubscriptions(req.DefaultSubscriptions)
	if req.TurnstileEnabled {
		if req.TurnstileSiteKey == "" {
			response.BadRequest(c, "Turnstile Site Key is required when enabled")
			return
		}
		if req.TurnstileSecretKey == "" {
			if previousSettings.TurnstileSecretKey == "" {
				response.BadRequest(c, "Turnstile Secret Key is required when enabled")
				return
			}
			req.TurnstileSecretKey = previousSettings.TurnstileSecretKey
		}
		siteKeyChanged := previousSettings.TurnstileSiteKey != req.TurnstileSiteKey
		secretKeyChanged := previousSettings.TurnstileSecretKey != req.TurnstileSecretKey
		if siteKeyChanged || secretKeyChanged {
			if err := h.turnstileService.ValidateSecretKey(c.Request.Context(), req.TurnstileSecretKey); err != nil {
				response.ErrorFrom(c, err)
				return
			}
		}
	}
	if req.TotpEnabled && !previousSettings.TotpEnabled {
		if !h.settingService.IsTotpEncryptionKeyConfigured() {
			response.BadRequest(c, "Cannot enable TOTP: TOTP_ENCRYPTION_KEY environment variable must be configured first. Generate a key with 'openssl rand -hex 32' and set it in your environment.")
			return
		}
	}
	if req.LinuxDoConnectEnabled {
		req.LinuxDoConnectClientID = strings.TrimSpace(req.LinuxDoConnectClientID)
		req.LinuxDoConnectClientSecret = strings.TrimSpace(req.LinuxDoConnectClientSecret)
		req.LinuxDoConnectRedirectURL = strings.TrimSpace(req.LinuxDoConnectRedirectURL)
		if req.LinuxDoConnectClientID == "" {
			response.BadRequest(c, "LinuxDo Client ID is required when enabled")
			return
		}
		if req.LinuxDoConnectRedirectURL == "" {
			response.BadRequest(c, "LinuxDo Redirect URL is required when enabled")
			return
		}
		if err := config.ValidateAbsoluteHTTPURL(req.LinuxDoConnectRedirectURL); err != nil {
			response.BadRequest(c, "LinuxDo Redirect URL must be an absolute http(s) URL")
			return
		}
		if req.LinuxDoConnectClientSecret == "" {
			if previousSettings.LinuxDoConnectClientSecret == "" {
				response.BadRequest(c, "LinuxDo Client Secret is required when enabled")
				return
			}
			req.LinuxDoConnectClientSecret = previousSettings.LinuxDoConnectClientSecret
		}
	}
	if req.GitHubOAuthEnabled {
		req.GitHubOAuthClientID = strings.TrimSpace(req.GitHubOAuthClientID)
		req.GitHubOAuthClientSecret = strings.TrimSpace(req.GitHubOAuthClientSecret)
		req.GitHubOAuthRedirectURL = strings.TrimSpace(req.GitHubOAuthRedirectURL)
		if req.GitHubOAuthClientID == "" {
			response.BadRequest(c, "GitHub Client ID is required when enabled")
			return
		}
		if req.GitHubOAuthRedirectURL == "" {
			response.BadRequest(c, "GitHub Redirect URL is required when enabled")
			return
		}
		if err := config.ValidateAbsoluteHTTPURL(req.GitHubOAuthRedirectURL); err != nil {
			response.BadRequest(c, "GitHub Redirect URL must be an absolute http(s) URL")
			return
		}
		if req.GitHubOAuthClientSecret == "" {
			if previousSettings.GitHubOAuthClientSecret == "" {
				response.BadRequest(c, "GitHub Client Secret is required when enabled")
				return
			}
			req.GitHubOAuthClientSecret = previousSettings.GitHubOAuthClientSecret
		}
	}
	if req.GoogleOAuthEnabled {
		req.GoogleOAuthClientID = strings.TrimSpace(req.GoogleOAuthClientID)
		req.GoogleOAuthClientSecret = strings.TrimSpace(req.GoogleOAuthClientSecret)
		req.GoogleOAuthRedirectURL = strings.TrimSpace(req.GoogleOAuthRedirectURL)
		if req.GoogleOAuthClientID == "" {
			response.BadRequest(c, "Google Client ID is required when enabled")
			return
		}
		if req.GoogleOAuthRedirectURL == "" {
			response.BadRequest(c, "Google Redirect URL is required when enabled")
			return
		}
		if err := config.ValidateAbsoluteHTTPURL(req.GoogleOAuthRedirectURL); err != nil {
			response.BadRequest(c, "Google Redirect URL must be an absolute http(s) URL")
			return
		}
		if req.GoogleOAuthClientSecret == "" {
			if previousSettings.GoogleOAuthClientSecret == "" {
				response.BadRequest(c, "Google Client Secret is required when enabled")
				return
			}
			req.GoogleOAuthClientSecret = previousSettings.GoogleOAuthClientSecret
		}
	}
	if req.DingTalkOAuthEnabled {
		req.DingTalkOAuthClientID = strings.TrimSpace(req.DingTalkOAuthClientID)
		req.DingTalkOAuthClientSecret = strings.TrimSpace(req.DingTalkOAuthClientSecret)
		req.DingTalkOAuthRedirectURL = strings.TrimSpace(req.DingTalkOAuthRedirectURL)
		if req.DingTalkOAuthClientID == "" {
			response.BadRequest(c, "DingTalk Client ID is required when enabled")
			return
		}
		if req.DingTalkOAuthRedirectURL == "" {
			response.BadRequest(c, "DingTalk Redirect URL is required when enabled")
			return
		}
		if err := config.ValidateAbsoluteHTTPURL(req.DingTalkOAuthRedirectURL); err != nil {
			response.BadRequest(c, "DingTalk Redirect URL must be an absolute http(s) URL")
			return
		}
		if req.DingTalkOAuthClientSecret == "" {
			if previousSettings.DingTalkOAuthClientSecret == "" {
				response.BadRequest(c, "DingTalk Client Secret is required when enabled")
				return
			}
			req.DingTalkOAuthClientSecret = previousSettings.DingTalkOAuthClientSecret
		}
	}
	req.ContentModerationProvider = strings.TrimSpace(req.ContentModerationProvider)
	req.ContentModerationBaseURL = strings.TrimSpace(req.ContentModerationBaseURL)
	req.ContentModerationAPIKey = strings.TrimSpace(req.ContentModerationAPIKey)
	req.ContentModerationModel = strings.TrimSpace(req.ContentModerationModel)
	newModerationKeys := append([]string{}, req.ContentModerationAPIKeys...)
	if req.ContentModerationAPIKey != "" {
		newModerationKeys = append(newModerationKeys, req.ContentModerationAPIKey)
	}
	hasModerationKeyMutation := len(newModerationKeys) > 0 ||
		len(req.DeleteContentModerationAPIKeyHashes) > 0 ||
		strings.TrimSpace(req.ContentModerationAPIKeysMode) != ""
	contentModerationAPIKeys := service.BuildContentModerationAPIKeyUpdate(service.ContentModerationAPIKeyUpdate{
		Existing:     previousSettings.ContentModerationAPIKeys,
		NewKeys:      newModerationKeys,
		Mode:         req.ContentModerationAPIKeysMode,
		DeleteHashes: req.DeleteContentModerationAPIKeyHashes,
	})
	if len(contentModerationAPIKeys) == 0 && !hasModerationKeyMutation {
		contentModerationAPIKeys = nil
	}
	if req.ContentModerationTimeoutMs <= 0 {
		req.ContentModerationTimeoutMs = previousSettings.ContentModerationTimeoutMs
		if req.ContentModerationTimeoutMs <= 0 {
			req.ContentModerationTimeoutMs = 1500
		}
	}
	if req.ContentModerationDedupeWindowSeconds < 0 {
		req.ContentModerationDedupeWindowSeconds = 0
	}
	if req.ContentModerationEnabled && req.ContentModerationProvider == "" {
		req.ContentModerationProvider = "openai"
	}
	if req.ContentModerationEnabled && !hasModerationKeyMutation && len(contentModerationAPIKeys) == 0 && req.ContentModerationAPIKey == "" {
		req.ContentModerationAPIKey = previousSettings.ContentModerationAPIKey
		contentModerationAPIKeys = previousSettings.ContentModerationAPIKeys
	}
	contentModerationKeywordBlockEnabled := previousSettings.ContentModerationKeywordBlockEnabled
	if req.ContentModerationKeywordBlockEnabled != nil {
		contentModerationKeywordBlockEnabled = *req.ContentModerationKeywordBlockEnabled
	}
	contentModerationKeywords := previousSettings.ContentModerationKeywords
	if req.ContentModerationKeywords != nil {
		contentModerationKeywords = service.NormalizeContentModerationKeywordList(req.ContentModerationKeywords)
	}
	contentModerationModelFilter := previousSettings.ContentModerationModelFilter
	if req.ContentModerationModelFilter != nil {
		filterJSON, err := service.MarshalContentModerationModelFilter(*req.ContentModerationModelFilter)
		if err != nil {
			response.BadRequest(c, "Invalid content moderation model filter")
			return
		}
		contentModerationModelFilter = service.NormalizeContentModerationModelFilter(filterJSON)
	}
	paymentAirwallexEnabled := previousSettings.PaymentProviderAirwallexEnabled
	if req.PaymentProviderAirwallexEnabled != nil {
		paymentAirwallexEnabled = *req.PaymentProviderAirwallexEnabled
	}
	airwallexEnv := previousSettings.AirwallexEnv
	if req.AirwallexEnv != nil {
		airwallexEnv = service.NormalizeAirwallexEnv(*req.AirwallexEnv)
	}
	airwallexClientID := previousSettings.AirwallexClientID
	if req.AirwallexClientID != nil {
		airwallexClientID = strings.TrimSpace(*req.AirwallexClientID)
	}
	airwallexAPIKey := ""
	if req.AirwallexAPIKey != nil {
		airwallexAPIKey = strings.TrimSpace(*req.AirwallexAPIKey)
	}
	if airwallexAPIKey == "" {
		airwallexAPIKey = previousSettings.AirwallexAPIKey
	}
	airwallexWebhookSecret := ""
	if req.AirwallexWebhookSecret != nil {
		airwallexWebhookSecret = strings.TrimSpace(*req.AirwallexWebhookSecret)
	}
	if airwallexWebhookSecret == "" {
		airwallexWebhookSecret = previousSettings.AirwallexWebhookSecret
	}
	paymentMobileForceQRCodeEnabled := previousSettings.PaymentMobileForceQRCodeEnabled
	if req.PaymentMobileForceQRCodeEnabled != nil {
		paymentMobileForceQRCodeEnabled = *req.PaymentMobileForceQRCodeEnabled
	}
	paymentAllowedCurrencies := previousSettings.PaymentAllowedCurrencies
	if req.PaymentAllowedCurrencies != nil {
		paymentAllowedCurrencies = service.NormalizePaymentAllowedCurrencies(*req.PaymentAllowedCurrencies)
	}
	if len(paymentAllowedCurrencies) == 0 {
		paymentAllowedCurrencies = service.DefaultPaymentCurrencies()
	}
	paymentDefaultCurrency := previousSettings.PaymentDefaultCurrency
	if req.PaymentDefaultCurrency != nil {
		paymentDefaultCurrency = service.NormalizePaymentCurrency(*req.PaymentDefaultCurrency)
	}
	if paymentDefaultCurrency == "" || !service.PaymentCurrencyAllowed(paymentDefaultCurrency, paymentAllowedCurrencies) {
		paymentDefaultCurrency = paymentAllowedCurrencies[0]
	}
	paymentMinTopupAmount := previousSettings.PaymentMinTopupAmount
	if req.PaymentMinTopupAmount != nil {
		paymentMinTopupAmount = *req.PaymentMinTopupAmount
	}
	if paymentMinTopupAmount <= 0 {
		paymentMinTopupAmount = 1
	}
	paymentMaxTopupAmount := previousSettings.PaymentMaxTopupAmount
	if req.PaymentMaxTopupAmount != nil {
		paymentMaxTopupAmount = *req.PaymentMaxTopupAmount
	}
	if paymentMaxTopupAmount < paymentMinTopupAmount {
		paymentMaxTopupAmount = paymentMinTopupAmount
	}
	paymentPlans := previousSettings.PaymentSubscriptionPlans
	if req.PaymentSubscriptionPlans != nil {
		paymentPlans = make([]service.PaymentSubscriptionPlan, 0, len(*req.PaymentSubscriptionPlans))
		for _, plan := range *req.PaymentSubscriptionPlans {
			paymentPlans = append(paymentPlans, service.PaymentSubscriptionPlan{
				PlanID:           strings.TrimSpace(plan.PlanID),
				Name:             strings.TrimSpace(plan.Name),
				GroupID:          plan.GroupID,
				ValidityDays:     plan.ValidityDays,
				PricesByCurrency: plan.PricesByCurrency,
				Enabled:          plan.Enabled,
			})
		}
	}
	antigravityVersion := previousSettings.AntigravityUserAgentVersion
	if req.AntigravityUserAgentVersion != nil {
		var ok bool
		antigravityVersion, ok = service.NormalizeAntigravityUserAgentVersion(*req.AntigravityUserAgentVersion)
		if !ok {
			response.BadRequest(c, "Antigravity User-Agent version must match major.minor.patch[-suffix]")
			return
		}
	}
	codexUAMode := previousSettings.CodexOAuthUserAgentMode
	if req.CodexOAuthUserAgentMode != nil {
		codexUAMode = *req.CodexOAuthUserAgentMode
	}
	codexUAOverride := previousSettings.CodexOAuthUserAgentOverride
	if req.CodexOAuthUserAgentOverride != nil {
		codexUAOverride = *req.CodexOAuthUserAgentOverride
	}
	codexUAPolicy := service.NormalizeCodexOAuthUserAgentPolicy(codexUAMode, codexUAOverride)
	if paymentAirwallexEnabled {
		if airwallexClientID == "" {
			response.BadRequest(c, "Airwallex Client ID is required when enabled")
			return
		}
		if airwallexAPIKey == "" {
			response.BadRequest(c, "Airwallex API Key is required when enabled")
			return
		}
	}
	purchaseEnabled := previousSettings.PurchaseSubscriptionEnabled
	if req.PurchaseSubscriptionEnabled != nil {
		purchaseEnabled = *req.PurchaseSubscriptionEnabled
	}
	purchaseURL := previousSettings.PurchaseSubscriptionURL
	if req.PurchaseSubscriptionURL != nil {
		purchaseURL = strings.TrimSpace(*req.PurchaseSubscriptionURL)
	}
	if purchaseEnabled && purchaseURL == "" && !paymentAirwallexEnabled {
		response.BadRequest(c, "Purchase Subscription URL is required when enabled")
		return
	}
	if purchaseURL != "" {
		if err := config.ValidateAbsoluteHTTPURL(purchaseURL); err != nil {
			response.BadRequest(c, "Purchase Subscription URL must be an absolute http(s) URL")
			return
		}
	}
	req.FrontendURL = strings.TrimSpace(req.FrontendURL)
	if req.FrontendURL != "" {
		if err := config.ValidateAbsoluteHTTPURL(req.FrontendURL); err != nil {
			response.BadRequest(c, "Frontend URL must be an absolute http(s) URL")
			return
		}
	}
	const (
		maxCustomMenuItems    = 20
		maxMenuItemLabelLen   = 50
		maxMenuItemURLLen     = 2048
		maxMenuItemIconSVGLen = 10 * 1024
		maxMenuItemIDLen      = 32
		maxMenuItemSlugLen    = 64
		maxMenuItemContentLen = 128 * 1024
	)
	customMenuJSON := previousSettings.CustomMenuItems
	if req.CustomMenuItems != nil {
		items := *req.CustomMenuItems
		if len(items) > maxCustomMenuItems {
			response.BadRequest(c, "Too many custom menu items (max 20)")
			return
		}
		for i, item := range items {
			items[i].Label = strings.TrimSpace(item.Label)
			items[i].Visibility = strings.TrimSpace(item.Visibility)
			items[i].IconSVG = strings.TrimSpace(item.IconSVG)
			items[i].URL = strings.TrimSpace(item.URL)
			items[i].PageMode = strings.TrimSpace(item.PageMode)
			items[i].PageSlug = strings.TrimSpace(item.PageSlug)
			items[i].ID = strings.TrimSpace(item.ID)

			if items[i].Label == "" {
				response.BadRequest(c, "Custom menu item label is required")
				return
			}
			if len(items[i].Label) > maxMenuItemLabelLen {
				response.BadRequest(c, "Custom menu item label is too long (max 50 characters)")
				return
			}
			if items[i].Visibility != "user" && items[i].Visibility != "admin" {
				response.BadRequest(c, "Custom menu item visibility must be 'user' or 'admin'")
				return
			}
			switch strings.ToLower(items[i].PageMode) {
			case "", "iframe":
				items[i].PageMode = "iframe"
				items[i].PageSlug = ""
				items[i].PageContent = ""
				items[i].PagePublished = false
				items[i].PagePublic = false
				if items[i].URL == "" {
					response.BadRequest(c, "Custom menu item URL is required")
					return
				}
				if len(items[i].URL) > maxMenuItemURLLen {
					response.BadRequest(c, "Custom menu item URL is too long (max 2048 characters)")
					return
				}
				if err := config.ValidateAbsoluteHTTPURL(items[i].URL); err != nil {
					response.BadRequest(c, "Custom menu item URL must be an absolute http(s) URL")
					return
				}
			case "markdown":
				items[i].URL = ""
				items[i].PageMode = "markdown"
				if items[i].PageSlug == "" {
					response.BadRequest(c, "Custom markdown page slug is required")
					return
				}
				if len(items[i].PageSlug) > maxMenuItemSlugLen {
					response.BadRequest(c, "Custom markdown page slug is too long (max 64 characters)")
					return
				}
				normalizedSlug := service.NormalizeCustomPageSlugForAdmin(items[i].PageSlug)
				if normalizedSlug == "" {
					response.BadRequest(c, "Custom markdown page slug is invalid")
					return
				}
				items[i].PageSlug = normalizedSlug
				if len(items[i].PageContent) > maxMenuItemContentLen {
					response.BadRequest(c, "Custom markdown page content is too large (max 128KB)")
					return
				}
				items[i].PageContent = strings.ReplaceAll(items[i].PageContent, "\r\n", "\n")
			default:
				response.BadRequest(c, "Custom menu item page_mode must be 'iframe' or 'markdown'")
				return
			}
			if len(items[i].IconSVG) > maxMenuItemIconSVGLen {
				response.BadRequest(c, "Custom menu item icon SVG is too large (max 10KB)")
				return
			}
			if items[i].ID == "" {
				id, err := generateMenuItemID()
				if err != nil {
					response.Error(c, http.StatusInternalServerError, "Failed to generate menu item ID")
					return
				}
				items[i].ID = id
			} else if len(items[i].ID) > maxMenuItemIDLen {
				response.BadRequest(c, "Custom menu item ID is too long (max 32 characters)")
				return
			} else if !menuItemIDPattern.MatchString(items[i].ID) {
				response.BadRequest(c, "Custom menu item ID contains invalid characters (only a-z, A-Z, 0-9, - and _ are allowed)")
				return
			}
		}
		seen := make(map[string]struct{}, len(items))
		seenSlugs := make(map[string]struct{}, len(items))
		for _, item := range items {
			if _, exists := seen[item.ID]; exists {
				response.BadRequest(c, "Duplicate custom menu item ID: "+item.ID)
				return
			}
			seen[item.ID] = struct{}{}
			if item.PageMode == "markdown" {
				if _, exists := seenSlugs[item.PageSlug]; exists {
					response.BadRequest(c, "Duplicate custom markdown page slug: "+item.PageSlug)
					return
				}
				seenSlugs[item.PageSlug] = struct{}{}
			}
		}
		menuBytes, err := json.Marshal(items)
		if err != nil {
			response.BadRequest(c, "Failed to serialize custom menu items")
			return
		}
		customMenuJSON = string(menuBytes)
	}
	if req.OpsMetricsIntervalSeconds != nil {
		v := *req.OpsMetricsIntervalSeconds
		if v < 60 {
			v = 60
		}
		if v > 3600 {
			v = 3600
		}
		req.OpsMetricsIntervalSeconds = &v
	}
	availableChannelsEnabled := previousSettings.AvailableChannelsEnabled
	if req.AvailableChannelsEnabled != nil {
		availableChannelsEnabled = *req.AvailableChannelsEnabled
	}
	channelMonitorEnabled := previousSettings.ChannelMonitorEnabled
	if req.ChannelMonitorEnabled != nil {
		channelMonitorEnabled = *req.ChannelMonitorEnabled
	}
	channelMonitorDefaultIntervalSeconds := previousSettings.ChannelMonitorDefaultIntervalSeconds
	if req.ChannelMonitorDefaultIntervalSeconds != nil {
		channelMonitorDefaultIntervalSeconds = *req.ChannelMonitorDefaultIntervalSeconds
	}
	if channelMonitorDefaultIntervalSeconds <= 0 {
		channelMonitorDefaultIntervalSeconds = 60
	}
	if channelMonitorDefaultIntervalSeconds < 15 {
		channelMonitorDefaultIntervalSeconds = 15
	}
	if channelMonitorDefaultIntervalSeconds > 3600 {
		channelMonitorDefaultIntervalSeconds = 3600
	}

	affiliateEnabled := previousSettings.AffiliateEnabled
	if req.AffiliateEnabled != nil {
		affiliateEnabled = *req.AffiliateEnabled
	}
	affiliateTransferEnabled := previousSettings.AffiliateTransferEnabled
	if req.AffiliateTransferEnabled != nil {
		affiliateTransferEnabled = *req.AffiliateTransferEnabled
	}
	affiliateRebateOnUsageEnabled := previousSettings.AffiliateRebateOnUsageEnabled
	if req.AffiliateRebateOnUsageEnabled != nil {
		affiliateRebateOnUsageEnabled = *req.AffiliateRebateOnUsageEnabled
	}
	affiliateRebateOnTopupEnabled := previousSettings.AffiliateRebateOnTopupEnabled
	if req.AffiliateRebateOnTopupEnabled != nil {
		affiliateRebateOnTopupEnabled = *req.AffiliateRebateOnTopupEnabled
	}

	affiliateRebateRate := previousSettings.AffiliateRebateRate
	if req.AffiliateRebateRate != nil {
		affiliateRebateRate = *req.AffiliateRebateRate
	}
	if affiliateRebateRate < 0 {
		affiliateRebateRate = 0
	}
	if affiliateRebateRate > 100 {
		affiliateRebateRate = 100
	}
	affiliateRebateFreezeHours := previousSettings.AffiliateRebateFreezeHours
	if req.AffiliateRebateFreezeHours != nil {
		affiliateRebateFreezeHours = *req.AffiliateRebateFreezeHours
	}
	if affiliateRebateFreezeHours < 0 {
		affiliateRebateFreezeHours = 0
	}
	if affiliateRebateFreezeHours > 720 {
		affiliateRebateFreezeHours = 720
	}
	affiliateRebateDurationDays := previousSettings.AffiliateRebateDurationDays
	if req.AffiliateRebateDurationDays != nil {
		affiliateRebateDurationDays = *req.AffiliateRebateDurationDays
	}
	if affiliateRebateDurationDays < 0 {
		affiliateRebateDurationDays = 0
	}
	if affiliateRebateDurationDays > 3650 {
		affiliateRebateDurationDays = 3650
	}
	affiliateRebatePerInviteeCap := previousSettings.AffiliateRebatePerInviteeCap
	if req.AffiliateRebatePerInviteeCap != nil {
		affiliateRebatePerInviteeCap = *req.AffiliateRebatePerInviteeCap
	}
	if affiliateRebatePerInviteeCap < 0 {
		affiliateRebatePerInviteeCap = 0
	}
	affiliateAffCodeLength := previousSettings.AffiliateAffCodeLength
	if req.AffiliateAffCodeLength != nil {
		affiliateAffCodeLength = *req.AffiliateAffCodeLength
	}
	if affiliateAffCodeLength < 6 {
		affiliateAffCodeLength = 6
	}
	if affiliateAffCodeLength > 32 {
		affiliateAffCodeLength = 32
	}

	defaultSubscriptions := make([]service.DefaultSubscriptionSetting, 0, len(req.DefaultSubscriptions))
	for _, sub := range req.DefaultSubscriptions {
		defaultSubscriptions = append(defaultSubscriptions, service.DefaultSubscriptionSetting{GroupID: sub.GroupID, ValidityDays: sub.ValidityDays})
	}

	loginAgreementEnabled := previousSettings.LoginAgreementEnabled
	if req.LoginAgreementEnabled != nil {
		loginAgreementEnabled = *req.LoginAgreementEnabled
	}
	loginAgreementMode := previousSettings.LoginAgreementMode
	if req.LoginAgreementMode != nil {
		loginAgreementMode = service.NormalizeLoginAgreementMode(*req.LoginAgreementMode)
	}
	loginAgreementUpdatedAt := previousSettings.LoginAgreementUpdatedAt
	if req.LoginAgreementUpdatedAt != nil {
		loginAgreementUpdatedAt = strings.TrimSpace(*req.LoginAgreementUpdatedAt)
	}
	loginAgreementDocuments := previousSettings.LoginAgreementDocuments
	if req.LoginAgreementDocuments != nil {
		loginAgreementDocuments = make([]service.LoginAgreementDocument, 0, len(*req.LoginAgreementDocuments))
		for _, item := range *req.LoginAgreementDocuments {
			loginAgreementDocuments = append(loginAgreementDocuments, service.LoginAgreementDocument{
				ID:       item.ID,
				Title:    item.Title,
				PageSlug: item.PageSlug,
			})
		}
		loginAgreementDocuments = service.NormalizeLoginAgreementDocuments(loginAgreementDocuments)
	}
	if loginAgreementEnabled && len(loginAgreementDocuments) == 0 {
		response.BadRequest(c, "Login agreement requires at least one published markdown page")
		return
	}
	if loginAgreementEnabled {
		publishedPages := collectPublishedMarkdownPageSlugs(customMenuJSON)
		for _, doc := range loginAgreementDocuments {
			if _, ok := publishedPages[doc.PageSlug]; !ok {
				response.BadRequest(c, "Login agreement document must reference a published markdown page")
				return
			}
		}
	}

	openAIFastPolicy := previousSettings.OpenAIFastPolicySettings
	if req.OpenAIFastPolicySettings != nil {
		rules := make([]service.OpenAIFastPolicyRule, 0, len(req.OpenAIFastPolicySettings.Rules))
		for _, r := range req.OpenAIFastPolicySettings.Rules {
			rules = append(rules, service.OpenAIFastPolicyRule{
				ServiceTier:    r.ServiceTier,
				Action:         r.Action,
				Scope:          r.Scope,
				ModelWhitelist: append([]string(nil), r.ModelWhitelist...),
				FallbackAction: r.FallbackAction,
			})
		}
		openAIFastPolicy = &service.OpenAIFastPolicySettings{Rules: rules}
	}
	if openAIFastPolicy == nil {
		openAIFastPolicy = service.DefaultOpenAIFastPolicySettings()
	}

	enableAnthropicTTL1hInjection := previousSettings.EnableAnthropicCacheTTL1hInjection
	if req.EnableAnthropicCacheTTL1hInjection != nil {
		enableAnthropicTTL1hInjection = *req.EnableAnthropicCacheTTL1hInjection
	}

	if req.MinClaudeCodeVersion != "" {
		if !semverPattern.MatchString(req.MinClaudeCodeVersion) {
			response.Error(c, http.StatusBadRequest, "min_claude_code_version must be empty or a valid semver (e.g. 2.1.63)")
			return
		}
	}
	if req.MaxClaudeCodeVersion != "" {
		if !semverPattern.MatchString(req.MaxClaudeCodeVersion) {
			response.Error(c, http.StatusBadRequest, "max_claude_code_version must be empty or a valid semver (e.g. 3.0.0)")
			return
		}
	}
	if req.MinClaudeCodeVersion != "" && req.MaxClaudeCodeVersion != "" {
		if service.CompareVersions(req.MaxClaudeCodeVersion, req.MinClaudeCodeVersion) < 0 {
			response.Error(c, http.StatusBadRequest, "max_claude_code_version must be greater than or equal to min_claude_code_version")
			return
		}
	}
	settings := &service.SystemSettings{RegistrationEnabled: req.RegistrationEnabled, EmailVerifyEnabled: req.EmailVerifyEnabled, RegistrationEmailSuffixWhitelist: req.RegistrationEmailSuffixWhitelist, PromoCodeEnabled: req.PromoCodeEnabled, PasswordResetEnabled: req.PasswordResetEnabled, FrontendURL: req.FrontendURL, InvitationCodeEnabled: req.InvitationCodeEnabled, TotpEnabled: req.TotpEnabled, SMTPHost: req.SMTPHost, SMTPPort: req.SMTPPort, SMTPUsername: req.SMTPUsername, SMTPPassword: req.SMTPPassword, SMTPFrom: req.SMTPFrom, SMTPFromName: req.SMTPFromName, SMTPUseTLS: req.SMTPUseTLS, TelegramChatID: req.TelegramChatID, TelegramBotToken: req.TelegramBotToken, TurnstileEnabled: req.TurnstileEnabled, TurnstileSiteKey: req.TurnstileSiteKey, TurnstileSecretKey: req.TurnstileSecretKey, LinuxDoConnectEnabled: req.LinuxDoConnectEnabled, LinuxDoConnectClientID: req.LinuxDoConnectClientID, LinuxDoConnectClientSecret: req.LinuxDoConnectClientSecret, LinuxDoConnectRedirectURL: req.LinuxDoConnectRedirectURL, GitHubOAuthEnabled: req.GitHubOAuthEnabled, GitHubOAuthClientID: req.GitHubOAuthClientID, GitHubOAuthClientSecret: req.GitHubOAuthClientSecret, GitHubOAuthRedirectURL: req.GitHubOAuthRedirectURL, GoogleOAuthEnabled: req.GoogleOAuthEnabled, GoogleOAuthClientID: req.GoogleOAuthClientID, GoogleOAuthClientSecret: req.GoogleOAuthClientSecret, GoogleOAuthRedirectURL: req.GoogleOAuthRedirectURL, DingTalkOAuthEnabled: req.DingTalkOAuthEnabled, DingTalkOAuthClientID: req.DingTalkOAuthClientID, DingTalkOAuthClientSecret: req.DingTalkOAuthClientSecret, DingTalkOAuthRedirectURL: req.DingTalkOAuthRedirectURL, ContentModerationEnabled: req.ContentModerationEnabled, ContentModerationProvider: req.ContentModerationProvider, ContentModerationBaseURL: req.ContentModerationBaseURL, ContentModerationAPIKey: req.ContentModerationAPIKey, ContentModerationModel: req.ContentModerationModel, ContentModerationTimeoutMs: req.ContentModerationTimeoutMs, ContentModerationDedupeWindowSeconds: req.ContentModerationDedupeWindowSeconds, ContentModerationFailOpen: func() bool {
		if req.ContentModerationFailOpen != nil {
			return *req.ContentModerationFailOpen
		}
		return previousSettings.ContentModerationFailOpen
	}(), ContentModerationKeywordBlockEnabled: contentModerationKeywordBlockEnabled, ContentModerationKeywords: contentModerationKeywords, ContentModerationModelFilter: contentModerationModelFilter, SiteName: req.SiteName, SiteLogo: req.SiteLogo, SiteSubtitle: req.SiteSubtitle, VisualPresetDefault: req.VisualPresetDefault, AccountAiryWhiteSurfaceEnabled: req.AccountAiryWhiteSurfaceEnabled, APIBaseURL: req.APIBaseURL, ContactInfo: req.ContactInfo, DocURL: req.DocURL, HomeContent: req.HomeContent, HideCcsImportButton: req.HideCcsImportButton, AvailableChannelsEnabled: availableChannelsEnabled, ChannelMonitorEnabled: channelMonitorEnabled, ChannelMonitorDefaultIntervalSeconds: channelMonitorDefaultIntervalSeconds, PublicModelCatalogEnabled: req.PublicModelCatalogEnabled, PurchaseSubscriptionEnabled: purchaseEnabled, PurchaseSubscriptionURL: purchaseURL, PaymentProviderAirwallexEnabled: paymentAirwallexEnabled, AirwallexEnv: airwallexEnv, AirwallexClientID: airwallexClientID, AirwallexAPIKey: airwallexAPIKey, AirwallexWebhookSecret: airwallexWebhookSecret, PaymentMobileForceQRCodeEnabled: paymentMobileForceQRCodeEnabled, PaymentAllowedCurrencies: paymentAllowedCurrencies, PaymentDefaultCurrency: paymentDefaultCurrency, PaymentMinTopupAmount: paymentMinTopupAmount, PaymentMaxTopupAmount: paymentMaxTopupAmount, PaymentSubscriptionPlans: paymentPlans, AntigravityUserAgentVersion: antigravityVersion, CodexOAuthUserAgentMode: codexUAPolicy.Mode, CodexOAuthUserAgentOverride: codexUAPolicy.Override, CustomMenuItems: customMenuJSON, DefaultConcurrency: req.DefaultConcurrency, DefaultBalance: req.DefaultBalance, DefaultSubscriptions: defaultSubscriptions, EnableModelFallback: req.EnableModelFallback, FallbackModelAnthropic: req.FallbackModelAnthropic, FallbackModelOpenAI: req.FallbackModelOpenAI, FallbackModelGemini: req.FallbackModelGemini, FallbackModelAntigravity: req.FallbackModelAntigravity, EnableIdentityPatch: req.EnableIdentityPatch, IdentityPatchPrompt: req.IdentityPatchPrompt, MinClaudeCodeVersion: req.MinClaudeCodeVersion, MaxClaudeCodeVersion: req.MaxClaudeCodeVersion, AllowUngroupedKeyScheduling: req.AllowUngroupedKeyScheduling, BackendModeEnabled: req.BackendModeEnabled, MaintenanceModeEnabled: req.MaintenanceModeEnabled, OpsMonitoringEnabled: func() bool {
		if req.OpsMonitoringEnabled != nil {
			return *req.OpsMonitoringEnabled
		}
		return previousSettings.OpsMonitoringEnabled
	}(), OpsRealtimeMonitoringEnabled: func() bool {
		if req.OpsRealtimeMonitoringEnabled != nil {
			return *req.OpsRealtimeMonitoringEnabled
		}
		return previousSettings.OpsRealtimeMonitoringEnabled
	}(), OpsQueryModeDefault: func() string {
		if req.OpsQueryModeDefault != nil {
			return *req.OpsQueryModeDefault
		}
		return previousSettings.OpsQueryModeDefault
	}(), OpsMetricsIntervalSeconds: func() int {
		if req.OpsMetricsIntervalSeconds != nil {
			return *req.OpsMetricsIntervalSeconds
		}
		return previousSettings.OpsMetricsIntervalSeconds
	}(), OpenAIFastPolicySettings: openAIFastPolicy, EnableAnthropicCacheTTL1hInjection: enableAnthropicTTL1hInjection}
	settings.AffiliateEnabled = affiliateEnabled
	settings.AffiliateTransferEnabled = affiliateTransferEnabled
	settings.AffiliateRebateOnUsageEnabled = affiliateRebateOnUsageEnabled
	settings.AffiliateRebateOnTopupEnabled = affiliateRebateOnTopupEnabled
	settings.AffiliateRebateRate = affiliateRebateRate
	settings.AffiliateRebateFreezeHours = affiliateRebateFreezeHours
	settings.AffiliateRebateDurationDays = affiliateRebateDurationDays
	settings.AffiliateRebatePerInviteeCap = affiliateRebatePerInviteeCap
	settings.AffiliateAffCodeLength = affiliateAffCodeLength
	settings.ContentModerationAPIKeys = contentModerationAPIKeys
	if settings.ContentModerationAPIKey == "" && len(contentModerationAPIKeys) > 0 {
		settings.ContentModerationAPIKey = contentModerationAPIKeys[0].Key
	}
	settings.LoginAgreementEnabled = loginAgreementEnabled
	settings.LoginAgreementMode = loginAgreementMode
	settings.LoginAgreementUpdatedAt = loginAgreementUpdatedAt
	settings.LoginAgreementDocuments = loginAgreementDocuments
	if err := h.settingService.UpdateSettings(c.Request.Context(), settings); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.auditSettingsUpdate(c, previousSettings, settings, req)
	updatedSettings, err := h.settingService.GetAllSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, buildSystemSettingsDTO(h.settingService, updatedSettings, dto.ParseCustomMenuItems(updatedSettings.CustomMenuItems)))
}

func collectPublishedMarkdownPageSlugs(raw string) map[string]struct{} {
	var items []dto.CustomMenuItem
	_ = json.Unmarshal([]byte(strings.TrimSpace(raw)), &items)
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(item.PageMode), "markdown") || !item.PagePublished {
			continue
		}
		slug := service.NormalizeCustomPageSlugForAdmin(item.PageSlug)
		if slug != "" {
			out[slug] = struct{}{}
		}
	}
	return out
}
