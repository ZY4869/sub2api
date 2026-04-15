package service

func filterChatGPTOpenAIKnownTestModelCandidates(account *Account, sourceProtocol string, candidates []testModelCandidate) []testModelCandidate {
	if !isChatGPTOpenAIOAuthAccount(account) || len(candidates) == 0 {
		return candidates
	}

	candidates = filterChatGPTOpenAIUnsupportedTestModelCandidates(candidates)
	knownModels := normalizeStringSliceAny(account.Extra["openai_known_models"], NormalizeModelCatalogModelID)
	if len(knownModels) == 0 {
		return candidates
	}

	knownSet := make(map[string]struct{}, len(knownModels))
	for _, modelID := range knownModels {
		knownSet[modelID] = struct{}{}
	}

	filtered := make([]testModelCandidate, 0, len(candidates)+len(knownModels))
	seen := make(map[string]struct{}, len(knownModels))
	for _, candidate := range candidates {
		candidateID := NormalizeModelCatalogModelID(candidate.model.ID)
		canonicalID := NormalizeModelCatalogModelID(candidate.model.CanonicalID)
		if _, ok := knownSet[candidateID]; !ok {
			if _, ok := knownSet[canonicalID]; !ok {
				continue
			}
		}
		if canonicalID == "" {
			canonicalID = candidateID
		}
		if canonicalID != "" {
			seen[canonicalID] = struct{}{}
		}
		filtered = append(filtered, candidate)
	}

	provider := inferAvailableTestModelProvider(account, sourceProtocol)
	for _, modelID := range knownModels {
		if isChatGPTOpenAIUnsupportedTestModelID(modelID) {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}
		filtered = append(filtered, testModelCandidate{
			model: applyAvailableTestModelProvider(AvailableTestModel{
				ID:             modelID,
				Type:           "model",
				DisplayName:    firstNonEmptyTestModelLabel(FormatModelCatalogDisplayName(modelID), modelID),
				SourceProtocol: normalizeTestSourceProtocol(sourceProtocol),
				Status:         "stable",
			}, provider),
			source:     "runtime",
			uiPriority: fallbackTestModelPriority(modelID),
		})
	}

	return filtered
}

func filterChatGPTOpenAIUnsupportedTestModelCandidates(candidates []testModelCandidate) []testModelCandidate {
	filtered := make([]testModelCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if isChatGPTOpenAIUnsupportedTestModelID(candidate.model.ID) ||
			isChatGPTOpenAIUnsupportedTestModelID(candidate.model.CanonicalID) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}
