package service

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

func DefaultPaymentSettings() PaymentSettings {
	return PaymentSettings{
		AllowedCurrencies: NormalizePaymentAllowedCurrencies(nil),
		DefaultCurrency:   "USD",
		MinTopupAmount:    1,
		MaxTopupAmount:    5000,
	}
}

var antigravityUserAgentVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[A-Za-z0-9._-]+)?$`)

const (
	PaymentModeDefault      = "default"
	PaymentModeQRCode       = "qrcode"
	CodexOAuthUAModeDefault = "default"
	CodexOAuthUAModeForce   = "force"
	CodexOAuthUAModeCustom  = "custom"
)

func (s *SettingService) GetAntigravityUserAgentVersion(ctx context.Context) string {
	if s == nil || s.settingRepo == nil {
		return ""
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAntigravityUserAgentVersion)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func (s *SettingService) GetCodexOAuthUserAgentPolicy(ctx context.Context) CodexOAuthUserAgentPolicy {
	if s == nil || s.settingRepo == nil {
		return NormalizeCodexOAuthUserAgentPolicy("", "")
	}
	raw, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyCodexOAuthUserAgentMode,
		SettingKeyCodexOAuthUserAgentOverride,
	})
	if err != nil {
		return NormalizeCodexOAuthUserAgentPolicy("", "")
	}
	return NormalizeCodexOAuthUserAgentPolicy(raw[SettingKeyCodexOAuthUserAgentMode], raw[SettingKeyCodexOAuthUserAgentOverride])
}

func NormalizeAntigravityUserAgentVersion(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if !antigravityUserAgentVersionPattern.MatchString(value) {
		return "", false
	}
	return value, true
}

type CodexOAuthUserAgentPolicy struct {
	Mode      string
	Override  string
	Force     bool
	HasCustom bool
}

func NormalizeCodexOAuthUserAgentPolicy(mode string, override string) CodexOAuthUserAgentPolicy {
	normalizedOverride := sanitizeCodexOAuthUserAgentOverride(override)
	normalizedMode := strings.ToLower(strings.TrimSpace(mode))
	switch normalizedMode {
	case CodexOAuthUAModeForce:
		return CodexOAuthUserAgentPolicy{Mode: CodexOAuthUAModeForce, Override: normalizedOverride, Force: true, HasCustom: normalizedOverride != ""}
	case CodexOAuthUAModeCustom:
		if normalizedOverride == "" {
			return CodexOAuthUserAgentPolicy{Mode: CodexOAuthUAModeDefault, Override: "", Force: false}
		}
		return CodexOAuthUserAgentPolicy{Mode: CodexOAuthUAModeCustom, Override: normalizedOverride, Force: true, HasCustom: true}
	default:
		return CodexOAuthUserAgentPolicy{Mode: CodexOAuthUAModeDefault, Override: normalizedOverride, Force: false, HasCustom: normalizedOverride != ""}
	}
}

func sanitizeCodexOAuthUserAgentOverride(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t':
			return -1
		default:
			return r
		}
	}, value)
	if len(value) > 256 {
		value = value[:256]
	}
	return strings.TrimSpace(value)
}

func (s *SettingService) GetPaymentSettings(ctx context.Context) PaymentSettings {
	defaults := DefaultPaymentSettings()
	if s == nil || s.settingRepo == nil {
		return defaults
	}
	keys := []string{
		SettingKeyPurchaseSubscriptionEnabled,
		SettingKeyPaymentProviderAirwallexEnabled,
		SettingKeyAirwallexEnv,
		SettingKeyAirwallexClientID,
		SettingKeyAirwallexAPIKey,
		SettingKeyAirwallexWebhookSecret,
		SettingKeyPaymentMobileForceQRCodeEnabled,
		SettingKeyPaymentAllowedCurrencies,
		SettingKeyPaymentDefaultCurrency,
		SettingKeyPaymentMinTopupAmount,
		SettingKeyPaymentMaxTopupAmount,
		SettingKeyPaymentSubscriptionPlans,
	}
	raw, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return defaults
	}
	out := paymentSettingsFromRaw(raw)
	out.FrontendURL = s.GetFrontendURL(ctx)
	return out
}

func paymentSettingsFromRaw(raw map[string]string) PaymentSettings {
	out := DefaultPaymentSettings()
	if raw == nil {
		return out
	}
	out.Enabled = raw[SettingKeyPurchaseSubscriptionEnabled] == "true"
	out.AirwallexEnabled = raw[SettingKeyPaymentProviderAirwallexEnabled] == "true"
	out.AirwallexEnv = NormalizeAirwallexEnv(raw[SettingKeyAirwallexEnv])
	out.AirwallexClientID = strings.TrimSpace(raw[SettingKeyAirwallexClientID])
	out.AirwallexAPIKey = strings.TrimSpace(raw[SettingKeyAirwallexAPIKey])
	out.AirwallexAPIKeyConfigured = out.AirwallexAPIKey != ""
	out.AirwallexWebhookSecret = strings.TrimSpace(raw[SettingKeyAirwallexWebhookSecret])
	out.AirwallexWebhookSecretConfigured = out.AirwallexWebhookSecret != ""
	out.MobileForceQRCodeEnabled = raw[SettingKeyPaymentMobileForceQRCodeEnabled] == "true"
	out.AllowedCurrencies = parsePaymentCurrencies(raw[SettingKeyPaymentAllowedCurrencies])
	out.DefaultCurrency = NormalizePaymentCurrency(raw[SettingKeyPaymentDefaultCurrency])
	if out.DefaultCurrency == "" || !PaymentCurrencyAllowed(out.DefaultCurrency, out.AllowedCurrencies) {
		out.DefaultCurrency = out.AllowedCurrencies[0]
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(raw[SettingKeyPaymentMinTopupAmount]), 64); err == nil {
		if normalized, normErr := NormalizePaymentAmountToCurrency(v, out.DefaultCurrency); normErr == nil && normalized > 0 {
			out.MinTopupAmount = normalized
		}
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(raw[SettingKeyPaymentMaxTopupAmount]), 64); err == nil {
		if normalized, normErr := NormalizePaymentAmountToCurrency(v, out.DefaultCurrency); normErr == nil && normalized > 0 {
			out.MaxTopupAmount = normalized
		}
	}
	if out.MaxTopupAmount < out.MinTopupAmount {
		out.MaxTopupAmount = out.MinTopupAmount
	}
	out.SubscriptionPlans = ParsePaymentSubscriptionPlans(raw[SettingKeyPaymentSubscriptionPlans])
	return out
}

func IsPaymentPublicAirwallexEnabled(settings PaymentSettings) bool {
	return isPaymentPublicAirwallexEnabled(settings, true)
}

func IsPaymentPublicAirwallexEnabledWithoutSecrets(settings PaymentSettings) bool {
	return isPaymentPublicAirwallexEnabled(settings, false)
}

func isPaymentPublicAirwallexEnabled(settings PaymentSettings, requireProviderCredentials bool) bool {
	if !settings.Enabled || !settings.AirwallexEnabled {
		return false
	}
	if requireProviderCredentials && (strings.TrimSpace(settings.AirwallexClientID) == "" || !settings.AirwallexAPIKeyConfigured) {
		return false
	}
	if len(settings.AllowedCurrencies) == 0 || !PaymentCurrencyAllowed(settings.DefaultCurrency, settings.AllowedCurrencies) {
		return false
	}
	if settings.MinTopupAmount <= 0 || settings.MaxTopupAmount < settings.MinTopupAmount {
		return false
	}
	return true
}

func NormalizeAirwallexEnv(input string) string {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "prod", "production":
		return "prod"
	default:
		return "demo"
	}
}

func parsePaymentCurrencies(raw string) []string {
	var items []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &items); err != nil {
		return DefaultPaymentCurrencies()
	}
	return NormalizePaymentAllowedCurrencies(items)
}

func ParsePaymentSubscriptionPlans(raw string) []PaymentSubscriptionPlan {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var plans []PaymentSubscriptionPlan
	if err := json.Unmarshal([]byte(raw), &plans); err != nil {
		return nil
	}
	out := make([]PaymentSubscriptionPlan, 0, len(plans))
	for _, plan := range plans {
		plan.PlanID = strings.TrimSpace(plan.PlanID)
		plan.Name = strings.TrimSpace(plan.Name)
		if plan.PlanID == "" || plan.GroupID <= 0 || plan.ValidityDays <= 0 {
			continue
		}
		if plan.PricesByCurrency == nil {
			plan.PricesByCurrency = map[string]float64{}
		}
		normalizedPrices := make(map[string]float64, len(plan.PricesByCurrency))
		for currency, price := range plan.PricesByCurrency {
			normalized := NormalizePaymentCurrency(currency)
			if normalized == "" {
				continue
			}
			normalizedPrice, err := NormalizePaymentAmountToCurrency(price, normalized)
			if err != nil || normalizedPrice <= 0 {
				continue
			}
			normalizedPrices[normalized] = normalizedPrice
		}
		plan.PricesByCurrency = normalizedPrices
		out = append(out, plan)
	}
	return out
}

func MarshalPaymentSubscriptionPlans(plans []PaymentSubscriptionPlan) string {
	normalized := ParsePaymentSubscriptionPlans(string(mustJSON(plans)))
	data, err := json.Marshal(normalized)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte("[]")
	}
	return data
}

func findPaymentSubscriptionPlan(plans []PaymentSubscriptionPlan, planID string) (PaymentSubscriptionPlan, bool) {
	planID = strings.TrimSpace(planID)
	for _, plan := range plans {
		if plan.Enabled && plan.PlanID == planID {
			return plan, true
		}
	}
	return PaymentSubscriptionPlan{}, false
}
