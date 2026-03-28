package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"go.uber.org/zap"
)

const modelRegistryAccountScanPageSize = 500

var modelRegistryAvailableBootstrapInputsV20260313 = []string{
	"claude-opus-4-6",
	"claude-sonnet-4-5",
	"claude-haiku-4-5",
	"gpt-5.4",
	"gpt-5.2",
	"gpt-5.4-pro",
	"gemini-3.1-flash-image",
	"gemini-3.1-flash-image-preview",
	"gemini-3.1-pro-preview",
	"gemini-3-pro-image",
	"gemini-2.5-flash-image-preview",
	"gemini-2.5-flash-image",
}

var modelRegistryAvailableBootstrapInputsV20260317 = []string{
	"claude-opus-4.1",
	"claude-opus-4-6",
	"claude-sonnet-4.5",
	"claude-sonnet-4-6",
	"claude-haiku-4.5",
}

var modelRegistryAvailableBootstrapInputsV20260328 = []string{
	"grok-4",
	"grok-4-0709",
	"grok-3-beta",
	"grok-3-mini-beta",
	"grok-3-fast-beta",
	"grok-2",
	"grok-2-vision",
	"grok-imagine-image",
	"grok-imagine-video",
	"grok-2-image",
}

var modelRegistryAvailableBootstrapRuntimeEntriesV20260313 = []modelregistry.ModelEntry{
	{
		ID:               "gpt-5.4-pro",
		DisplayName:      "GPT-5.4-pro",
		Provider:         "openai",
		Platforms:        []string{"openai"},
		ProtocolIDs:      []string{"gpt-5.4-pro"},
		Aliases:          []string{},
		PricingLookupIDs: []string{"gpt-5.4-pro"},
		Modalities:       []string{"text"},
		Capabilities:     []string{},
		UIPriority:       50,
		ExposedIn:        []string{"runtime", "test", "whitelist"},
	},
}

func (s *ModelRegistryService) ActivateModels(ctx context.Context, modelIDs []string) ([]modelregistry.AdminModelDetail, error) {
	changedIDs, err := s.updateAvailableModels(ctx, modelIDs, true)
	if err != nil {
		return nil, err
	}
	return s.lookupAdminDetails(ctx, changedIDs)
}

func (s *ModelRegistryService) DeactivateModels(ctx context.Context, modelIDs []string) ([]modelregistry.AdminModelDetail, error) {
	changedIDs, err := s.updateAvailableModels(ctx, modelIDs, false)
	if err != nil {
		return nil, err
	}
	return s.lookupAdminDetails(ctx, changedIDs)
}

func (s *ModelRegistryService) EnsureModelsAvailable(ctx context.Context, modelIDs []string) ([]string, error) {
	return s.updateAvailableModels(ctx, modelIDs, true)
}

func (s *ModelRegistryService) IsModelAvailable(ctx context.Context, modelID string) bool {
	availableSet, err := s.loadAvailableModelSet(ctx)
	if err != nil {
		return false
	}
	normalized, _ := s.resolveCanonicalModelForAvailability(ctx, modelID)
	if normalized == "" {
		normalized = normalizeRegistryID(modelID)
	}
	_, ok := availableSet[normalized]
	return ok
}

func (s *ModelRegistryService) pricingEntries(ctx context.Context) ([]modelregistry.ModelEntry, error) {
	details, err := s.pricingDetails(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]modelregistry.ModelEntry, 0, len(details))
	for _, detail := range details {
		entries = append(entries, detail.ModelEntry)
	}
	return entries, nil
}

func (s *ModelRegistryService) pricingDetails(ctx context.Context) ([]modelregistry.AdminModelDetail, error) {
	details, err := s.adminDetails(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]modelregistry.AdminModelDetail, 0, len(details))
	for _, detail := range details {
		if detail.Tombstoned {
			continue
		}
		filtered = append(filtered, detail)
	}
	return filtered, nil
}

