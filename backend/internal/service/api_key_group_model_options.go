package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
)

type UserGroupModelOption struct {
	PublicID         string   `json:"public_id"`
	DisplayName      string   `json:"display_name"`
	RequestProtocols []string `json:"request_protocols,omitempty"`
	SourceIDs        []string `json:"source_ids,omitempty"`
}

type UserGroupModelOptionGroup struct {
	GroupID   int64                  `json:"group_id"`
	Name      string                 `json:"name"`
	Platform  string                 `json:"platform"`
	Priority  int                    `json:"priority"`
	Models    []UserGroupModelOption `json:"models"`
	ModelCount int                   `json:"model_count"`
}

func (s *APIKeyService) GetAvailableGroupModelOptions(
	ctx context.Context,
	userID int64,
) ([]UserGroupModelOptionGroup, error) {
	groups, err := s.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}

	catalogItemsByModel := make(map[string]PublicModelCatalogItem)
	if s != nil && s.modelCatalogService != nil {
		snapshot, err := s.modelCatalogService.PublicModelCatalogSnapshot(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range snapshot.Items {
			catalogItemsByModel[item.Model] = item
		}
	}

	result := make([]UserGroupModelOptionGroup, 0, len(groups))
	for i := range groups {
		group := &groups[i]
		models := make([]UserGroupModelOption, 0)
		if s != nil && s.gatewayService != nil {
			projection, err := s.gatewayService.ListGroupPublicModelProjection(ctx, group, nil)
			if err != nil {
				return nil, err
			}
			for _, entry := range projection {
				catalogItem, ok := catalogItemsByModel[NormalizeModelCatalogModelID(entry.PublicID)]
				if len(catalogItemsByModel) > 0 && !ok {
					continue
				}
				displayName := strings.TrimSpace(entry.DisplayName)
				if ok && strings.TrimSpace(catalogItem.DisplayName) != "" {
					displayName = strings.TrimSpace(catalogItem.DisplayName)
				}
				if displayName == "" {
					displayName = FormatModelCatalogDisplayName(entry.PublicID)
				}
				models = append(models, UserGroupModelOption{
					PublicID:         NormalizeModelCatalogModelID(entry.PublicID),
					DisplayName:      displayName,
					RequestProtocols: append([]string(nil), catalogItem.RequestProtocols...),
					SourceIDs:        append([]string(nil), entry.SourceIDs...),
				})
			}
		}
		sort.SliceStable(models, func(left, right int) bool {
			leftName := strings.ToLower(strings.TrimSpace(models[left].DisplayName))
			rightName := strings.ToLower(strings.TrimSpace(models[right].DisplayName))
			if leftName == rightName {
				return models[left].PublicID < models[right].PublicID
			}
			return leftName < rightName
		})
		result = append(result, UserGroupModelOptionGroup{
			GroupID:    group.ID,
			Name:       group.Name,
			Platform:   group.Platform,
			Priority:   group.Priority,
			Models:     models,
			ModelCount: len(models),
		})
	}

	sort.SliceStable(result, func(left, right int) bool {
		if result[left].Priority != result[right].Priority {
			return result[left].Priority < result[right].Priority
		}
		return result[left].GroupID < result[right].GroupID
	})

	slog.Info(
		"user_group_model_options_built",
		"user_id", userID,
		"group_count", len(result),
	)
	return result, nil
}
