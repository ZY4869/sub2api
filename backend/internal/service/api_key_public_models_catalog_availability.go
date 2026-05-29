package service

import (
	"context"
	"strings"
)

func publishedCatalogAccountUsableForBinding(
	ctx context.Context,
	registry *ModelRegistryService,
	account *Account,
	groupID int64,
	projectionPlatform string,
	item PublicModelCatalogItem,
) bool {
	if account == nil || !account.IsSchedulable() {
		return false
	}
	if isOpenAIGroupPlatform(projectionPlatform) && !isOpenAITextRuntimeAccount(ResolveProtocolGatewayInboundAccount(account, projectionPlatform)) {
		return false
	}
	if !accountBoundToGroupID(account, &groupID) {
		return false
	}
	if !MatchesGroupPlatform(account, projectionPlatform) {
		return false
	}
	resolved := ResolveProtocolGatewayInboundAccount(account, firstNonEmptyTrimmed(item.SourceProtocol, projectionPlatform))
	if resolved == nil || !resolved.IsSchedulable() {
		return false
	}
	sourceModel := firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel)
	return accountSupportsPublishedCatalogSourceModel(ctx, registry, resolved, sourceModel)
}

func isOpenAIGroupPlatform(platform string) bool {
	switch strings.TrimSpace(strings.ToLower(platform)) {
	case PlatformOpenAI, PlatformDeepSeek, PlatformOpenRouter:
		return true
	default:
		return false
	}
}

func accountSupportsPublishedCatalogSourceModel(ctx context.Context, registry *ModelRegistryService, account *Account, sourceModel string) bool {
	sourceModel = strings.TrimSpace(sourceModel)
	if sourceModel == "" {
		return true
	}
	if isRequestedModelSupportedByAccount(ctx, registry, account, sourceModel) {
		return true
	}
	sourceCandidates := modelIDComparisonSet(ctx, registry, sourceModel)
	for _, model := range BuildAvailableTestModels(ctx, account, registry) {
		for _, candidate := range []string{model.ID, model.TargetModelID, model.CanonicalID} {
			if modelIDComparisonSetsOverlap(sourceCandidates, modelIDComparisonSet(ctx, registry, candidate)) {
				return true
			}
		}
	}
	return false
}

func modelIDComparisonSet(ctx context.Context, registry *ModelRegistryService, modelID string) map[string]struct{} {
	set := collectModelSupportVariants(ctx, registry, "", modelID)
	for _, candidate := range []string{
		modelID,
		NormalizeRequestedModelForClaudeCapability(modelID),
		NormalizeModelCatalogModelID(modelID),
		normalizeRegistryID(modelID),
	} {
		normalized := strings.TrimSpace(candidate)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
}

func modelIDComparisonSetsOverlap(left, right map[string]struct{}) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	for candidate := range left {
		if _, ok := right[candidate]; ok {
			return true
		}
	}
	return false
}
