package admin

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (h *AccountHandler) prepareAccountModelScope(ctx context.Context, platform string, accountType string, credentials map[string]any, extra map[string]any) (map[string]any, map[string]any, error) {
	if h.modelRegistryService == nil {
		return credentials, extra, nil
	}
	effectivePlatform := service.EffectiveProtocolFromValues(platform, extra)
	mapping, selectedModels, hasScope, err := h.modelRegistryService.BuildModelMappingFromScopeV2(ctx, effectivePlatform, accountType, extra)
	if err != nil {
		return credentials, extra, err
	}
	if !hasScope {
		nextCredentials := cloneStringAnyMap(credentials)
		if nextCredentials != nil && extra != nil {
			delete(nextCredentials, "model_mapping")
		}
		nextExtra := cloneStringAnyMap(extra)
		delete(nextExtra, "model_scope_v2")
		if len(nextExtra) == 0 {
			nextExtra = nil
		}
		return nextCredentials, nextExtra, nil
	}
	if len(selectedModels) > 0 {
		if _, err := h.modelRegistryService.EnsureModelsAvailable(ctx, selectedModels); err != nil {
			return credentials, extra, err
		}
	}
	nextCredentials := cloneStringAnyMap(credentials)
	if nextCredentials == nil {
		nextCredentials = map[string]any{}
	}
	nextExtra := cloneStringAnyMap(extra)
	if scope, ok := service.ExtractAccountModelScopeV2(nextExtra); ok && scope != nil {
		if nextExtra == nil {
			nextExtra = map[string]any{}
		}
		nextExtra["model_scope_v2"] = scope.ToMap()
		nextExtra = service.MergeStringAnyMap(nextExtra, service.BuildAccountModelScopePreviewSnapshotExtra(scope))
	}
	if len(mapping) == 0 {
		delete(nextCredentials, "model_mapping")
		return nextCredentials, nextExtra, nil
	}
	nextCredentials["model_mapping"] = stringifyModelMapping(mapping)
	return nextCredentials, nextExtra, nil
}

func (h *AccountHandler) enrichAccountExtraWithModelScope(ctx context.Context, account *service.Account, current map[string]any) map[string]any {
	extra := cloneStringAnyMap(current)
	if h.modelRegistryService == nil || account == nil {
		return extra
	}
	if _, ok := service.ExtractAccountModelScopeV2(extra); ok {
		return extra
	}
	scope := h.modelRegistryService.InferAccountModelScopeV2(ctx, account.EffectiveProtocol(), account.Type, account.GetModelMapping())
	if scope == nil {
		return extra
	}
	if extra == nil {
		extra = map[string]any{}
	}
	extra["model_scope_v2"] = scope.ToMap()
	return extra
}

func cloneStringAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func stringifyModelMapping(mapping map[string]string) map[string]any {
	result := make(map[string]any, len(mapping))
	for key, value := range mapping {
		result[key] = value
	}
	return result
}
