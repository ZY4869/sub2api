package service

import "github.com/Wei-Shaw/sub2api/internal/modelregistry"

func resolveModelCatalogRecord(records map[string]*modelCatalogRecord, model string) (*modelCatalogRecord, bool) {
	alias := normalizeModelCatalogAlias(model)
	if alias == "" {
		return nil, false
	}
	if record, ok := records[alias]; ok {
		return record, true
	}
	if pricingKey, ok := modelregistry.ResolveToPricingID(alias); ok {
		if record, exists := records[pricingKey]; exists {
			return record, true
		}
	}
	if normalized := NormalizeModelCatalogModelID(alias); normalized != "" {
		if record, ok := records[normalized]; ok {
			return record, true
		}
	}
	return nil, false
}
