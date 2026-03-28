package service

import "strings"

func dedupeModelCatalogItems(items []ModelCatalogItem) []ModelCatalogItem {
	selected := make(map[string]ModelCatalogItem, len(items))
	order := make([]string, 0, len(items))
	for _, item := range items {
		key := NormalizeModelCatalogModelID(item.Model)
		if key == "" {
			key = strings.TrimSpace(item.DisplayName)
		}
		if key == "" {
			key = item.Model
		}
		current, ok := selected[key]
		if !ok {
			selected[key] = item
			order = append(order, key)
			continue
		}
		selected[key] = preferModelCatalogItem(current, item)
	}
	result := make([]ModelCatalogItem, 0, len(order))
	for _, key := range order {
		result = append(result, selected[key])
	}
	return result
}

func preferModelCatalogItem(current ModelCatalogItem, candidate ModelCatalogItem) ModelCatalogItem {
	currentCanonicalDisplay := CanonicalizeModelNameForPricing(current.DisplayName)
	candidateCanonicalDisplay := CanonicalizeModelNameForPricing(candidate.DisplayName)
	currentCanonicalModel := CanonicalizeModelNameForPricing(NormalizeModelCatalogModelID(current.Model))
	candidateCanonicalModel := CanonicalizeModelNameForPricing(NormalizeModelCatalogModelID(candidate.Model))
	currentMatchesCanonical := currentCanonicalDisplay != "" && currentCanonicalDisplay == currentCanonicalModel
	candidateMatchesCanonical := candidateCanonicalDisplay != "" && candidateCanonicalDisplay == candidateCanonicalModel
	if candidateMatchesCanonical != currentMatchesCanonical {
		if candidateMatchesCanonical {
			return candidate
		}
		return current
	}
	if candidate.HasOverride != current.HasOverride {
		if candidate.HasOverride {
			return candidate
		}
		return current
	}
	if candidate.DefaultAvailable != current.DefaultAvailable {
		if candidate.DefaultAvailable {
			return candidate
		}
		return current
	}
	currentScore := modelCatalogPricingScore(current)
	candidateScore := modelCatalogPricingScore(candidate)
	if candidateScore != currentScore {
		if candidateScore > currentScore {
			return candidate
		}
		return current
	}
	currentHasDate := modelCatalogDateVersionSuffixPattern.MatchString(current.Model)
	candidateHasDate := modelCatalogDateVersionSuffixPattern.MatchString(candidate.Model)
	if currentHasDate != candidateHasDate {
		if !candidateHasDate {
			return candidate
		}
		return current
	}
	currentDate := modelCatalogDateVersionSuffixPattern.FindString(strings.ToLower(current.Model))
	candidateDate := modelCatalogDateVersionSuffixPattern.FindString(strings.ToLower(candidate.Model))
	if candidateDate != currentDate {
		if candidateDate > currentDate {
			return candidate
		}
		return current
	}
	if len(candidate.Model) != len(current.Model) {
		if len(candidate.Model) < len(current.Model) {
			return candidate
		}
		return current
	}
	if candidate.Model < current.Model {
		return candidate
	}
	return current
}

func modelCatalogPricingScore(item ModelCatalogItem) int {
	return countModelCatalogPricingFields(item.OfficialPricing) + countModelCatalogPricingFields(item.SalePricing)
}

func countModelCatalogPricingFields(pricing *ModelCatalogPricing) int {
	if pricing == nil {
		return 0
	}
	count := 0
	values := []bool{
		pricing.InputCostPerToken != nil,
		pricing.InputCostPerTokenPriority != nil,
		pricing.InputTokenThreshold != nil,
		pricing.InputCostPerTokenAboveThreshold != nil,
		pricing.InputCostPerTokenPriorityAboveThreshold != nil,
		pricing.OutputCostPerToken != nil,
		pricing.OutputCostPerTokenPriority != nil,
		pricing.OutputTokenThreshold != nil,
		pricing.OutputCostPerTokenAboveThreshold != nil,
		pricing.OutputCostPerTokenPriorityAboveThreshold != nil,
		pricing.CacheCreationInputTokenCost != nil,
		pricing.CacheCreationInputTokenCostAbove1hr != nil,
		pricing.CacheReadInputTokenCost != nil,
		pricing.CacheReadInputTokenCostPriority != nil,
		pricing.OutputCostPerImage != nil,
		pricing.OutputCostPerVideoRequest != nil,
	}
	for _, present := range values {
		if present {
			count++
		}
	}
	return count
}
