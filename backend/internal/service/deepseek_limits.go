package service

import (
	"encoding/json"
	"sort"
)

const DeepSeekModelConcurrencyLimitsExtraKey = "deepseek_model_concurrency_limits"

var DefaultDeepSeekModelConcurrencyLimits = map[string]int{
	"deepseek-v4-pro":   500,
	"deepseek-v4-flash": 2500,
}

func DeepSeekEffectiveAccountConcurrency(account *Account, model string) int {
	if account == nil {
		return 0
	}
	accountConcurrency := account.Concurrency
	if RoutingPlatformForAccount(account) != PlatformDeepSeek {
		return accountConcurrency
	}
	modelLimit := account.DeepSeekModelConcurrencyLimit(model)
	switch {
	case accountConcurrency > 0 && modelLimit > 0 && modelLimit < accountConcurrency:
		return modelLimit
	case accountConcurrency > 0:
		return accountConcurrency
	case modelLimit > 0:
		return modelLimit
	default:
		return accountConcurrency
	}
}

func (a *Account) DeepSeekModelConcurrencyLimit(model string) int {
	if a == nil || RoutingPlatformForAccount(a) != PlatformDeepSeek {
		return 0
	}
	limits := a.DeepSeekModelConcurrencyLimits()
	if len(limits) == 0 {
		return 0
	}
	canonical := normalizeDeepSeekModelID(model)
	if canonical == "" {
		return 0
	}
	return limits[canonical]
}

func (a *Account) DeepSeekModelConcurrencyLimits() map[string]int {
	if a == nil || len(a.Extra) == 0 {
		return nil
	}
	raw, ok := a.Extra[DeepSeekModelConcurrencyLimitsExtraKey]
	if !ok || raw == nil {
		return nil
	}
	limits := make(map[string]int)
	add := func(model string, rawLimit any) {
		canonical := normalizeDeepSeekModelID(model)
		limit := ParseExtraInt(rawLimit)
		if !isDeepSeekModelConcurrencyLimitSupported(canonical) || limit <= 0 {
			return
		}
		limits[canonical] = limit
	}
	switch value := raw.(type) {
	case map[string]any:
		for model, limit := range value {
			add(model, limit)
		}
	case map[string]int:
		for model, limit := range value {
			add(model, limit)
		}
	case map[string]float64:
		for model, limit := range value {
			add(model, limit)
		}
	case map[string]json.Number:
		for model, limit := range value {
			add(model, limit)
		}
	}
	if len(limits) == 0 {
		return nil
	}
	return limits
}

func NormalizeDeepSeekModelConcurrencyLimits(raw any) map[string]int {
	account := &Account{
		Platform: PlatformDeepSeek,
		Extra: map[string]any{
			DeepSeekModelConcurrencyLimitsExtraKey: raw,
		},
	}
	return account.DeepSeekModelConcurrencyLimits()
}

func DeepSeekModelConcurrencyLimitsForStorage(raw any) map[string]any {
	limits := NormalizeDeepSeekModelConcurrencyLimits(raw)
	if len(limits) == 0 {
		return nil
	}
	keys := make([]string, 0, len(limits))
	for key := range limits {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	stored := make(map[string]any, len(keys))
	for _, key := range keys {
		stored[key] = limits[key]
	}
	return stored
}

func isDeepSeekModelConcurrencyLimitSupported(model string) bool {
	switch normalizeDeepSeekModelID(model) {
	case "deepseek-v4-pro", "deepseek-v4-flash":
		return true
	default:
		return false
	}
}
