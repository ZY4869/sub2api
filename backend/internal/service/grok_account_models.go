package service

import (
	"sort"
	"strings"
)

func GrokCapabilityModelIDsForAccount(account *Account) []string {
	tier := GrokTierBasic
	if account != nil {
		tier = ResolveGrokTier(account.Extra)
	}
	return DefaultGrokModelIDsForTier(tier)
}

func GrokVisibleModelIDsForAccount(account *Account) []string {
	capabilityModels := GrokCapabilityModelIDsForAccount(account)
	if account == nil || !account.IsGrokSSO() {
		return capabilityModels
	}

	mapping := account.GetModelMapping()
	if len(mapping) == 0 {
		return capabilityModels
	}

	allowedUpstream := make(map[string]struct{}, len(capabilityModels))
	for _, modelID := range capabilityModels {
		allowedUpstream[normalizeRegistryID(modelID)] = struct{}{}
	}

	result := make([]string, 0, len(mapping))
	seen := make(map[string]struct{}, len(mapping))
	appendModel := func(requestedModel string) {
		normalized := normalizeRegistryID(requestedModel)
		if normalized == "" {
			return
		}
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		result = append(result, requestedModel)
	}

	for _, modelID := range capabilityModels {
		mappedModel, ok := mapping[modelID]
		if !ok {
			continue
		}
		if mappedModel = strings.TrimSpace(mappedModel); mappedModel == "" {
			mappedModel = modelID
		}
		if _, ok := allowedUpstream[normalizeRegistryID(mappedModel)]; ok {
			appendModel(modelID)
		}
	}

	extras := make([]string, 0, len(mapping))
	for requestedModel, mappedModel := range mapping {
		requestedModel = strings.TrimSpace(requestedModel)
		if requestedModel == "" {
			continue
		}
		if mappedModel = strings.TrimSpace(mappedModel); mappedModel == "" {
			mappedModel = requestedModel
		}
		if _, ok := allowedUpstream[normalizeRegistryID(mappedModel)]; !ok {
			continue
		}
		if _, ok := seen[normalizeRegistryID(requestedModel)]; ok {
			continue
		}
		extras = append(extras, requestedModel)
	}
	sort.Strings(extras)
	for _, requestedModel := range extras {
		appendModel(requestedModel)
	}

	return result
}
