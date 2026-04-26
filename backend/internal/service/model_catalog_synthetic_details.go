package service

import (
	"context"
	"sort"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

type syntheticCatalogSource struct {
	model    string
	provider string
	mode     string
}

func (s *ModelCatalogService) appendPricingBackedSyntheticCatalogDetails(ctx context.Context, base []modelregistry.AdminModelDetail) []modelregistry.AdminModelDetail {
	sources := s.collectSyntheticCatalogSources(ctx)
	if len(sources) == 0 {
		return base
	}

	seen := make(map[string]struct{}, len(base)*6)
	for _, detail := range base {
		for _, candidate := range syntheticCatalogSeenCandidates(detail.ModelEntry, detail.ID) {
			seen[candidate] = struct{}{}
		}
	}

	extras := make([]modelregistry.AdminModelDetail, 0, len(sources))
	for _, source := range sources {
		skip := false
		for _, candidate := range syntheticCatalogSeenCandidates(modelregistry.ModelEntry{}, source.model) {
			if _, exists := seen[candidate]; exists {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		modelID := normalizeModelCatalogAlias(source.model)
		if modelID == "" {
			modelID = CanonicalizeModelNameForPricing(source.model)
		}
		if modelID == "" {
			continue
		}

		provider := providerOrPlatform(inferModelProvider(modelID), source.provider)
		if provider == "" {
			provider = "unknown"
		}

		entry, err := normalizePersistedEntry(modelregistry.ModelEntry{
			ID:               modelID,
			DisplayName:      FormatModelCatalogDisplayName(modelID),
			Provider:         provider,
			Platforms:        defaultPlatformsForProvider(provider),
			ProtocolIDs:      compactRegistryStrings(modelID),
			Aliases:          compactRegistryStrings(source.model),
			PricingLookupIDs: compactRegistryStrings(source.model, modelID),
			Modalities:       defaultModalitiesForMode(inferModelMode(modelID, source.mode)),
			Capabilities:     defaultCapabilitiesForMode(inferModelMode(modelID, source.mode)),
			UIPriority:       6000,
			ExposedIn:        []string{"billing"},
		})
		if err != nil {
			continue
		}

		extras = append(extras, modelregistry.AdminModelDetail{
			ModelEntry: entry,
			Source:     "pricing_synthetic",
			Available:  false,
		})
		for _, candidate := range syntheticCatalogSeenCandidates(entry, source.model) {
			seen[candidate] = struct{}{}
		}
	}

	if len(extras) == 0 {
		return base
	}
	return append(base, extras...)
}

func (s *ModelCatalogService) collectSyntheticCatalogSources(ctx context.Context) []syntheticCatalogSource {
	seen := make(map[string]syntheticCatalogSource)
	appendSource := func(model string, provider string, mode string) {
		model = CanonicalizeModelNameForPricing(model)
		if model == "" {
			return
		}
		existing, ok := seen[model]
		if !ok {
			seen[model] = syntheticCatalogSource{model: model, provider: provider, mode: mode}
			return
		}
		if existing.provider == "" {
			existing.provider = provider
		}
		if existing.mode == "" {
			existing.mode = mode
		}
		seen[model] = existing
	}

	if s.pricingService != nil {
		for model, pricing := range s.pricingService.GetPricingSnapshot() {
			provider := ""
			mode := ""
			if pricing != nil {
				provider = pricing.LiteLLMProvider
				mode = pricing.Mode
			}
			appendSource(model, provider, mode)
		}
	}
	if s.billingService != nil {
		for _, model := range s.billingService.ListSupportedModels() {
			appendSource(model, "", "")
		}
	}

	if s.settingRepo != nil {
		for model := range s.loadOfficialPriceOverrides(ctx) {
			appendSource(model, "", "")
		}
		for model := range s.loadSalePriceOverrides(ctx) {
			appendSource(model, "", "")
		}
		for model := range s.loadModelPricingCurrencies(ctx) {
			appendSource(model, "", "")
		}
	}

	items := make([]syntheticCatalogSource, 0, len(seen))
	for _, source := range seen {
		items = append(items, source)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].model < items[j].model })
	return items
}

func syntheticCatalogSeenCandidates(entry modelregistry.ModelEntry, rawModel string) []string {
	seen := map[string]struct{}{}
	items := make([]string, 0, 12)
	appendCandidate := func(values ...string) {
		for _, value := range values {
			normalized := CanonicalizeModelNameForPricing(value)
			if normalized == "" {
				continue
			}
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			items = append(items, normalized)
		}
	}

	appendCandidate(rawModel)
	appendCandidate(normalizeModelCatalogAlias(rawModel))
	appendCandidate(entry.ID)
	appendCandidate(normalizeModelCatalogAlias(entry.ID))
	appendCandidate(entry.ProtocolIDs...)
	appendCandidate(entry.PricingLookupIDs...)
	appendCandidate(entry.Aliases...)
	return items
}
