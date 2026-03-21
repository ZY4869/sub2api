package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *ModelRegistryService) List(ctx context.Context, filter ModelRegistryListFilter) ([]modelregistry.AdminModelDetail, int64, error) {
	details, err := s.adminDetails(ctx)
	if err != nil {
		return nil, 0, err
	}
	filtered := make([]modelregistry.AdminModelDetail, 0, len(details))
	search := strings.TrimSpace(strings.ToLower(filter.Search))
	provider := strings.TrimSpace(strings.ToLower(filter.Provider))
	platform := normalizeRegistryPlatform(filter.Platform)
	availability := strings.TrimSpace(strings.ToLower(filter.Availability))
	for _, detail := range details {
		if !filter.IncludeHidden && detail.Hidden {
			continue
		}
		if !filter.IncludeTombstoned && detail.Tombstoned {
			continue
		}
		switch availability {
		case "", "all":
		case "available":
			if !detail.Available {
				continue
			}
		case "unavailable":
			if detail.Available {
				continue
			}
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

func (s *ModelRegistryService) ListProviderSummaries(ctx context.Context, page int, pageSize int) ([]ModelRegistryProviderSummary, int64, error) {
	details, err := s.adminDetails(ctx)
	if err != nil {
		return nil, 0, err
	}

	groups := make(map[string]*ModelRegistryProviderSummary)
	for _, detail := range details {
		if detail.Hidden || detail.Tombstoned {
			continue
		}
		provider := strings.TrimSpace(strings.ToLower(detail.Provider))
		if provider == "" {
			provider = "unknown"
		}
		group := groups[provider]
		if group == nil {
			group = &ModelRegistryProviderSummary{Provider: provider}
			groups[provider] = group
		}
		group.TotalCount++
		if detail.Available {
			group.AvailableCount++
		}
	}

	summaries := make([]ModelRegistryProviderSummary, 0, len(groups))
	for _, summary := range groups {
		summaries = append(summaries, *summary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].TotalCount == summaries[j].TotalCount {
			return summaries[i].Provider < summaries[j].Provider
		}
		return summaries[i].TotalCount > summaries[j].TotalCount
	})

	total := int64(len(summaries))
	page, pageSize = normalizeListPagination(page, pageSize)
	start := (page - 1) * pageSize
	if start >= len(summaries) {
		return []ModelRegistryProviderSummary{}, total, nil
	}
	end := start + pageSize
	if end > len(summaries) {
		end = len(summaries)
	}
	return summaries[start:end], total, nil
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
	canonicalModelID := sourceModelID
	if resolved, ok := modelregistry.ResolveToCanonicalID(sourceModelID); ok {
		canonicalModelID = resolved
	} else if resolution, err := s.ExplainResolution(ctx, sourceModelID); err == nil && resolution != nil {
		if resolution.EffectiveID != "" {
			canonicalModelID = resolution.EffectiveID
		} else if resolution.CanonicalID != "" {
			canonicalModelID = resolution.CanonicalID
		}
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
	runtimeEntries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return nil, err
	}
	for _, runtimeEntry := range runtimeEntries {
		if runtimeEntry.ID != sourceModelID {
			continue
		}
		changed, err := s.ensureDiscoveredRuntimeAvailability(ctx, runtimeEntry, sourceModelID, canonicalModelID, input.SourcePlatform)
		if err != nil {
			return nil, err
		}
		return &UpsertDiscoveredEntryResult{
			RegistryModelID: runtimeEntry.ID,
			CanonicalModel:  canonicalModelID,
			Changed:         changed,
			Existing:        !changed,
		}, nil
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
	if err := s.persistStringSet(ctx, SettingKeyModelRegistryHiddenModels, hidden); err != nil {
		return err
	}
	available, err := s.loadStringSet(ctx, SettingKeyModelRegistryAvailableModels)
	if err != nil {
		return err
	}
	delete(available, modelID)
	return s.persistStringSet(ctx, SettingKeyModelRegistryAvailableModels, available)
}

func (s *ModelRegistryService) adminDetails(ctx context.Context) ([]modelregistry.AdminModelDetail, error) {
	availableSet, err := s.loadAvailableModelSet(ctx)
	if err != nil {
		return nil, err
	}
	entries, sources, hidden, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return nil, err
	}
	details := make([]modelregistry.AdminModelDetail, 0, len(entries)+len(tombstones))
	for id, entry := range entries {
		_, isHidden := hidden[id]
		_, isTombstoned := tombstones[id]
		_, isAvailable := availableSet[id]
		details = append(details, modelregistry.AdminModelDetail{
			ModelEntry: entry,
			Source:     sources[id],
			Hidden:     isHidden,
			Tombstoned: isTombstoned,
			Available:  isAvailable && !isTombstoned,
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
			Available:  false,
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
