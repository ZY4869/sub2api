package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func (s *ModelRegistryService) mergedEntries(ctx context.Context) (map[string]modelregistry.ModelEntry, map[string]string, map[string]struct{}, map[string]struct{}, error) {
	if err := s.ensureLegacyCatalogMigrated(ctx); err != nil {
		return nil, nil, nil, nil, err
	}
	entries := make(map[string]modelregistry.ModelEntry)
	sources := make(map[string]string)
	for _, entry := range modelregistry.SeedModels() {
		entries[entry.ID] = entry
		sources[entry.ID] = "seed"
	}
	runtimeEntries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for _, entry := range runtimeEntries {
		entries[entry.ID] = entry
		sources[entry.ID] = "runtime"
	}
	hidden, err := s.loadStringSet(ctx, SettingKeyModelRegistryHiddenModels)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	tombstones, err := s.loadStringSet(ctx, SettingKeyModelRegistryTombstones)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return entries, sources, hidden, tombstones, nil
}

func (s *ModelRegistryService) loadRuntimeEntries(ctx context.Context) ([]modelregistry.ModelEntry, error) {
	if s.settingRepo == nil {
		return []modelregistry.ModelEntry{}, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelRegistryEntries)
	if err != nil || strings.TrimSpace(raw) == "" {
		return []modelregistry.ModelEntry{}, nil
	}
	var entries []modelregistry.ModelEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, infraerrors.InternalServer("MODEL_REGISTRY_INVALID_JSON", "invalid model registry entries json").WithCause(err)
	}
	normalized := make([]modelregistry.ModelEntry, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		normalizedEntry, err := normalizePersistedEntry(entry)
		if err != nil {
			continue
		}
		if _, exists := seen[normalizedEntry.ID]; exists {
			continue
		}
		seen[normalizedEntry.ID] = struct{}{}
		normalized = append(normalized, normalizedEntry)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].ID < normalized[j].ID })
	return normalized, nil
}

func (s *ModelRegistryService) persistRuntimeEntries(ctx context.Context, entries []modelregistry.ModelEntry) error {
	if s.settingRepo == nil {
		return nil
	}
	normalized := make([]modelregistry.ModelEntry, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		normalizedEntry, err := normalizePersistedEntry(entry)
		if err != nil {
			continue
		}
		if _, exists := seen[normalizedEntry.ID]; exists {
			continue
		}
		seen[normalizedEntry.ID] = struct{}{}
		normalized = append(normalized, normalizedEntry)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].ID < normalized[j].ID })
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, SettingKeyModelRegistryEntries, string(payload))
}

func (s *ModelRegistryService) clearStates(ctx context.Context, modelID string) error {
	if modelID == "" {
		return nil
	}
	hidden, err := s.loadStringSet(ctx, SettingKeyModelRegistryHiddenModels)
	if err != nil {
		return err
	}
	delete(hidden, modelID)
	if err := s.persistStringSet(ctx, SettingKeyModelRegistryHiddenModels, hidden); err != nil {
		return err
	}
	tombstones, err := s.loadStringSet(ctx, SettingKeyModelRegistryTombstones)
	if err != nil {
		return err
	}
	delete(tombstones, modelID)
	return s.persistStringSet(ctx, SettingKeyModelRegistryTombstones, tombstones)
}

func (s *ModelRegistryService) ensureLegacyCatalogMigrated(ctx context.Context) error {
	if s.settingRepo == nil {
		return nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelCatalogEntries)
	if err != nil {
		return nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	legacyEntries, err := s.loadLegacyCatalogEntries(ctx)
	if err != nil {
		return err
	}
	if len(legacyEntries) == 0 {
		return s.settingRepo.Set(ctx, SettingKeyModelCatalogEntries, "[]")
	}
	log := logger.FromContext(ctx)
	log.Info("model registry: migrating legacy model catalog entries", zap.Int("entry_count", len(legacyEntries)))
	runtimeEntries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return err
	}
	merged := make(map[string]modelregistry.ModelEntry, len(runtimeEntries)+len(legacyEntries))
	for _, entry := range runtimeEntries {
		merged[entry.ID] = entry
	}
	for _, entry := range legacyEntries {
		if _, exists := merged[entry.ID]; exists {
			continue
		}
		merged[entry.ID] = entry
	}
	items := make([]modelregistry.ModelEntry, 0, len(merged))
	for _, entry := range merged {
		items = append(items, entry)
	}
	if err := s.persistRuntimeEntries(ctx, items); err != nil {
		log.Warn("model registry: failed to persist migrated legacy model catalog entries", zap.Error(err))
		return err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyModelCatalogEntries, "[]"); err != nil {
		log.Warn("model registry: failed to clear legacy model catalog entries after migration", zap.Error(err))
		return err
	}
	log.Info("model registry: migrated legacy model catalog entries", zap.Int("entry_count", len(legacyEntries)))
	return nil
}

