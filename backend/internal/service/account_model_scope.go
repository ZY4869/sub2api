package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

const (
	AccountModelPolicyModeWhitelist = "whitelist"
	AccountModelPolicyModeMapping   = "mapping"

	AccountModelVisibilityModeAlias   = "alias"
	AccountModelVisibilityModeDirect  = "direct"
	AccountModelVisibilityModeDefault = "default_library"
)

type AccountModelScopeV2 struct {
	PolicyMode string                   `json:"policy_mode,omitempty"`
	Entries    []AccountModelScopeEntry `json:"entries,omitempty"`

	// Legacy fields kept for compatibility reads only.
	SupportedProviders        []string                            `json:"supported_providers,omitempty"`
	SupportedModelsByProvider map[string][]string                 `json:"supported_models_by_provider,omitempty"`
	AdvancedProviderOverride  bool                                `json:"advanced_provider_override,omitempty"`
	SelectedModelIDs          []string                            `json:"selected_model_ids,omitempty"`
	ManualMappingRows         []AccountModelScopeManualMappingRow `json:"manual_mapping_rows,omitempty"`
	ManualMappings            map[string]string                   `json:"manual_mappings,omitempty"`
}

type AccountModelScopeEntry struct {
	DisplayModelID string `json:"display_model_id"`
	TargetModelID  string `json:"target_model_id"`
	Provider       string `json:"provider,omitempty"`
	SourceProtocol string `json:"source_protocol,omitempty"`
	VisibilityMode string `json:"visibility_mode,omitempty"`
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
		PolicyMode:                normalizeAccountModelPolicyMode(scopeMap["policy_mode"]),
		Entries:                   normalizeAccountModelScopeEntriesAny(scopeMap["entries"]),
		SupportedProviders:        normalizeStringSliceAny(scopeMap["supported_providers"], normalizeRegistryPlatform),
		SupportedModelsByProvider: normalizeStringSliceMapAny(scopeMap["supported_models_by_provider"], normalizeRegistryID),
		AdvancedProviderOverride:  parseBoolAny(scopeMap["advanced_provider_override"]),
		SelectedModelIDs:          normalizeStringSliceAny(scopeMap["selected_model_ids"], strings.TrimSpace),
		ManualMappingRows:         normalizeManualMappingRowsAny(scopeMap["manual_mapping_rows"]),
		ManualMappings:            normalizeStringMapAny(scopeMap["manual_mappings"], strings.TrimSpace, strings.TrimSpace),
	}
	scope.normalize()
	_, hasStructuredPolicyMode := scopeMap["policy_mode"]
	_, hasStructuredEntries := scopeMap["entries"]
	if len(scope.Entries) == 0 &&
		len(scope.SupportedProviders) == 0 &&
		len(scope.SupportedModelsByProvider) == 0 &&
		len(scope.SelectedModelIDs) == 0 &&
		len(scope.ManualMappings) == 0 &&
		len(scope.ManualMappingRows) == 0 &&
		!hasStructuredPolicyMode &&
		!hasStructuredEntries {
		return nil, false
	}
	return scope, true
}

