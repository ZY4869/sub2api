package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func encodeBillingHoldBreakdown(parts map[string]service.BillingMoney) (string, error) {
	if len(parts) == 0 {
		return "{}", nil
	}
	payload := map[string]float64{}
	for currency, amount := range parts {
		currency = service.NormalizeUsageBillingCurrency(currency)
		if amount.IsZero() {
			continue
		}
		payload[currency] = amount.Float64()
	}
	if len(payload) == 0 {
		return "{}", nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeBillingHoldBreakdown(raw string) map[string]service.BillingMoney {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return nil
	}
	values := map[string]float64{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	out := map[string]service.BillingMoney{}
	for currency, value := range values {
		money, err := service.NewNonNegativeBillingMoneyFromFloat(value)
		if err != nil || money.IsZero() {
			continue
		}
		out[service.NormalizeUsageBillingCurrency(currency)] = money
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encodeBillingHoldConversionPolicy(settings service.BillingCurrencyConversionSettings) (string, error) {
	payload := map[string]any{
		"enabled":         settings.Enabled,
		"cny_to_usd_rate": settings.CNYToUSDRate,
		"usd_to_cny_rate": settings.USDToCNYRate,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeBillingHoldConversionPolicy(raw string) service.BillingCurrencyConversionSettings {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return service.DefaultBillingCurrencyConversionSettings()
	}
	var payload struct {
		Enabled      bool    `json:"enabled"`
		CNYToUSDRate float64 `json:"cny_to_usd_rate"`
		USDToCNYRate float64 `json:"usd_to_cny_rate"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return service.DefaultBillingCurrencyConversionSettings()
	}
	return service.NormalizeBillingCurrencyConversionSettings(service.BillingCurrencyConversionSettings{
		Enabled:      payload.Enabled,
		CNYToUSDRate: payload.CNYToUSDRate,
		USDToCNYRate: payload.USDToCNYRate,
	})
}

func billingHoldBreakdownFloatMap(parts map[string]service.BillingMoney) map[string]float64 {
	if len(parts) == 0 {
		return nil
	}
	out := map[string]float64{}
	for currency, amount := range parts {
		if amount.IsZero() {
			continue
		}
		out[service.NormalizeUsageBillingCurrency(currency)] = amount.Float64()
	}
	return out
}

func billingHoldBreakdownMoneyMap(values map[string]float64) map[string]service.BillingMoney {
	if len(values) == 0 {
		return nil
	}
	out := map[string]service.BillingMoney{}
	for currency, value := range values {
		money, err := service.NewNonNegativeBillingMoneyFromFloat(value)
		if err != nil || money.IsZero() {
			continue
		}
		out[service.NormalizeUsageBillingCurrency(currency)] = money
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func addBillingHoldBreakdown(left, right map[string]service.BillingMoney) (map[string]service.BillingMoney, error) {
	out := map[string]service.BillingMoney{}
	for currency, amount := range left {
		if amount.IsZero() {
			continue
		}
		out[service.NormalizeUsageBillingCurrency(currency)] = amount
	}
	for currency, amount := range right {
		currency = service.NormalizeUsageBillingCurrency(currency)
		if amount.IsZero() {
			continue
		}
		if existing, ok := out[currency]; ok {
			next, err := existing.Add(amount)
			if err != nil {
				return nil, err
			}
			out[currency] = next
		} else {
			out[currency] = amount
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func subtractBillingHoldBreakdown(left, right map[string]service.BillingMoney) (map[string]service.BillingMoney, error) {
	out := map[string]service.BillingMoney{}
	for currency, amount := range left {
		currency = service.NormalizeUsageBillingCurrency(currency)
		if amount.IsZero() {
			continue
		}
		next := amount
		if refund, ok := right[currency]; ok && !refund.IsZero() {
			value, err := amount.Sub(refund)
			if err != nil {
				return nil, err
			}
			next = value
		}
		if next.IsPositive() {
			out[currency] = next
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func applyBillingHoldBalanceDeltas(ctx context.Context, tx *sql.Tx, userID int64, parts map[string]service.BillingMoney, sign int) error {
	for currency, amount := range parts {
		if amount.IsZero() {
			continue
		}
		delta := amount
		var err error
		if sign < 0 {
			delta, err = amount.Neg()
			if err != nil {
				return err
			}
		}
		if err := addUsageBillingWalletBalance(ctx, tx, userID, currency, delta, service.NormalizeUsageBillingCurrency(currency) == service.ModelPricingCurrencyUSD); err != nil {
			return err
		}
	}
	return nil
}

func prorateBillingHoldBreakdown(parts map[string]service.BillingMoney, numerator, denominator service.BillingMoney) map[string]service.BillingMoney {
	if len(parts) == 0 || numerator.IsZero() || denominator.IsZero() {
		return nil
	}
	ratio := numerator.Float64() / denominator.Float64()
	if ratio <= 0 {
		return nil
	}
	if ratio > 1 {
		ratio = 1
	}
	out := map[string]service.BillingMoney{}
	for currency, amount := range parts {
		value := math.Min(amount.Float64(), amount.Float64()*ratio)
		money, err := service.NewNonNegativeBillingMoneyFromFloat(value)
		if err != nil || money.IsZero() {
			continue
		}
		out[service.NormalizeUsageBillingCurrency(currency)] = money
	}
	return out
}
