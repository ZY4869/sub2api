package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

type AccountModelScopeV2 struct {
	SupportedProviders        []string                            `json:"supported_providers"`
	SupportedModelsByProvider map[string][]string                 `json:"supported_models_by_provider"`
	AdvancedProviderOverride  bool                                `json:"advanced_provider_override"`
	SelectedModelIDs          []string                            `json:"selected_model_ids,omitempty"`
	ManualMappingRows         []AccountModelScopeManualMappingRow `json:"manual_mapping_rows,omitempty"`
	ManualMappings            map[string]string                   `json:"manual_mappings"`
}

type AccountModelScopeManualMappingRow struct {
	From string `json:"from"`
	To   string `json:"to"`
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
		SelectedModelIDs:          normalizeStringSliceAny(scopeMap["selected_model_ids"], strings.TrimSpace),
		ManualMappingRows:         normalizeManualMappingRowsAny(scopeMap["manual_mapping_rows"]),
		ManualMappings:            normalizeStringMapAny(scopeMap["manual_mappings"], normalizeRegistryID, normalizeRegistryID),
	}
	if len(scope.SupportedProviders) == 0 &&
		len(scope.SupportedModelsByProvider) == 0 &&
		len(scope.SelectedModelIDs) == 0 &&
		len(scope.ManualMappings) == 0 &&
		len(scope.ManualMappingRows) == 0 {
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

	for _, modelID := range scopeSelectedModels(scope) {
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
			mapping[canonicalID] = canonicalID
			continue
		}
		targetModel := canonicalID
		if resolved, ok := index.ResolveProtocolID(canonicalID, routeKey); ok && strings.TrimSpace(resolved) != "" {
			targetModel = normalizeRegistryID(resolved)
		}
		for _, key := range accountModelScopeRequestKeys(canonicalID, detail) {
			mapping[key] = targetModel
		}
	}

	if len(scope.ManualMappingRows) > 0 {
		for _, row := range scope.ManualMappingRows {
			from := strings.TrimSpace(row.From)
			to := strings.TrimSpace(row.To)
			if from == "" || to == "" {
				continue
			}
			mapping[from] = to
		}
	} else {
		for from, to := range scope.ManualMappings {
			if from == "" || to == "" {
				continue
			}
			mapping[from] = to
		}
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
		SelectedModelIDs:          []string{},
		ManualMappingRows:         []AccountModelScopeManualMappingRow{},
		ManualMappings:            map[string]string{},
	}
	providerSet := map[string]struct{}{}
	selectedModelIDsByCanonical := map[string][]string{}

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
			scope.ManualMappingRows = append(scope.ManualMappingRows, AccountModelScopeManualMappingRow{From: from, To: to})
			scope.ManualMappings[from] = to
			continue
		}
		detail, ok := detailIndex[canonicalID]
		if !ok {
			scope.ManualMappingRows = append(scope.ManualMappingRows, AccountModelScopeManualMappingRow{From: from, To: to})
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

		normalizedFrom := normalizeRegistryID(from)
		normalizedTo := normalizeRegistryID(to)
		requestKeySet := make(map[string]struct{})
		for _, key := range accountModelScopeRequestKeys(canonicalID, detail) {
			requestKeySet[key] = struct{}{}
		}
		_, isKnownRequestKey := requestKeySet[normalizedFrom]
		if isKnownRequestKey && normalizedFrom == normalizedTo {
			selectedModelIDsByCanonical[canonicalID] = mergeRegistryStrings(
				selectedModelIDsByCanonical[canonicalID],
				strings.TrimSpace(from),
			)
			continue
		}
		if isKnownRequestKey && normalizedTo == expectedTarget {
			continue
		}
		if normalizedFrom != canonicalID || normalizedTo != expectedTarget {
			scope.ManualMappingRows = append(scope.ManualMappingRows, AccountModelScopeManualMappingRow{From: from, To: to})
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
	for _, provider := range scope.SupportedProviders {
		for _, modelID := range scope.SupportedModelsByProvider[provider] {
			selectedIDs := append([]string(nil), selectedModelIDsByCanonical[modelID]...)
			if len(selectedIDs) == 0 {
				selectedIDs = []string{modelID}
			} else {
				sort.Strings(selectedIDs)
			}
			scope.SelectedModelIDs = mergeRegistryStrings(scope.SelectedModelIDs, selectedIDs...)
		}
	}
	sort.Slice(scope.ManualMappingRows, func(i, j int) bool {
		if scope.ManualMappingRows[i].From == scope.ManualMappingRows[j].From {
			return scope.ManualMappingRows[i].To < scope.ManualMappingRows[j].To
		}
		return scope.ManualMappingRows[i].From < scope.ManualMappingRows[j].From
	})
	if len(scope.SupportedProviders) == 0 &&
		len(scope.SelectedModelIDs) == 0 &&
		len(scope.ManualMappings) == 0 &&
		len(scope.ManualMappingRows) == 0 {
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
	manualMappingRows := make([]map[string]any, 0, len(scope.ManualMappingRows))
	for _, row := range scope.ManualMappingRows {
		if strings.TrimSpace(row.From) == "" || strings.TrimSpace(row.To) == "" {
			continue
		}
		manualMappingRows = append(manualMappingRows, map[string]any{
			"from": row.From,
			"to":   row.To,
		})
	}
	result := map[string]any{
		"supported_providers":          append([]string(nil), scope.SupportedProviders...),
		"supported_models_by_provider": modelsByProvider,
		"advanced_provider_override":   scope.AdvancedProviderOverride,
		"manual_mapping_rows":          manualMappingRows,
		"manual_mappings":              manualMappings,
	}
	if len(scope.SelectedModelIDs) > 0 {
		result["selected_model_ids"] = append([]string(nil), scope.SelectedModelIDs...)
	}
	return result
}

func scopeSelectedModels(scope *AccountModelScopeV2) []string {
	if scope == nil {
		return nil
	}
	if len(scope.SupportedModelsByProvider) > 0 {
		providers := make([]string, 0, len(scope.SupportedModelsByProvider))
		for provider := range scope.SupportedModelsByProvider {
			providers = append(providers, provider)
		}
		sort.Strings(providers)
		models := make([]string, 0)
		for _, provider := range providers {
			models = append(models, scope.SupportedModelsByProvider[provider]...)
		}
		return models
	}
	return append([]string(nil), scope.SelectedModelIDs...)
}

func accountModelScopeRequestKeys(canonicalID string, detail modelregistry.AdminModelDetail) []string {
	requestKeys := compactRegistryStrings(canonicalID)
	requestKeys = mergeRegistryStrings(requestKeys, detail.Aliases...)
	requestKeys = mergeRegistryStrings(requestKeys, detail.ProtocolIDs...)
	return requestKeys
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
	case PlatformGrok:
		return "grok"
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

func normalizeManualMappingRowsAny(raw any) []AccountModelScopeManualMappingRow {
	values, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]map[string]any); ok {
			values = make([]any, 0, len(typed))
			for _, item := range typed {
				values = append(values, item)
			}
		} else {
			return nil
		}
	}
	rows := make([]AccountModelScopeManualMappingRow, 0, len(values))
	seen := map[string]struct{}{}
	for _, item := range values {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		from, ok := entry["from"].(string)
		if !ok {
			continue
		}
		to, ok := entry["to"].(string)
		if !ok {
			continue
		}
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		if from == "" || to == "" {
			continue
		}
		key := from + "\x00" + to
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		rows = append(rows, AccountModelScopeManualMappingRow{From: from, To: to})
	}
	return rows
}

func parseBoolAny(raw any) bool {
	value, ok := raw.(bool)
	return ok && value
}
