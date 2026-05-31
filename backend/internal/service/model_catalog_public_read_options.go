package service

import "strings"

func normalizePublicModelCatalogReadOptions(options PublicModelCatalogReadOptions) PublicModelCatalogReadOptions {
	mode := strings.ToLower(strings.TrimSpace(options.CatalogMode))
	switch mode {
	case PublicModelCatalogModeDemo:
		options.CatalogMode = PublicModelCatalogModeDemo
	default:
		options.CatalogMode = PublicModelCatalogModeReal
	}
	return options
}

func (s *ModelCatalogService) publicModelCatalogDemoModeEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.PublicModelCatalog.DemoMode
}

func (s *ModelCatalogService) publicModelCatalogReadAllowsDemo(options PublicModelCatalogReadOptions) bool {
	options = normalizePublicModelCatalogReadOptions(options)
	return options.CatalogMode == PublicModelCatalogModeDemo && s.publicModelCatalogDemoModeEnabled()
}

func publicModelCatalogItemIsDemo(item PublicModelCatalogItem) bool {
	return item.IsDemo || strings.EqualFold(strings.TrimSpace(item.CatalogEntrySource), PublicModelCatalogEntrySourceDemo)
}

func filterPublicModelCatalogItemsByDemoMode(items []PublicModelCatalogItem, allowDemo bool) []PublicModelCatalogItem {
	if len(items) == 0 {
		return []PublicModelCatalogItem{}
	}
	filtered := make([]PublicModelCatalogItem, 0, len(items))
	for _, item := range items {
		isDemo := publicModelCatalogItemIsDemo(item)
		if allowDemo != isDemo {
			continue
		}
		filtered = append(filtered, clonePublicModelCatalogItem(item))
	}
	return filtered
}

func filterPublicModelCatalogSnapshotByDemoMode(snapshot *PublicModelCatalogSnapshot, allowDemo bool) *PublicModelCatalogSnapshot {
	cloned := clonePublicModelCatalogSnapshot(snapshot)
	if cloned == nil {
		return nil
	}
	cloned.Items = filterPublicModelCatalogItemsByDemoMode(cloned.Items, allowDemo)
	if etag, err := computePublicModelCatalogETag(cloned); err == nil {
		cloned.ETag = etag
	}
	return cloned
}

func filterPublicModelCatalogPublishedSnapshotByDemoMode(
	snapshot *PublicModelCatalogPublishedSnapshot,
	allowDemo bool,
) *PublicModelCatalogPublishedSnapshot {
	cloned := clonePublicModelCatalogPublishedSnapshot(snapshot)
	if cloned == nil {
		return nil
	}
	cloned.Snapshot.Items = filterPublicModelCatalogItemsByDemoMode(cloned.Snapshot.Items, allowDemo)
	if len(cloned.Details) > 0 {
		filteredDetails := make(map[string]PublicModelCatalogDetail, len(cloned.Details))
		for _, item := range cloned.Snapshot.Items {
			publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
			if publicID == "" {
				continue
			}
			if detail, ok := cloned.Details[publicID]; ok {
				filteredDetails[publicID] = detail
			}
		}
		cloned.Details = filteredDetails
	}
	if etag, err := computePublicModelCatalogETag(&cloned.Snapshot); err == nil {
		cloned.Snapshot.ETag = etag
	}
	return cloned
}
