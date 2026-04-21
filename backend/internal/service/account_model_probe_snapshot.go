package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

const (
	accountModelProbeSnapshotExtraKey               = "model_probe_snapshot"
	AccountModelProbeSnapshotSourceImportModels     = "import_models"
	AccountModelProbeSnapshotSourceTestProbe        = "test_probe"
	AccountModelProbeSnapshotSourceManualProbe      = "manual_probe"
	AccountModelProbeSnapshotSourcePublicModelsLive = "public_models_live_probe"
)

type AccountModelProbeSnapshot struct {
	Models      []string `json:"models"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
	Source      string   `json:"source,omitempty"`
	ProbeSource string   `json:"probe_source,omitempty"`
}

func BuildAccountModelProbeSnapshotExtra(models []string, updatedAt time.Time, source string, probeSource string) map[string]any {
	normalizedModels := normalizeAccountModelProbeSnapshotModels(models)
	if len(normalizedModels) == 0 {
		return nil
	}

	snapshot := map[string]any{
		"models": normalizedModels,
	}
	if !updatedAt.IsZero() {
		snapshot["updated_at"] = updatedAt.UTC().Format(time.RFC3339)
	}
	if trimmedSource := strings.TrimSpace(source); trimmedSource != "" {
		snapshot["source"] = trimmedSource
	}
	if trimmedProbeSource := strings.TrimSpace(probeSource); trimmedProbeSource != "" {
		snapshot["probe_source"] = trimmedProbeSource
	}

	return map[string]any{
		accountModelProbeSnapshotExtraKey: snapshot,
	}
}

func AccountModelProbeSnapshotFromExtra(extra map[string]any) (*AccountModelProbeSnapshot, bool) {
	if len(extra) == 0 {
		return nil, false
	}
	rawSnapshot, ok := extra[accountModelProbeSnapshotExtraKey]
	if !ok || rawSnapshot == nil {
		return nil, false
	}
	snapshotMap, ok := rawSnapshot.(map[string]any)
	if !ok || len(snapshotMap) == 0 {
		return nil, false
	}

	models := normalizeStringSliceAny(snapshotMap["models"], NormalizeModelCatalogModelID)
	if len(models) == 0 {
		return nil, false
	}

	snapshot := &AccountModelProbeSnapshot{
		Models:      models,
		UpdatedAt:   stringValueFromAny(snapshotMap["updated_at"]),
		Source:      stringValueFromAny(snapshotMap["source"]),
		ProbeSource: stringValueFromAny(snapshotMap["probe_source"]),
	}
	return snapshot, true
}

func AccountSavedModelProbeSummary(account *Account) *AccountModelProbeSummary {
	if account == nil {
		return nil
	}

	if snapshot, ok := AccountModelProbeSnapshotFromExtra(account.Extra); ok && snapshot != nil {
		return buildSavedAccountModelProbeSummary(
			snapshot.Models,
			firstNonEmptyTrimmed(snapshot.ProbeSource, snapshot.Source),
		)
	}

	return buildSavedAccountModelProbeSummary(AccountSavedModelIDs(account), "")
}

func AccountSavedModelIDs(account *Account) []string {
	if account == nil {
		return nil
	}

	allowSourceProtocol := IsProtocolGatewayAccount(account)
	ordered := make([]string, 0)
	seen := make(map[string]struct{})
	appendModel := func(modelID string) {
		normalized := NormalizeModelCatalogModelID(modelID)
		if normalized == "" {
			return
		}
		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		ordered = append(ordered, normalized)
	}

	for _, model := range AccountManualModelsFromExtra(account.Extra, allowSourceProtocol) {
		appendModel(model.ModelID)
	}

	if scope, ok := ExtractAccountModelScopeV2(account.Extra); ok && scope != nil {
		providers := make([]string, 0, len(scope.SupportedModelsByProvider))
		for provider := range scope.SupportedModelsByProvider {
			providers = append(providers, provider)
		}
		sort.Strings(providers)
		for _, provider := range providers {
			for _, modelID := range scope.SupportedModelsByProvider[provider] {
				appendModel(modelID)
			}
		}
		for _, row := range scope.ManualMappingRows {
			appendModel(row.To)
		}
		keys := make([]string, 0, len(scope.ManualMappings))
		for from := range scope.ManualMappings {
			keys = append(keys, from)
		}
		sort.Strings(keys)
		for _, from := range keys {
			appendModel(scope.ManualMappings[from])
		}
	}

	mapping := account.GetModelMapping()
	if len(mapping) > 0 {
		aliases := make([]string, 0, len(mapping))
		for alias := range mapping {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
		for _, alias := range aliases {
			appendModel(mapping[alias])
		}
	}

	for _, modelID := range normalizeStringSliceAny(account.Extra["openai_known_models"], NormalizeModelCatalogModelID) {
		appendModel(modelID)
	}

	if len(ordered) == 0 {
		return nil
	}
	return ordered
}

func buildSavedAccountModelProbeSummary(modelIDs []string, probeSource string) *AccountModelProbeSummary {
	normalized := normalizeAccountModelProbeSnapshotModels(modelIDs)
	if len(normalized) == 0 {
		return nil
	}

	details := make([]AccountModelProbeModel, 0, len(normalized))
	for _, modelID := range normalized {
		details = append(details, applyAccountModelProbeProvider(AccountModelProbeModel{
			ID:          modelID,
			DisplayName: FormatModelCatalogDisplayName(modelID),
		}, ""))
	}

	return &AccountModelProbeSummary{
		DetectedModels: normalized,
		Models:         details,
		ProbeSource:    strings.TrimSpace(probeSource),
	}
}

func AvailableTestModelsFromProbeSnapshot(
	ctx context.Context,
	account *Account,
	registry *ModelRegistryService,
	snapshot *AccountModelProbeSnapshot,
) []AvailableTestModel {
	if snapshot == nil || len(snapshot.Models) == 0 {
		return []AvailableTestModel{}
	}
	sourceProtocols := protocolGatewayTestSourceProtocols(account)
	if len(sourceProtocols) == 0 {
		return buildAvailableTestModelsFromProbeSnapshotSource(ctx, registry, snapshot.Models, "")
	}

	groups := make([][]AvailableTestModel, 0, len(sourceProtocols))
	for _, sourceProtocol := range sourceProtocols {
		groups = append(groups, buildAvailableTestModelsFromProbeSnapshotSource(ctx, registry, snapshot.Models, sourceProtocol))
	}
	return MergeAvailableTestModels(groups...)
}

func buildAvailableTestModelsFromProbeSnapshotSource(
	ctx context.Context,
	registry *ModelRegistryService,
	modelIDs []string,
	sourceProtocol string,
) []AvailableTestModel {
	models := make([]AvailableTestModel, 0, len(modelIDs))
	for _, modelID := range normalizeAccountModelProbeSnapshotModels(modelIDs) {
		models = append(models, buildAvailableTestModelFromSnapshotModelID(ctx, registry, modelID, sourceProtocol))
	}
	sort.SliceStable(models, func(i, j int) bool {
		return compareAvailableTestModels(models[i], models[j]) < 0
	})
	return models
}

func buildAvailableTestModelFromSnapshotModelID(
	ctx context.Context,
	registry *ModelRegistryService,
	modelID string,
	sourceProtocol string,
) AvailableTestModel {
	normalizedModelID := NormalizeModelCatalogModelID(modelID)
	if normalizedModelID == "" {
		normalizedModelID = normalizeRegistryID(modelID)
	}
	if registry != nil {
		if detail, err := registry.GetDetail(ctx, normalizedModelID); err == nil && detail != nil {
			model := buildAvailableTestModelFromRegistryDetail(*detail, sourceProtocol)
			if model.CanonicalID == "" {
				model.CanonicalID = normalizeRegistryID(detail.ID)
			}
			return model
		}
		if resolution, err := registry.ExplainResolution(ctx, normalizedModelID); err == nil && resolution != nil {
			entry := resolution.Entry
			if normalizeRegistryID(resolution.EffectiveID) != "" && resolution.ReplacementEntry != nil {
				entry = *resolution.ReplacementEntry
			}
			model := applyAvailableTestModelProvider(AvailableTestModel{
				ID:             normalizedModelID,
				Type:           "model",
				DisplayName:    firstNonEmptyTestModelLabel(entry.DisplayName, FormatModelCatalogDisplayName(normalizedModelID), normalizedModelID),
				CanonicalID:    firstNonEmptyTrimmed(resolution.CanonicalID, normalizedModelID),
				Mode:           inferAvailableTestModelMode(normalizedModelID, &entry),
				SourceProtocol: normalizeTestSourceProtocol(sourceProtocol),
				Status:         "stable",
			}, entry.Provider)
			return model
		}
	}

	return applyAvailableTestModelProvider(AvailableTestModel{
		ID:             normalizedModelID,
		Type:           "model",
		DisplayName:    firstNonEmptyTestModelLabel(FormatModelCatalogDisplayName(normalizedModelID), normalizedModelID),
		CanonicalID:    normalizedModelID,
		Mode:           inferAvailableTestModelMode(normalizedModelID, (*modelregistry.ModelEntry)(nil)),
		SourceProtocol: normalizeTestSourceProtocol(sourceProtocol),
		Status:         "stable",
	}, inferModelProvider(normalizedModelID))
}

func normalizeAccountModelProbeSnapshotModels(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, modelID := range models {
		normalizedID := NormalizeModelCatalogModelID(modelID)
		if normalizedID == "" {
			continue
		}
		if _, exists := seen[normalizedID]; exists {
			continue
		}
		seen[normalizedID] = struct{}{}
		normalized = append(normalized, normalizedID)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
