package service

import (
	"context"
	"sort"
	"strings"
)

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
			PublicID:          publicID,
			AliasID:           candidate.AliasID,
			SourceID:          projectedSourceID,
			DisplayName:       displayName,
			Platform:          platform,
			AvailabilityState: AccountModelAvailabilityUnknown,
			StaleState:        AccountModelStaleStateUnverified,
			LifecycleStatus:   normalizePublicModelLifecycleStatus("", displayName, publicID, projectedSourceID),
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