func (s *ModelRegistryService) loadAvailableModelSet(ctx context.Context) (map[string]struct{}, error) {
	if err := s.ensureAvailableModelsInitialized(ctx); err != nil {
		return nil, err
	}
	return s.loadStringSet(ctx, SettingKeyModelRegistryAvailableModels)
}

func (s *ModelRegistryService) ensureAvailableModelsInitialized(ctx context.Context) error {
	if s.settingRepo == nil {
		return nil
	}
	raw, _ := s.settingRepo.GetValue(ctx, SettingKeyModelRegistryAvailableModels)
	if strings.TrimSpace(raw) == "" {
		if err := s.migrateAvailableModels(ctx); err != nil {
			return err
		}
	}
	if err := s.ensureAvailableModelsBootstrapV20260313(ctx); err != nil {
		return err
	}
	if err := s.ensureAvailableModelsBootstrapV20260317(ctx); err != nil {
		return err
	}
	return s.ensureAvailableModelsBootstrapV20260328(ctx)
}

func (s *ModelRegistryService) migrateAvailableModels(ctx context.Context) error {
	entries, _, _, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return err
	}
	availableSet := make(map[string]struct{}, len(entries))
	for id := range entries {
		if _, tombstoned := tombstones[id]; tombstoned {
			continue
		}
		availableSet[id] = struct{}{}
	}
	importedIDs, err := s.collectLegacyAccountModelIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range importedIDs {
		if id = normalizeRegistryID(id); id != "" {
			availableSet[id] = struct{}{}
		}
	}
	return s.persistStringSet(ctx, SettingKeyModelRegistryAvailableModels, availableSet)
}

func (s *ModelRegistryService) ensureAvailableModelsBootstrapV20260313(ctx context.Context) error {
	return s.ensureAvailableModelsBootstrap(ctx, "20260313", SettingKeyModelRegistryAvailableModelsBootstrapV20260313, modelRegistryAvailableBootstrapInputsV20260313)
}

func (s *ModelRegistryService) ensureAvailableModelsBootstrapV20260317(ctx context.Context) error {
	return s.ensureAvailableModelsBootstrap(ctx, "20260317", SettingKeyModelRegistryAvailableModelsBootstrapV20260317, modelRegistryAvailableBootstrapInputsV20260317)
}

func (s *ModelRegistryService) ensureAvailableModelsBootstrapV20260328(ctx context.Context) error {
	return s.ensureAvailableModelsBootstrap(ctx, "20260328", SettingKeyModelRegistryAvailableModelsBootstrapV20260328, modelRegistryAvailableBootstrapInputsV20260328)
}

func (s *ModelRegistryService) ensureAvailableModelsBootstrap(ctx context.Context, version string, markerKey string, inputs []string) error {
	if s.settingRepo == nil {
		return nil
	}
	raw, err := s.settingRepo.GetValue(ctx, markerKey)
	if err == nil && strings.TrimSpace(raw) != "" {
		return nil
	}
	if err := s.ensureBootstrapRuntimeEntriesV20260313(ctx); err != nil {
		return err
	}
	availableSet, err := s.loadStringSet(ctx, SettingKeyModelRegistryAvailableModels)
	if err != nil {
		return err
	}
	resolvedIDs, skippedInputs, err := s.resolveAvailableBootstrapModelIDs(ctx, inputs)
	if err != nil {
		return err
	}
	addedIDs := make([]string, 0, len(resolvedIDs))
	for _, modelID := range resolvedIDs {
		if _, exists := availableSet[modelID]; exists {
			continue
		}
		availableSet[modelID] = struct{}{}
		addedIDs = append(addedIDs, modelID)
	}
	if len(addedIDs) > 0 {
		if err := s.persistStringSet(ctx, SettingKeyModelRegistryAvailableModels, availableSet); err != nil {
			return err
		}
	}
	if err := s.settingRepo.Set(ctx, markerKey, "true"); err != nil {
		return err
	}
	log := logger.FromContext(ctx)
	if len(skippedInputs) > 0 {
		log.Warn("model registry: skipped default available bootstrap inputs",
			zap.Strings("model_inputs", skippedInputs),
		)
	}
	log.Info("model registry: applied default available model bootstrap",
		zap.String("version", version),
		zap.Int("added_count", len(addedIDs)),
		zap.Strings("added_models", addedIDs),
	)
	return nil
}

