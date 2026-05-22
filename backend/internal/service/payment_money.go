package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

var paymentCurrencyExponents = map[string]int{
	"USD": 2,
	"CNY": 2,
	"HKD": 2,
	"EUR": 2,
	"GBP": 2,
	"JPY": 0,
}

func NormalizePaymentCurrency(input string) string {
	currency := strings.ToUpper(strings.TrimSpace(input))
	if _, ok := paymentCurrencyExponents[currency]; !ok {
		return ""
	}
	return currency
}

func DefaultPaymentCurrencies() []string {
	return []string{"USD", "CNY", "HKD"}
}

func PaymentCurrencyExponent(currency string) int {
	if v, ok := paymentCurrencyExponents[NormalizePaymentCurrency(currency)]; ok {
		return v
	}
	return 2
}

func PaymentAmountToMinor(amount float64, currency string) (int64, error) {
	if amount <= 0 {
		return 0, ErrPaymentInvalidAmount
	}
	exp := PaymentCurrencyExponent(currency)
	scale := math.Pow10(exp)
	return int64(math.Round(amount * scale)), nil
}

func PaymentMinorToAmount(minor int64, currency string) float64 {
	exp := PaymentCurrencyExponent(currency)
	scale := math.Pow10(exp)
	return float64(minor) / scale
}

func FormatPaymentAmount(minor int64, currency string) string {
	exp := PaymentCurrencyExponent(currency)
	amount := PaymentMinorToAmount(minor, currency)
	return strconv.FormatFloat(amount, 'f', exp, 64) + " " + NormalizePaymentCurrency(currency)
}

func NormalizePaymentAllowedCurrencies(input []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(input))
	for _, item := range input {
		currency := NormalizePaymentCurrency(item)
		if currency == "" {
			continue
		}
		if _, ok := seen[currency]; ok {
			continue
		}
		seen[currency] = struct{}{}
		out = append(out, currency)
	}
	if len(out) == 0 {
		return DefaultPaymentCurrencies()
	}
	return out
}

func PaymentCurrencyAllowed(currency string, allowed []string) bool {
	currency = NormalizePaymentCurrency(currency)
	if currency == "" {
		return false
	}
	for _, item := range NormalizePaymentAllowedCurrencies(allowed) {
		if item == currency {
			return true
		}
	}
	return false
}

func validatePaymentCurrency(currency string, allowed []string) (string, error) {
	normalized := NormalizePaymentCurrency(currency)
	if normalized == "" {
		return "", ErrPaymentUnsupportedCurrency
	}
	if !PaymentCurrencyAllowed(normalized, allowed) {
		return "", ErrPaymentUnsupportedCurrency.WithMetadata(map[string]string{"currency": normalized})
	}
	return normalized, nil
}

func paymentOrderNo(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return fmt.Sprintf("pay_%s", id)
}
