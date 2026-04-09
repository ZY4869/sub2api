package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"
)

type APIKeyPublicModelEntry struct {
	PublicID    string
	AliasID     string
	SourceID    string
	DisplayName string
	Platform    string
}

type apiKeyPublicProjectionCandidate struct {
	MatchID     string
	AliasID     string
	SourceID    string
	DisplayName string
	Platform    string
}

func (s *GatewayService) GetAPIKeyPublicModels(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
) ([]APIKeyPublicModelEntry, error) {
	if s == nil || s.accountRepo == nil || apiKey == nil {
		return nil, nil
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil, nil
	}

	normalizedPlatform := strings.TrimSpace(strings.ToLower(platform))
	mode := apiKey.EffectiveModelDisplayMode()
	entriesByID := make(map[string]APIKeyPublicModelEntry)
	var firstErr error

	for _, binding := range bindings {
		if binding.Group == nil || !binding.Group.IsActive() {
			continue
		}
		bindingPlatform := strings.TrimSpace(binding.Group.Platform)
		if normalizedPlatform != "" && !strings.EqualFold(bindingPlatform, normalizedPlatform) {
			continue
		}

		queryPlatforms := QueryPlatformsForGroupPlatform(bindingPlatform, false)
		accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, binding.GroupID, queryPlatforms)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for i := range accounts {
			account := &accounts[i]
			if account == nil || !account.IsSchedulable() {
				continue
			}
			entries, err := s.publicModelEntriesForAccount(
				ctx,
				account,
				mode,
				bindingPlatform,
				binding.ModelPatterns,
				account.GetModelMapping(),
			)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			for _, entry := range entries {
				if _, exists := entriesByID[entry.PublicID]; exists {
					continue
				}
				entriesByID[entry.PublicID] = entry
			}
		}
	}

	if len(entriesByID) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, nil
	}
	entries := make([]APIKeyPublicModelEntry, 0, len(entriesByID))
	for _, entry := range entriesByID {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PublicID < entries[j].PublicID
	})
	return entries, nil
}

func (s *GatewayService) publicModelEntriesForAccount(
	ctx context.Context,
	account *Account,
	mode string,
	platform string,
	modelPatterns []string,
	mapping map[string]string,
) ([]APIKeyPublicModelEntry, error) {
	if account == nil {
		return nil, nil
	}
	if account.IsGeminiVertexSource() && strings.EqualFold(platform, PlatformGemini) {
		return s.vertexPublicModelEntries(ctx, account, mode, platform, modelPatterns, mapping)
	}

	if savedSummary := AccountSavedModelProbeSummary(account); savedSummary != nil {
		return projectProbeSummaryToPublicEntries(mode, platform, modelPatterns, mapping, savedSummary, account), nil
	}
	if s.accountModelImportService == nil {
		return nil, nil
	}

	probeSummary, err := s.accountModelImportService.ListAccountModels(ctx, account, false)
	if err != nil {
		slog.Info(
			"api_key_public_models_live_probe_degraded",
			"account_id", account.ID,
			"platform", platform,
			"error", err,
		)
		return nil, nil
	}
	if probeSummary != nil {
		s.bestEffortPersistAccountModelProbeSnapshot(
			ctx,
			account,
			probeSummary,
			platform,
			AccountModelProbeSnapshotSourcePublicModelsLive,
		)
	}
	return projectProbeSummaryToPublicEntries(mode, platform, modelPatterns, mapping, probeSummary, account), nil
}

