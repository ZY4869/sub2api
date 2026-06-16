package service

import (
	"context"
	"sort"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type ExternalModelCatalogGroupSummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Platform    string `json:"platform"`
	Priority    int    `json:"priority"`
	ModelCount  int    `json:"model_count"`
}

type ExternalModelCatalogView struct {
	ExternalModelCatalogViewMode          string                              `json:"external_model_catalog_view_mode"`
	EffectiveExternalModelCatalogViewMode string                              `json:"effective_external_model_catalog_view_mode"`
	ETag                                  string                              `json:"etag,omitempty"`
	UpdatedAt                             string                              `json:"updated_at,omitempty"`
	PublishedAt                           string                              `json:"published_at,omitempty"`
	LastRevalidatedAt                     string                              `json:"last_revalidated_at,omitempty"`
	StaleReason                           string                              `json:"stale_reason,omitempty"`
	PageSize                              int                                 `json:"page_size,omitempty"`
	CatalogSource                         string                              `json:"catalog_source,omitempty"`
	Groups                                []ExternalModelCatalogGroupSummary  `json:"groups"`
	Items                                 []PublicModelCatalogItem            `json:"items"`
	GroupCatalogs                         map[string][]PublicModelCatalogItem `json:"group_catalogs,omitempty"`
}

func (s *APIKeyService) GetExternalModelCatalogView(
	ctx context.Context,
	userID int64,
) (*ExternalModelCatalogView, error) {
	if s == nil || s.userRepo == nil {
		return nil, infraerrors.ServiceUnavailable("USER_SERVICE_UNAVAILABLE", "user service unavailable")
	}
	if s.modelCatalogService == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_CATALOG_UNAVAILABLE", "model catalog service unavailable")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.modelCatalogService.PublicModelCatalogSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		snapshot = emptyPublishedPublicModelCatalogSnapshot()
	}

	options, err := s.GetAvailableGroupModelOptions(ctx, userID)
	if err != nil {
		return nil, err
	}

	availableGroups, err := s.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	groupByID := make(map[int64]Group, len(availableGroups))
	for _, group := range availableGroups {
		groupByID[group.ID] = group
	}

	itemsByModel := make(map[string]PublicModelCatalogItem, len(snapshot.Items))
	for _, item := range snapshot.Items {
		publicItem := externalModelCatalogItemFromPublicSnapshot(item)
		modelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(publicItem.PublicModelID, publicItem.Model))
		if modelID == "" {
			continue
		}
		itemsByModel[modelID] = publicItem
	}

	groupCatalogs := make(map[string][]PublicModelCatalogItem, len(options))
	groups := make([]ExternalModelCatalogGroupSummary, 0, len(options))
	aggregate := make(map[string]PublicModelCatalogItem)
	for _, optionGroup := range options {
		group, ok := groupByID[optionGroup.GroupID]
		if !ok {
			group = Group{
				ID:       optionGroup.GroupID,
				Name:     optionGroup.Name,
				Platform: optionGroup.Platform,
				Priority: optionGroup.Priority,
			}
		}
		items := make([]PublicModelCatalogItem, 0, len(optionGroup.Models))
		for _, model := range optionGroup.Models {
			modelID := NormalizeModelCatalogModelID(model.PublicID)
			if modelID == "" {
				continue
			}
			item, ok := itemsByModel[modelID]
			if !ok {
				continue
			}
			items = append(items, item)
			if _, exists := aggregate[modelID]; !exists {
				aggregate[modelID] = item
			}
		}
		sortPublicModelCatalogItemsForExternalView(items)
		groupCatalogs[strconv.FormatInt(optionGroup.GroupID, 10)] = items
		groups = append(groups, ExternalModelCatalogGroupSummary{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Platform:    CanonicalizePlatformValue(group.Platform),
			Priority:    group.Priority,
			ModelCount:  len(items),
		})
	}
	sort.SliceStable(groups, func(left, right int) bool {
		if groups[left].Priority != groups[right].Priority {
			return groups[left].Priority < groups[right].Priority
		}
		if groups[left].ID != groups[right].ID {
			return groups[left].ID < groups[right].ID
		}
		return strings.ToLower(groups[left].Name) < strings.ToLower(groups[right].Name)
	})

	items := make([]PublicModelCatalogItem, 0, len(aggregate))
	for _, item := range aggregate {
		items = append(items, item)
	}
	sortPublicModelCatalogItemsForExternalView(items)

	return &ExternalModelCatalogView{
		ExternalModelCatalogViewMode:          NormalizeExternalModelCatalogViewMode(user.ExternalModelCatalogViewMode),
		EffectiveExternalModelCatalogViewMode: user.EffectiveExternalModelCatalogViewMode(),
		ETag:                                  snapshot.ETag,
		UpdatedAt:                             snapshot.UpdatedAt,
		PublishedAt:                           snapshot.PublishedAt,
		LastRevalidatedAt:                     snapshot.LastRevalidatedAt,
		StaleReason:                           snapshot.StaleReason,
		PageSize:                              normalizePublicModelCatalogPageSize(snapshot.PageSize),
		CatalogSource:                         snapshot.CatalogSource,
		Groups:                                groups,
		Items:                                 items,
		GroupCatalogs:                         groupCatalogs,
	}, nil
}

func (s *APIKeyService) SubscribePublicModelCatalogEvents(ctx context.Context) (<-chan PublicModelCatalogEvent, error) {
	if s == nil || s.modelCatalogService == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_CATALOG_UNAVAILABLE", "model catalog service unavailable")
	}
	return s.modelCatalogService.SubscribePublicModelCatalogEvents(ctx), nil
}

func externalModelCatalogItemFromPublicSnapshot(item PublicModelCatalogItem) PublicModelCatalogItem {
	cloned := clonePublicModelCatalogItem(item)
	cloned.EntryID = ""
	cloned.BaseModel = ""
	cloned.SourceModelID = ""
	cloned.SourceProtocol = ""
	cloned.SourceAlias = ""
	cloned.SourceAccountID = 0
	cloned.SourceAccountName = ""
	cloned.SourceIDs = nil
	cloned.RuntimePriceSpec = PublicModelCatalogRuntimePriceSpec{}
	if cloned.PublicModelID == "" {
		cloned.PublicModelID = cloned.Model
	}
	if cloned.Model == "" {
		cloned.Model = cloned.PublicModelID
	}
	return cloned
}

func sortPublicModelCatalogItemsForExternalView(items []PublicModelCatalogItem) {
	sort.SliceStable(items, func(left, right int) bool {
		leftName := strings.ToLower(strings.TrimSpace(items[left].DisplayName))
		rightName := strings.ToLower(strings.TrimSpace(items[right].DisplayName))
		if leftName != "" && rightName != "" && leftName != rightName {
			return leftName < rightName
		}
		if leftName != "" && rightName == "" {
			return true
		}
		if leftName == "" && rightName != "" {
			return false
		}
		leftID := strings.TrimSpace(firstNonEmptyTrimmed(items[left].PublicModelID, items[left].Model))
		rightID := strings.TrimSpace(firstNonEmptyTrimmed(items[right].PublicModelID, items[right].Model))
		return leftID < rightID
	})
}
