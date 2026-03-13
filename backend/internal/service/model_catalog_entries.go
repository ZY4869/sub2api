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

type ModelCatalogEntry struct {
	Model                string `json:"model"`
	DisplayName          string `json:"display_name,omitempty"`
	Provider             string `json:"provider,omitempty"`
	Mode                 string `json:"mode,omitempty"`
	CanonicalModelID     string `json:"canonical_model_id,omitempty"`
	PricingLookupModelID string `json:"pricing_lookup_model_id,omitempty"`
}

type CopyModelCatalogPricingFromOfficialInput struct {
	Model string `json:"model"`
}

func normalizeModelCatalogAlias(model string) string {
	canonical := CanonicalizeModelNameForPricing(model)
	if canonical == "" {
		return ""
	}
	if resolution, ok := modelregistry.ExplainSeedResolution(canonical); ok && resolution != nil {
		if resolution.EffectiveID != "" {
			return resolution.EffectiveID
		}
		if resolution.CanonicalID != "" {
			return resolution.CanonicalID
		}
	}
	trimmed := modelCatalogDateVersionSuffixPattern.ReplaceAllString(canonical, "")
	if resolution, ok := modelregistry.ExplainSeedResolution(trimmed); ok && resolution != nil {
		if resolution.EffectiveID != "" {
			return resolution.EffectiveID
		}
		if resolution.CanonicalID != "" {
			return resolution.CanonicalID
		}
	}
	if trimmed == "" {
		return canonical
	}
	return trimmed
}

func normalizeModelCatalogEntry(entry ModelCatalogEntry) ModelCatalogEntry {
	entry.Model = normalizeModelCatalogAlias(entry.Model)
	entry.CanonicalModelID = CanonicalizeModelNameForPricing(strings.TrimSpace(entry.CanonicalModelID))
	entry.PricingLookupModelID = CanonicalizeModelNameForPricing(strings.TrimSpace(entry.PricingLookupModelID))
	entry.Provider = strings.TrimSpace(strings.ToLower(entry.Provider))
	entry.Mode = strings.TrimSpace(strings.ToLower(entry.Mode))
	entry.DisplayName = strings.TrimSpace(entry.DisplayName)
	if entry.DisplayName == "" {
		entry.DisplayName = FormatModelCatalogDisplayName(entry.Model)
	}
	if entry.Provider == "" {
		entry.Provider = inferModelProvider(entry.Model)
	}
	if entry.Mode == "" {
		entry.Mode = inferModelMode(entry.Model, "")
	}
	if entry.CanonicalModelID == "" {
		entry.CanonicalModelID = CanonicalizeModelNameForPricing(entry.Model)
	}
	if entry.PricingLookupModelID == "" {
		entry.PricingLookupModelID = entry.CanonicalModelID
	}
	return entry
}

func loadSeedModelCatalogEntries() []ModelCatalogEntry {
	registryEntries := buildCatalogBaselineRegistryEntries(modelregistry.SeedModels(), false)
	normalized := make([]ModelCatalogEntry, 0, len(registryEntries))
	seen := map[string]struct{}{}
	for _, entry := range registryEntries {
		item := normalizeModelCatalogEntry(ModelCatalogEntry{
			Model:                entry.ID,
			DisplayName:          entry.DisplayName,
			Provider:             entry.Provider,
			Mode:                 inferModelMode(entry.ID, ""),
			CanonicalModelID:     entry.ID,
			PricingLookupModelID: firstRegistryString(entry.PricingLookupIDs...),
		})
		if item.Model == "" {
			continue
		}
		if _, exists := seen[item.Model]; exists {
			continue
		}
		seen[item.Model] = struct{}{}
		normalized = append(normalized, item)
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Provider == normalized[j].Provider {
			return normalized[i].Model < normalized[j].Model
		}
		return normalized[i].Provider < normalized[j].Provider
	})
	return normalized
}

func buildCatalogBaselineRegistryEntries(entries []modelregistry.ModelEntry, includeRuntime bool) []modelregistry.ModelEntry {
	items := make([]modelregistry.ModelEntry, 0, len(entries))
	seen := map[string]struct{}{}
	appendEntry := func(entry modelregistry.ModelEntry) {
		if strings.EqualFold(strings.TrimSpace(entry.Status), "deprecated") {
			return
		}
		key := normalizeModelCatalogAlias(entry.ID)
		if key == "" {
			key = CanonicalizeModelNameForPricing(entry.ID)
		}
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		items = append(items, entry)
	}

	for _, entry := range entries {
		if modelregistry.HasExposure(entry, "whitelist") || modelregistry.HasExposure(entry, "use_key") {
			appendEntry(entry)
		}
	}
	if !includeRuntime {
		return items
	}
	for _, entry := range entries {
		if modelregistry.HasExposure(entry, "whitelist") || modelregistry.HasExposure(entry, "use_key") {
			continue
		}
		if modelregistry.HasExposure(entry, "runtime") {
			appendEntry(entry)
		}
	}
	return items
}

