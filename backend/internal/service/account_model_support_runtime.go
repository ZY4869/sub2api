package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

func resolveCanonicalRequestModelWithRegistry(ctx context.Context, registry *ModelRegistryService, requestedModel string) string {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		return ""
	}
	if registry != nil {
		if resolved, ok, err := registry.ResolveModel(ctx, requestedModel); err == nil && ok && resolved != "" {
			return resolved
		}
	}
	if resolved, ok := modelregistry.ResolveToCanonicalID(requestedModel); ok {
		return resolved
	}
	return normalizeRegistryID(requestedModel)
}

func resolveUpstreamModelIDWithRegistry(ctx context.Context, registry *ModelRegistryService, account *Account, requestedModel string) string {
	requestedModel = resolveCanonicalRequestModelWithRegistry(ctx, registry, requestedModel)
	if requestedModel == "" || account == nil {
		return requestedModel
	}
	route := registryRouteForAccount(account)
	if registry != nil {
		if resolved, ok, err := registry.ResolveProtocolModel(ctx, requestedModel, route); err == nil && ok && resolved != "" {
			return resolved
		}
	}
	if resolved, ok := modelregistry.ResolveToProtocolID(requestedModel, route); ok {
		return resolved
	}
	return requestedModel
}

func accountConfiguredSourceModelIDs(account *Account, sourceProtocol string) []string {
	if account == nil {
		return nil
	}
	normalizedSourceProtocol := NormalizeGatewayProtocol(sourceProtocol)
	seen := map[string]struct{}{}
	ordered := make([]string, 0)
	appendModel := func(modelID string) {
		normalized := normalizeRegistryID(modelID)
		if normalized == "" {
			return
		}
		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		ordered = append(ordered, normalized)
	}

	for _, model := range AccountManualModelsFromExtra(account.Extra, IsProtocolGatewayAccount(account)) {
		if normalizedSourceProtocol != "" {
			manualProtocol := NormalizeGatewayProtocol(model.SourceProtocol)
			if manualProtocol != "" && manualProtocol != normalizedSourceProtocol {
				continue
			}
		}
		appendModel(model.ModelID)
	}

	if scope, ok := ExtractAccountModelScopeV2(account.Extra); ok && scope != nil {
		for _, models := range scope.SupportedModelsByProvider {
			for _, modelID := range models {
				appendModel(modelID)
			}
		}
		for _, row := range scope.ManualMappingRows {
			appendModel(row.To)
		}
		for _, modelID := range scope.ManualMappings {
			appendModel(modelID)
		}
	}

	for _, modelID := range account.GetModelMapping() {
		appendModel(modelID)
	}

	return ordered
}

func accountHasExplicitModelRestrictions(account *Account) bool {
	if account == nil {
		return false
	}
	if account.Type == AccountTypeBedrock || account.Platform == PlatformAntigravity {
		return true
	}
	if len(account.GetModelMapping()) > 0 {
		return true
	}
	return len(accountConfiguredSourceModelIDs(account, "")) > 0
}

func collectModelSupportVariants(ctx context.Context, registry *ModelRegistryService, route string, modelID string) map[string]struct{} {
	set := map[string]struct{}{}
	add := func(value string) {
		normalized := normalizeRegistryID(value)
		if normalized == "" {
			return
		}
		set[normalized] = struct{}{}
	}

	add(modelID)

	if registry != nil {
		if resolution, err := registry.ExplainResolution(ctx, modelID); err == nil && resolution != nil {
			add(resolution.CanonicalID)
			add(resolution.EffectiveID)
			add(resolution.PricingID)
			add(resolution.Entry.ID)
			for _, alias := range resolution.Entry.Aliases {
				add(alias)
			}
			for _, protocolID := range resolution.Entry.ProtocolIDs {
				add(protocolID)
			}
			if resolution.ReplacementEntry != nil {
				add(resolution.ReplacementEntry.ID)
				for _, alias := range resolution.ReplacementEntry.Aliases {
					add(alias)
				}
				for _, protocolID := range resolution.ReplacementEntry.ProtocolIDs {
					add(protocolID)
				}
			}
		}
		if resolved, ok, err := registry.ResolveModel(ctx, modelID); err == nil && ok {
			add(resolved)
		}
		if resolved, ok, err := registry.ResolveProtocolModel(ctx, modelID, route); err == nil && ok {
			add(resolved)
		}
		return set
	}

	if resolution, ok := modelregistry.ExplainSeedResolution(modelID); ok && resolution != nil {
		add(resolution.CanonicalID)
		add(resolution.EffectiveID)
		add(resolution.PricingID)
		add(resolution.Entry.ID)
		for _, alias := range resolution.Entry.Aliases {
			add(alias)
		}
		for _, protocolID := range resolution.Entry.ProtocolIDs {
			add(protocolID)
		}
		if resolution.ReplacementEntry != nil {
			add(resolution.ReplacementEntry.ID)
			for _, alias := range resolution.ReplacementEntry.Aliases {
				add(alias)
			}
			for _, protocolID := range resolution.ReplacementEntry.ProtocolIDs {
				add(protocolID)
			}
		}
	}
	if resolved, ok := modelregistry.ResolveToCanonicalID(modelID); ok {
		add(resolved)
	}
	if resolved, ok := modelregistry.ResolveToProtocolID(modelID, route); ok {
		add(resolved)
	}
	return set
}

func collectRequestedModelSupportVariants(ctx context.Context, registry *ModelRegistryService, account *Account, requestedModel string) map[string]struct{} {
	set := map[string]struct{}{}
	addAll := func(values map[string]struct{}) {
		for value := range values {
			set[value] = struct{}{}
		}
	}

	route := registryRouteForAccount(account)
	addAll(collectModelSupportVariants(ctx, registry, route, requestedModel))

	if canonical := resolveCanonicalRequestModelWithRegistry(ctx, registry, requestedModel); canonical != "" {
		addAll(collectModelSupportVariants(ctx, registry, route, canonical))
	}
	if upstream := resolveUpstreamModelIDWithRegistry(ctx, registry, account, requestedModel); upstream != "" {
		addAll(collectModelSupportVariants(ctx, registry, route, upstream))
	}
	return set
}

func isRequestedModelSupportedByAccount(ctx context.Context, registry *ModelRegistryService, account *Account, requestedModel string) bool {
	if account == nil {
		return false
	}
	if strings.TrimSpace(requestedModel) == "" {
		return true
	}
	if account.Type == AccountTypeBedrock {
		_, ok := ResolveBedrockModelID(account, requestedModel)
		return ok
	}
	if account.Platform == PlatformAntigravity {
		return mapAntigravityModel(account, requestedModel) != ""
	}

	requestedVariants := collectRequestedModelSupportVariants(ctx, registry, account, requestedModel)
	if len(requestedVariants) == 0 {
		return account.IsModelSupported(requestedModel)
	}

	if len(account.GetModelMapping()) > 0 {
		for candidate := range requestedVariants {
			if account.IsModelSupported(candidate) {
				return true
			}
		}
	}

	explicitModelIDs := accountConfiguredSourceModelIDs(account, "")
	if len(explicitModelIDs) == 0 {
		return !accountHasExplicitModelRestrictions(account)
	}

	route := registryRouteForAccount(account)
	for _, modelID := range explicitModelIDs {
		allowedVariants := collectModelSupportVariants(ctx, registry, route, modelID)
		for candidate := range allowedVariants {
			if _, ok := requestedVariants[candidate]; ok {
				return true
			}
		}
	}
	return false
}
