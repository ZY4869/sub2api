package service

import (
	_ "embed"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

//go:embed model_catalog_seed.json
var modelCatalogSeedJSON []byte

var bundledModelCatalogSeedEntries []ModelCatalogEntry

func init() {
	if err := json.Unmarshal(modelCatalogSeedJSON, &bundledModelCatalogSeedEntries); err != nil {
		panic(err)
	}
}

func appendBundledModelCatalogSeedDetails(base []modelregistry.AdminModelDetail) []modelregistry.AdminModelDetail {
	if len(bundledModelCatalogSeedEntries) == 0 {
		return base
	}

	seen := make(map[string]struct{}, len(base))
	availability := make(map[string]bool, len(base)*4)
	for _, detail := range base {
		for _, candidate := range bundledSeedAvailabilityCandidates(detail.ModelEntry) {
			if detail.Available {
				availability[candidate] = true
				continue
			}
			if _, exists := availability[candidate]; !exists {
				availability[candidate] = false
			}
		}
		if id := normalizeRegistryID(detail.ModelEntry.ID); id != "" {
			seen[id] = struct{}{}
		}
	}

	extras := make([]modelregistry.AdminModelDetail, 0, len(bundledModelCatalogSeedEntries))
	for _, raw := range bundledModelCatalogSeedEntries {
		entry := normalizeModelCatalogEntry(raw)
		modelID := normalizeRegistryID(entry.Model)
		if modelID == "" {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}

		registryEntry, err := normalizePersistedEntry(modelregistry.ModelEntry{
			ID:               entry.Model,
			DisplayName:      entry.DisplayName,
			Provider:         entry.Provider,
			Platforms:        defaultPlatformsForProvider(entry.Provider),
			ProtocolIDs:      compactRegistryStrings(entry.CanonicalModelID, entry.Model),
			Aliases:          []string{},
			PricingLookupIDs: compactRegistryStrings(entry.PricingLookupModelID, entry.CanonicalModelID, entry.Model),
			Modalities:       defaultModalitiesForMode(inferModelMode(entry.Model, entry.Mode)),
			Capabilities:     defaultCapabilitiesForMode(inferModelMode(entry.Model, entry.Mode)),
			UIPriority:       5000,
			ExposedIn:        []string{"runtime", "legacy_catalog"},
		})
		if err != nil {
			continue
		}

		available := false
		for _, candidate := range bundledSeedAvailabilityCandidates(registryEntry) {
			if availability[candidate] {
				available = true
				break
			}
		}

		extras = append(extras, modelregistry.AdminModelDetail{
			ModelEntry: registryEntry,
			Available:  available,
		})
		seen[modelID] = struct{}{}
		for _, candidate := range bundledSeedAvailabilityCandidates(registryEntry) {
			if available {
				availability[candidate] = true
				continue
			}
			if _, exists := availability[candidate]; !exists {
				availability[candidate] = false
			}
		}
	}

	return append(base, extras...)
}

func bundledSeedAvailabilityCandidates(entry modelregistry.ModelEntry) []string {
	seen := map[string]struct{}{}
	items := make([]string, 0, 8)
	appendCandidate := func(values ...string) {
		for _, value := range values {
			normalized := normalizeRegistryID(value)
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

	appendCandidate(entry.ID)
	appendCandidate(entry.ProtocolIDs...)
	appendCandidate(entry.PricingLookupIDs...)
	appendCandidate(entry.Aliases...)
	return items
}
