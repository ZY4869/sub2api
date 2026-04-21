package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *APIKeyService) GetGroupModelCatalogSnapshot(
	ctx context.Context,
	userID int64,
	groupID int64,
) (*PublicModelCatalogSnapshot, error) {
	if groupID <= 0 {
		return nil, infraerrors.BadRequest("GROUP_REQUIRED", "group_id is required")
	}
	if s == nil || s.modelCatalogService == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_CATALOG_UNAVAILABLE", "model catalog service unavailable")
	}

	groups, err := s.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}

	var target *Group
	for i := range groups {
		if groups[i].ID == groupID {
			target = &groups[i]
			break
		}
	}
	if target == nil {
		return nil, ErrGroupNotAllowed
	}

	snapshot, err := s.modelCatalogService.PublicModelCatalogSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return scalePublicModelCatalogSnapshot(snapshot, normalizePublicModelCatalogGroupMultiplier(target.RateMultiplier))
}

func normalizePublicModelCatalogGroupMultiplier(multiplier float64) float64 {
	if multiplier < 0 {
		return 1
	}
	return multiplier
}

func scalePublicModelCatalogSnapshot(
	snapshot *PublicModelCatalogSnapshot,
	multiplier float64,
) (*PublicModelCatalogSnapshot, error) {
	if snapshot == nil {
		return emptyPublishedPublicModelCatalogSnapshot(), nil
	}

	items := make([]PublicModelCatalogItem, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		items = append(items, scalePublicModelCatalogItem(item, multiplier))
	}

	pageSize := normalizePublicModelCatalogPageSize(snapshot.PageSize)
	etag, err := computePublicModelCatalogETagWithPageSize(pageSize, items)
	if err != nil {
		return nil, err
	}

	return &PublicModelCatalogSnapshot{
		ETag:          etag,
		UpdatedAt:     snapshot.UpdatedAt,
		PageSize:      pageSize,
		CatalogSource: snapshot.CatalogSource,
		Items:         items,
	}, nil
}

func scalePublicModelCatalogItem(item PublicModelCatalogItem, multiplier float64) PublicModelCatalogItem {
	cloned := clonePublicModelCatalogItem(item)
	cloned.PriceDisplay = scalePublicModelCatalogPriceDisplay(cloned.PriceDisplay, multiplier)
	return cloned
}

func scalePublicModelCatalogPriceDisplay(
	display PublicModelCatalogPriceDisplay,
	multiplier float64,
) PublicModelCatalogPriceDisplay {
	return PublicModelCatalogPriceDisplay{
		Primary:   scalePublicModelCatalogPriceEntries(display.Primary, multiplier),
		Secondary: scalePublicModelCatalogPriceEntries(display.Secondary, multiplier),
	}
}

func scalePublicModelCatalogPriceEntries(
	entries []PublicModelCatalogPriceEntry,
	multiplier float64,
) []PublicModelCatalogPriceEntry {
	if len(entries) == 0 {
		return nil
	}
	scaled := make([]PublicModelCatalogPriceEntry, 0, len(entries))
	for _, entry := range entries {
		next := entry
		next.Value = entry.Value * multiplier
		scaled = append(scaled, next)
	}
	return scaled
}
