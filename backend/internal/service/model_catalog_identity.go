package service

import "strings"

func resolveModelCatalogRecord(records map[string]*modelCatalogRecord, model string) (*modelCatalogRecord, bool) {
	canonical := CanonicalizeModelNameForPricing(model)
	if canonical == "" {
		return nil, false
	}
	var selected *modelCatalogRecord
	if record, ok := records[canonical]; ok {
		selected = record
	}
	normalized := NormalizeModelCatalogModelID(canonical)
	if normalized != "" {
		if record, ok := records[normalized]; ok {
			selected = preferModelCatalogRecord(selected, record)
		}
	}
	for _, record := range records {
		if NormalizeModelCatalogModelID(record.model) != normalized {
			continue
		}
		selected = preferModelCatalogRecord(selected, record)
	}
	return selected, selected != nil
}

func preferModelCatalogRecord(current *modelCatalogRecord, candidate *modelCatalogRecord) *modelCatalogRecord {
	if current == nil {
		return candidate
	}
	if candidate == nil {
		return current
	}
	if candidate.officialOverridePricing != nil || candidate.saleOverridePricing != nil {
		if current.officialOverridePricing == nil && current.saleOverridePricing == nil {
			return candidate
		}
	}
	if current.defaultAvailable != candidate.defaultAvailable {
		if candidate.defaultAvailable {
			return candidate
		}
		return current
	}
	currentScore := modelCatalogRecordPricingScore(current)
	candidateScore := modelCatalogRecordPricingScore(candidate)
	if currentScore != candidateScore {
		if candidateScore > currentScore {
			return candidate
		}
		return current
	}
	currentHasDate := modelCatalogDateVersionSuffixPattern.MatchString(current.model)
	candidateHasDate := modelCatalogDateVersionSuffixPattern.MatchString(candidate.model)
	if currentHasDate != candidateHasDate {
		if !candidateHasDate {
			return candidate
		}
		return current
	}
	currentDate := modelCatalogDateVersionSuffixPattern.FindString(strings.ToLower(current.model))
	candidateDate := modelCatalogDateVersionSuffixPattern.FindString(strings.ToLower(candidate.model))
	if currentDate != candidateDate {
		if candidateDate > currentDate {
			return candidate
		}
		return current
	}
	if len(candidate.model) != len(current.model) {
		if len(candidate.model) < len(current.model) {
			return candidate
		}
		return current
	}
	if candidate.model < current.model {
		return candidate
	}
	return current
}

func modelCatalogRecordPricingScore(record *modelCatalogRecord) int {
	if record == nil {
		return 0
	}
	return countModelCatalogPricingFields(record.officialPricing) + countModelCatalogPricingFields(record.salePricing)
}
