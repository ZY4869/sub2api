package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

type ModelRegistryListFilter struct {
	Search            string
	Provider          string
	Platform          string
	IncludeHidden     bool
	IncludeTombstoned bool
	Page              int
	PageSize          int
}

type UpsertModelRegistryEntryInput struct {
	ID               string   `json:"id"`
	DisplayName      string   `json:"display_name"`
	Provider         string   `json:"provider"`
	Platforms        []string `json:"platforms"`
	ProtocolIDs      []string `json:"protocol_ids"`
	Aliases          []string `json:"aliases"`
	PricingLookupIDs []string `json:"pricing_lookup_ids"`
	Modalities       []string `json:"modalities"`
	Capabilities     []string `json:"capabilities"`
	UIPriority       int      `json:"ui_priority"`
	ExposedIn        []string `json:"exposed_in"`
}

type UpdateModelRegistryVisibilityInput struct {
	Model  string `json:"model"`
	Hidden bool   `json:"hidden"`
}

type BatchSyncModelRegistryExposuresInput struct {
	Models    []string `json:"models"`
	Exposures []string `json:"exposures"`
}

type ModelRegistryExposureSyncFailure struct {
	Model string `json:"model"`
	Error string `json:"error"`
}

type BatchSyncModelRegistryExposuresResult struct {
	Exposures     []string                           `json:"exposures"`
	UpdatedCount  int                                `json:"updated_count"`
	SkippedCount  int                                `json:"skipped_count"`
	FailedCount   int                                `json:"failed_count"`
	UpdatedModels []string                           `json:"updated_models"`
	SkippedModels []string                           `json:"skipped_models,omitempty"`
	FailedModels  []ModelRegistryExposureSyncFailure `json:"failed_models,omitempty"`
}

type UpsertDiscoveredEntryInput struct {
	ModelID        string
	SourcePlatform string
}

type UpsertDiscoveredEntryResult struct {
	RegistryModelID string
	CanonicalModel  string
	Changed         bool
	Existing        bool
	Blocked         bool
}

type ModelRegistryService struct {
	settingRepo SettingRepository
}

var modelRegistryCapabilityOrder = []string{
	"text",
	"vision",
	"image_generation",
	"web_search",
	"audio_understanding",
	"video_understanding",
	"audio_generation",
	"video_generation",
}

var modelRegistryCapabilityAliases = map[string]string{
	"reasoning": "text",
	"image":     "image_generation",
	"video":     "video_generation",
	"audio":     "audio_understanding",
	"web":       "web_search",
}

func NewModelRegistryService(settingRepo SettingRepository) *ModelRegistryService {
	return &ModelRegistryService{settingRepo: settingRepo}
}

func (s *ModelRegistryService) PublicSnapshot(ctx context.Context) (*modelregistry.PublicSnapshot, error) {
	models, presets, err := s.visibleSnapshotData(ctx)
	if err != nil {
		return nil, err
	}
	etag, err := computeRegistryETag(models, presets)
	if err != nil {
		return nil, err
	}
	return &modelregistry.PublicSnapshot{
		ETag:      etag,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Models:    models,
		Presets:   presets,
	}, nil
}

func (s *ModelRegistryService) GetModelsByPlatform(ctx context.Context, platform string, exposures ...string) ([]modelregistry.ModelEntry, error) {
	snapshot, err := s.PublicSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return modelregistry.ModelsByPlatform(snapshot.Models, platform, exposures...), nil
}

