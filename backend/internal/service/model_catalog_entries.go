package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

//go:embed model_catalog_seed.json
var modelCatalogSeedJSON []byte

type ModelCatalogEntry struct {
	Model                string `json:"model"`
	DisplayName          string `json:"display_name,omitempty"`
	Provider             string `json:"provider,omitempty"`
	Mode                 string `json:"mode,omitempty"`
	CanonicalModelID     string `json:"canonical_model_id,omitempty"`
	PricingLookupModelID string `json:"pricing_lookup_model_id,omitempty"`
}

type UpsertModelCatalogEntryInput struct {
	Model string `json:"model"`
}

type CopyModelCatalogPricingFromOfficialInput struct {
	Model string `json:"model"`
}

var modelCatalogExplicitAliases = map[string]string{
	"claude-opus-4-1":                "claude-opus-4.1",
	"claude-opus-4-1-20250805":       "claude-opus-4.1",
	"claude-opus-4.1-20250805":       "claude-opus-4.1",
	"claude-opus-4-5":                "claude-opus-4.1",
	"claude-opus-4-5-20251101":       "claude-opus-4.1",
	"claude-opus-4.5-20251101":       "claude-opus-4.1",
	"claude-opus-4-5-thinking":       "claude-opus-4.1",
	"claude-opus-4.5-thinking":       "claude-opus-4.1",
	"claude-opus-4-6":                "claude-opus-4.1",
	"claude-opus-4-6-thinking":       "claude-opus-4.1",
	"claude-sonnet-4-5":              "claude-sonnet-4.5",
	"claude-sonnet-4-5-20250929":     "claude-sonnet-4.5",
	"claude-sonnet-4.5-20250929":     "claude-sonnet-4.5",
	"claude-sonnet-4-5-thinking":     "claude-sonnet-4.5",
	"claude-sonnet-4.5-thinking":     "claude-sonnet-4.5",
	"claude-sonnet-4-6":              "claude-sonnet-4.5",
	"claude-sonnet-4-6-thinking":     "claude-sonnet-4.5",
	"claude-haiku-4-5":               "claude-haiku-4.5",
	"claude-haiku-4-5-20251001":      "claude-haiku-4.5",
	"claude-haiku-4.5-20251001":      "claude-haiku-4.5",
	"gpt-5-4":                        "gpt-5.4",
	"gpt-5.4":                        "gpt-5.4",
	"gpt-5-4-2026-03-05":             "gpt-5.4",
	"gpt-5.4-2026-03-05":             "gpt-5.4",
	"gpt-5-4-chat-latest":            "gpt-5.4",
	"gpt-5.4-chat-latest":            "gpt-5.4",
	"gpt-5-4-pro":                    "gpt-5.4-pro",
	"gpt-5.4-pro":                    "gpt-5.4-pro",
	"gpt-5-3-codex":                  "gpt-5-codex",
	"gpt-5.3-codex":                  "gpt-5-codex",
	"gpt-5-2-codex":                  "gpt-5-codex",
	"gpt-5.2-codex":                  "gpt-5-codex",
	"gpt-5-1-codex":                  "gpt-5-codex",
	"gpt-5.1-codex":                  "gpt-5-codex",
	"gemini-2.5-flash-image-preview": "gemini-2.5-flash-image",
	"gemini-2.5-flash-thinking":      "gemini-2.5-flash",
	"gemini-3-flash-preview":         "gemini-3-flash",
	"gemini-3-pro-preview":           "gemini-3-pro",
	"gemini-3-pro-high":              "gemini-3-pro",
	"gemini-3-pro-low":               "gemini-3-pro",
	"gemini-3-pro-image-preview":     "gemini-3-pro-image",
	"gemini-3.1-pro-preview":         "gemini-3.1-pro",
	"gemini-3.1-pro-high":            "gemini-3.1-pro",
	"gemini-3.1-pro-low":             "gemini-3.1-pro",
	"gemini-3.1-flash-lite-preview":  "gemini-3.1-flash-lite",
	"gemini-3.1-flash-image-preview": "gemini-3.1-flash-image",
}

var modelCatalogCanonicalDefaults = map[string]string{
	"claude-opus-4.1":        "claude-opus-4-1-20250805",
	"claude-sonnet-4.5":      "claude-sonnet-4-5-20250929",
	"claude-haiku-4.5":       "claude-haiku-4-5-20251001",
	"gpt-5.4":                "gpt-5.4",
	"gpt-5.4-pro":            "gpt-5.4-pro",
	"gpt-5-mini":             "gpt-5-mini",
	"gpt-5-nano":             "gpt-5-nano",
	"gpt-5-codex":            "gpt-5-codex",
	"gemini-2.5-flash-image": "gemini-2.5-flash-image",
	"gemini-3-pro":           "gemini-3-pro-preview",
	"gemini-3-flash":         "gemini-3-flash-preview",
	"gemini-3-pro-image":     "gemini-3-pro-image-preview",
	"gemini-3.1-pro":         "gemini-3.1-pro-preview",
	"gemini-3.1-flash-lite":  "gemini-3.1-flash-lite-preview",
	"gemini-3.1-flash-image": "gemini-3.1-flash-image-preview",
}

func normalizeModelCatalogAlias(model string) string {
	canonical := CanonicalizeModelNameForPricing(model)
	if canonical == "" {
		return ""
	}
	trimmed := modelCatalogDateVersionSuffixPattern.ReplaceAllString(canonical, "")
	if alias, ok := modelCatalogExplicitAliases[canonical]; ok {
		return alias
	}
	if alias, ok := modelCatalogExplicitAliases[trimmed]; ok {
		return alias
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
		if canonical, ok := modelCatalogCanonicalDefaults[entry.Model]; ok {
			entry.CanonicalModelID = canonical
		} else {
			entry.CanonicalModelID = CanonicalizeModelNameForPricing(entry.Model)
		}
	}
	if entry.PricingLookupModelID == "" {
		entry.PricingLookupModelID = entry.CanonicalModelID
	}
	return entry
}

func loadSeedModelCatalogEntries() []ModelCatalogEntry {
	var entries []ModelCatalogEntry
	if err := json.Unmarshal(modelCatalogSeedJSON, &entries); err != nil {
		panic(err)
	}
	normalized := make([]ModelCatalogEntry, 0, len(entries))
	for _, entry := range entries {
		entry = normalizeModelCatalogEntry(entry)
		if entry.Model == "" {
			continue
		}
		normalized = append(normalized, entry)
	}
	return normalized
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
