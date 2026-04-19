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
	RegistrationEnabled              bool                             `json:"registration_enabled"`
	EmailVerifyEnabled               bool                             `json:"email_verify_enabled"`
	RegistrationEmailSuffixWhitelist []string                         `json:"registration_email_suffix_whitelist"`
	PromoCodeEnabled                 bool                             `json:"promo_code_enabled"`
	PasswordResetEnabled             bool                             `json:"password_reset_enabled"`
	FrontendURL                      string                           `json:"frontend_url"`
	InvitationCodeEnabled            bool                             `json:"invitation_code_enabled"`
	TotpEnabled                      bool                             `json:"totp_enabled"`
	SMTPHost                         string                           `json:"smtp_host"`
	SMTPPort                         int                              `json:"smtp_port"`
	SMTPUsername                     string                           `json:"smtp_username"`
	SMTPPassword                     string                           `json:"smtp_password"`
	SMTPFrom                         string                           `json:"smtp_from_email"`
	SMTPFromName                     string                           `json:"smtp_from_name"`
	SMTPUseTLS                       bool                             `json:"smtp_use_tls"`
	TelegramChatID                   string                           `json:"telegram_chat_id"`
	TelegramBotToken                 string                           `json:"telegram_bot_token"`
	TurnstileEnabled                 bool                             `json:"turnstile_enabled"`
	TurnstileSiteKey                 string                           `json:"turnstile_site_key"`
	TurnstileSecretKey               string                           `json:"turnstile_secret_key"`
	LinuxDoConnectEnabled            bool                             `json:"linuxdo_connect_enabled"`
	LinuxDoConnectClientID           string                           `json:"linuxdo_connect_client_id"`
	LinuxDoConnectClientSecret       string                           `json:"linuxdo_connect_client_secret"`
	LinuxDoConnectRedirectURL        string                           `json:"linuxdo_connect_redirect_url"`
	SiteName                         string                           `json:"site_name"`
	SiteLogo                         string                           `json:"site_logo"`
	SiteSubtitle                     string                           `json:"site_subtitle"`
	APIBaseURL                       string                           `json:"api_base_url"`
	ContactInfo                      string                           `json:"contact_info"`
	DocURL                           string                           `json:"doc_url"`
	HomeContent                      string                           `json:"home_content"`
	HideCcsImportButton              bool                             `json:"hide_ccs_import_button"`
	PublicModelCatalogEnabled        bool                             `json:"public_model_catalog_enabled"`
	PurchaseSubscriptionEnabled      *bool                            `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL          *string                          `json:"purchase_subscription_url"`
	CustomMenuItems                  *[]dto.CustomMenuItem            `json:"custom_menu_items"`
	DefaultConcurrency               int                              `json:"default_concurrency"`
	DefaultBalance                   float64                          `json:"default_balance"`
	DefaultSubscriptions             []dto.DefaultSubscriptionSetting `json:"default_subscriptions"`
	EnableModelFallback              bool                             `json:"enable_model_fallback"`
	FallbackModelAnthropic           string                           `json:"fallback_model_anthropic"`
	FallbackModelOpenAI              string                           `json:"fallback_model_openai"`
	FallbackModelGemini              string                           `json:"fallback_model_gemini"`
	FallbackModelAntigravity         string                           `json:"fallback_model_antigravity"`
	EnableIdentityPatch              bool                             `json:"enable_identity_patch"`
	IdentityPatchPrompt              string                           `json:"identity_patch_prompt"`
	OpsMonitoringEnabled             *bool                            `json:"ops_monitoring_enabled"`
	OpsRealtimeMonitoringEnabled     *bool                            `json:"ops_realtime_monitoring_enabled"`
	OpsQueryModeDefault              *string                          `json:"ops_query_mode_default"`
	OpsMetricsIntervalSeconds        *int                             `json:"ops_metrics_interval_seconds"`
	MinClaudeCodeVersion             string                           `json:"min_claude_code_version"`
	MaxClaudeCodeVersion             string                           `json:"max_claude_code_version"`
	AllowUngroupedKeyScheduling      bool                             `json:"allow_ungrouped_key_scheduling"`
	BackendModeEnabled               bool                             `json:"backend_mode_enabled"`
	MaintenanceModeEnabled           bool                             `json:"maintenance_mode_enabled"`
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
	purchaseEnabled := previousSettings.PurchaseSubscriptionEnabled
	if req.PurchaseSubscriptionEnabled != nil {
		purchaseEnabled = *req.PurchaseSubscriptionEnabled
	}
	purchaseURL := previousSettings.PurchaseSubscriptionURL
	if req.PurchaseSubscriptionURL != nil {
		purchaseURL = strings.TrimSpace(*req.PurchaseSubscriptionURL)
	}
	if purchaseEnabled {
		if purchaseURL == "" {
			response.BadRequest(c, "Purchase Subscription URL is required when enabled")
			return
		}
		if err := config.ValidateAbsoluteHTTPURL(purchaseURL); err != nil {
			response.BadRequest(c, "Purchase Subscription URL must be an absolute http(s) URL")
			return
		}
	} else if purchaseURL != "" {
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
	)
	customMenuJSON := previousSettings.CustomMenuItems
	if req.CustomMenuItems != nil {
		items := *req.CustomMenuItems
		if len(items) > maxCustomMenuItems {
			response.BadRequest(c, "Too many custom menu items (max 20)")
			return
		}
		for i, item := range items {
			if strings.TrimSpace(item.Label) == "" {
				response.BadRequest(c, "Custom menu item label is required")
				return
			}
			if len(item.Label) > maxMenuItemLabelLen {
				response.BadRequest(c, "Custom menu item label is too long (max 50 characters)")
				return
			}
			if strings.TrimSpace(item.URL) == "" {
				response.BadRequest(c, "Custom menu item URL is required")
				return
			}
			if len(item.URL) > maxMenuItemURLLen {
				response.BadRequest(c, "Custom menu item URL is too long (max 2048 characters)")
				return
			}
			if err := config.ValidateAbsoluteHTTPURL(strings.TrimSpace(item.URL)); err != nil {
				response.BadRequest(c, "Custom menu item URL must be an absolute http(s) URL")
				return
			}
			if item.Visibility != "user" && item.Visibility != "admin" {
				response.BadRequest(c, "Custom menu item visibility must be 'user' or 'admin'")
				return
			}
			if len(item.IconSVG) > maxMenuItemIconSVGLen {
				response.BadRequest(c, "Custom menu item icon SVG is too large (max 10KB)")
				return
			}
			if strings.TrimSpace(item.ID) == "" {
				id, err := generateMenuItemID()
				if err != nil {
					response.Error(c, http.StatusInternalServerError, "Failed to generate menu item ID")
					return
				}
				items[i].ID = id
			} else if len(item.ID) > maxMenuItemIDLen {
				response.BadRequest(c, "Custom menu item ID is too long (max 32 characters)")
				return
			} else if !menuItemIDPattern.MatchString(item.ID) {
				response.BadRequest(c, "Custom menu item ID contains invalid characters (only a-z, A-Z, 0-9, - and _ are allowed)")
				return
			}
		}
		seen := make(map[string]struct{}, len(items))
		for _, item := range items {
			if _, exists := seen[item.ID]; exists {
				response.BadRequest(c, "Duplicate custom menu item ID: "+item.ID)
				return
			}
			seen[item.ID] = struct{}{}
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
	defaultSubscriptions := make([]service.DefaultSubscriptionSetting, 0, len(req.DefaultSubscriptions))
	for _, sub := range req.DefaultSubscriptions {
		defaultSubscriptions = append(defaultSubscriptions, service.DefaultSubscriptionSetting{GroupID: sub.GroupID, ValidityDays: sub.ValidityDays})
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
	settings := &service.SystemSettings{RegistrationEnabled: req.RegistrationEnabled, EmailVerifyEnabled: req.EmailVerifyEnabled, RegistrationEmailSuffixWhitelist: req.RegistrationEmailSuffixWhitelist, PromoCodeEnabled: req.PromoCodeEnabled, PasswordResetEnabled: req.PasswordResetEnabled, FrontendURL: req.FrontendURL, InvitationCodeEnabled: req.InvitationCodeEnabled, TotpEnabled: req.TotpEnabled, SMTPHost: req.SMTPHost, SMTPPort: req.SMTPPort, SMTPUsername: req.SMTPUsername, SMTPPassword: req.SMTPPassword, SMTPFrom: req.SMTPFrom, SMTPFromName: req.SMTPFromName, SMTPUseTLS: req.SMTPUseTLS, TelegramChatID: req.TelegramChatID, TelegramBotToken: req.TelegramBotToken, TurnstileEnabled: req.TurnstileEnabled, TurnstileSiteKey: req.TurnstileSiteKey, TurnstileSecretKey: req.TurnstileSecretKey, LinuxDoConnectEnabled: req.LinuxDoConnectEnabled, LinuxDoConnectClientID: req.LinuxDoConnectClientID, LinuxDoConnectClientSecret: req.LinuxDoConnectClientSecret, LinuxDoConnectRedirectURL: req.LinuxDoConnectRedirectURL, SiteName: req.SiteName, SiteLogo: req.SiteLogo, SiteSubtitle: req.SiteSubtitle, APIBaseURL: req.APIBaseURL, ContactInfo: req.ContactInfo, DocURL: req.DocURL, HomeContent: req.HomeContent, HideCcsImportButton: req.HideCcsImportButton, PublicModelCatalogEnabled: req.PublicModelCatalogEnabled, PurchaseSubscriptionEnabled: purchaseEnabled, PurchaseSubscriptionURL: purchaseURL, CustomMenuItems: customMenuJSON, DefaultConcurrency: req.DefaultConcurrency, DefaultBalance: req.DefaultBalance, DefaultSubscriptions: defaultSubscriptions, EnableModelFallback: req.EnableModelFallback, FallbackModelAnthropic: req.FallbackModelAnthropic, FallbackModelOpenAI: req.FallbackModelOpenAI, FallbackModelGemini: req.FallbackModelGemini, FallbackModelAntigravity: req.FallbackModelAntigravity, EnableIdentityPatch: req.EnableIdentityPatch, IdentityPatchPrompt: req.IdentityPatchPrompt, MinClaudeCodeVersion: req.MinClaudeCodeVersion, MaxClaudeCodeVersion: req.MaxClaudeCodeVersion, AllowUngroupedKeyScheduling: req.AllowUngroupedKeyScheduling, BackendModeEnabled: req.BackendModeEnabled, MaintenanceModeEnabled: req.MaintenanceModeEnabled, OpsMonitoringEnabled: func() bool {
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
	}()}
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
