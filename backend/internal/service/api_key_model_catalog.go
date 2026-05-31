package service

import (
	"context"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type APIKeyModelCatalogOptions struct {
	IncludeUnavailable bool
	Platform           string
}

func (s *APIKeyService) GetAPIKeyModelCatalogSnapshot(
	ctx context.Context,
	userID int64,
	keyID int64,
	options APIKeyModelCatalogOptions,
) (*PublicModelCatalogSnapshot, error) {
	if keyID <= 0 {
		return nil, infraerrors.BadRequest("API_KEY_REQUIRED", "api key id is required")
	}
	if s == nil || s.gatewayService == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_CATALOG_UNAVAILABLE", "model catalog service unavailable")
	}
	apiKey, err := s.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if apiKey == nil || apiKey.UserID != userID {
		return nil, infraerrors.Forbidden("API_KEY_FORBIDDEN", "api key does not belong to current user")
	}
	return s.gatewayService.APIKeyModelCatalogSnapshot(ctx, apiKey, options)
}

func (s *GatewayService) APIKeyModelCatalogSnapshot(
	ctx context.Context,
	apiKey *APIKey,
	options APIKeyModelCatalogOptions,
) (*PublicModelCatalogSnapshot, error) {
	if s == nil || s.modelCatalogService == nil || apiKey == nil {
		return emptyPublishedPublicModelCatalogSnapshot(), nil
	}
	rawPublished := s.modelCatalogService.loadPublishedPublicModelCatalogSnapshot(ctx)
	if rawPublished == nil {
		return emptyPublishedPublicModelCatalogSnapshot(), nil
	}
	rawPublished = filterPublicModelCatalogPublishedSnapshotByDemoMode(rawPublished, false)
	rawSnapshot := clonePublicModelCatalogSnapshot(&rawPublished.Snapshot)
	rawSnapshot.CatalogSource = PublicModelCatalogSourcePublished
	availableMatches, _, err := s.apiKeyPublishedPublicCatalogVisibleMatches(ctx, apiKey, options.Platform, "")
	if err != nil {
		return nil, err
	}
	available := apiKeyCatalogAvailableSet(availableMatches)
	items := make([]PublicModelCatalogItem, 0, len(rawSnapshot.Items))
	for _, item := range rawSnapshot.Items {
		publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if publicID == "" {
			continue
		}
		if _, ok := available[publicID]; ok {
			next := sanitizePublicModelCatalogItemForPublicWithSource(item, PublicModelCatalogSourcePublished)
			next.KeyAvailability = PublicModelKeyAvailabilityAvailable
			items = append(items, next)
			continue
		}
		if !options.IncludeUnavailable {
			continue
		}
		next := sanitizePublicModelCatalogItemForPublicWithSource(item, PublicModelCatalogSourcePublished)
		next.KeyAvailability = PublicModelKeyAvailabilityUnavailable
		next.UnavailableReason = s.apiKeyPublishedCatalogUnavailableReason(ctx, apiKey, item, options.Platform)
		items = append(items, next)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return strings.ToLower(firstNonEmptyTrimmed(items[i].DisplayName, items[i].Model)) <
			strings.ToLower(firstNonEmptyTrimmed(items[j].DisplayName, items[j].Model))
	})
	rawSnapshot.Items = items
	if etag, err := computePublicModelCatalogETag(rawSnapshot); err == nil {
		rawSnapshot.ETag = etag
	} else {
		rawSnapshot.ETag = ""
	}
	return rawSnapshot, nil
}

func apiKeyCatalogAvailableSet(matches []apiKeyPublishedPublicCatalogMatch) map[string]struct{} {
	out := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(match.Entry.PublicID, match.SourceItem.PublicModelID, match.SourceItem.Model))
		if publicID != "" {
			out[publicID] = struct{}{}
		}
	}
	return out
}

func (s *GatewayService) apiKeyPublishedCatalogUnavailableReason(
	ctx context.Context,
	apiKey *APIKey,
	item PublicModelCatalogItem,
	platform string,
) string {
	if apiKey != nil && apiKey.IsImageOnly() && !publicModelCatalogItemIsImage(item) {
		return PublicModelUnavailableReasonImageOnlyKeyRestricted
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return PublicModelUnavailableReasonGroupUnavailable
	}
	publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
	hasActiveGroup, hasModelBinding, hasProtocolBinding, hasVisibleBinding := false, false, false, false
	for _, binding := range bindings {
		if binding.Group == nil || !binding.Group.IsActive() {
			continue
		}
		hasActiveGroup = true
		if _, matched := bindingMatchesModel(binding.ModelPatterns, publicID); !matched {
			continue
		}
		hasModelBinding = true
		if _, ok := publishedCatalogBindingItemPlatform(binding, platform, item); !ok {
			continue
		}
		hasProtocolBinding = true
		entry := APIKeyPublicModelEntry{
			PublicID:    publicID,
			AliasID:     publicID,
			SourceID:    NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel)),
			DisplayName: firstNonEmptyTrimmed(item.DisplayName, item.BaseModel, publicID),
		}
		if binding.Group.AllowsVisibleModel(entry.PublicID, entry.AliasID, entry.SourceID, entry.DisplayName) {
			hasVisibleBinding = true
			break
		}
	}
	if !hasActiveGroup {
		return PublicModelUnavailableReasonGroupUnavailable
	}
	if !hasModelBinding {
		return PublicModelUnavailableReasonNotSelectedByKey
	}
	if !hasProtocolBinding || !hasVisibleBinding {
		return PublicModelUnavailableReasonGroupUnavailable
	}
	if !s.modelCatalogService.publicModelCatalogItemRouteConfirmed(ctx, item) {
		return PublicModelUnavailableReasonPublishedSourceUnavailable
	}
	return PublicModelUnavailableReasonGroupUnavailable
}

func publicModelCatalogItemIsImage(item PublicModelCatalogItem) bool {
	if strings.EqualFold(strings.TrimSpace(item.Mode), "image") {
		return true
	}
	for _, value := range append(append([]string{}, item.Modalities...), item.Capabilities...) {
		normalized := strings.TrimSpace(strings.ToLower(value))
		if normalized == "image" || normalized == "image_generation" {
			return true
		}
	}
	return false
}