func buildCatalogBaselineAdminDetails(details []modelregistry.AdminModelDetail, includeRuntime bool) []modelregistry.AdminModelDetail {
	filtered := make([]modelregistry.AdminModelDetail, 0, len(details))
	seen := map[string]struct{}{}
	appendDetail := func(detail modelregistry.AdminModelDetail) {
		if strings.EqualFold(strings.TrimSpace(detail.Status), "deprecated") {
			return
		}
		key := normalizeModelCatalogAlias(detail.ID)
		if key == "" {
			key = CanonicalizeModelNameForPricing(detail.ID)
		}
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		filtered = append(filtered, detail)
	}

	for _, detail := range details {
		if modelregistry.HasExposure(detail.ModelEntry, "whitelist") || modelregistry.HasExposure(detail.ModelEntry, "use_key") {
			appendDetail(detail)
		}
	}
	if !includeRuntime {
		return filtered
	}
	for _, detail := range details {
		if modelregistry.HasExposure(detail.ModelEntry, "whitelist") || modelregistry.HasExposure(detail.ModelEntry, "use_key") {
			continue
		}
		if modelregistry.HasExposure(detail.ModelEntry, "runtime") {
			appendDetail(detail)
		}
	}
	return filtered
}

func (s *ModelCatalogService) loadCatalogEntries(ctx context.Context) []ModelCatalogEntry {
	seed := loadSeedModelCatalogEntries()
	if s.settingRepo == nil {
		return seed
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyModelCatalogEntries)
	if err != nil || raw == "" {
		return seed
	}
	var entries []ModelCatalogEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		logger.FromContext(ctx).Warn("model catalog: invalid entries json, using seed", zap.Error(err))
		return seed
	}
	normalized := make([]ModelCatalogEntry, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		entry = normalizeModelCatalogEntry(entry)
		if entry.Model == "" {
			continue
		}
		if _, ok := seen[entry.Model]; ok {
			continue
		}
		seen[entry.Model] = struct{}{}
		normalized = append(normalized, entry)
	}
	return normalized
}

func (s *ModelCatalogService) persistCatalogEntries(ctx context.Context, entries []ModelCatalogEntry) error {
	if s.settingRepo == nil {
		return nil
	}
	normalized := make([]ModelCatalogEntry, 0, len(entries))
	seen := map[string]struct{}{}
	for _, entry := range entries {
		entry = normalizeModelCatalogEntry(entry)
		if entry.Model == "" {
			continue
		}
		if _, ok := seen[entry.Model]; ok {
			continue
		}
		seen[entry.Model] = struct{}{}
		normalized = append(normalized, entry)
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Provider == normalized[j].Provider {
			return normalized[i].Model < normalized[j].Model
		}
		return normalized[i].Provider < normalized[j].Provider
	})
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, SettingKeyModelCatalogEntries, string(payload))
}

func (s *ModelCatalogService) deriveCatalogEntry(model string) (*ModelCatalogEntry, error) {
	alias := normalizeModelCatalogAlias(model)
	if alias == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	provider := inferModelProvider(alias)
	if provider == "" {
		return nil, infraerrors.BadRequest("MODEL_PROVIDER_UNKNOWN", "unable to infer provider from model")
	}
	entry := normalizeModelCatalogEntry(ModelCatalogEntry{
		Model:       alias,
		DisplayName: FormatModelCatalogDisplayName(alias),
		Provider:    provider,
		Mode:        inferModelMode(alias, ""),
	})
	for _, seedEntry := range loadSeedModelCatalogEntries() {
		if seedEntry.Model == alias {
			copy := seedEntry
			return &copy, nil
		}
	}
	return &entry, nil
}

func modelCatalogEntryByModel(entries []ModelCatalogEntry, model string) (*ModelCatalogEntry, int) {
	alias := normalizeModelCatalogAlias(model)
	for index := range entries {
		if entries[index].Model == alias {
			return &entries[index], index
		}
	}
	return nil, -1
}
