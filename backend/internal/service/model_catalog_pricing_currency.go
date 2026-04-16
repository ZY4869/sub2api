package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	ModelPricingCurrencyUSD = "USD"
	ModelPricingCurrencyCNY = "CNY"
)

func normalizeModelPricingCurrency(currency string) string {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case ModelPricingCurrencyUSD:
		return ModelPricingCurrencyUSD
	case ModelPricingCurrencyCNY:
		return ModelPricingCurrencyCNY
	default:
		return ""
	}
}

func defaultModelPricingCurrency(currency string) string {
	normalized := normalizeModelPricingCurrency(currency)
	if normalized == "" {
		return ModelPricingCurrencyUSD
	}
	return normalized
}

func cloneBillingPricingCurrencyPreference(pref *BillingPricingCurrencyPreference) *BillingPricingCurrencyPreference {
	if pref == nil {
		return nil
	}
	copy := *pref
	copy.Currency = defaultModelPricingCurrency(pref.Currency)
	return &copy
}

func loadModelPricingCurrenciesBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
) map[string]*BillingPricingCurrencyPreference {
	if settingRepo == nil {
		return map[string]*BillingPricingCurrencyPreference{}
	}
	raw, err := settingRepo.GetValue(ctx, settingKey)
	if err != nil || raw == "" {
		return map[string]*BillingPricingCurrencyPreference{}
	}
	var prefs map[string]*BillingPricingCurrencyPreference
	if err := json.Unmarshal([]byte(raw), &prefs); err != nil {
		logger.FromContext(ctx).Warn(
			"model catalog: invalid pricing currency json",
			zap.String("setting_key", settingKey),
			zap.Error(err),
		)
		return map[string]*BillingPricingCurrencyPreference{}
	}
	normalized := make(map[string]*BillingPricingCurrencyPreference, len(prefs))
	for model, pref := range prefs {
		key := NormalizeModelCatalogModelID(model)
		if key == "" || pref == nil {
			continue
		}
		currency := normalizeModelPricingCurrency(pref.Currency)
		if currency == "" || currency == ModelPricingCurrencyUSD {
			continue
		}
		copy := cloneBillingPricingCurrencyPreference(pref)
		copy.Currency = currency
		normalized[key] = copy
	}
	return normalized
}

func persistModelPricingCurrenciesBySetting(
	ctx context.Context,
	settingRepo SettingRepository,
	settingKey string,
	prefs map[string]*BillingPricingCurrencyPreference,
) error {
	if settingRepo == nil {
		return nil
	}
	if len(prefs) == 0 {
		return settingRepo.Delete(ctx, settingKey)
	}
	keys := make([]string, 0, len(prefs))
	for key := range prefs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make(map[string]*BillingPricingCurrencyPreference, len(keys))
	for _, key := range keys {
		ordered[key] = cloneBillingPricingCurrencyPreference(prefs[key])
	}
	payload, err := json.Marshal(ordered)
	if err != nil {
		return err
	}
	return settingRepo.Set(ctx, settingKey, string(payload))
}

func (s *ModelCatalogService) loadModelPricingCurrencies(
	ctx context.Context,
) map[string]*BillingPricingCurrencyPreference {
	return loadModelPricingCurrenciesBySetting(
		ctx,
		s.settingRepo,
		SettingKeyModelPricingCurrencies,
	)
}

func (s *ModelCatalogService) persistModelPricingCurrencies(
	ctx context.Context,
	prefs map[string]*BillingPricingCurrencyPreference,
) error {
	return persistModelPricingCurrenciesBySetting(
		ctx,
		s.settingRepo,
		SettingKeyModelPricingCurrencies,
		prefs,
	)
}

func (s *ModelCatalogService) saveModelPricingCurrency(
	ctx context.Context,
	actor ModelCatalogActor,
	model string,
	currency string,
) error {
	alias := NormalizeModelCatalogModelID(model)
	if alias == "" {
		return nil
	}
	normalized := defaultModelPricingCurrency(currency)
	prefs := s.loadModelPricingCurrencies(ctx)
	if normalized == ModelPricingCurrencyUSD {
		if _, ok := prefs[alias]; !ok {
			return nil
		}
		delete(prefs, alias)
		return s.persistModelPricingCurrencies(ctx, prefs)
	}
	prefs[alias] = &BillingPricingCurrencyPreference{
		Currency:        normalized,
		UpdatedAt:       time.Now().UTC(),
		UpdatedByUserID: actor.UserID,
		UpdatedByEmail:  strings.TrimSpace(actor.Email),
	}
	return s.persistModelPricingCurrencies(ctx, prefs)
}
