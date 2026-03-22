package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

type AccountModelScopeV2 struct {
	SupportedProviders        []string            `json:"supported_providers"`
	SupportedModelsByProvider map[string][]string `json:"supported_models_by_provider"`
	AdvancedProviderOverride  bool                `json:"advanced_provider_override"`
	ManualMappings            map[string]string   `json:"manual_mappings"`
}

func ExtractAccountModelScopeV2(extra map[string]any) (*AccountModelScopeV2, bool) {
	if extra == nil {
		return nil, false
	}
	raw, ok := extra["model_scope_v2"]
	if !ok || raw == nil {
		return nil, false
	}
	scopeMap, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	scope := &AccountModelScopeV2{
		SupportedProviders:        normalizeStringSliceAny(scopeMap["supported_providers"], normalizeRegistryPlatform),
		SupportedModelsByProvider: normalizeStringSliceMapAny(scopeMap["supported_models_by_provider"], normalizeRegistryID),
		AdvancedProviderOverride:  parseBoolAny(scopeMap["advanced_provider_override"]),
		ManualMappings:            normalizeStringMapAny(scopeMap["manual_mappings"], normalizeRegistryID, normalizeRegistryID),
	}
	if len(scope.SupportedProviders) == 0 && len(scope.SupportedModelsByProvider) == 0 && len(scope.ManualMappings) == 0 {
		return nil, false
	}
	return scope, true
}

