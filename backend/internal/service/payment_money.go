package service

import (
	"fmt"
	"math"
	"math/big"
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
	money, err := NewPositiveBillingMoneyFromFloat(amount)
	if err != nil {
		return 0, ErrPaymentInvalidAmount
	}
	minor, err := billingMoneyToPaymentMinor(money, currency)
	if err != nil || minor <= 0 {
		return 0, ErrPaymentInvalidAmount
	}
	return minor, nil
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

func NormalizePaymentAmountToCurrency(amount float64, currency string) (float64, error) {
	minor, err := PaymentAmountToMinor(amount, currency)
	if err != nil {
		return 0, err
	}
	return PaymentMinorToAmount(minor, currency), nil
}

func billingMoneyToPaymentMinor(money BillingMoney, currency string) (int64, error) {
	exp := PaymentCurrencyExponent(currency)
	scaleUnits := int64(1)
	for i := 0; i < exp; i++ {
		scaleUnits *= 10
	}
	// BillingMoney uses 8 decimal places. Payment minor units are currency-specific.
	num := new(big.Int).Mul(big.NewInt(money.Units()), big.NewInt(scaleUnits))
	den := big.NewInt(BillingAmountScale)
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(num, den, rem)
	if new(big.Int).Abs(rem).Mul(new(big.Int).Abs(rem), big.NewInt(2)).Cmp(den) >= 0 {
		if num.Sign() >= 0 {
			q.Add(q, big.NewInt(1))
		} else {
			q.Sub(q, big.NewInt(1))
		}
	}
	if !q.IsInt64() || q.Cmp(big.NewInt(math.MaxInt64)) > 0 || q.Cmp(big.NewInt(math.MinInt64)) < 0 {
		return 0, ErrPaymentInvalidAmount
	}
	return q.Int64(), nil
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
