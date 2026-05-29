package service

import (
	"context"
	"strings"
)

func (s *OpenAIGatewayService) ResolveAPIKeyVisibleModelCandidates(ctx context.Context, apiKey *APIKey, platform, modelID string) []string {
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

func (s *OpenAIGatewayService) findPublishedPublicCatalogModel(ctx context.Context, apiKey *APIKey, platform, modelID string) (*APIKeyPublicModelEntry, bool, bool, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false, false, nil
	}
	matches, active, err := s.apiKeyPublishedPublicCatalogVisibleMatches(ctx, apiKey, platform, modelID)
	if err != nil || !active {
		return nil, false, active, err
	}
	if len(matches) == 0 {
		return nil, false, true, nil
	}
	entry := matches[0].Entry
	return &entry, true, true, nil
}

func (s *OpenAIGatewayService) findConfiguredAPIKeyModelByAnyID(ctx context.Context, apiKey *APIKey, platform, modelID string) (*APIKeyPublicModelEntry, bool) {
	if s == nil || s.accountRepo == nil || apiKey == nil {
		return nil, false
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false
	}
	platform = firstNonEmptyTrimmed(platform, OpenAIPlatformFromContext(ctx))
	return s.findConfiguredAPIKeyModelByAnyIDForBindings(ctx, apiKey, platform, modelID)
}

func (s *OpenAIGatewayService) findConfiguredAPIKeyModelByAnyIDForBindings(ctx context.Context, apiKey *APIKey, platform, modelID string) (*APIKeyPublicModelEntry, bool) {
	for _, binding := range apiKeyBindingsForSelection(apiKey) {
		entry, ok := s.findConfiguredAPIKeyModelByAnyIDForBinding(ctx, apiKey, binding, platform, modelID)
		if ok {
			return entry, true
		}
	}
	return nil, false
}

func (s *OpenAIGatewayService) findConfiguredAPIKeyModelByAnyIDForBinding(ctx context.Context, apiKey *APIKey, binding APIKeyGroupBinding, platform, modelID string) (*APIKeyPublicModelEntry, bool) {
	if binding.Group == nil || !binding.Group.IsActive() {
		return nil, false
	}
	projectionPlatform := apiKeyPublicProjectionPlatform(binding.Group.Platform, platform)
	if strings.TrimSpace(platform) != "" && projectionPlatform == "" {
		return nil, false
	}
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, binding.GroupID, QueryPlatformsForGroupPlatform(binding.Group.Platform, false))
	if err != nil {
		return nil, false
	}
	return s.findConfiguredAPIKeyModelByAnyIDInAccounts(ctx, accounts, apiKey, binding, projectionPlatform, modelID)
}

func (s *OpenAIGatewayService) findConfiguredAPIKeyModelByAnyIDInAccounts(ctx context.Context, accounts []Account, apiKey *APIKey, binding APIKeyGroupBinding, platform, modelID string) (*APIKeyPublicModelEntry, bool) {
	for i := range accounts {
		account := ResolveProtocolGatewayInboundAccount(&accounts[i], platform)
		if account == nil || !account.IsSchedulable() || !isOpenAITextRuntimeAccount(account) {
			continue
		}
		entries := projectAccountModelProjectionToPublicEntries(platform, binding.ModelPatterns, BuildAccountModelProjection(ctx, account, s.modelRegistryService))
		entries = filterOpenAIAPIKeyPublicEntriesForRuntimeQuota(account, entries)
		entries = filterAPIKeyPublicEntriesByGroupVisibleModels(entries, binding.Group)
		for _, entry := range entries {
			if apiKey != nil && apiKey.IsImageOnly() && !IsOpenAINativeImageModelID(entry.SourceID) {
				continue
			}
			if apiKeyPublicEntryMatchesID(entry, modelID) {
				return &entry, true
			}
		}
	}
	return nil, false
}
