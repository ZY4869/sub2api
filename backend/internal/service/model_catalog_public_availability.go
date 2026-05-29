package service

import (
	"context"
	"strings"
)

func (s *ModelCatalogService) publicModelCatalogItemRouteConfirmed(ctx context.Context, item PublicModelCatalogItem) bool {
	if !publicModelCatalogItemConfirmedAvailable(item) {
		return false
	}
	if s == nil || s.gatewayService == nil {
		return true
	}
	return s.gatewayService.publicModelCatalogItemRouteConfirmed(ctx, item)
}

func (s *GatewayService) publicModelCatalogItemRouteConfirmed(ctx context.Context, item PublicModelCatalogItem) bool {
	if s == nil {
		return false
	}
	if !publicModelCatalogItemConfirmedAvailable(item) {
		return false
	}
	if item.SourceAccountID <= 0 {
		return true
	}
	if s.accountRepo == nil {
		return true
	}
	account, err := s.accountRepo.GetByID(ctx, item.SourceAccountID)
	if err != nil || account == nil || !account.IsSchedulable() {
		return false
	}
	sourceProtocol := firstNonEmptyTrimmed(item.SourceProtocol, RoutingPlatformForAccount(account))
	resolved := ResolveProtocolGatewayInboundAccount(account, sourceProtocol)
	if resolved == nil || !resolved.IsSchedulable() {
		return false
	}
	if sourceModel := firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel); sourceModel != "" &&
		!accountSupportsPublishedCatalogSourceModel(ctx, s.modelRegistryService, resolved, sourceModel) {
		return false
	}
	if s.groupRepo == nil {
		return true
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return false
	}
	for i := range groups {
		group := &groups[i]
		if group == nil || !group.IsActive() || !accountBoundToGroupID(account, &group.ID) {
			continue
		}
		for _, platform := range groupProjectionPlatformsForAccount(group.Platform, account) {
			if strings.TrimSpace(platform) == "" {
				continue
			}
			if !publicCatalogItemMatchesProtocol(item, platform) {
				continue
			}
			if publishedCatalogAccountUsableForBinding(ctx, s.modelRegistryService, account, group.ID, platform, item) {
				return true
			}
		}
	}
	return false
}
