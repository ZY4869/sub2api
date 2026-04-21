package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

const (
	accountModelProbeSnapshotExtraKey                = "model_probe_snapshot"
	AccountModelProbeSnapshotSourceImportModels      = "import_models"
	AccountModelProbeSnapshotSourceModelScopePreview = "model_scope_preview"
	AccountModelProbeSnapshotSourceTestProbe         = "test_probe"
	AccountModelProbeSnapshotSourceManualProbe       = "manual_probe"
	AccountModelProbeSnapshotSourcePublicModelsLive  = "public_models_live_probe"
	AccountModelProbeSnapshotSourcePolicyUpdate      = "policy_update"
)

type AccountModelProbeSnapshotEntry struct {
	DisplayModelID    string `json:"display_model_id,omitempty"`
	TargetModelID     string `json:"target_model_id,omitempty"`
	AvailabilityState string `json:"availability_state,omitempty"`
	StaleState        string `json:"stale_state,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
	Source            string `json:"source,omitempty"`
}

type AccountModelProbeSnapshot struct {
	Models      []string                         `json:"models,omitempty"`
	Entries     []AccountModelProbeSnapshotEntry `json:"entries,omitempty"`
	UpdatedAt   string                           `json:"updated_at,omitempty"`
	Source      string                           `json:"source,omitempty"`
	ProbeSource string                           `json:"probe_source,omitempty"`
}

func BuildAccountModelProbeSnapshotExtra(models []string, updatedAt time.Time, source string, probeSource string) map[string]any {
	normalizedModels := normalizeAccountModelProbeSnapshotModels(models)
	if len(normalizedModels) == 0 {
		return nil
	}

	entries := make([]AccountModelProbeSnapshotEntry, 0, len(normalizedModels))
	for _, modelID := range normalizedModels {
		entries = append(entries, AccountModelProbeSnapshotEntry{
			DisplayModelID:    modelID,
			TargetModelID:     modelID,
			AvailabilityState: AccountModelAvailabilityVerified,
			StaleState:        buildAccountModelProbeSnapshotStaleState(updatedAt, source, probeSource),
			UpdatedAt:         formatAccountModelProbeSnapshotUpdatedAt(updatedAt),
			Source:            firstNonEmptyTrimmed(probeSource, source),
		})
	}

	snapshot := map[string]any{
		"models":  normalizedModels,
		"entries": accountModelProbeSnapshotEntriesToAny(entries),
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

func BuildAccountModelScopePreviewSnapshotExtra(scope *AccountModelScopeV2) map[string]any {
	if scope == nil {
		return nil
	}
	scope.normalize()
	entries := make([]AccountModelProbeSnapshotEntry, 0, len(scope.Entries))
	for _, entry := range scope.Entries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		if displayModelID == "" {
			continue
		}
		targetModelID := strings.TrimSpace(entry.TargetModelID)
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		entries = append(entries, AccountModelProbeSnapshotEntry{
			DisplayModelID:    displayModelID,
			TargetModelID:     targetModelID,
			AvailabilityState: AccountModelAvailabilityUnknown,
			StaleState:        AccountModelStaleStateUnverified,
			UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
			Source:            AccountModelProbeSnapshotSourceModelScopePreview,
		})
	}
	if len(entries) == 0 {
		return nil
	}
	return buildAccountModelProbeSnapshotExtraWithEntries(
		collectLegacyProbeSnapshotModelsFromEntries(entries),
		entries,
		time.Now().UTC(),
		AccountModelProbeSnapshotSourceModelScopePreview,
		AccountModelProbeSnapshotSourceModelScopePreview,
	)
}

func BuildAccountModelAvailabilitySnapshotExtra(
	projection *AccountModelProjection,
	detectedModels []string,
	updatedAt time.Time,
	source string,
	probeSource string,
) map[string]any {
	if projection == nil || len(projection.Entries) == 0 {
		return BuildAccountModelProbeSnapshotExtra(detectedModels, updatedAt, source, probeSource)
	}

	detectedSet := buildAccountModelProbeDetectedSet(detectedModels)
	entries := make([]AccountModelProbeSnapshotEntry, 0, len(projection.Entries))
	staleState := buildAccountModelProbeSnapshotStaleState(updatedAt, source, probeSource)
	entrySource := firstNonEmptyTrimmed(probeSource, source)
	entryUpdatedAt := formatAccountModelProbeSnapshotUpdatedAt(updatedAt)
	for _, projectionEntry := range projection.Entries {
		displayModelID := strings.TrimSpace(projectionEntry.DisplayModelID)
		targetModelID := strings.TrimSpace(firstNonEmptyTrimmed(projectionEntry.TargetModelID, projectionEntry.RouteModelID, projectionEntry.CanonicalID, displayModelID))
		if displayModelID == "" && targetModelID == "" {
			continue
		}
		availabilityState := AccountModelAvailabilityUnavailable
		if accountModelProbeDetectedSetContains(detectedSet,
			displayModelID,
			targetModelID,
			projectionEntry.RouteModelID,
			projectionEntry.CanonicalID,
		) {
			availabilityState = AccountModelAvailabilityVerified
		}
		entries = append(entries, AccountModelProbeSnapshotEntry{
			DisplayModelID:    displayModelID,
			TargetModelID:     targetModelID,
			AvailabilityState: availabilityState,
			StaleState:        staleState,
			UpdatedAt:         entryUpdatedAt,
			Source:            entrySource,
		})
	}
	if len(entries) == 0 {
		return nil
	}
	return buildAccountModelProbeSnapshotExtraWithEntries(
		normalizeAccountModelProbeSnapshotModels(detectedModels),
		entries,
		updatedAt,
		source,
		probeSource,
	)
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
	entries := normalizeAccountModelProbeSnapshotEntriesAny(snapshotMap["entries"])
	if len(entries) == 0 && len(models) > 0 {
		entries = buildLegacyAccountModelProbeSnapshotEntries(
			models,
			stringValueFromAny(snapshotMap["updated_at"]),
			stringValueFromAny(snapshotMap["source"]),
			stringValueFromAny(snapshotMap["probe_source"]),
		)
	}
	if len(entries) == 0 && len(models) == 0 {
		return nil, false
	}
	if len(models) == 0 && len(entries) > 0 {
		models = collectLegacyProbeSnapshotModelsFromEntries(entries)
	}

	snapshot := &AccountModelProbeSnapshot{
		Models:      models,
		Entries:     entries,
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
	modelIDs := snapshotModelIDsForAvailableTestModels(snapshot)
	if snapshot == nil || len(modelIDs) == 0 {
		return []AvailableTestModel{}
	}
	sourceProtocols := protocolGatewayTestSourceProtocols(account)
	if len(sourceProtocols) == 0 {
		return buildAvailableTestModelsFromProbeSnapshotSource(ctx, registry, modelIDs, "")
	}

	groups := make([][]AvailableTestModel, 0, len(sourceProtocols))
	for _, sourceProtocol := range sourceProtocols {
		groups = append(groups, buildAvailableTestModelsFromProbeSnapshotSource(ctx, registry, modelIDs, sourceProtocol))
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

func buildAccountModelProbeSnapshotExtraWithEntries(
	models []string,
	entries []AccountModelProbeSnapshotEntry,
	updatedAt time.Time,
	source string,
	probeSource string,
) map[string]any {
	entries = normalizeAccountModelProbeSnapshotEntries(entries)
	if len(entries) == 0 && len(models) == 0 {
		return nil
	}

	snapshot := map[string]any{}
	if len(models) > 0 {
		snapshot["models"] = models
	}
	if len(entries) > 0 {
		snapshot["entries"] = accountModelProbeSnapshotEntriesToAny(entries)
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
	if len(snapshot) == 0 {
		return nil
	}
	return map[string]any{
		accountModelProbeSnapshotExtraKey: snapshot,
	}
}

func accountModelProbeSnapshotEntriesToAny(entries []AccountModelProbeSnapshotEntry) []map[string]any {
	if len(entries) == 0 {
		return nil
	}
	items := make([]map[string]any, 0, len(entries))
	for _, entry := range normalizeAccountModelProbeSnapshotEntries(entries) {
		item := map[string]any{}
		if displayModelID := strings.TrimSpace(entry.DisplayModelID); displayModelID != "" {
			item["display_model_id"] = displayModelID
		}
		if targetModelID := strings.TrimSpace(entry.TargetModelID); targetModelID != "" {
			item["target_model_id"] = targetModelID
		}
		if availabilityState := normalizeAccountModelAvailabilityState(entry.AvailabilityState); availabilityState != "" {
			item["availability_state"] = availabilityState
		}
		if staleState := normalizeAccountModelStaleState(entry.StaleState); staleState != "" {
			item["stale_state"] = staleState
		}
		if updatedAt := strings.TrimSpace(entry.UpdatedAt); updatedAt != "" {
			item["updated_at"] = updatedAt
		}
		if source := strings.TrimSpace(entry.Source); source != "" {
			item["source"] = source
		}
		if len(item) > 0 {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func normalizeAccountModelProbeSnapshotEntriesAny(raw any) []AccountModelProbeSnapshotEntry {
	values, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]map[string]any); ok {
			values = make([]any, 0, len(typed))
			for _, item := range typed {
				values = append(values, item)
			}
		} else {
			return nil
		}
	}

	entries := make([]AccountModelProbeSnapshotEntry, 0, len(values))
	for _, item := range values {
		entryMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		displayModelID := strings.TrimSpace(stringValueFromAny(entryMap["display_model_id"]))
		targetModelID := strings.TrimSpace(stringValueFromAny(entryMap["target_model_id"]))
		if displayModelID == "" && targetModelID == "" {
			continue
		}
		if displayModelID == "" {
			displayModelID = targetModelID
		}
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		entries = append(entries, AccountModelProbeSnapshotEntry{
			DisplayModelID:    displayModelID,
			TargetModelID:     targetModelID,
			AvailabilityState: normalizeAccountModelAvailabilityState(stringValueFromAny(entryMap["availability_state"])),
			StaleState:        normalizeAccountModelStaleState(stringValueFromAny(entryMap["stale_state"])),
			UpdatedAt:         strings.TrimSpace(stringValueFromAny(entryMap["updated_at"])),
			Source:            strings.TrimSpace(stringValueFromAny(entryMap["source"])),
		})
	}
	return normalizeAccountModelProbeSnapshotEntries(entries)
}

func normalizeAccountModelProbeSnapshotEntries(entries []AccountModelProbeSnapshotEntry) []AccountModelProbeSnapshotEntry {
	if len(entries) == 0 {
		return nil
	}
	normalized := make([]AccountModelProbeSnapshotEntry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		targetModelID := strings.TrimSpace(entry.TargetModelID)
		if displayModelID == "" && targetModelID == "" {
			continue
		}
		if displayModelID == "" {
			displayModelID = targetModelID
		}
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		key := displayModelID + "\x00" + targetModelID
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, AccountModelProbeSnapshotEntry{
			DisplayModelID:    displayModelID,
			TargetModelID:     targetModelID,
			AvailabilityState: normalizeAccountModelAvailabilityState(entry.AvailabilityState),
			StaleState:        normalizeAccountModelStaleState(entry.StaleState),
			UpdatedAt:         strings.TrimSpace(entry.UpdatedAt),
			Source:            strings.TrimSpace(entry.Source),
		})
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func buildLegacyAccountModelProbeSnapshotEntries(
	models []string,
	updatedAt string,
	source string,
	probeSource string,
) []AccountModelProbeSnapshotEntry {
	if len(models) == 0 {
		return nil
	}
	staleState := buildAccountModelProbeSnapshotStaleStateFromRaw(updatedAt, source, probeSource)
	entrySource := firstNonEmptyTrimmed(probeSource, source)
	entries := make([]AccountModelProbeSnapshotEntry, 0, len(models))
	for _, modelID := range models {
		entries = append(entries, AccountModelProbeSnapshotEntry{
			DisplayModelID:    modelID,
			TargetModelID:     modelID,
			AvailabilityState: AccountModelAvailabilityVerified,
			StaleState:        staleState,
			UpdatedAt:         strings.TrimSpace(updatedAt),
			Source:            entrySource,
		})
	}
	return normalizeAccountModelProbeSnapshotEntries(entries)
}

func collectLegacyProbeSnapshotModelsFromEntries(entries []AccountModelProbeSnapshotEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	models := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		modelID := NormalizeModelCatalogModelID(strings.TrimSpace(firstNonEmptyTrimmed(entry.TargetModelID, entry.DisplayModelID)))
		if modelID == "" {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}
		seen[modelID] = struct{}{}
		models = append(models, modelID)
	}
	if len(models) == 0 {
		return nil
	}
	return models
}

func snapshotModelIDsForAvailableTestModels(snapshot *AccountModelProbeSnapshot) []string {
	if snapshot == nil {
		return nil
	}
	if len(snapshot.Models) > 0 {
		return snapshot.Models
	}
	return collectLegacyProbeSnapshotModelsFromEntries(snapshot.Entries)
}

func buildAccountModelProbeDetectedSet(models []string) map[string]struct{} {
	if len(models) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(models)*2)
	for _, modelID := range normalizeAccountModelProbeSnapshotModels(models) {
		if modelID == "" {
			continue
		}
		set[modelID] = struct{}{}
		if normalized := normalizeRegistryID(modelID); normalized != "" {
			set[normalized] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func accountModelProbeDetectedSetContains(detectedSet map[string]struct{}, values ...string) bool {
	if len(detectedSet) == 0 {
		return false
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := detectedSet[value]; ok {
			return true
		}
		if normalized := normalizeRegistryID(value); normalized != "" {
			if _, ok := detectedSet[normalized]; ok {
				return true
			}
		}
	}
	return false
}

func buildAccountModelProbeSnapshotStaleState(updatedAt time.Time, source string, probeSource string) string {
	return buildAccountModelProbeSnapshotStaleStateFromRaw(
		formatAccountModelProbeSnapshotUpdatedAt(updatedAt),
		source,
		probeSource,
	)
}

func buildAccountModelProbeSnapshotStaleStateFromRaw(updatedAt string, source string, probeSource string) string {
	if strings.EqualFold(strings.TrimSpace(source), AccountModelProbeSnapshotSourceModelScopePreview) ||
		strings.EqualFold(strings.TrimSpace(probeSource), AccountModelProbeSnapshotSourceModelScopePreview) {
		return AccountModelStaleStateUnverified
	}
	if parsed, ok := parseSnapshotUpdatedAt(updatedAt); ok {
		if time.Since(parsed) > accountModelProjectionStaleTTL {
			return AccountModelStaleStateStale
		}
		return AccountModelStaleStateFresh
	}
	if strings.TrimSpace(updatedAt) != "" {
		return AccountModelStaleStateStale
	}
	return AccountModelStaleStateUnverified
}

func formatAccountModelProbeSnapshotUpdatedAt(updatedAt time.Time) string {
	if updatedAt.IsZero() {
		return ""
	}
	return updatedAt.UTC().Format(time.RFC3339)
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
