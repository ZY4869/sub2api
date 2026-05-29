package service

import (
	"context"
	"sort"
	"strings"
)

type apiKeyPublishedPublicCatalogMatch struct {
	Entry      APIKeyPublicModelEntry
	Catalog    *PublishedPublicCatalogEntry
	Binding    APIKeyGroupBinding
	GroupID    *int64
	Account    *Account
	SourceItem PublicModelCatalogItem
}

func (s *GatewayService) apiKeyPublishedPublicCatalogModels(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
) ([]APIKeyPublicModelEntry, bool, error) {
	if s == nil || s.modelCatalogService == nil || apiKey == nil {
		return nil, false, nil
	}
	matches, active, err := s.apiKeyPublishedPublicCatalogVisibleMatches(ctx, apiKey, platform, "")
	if err != nil || !active {
		return nil, active, err
	}
	entriesByID := make(map[string]APIKeyPublicModelEntry, len(matches))
	for _, match := range matches {
		publicID := strings.TrimSpace(match.Entry.PublicID)
		if publicID == "" {
			continue
		}
		if _, exists := entriesByID[publicID]; exists {
			continue
		}
		entriesByID[publicID] = match.Entry
	}
	entries := make([]APIKeyPublicModelEntry, 0, len(entriesByID))
	for _, entry := range entriesByID {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PublicID < entries[j].PublicID
	})
	return entries, true, nil
}

func (s *GatewayService) findPublishedPublicCatalogModel(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
	modelID string,
) (*APIKeyPublicModelEntry, bool, bool, error) {
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

func (s *GatewayService) apiKeyPublishedPublicCatalogVisibleMatches(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
	modelID string,
) ([]apiKeyPublishedPublicCatalogMatch, bool, error) {
	if s == nil || s.modelCatalogService == nil || apiKey == nil {
		return nil, false, nil
	}
	published, active := s.modelCatalogService.activePublishedPublicModelCatalogSnapshot(ctx)
	if !active {
		return nil, false, nil
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil, true, nil
	}
	requestedID := NormalizeModelCatalogModelID(modelID)
	matches := make([]apiKeyPublishedPublicCatalogMatch, 0, len(published.Snapshot.Items))
	for _, item := range published.Snapshot.Items {
		if requestedID != "" && !publicModelCatalogItemMatchesPublicID(item, requestedID) {
			continue
		}
		for _, binding := range bindings {
			match, ok, err := s.publishedPublicCatalogItemForBinding(ctx, binding, platform, item)
			if err != nil {
				return nil, true, err
			}
			if !ok {
				recordPublicCatalogRouteMiss(ctx, apiKey, bindingGroupIDPtr(binding), firstNonEmptyTrimmed(item.PublicModelID, item.Model), platform)
				continue
			}
			if apiKey.IsImageOnly() {
				native, _ := s.resolvePublicImageCapability(ctx, &match.Entry)
				if !native && strings.TrimSpace(item.Mode) != "image" {
					continue
				}
			}
			matches = append(matches, match)
			break
		}
	}
	return matches, true, nil
}

func (s *OpenAIGatewayService) apiKeyPublishedPublicCatalogVisibleMatches(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
	modelID string,
) ([]apiKeyPublishedPublicCatalogMatch, bool, error) {
	if s == nil || s.modelCatalogService == nil || apiKey == nil {
		return nil, false, nil
	}
	published, active := s.modelCatalogService.activePublishedPublicModelCatalogSnapshot(ctx)
	if !active {
		return nil, false, nil
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil, true, nil
	}
	requestedID := NormalizeModelCatalogModelID(modelID)
	matches := make([]apiKeyPublishedPublicCatalogMatch, 0, len(published.Snapshot.Items))
	for _, item := range published.Snapshot.Items {
		if requestedID != "" && !publicModelCatalogItemMatchesPublicID(item, requestedID) {
			continue
		}
		for _, binding := range bindings {
			match, ok, err := s.publishedPublicCatalogItemForBinding(ctx, binding, platform, item)
			if err != nil {
				return nil, true, err
			}
			if !ok {
				recordPublicCatalogRouteMiss(ctx, apiKey, bindingGroupIDPtr(binding), firstNonEmptyTrimmed(item.PublicModelID, item.Model), platform)
				continue
			}
			if apiKey.IsImageOnly() && strings.TrimSpace(item.Mode) != "image" {
				continue
			}
			matches = append(matches, match)
			break
		}
	}
	return matches, true, nil
}
