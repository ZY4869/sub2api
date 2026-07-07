package service

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

type paymentOrderSnapshot struct {
	ProductType      string             `json:"product_type"`
	AmountMinor      int64              `json:"amount_minor"`
	Currency         string             `json:"currency"`
	CountryCode      string             `json:"country_code,omitempty"`
	PlanID           string             `json:"plan_id,omitempty"`
	GroupID          int64              `json:"group_id,omitempty"`
	ValidityDays     int                `json:"validity_days,omitempty"`
	PricesByCurrency map[string]float64 `json:"prices_by_currency,omitempty"`
	ConvertedFrom    string             `json:"converted_from,omitempty"`
	ConversionRate   float64            `json:"conversion_rate,omitempty"`
	CreatedAt        string             `json:"created_at"`
}

func (s *PaymentService) buildOrderSnapshot(settings PaymentSettings, input CreatePaymentOrderInput, currency string) (json.RawMessage, int64, error) {
	switch input.ProductType {
	case PaymentProductBalanceTopup:
		amountMinor, err := PaymentAmountToMinor(input.Amount, currency)
		if err != nil {
			return nil, 0, err
		}
		minMinor, err := PaymentAmountToMinor(settings.MinTopupAmount, currency)
		if err != nil {
			return nil, 0, ErrPaymentInvalidAmount
		}
		maxMinor, err := PaymentAmountToMinor(settings.MaxTopupAmount, currency)
		if err != nil {
			return nil, 0, ErrPaymentInvalidAmount
		}
		if amountMinor < minMinor || amountMinor > maxMinor {
			return nil, 0, ErrPaymentInvalidAmount
		}
		return marshalPaymentSnapshot(paymentOrderSnapshot{ProductType: input.ProductType, AmountMinor: amountMinor, Currency: currency, CountryCode: input.CountryCode, CreatedAt: time.Now().UTC().Format(time.RFC3339)}), amountMinor, nil
	case PaymentProductSubscription:
		plan, ok := findPaymentSubscriptionPlan(settings.SubscriptionPlans, input.PlanID)
		if !ok {
			return nil, 0, ErrPaymentInvalidProduct
		}
		price, convertedFrom, conversionRate := subscriptionPriceForCurrency(plan, currency, settings.SubscriptionUSDToCNYRate)
		if price <= 0 {
			return nil, 0, ErrPaymentUnsupportedCurrency.WithMetadata(map[string]string{"currency": currency})
		}
		amountMinor, err := PaymentAmountToMinor(price, currency)
		if err != nil {
			return nil, 0, err
		}
		return marshalPaymentSnapshot(paymentOrderSnapshot{ProductType: input.ProductType, AmountMinor: amountMinor, Currency: currency, CountryCode: input.CountryCode, PlanID: plan.PlanID, GroupID: plan.GroupID, ValidityDays: plan.ValidityDays, PricesByCurrency: plan.PricesByCurrency, ConvertedFrom: convertedFrom, ConversionRate: conversionRate, CreatedAt: time.Now().UTC().Format(time.RFC3339)}), amountMinor, nil
	default:
		return nil, 0, ErrPaymentInvalidProduct
	}
}

func subscriptionPriceForCurrency(plan PaymentSubscriptionPlan, currency string, usdToCNYRate float64) (float64, string, float64) {
	currency = NormalizePaymentCurrency(currency)
	if currency == "" {
		return 0, "", 0
	}
	if price := plan.PricesByCurrency[currency]; price > 0 {
		return price, "", 0
	}
	rate := normalizeSubscriptionUSDToCNYRate(usdToCNYRate)
	if currency != "CNY" || rate <= 0 {
		return 0, "", 0
	}
	usdPrice := plan.PricesByCurrency["USD"]
	if usdPrice <= 0 {
		return 0, "", 0
	}
	converted, err := NormalizePaymentAmountToCurrency(usdPrice*rate, "CNY")
	if err != nil || converted <= 0 {
		return 0, "", 0
	}
	return converted, "USD", rate
}

func marshalPaymentSnapshot(snapshot paymentOrderSnapshot) json.RawMessage {
	data, _ := json.Marshal(snapshot)
	return data
}

func resolvePaymentReturnURL(inputReturnURL string, frontendURL string, orderNo string) string {
	if returnURL := normalizePaymentReturnURL(inputReturnURL); returnURL != "" {
		return strings.ReplaceAll(returnURL, "__ORDER_NO__", url.PathEscape(strings.TrimSpace(orderNo)))
	}
	return buildPaymentResultReturnURL(frontendURL, orderNo)
}

func buildPaymentResultReturnURL(frontendURL string, orderNo string) string {
	frontendURL = strings.TrimSpace(frontendURL)
	orderNo = strings.TrimSpace(orderNo)
	if frontendURL == "" || orderNo == "" {
		return ""
	}
	base, err := url.Parse(frontendURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return ""
	}
	base.RawQuery = ""
	base.Fragment = ""
	base.Path = strings.TrimRight(base.Path, "/") + "/payment/result/" + url.PathEscape(orderNo)
	return base.String()
}

func normalizePaymentReturnURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return parsed.String()
}
