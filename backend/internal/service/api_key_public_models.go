package service

import (
	"context"
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

func (s *GatewayService) GetAPIKeyPublicModels(ctx context.Context, apiKey *APIKey, platform string) []APIKeyPublicModelEntry {
	if s == nil || s.accountRepo == nil || apiKey == nil {
		return nil
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil
	}

	normalizedPlatform := strings.TrimSpace(strings.ToLower(platform))
	mode := apiKey.EffectiveModelDisplayMode()
	entriesByID := make(map[string]APIKeyPublicModelEntry)

	for _, binding := range bindings {
		if binding.Group == nil || !binding.Group.IsActive() {
			continue
		}
		bindingPlatform := strings.TrimSpace(binding.Group.Platform)
		if normalizedPlatform != "" && !strings.EqualFold(bindingPlatform, normalizedPlatform) {
			continue
		}

		accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, binding.GroupID, bindingPlatform)
		if err != nil {
			continue
		}
		for i := range accounts {
			account := &accounts[i]
			if account == nil || !account.IsSchedulable() {
				continue
			}
			mapping := account.GetModelMapping()
			if len(mapping) == 0 && account.IsGeminiVertexSource() && strings.EqualFold(bindingPlatform, PlatformGemini) {
				for _, source := range GeminiVertexCatalogModelIDs() {
					entry, ok := buildAPIKeyPublicModelEntry(mode, DefaultVertexPublicModelAlias(source), source, bindingPlatform)
					if !ok {
						continue
					}
					if _, matched := bindingMatchesModel(binding.ModelPatterns, entry.PublicID); !matched {
						continue
					}
					if _, exists := entriesByID[entry.PublicID]; exists {
						continue
					}
					entriesByID[entry.PublicID] = entry
				}
				continue
			}
			if len(mapping) == 0 {
				continue
			}
			for alias, source := range mapping {
				entry, ok := buildAPIKeyPublicModelEntry(mode, alias, source, bindingPlatform)
				if !ok {
					continue
				}
				if _, matched := bindingMatchesModel(binding.ModelPatterns, entry.PublicID); !matched {
					continue
				}
				if _, exists := entriesByID[entry.PublicID]; exists {
					continue
				}
				entriesByID[entry.PublicID] = entry
			}
		}
	}

	if len(entriesByID) == 0 {
		return nil
	}
	entries := make([]APIKeyPublicModelEntry, 0, len(entriesByID))
	for _, entry := range entriesByID {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PublicID < entries[j].PublicID
	})
	return entries
}

func (s *GatewayService) FindAPIKeyPublicModel(ctx context.Context, apiKey *APIKey, platform, modelID string) (*APIKeyPublicModelEntry, bool) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false
	}
	entries := s.GetAPIKeyPublicModels(ctx, apiKey, platform)
	for i := range entries {
		if entries[i].PublicID == modelID {
			entry := entries[i]
			return &entry, true
		}
	}
	return nil, false
}

func (s *GatewayService) ResolveAPIKeySelectionModel(ctx context.Context, apiKey *APIKey, platform, modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ""
	}
	entry, ok := s.findAPIKeyPublicModelByAnyID(ctx, apiKey, platform, modelID)
	if !ok || strings.TrimSpace(entry.AliasID) == "" {
		return modelID
	}
	return entry.AliasID
}

func (s *GatewayService) findAPIKeyPublicModelByAnyID(ctx context.Context, apiKey *APIKey, platform, modelID string) (*APIKeyPublicModelEntry, bool) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false
	}
	entries := s.GetAPIKeyPublicModels(ctx, apiKey, platform)
	for i := range entries {
		if entries[i].PublicID == modelID || entries[i].SourceID == modelID || entries[i].AliasID == modelID {
			entry := entries[i]
			return &entry, true
		}
	}
	return nil, false
}

func buildAPIKeyPublicModelEntry(mode, alias, source, platform string) (APIKeyPublicModelEntry, bool) {
	alias = strings.TrimSpace(alias)
	source = strings.TrimSpace(source)
	if alias == "" && source == "" {
		return APIKeyPublicModelEntry{}, false
	}
	if alias == "" {
		alias = source
	}
	if source == "" {
		source = alias
	}

	switch NormalizeAPIKeyModelDisplayMode(mode) {
	case APIKeyModelDisplayModeSourceOnly:
		return APIKeyPublicModelEntry{
			PublicID:    source,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: source,
			Platform:    platform,
		}, true
	case APIKeyModelDisplayModeAliasAndSource:
		return APIKeyPublicModelEntry{
			PublicID:    alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias + " | " + source,
			Platform:    platform,
		}, true
	default:
		return APIKeyPublicModelEntry{
			PublicID:    alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias,
			Platform:    platform,
		}, true
	}
}
