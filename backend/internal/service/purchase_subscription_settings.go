package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	PurchaseSubscriptionProviderCustom    = "custom"
	PurchaseSubscriptionProviderAirwallex = "airwallex"

	PurchaseSubscriptionPaymentEnvProduction = "production"
	PurchaseSubscriptionPaymentEnvSandbox    = "sandbox"
)

var (
	purchaseSubscriptionCurrencyPattern    = regexp.MustCompile(`^[A-Z]{3}$`)
	purchaseSubscriptionCountryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)
	purchaseSubscriptionParamKeyPattern    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]{0,63}$`)
)

var purchaseSubscriptionBlockedParamKeys = map[string]struct{}{
	"token":         {},
	"access_token":  {},
	"refresh_token": {},
	"api_key":       {},
	"apikey":        {},
	"authorization": {},
	"user_id":       {},
	"src_host":      {},
	"src_url":       {},
}

func NormalizePurchaseSubscriptionProvider(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", PurchaseSubscriptionProviderCustom:
		return PurchaseSubscriptionProviderCustom
	case PurchaseSubscriptionProviderAirwallex:
		return PurchaseSubscriptionProviderAirwallex
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func NormalizePurchaseSubscriptionPaymentEnv(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), PurchaseSubscriptionPaymentEnvSandbox) {
		return PurchaseSubscriptionPaymentEnvSandbox
	}
	return PurchaseSubscriptionPaymentEnvProduction
}

func NormalizePurchaseSubscriptionCurrencies(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(items))
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		currency := strings.ToUpper(strings.TrimSpace(item))
		if !purchaseSubscriptionCurrencyPattern.MatchString(currency) {
			continue
		}
		if _, exists := seen[currency]; exists {
			continue
		}
		seen[currency] = struct{}{}
		normalized = append(normalized, currency)
	}

	if len(normalized) == 0 {
		return []string{}
	}
	return normalized
}

func ParsePurchaseSubscriptionCurrencies(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}

	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err == nil {
		return NormalizePurchaseSubscriptionCurrencies(items)
	}

	return NormalizePurchaseSubscriptionCurrencies(strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	}))
}

func MarshalPurchaseSubscriptionCurrencies(items []string) (string, error) {
	normalized := NormalizePurchaseSubscriptionCurrencies(items)
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func NormalizePurchaseSubscriptionCountryCode(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func NormalizePurchaseSubscriptionExtraParams(input map[string]string) (map[string]string, error) {
	if len(input) == 0 {
		return map[string]string{}, nil
	}
	if len(input) > 16 {
		return nil, fmt.Errorf("purchase extra params cannot exceed 16 entries")
	}

	normalized := make(map[string]string, len(input))
	for rawKey, rawValue := range input {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			continue
		}
		if !purchaseSubscriptionParamKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("purchase extra param key %q is invalid", key)
		}
		if _, blocked := purchaseSubscriptionBlockedParamKeys[strings.ToLower(key)]; blocked {
			return nil, fmt.Errorf("purchase extra param key %q is not allowed", key)
		}

		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}
		if len(value) > 200 {
			return nil, fmt.Errorf("purchase extra param %q is too long", key)
		}
		normalized[key] = value
	}

	if len(normalized) == 0 {
		return map[string]string{}, nil
	}
	return normalized, nil
}

func ParsePurchaseSubscriptionExtraParams(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return map[string]string{}
	}

	normalized, err := NormalizePurchaseSubscriptionExtraParams(parsed)
	if err != nil {
		return map[string]string{}
	}
	return normalized
}

func MarshalPurchaseSubscriptionExtraParams(input map[string]string) (string, error) {
	normalized, err := NormalizePurchaseSubscriptionExtraParams(input)
	if err != nil {
		return "", err
	}
	if len(normalized) == 0 {
		return "{}", nil
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ValidatePurchaseSubscriptionSettings(
	provider string,
	supportedCurrencies []string,
	defaultCurrency string,
	defaultCountryCode string,
	paymentEnv string,
	extraParams map[string]string,
) error {
	switch provider {
	case PurchaseSubscriptionProviderCustom, PurchaseSubscriptionProviderAirwallex:
	default:
		return fmt.Errorf("purchase provider %q is unsupported", provider)
	}

	if defaultCurrency != "" && !purchaseSubscriptionCurrencyPattern.MatchString(defaultCurrency) {
		return fmt.Errorf("purchase default currency must be a 3-letter ISO code")
	}

	if len(supportedCurrencies) > 0 && defaultCurrency != "" {
		matched := false
		for _, item := range supportedCurrencies {
			if item == defaultCurrency {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("purchase default currency must be included in supported currencies")
		}
	}

	if defaultCountryCode != "" && !purchaseSubscriptionCountryCodePattern.MatchString(defaultCountryCode) {
		return fmt.Errorf("purchase default country code must be a 2-letter ISO code")
	}

	switch paymentEnv {
	case PurchaseSubscriptionPaymentEnvProduction, PurchaseSubscriptionPaymentEnvSandbox:
	default:
		return fmt.Errorf("purchase payment env %q is unsupported", paymentEnv)
	}

	if _, err := NormalizePurchaseSubscriptionExtraParams(extraParams); err != nil {
		return err
	}

	return nil
}