func (s *GatewayService) bestEffortPersistAccountModelProbeSnapshot(
	ctx context.Context,
	account *Account,
	probeSummary *AccountModelProbeSummary,
	platform string,
	source string,
) {
	if s == nil || s.accountRepo == nil || account == nil || probeSummary == nil {
		return
	}
	models := normalizeAccountModelProbeSnapshotModels(probeSummary.DetectedModels)
	if len(models) == 0 {
		return
	}

	updatedAt := time.Now().UTC()
	updates := BuildAccountModelProbeSnapshotExtra(
		models,
		updatedAt,
		source,
		probeSummary.ProbeSource,
	)
	if account.IsOpenAI() {
		updates = MergeStringAnyMap(
			BuildOpenAIKnownModelsExtra(models, updatedAt, source),
			updates,
		)
	}
	if len(updates) == 0 {
		return
	}
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
		slog.Warn(
			"api_key_public_models_snapshot_backfill_failed",
			"account_id", account.ID,
			"platform", platform,
			"source", source,
			"error", err,
		)
		return
	}
	mergeAccountExtra(account, updates)
}

func projectProbeSummaryToPublicEntries(
	mode string,
	platform string,
	modelPatterns []string,
	mapping map[string]string,
	probeSummary *AccountModelProbeSummary,
	account *Account,
) []APIKeyPublicModelEntry {
	if probeSummary == nil {
		return nil
	}

	detectedSet := make(map[string]AccountModelProbeModel, len(probeSummary.Models))
	for _, detail := range probeSummary.Models {
		sourceID := normalizeRegistryID(detail.ID)
		if sourceID == "" {
			continue
		}
		detail.ID = sourceID
		if strings.TrimSpace(detail.DisplayName) == "" {
			detail.DisplayName = FormatModelCatalogDisplayName(sourceID)
		}
		detectedSet[sourceID] = detail
	}
	for _, modelID := range probeSummary.DetectedModels {
		sourceID := normalizeRegistryID(modelID)
		if sourceID == "" {
			continue
		}
		if _, exists := detectedSet[sourceID]; exists {
			continue
		}
		detectedSet[sourceID] = applyAccountModelProbeProvider(AccountModelProbeModel{
			ID:          sourceID,
			DisplayName: FormatModelCatalogDisplayName(sourceID),
		}, platform)
	}

	candidates := make([]apiKeyPublicProjectionCandidate, 0, len(detectedSet))
	if len(mapping) == 0 {
		if account != nil && account.IsGrokAPIKey() && strings.EqualFold(platform, PlatformGrok) {
			for sourceID, detail := range detectedSet {
				publicID := grokPublicModelForDetectedSource(sourceID)
				candidate, ok := buildAPIKeyPublicProjectionCandidate(mode, publicID, sourceID, platform)
				if !ok {
					continue
				}
				candidate.DisplayName = strings.TrimSpace(detail.DisplayName)
				candidates = append(candidates, candidate)
			}
		} else {
			for sourceID, detail := range detectedSet {
				candidate, ok := buildAPIKeyPublicProjectionCandidate(mode, sourceID, sourceID, platform)
				if !ok {
					continue
				}
				candidate.DisplayName = strings.TrimSpace(detail.DisplayName)
				candidates = append(candidates, candidate)
			}
		}
	} else {
		for alias, source := range mapping {
			candidate, ok := buildAPIKeyPublicProjectionCandidate(mode, alias, source, platform)
			if !ok {
				continue
			}
			candidates = append(candidates, candidate)
		}
	}

	projected := make(map[string]APIKeyPublicModelEntry)
	for _, candidate := range candidates {
		if _, matched := bindingMatchesModel(modelPatterns, candidate.MatchID); !matched {
			continue
		}
		sourceID, detail, ok := resolveAPIKeyProjectionDetectedDetail(detectedSet, candidate.SourceID)
		if !ok {
			continue
		}
		projectedSourceID := sourceID
		if !strings.EqualFold(platform, PlatformGrok) {
			if rawSourceID := normalizeRegistryID(candidate.SourceID); rawSourceID != "" {
				projectedSourceID = rawSourceID
			}
		}
		publicID := apiKeyPublicProjectionPublicID(platform, candidate, projectedSourceID)
		if publicID == "" {
			publicID = projectedSourceID
		}
		if _, exists := projected[publicID]; exists {
			continue
		}
		displayName := strings.TrimSpace(detail.DisplayName)
		if displayName == "" {
			displayName = strings.TrimSpace(candidate.DisplayName)
		}
		if displayName == "" {
			displayName = FormatModelCatalogDisplayName(projectedSourceID)
		}
		projected[publicID] = APIKeyPublicModelEntry{
			PublicID:    publicID,
			AliasID:     candidate.AliasID,
			SourceID:    projectedSourceID,
			DisplayName: displayName,
			Platform:    platform,
		}
	}

	result := make([]APIKeyPublicModelEntry, 0, len(projected))
	for _, entry := range projected {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PublicID < result[j].PublicID
	})
	return result
}