func (s *ModelRegistryService) loadLegacyCatalogEntries(ctx context.Context) ([]modelregistry.ModelEntry, error) {
	if s.settingRepo == nil {
		return []modelregistry.ModelEntry{}, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelCatalogEntries)
	if err != nil || strings.TrimSpace(raw) == "" {
		return []modelregistry.ModelEntry{}, nil
	}
	var entries []ModelCatalogEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, infraerrors.InternalServer("MODEL_CATALOG_INVALID_JSON", "invalid legacy model catalog entries json").WithCause(err)
	}
	result := make([]modelregistry.ModelEntry, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		normalized := normalizeModelCatalogEntry(entry)
		if normalized.Model == "" {
			continue
		}
		registryEntry, err := normalizePersistedEntry(modelregistry.ModelEntry{
			ID:               normalized.Model,
			DisplayName:      normalized.DisplayName,
			Provider:         normalized.Provider,
			Platforms:        defaultPlatformsForProvider(normalized.Provider),
			ProtocolIDs:      compactRegistryStrings(normalized.CanonicalModelID, normalized.Model),
			Aliases:          []string{},
			PricingLookupIDs: compactRegistryStrings(normalized.PricingLookupModelID, normalized.CanonicalModelID, normalized.Model),
			Modalities:       defaultModalitiesForMode(inferModelMode(normalized.Model, normalized.Mode)),
			Capabilities:     defaultCapabilitiesForMode(inferModelMode(normalized.Model, normalized.Mode)),
			UIPriority:       5000,
			ExposedIn:        []string{"runtime", "legacy_catalog"},
		})
		if err != nil {
			continue
		}
		if _, exists := seen[registryEntry.ID]; exists {
			continue
		}
		seen[registryEntry.ID] = struct{}{}
		result = append(result, registryEntry)
	}
	return result, nil
}

func (s *ModelRegistryService) ensureDiscoveredRuntimeAvailability(ctx context.Context, entry modelregistry.ModelEntry, sourceModelID string, canonicalModelID string, sourcePlatform string) (bool, error) {
	updated, changed, err := augmentDiscoveredEntry(entry, sourceModelID, canonicalModelID, sourcePlatform)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	runtimeEntries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return false, err
	}
	replaced := false
	for index := range runtimeEntries {
		if runtimeEntries[index].ID == updated.ID {
			runtimeEntries[index] = updated
			replaced = true
			break
		}
	}
	if !replaced {
		runtimeEntries = append(runtimeEntries, updated)
	}
	if err := s.persistRuntimeEntries(ctx, runtimeEntries); err != nil {
		return false, err
	}
	if err := s.clearStates(ctx, updated.ID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *ModelRegistryService) buildDiscoveredRuntimeEntry(sourceModelID string, canonicalModelID string, sourcePlatform string) (modelregistry.ModelEntry, error) {
	provider := inferModelProvider(sourceModelID)
	platforms, err := discoveredPlatforms(provider, sourcePlatform)
	if err != nil {
		return modelregistry.ModelEntry{}, err
	}
	pricingLookupIDs := compactRegistryStrings(sourceModelID)
	if pricingID, ok := modelregistry.ResolveToPricingID(canonicalModelID); ok {
		pricingLookupIDs = compactRegistryStrings(pricingID, sourceModelID)
	}
	entry, err := normalizePersistedEntry(modelregistry.ModelEntry{
		ID:                   sourceModelID,
		DisplayName:          FormatModelCatalogDisplayName(sourceModelID),
		Provider:             providerOrPlatform(provider, sourcePlatform),
		Platforms:            platforms,
		ProtocolIDs:          compactRegistryStrings(sourceModelID),
		Aliases:              []string{},
		PricingLookupIDs:     pricingLookupIDs,
		PreferredProtocolIDs: map[string]string{"default": sourceModelID},
		Modalities:           defaultModalitiesForMode(inferModelMode(sourceModelID, "")),
		Capabilities:         defaultCapabilitiesForMode(inferModelMode(sourceModelID, "")),
		UIPriority:           5000,
		ExposedIn:            []string{"runtime", "test"},
	})
	if err != nil {
		return modelregistry.ModelEntry{}, err
	}
	return entry, nil
}

func (s *ModelRegistryService) loadStringSet(ctx context.Context, key string) (map[string]struct{}, error) {
	set := map[string]struct{}{}
	if s.settingRepo == nil {
		return set, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, key)
	if err != nil || strings.TrimSpace(raw) == "" {
		return set, nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, infraerrors.InternalServer("MODEL_REGISTRY_INVALID_STATE", "invalid model registry state json").WithCause(err)
	}
	for _, item := range items {
		item = normalizeRegistryID(item)
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	return set, nil
}

func (s *ModelRegistryService) persistStringSet(ctx context.Context, key string, set map[string]struct{}) error {
	if s.settingRepo == nil {
		return nil
	}
	items := make([]string, 0, len(set))
	for item := range set {
		if item = normalizeRegistryID(item); item != "" {
			items = append(items, item)
		}
	}
	sort.Strings(items)
	payload, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, key, string(payload))
}