func (s *ModelRegistryService) BuildModelMappingFromScopeV2(ctx context.Context, platform string, accountType string, extra map[string]any) (map[string]string, []string, bool, error) {
	scope, ok := ExtractAccountModelScopeV2(extra)
	if !ok || scope == nil {
		return nil, nil, false, nil
	}
	entries, err := s.pricingEntries(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	index := modelregistry.BuildIndex(entries)
	routeKey := accountModelScopeRouteKey(platform, accountType)
	details, err := s.adminDetails(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	detailIndex := make(map[string]modelregistry.AdminModelDetail, len(details))
	for _, detail := range details {
		detailIndex[detail.ID] = detail
	}

	selectedModels := make([]string, 0)
	selectedSet := map[string]struct{}{}
	mapping := make(map[string]string)

	for _, models := range scope.SupportedModelsByProvider {
		for _, modelID := range models {
			canonicalID, err := s.resolveCanonicalModelForAvailability(ctx, modelID)
			if err != nil {
				return nil, nil, false, err
			}
			if canonicalID == "" {
				canonicalID = normalizeRegistryID(modelID)
			}
			if canonicalID == "" {
				continue
			}
			if _, exists := selectedSet[canonicalID]; !exists {
				selectedSet[canonicalID] = struct{}{}
				selectedModels = append(selectedModels, canonicalID)
			}
			detail, ok := detailIndex[canonicalID]
			if !ok {
				continue
			}
			targetModel := canonicalID
			if resolved, ok := index.ResolveProtocolID(canonicalID, routeKey); ok && strings.TrimSpace(resolved) != "" {
				targetModel = normalizeRegistryID(resolved)
			}
			requestKeys := compactRegistryStrings(canonicalID)
			requestKeys = mergeRegistryStrings(requestKeys, detail.Aliases...)
			requestKeys = mergeRegistryStrings(requestKeys, detail.ProtocolIDs...)
			for _, key := range requestKeys {
				mapping[key] = targetModel
			}
		}
	}

	for from, to := range scope.ManualMappings {
		if from == "" || to == "" {
			continue
		}
		mapping[from] = to
	}

	sort.Strings(selectedModels)
	return mapping, selectedModels, true, nil
}

func (s *ModelRegistryService) InferAccountModelScopeV2(ctx context.Context, platform string, accountType string, mapping map[string]string) *AccountModelScopeV2 {
	if len(mapping) == 0 {
		return nil
	}
	entries, err := s.pricingEntries(ctx)
	if err != nil {
		return nil
	}
	index := modelregistry.BuildIndex(entries)
	details, err := s.adminDetails(ctx)
	if err != nil {
		return nil
	}
	detailIndex := make(map[string]modelregistry.AdminModelDetail, len(details))
	for _, detail := range details {
		detailIndex[detail.ID] = detail
	}

	routeKey := accountModelScopeRouteKey(platform, accountType)
	scope := &AccountModelScopeV2{
		SupportedProviders:        []string{},
		SupportedModelsByProvider: map[string][]string{},
		ManualMappings:            map[string]string{},
	}
	providerSet := map[string]struct{}{}

	for from, to := range mapping {
		canonicalID := ""
		if resolved, ok := index.ResolveCanonicalID(from); ok && resolved != "" {
			canonicalID = resolved
		} else if resolved, ok := index.ResolveCanonicalID(to); ok && resolved != "" {
			canonicalID = resolved
		} else {
			canonicalID = normalizeRegistryID(from)
		}
		if canonicalID == "" {
			scope.ManualMappings[from] = to
			continue
		}
		detail, ok := detailIndex[canonicalID]
		if !ok {
			scope.ManualMappings[from] = to
			continue
		}
		provider := normalizeRegistryPlatform(detail.Provider)
		if provider == "" {
			provider = inferModelProvider(canonicalID)
		}
		if provider == "" {
			provider = normalizeRegistryPlatform(platform)
		}
		if provider != "" {
			providerSet[provider] = struct{}{}
			scope.SupportedModelsByProvider[provider] = mergeRegistryStrings(scope.SupportedModelsByProvider[provider], canonicalID)
		}
		expectedTarget := canonicalID
		if resolved, ok := index.ResolveProtocolID(canonicalID, routeKey); ok && strings.TrimSpace(resolved) != "" {
			expectedTarget = normalizeRegistryID(resolved)
		}
		if normalizeRegistryID(from) != canonicalID || normalizeRegistryID(to) != expectedTarget {
			scope.ManualMappings[from] = to
		}
	}

	for provider := range providerSet {
		scope.SupportedProviders = append(scope.SupportedProviders, provider)
	}
	sort.Strings(scope.SupportedProviders)
	for provider, models := range scope.SupportedModelsByProvider {
		sort.Strings(models)
		scope.SupportedModelsByProvider[provider] = models
	}
	if len(scope.SupportedProviders) == 0 && len(scope.ManualMappings) == 0 {
		return nil
	}
	return scope
}

func (scope *AccountModelScopeV2) ToMap() map[string]any {
	if scope == nil {
		return nil
	}
	modelsByProvider := make(map[string]any, len(scope.SupportedModelsByProvider))
	for provider, models := range scope.SupportedModelsByProvider {
		modelsByProvider[provider] = append([]string(nil), models...)
	}
	manualMappings := make(map[string]any, len(scope.ManualMappings))
	for from, to := range scope.ManualMappings {
		manualMappings[from] = to
	}
	return map[string]any{
		"supported_providers":          append([]string(nil), scope.SupportedProviders...),
		"supported_models_by_provider": modelsByProvider,
		"advanced_provider_override":   scope.AdvancedProviderOverride,
		"manual_mappings":              manualMappings,
	}
}

func accountModelScopeRouteKey(platform string, accountType string) string {
	normalizedPlatform := normalizeRegistryPlatform(platform)
	switch normalizedPlatform {
	case PlatformAnthropic:
		if strings.TrimSpace(strings.ToLower(accountType)) == AccountTypeAPIKey {
			return "anthropic_apikey"
		}
		return "anthropic_oauth"
	case PlatformKiro:
		return "kiro"
	case PlatformOpenAI:
		return "openai"
	case PlatformCopilot:
		return "copilot"
	case PlatformGemini:
		return "gemini"
	case PlatformAntigravity:
		return "antigravity"
	case PlatformSora:
		return "sora"
	default:
		return normalizedPlatform
	}
}

func normalizeStringSliceAny(raw any, normalize func(string) string) []string {
	values, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]string); ok {
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				items = append(items, item)
			}
			values = items
		} else {
			return nil
		}
	}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, item := range values {
		value, ok := item.(string)
		if !ok {
			continue
		}
		value = normalize(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeStringSliceMapAny(raw any, normalize func(string) string) map[string][]string {
	value, ok := raw.(map[string]any)
	if !ok {
		return map[string][]string{}
	}
	result := make(map[string][]string, len(value))
	for provider, items := range value {
		normalizedProvider := normalizeRegistryPlatform(provider)
		if normalizedProvider == "" {
			continue
		}
		result[normalizedProvider] = normalizeStringSliceAny(items, normalize)
	}
	return result
}

func normalizeStringMapAny(raw any, normalizeKey func(string) string, normalizeValue func(string) string) map[string]string {
	value, ok := raw.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		normalizedKey := normalizeKey(key)
		stringValue, ok := item.(string)
		if !ok {
			continue
		}
		normalizedValue := normalizeValue(stringValue)
		if normalizedKey == "" || normalizedValue == "" {
			continue
		}
		result[normalizedKey] = normalizedValue
	}
	return result
}

func parseBoolAny(raw any) bool {
	value, ok := raw.(bool)
	return ok && value
}
