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
	CreatedAt        string             `json:"created_at"`
}

func (s *PaymentService) buildOrderSnapshot(settings PaymentSettings, input CreatePaymentOrderInput, currency string) (json.RawMessage, int64, error) {
	switch input.ProductType {
	case PaymentProductBalanceTopup:
		if input.Amount < settings.MinTopupAmount || input.Amount > settings.MaxTopupAmount {
			return nil, 0, ErrPaymentInvalidAmount
		}
		amountMinor, err := PaymentAmountToMinor(input.Amount, currency)
		if err != nil {
			return nil, 0, err
		}
		return marshalPaymentSnapshot(paymentOrderSnapshot{ProductType: input.ProductType, AmountMinor: amountMinor, Currency: currency, CountryCode: input.CountryCode, CreatedAt: time.Now().UTC().Format(time.RFC3339)}), amountMinor, nil
	case PaymentProductSubscription:
		plan, ok := findPaymentSubscriptionPlan(settings.SubscriptionPlans, input.PlanID)
		if !ok {
			return nil, 0, ErrPaymentInvalidProduct
		}
		price := plan.PricesByCurrency[currency]
		if price <= 0 {
			return nil, 0, ErrPaymentUnsupportedCurrency.WithMetadata(map[string]string{"currency": currency})
		}
		amountMinor, err := PaymentAmountToMinor(price, currency)
		if err != nil {
			return nil, 0, err
		}
		return marshalPaymentSnapshot(paymentOrderSnapshot{ProductType: input.ProductType, AmountMinor: amountMinor, Currency: currency, CountryCode: input.CountryCode, PlanID: plan.PlanID, GroupID: plan.GroupID, ValidityDays: plan.ValidityDays, PricesByCurrency: plan.PricesByCurrency, CreatedAt: time.Now().UTC().Format(time.RFC3339)}), amountMinor, nil
	default:
		return nil, 0, ErrPaymentInvalidProduct
	}
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
