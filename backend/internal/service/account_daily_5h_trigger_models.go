package service

import (
	"math"
	"sort"
	"strings"
)

func accountDaily5HModelIdentity(model AvailableTestModel) string {
	return strings.ToLower(firstNonEmptyTrimmed(model.TargetModelID, model.CanonicalID, model.ID))
}

func accountDaily5HTextModels(models []AvailableTestModel) []AvailableTestModel {
	out := make([]AvailableTestModel, 0, len(models))
	for _, model := range models {
		id := accountDaily5HModelIdentity(model)
		mode := inferModelMode(id, model.Mode)
		if mode != "chat" && mode != "text" {
			continue
		}
		// Some legacy entries lack mode metadata. These require non-text endpoints/input.
		specialized := false
		for _, marker := range []string{"image", "imagine", "embedding", "realtime", "transcribe", "tts", "audio", "live", "moderation", "video"} {
			if strings.Contains(id, marker) {
				specialized = true
				break
			}
		}
		if specialized || model.AvailabilityState == "unavailable" || model.AvailabilityState == "unsupported" {
			continue
		}
		out = append(out, model)
	}
	return out
}

func daily5HLightweight(typeKey string, model AvailableTestModel) bool {
	id := accountDaily5HModelIdentity(model)
	switch typeKey {
	case AccountDaily5HTypeOpenAI:
		return strings.Contains(id, "mini") || strings.Contains(id, "nano") || strings.Contains(id, "luna")
	case AccountDaily5HTypeAnthropic:
		return strings.Contains(id, "haiku")
	case AccountDaily5HTypeGemini:
		return strings.Contains(id, "flash")
	}
	return false
}

// Exact local prices only: family fallbacks would make an unknown model appear cheap.
func (s *AccountDaily5HTriggerService) daily5HCost(model AvailableTestModel) float64 {
	if s.pricingService == nil {
		return math.Inf(1)
	}
	s.pricingService.mu.RLock()
	defer s.pricingService.mu.RUnlock()
	for _, identity := range []string{model.TargetModelID, model.CanonicalID, model.ID} {
		for _, key := range s.pricingService.buildModelLookupCandidates(strings.ToLower(identity)) {
			price := s.pricingService.pricingData[key]
			if price == nil || price.InputCostPerToken < 0 || price.OutputCostPerToken < 0 {
				continue
			}
			cost := price.InputCostPerToken + price.OutputCostPerToken
			if cost <= 0 {
				continue
			}
			if normalizeBillingCurrency(price.Currency) == "CNY" {
				if price.USDToCNYRate <= 0 {
					continue
				}
				cost /= price.USDToCNYRate
			}
			return cost
		}
	}
	return math.Inf(1)
}

func (s *AccountDaily5HTriggerService) pickDaily5HModel(typeKey string, models []AvailableTestModel) string {
	if len(models) == 0 {
		return ""
	}
	models = append([]AvailableTestModel(nil), models...)
	costs := make(map[string]float64, len(models))
	for _, model := range models {
		costs[model.ID] = s.daily5HCost(model)
	}
	sort.SliceStable(models, func(i, j int) bool {
		a, b := models[i], models[j]
		if daily5HLightweight(typeKey, a) != daily5HLightweight(typeKey, b) {
			return daily5HLightweight(typeKey, a)
		}
		if costs[a.ID] != costs[b.ID] {
			return costs[a.ID] < costs[b.ID]
		}
		stable := func(m AvailableTestModel) bool {
			id := accountDaily5HModelIdentity(m)
			return (m.Status == "" || m.Status == "stable") && !strings.Contains(id, "preview") && !strings.Contains(id, "experimental")
		}
		if stable(a) != stable(b) {
			return stable(a)
		}
		return compareAvailableTestModels(a, b) < 0
	})
	return models[0].ID
}
