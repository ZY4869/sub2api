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

type ModelRegistryService struct {
	settingRepo SettingRepository
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

func (s *ModelRegistryService) UpsertDiscoveredEntry(ctx context.Context, modelID string) (bool, bool, error) {
	modelID = normalizeModelCatalogAlias(modelID)
	if modelID == "" {
		return false, false, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	tombstones, err := s.loadStringSet(ctx, SettingKeyModelRegistryTombstones)
	if err != nil {
		return false, false, err
	}
	if _, blocked := tombstones[modelID]; blocked {
		return false, true, nil
	}
	entries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return false, false, err
	}
	for _, current := range entries {
		if current.ID == modelID {
			return false, false, nil
		}
	}
	entry, err := normalizeRuntimeRegistryEntry(UpsertModelRegistryEntryInput{ID: modelID})
	if err != nil {
		return false, false, err
	}
	entries = append(entries, entry)
	if err := s.persistRuntimeEntries(ctx, entries); err != nil {
		return false, false, err
	}
	if err := s.clearStates(ctx, modelID); err != nil {
		return true, false, err
	}
	return true, false, nil
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
	legacyEntries, err := s.loadLegacyCatalogEntries(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for _, entry := range legacyEntries {
		if _, exists := entries[entry.ID]; exists {
			continue
		}
		entries[entry.ID] = entry
		sources[entry.ID] = "legacy_catalog"
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
			ExposedIn:        []string{"legacy_catalog"},
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
	entry.Capabilities = normalizeStringList(entry.Capabilities, normalizeLowerTrimmed)
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
		return []string{"openai"}
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
		return []string{"image"}
	}
	return []string{}
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

func normalizeLowerTrimmed(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
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