func (s *ModelRegistryService) ensureBootstrapRuntimeEntriesV20260313(ctx context.Context) error {
	entries, _, _, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return err
	}
	runtimeEntries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return err
	}
	runtimeIndex := make(map[string]int, len(runtimeEntries))
	for index, entry := range runtimeEntries {
		runtimeIndex[entry.ID] = index
	}
	addedIDs := make([]string, 0, len(modelRegistryAvailableBootstrapRuntimeEntriesV20260313))
	for _, rawEntry := range modelRegistryAvailableBootstrapRuntimeEntriesV20260313 {
		entry, err := normalizePersistedEntry(rawEntry)
		if err != nil {
			return err
		}
		if _, tombstoned := tombstones[entry.ID]; tombstoned {
			continue
		}
		if _, exists := entries[entry.ID]; exists {
			continue
		}
		if existingIndex, exists := runtimeIndex[entry.ID]; exists {
			runtimeEntries[existingIndex] = entry
		} else {
			runtimeIndex[entry.ID] = len(runtimeEntries)
			runtimeEntries = append(runtimeEntries, entry)
		}
		entries[entry.ID] = entry
		addedIDs = append(addedIDs, entry.ID)
	}
	if len(addedIDs) == 0 {
		return nil
	}
	if err := s.persistRuntimeEntries(ctx, runtimeEntries); err != nil {
		return err
	}
	for _, modelID := range addedIDs {
		if err := s.clearStates(ctx, modelID); err != nil {
			return err
		}
	}
	logger.FromContext(ctx).Info("model registry: added default bootstrap runtime entries",
		zap.String("version", "20260313"),
		zap.Strings("model_ids", addedIDs),
	)
	return nil
}

func (s *ModelRegistryService) resolveAvailableBootstrapModelIDs(ctx context.Context, inputs []string) ([]string, []string, error) {
	resolvedSet := make(map[string]struct{}, len(inputs))
	skippedInputs := make([]string, 0)
	for _, modelID := range inputs {
		canonicalID, err := s.resolveCanonicalModelForAvailability(ctx, modelID)
		if err != nil {
			return nil, nil, err
		}
		if canonicalID == "" {
			skippedInputs = append(skippedInputs, modelID)
			continue
		}
		resolvedSet[canonicalID] = struct{}{}
	}
	resolvedIDs := make([]string, 0, len(resolvedSet))
	for modelID := range resolvedSet {
		resolvedIDs = append(resolvedIDs, modelID)
	}
	sort.Strings(resolvedIDs)
	sort.Strings(skippedInputs)
	return resolvedIDs, skippedInputs, nil
}

