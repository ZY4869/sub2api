package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

const (
	KiroBuiltinCatalogSource = accountModelProbeSourceKiroBuiltinCatalog
	KiroBuiltinCatalogNotice = "using built-in Kiro model catalog"
)

type KiroModelCatalogItem struct {
	ID          string
	Type        string
	DisplayName string
	CreatedAt   string
}

func KiroBuiltinModelCatalog() []KiroModelCatalogItem {
	items := make([]KiroModelCatalogItem, 0, len(claude.DefaultModels))
	for _, model := range claude.DefaultModels {
		id := NormalizeModelCatalogModelID(model.ID)
		if id == "" {
			continue
		}
		displayName := strings.TrimSpace(model.DisplayName)
		if displayName == "" {
			displayName = id
		}
		items = append(items, KiroModelCatalogItem{
			ID:          id,
			Type:        strings.TrimSpace(model.Type),
			DisplayName: displayName,
			CreatedAt:   strings.TrimSpace(model.CreatedAt),
		})
	}
	return items
}

func KiroBuiltinModelIDs() []string {
	items := KiroBuiltinModelCatalog()
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}