func (s *ModelRegistryService) GetModel(ctx context.Context, modelID string) (*modelregistry.ModelEntry, error) {
	snapshot, err := s.PublicSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	entry, ok := modelregistry.FindModel(snapshot.Models, normalizeRegistryID(modelID))
	if !ok {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	return &entry, nil
}

func (s *ModelRegistryService) List(ctx context.Context, filter ModelRegistryListFilter) ([]modelregistry.AdminModelDetail, int64, error) {
	details, err := s.adminDetails(ctx)
	if err != nil {
		return nil, 0, err
	}
	filtered := make([]modelregistry.AdminModelDetail, 0, len(details))
	search := strings.TrimSpace(strings.ToLower(filter.Search))
	provider := strings.TrimSpace(strings.ToLower(filter.Provider))
	platform := normalizeRegistryPlatform(filter.Platform)
	for _, detail := range details {
		if !filter.IncludeHidden && detail.Hidden {
			continue
		}
		if !filter.IncludeTombstoned && detail.Tombstoned {
			continue
		}
		if provider != "" && strings.ToLower(detail.Provider) != provider {
			continue
		}
		if platform != "" {
			matched := false
			for _, current := range detail.Platforms {
				if normalizeRegistryPlatform(current) == platform {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if search != "" {
			haystack := strings.ToLower(detail.ID + " " + detail.DisplayName + " " + detail.Provider)
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		filtered = append(filtered, detail)
	}
	total := int64(len(filtered))
	page, pageSize := normalizeListPagination(filter.Page, filter.PageSize)
	start := (page - 1) * pageSize
	if start >= len(filtered) {
		return []modelregistry.AdminModelDetail{}, total, nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

func (s *ModelRegistryService) GetDetail(ctx context.Context, modelID string) (*modelregistry.AdminModelDetail, error) {
	details, err := s.adminDetails(ctx)
	if err != nil {
		return nil, err
	}
	modelID = normalizeRegistryID(modelID)
	for _, detail := range details {
		if detail.ID == modelID {
			copy := detail
			return &copy, nil
		}
	}
	return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
}

func (s *ModelRegistryService) UpsertEntry(ctx context.Context, input UpsertModelRegistryEntryInput) (*modelregistry.AdminModelDetail, error) {
	entry, err := normalizeRuntimeRegistryEntry(input)
	if err != nil {
		return nil, err
	}
	entries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return nil, err
	}
	replaced := false
	for index := range entries {
		if entries[index].ID == entry.ID {
			entries[index] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		entries = append(entries, entry)
	}
	if err := s.persistRuntimeEntries(ctx, entries); err != nil {
		return nil, err
	}
	if err := s.clearStates(ctx, entry.ID); err != nil {
		return nil, err
	}
	return s.GetDetail(ctx, entry.ID)
}

func (s *ModelRegistryService) UpsertDiscoveredEntry(ctx context.Context, input UpsertDiscoveredEntryInput) (*UpsertDiscoveredEntryResult, error) {
	sourceModelID := normalizeRegistryID(input.ModelID)
	if sourceModelID == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	canonicalModelID := normalizeModelCatalogAlias(sourceModelID)
	if canonicalModelID == "" {
		canonicalModelID = sourceModelID
	}
	entries, _, _, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return nil, err
	}
	if _, blocked := tombstones[sourceModelID]; blocked {
		return &UpsertDiscoveredEntryResult{RegistryModelID: sourceModelID, CanonicalModel: canonicalModelID, Blocked: true}, nil
	}
	if _, blocked := tombstones[canonicalModelID]; blocked {
		return &UpsertDiscoveredEntryResult{RegistryModelID: sourceModelID, CanonicalModel: canonicalModelID, Blocked: true}, nil
	}
	mergedEntries := make([]modelregistry.ModelEntry, 0, len(entries))
	for _, entry := range entries {
		mergedEntries = append(mergedEntries, entry)
	}
	if existing, found := modelregistry.FindModel(mergedEntries, sourceModelID); found {
		changed, err := s.ensureDiscoveredRuntimeAvailability(ctx, existing, sourceModelID, canonicalModelID, input.SourcePlatform)
		if err != nil {
			return nil, err
		}
		return &UpsertDiscoveredEntryResult{
			RegistryModelID: existing.ID,
			CanonicalModel:  canonicalModelID,
			Changed:         changed,
			Existing:        !changed,
		}, nil
	}
	entry, err := s.buildDiscoveredRuntimeEntry(sourceModelID, canonicalModelID, input.SourcePlatform)
	if err != nil {
		return nil, err
	}
	runtimeEntries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return nil, err
	}
	runtimeEntries = append(runtimeEntries, entry)
	if err := s.persistRuntimeEntries(ctx, runtimeEntries); err != nil {
		return nil, err
	}
	if err := s.clearStates(ctx, entry.ID); err != nil {
		return nil, err
	}
	return &UpsertDiscoveredEntryResult{
		RegistryModelID: entry.ID,
		CanonicalModel:  canonicalModelID,
		Changed:         true,
	}, nil
}

func (s *ModelRegistryService) BatchSyncExposures(ctx context.Context, input BatchSyncModelRegistryExposuresInput) (*BatchSyncModelRegistryExposuresResult, error) {
	models := normalizeStringList(input.Models, normalizeRegistryID)
	if len(models) == 0 {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "at least one model is required")
	}
	exposures, err := normalizeBatchSyncExposureTargets(input.Exposures)
	if err != nil {
		return nil, err
	}
	entries, _, _, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return nil, err
	}
	runtimeEntries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return nil, err
	}
	runtimeIndex := make(map[string]int, len(runtimeEntries))
	for index, entry := range runtimeEntries {
		runtimeIndex[entry.ID] = index
	}
	result := &BatchSyncModelRegistryExposuresResult{
		Exposures:     exposures,
		UpdatedModels: []string{},
		SkippedModels: []string{},
		FailedModels:  []ModelRegistryExposureSyncFailure{},
	}
	changedIDs := make([]string, 0, len(models))
	for _, modelID := range models {
		if _, tombstoned := tombstones[modelID]; tombstoned {
			result.SkippedCount++
			result.SkippedModels = append(result.SkippedModels, modelID)
			continue
		}
		entry, exists := entries[modelID]
		if !exists {
			result.SkippedCount++
			result.SkippedModels = append(result.SkippedModels, modelID)
			continue
		}
		mergedExposures := mergeRegistryStrings(entry.ExposedIn, exposures...)
		if sameStringSlice(entry.ExposedIn, mergedExposures) {
			result.SkippedCount++
			result.SkippedModels = append(result.SkippedModels, modelID)
			continue
		}
		updated := entry
		updated.ExposedIn = mergedExposures
		normalized, normalizeErr := normalizePersistedEntry(updated)
		if normalizeErr != nil {
			result.FailedModels = append(result.FailedModels, ModelRegistryExposureSyncFailure{Model: modelID, Error: summarizeAccountModelImportError(normalizeErr)})
			continue
		}
		if index, exists := runtimeIndex[normalized.ID]; exists {
			runtimeEntries[index] = normalized
		} else {
			runtimeIndex[normalized.ID] = len(runtimeEntries)
			runtimeEntries = append(runtimeEntries, normalized)
		}
		changedIDs = append(changedIDs, normalized.ID)
		result.UpdatedCount++
		result.UpdatedModels = append(result.UpdatedModels, normalized.ID)
	}
	sort.Strings(result.UpdatedModels)
	sort.Strings(result.SkippedModels)
	sort.Slice(result.FailedModels, func(i, j int) bool {
		return result.FailedModels[i].Model < result.FailedModels[j].Model
	})
	result.FailedCount = len(result.FailedModels)
	if len(changedIDs) == 0 {
		return result, nil
	}
	if err := s.persistRuntimeEntries(ctx, runtimeEntries); err != nil {
		return nil, err
	}
	for _, modelID := range changedIDs {
		if err := s.clearStates(ctx, modelID); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *ModelRegistryService) SetVisibility(ctx context.Context, input UpdateModelRegistryVisibilityInput) (*modelregistry.AdminModelDetail, error) {
	modelID := normalizeRegistryID(input.Model)
	if modelID == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	set, err := s.loadStringSet(ctx, SettingKeyModelRegistryHiddenModels)
	if err != nil {
		return nil, err
	}
	if input.Hidden {
		set[modelID] = struct{}{}
	} else {
		delete(set, modelID)
	}
	if err := s.persistStringSet(ctx, SettingKeyModelRegistryHiddenModels, set); err != nil {
		return nil, err
	}
	return s.GetDetail(ctx, modelID)
}

func (s *ModelRegistryService) DeleteEntry(ctx context.Context, modelID string) error {
	modelID = normalizeRegistryID(modelID)
	if modelID == "" {
		return infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	entries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return err
	}
	filtered := entries[:0]
	for _, entry := range entries {
		if entry.ID != modelID {
			filtered = append(filtered, entry)
		}
	}
	if err := s.persistRuntimeEntries(ctx, filtered); err != nil {
		return err
	}
	tombstones, err := s.loadStringSet(ctx, SettingKeyModelRegistryTombstones)
	if err != nil {
		return err
	}
	tombstones[modelID] = struct{}{}
	if err := s.persistStringSet(ctx, SettingKeyModelRegistryTombstones, tombstones); err != nil {
		return err
	}
	hidden, err := s.loadStringSet(ctx, SettingKeyModelRegistryHiddenModels)
	if err != nil {
		return err
	}
	delete(hidden, modelID)
	return s.persistStringSet(ctx, SettingKeyModelRegistryHiddenModels, hidden)
}

func (s *ModelRegistryService) visibleSnapshotData(ctx context.Context) ([]modelregistry.ModelEntry, []modelregistry.PresetMapping, error) {
	entries, _, hidden, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return nil, nil, err
	}
	models := make([]modelregistry.ModelEntry, 0, len(entries))
	for id, entry := range entries {
		if _, isHidden := hidden[id]; isHidden {
			continue
		}
		if _, isTombstoned := tombstones[id]; isTombstoned {
			continue
		}
		models = append(models, entry)
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].UIPriority == models[j].UIPriority {
			return models[i].ID < models[j].ID
		}
		return models[i].UIPriority < models[j].UIPriority
	})
	presets := make([]modelregistry.PresetMapping, 0)
	for _, preset := range modelregistry.SeedPresets() {
		if _, hiddenFrom := hidden[normalizeRegistryID(preset.From)]; hiddenFrom {
			continue
		}
		if _, hiddenTo := hidden[normalizeRegistryID(preset.To)]; hiddenTo {
			continue
		}
		if _, tombstoneFrom := tombstones[normalizeRegistryID(preset.From)]; tombstoneFrom {
			continue
		}
		if _, tombstoneTo := tombstones[normalizeRegistryID(preset.To)]; tombstoneTo {
			continue
		}
		presets = append(presets, preset)
	}
	return models, presets, nil
}

