package service

import (
	"context"
	"sort"
	"strings"
)

type APIKeyPublicModelEntry struct {
	PublicID          string
	AliasID           string
	SourceID          string
	DisplayName       string
	Platform          string
	AvailabilityState string
	StaleState        string
	LifecycleStatus   string
	LifecycleInferred bool
}

func (s *GatewayService) GetAPIKeyPublicModels(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
) ([]APIKeyPublicModelEntry, error) {
	if s == nil || s.accountRepo == nil || apiKey == nil {
		return nil, nil
	}
	if publishedEntries, ok, err := s.apiKeyPublishedPublicCatalogModels(ctx, apiKey, platform); err != nil {
		return nil, err
	} else if ok {
		return publishedEntries, nil
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
		projectionPlatform := apiKeyPublicProjectionPlatform(bindingPlatform, normalizedPlatform)
		if normalizedPlatform != "" && projectionPlatform == "" {
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
			if !MatchesGroupPlatform(account, projectionPlatform) {
				continue
			}
			accountForProjection := ResolveProtocolGatewayInboundAccount(account, projectionPlatform)
			entries, err := s.publicModelEntriesForAccount(
				ctx,
				accountForProjection,
				mode,
				projectionPlatform,
				binding.ModelPatterns,
				accountForProjection.GetModelMapping(),
			)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			entries = s.filterPublicEntriesByActiveChannel(ctx, binding.GroupID, projectionPlatform, entries)
			entries = filterOpenAIAPIKeyPublicEntriesForRuntimeQuota(accountForProjection, entries)
			entries = filterAPIKeyPublicEntriesByGroupVisibleModels(entries, binding.Group)
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
	if apiKey.IsImageOnly() {
		filtered := make([]APIKeyPublicModelEntry, 0, len(entries))
		for _, entry := range entries {
			// image-only key: only expose native image generation models (capability=image_generation).
			native, _ := s.resolvePublicImageCapability(ctx, &entry)
			if native {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PublicID < entries[j].PublicID
	})
	return entries, nil
}