func (s *ModelRegistryService) BuildModelMappingFromScopeV2(ctx context.Context, platform string, accountType string, extra map[string]any) (map[string]string, []string, bool, error) {
	scope, ok := ExtractAccountModelScopeV2(extra)
	if !ok || scope == nil {
		return nil, nil, false, nil
	}
	if len(scope.Entries) == 0 {
		return nil, nil, false, nil
	}

	entries, err := s.pricingEntries(ctx)
	if err != nil {
		return nil, nil, false, err
	}
	index := modelregistry.BuildIndex(entries)
	routeKey := accountModelScopeRouteKey(platform, accountType)

	selectedModels := make([]string, 0)
	selectedSet := map[string]struct{}{}
	mapping := make(map[string]string, len(scope.Entries))

	for _, entry := range scope.Entries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		targetModelID := strings.TrimSpace(entry.TargetModelID)
		if displayModelID == "" {
			continue
		}
		if targetModelID == "" {
			targetModelID = displayModelID
		}

		canonicalTarget := targetModelID
		if resolved, resolveErr := s.resolveCanonicalModelForAvailability(ctx, targetModelID); resolveErr != nil {
			return nil, nil, false, resolveErr
		} else if strings.TrimSpace(resolved) != "" {
			canonicalTarget = normalizeRegistryID(resolved)
		}

		routeTarget := canonicalTarget
		if routeTarget == "" {
			routeTarget = targetModelID
		}
		if normalizedCanonical := normalizeRegistryID(canonicalTarget); normalizedCanonical != "" {
			if resolved, ok := index.ResolveProtocolID(normalizedCanonical, routeKey); ok && strings.TrimSpace(resolved) != "" {
				routeTarget = strings.TrimSpace(resolved)
			}
			if _, exists := selectedSet[normalizedCanonical]; !exists {
				selectedSet[normalizedCanonical] = struct{}{}
				selectedModels = append(selectedModels, normalizedCanonical)
			}
		}

		mapping[displayModelID] = routeTarget
	}

	if len(mapping) == 0 {
		return nil, selectedModels, true, nil
	}

	sort.Strings(selectedModels)
	return mapping, selectedModels, true, nil
}

func (s *ModelRegistryService) InferAccountModelScopeV2(ctx context.Context, platform string, _ string, mapping map[string]string) *AccountModelScopeV2 {
	if len(mapping) == 0 {
		return nil
	}

	scope := &AccountModelScopeV2{
		PolicyMode: AccountModelPolicyModeWhitelist,
		Entries:    make([]AccountModelScopeEntry, 0, len(mapping)),
	}

	keys := make([]string, 0, len(mapping))
	for from := range mapping {
		keys = append(keys, from)
	}
	sort.Strings(keys)

	for _, from := range keys {
		displayModelID := strings.TrimSpace(from)
		targetModelID := strings.TrimSpace(mapping[from])
		if displayModelID == "" || targetModelID == "" {
			continue
		}
		entry := AccountModelScopeEntry{
			DisplayModelID: displayModelID,
			TargetModelID:  targetModelID,
			Provider:       buildScopeEntryProvider(platform, targetModelID),
		}
		if displayModelID == targetModelID {
			entry.VisibilityMode = AccountModelVisibilityModeDirect
		} else {
			entry.VisibilityMode = AccountModelVisibilityModeAlias
			scope.PolicyMode = AccountModelPolicyModeMapping
		}
		if s != nil {
			if resolved, err := s.resolveCanonicalModelForAvailability(ctx, targetModelID); err == nil && strings.TrimSpace(resolved) != "" {
				entry.TargetModelID = normalizeRegistryID(resolved)
				if entry.Provider == "" {
					entry.Provider = buildScopeEntryProvider(platform, entry.TargetModelID)
				}
			}
		}
		entry.VisibilityMode = normalizeAccountModelVisibilityMode("", entry.DisplayModelID, entry.TargetModelID)
		if entry.VisibilityMode == AccountModelVisibilityModeAlias {
			scope.PolicyMode = AccountModelPolicyModeMapping
		}
		scope.Entries = append(scope.Entries, entry)
	}

	scope.normalize()
	if len(scope.Entries) == 0 {
		return nil
	}
	return scope
}

func (scope *AccountModelScopeV2) ToMap() map[string]any {
	if scope == nil {
		return nil
	}
	scope.normalize()
	entries := make([]map[string]any, 0, len(scope.Entries))
	for _, entry := range scope.Entries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		targetModelID := strings.TrimSpace(entry.TargetModelID)
		if displayModelID == "" {
			continue
		}
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		item := map[string]any{
			"display_model_id": displayModelID,
			"target_model_id":  targetModelID,
		}
		if provider := NormalizeModelProvider(entry.Provider); provider != "" {
			item["provider"] = provider
		}
		if sourceProtocol := NormalizeGatewayProtocol(entry.SourceProtocol); sourceProtocol != "" {
			item["source_protocol"] = sourceProtocol
		}
		if visibilityMode := normalizeAccountModelVisibilityMode(entry.VisibilityMode, displayModelID, targetModelID); visibilityMode != "" {
			item["visibility_mode"] = visibilityMode
		}
		entries = append(entries, item)
	}
	return map[string]any{
		"policy_mode": normalizeAccountModelPolicyMode(scope.PolicyMode),
		"entries":     entries,
	}
}