func (s *ModelRegistryService) adminDetails(ctx context.Context) ([]modelregistry.AdminModelDetail, error) {
	entries, sources, hidden, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return nil, err
	}
	details := make([]modelregistry.AdminModelDetail, 0, len(entries)+len(tombstones))
	for id, entry := range entries {
		_, isHidden := hidden[id]
		_, isTombstoned := tombstones[id]
		details = append(details, modelregistry.AdminModelDetail{
			ModelEntry: entry,
			Source:     sources[id],
			Hidden:     isHidden,
			Tombstoned: isTombstoned,
		})
	}
	for id := range tombstones {
		if _, exists := entries[id]; exists {
			continue
		}
		details = append(details, modelregistry.AdminModelDetail{
			ModelEntry: modelregistry.ModelEntry{
				ID:               id,
				DisplayName:      FormatModelCatalogDisplayName(id),
				Provider:         inferModelProvider(id),
				Platforms:        defaultPlatformsForProvider(inferModelProvider(id)),
				ProtocolIDs:      []string{id},
				PricingLookupIDs: []string{id},
				Modalities:       defaultModalitiesForMode(inferModelMode(id, "")),
				Capabilities:     defaultCapabilitiesForMode(inferModelMode(id, "")),
				UIPriority:       9999,
				ExposedIn:        []string{"runtime"},
			},
			Source:     "tombstone",
			Hidden:     false,
			Tombstoned: true,
		})
	}
	sort.Slice(details, func(i, j int) bool {
		if details[i].UIPriority == details[j].UIPriority {
			return details[i].ID < details[j].ID
		}
		return details[i].UIPriority < details[j].UIPriority
	})
	return details, nil
}

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
	if canonicalDefault, ok := modelregistry.ModelCatalogCanonicalDefaults()[canonicalModelID]; ok {
		pricingLookupIDs = compactRegistryStrings(canonicalDefault, sourceModelID)
	}
	entry, err := normalizePersistedEntry(modelregistry.ModelEntry{
		ID:               sourceModelID,
		DisplayName:      FormatModelCatalogDisplayName(sourceModelID),
		Provider:         providerOrPlatform(provider, sourcePlatform),
		Platforms:        platforms,
		ProtocolIDs:      compactRegistryStrings(sourceModelID),
		Aliases:          []string{},
		PricingLookupIDs: pricingLookupIDs,
		Modalities:       defaultModalitiesForMode(inferModelMode(sourceModelID, "")),
		Capabilities:     defaultCapabilitiesForMode(inferModelMode(sourceModelID, "")),
		UIPriority:       5000,
		ExposedIn:        []string{"runtime", "test"},
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

func normalizeRuntimeRegistryEntry(input UpsertModelRegistryEntryInput) (modelregistry.ModelEntry, error) {
	return normalizePersistedEntry(modelregistry.ModelEntry{
		ID:               input.ID,
		DisplayName:      input.DisplayName,
		Provider:         input.Provider,
		Platforms:        input.Platforms,
		ProtocolIDs:      input.ProtocolIDs,
		Aliases:          input.Aliases,
		PricingLookupIDs: input.PricingLookupIDs,
		Modalities:       input.Modalities,
		Capabilities:     input.Capabilities,
		UIPriority:       input.UIPriority,
		ExposedIn:        input.ExposedIn,
	})
}

func normalizePersistedEntry(entry modelregistry.ModelEntry) (modelregistry.ModelEntry, error) {
	entry.ID = normalizeRegistryID(entry.ID)
	if entry.ID == "" {
		return modelregistry.ModelEntry{}, infraerrors.BadRequest("MODEL_REQUIRED", "model id is required")
	}
	entry.DisplayName = strings.TrimSpace(entry.DisplayName)
	if entry.DisplayName == "" {
		entry.DisplayName = FormatModelCatalogDisplayName(entry.ID)
	}
	entry.Provider = strings.TrimSpace(strings.ToLower(entry.Provider))
	if entry.Provider == "" {
		entry.Provider = inferModelProvider(entry.ID)
	}
	if len(entry.Platforms) == 0 {
		entry.Platforms = defaultPlatformsForProvider(entry.Provider)
	}
	entry.Platforms = normalizeStringList(entry.Platforms, normalizeRegistryPlatform)
	if len(entry.Platforms) == 0 {
		entry.Platforms = defaultPlatformsForProvider(entry.Provider)
	}
	entry.ProtocolIDs = normalizeStringList(entry.ProtocolIDs, normalizeRegistryID)
	if len(entry.ProtocolIDs) == 0 {
		if defaultID, ok := modelregistry.ModelCatalogCanonicalDefaults()[entry.ID]; ok {
			entry.ProtocolIDs = compactRegistryStrings(defaultID, entry.ID)
		} else {
			entry.ProtocolIDs = []string{entry.ID}
		}
	}
	entry.Aliases = normalizeStringList(entry.Aliases, normalizeRegistryID)
	entry.PricingLookupIDs = normalizeStringList(entry.PricingLookupIDs, normalizeRegistryID)
	if len(entry.PricingLookupIDs) == 0 {
		if defaultID, ok := modelregistry.ModelCatalogCanonicalDefaults()[entry.ID]; ok {
			entry.PricingLookupIDs = []string{defaultID}
		} else {
			entry.PricingLookupIDs = []string{entry.ProtocolIDs[0]}
		}
	}
	entry.Modalities = normalizeStringList(entry.Modalities, normalizeLowerTrimmed)
	if len(entry.Modalities) == 0 {
		entry.Modalities = defaultModalitiesForMode(inferModelMode(entry.ID, ""))
	}
	capabilities, err := normalizeRegistryCapabilities(entry.Capabilities)
	if err != nil {
		return modelregistry.ModelEntry{}, err
	}
	entry.Capabilities = capabilities
	if len(entry.Capabilities) == 0 {
		entry.Capabilities = defaultCapabilitiesForMode(inferModelMode(entry.ID, ""))
	}
	if entry.UIPriority <= 0 {
		if seedEntry, ok := modelregistry.SeedModelByID(entry.ID); ok {
			entry.UIPriority = seedEntry.UIPriority
		} else {
			entry.UIPriority = 5000
		}
	}
	entry.ExposedIn = normalizeStringList(entry.ExposedIn, normalizeLowerTrimmed)
	if len(entry.ExposedIn) == 0 {
		if seedEntry, ok := modelregistry.SeedModelByID(entry.ID); ok && len(seedEntry.ExposedIn) > 0 {
			entry.ExposedIn = append([]string(nil), seedEntry.ExposedIn...)
		} else {
			entry.ExposedIn = []string{"runtime", "whitelist"}
		}
	}
	return entry, nil
}

func defaultPlatformsForProvider(provider string) []string {
	provider = normalizeRegistryPlatform(provider)
	if provider == "" {
		return nil
	}
	return []string{provider}
}

func defaultModalitiesForMode(mode string) []string {
	if mode == "image" {
		return []string{"text", "image"}
	}
	return []string{"text"}
}

func defaultCapabilitiesForMode(mode string) []string {
	if mode == "image" {
		return []string{"image_generation"}
	}
	return []string{"text"}
}

func normalizeRegistryID(value string) string {
	return CanonicalizeModelNameForPricing(strings.TrimSpace(value))
}

func normalizeRegistryPlatform(value string) string {
	value = normalizeLowerTrimmed(value)
	if value == "claude" {
		return PlatformAnthropic
	}
	return value
}

var batchSyncExposureTargets = map[string]struct{}{
	"whitelist": {},
	"use_key":   {},
	"test":      {},
	"runtime":   {},
}

func normalizeBatchSyncExposureTargets(exposures []string) ([]string, error) {
	targets := normalizeStringList(exposures, normalizeLowerTrimmed)
	if len(targets) == 0 {
		return nil, infraerrors.BadRequest("MODEL_REGISTRY_EXPOSURE_REQUIRED", "at least one exposure target is required")
	}
	for _, target := range targets {
		if _, ok := batchSyncExposureTargets[target]; !ok {
			return nil, infraerrors.BadRequest("MODEL_REGISTRY_EXPOSURE_INVALID", "invalid exposure target: "+target)
		}
	}
	return targets, nil
}

func normalizeLowerTrimmed(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func normalizeRegistryCapabilities(items []string) ([]string, error) {
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		normalized, err := normalizeRegistryCapability(item)
		if err != nil {
			return nil, err
		}
		if normalized == "" {
			continue
		}
		seen[normalized] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for _, capability := range modelRegistryCapabilityOrder {
		if _, ok := seen[capability]; ok {
			result = append(result, capability)
		}
	}
	return result, nil
}

func normalizeRegistryCapability(value string) (string, error) {
	value = normalizeLowerTrimmed(value)
	if value == "" {
		return "", nil
	}
	if alias, ok := modelRegistryCapabilityAliases[value]; ok {
		value = alias
	}
	for _, capability := range modelRegistryCapabilityOrder {
		if value == capability {
			return value, nil
		}
	}
	return "", infraerrors.BadRequest("MODEL_REGISTRY_CAPABILITY_INVALID", "invalid capability: "+value)
}

func normalizeStringList(items []string, normalize func(string) string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = normalize(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func augmentDiscoveredEntry(entry modelregistry.ModelEntry, sourceModelID string, canonicalModelID string, sourcePlatform string) (modelregistry.ModelEntry, bool, error) {
	updated := modelregistry.ModelEntry{
		ID:               entry.ID,
		DisplayName:      entry.DisplayName,
		Provider:         entry.Provider,
		Platforms:        append([]string(nil), entry.Platforms...),
		ProtocolIDs:      append([]string(nil), entry.ProtocolIDs...),
		Aliases:          append([]string(nil), entry.Aliases...),
		PricingLookupIDs: append([]string(nil), entry.PricingLookupIDs...),
		Modalities:       append([]string(nil), entry.Modalities...),
		Capabilities:     append([]string(nil), entry.Capabilities...),
		UIPriority:       entry.UIPriority,
		ExposedIn:        append([]string(nil), entry.ExposedIn...),
	}
	changed := false
	platforms, err := discoveredPlatforms(updated.Provider, sourcePlatform)
	if err != nil {
		return modelregistry.ModelEntry{}, false, err
	}
	if merged := mergeRegistryStrings(updated.Platforms, platforms...); !sameStringSlice(updated.Platforms, merged) {
		updated.Platforms = merged
		changed = true
	}
	if merged := mergeRegistryStrings(updated.ExposedIn, "runtime", "test"); !sameStringSlice(updated.ExposedIn, merged) {
		updated.ExposedIn = merged
		changed = true
	}
	if sourceModelID != "" && sourceModelID != updated.ID {
		merged := mergeRegistryStrings(updated.ProtocolIDs, sourceModelID)
		if !sameStringSlice(updated.ProtocolIDs, merged) {
			updated.ProtocolIDs = merged
			changed = true
		}
	}
	if canonicalDefault, ok := modelregistry.ModelCatalogCanonicalDefaults()[canonicalModelID]; ok {
		merged := mergeRegistryStrings(updated.PricingLookupIDs, canonicalDefault)
		if !sameStringSlice(updated.PricingLookupIDs, merged) {
			updated.PricingLookupIDs = merged
			changed = true
		}
	}
	normalized, err := normalizePersistedEntry(updated)
	if err != nil {
		return modelregistry.ModelEntry{}, false, err
	}
	return normalized, changed, nil
}

func discoveredPlatforms(provider string, sourcePlatform string) ([]string, error) {
	platform := normalizeRegistryPlatform(sourcePlatform)
	if isRuntimeSupportedPlatform(platform) {
		return []string{platform}, nil
	}
	platforms := defaultPlatformsForProvider(provider)
	if len(platforms) > 0 {
		return platforms, nil
	}
	return nil, infraerrors.BadRequest("MODEL_RUNTIME_PLATFORM_UNSUPPORTED", "unable to infer runtime platform from imported model")
}

func providerOrPlatform(provider string, sourcePlatform string) string {
	provider = normalizeRegistryPlatform(provider)
	if provider != "" {
		return provider
	}
	return normalizeRegistryPlatform(sourcePlatform)
}

func isRuntimeSupportedPlatform(platform string) bool {
	switch normalizeRegistryPlatform(platform) {
	case PlatformOpenAI, PlatformAnthropic, PlatformGemini, PlatformAntigravity, PlatformSora:
		return true
	default:
		return false
	}
}

func mergeRegistryStrings(current []string, items ...string) []string {
	merged := append([]string(nil), current...)
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		exists := false
		for _, existing := range merged {
			if existing == item {
				exists = true
				break
			}
		}
		if !exists {
			merged = append(merged, item)
		}
	}
	return merged
}

func sameStringSlice(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func compactRegistryStrings(items ...string) []string {
	return normalizeStringList(items, normalizeRegistryID)
}

func computeRegistryETag(models []modelregistry.ModelEntry, presets []modelregistry.PresetMapping) (string, error) {
	payload, err := json.Marshal(struct {
		Models  []modelregistry.ModelEntry    `json:"models"`
		Presets []modelregistry.PresetMapping `json:"presets"`
	}{Models: models, Presets: presets})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "W/\"" + hex.EncodeToString(sum[:]) + "\"", nil
}
