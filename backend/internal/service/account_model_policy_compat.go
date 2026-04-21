package service

import (
	"context"
	"sort"
	"strings"
)

func accountModelScopeUsesStructuredEntries(extra map[string]any) bool {
	if len(extra) == 0 {
		return false
	}
	rawScope, ok := extra["model_scope_v2"]
	if !ok || rawScope == nil {
		return false
	}
	scopeMap, ok := rawScope.(map[string]any)
	if !ok {
		return false
	}
	_, hasEntries := scopeMap["entries"]
	return hasEntries
}

func buildLegacyAccountModelProjectionScopeEntries(ctx context.Context, account *Account, registry *ModelRegistryService, scope *AccountModelScopeV2) []AccountModelScopeEntry {
	if scope == nil {
		return nil
	}

	baseEntries := normalizeAccountModelScopeEntries(append([]AccountModelScopeEntry(nil), scope.Entries...))
	mapping := account.GetModelMapping()
	if len(baseEntries) == 0 || len(mapping) == 0 {
		return baseEntries
	}

	keys := make([]string, 0, len(mapping))
	for alias := range mapping {
		keys = append(keys, alias)
	}
	sort.Strings(keys)

	compatEntries := make([]AccountModelScopeEntry, 0, len(baseEntries)+len(keys))
	for _, entry := range baseEntries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		targetModelID := strings.TrimSpace(entry.TargetModelID)
		if targetModelID == "" {
			targetModelID = displayModelID
		}
		if displayModelID == "" {
			continue
		}
		if normalizeAccountModelVisibilityMode(entry.VisibilityMode, displayModelID, targetModelID) == AccountModelVisibilityModeAlias {
			compatEntries = append(compatEntries, entry)
			continue
		}

		overlayEntries := buildLegacyProjectionAliasOverlayEntries(ctx, account, registry, entry, keys, mapping)
		if len(overlayEntries) > 0 {
			compatEntries = append(compatEntries, overlayEntries...)
			continue
		}
		compatEntries = append(compatEntries, entry)
	}
	return normalizeAccountModelScopeEntries(compatEntries)
}

func buildLegacyProjectionAliasOverlayEntries(ctx context.Context, account *Account, registry *ModelRegistryService, scopeEntry AccountModelScopeEntry, orderedAliases []string, mapping map[string]string) []AccountModelScopeEntry {
	if len(orderedAliases) == 0 {
		return nil
	}

	displayModelID := strings.TrimSpace(scopeEntry.DisplayModelID)
	targetModelID := strings.TrimSpace(scopeEntry.TargetModelID)
	if targetModelID == "" {
		targetModelID = displayModelID
	}
	if targetModelID == "" {
		return nil
	}

	overlays := make([]AccountModelScopeEntry, 0)
	for _, alias := range orderedAliases {
		alias = strings.TrimSpace(alias)
		mappedTargetID := strings.TrimSpace(mapping[alias])
		if alias == "" || mappedTargetID == "" {
			continue
		}
		if !legacyProjectionMappingMatchesScopeTarget(ctx, registry, account, targetModelID, mappedTargetID) {
			continue
		}
		overlays = append(overlays, AccountModelScopeEntry{
			DisplayModelID: alias,
			TargetModelID:  mappedTargetID,
			Provider:       firstNonEmptyString(NormalizeModelProvider(scopeEntry.Provider), buildScopeEntryProvider(account.EffectiveProtocol(), mappedTargetID)),
			SourceProtocol: NormalizeGatewayProtocol(scopeEntry.SourceProtocol),
			VisibilityMode: normalizeAccountModelVisibilityMode("", alias, mappedTargetID),
		})
	}
	return overlays
}

func legacyProjectionMappingMatchesScopeTarget(ctx context.Context, registry *ModelRegistryService, account *Account, scopeTargetModelID string, mappedTargetModelID string) bool {
	if accountModelIDsEqual(scopeTargetModelID, mappedTargetModelID) {
		return true
	}
	route := registryRouteForAccount(account)
	scopeVariants := collectModelSupportVariants(ctx, registry, route, scopeTargetModelID)
	mappedVariants := collectModelSupportVariants(ctx, registry, route, mappedTargetModelID)
	for candidate := range scopeVariants {
		if _, ok := mappedVariants[candidate]; ok {
			return true
		}
	}
	return false
}