func resolveAPIKeyProjectionDetectedDetail(
	detectedSet map[string]AccountModelProbeModel,
	source string,
) (string, AccountModelProbeModel, bool) {
	candidates := []string{
		normalizeRegistryID(source),
		NormalizeModelCatalogModelID(source),
		modelCatalogDateVersionSuffixPattern.ReplaceAllString(normalizeRegistryID(source), ""),
	}
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = normalizeRegistryID(candidate)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		if detail, ok := detectedSet[candidate]; ok {
			return candidate, detail, true
		}
	}
	return "", AccountModelProbeModel{}, false
}

func (s *GatewayService) vertexPublicModelEntries(
	ctx context.Context,
	account *Account,
	mode string,
	platform string,
	modelPatterns []string,
	mapping map[string]string,
) ([]APIKeyPublicModelEntry, error) {
	if s == nil || s.vertexCatalogService == nil || account == nil {
		return nil, nil
	}
	catalog, err := s.vertexCatalogService.GetCatalog(ctx, account, false)
	if err != nil || catalog == nil {
		return nil, err
	}

	callableSet := make(map[string]VertexCatalogModel, len(catalog.CallableUnion))
	for _, model := range catalog.CallableUnion {
		callableSet[strings.TrimSpace(model.ID)] = model
	}

	candidates := make([]apiKeyPublicProjectionCandidate, 0)
	if len(mapping) == 0 {
		for _, model := range catalog.CallableUnion {
			entry, ok := buildAPIKeyPublicProjectionCandidate(mode, DefaultVertexPublicModelAlias(model.ID), model.ID, platform)
			if ok {
				entry.DisplayName = strings.TrimSpace(model.DisplayName)
				candidates = append(candidates, entry)
			}
		}
	} else {
		for alias, source := range mapping {
			entry, ok := buildAPIKeyPublicProjectionCandidate(mode, alias, source, platform)
			if ok {
				candidates = append(candidates, entry)
			}
		}
	}

	projected := make(map[string]APIKeyPublicModelEntry)
	for _, candidate := range candidates {
		if _, matched := bindingMatchesModel(modelPatterns, candidate.MatchID); !matched {
			continue
		}
		sourceID := normalizeVertexUpstreamModelID(candidate.SourceID)
		model, ok := callableSet[sourceID]
		if !ok {
			continue
		}
		if _, exists := projected[sourceID]; exists {
			continue
		}
		displayName := strings.TrimSpace(model.DisplayName)
		if displayName == "" {
			displayName = FormatModelCatalogDisplayName(sourceID)
		}
		projected[sourceID] = APIKeyPublicModelEntry{
			PublicID:    sourceID,
			AliasID:     candidate.AliasID,
			SourceID:    sourceID,
			DisplayName: displayName,
			Platform:    platform,
		}
	}

	result := make([]APIKeyPublicModelEntry, 0, len(projected))
	for _, entry := range projected {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PublicID < result[j].PublicID
	})
	return result, nil
}

func (s *GatewayService) FindAPIKeyPublicModel(
	ctx context.Context,
	apiKey *APIKey,
	platform, modelID string,
) (*APIKeyPublicModelEntry, bool, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false, nil
	}
	entries, err := s.GetAPIKeyPublicModels(ctx, apiKey, platform)
	if err != nil {
		return nil, false, err
	}
	for i := range entries {
		if entries[i].PublicID == modelID {
			entry := entries[i]
			return &entry, true, nil
		}
	}
	return nil, false, nil
}

