package service

import (
	"math"
	"strings"
	"time"
)

type ModelPricingCurrencyMetadata struct {
	Currency     string
	USDToCNYRate *float64
	FXRateDate   string
	FXLockedAt   *time.Time
}

func normalizeBillingCurrency(currency string) string {
	return defaultModelPricingCurrency(currency)
}

func cloneBillingTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func cloneBillingStringMapFloat64(values map[string]float64) map[string]float64 {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]float64, len(values))
	for key, value := range values {
		currency := normalizeBillingCurrency(key)
		if currency == "" || !validBillingAmount(value) {
			continue
		}
		out[currency] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func CloneBillingCurrencyMap(values map[string]float64) map[string]float64 {
	return cloneBillingStringMapFloat64(values)
}

func validBillingAmount(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func normalizedBillingCostMap(currency string, amount float64) map[string]float64 {
	currency = normalizeBillingCurrency(currency)
	if currency == "" || !validBillingAmount(amount) || amount == 0 {
		return nil
	}
	return map[string]float64{currency: amount}
}

func pricingCurrencyMetadataFromCatalog(pricing *ModelCatalogPricing) ModelPricingCurrencyMetadata {
	if pricing == nil {
		return ModelPricingCurrencyMetadata{Currency: ModelPricingCurrencyUSD}
	}
	return ModelPricingCurrencyMetadata{
		Currency:     normalizeBillingCurrency(pricing.Currency),
		USDToCNYRate: cloneBillingFloat64(pricing.USDToCNYRate),
		FXRateDate:   strings.TrimSpace(pricing.FXRateDate),
		FXLockedAt:   cloneBillingTime(pricing.FXLockedAt),
	}
}

func applyCurrencyMetadataToCatalogPricing(pricing *ModelCatalogPricing, meta ModelPricingCurrencyMetadata) {
	if pricing == nil {
		return
	}
	currency := normalizeBillingCurrency(meta.Currency)
	if currency == "" {
		currency = ModelPricingCurrencyUSD
	}
	pricing.Currency = currency
	pricing.USDToCNYRate = cloneBillingFloat64(meta.USDToCNYRate)
	pricing.FXRateDate = strings.TrimSpace(meta.FXRateDate)
	pricing.FXLockedAt = cloneBillingTime(meta.FXLockedAt)
}

func modelPricingMetadataFromPreference(pref *BillingPricingCurrencyPreference) ModelPricingCurrencyMetadata {
	if pref == nil {
		return ModelPricingCurrencyMetadata{Currency: ModelPricingCurrencyUSD}
	}
	return ModelPricingCurrencyMetadata{
		Currency:     normalizeBillingCurrency(pref.Currency),
		USDToCNYRate: cloneBillingFloat64(pref.USDToCNYRate),
		FXRateDate:   strings.TrimSpace(pref.FXRateDate),
		FXLockedAt:   cloneBillingTime(pref.FXLockedAt),
	}
}

func costUSDEquivalent(amount float64, currency string, usdToCNYRate float64) float64 {
	switch normalizeBillingCurrency(currency) {
	case ModelPricingCurrencyCNY:
		if usdToCNYRate > 0 {
			return amount / usdToCNYRate
		}
		return 0
	default:
		return amount
	}
}

func NormalizeUsageBillingCurrency(currency string) string {
	return normalizeBillingCurrency(currency)
}

func CostUSDEquivalentForPersistence(amount float64, currency string, usdToCNYRate float64) float64 {
	return costUSDEquivalent(amount, currency, usdToCNYRate)
}
