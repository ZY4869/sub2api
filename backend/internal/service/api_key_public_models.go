package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
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
	ExposeAlias bool
}

const apiKeyPublicModelsSourcePolicyProjection = "policy_projection"

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
			entries = s.filterPublicEntriesByActiveChannel(ctx, binding.GroupID, bindingPlatform, entries)
			entries = filterOpenAIAPIKeyPublicEntriesForRuntimeQuota(account, entries)
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
	_ = mode
	_ = mapping

	projection := BuildAccountModelProjection(ctx, account, s.modelRegistryService)
	entries := projectAccountModelProjectionToPublicEntries(platform, modelPatterns, projection)
	recordPublicModelProjectionSource(apiKeyPublicModelsSourcePolicyProjection)
	slog.Info(
		"api_key_public_models_policy_projection",
		"account_id", account.ID,
		"platform", platform,
		"source", apiKeyPublicModelsSourcePolicyProjection,
		"policy_mode", firstNonEmptyString(projectionPolicyMode(projection), AccountModelPolicyModeWhitelist),
		"projection_source", projectionSource(projection),
		"count", len(entries),
		"alias_only_count", countAliasOnlyPublicEntries(entries),
	)
	return entries, nil
}

func projectAccountModelProjectionToPublicEntries(
	platform string,
	modelPatterns []string,
	projection *AccountModelProjection,
) []APIKeyPublicModelEntry {
	if projection == nil || len(projection.Entries) == 0 {
		return nil
	}

	projected := make(map[string]APIKeyPublicModelEntry, len(projection.Entries))
	for _, candidate := range projection.Entries {
		publicID := normalizeRegistryID(candidate.DisplayModelID)
		if publicID == "" {
			continue
		}
		targetID := normalizeRegistryID(firstNonEmptyString(candidate.TargetModelID, candidate.RouteModelID))
		if !bindingAllowsProjectedPublicModel(modelPatterns, publicID, targetID) {
			continue
		}
		if _, exists := projected[publicID]; exists {
			continue
		}
		displayName := strings.TrimSpace(candidate.DisplayModelID)
		if candidate.VisibilityMode != AccountModelVisibilityModeAlias {
			displayName = firstNonEmptyString(strings.TrimSpace(candidate.DisplayName), displayName)
		}
		projected[publicID] = APIKeyPublicModelEntry{
			PublicID:    publicID,
			AliasID:     publicID,
			SourceID:    targetID,
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

func bindingAllowsProjectedPublicModel(modelPatterns []string, publicID string, targetID string) bool {
	for _, candidate := range []string{publicID, targetID} {
		if _, matched := bindingMatchesModel(modelPatterns, candidate); matched {
			return true
		}
	}
	return false
}

func projectionPolicyMode(projection *AccountModelProjection) string {
	if projection == nil {
		return ""
	}
	return strings.TrimSpace(projection.PolicyMode)
}

func projectionSource(projection *AccountModelProjection) string {
	if projection == nil {
		return ""
	}
	return strings.TrimSpace(projection.Source)
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
		for sourceID, detail := range detectedSet {
			if account != nil && !isRequestedModelSupportedByAccount(context.Background(), nil, account, sourceID) {
				continue
			}
			candidate, ok := buildAPIKeyPublicProjectionCandidate(mode, sourceID, sourceID, platform)
			if !ok {
				continue
			}
			candidate.DisplayName = strings.TrimSpace(detail.DisplayName)
			candidates = append(candidates, candidate)
		}
	}

	projected := make(map[string]APIKeyPublicModelEntry)
	hiddenSourceIDs := make(map[string]struct{})
	for _, candidate := range candidates {
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
		if !bindingMatchesProjectionCandidate(modelPatterns, publicID, candidate) {
			continue
		}
		if !candidate.ExposeAlias && publicID == projectedSourceID {
			if _, hidden := hiddenSourceIDs[projectedSourceID]; hidden {
				continue
			}
		}
		if _, exists := projected[publicID]; exists {
			continue
		}
		displayName := strings.TrimSpace(candidate.AliasID)
		if !candidate.ExposeAlias {
			displayName = strings.TrimSpace(detail.DisplayName)
			if displayName == "" {
				displayName = strings.TrimSpace(candidate.DisplayName)
			}
			if displayName == "" {
				displayName = FormatModelCatalogDisplayName(projectedSourceID)
			}
		}
		projected[publicID] = APIKeyPublicModelEntry{
			PublicID:    publicID,
			AliasID:     candidate.AliasID,
			SourceID:    projectedSourceID,
			DisplayName: displayName,
			Platform:    platform,
		}
		if candidate.ExposeAlias && projectedSourceID != "" {
			hiddenSourceIDs[projectedSourceID] = struct{}{}
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
		if apiKeyPublicEntryMatchesID(entries[i], modelID) {
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
	if !ok || strings.TrimSpace(entry.PublicID) == "" {
		return modelID
	}
	return entry.PublicID
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
			entries, err := s.publicModelEntriesForAccount(ctx, account, mode, bindingPlatform, binding.ModelPatterns, account.GetModelMapping())
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if apiKeyPublicEntryMatchesID(entry, modelID) {
					return &entry, true
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
	explicitAlias := shouldExposePublicAlias(platform, alias, source)

	if explicitAlias {
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias,
			Platform:    platform,
			ExposeAlias: true,
		}, true
	}

	switch NormalizeAPIKeyModelDisplayMode(mode) {
	case APIKeyModelDisplayModeSourceOnly:
		return apiKeyPublicProjectionCandidate{
			MatchID:     source,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: source,
			Platform:    platform,
			ExposeAlias: false,
		}, true
	case APIKeyModelDisplayModeAliasAndSource:
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias + " | " + source,
			Platform:    platform,
			ExposeAlias: false,
		}, true
	default:
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias,
			Platform:    platform,
			ExposeAlias: false,
		}, true
	}
}

func apiKeyPublicProjectionPublicID(platform string, candidate apiKeyPublicProjectionCandidate, sourceID string) string {
	if candidate.ExposeAlias {
		if aliasID := normalizeRegistryID(candidate.AliasID); aliasID != "" {
			return aliasID
		}
		if matchID := normalizeRegistryID(candidate.MatchID); matchID != "" {
			return matchID
		}
	}
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

func shouldExposePublicAlias(platform, alias, source string) bool {
	alias = strings.TrimSpace(alias)
	source = strings.TrimSpace(source)
	if alias == "" || source == "" || alias == source {
		return false
	}
	if strings.Contains(alias, "*") {
		return false
	}
	if strings.EqualFold(platform, PlatformGemini) && alias == DefaultVertexPublicModelAlias(source) {
		return false
	}
	return true
}

func apiKeyPublicEntryMatchesID(entry APIKeyPublicModelEntry, modelID string) bool {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	for _, candidate := range []string{entry.PublicID, entry.AliasID} {
		if strings.TrimSpace(candidate) == modelID {
			return true
		}
	}
	return false
}

func bindingMatchesProjectionCandidate(
	modelPatterns []string,
	publicID string,
	candidate apiKeyPublicProjectionCandidate,
) bool {
	for _, modelID := range []string{
		publicID,
		candidate.MatchID,
		candidate.AliasID,
		candidate.SourceID,
	} {
		if _, matched := bindingMatchesModel(modelPatterns, modelID); matched {
			return true
		}
	}
	return false
}

func countAliasOnlyPublicEntries(entries []APIKeyPublicModelEntry) int {
	count := 0
	for _, entry := range entries {
		if strings.TrimSpace(entry.PublicID) == "" {
			continue
		}
		if strings.TrimSpace(entry.PublicID) != strings.TrimSpace(entry.SourceID) {
			count++
		}
	}
	return count
}

func filterOpenAIAPIKeyPublicEntriesForRuntimeQuota(account *Account, entries []APIKeyPublicModelEntry) []APIKeyPublicModelEntry {
	if len(entries) == 0 || account == nil || !account.IsOpenAI() || !isOpenAIProPlan(account) {
		return entries
	}

	filtered := make([]APIKeyPublicModelEntry, 0, len(entries))
	for _, entry := range entries {
		if shouldHideOpenAIModelForRuntimeQuota(account, apiKeyPublicModelRuntimeQuotaCandidates(account, entry)...) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