func (s *ModelRegistryService) collectLegacyAccountModelIDs(ctx context.Context) ([]string, error) {
	if s.accountRepo == nil {
		return nil, nil
	}
	accounts, err := s.listAllAccounts(ctx)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, nil
	}
	runtimeEntries, err := s.loadRuntimeEntries(ctx)
	if err != nil {
		return nil, err
	}
	mergedEntries, _, _, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return nil, err
	}
	mergedSlice := make([]modelregistry.ModelEntry, 0, len(mergedEntries))
	for id, entry := range mergedEntries {
		if _, tombstoned := tombstones[id]; tombstoned {
			continue
		}
		mergedSlice = append(mergedSlice, entry)
	}
	index := modelregistry.BuildIndex(mergedSlice)
	runtimeIndex := make(map[string]int, len(runtimeEntries))
	for idx, entry := range runtimeEntries {
		runtimeIndex[entry.ID] = idx
	}

	collected := make(map[string]struct{})
	changed := false
	for _, account := range accounts {
		for requestedModel, targetModel := range account.GetModelMapping() {
			canonicalID := firstResolvedCanonicalID(index, targetModel, requestedModel)
			if canonicalID != "" {
				collected[canonicalID] = struct{}{}
				continue
			}
			entry, buildErr := s.buildLegacyMappedRuntimeEntry(requestedModel, targetModel, account.Platform)
			if buildErr != nil {
				continue
			}
			if _, tombstoned := tombstones[entry.ID]; tombstoned {
				continue
			}
			if existingIdx, ok := runtimeIndex[entry.ID]; ok {
				runtimeEntries[existingIdx] = mergeLegacyMappedRuntimeEntry(runtimeEntries[existingIdx], entry)
			} else {
				runtimeIndex[entry.ID] = len(runtimeEntries)
				runtimeEntries = append(runtimeEntries, entry)
			}
			mergedEntries[entry.ID] = entry
			mergedSlice = append(mergedSlice, entry)
			index = modelregistry.BuildIndex(mergedSlice)
			collected[entry.ID] = struct{}{}
			changed = true
		}
	}
	if changed {
		if err := s.persistRuntimeEntries(ctx, runtimeEntries); err != nil {
			return nil, err
		}
	}
	ids := make([]string, 0, len(collected))
	for id := range collected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *ModelRegistryService) updateAvailableModels(ctx context.Context, modelIDs []string, activate bool) ([]string, error) {
	availableSet, err := s.loadAvailableModelSet(ctx)
	if err != nil {
		return nil, err
	}
	changedIDs := make([]string, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		canonicalID, resolveErr := s.resolveCanonicalModelForAvailability(ctx, modelID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if canonicalID == "" {
			continue
		}
		if activate {
			if _, exists := availableSet[canonicalID]; exists {
				continue
			}
			availableSet[canonicalID] = struct{}{}
			changedIDs = append(changedIDs, canonicalID)
			continue
		}
		if _, exists := availableSet[canonicalID]; !exists {
			continue
		}
		delete(availableSet, canonicalID)
		changedIDs = append(changedIDs, canonicalID)
	}
	if len(changedIDs) == 0 {
		return []string{}, nil
	}
	if err := s.persistStringSet(ctx, SettingKeyModelRegistryAvailableModels, availableSet); err != nil {
		return nil, err
	}
	sort.Strings(changedIDs)
	return changedIDs, nil
}

func (s *ModelRegistryService) resolveCanonicalModelForAvailability(ctx context.Context, modelID string) (string, error) {
	modelID = normalizeRegistryID(modelID)
	if modelID == "" {
		return "", nil
	}
	entries, _, _, tombstones, err := s.mergedEntries(ctx)
	if err != nil {
		return "", err
	}
	if _, tombstoned := tombstones[modelID]; tombstoned {
		return "", nil
	}
	allEntries := make([]modelregistry.ModelEntry, 0, len(entries))
	for id, entry := range entries {
		if _, tombstoned := tombstones[id]; tombstoned {
			continue
		}
		allEntries = append(allEntries, entry)
	}
	index := modelregistry.BuildIndex(allEntries)
	if canonicalID, ok := index.ResolveCanonicalID(modelID); ok && canonicalID != "" {
		return canonicalID, nil
	}
	if _, exists := entries[modelID]; exists {
		return modelID, nil
	}
	return "", nil
}

func (s *ModelRegistryService) lookupAdminDetails(ctx context.Context, modelIDs []string) ([]modelregistry.AdminModelDetail, error) {
	if len(modelIDs) == 0 {
		return []modelregistry.AdminModelDetail{}, nil
	}
	details, err := s.adminDetails(ctx)
	if err != nil {
		return nil, err
	}
	index := make(map[string]modelregistry.AdminModelDetail, len(details))
	for _, detail := range details {
		index[detail.ID] = detail
	}
	result := make([]modelregistry.AdminModelDetail, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		detail, ok := index[normalizeRegistryID(modelID)]
		if !ok {
			continue
		}
		result = append(result, detail)
	}
	return result, nil
}

func (s *ModelRegistryService) listAllAccounts(ctx context.Context) ([]Account, error) {
	if s.accountRepo == nil {
		return nil, nil
	}
	page := 1
	items := make([]Account, 0)
	for {
		accounts, result, err := s.accountRepo.List(ctx, pagination.PaginationParams{
			Page:     page,
			PageSize: modelRegistryAccountScanPageSize,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, accounts...)
		if result == nil || page >= result.Pages || len(accounts) == 0 {
			break
		}
		page++
	}
	return items, nil
}

func firstResolvedCanonicalID(index *modelregistry.Index, values ...string) string {
	for _, value := range values {
		if canonicalID, ok := index.ResolveCanonicalID(value); ok && canonicalID != "" {
			return canonicalID
		}
	}
	return ""
}

func (s *ModelRegistryService) buildLegacyMappedRuntimeEntry(requestedModel string, targetModel string, sourcePlatform string) (modelregistry.ModelEntry, error) {
	requestedID := normalizeRegistryID(requestedModel)
	targetID := normalizeRegistryID(targetModel)
	canonicalID := requestedID
	if canonicalID == "" {
		canonicalID = targetID
	}
	provider := inferModelProvider(canonicalID)
	platforms, err := discoveredPlatforms(provider, sourcePlatform)
	if err != nil {
		normalizedPlatform := normalizeRegistryPlatform(sourcePlatform)
		if normalizedPlatform == "" {
			return modelregistry.ModelEntry{}, err
		}
		platforms = []string{normalizedPlatform}
	}
	preferred := map[string]string{}
	if targetID != "" {
		preferred["default"] = targetID
		preferred[normalizeRegistryRouteKey(sourcePlatform)] = targetID
	}
	return normalizePersistedEntry(modelregistry.ModelEntry{
		ID:                   canonicalID,
		DisplayName:          FormatModelCatalogDisplayName(canonicalID),
		Provider:             providerOrPlatform(provider, sourcePlatform),
		Platforms:            platforms,
		ProtocolIDs:          compactRegistryStrings(targetID, requestedID, canonicalID),
		Aliases:              compactRegistryStrings(requestedID),
		PricingLookupIDs:     compactRegistryStrings(targetID, canonicalID),
		PreferredProtocolIDs: preferred,
		Modalities:           defaultModalitiesForMode(inferModelMode(canonicalID, "")),
		Capabilities:         defaultCapabilitiesForMode(inferModelMode(canonicalID, "")),
		UIPriority:           5000,
		ExposedIn:            []string{"runtime", "test", "whitelist", "use_key"},
	})
}

func mergeLegacyMappedRuntimeEntry(current modelregistry.ModelEntry, discovered modelregistry.ModelEntry) modelregistry.ModelEntry {
	merged := current
	merged.Platforms = mergeRegistryStrings(current.Platforms, discovered.Platforms...)
	merged.ProtocolIDs = mergeRegistryStrings(current.ProtocolIDs, discovered.ProtocolIDs...)
	merged.Aliases = mergeRegistryStrings(current.Aliases, discovered.Aliases...)
	merged.PricingLookupIDs = mergeRegistryStrings(current.PricingLookupIDs, discovered.PricingLookupIDs...)
	merged.ExposedIn = mergeRegistryStrings(current.ExposedIn, discovered.ExposedIn...)
	if merged.DisplayName == "" {
		merged.DisplayName = discovered.DisplayName
	}
	if merged.Provider == "" {
		merged.Provider = discovered.Provider
	}
	if merged.PreferredProtocolIDs == nil {
		merged.PreferredProtocolIDs = map[string]string{}
	}
	for key, value := range discovered.PreferredProtocolIDs {
		if strings.TrimSpace(merged.PreferredProtocolIDs[key]) == "" {
			merged.PreferredProtocolIDs[key] = value
		}
	}
	normalized, err := normalizePersistedEntry(merged)
	if err != nil {
		return current
	}
	return normalized
}