func (scope *AccountModelScopeV2) normalize() {
	if scope == nil {
		return
	}
	if len(scope.Entries) == 0 {
		scope.Entries = legacyAccountModelScopeEntries(scope)
	}
	scope.Entries = normalizeAccountModelScopeEntries(scope.Entries)
	if scope.PolicyMode == "" {
		scope.PolicyMode = inferAccountModelPolicyMode(scope.Entries)
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
	case PlatformGrok:
		return "grok"
	default:
		return normalizedPlatform
	}
}

func legacyAccountModelScopeEntries(scope *AccountModelScopeV2) []AccountModelScopeEntry {
	if scope == nil {
		return nil
	}

	ordered := make([]AccountModelScopeEntry, 0)
	appendEntry := func(entry AccountModelScopeEntry) {
		ordered = append(ordered, entry)
	}

	if len(scope.ManualMappingRows) > 0 {
		for _, row := range scope.ManualMappingRows {
			displayModelID := strings.TrimSpace(row.From)
			targetModelID := strings.TrimSpace(row.To)
			if displayModelID == "" || targetModelID == "" {
				continue
			}
			appendEntry(AccountModelScopeEntry{
				DisplayModelID: displayModelID,
				TargetModelID:  targetModelID,
				Provider:       buildScopeEntryProvider("", targetModelID),
				VisibilityMode: normalizeAccountModelVisibilityMode("", displayModelID, targetModelID),
			})
		}
		return ordered
	}

	if len(scope.ManualMappings) > 0 {
		keys := make([]string, 0, len(scope.ManualMappings))
		for from := range scope.ManualMappings {
			keys = append(keys, from)
		}
		sort.Strings(keys)
		for _, from := range keys {
			displayModelID := strings.TrimSpace(from)
			targetModelID := strings.TrimSpace(scope.ManualMappings[from])
			if displayModelID == "" || targetModelID == "" {
				continue
			}
			appendEntry(AccountModelScopeEntry{
				DisplayModelID: displayModelID,
				TargetModelID:  targetModelID,
				Provider:       buildScopeEntryProvider("", targetModelID),
				VisibilityMode: normalizeAccountModelVisibilityMode("", displayModelID, targetModelID),
			})
		}
		return ordered
	}

	if len(scope.SelectedModelIDs) > 0 {
		for _, modelID := range scope.SelectedModelIDs {
			displayModelID := strings.TrimSpace(modelID)
			if displayModelID == "" {
				continue
			}
			appendEntry(AccountModelScopeEntry{
				DisplayModelID: displayModelID,
				TargetModelID:  displayModelID,
				Provider:       buildScopeEntryProvider("", displayModelID),
				VisibilityMode: AccountModelVisibilityModeDirect,
			})
		}
		return ordered
	}

	if len(scope.SupportedModelsByProvider) == 0 {
		return nil
	}
	providers := make([]string, 0, len(scope.SupportedModelsByProvider))
	for provider := range scope.SupportedModelsByProvider {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	for _, provider := range providers {
		for _, modelID := range scope.SupportedModelsByProvider[provider] {
			displayModelID := strings.TrimSpace(modelID)
			if displayModelID == "" {
				continue
			}
			appendEntry(AccountModelScopeEntry{
				DisplayModelID: displayModelID,
				TargetModelID:  displayModelID,
				Provider:       normalizeRegistryPlatform(provider),
				VisibilityMode: AccountModelVisibilityModeDirect,
			})
		}
	}
	return ordered
}

func normalizeAccountModelPolicyMode(raw any) string {
	switch strings.TrimSpace(strings.ToLower(stringValueFromAny(raw))) {
	case AccountModelPolicyModeMapping:
		return AccountModelPolicyModeMapping
	case AccountModelPolicyModeWhitelist:
		return AccountModelPolicyModeWhitelist
	default:
		return ""
	}
}

func inferAccountModelPolicyMode(entries []AccountModelScopeEntry) string {
	for _, entry := range entries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		targetModelID := strings.TrimSpace(entry.TargetModelID)
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		if normalizeAccountModelVisibilityMode(entry.VisibilityMode, displayModelID, targetModelID) == AccountModelVisibilityModeAlias {
			return AccountModelPolicyModeMapping
		}
	}
	return AccountModelPolicyModeWhitelist
}

