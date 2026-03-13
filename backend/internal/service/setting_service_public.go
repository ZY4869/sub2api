package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"net/url"
	"strconv"
	"strings"
)

func (s *SettingService) GetAllSettings(ctx context.Context) (*SystemSettings, error) {
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}
	return s.parseSettings(settings), nil
}
func (s *SettingService) GetPublicSettings(ctx context.Context) (*PublicSettings, error) {
	keys := []string{SettingKeyRegistrationEnabled, SettingKeyEmailVerifyEnabled, SettingKeyRegistrationEmailSuffixWhitelist, SettingKeyPromoCodeEnabled, SettingKeyPasswordResetEnabled, SettingKeyInvitationCodeEnabled, SettingKeyTotpEnabled, SettingKeyTurnstileEnabled, SettingKeyTurnstileSiteKey, SettingKeySiteName, SettingKeySiteLogo, SettingKeySiteSubtitle, SettingKeyAPIBaseURL, SettingKeyContactInfo, SettingKeyDocURL, SettingKeyHomeContent, SettingKeyHideCcsImportButton, SettingKeyPurchaseSubscriptionEnabled, SettingKeyPurchaseSubscriptionURL, SettingKeySoraClientEnabled, SettingKeyCustomMenuItems, SettingKeyLinuxDoConnectEnabled}
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
	emailVerifyEnabled := settings[SettingKeyEmailVerifyEnabled] == "true"
	passwordResetEnabled := emailVerifyEnabled && settings[SettingKeyPasswordResetEnabled] == "true"
	registrationEmailSuffixWhitelist := ParseRegistrationEmailSuffixWhitelist(settings[SettingKeyRegistrationEmailSuffixWhitelist])
	return &PublicSettings{RegistrationEnabled: settings[SettingKeyRegistrationEnabled] == "true", EmailVerifyEnabled: emailVerifyEnabled, RegistrationEmailSuffixWhitelist: registrationEmailSuffixWhitelist, PromoCodeEnabled: settings[SettingKeyPromoCodeEnabled] != "false", PasswordResetEnabled: passwordResetEnabled, InvitationCodeEnabled: settings[SettingKeyInvitationCodeEnabled] == "true", TotpEnabled: settings[SettingKeyTotpEnabled] == "true", TurnstileEnabled: settings[SettingKeyTurnstileEnabled] == "true", TurnstileSiteKey: settings[SettingKeyTurnstileSiteKey], SiteName: s.getStringOrDefault(settings, SettingKeySiteName, "Sub2API"), SiteLogo: settings[SettingKeySiteLogo], SiteSubtitle: s.getStringOrDefault(settings, SettingKeySiteSubtitle, "Subscription to API Conversion Platform"), APIBaseURL: settings[SettingKeyAPIBaseURL], ContactInfo: settings[SettingKeyContactInfo], DocURL: settings[SettingKeyDocURL], HomeContent: settings[SettingKeyHomeContent], HideCcsImportButton: settings[SettingKeyHideCcsImportButton] == "true", PurchaseSubscriptionEnabled: settings[SettingKeyPurchaseSubscriptionEnabled] == "true", PurchaseSubscriptionURL: strings.TrimSpace(settings[SettingKeyPurchaseSubscriptionURL]), SoraClientEnabled: settings[SettingKeySoraClientEnabled] == "true", CustomMenuItems: settings[SettingKeyCustomMenuItems], LinuxDoOAuthEnabled: linuxDoEnabled}, nil
}
func (s *SettingService) SetOnUpdateCallback(callback func()) {
	s.onUpdate = callback
}
func (s *SettingService) SetOnS3UpdateCallback(callback func()) {
	s.onS3Update = callback
}
func (s *SettingService) SetVersion(version string) {
	s.version = version
}
func (s *SettingService) GetPublicSettingsForInjection(ctx context.Context) (any, error) {
	settings, err := s.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &struct {
		RegistrationEnabled              bool            `json:"registration_enabled"`
		EmailVerifyEnabled               bool            `json:"email_verify_enabled"`
		RegistrationEmailSuffixWhitelist []string        `json:"registration_email_suffix_whitelist"`
		PromoCodeEnabled                 bool            `json:"promo_code_enabled"`
		PasswordResetEnabled             bool            `json:"password_reset_enabled"`
		InvitationCodeEnabled            bool            `json:"invitation_code_enabled"`
		TotpEnabled                      bool            `json:"totp_enabled"`
		TurnstileEnabled                 bool            `json:"turnstile_enabled"`
		TurnstileSiteKey                 string          `json:"turnstile_site_key,omitempty"`
		SiteName                         string          `json:"site_name"`
		SiteLogo                         string          `json:"site_logo,omitempty"`
		SiteSubtitle                     string          `json:"site_subtitle,omitempty"`
		APIBaseURL                       string          `json:"api_base_url,omitempty"`
		ContactInfo                      string          `json:"contact_info,omitempty"`
		DocURL                           string          `json:"doc_url,omitempty"`
		HomeContent                      string          `json:"home_content,omitempty"`
		HideCcsImportButton              bool            `json:"hide_ccs_import_button"`
		PurchaseSubscriptionEnabled      bool            `json:"purchase_subscription_enabled"`
		PurchaseSubscriptionURL          string          `json:"purchase_subscription_url,omitempty"`
		SoraClientEnabled                bool            `json:"sora_client_enabled"`
		CustomMenuItems                  json.RawMessage `json:"custom_menu_items"`
		LinuxDoOAuthEnabled              bool            `json:"linuxdo_oauth_enabled"`
		Version                          string          `json:"version,omitempty"`
	}{RegistrationEnabled: settings.RegistrationEnabled, EmailVerifyEnabled: settings.EmailVerifyEnabled, RegistrationEmailSuffixWhitelist: settings.RegistrationEmailSuffixWhitelist, PromoCodeEnabled: settings.PromoCodeEnabled, PasswordResetEnabled: settings.PasswordResetEnabled, InvitationCodeEnabled: settings.InvitationCodeEnabled, TotpEnabled: settings.TotpEnabled, TurnstileEnabled: settings.TurnstileEnabled, TurnstileSiteKey: settings.TurnstileSiteKey, SiteName: settings.SiteName, SiteLogo: settings.SiteLogo, SiteSubtitle: settings.SiteSubtitle, APIBaseURL: settings.APIBaseURL, ContactInfo: settings.ContactInfo, DocURL: settings.DocURL, HomeContent: settings.HomeContent, HideCcsImportButton: settings.HideCcsImportButton, PurchaseSubscriptionEnabled: settings.PurchaseSubscriptionEnabled, PurchaseSubscriptionURL: settings.PurchaseSubscriptionURL, SoraClientEnabled: settings.SoraClientEnabled, CustomMenuItems: filterUserVisibleMenuItems(settings.CustomMenuItems), LinuxDoOAuthEnabled: settings.LinuxDoOAuthEnabled, Version: s.version}, nil
}
func filterUserVisibleMenuItems(raw string) json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return json.RawMessage("[]")
	}
	var items []struct {
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return json.RawMessage("[]")
	}
	var fullItems []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fullItems); err != nil {
		return json.RawMessage("[]")
	}
	var filtered []json.RawMessage
	for i, item := range items {
		if item.Visibility != "admin" {
			filtered = append(filtered, fullItems[i])
		}
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
func (s *SettingService) GetFrameSrcOrigins(ctx context.Context) ([]string, error) {
	settings, err := s.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var origins []string
	addOrigin := func(rawURL string) {
		if origin := extractOriginFromURL(rawURL); origin != "" {
			if _, ok := seen[origin]; !ok {
				seen[origin] = struct{}{}
				origins = append(origins, origin)
			}
		}
	}
	if settings.PurchaseSubscriptionEnabled {
		addOrigin(settings.PurchaseSubscriptionURL)
	}
	for _, item := range parseCustomMenuItemURLs(settings.CustomMenuItems) {
		addOrigin(item)
	}
	return origins, nil
}
func extractOriginFromURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
func (s *SettingService) IsRegistrationEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyRegistrationEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}
func (s *SettingService) IsEmailVerifyEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyEmailVerifyEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}
func (s *SettingService) GetRegistrationEmailSuffixWhitelist(ctx context.Context) []string {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyRegistrationEmailSuffixWhitelist)
	if err != nil {
		return []string{}
	}
	return ParseRegistrationEmailSuffixWhitelist(value)
}
func (s *SettingService) IsPromoCodeEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyPromoCodeEnabled)
	if err != nil {
		return true
	}
	return value != "false"
}
func (s *SettingService) IsInvitationCodeEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyInvitationCodeEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}
func (s *SettingService) IsPasswordResetEnabled(ctx context.Context) bool {
	if !s.IsEmailVerifyEnabled(ctx) {
		return false
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyPasswordResetEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}
func (s *SettingService) IsTotpEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyTotpEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}
func (s *SettingService) IsTotpEncryptionKeyConfigured() bool {
	return s.cfg.Totp.EncryptionKeyConfigured
}
func (s *SettingService) GetSiteName(ctx context.Context) string {
	value, err := s.settingRepo.GetValue(ctx, SettingKeySiteName)
	if err != nil || value == "" {
		return "Sub2API"
	}
	return value
}
func (s *SettingService) GetDefaultConcurrency(ctx context.Context) int {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyDefaultConcurrency)
	if err != nil {
		return s.cfg.Default.UserConcurrency
	}
	if v, err := strconv.Atoi(value); err == nil && v > 0 {
		return v
	}
	return s.cfg.Default.UserConcurrency
}
func (s *SettingService) GetDefaultBalance(ctx context.Context) float64 {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyDefaultBalance)
	if err != nil {
		return s.cfg.Default.UserBalance
	}
	if v, err := strconv.ParseFloat(value, 64); err == nil && v >= 0 {
		return v
	}
	return s.cfg.Default.UserBalance
}
func (s *SettingService) GetDefaultSubscriptions(ctx context.Context) []DefaultSubscriptionSetting {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyDefaultSubscriptions)
	if err != nil {
		return nil
	}
	return parseDefaultSubscriptions(value)
}
func (s *SettingService) InitializeDefaultSettings(ctx context.Context) error {
	_, err := s.settingRepo.GetValue(ctx, SettingKeyRegistrationEnabled)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrSettingNotFound) {
		return fmt.Errorf("check existing settings: %w", err)
	}
	defaults := map[string]string{SettingKeyRegistrationEnabled: "true", SettingKeyEmailVerifyEnabled: "false", SettingKeyRegistrationEmailSuffixWhitelist: "[]", SettingKeyPromoCodeEnabled: "true", SettingKeySiteName: "Sub2API", SettingKeySiteLogo: "", SettingKeyPurchaseSubscriptionEnabled: "false", SettingKeyPurchaseSubscriptionURL: "", SettingKeySoraClientEnabled: "false", SettingKeyCustomMenuItems: "[]", SettingKeyDefaultConcurrency: strconv.Itoa(s.cfg.Default.UserConcurrency), SettingKeyDefaultBalance: strconv.FormatFloat(s.cfg.Default.UserBalance, 'f', 8, 64), SettingKeyDefaultSubscriptions: "[]", SettingKeySMTPPort: "587", SettingKeySMTPUseTLS: "false", SettingKeyEnableModelFallback: "false", SettingKeyFallbackModelAnthropic: "claude-3-5-sonnet-20241022", SettingKeyFallbackModelOpenAI: "gpt-4o", SettingKeyFallbackModelGemini: "gemini-2.5-pro", SettingKeyFallbackModelAntigravity: "gemini-2.5-pro", SettingKeyEnableIdentityPatch: "true", SettingKeyIdentityPatchPrompt: "", SettingKeyOpsMonitoringEnabled: "true", SettingKeyOpsRealtimeMonitoringEnabled: "true", SettingKeyOpsQueryModeDefault: "auto", SettingKeyOpsMetricsIntervalSeconds: "60", SettingKeyMinClaudeCodeVersion: "", SettingKeyAllowUngroupedKeyScheduling: "false", SettingKeyTelegramChatID: ""}
	return s.settingRepo.SetMultiple(ctx, defaults)
}
func (s *SettingService) parseSettings(settings map[string]string) *SystemSettings {
	emailVerifyEnabled := settings[SettingKeyEmailVerifyEnabled] == "true"
	result := &SystemSettings{RegistrationEnabled: settings[SettingKeyRegistrationEnabled] == "true", EmailVerifyEnabled: emailVerifyEnabled, RegistrationEmailSuffixWhitelist: ParseRegistrationEmailSuffixWhitelist(settings[SettingKeyRegistrationEmailSuffixWhitelist]), PromoCodeEnabled: settings[SettingKeyPromoCodeEnabled] != "false", PasswordResetEnabled: emailVerifyEnabled && settings[SettingKeyPasswordResetEnabled] == "true", InvitationCodeEnabled: settings[SettingKeyInvitationCodeEnabled] == "true", TotpEnabled: settings[SettingKeyTotpEnabled] == "true", SMTPHost: settings[SettingKeySMTPHost], SMTPUsername: settings[SettingKeySMTPUsername], SMTPFrom: settings[SettingKeySMTPFrom], SMTPFromName: settings[SettingKeySMTPFromName], SMTPUseTLS: settings[SettingKeySMTPUseTLS] == "true", SMTPPasswordConfigured: settings[SettingKeySMTPPassword] != "", TelegramChatID: strings.TrimSpace(settings[SettingKeyTelegramChatID]), TurnstileEnabled: settings[SettingKeyTurnstileEnabled] == "true", TurnstileSiteKey: settings[SettingKeyTurnstileSiteKey], TurnstileSecretKeyConfigured: settings[SettingKeyTurnstileSecretKey] != "", SiteName: s.getStringOrDefault(settings, SettingKeySiteName, "Sub2API"), SiteLogo: settings[SettingKeySiteLogo], SiteSubtitle: s.getStringOrDefault(settings, SettingKeySiteSubtitle, "Subscription to API Conversion Platform"), APIBaseURL: settings[SettingKeyAPIBaseURL], ContactInfo: settings[SettingKeyContactInfo], DocURL: settings[SettingKeyDocURL], HomeContent: settings[SettingKeyHomeContent], HideCcsImportButton: settings[SettingKeyHideCcsImportButton] == "true", PurchaseSubscriptionEnabled: settings[SettingKeyPurchaseSubscriptionEnabled] == "true", PurchaseSubscriptionURL: strings.TrimSpace(settings[SettingKeyPurchaseSubscriptionURL]), SoraClientEnabled: settings[SettingKeySoraClientEnabled] == "true", CustomMenuItems: settings[SettingKeyCustomMenuItems]}
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
	result.OpsMetricsIntervalSeconds = 60
	if raw := strings.TrimSpace(settings[SettingKeyOpsMetricsIntervalSeconds]); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			if v < 60 {
				v = 60
			}
			if v > 3600 {
				v = 3600
			}
			result.OpsMetricsIntervalSeconds = v
		}
	}
	result.MinClaudeCodeVersion = settings[SettingKeyMinClaudeCodeVersion]
	result.AllowUngroupedKeyScheduling = settings[SettingKeyAllowUngroupedKeyScheduling] == "true"
	return result
}
func isFalseSettingValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "off", "disabled":
		return true
	default:
		return false
	}
}
func parseDefaultSubscriptions(raw string) []DefaultSubscriptionSetting {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []DefaultSubscriptionSetting
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	normalized := make([]DefaultSubscriptionSetting, 0, len(items))
	for _, item := range items {
		if item.GroupID <= 0 || item.ValidityDays <= 0 {
			continue
		}
		if item.ValidityDays > MaxValidityDays {
			item.ValidityDays = MaxValidityDays
		}
		normalized = append(normalized, item)
	}
	return normalized
}

func maskTelegramBotToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 10 {
		return strings.Repeat("*", len(token))
	}
	return token[:6] + "..." + token[len(token)-4:]
}
func (s *SettingService) getStringOrDefault(settings map[string]string, key, defaultValue string) string {
	if value, ok := settings[key]; ok && value != "" {
		return value
	}
	return defaultValue
}
func (s *SettingService) IsTurnstileEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyTurnstileEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}
func (s *SettingService) GetTurnstileSecretKey(ctx context.Context) string {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyTurnstileSecretKey)
	if err != nil {
		return ""
	}
	return value
}
func (s *SettingService) IsIdentityPatchEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyEnableIdentityPatch)
	if err != nil {
		return true
	}
	return value == "true"
}
func (s *SettingService) GetIdentityPatchPrompt(ctx context.Context) string {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyIdentityPatchPrompt)
	if err != nil {
		return ""
	}
	return value
}
func (s *SettingService) GenerateAdminAPIKey(ctx context.Context) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	key := AdminAPIKeyPrefix + hex.EncodeToString(bytes)
	if err := s.settingRepo.Set(ctx, SettingKeyAdminAPIKey, key); err != nil {
		return "", fmt.Errorf("save admin api key: %w", err)
	}
	return key, nil
}
func (s *SettingService) GetAdminAPIKeyStatus(ctx context.Context) (maskedKey string, exists bool, err error) {
	key, err := s.settingRepo.GetValue(ctx, SettingKeyAdminAPIKey)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	if key == "" {
		return "", false, nil
	}
	if len(key) > 14 {
		maskedKey = key[:10] + "..." + key[len(key)-4:]
	} else {
		maskedKey = key
	}
	return maskedKey, true, nil
}
func (s *SettingService) GetAdminAPIKey(ctx context.Context) (string, error) {
	key, err := s.settingRepo.GetValue(ctx, SettingKeyAdminAPIKey)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return "", nil
		}
		return "", err
	}
	return key, nil
}
func (s *SettingService) DeleteAdminAPIKey(ctx context.Context) error {
	return s.settingRepo.Delete(ctx, SettingKeyAdminAPIKey)
}
func (s *SettingService) IsModelFallbackEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyEnableModelFallback)
	if err != nil {
		return false
	}
	return value == "true"
}
func (s *SettingService) GetFallbackModel(ctx context.Context, platform string) string {
	var key string
	var defaultModel string
	switch platform {
	case PlatformAnthropic:
		key = SettingKeyFallbackModelAnthropic
		defaultModel = "claude-3-5-sonnet-20241022"
	case PlatformOpenAI:
		key = SettingKeyFallbackModelOpenAI
		defaultModel = "gpt-4o"
	case PlatformGemini:
		key = SettingKeyFallbackModelGemini
		defaultModel = "gemini-2.5-pro"
	case PlatformAntigravity:
		key = SettingKeyFallbackModelAntigravity
		defaultModel = "gemini-2.5-pro"
	default:
		return ""
	}
	value, err := s.settingRepo.GetValue(ctx, key)
	if err != nil || value == "" {
		return defaultModel
	}
	return value
}
func (s *SettingService) GetLinuxDoConnectOAuthConfig(ctx context.Context) (config.LinuxDoConnectConfig, error) {
	if s == nil || s.cfg == nil {
		return config.LinuxDoConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}
	effective := s.cfg.LinuxDo
	keys := []string{SettingKeyLinuxDoConnectEnabled, SettingKeyLinuxDoConnectClientID, SettingKeyLinuxDoConnectClientSecret, SettingKeyLinuxDoConnectRedirectURL}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return config.LinuxDoConnectConfig{}, fmt.Errorf("get linuxdo connect settings: %w", err)
	}
	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		effective.Enabled = raw == "true"
	}
	if v, ok := settings[SettingKeyLinuxDoConnectClientID]; ok && strings.TrimSpace(v) != "" {
		effective.ClientID = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyLinuxDoConnectClientSecret]; ok && strings.TrimSpace(v) != "" {
		effective.ClientSecret = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyLinuxDoConnectRedirectURL]; ok && strings.TrimSpace(v) != "" {
		effective.RedirectURL = strings.TrimSpace(v)
	}
	if !effective.Enabled {
		return config.LinuxDoConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	if strings.TrimSpace(effective.ClientID) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client id not configured")
	}
	if strings.TrimSpace(effective.AuthorizeURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url not configured")
	}
	if strings.TrimSpace(effective.TokenURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url not configured")
	}
	if strings.TrimSpace(effective.UserInfoURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url not configured")
	}
	if strings.TrimSpace(effective.RedirectURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured")
	}
	if strings.TrimSpace(effective.FrontendRedirectURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url not configured")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.AuthorizeURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.TokenURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.UserInfoURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.RedirectURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url invalid")
	}
	if err := config.ValidateFrontendRedirectURL(effective.FrontendRedirectURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url invalid")
	}
	method := strings.ToLower(strings.TrimSpace(effective.TokenAuthMethod))
	switch method {
	case "", "client_secret_post", "client_secret_basic":
		if strings.TrimSpace(effective.ClientSecret) == "" {
			return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client secret not configured")
		}
	case "none":
		if !effective.UsePKCE {
			return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth pkce must be enabled when token_auth_method=none")
		}
	default:
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token_auth_method invalid")
	}
	return effective, nil
}
func (s *SettingService) GetStreamTimeoutSettings(ctx context.Context) (*StreamTimeoutSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyStreamTimeoutSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultStreamTimeoutSettings(), nil
		}
		return nil, fmt.Errorf("get stream timeout settings: %w", err)
	}
	if value == "" {
		return DefaultStreamTimeoutSettings(), nil
	}
	var settings StreamTimeoutSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultStreamTimeoutSettings(), nil
	}
	if settings.TempUnschedMinutes < 1 {
		settings.TempUnschedMinutes = 1
	}
	if settings.TempUnschedMinutes > 60 {
		settings.TempUnschedMinutes = 60
	}
	if settings.ThresholdCount < 1 {
		settings.ThresholdCount = 1
	}
	if settings.ThresholdCount > 10 {
		settings.ThresholdCount = 10
	}
	if settings.ThresholdWindowMinutes < 1 {
		settings.ThresholdWindowMinutes = 1
	}
	if settings.ThresholdWindowMinutes > 60 {
		settings.ThresholdWindowMinutes = 60
	}
	switch settings.Action {
	case StreamTimeoutActionTempUnsched, StreamTimeoutActionError, StreamTimeoutActionNone:
	default:
		settings.Action = StreamTimeoutActionTempUnsched
	}
	return &settings, nil
}
func (s *SettingService) IsUngroupedKeySchedulingAllowed(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAllowUngroupedKeyScheduling)
	if err != nil {
		return false
	}
	return value == "true"
}
