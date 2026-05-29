package service

import (
	"context"
	"strings"
)

func (s *GatewayService) FindAPIKeyPublicModel(
	ctx context.Context,
	apiKey *APIKey,
	platform, modelID string,
) (*APIKeyPublicModelEntry, bool, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false, nil
	}
	if entry, ok, active, err := s.findPublishedPublicCatalogModel(ctx, apiKey, platform, modelID); err != nil || ok || active {
		return entry, ok, err
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
	if entry, ok, active, err := s.findPublishedPublicCatalogModel(ctx, apiKey, platform, modelID); err == nil && ok && entry != nil {
		if sourceID := strings.TrimSpace(entry.SourceID); sourceID != "" {
			return sourceID
		}
		return strings.TrimSpace(entry.PublicID)
	} else if err == nil && active {
		return ""
	}
	entry, ok := s.findConfiguredAPIKeyModelByAnyID(ctx, apiKey, platform, modelID)
	if !ok || strings.TrimSpace(entry.PublicID) == "" {
		return modelID
	}
	return entry.PublicID
}

func (s *GatewayService) ResolveAPIKeyVisibleModelCandidates(ctx context.Context, apiKey *APIKey, platform, modelID string) []string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil
	}
	candidates := []string{modelID}
	if entry, ok, active, err := s.findPublishedPublicCatalogModel(ctx, apiKey, platform, modelID); err == nil && ok && entry != nil {
		candidates = appendAPIKeyPublicEntryCandidates(candidates, *entry)
	} else if err == nil && active {
		return visibleModelCandidates("", candidates...)
	}
	if entry, ok := s.findConfiguredAPIKeyModelByAnyID(ctx, apiKey, platform, modelID); ok && entry != nil {
		candidates = appendAPIKeyPublicEntryCandidates(candidates, *entry)
	}
	return visibleModelCandidates("", candidates...)
}

func appendAPIKeyPublicEntryCandidates(candidates []string, entry APIKeyPublicModelEntry) []string {
	return append(candidates, entry.PublicID, entry.AliasID, entry.SourceID, entry.DisplayName)
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
		projectionPlatform := apiKeyPublicProjectionPlatform(bindingPlatform, normalizedPlatform)
		if normalizedPlatform != "" && projectionPlatform == "" {
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
			if !MatchesGroupPlatform(account, projectionPlatform) {
				continue
			}
			accountForProjection := ResolveProtocolGatewayInboundAccount(account, projectionPlatform)
			entries, err := s.publicModelEntriesForAccount(ctx, accountForProjection, mode, projectionPlatform, binding.ModelPatterns, accountForProjection.GetModelMapping())
			if err != nil {
				continue
			}
			entries = filterAPIKeyPublicEntriesByGroupVisibleModels(entries, binding.Group)
			for _, entry := range entries {
				if apiKey.IsImageOnly() {
					native, _ := s.resolvePublicImageCapability(ctx, &entry)
					if !native {
						continue
					}
				}
				if apiKeyPublicEntryMatchesID(entry, modelID) {
					return &entry, true
				}
			}
		}
	}
	return nil, false
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