func normalizeAccountModelVisibilityMode(raw string, displayModelID string, targetModelID string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case AccountModelVisibilityModeAlias:
		return AccountModelVisibilityModeAlias
	case AccountModelVisibilityModeDefault:
		return AccountModelVisibilityModeDefault
	case AccountModelVisibilityModeDirect:
		return AccountModelVisibilityModeDirect
	}
	if strings.TrimSpace(displayModelID) != strings.TrimSpace(targetModelID) {
		return AccountModelVisibilityModeAlias
	}
	return AccountModelVisibilityModeDirect
}

func normalizeAccountModelScopeEntriesAny(raw any) []AccountModelScopeEntry {
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
	entries := make([]AccountModelScopeEntry, 0, len(values))
	for _, item := range values {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		displayModelID := strings.TrimSpace(stringValueFromAny(entry["display_model_id"]))
		targetModelID := strings.TrimSpace(stringValueFromAny(entry["target_model_id"]))
		if displayModelID == "" {
			continue
		}
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		entries = append(entries, AccountModelScopeEntry{
			DisplayModelID: displayModelID,
			TargetModelID:  targetModelID,
			Provider:       NormalizeModelProvider(stringValueFromAny(entry["provider"])),
			SourceProtocol: NormalizeGatewayProtocol(stringValueFromAny(entry["source_protocol"])),
			VisibilityMode: normalizeAccountModelVisibilityMode(stringValueFromAny(entry["visibility_mode"]), displayModelID, targetModelID),
		})
	}
	return normalizeAccountModelScopeEntries(entries)
}

func normalizeAccountModelScopeEntries(entries []AccountModelScopeEntry) []AccountModelScopeEntry {
	if len(entries) == 0 {
		return nil
	}
	normalized := make([]AccountModelScopeEntry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		targetModelID := strings.TrimSpace(entry.TargetModelID)
		if displayModelID == "" {
			continue
		}
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		provider := NormalizeModelProvider(entry.Provider)
		if provider == "" {
			provider = buildScopeEntryProvider("", targetModelID)
		}
		sourceProtocol := NormalizeGatewayProtocol(entry.SourceProtocol)
		visibilityMode := normalizeAccountModelVisibilityMode(entry.VisibilityMode, displayModelID, targetModelID)
		key := displayModelID + "\x00" + targetModelID + "\x00" + sourceProtocol
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, AccountModelScopeEntry{
			DisplayModelID: displayModelID,
			TargetModelID:  targetModelID,
			Provider:       provider,
			SourceProtocol: sourceProtocol,
			VisibilityMode: visibilityMode,
		})
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].DisplayModelID == normalized[j].DisplayModelID {
			if normalized[i].TargetModelID == normalized[j].TargetModelID {
				return normalized[i].SourceProtocol < normalized[j].SourceProtocol
			}
			return normalized[i].TargetModelID < normalized[j].TargetModelID
		}
		return normalized[i].DisplayModelID < normalized[j].DisplayModelID
	})
	return normalized
}

func buildScopeEntryProvider(platform string, modelID string) string {
	if provider := NormalizeModelProvider(platform); provider != "" {
		return provider
	}
	return NormalizeModelProvider(inferModelProvider(modelID))
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