func (s *GatewayService) ResolveAPIKeySelectionModel(ctx context.Context, apiKey *APIKey, platform, modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ""
	}
	entry, ok := s.findConfiguredAPIKeyModelByAnyID(ctx, apiKey, platform, modelID)
	if !ok || strings.TrimSpace(entry.AliasID) == "" {
		return modelID
	}
	return entry.AliasID
}

func (s *GatewayService) findConfiguredAPIKeyModelByAnyID(
	ctx context.Context,
	apiKey *APIKey,
	platform, modelID string,
) (*APIKeyPublicModelEntry, bool) {
	if s == nil || s.accountRepo == nil || apiKey == nil {
		return nil, false
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil, false
	}
	normalizedPlatform := strings.TrimSpace(strings.ToLower(platform))
	mode := apiKey.EffectiveModelDisplayMode()

	for _, binding := range bindings {
		if binding.Group == nil || !binding.Group.IsActive() {
			continue
		}
		bindingPlatform := strings.TrimSpace(binding.Group.Platform)
		if normalizedPlatform != "" && !strings.EqualFold(bindingPlatform, normalizedPlatform) {
			continue
		}
		queryPlatforms := QueryPlatformsForGroupPlatform(bindingPlatform, false)
		accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, binding.GroupID, queryPlatforms)
		if err != nil {
			continue
		}
		for i := range accounts {
			account := &accounts[i]
			if account == nil || !account.IsSchedulable() {
				continue
			}
			mapping := account.GetModelMapping()
			for alias, source := range mapping {
				candidate, ok := buildAPIKeyPublicProjectionCandidate(mode, alias, source, bindingPlatform)
				if !ok {
					continue
				}
				if _, matched := bindingMatchesModel(binding.ModelPatterns, candidate.MatchID); !matched {
					continue
				}
				if candidate.MatchID == modelID || candidate.AliasID == modelID || candidate.SourceID == modelID {
					return &APIKeyPublicModelEntry{
						PublicID:    normalizeRegistryID(candidate.SourceID),
						AliasID:     candidate.AliasID,
						SourceID:    normalizeRegistryID(candidate.SourceID),
						DisplayName: candidate.DisplayName,
						Platform:    bindingPlatform,
					}, true
				}
			}
		}
	}
	return nil, false
}

func buildAPIKeyPublicProjectionCandidate(mode, alias, source, platform string) (apiKeyPublicProjectionCandidate, bool) {
	alias = strings.TrimSpace(alias)
	source = strings.TrimSpace(source)
	if alias == "" && source == "" {
		return apiKeyPublicProjectionCandidate{}, false
	}
	if alias == "" {
		alias = source
	}
	if source == "" {
		source = alias
	}

	switch NormalizeAPIKeyModelDisplayMode(mode) {
	case APIKeyModelDisplayModeSourceOnly:
		return apiKeyPublicProjectionCandidate{
			MatchID:     source,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: source,
			Platform:    platform,
		}, true
	case APIKeyModelDisplayModeAliasAndSource:
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias + " | " + source,
			Platform:    platform,
		}, true
	default:
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias,
			Platform:    platform,
		}, true
	}
}

func apiKeyPublicProjectionPublicID(platform string, candidate apiKeyPublicProjectionCandidate, sourceID string) string {
	if !strings.EqualFold(platform, PlatformGrok) {
		return sourceID
	}
	if aliasID := normalizeRegistryID(candidate.AliasID); aliasID != "" && aliasID != sourceID {
		return aliasID
	}
	if matchID := normalizeRegistryID(candidate.MatchID); matchID != "" && matchID != sourceID {
		return matchID
	}
	if publicID := grokPublicModelForDetectedSource(sourceID); publicID != "" {
		return publicID
	}
	return sourceID
}
