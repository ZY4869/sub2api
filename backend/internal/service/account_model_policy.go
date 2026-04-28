package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

const (
	accountModelProjectionSourceScope         = "account_scope"
	accountModelProjectionSourceLegacyMapping = "legacy_mapping"
	accountModelProjectionSourceDefault       = "default_library"

	AccountModelAvailabilityVerified    = "verified"
	AccountModelAvailabilityUnavailable = "unavailable"
	AccountModelAvailabilityUnknown     = "unknown"

	AccountModelStaleStateFresh      = "fresh"
	AccountModelStaleStateStale      = "stale"
	AccountModelStaleStateUnverified = "unverified"

	accountModelProjectionStaleTTL = 6 * time.Hour
)

type AccountModelProjection struct {
	PolicyMode string
	Explicit   bool
	Source     string
	Entries    []AccountModelProjectionEntry
}

type AccountModelProjectionEntry struct {
	DisplayModelID    string
	TargetModelID     string
	RouteModelID      string
	CanonicalID       string
	DisplayName       string
	Provider          string
	ProviderLabel     string
	SourceProtocol    string
	VisibilityMode    string
	ExposureSource    string
	AvailabilityState string
	StaleState        string
	Mode              string
	Status            string
	DeprecatedAt      string
	ReplacedBy        string
}

func BuildAccountModelProjection(ctx context.Context, account *Account, registry *ModelRegistryService) *AccountModelProjection {
	return getCachedAccountModelProjection(ctx, account, registry)
}

func buildAccountModelProjectionUncached(ctx context.Context, account *Account, registry *ModelRegistryService) *AccountModelProjection {
	if account == nil {
		return nil
	}

	scopeEntries, policyMode, explicit, source := resolveAccountModelProjectionScopeEntries(ctx, account, registry)
	if len(scopeEntries) == 0 {
		return &AccountModelProjection{
			PolicyMode: firstNonEmptyString(policyMode, AccountModelPolicyModeWhitelist),
			Explicit:   explicit,
			Source:     firstNonEmptyString(source, accountModelProjectionSourceDefault),
		}
	}

	snapshot, _ := AccountModelProbeSnapshotFromExtra(account.Extra)
	projection := &AccountModelProjection{
		PolicyMode: firstNonEmptyString(policyMode, inferAccountModelPolicyMode(scopeEntries)),
		Explicit:   explicit,
		Source:     firstNonEmptyString(source, accountModelProjectionSourceDefault),
		Entries:    make([]AccountModelProjectionEntry, 0, len(scopeEntries)),
	}

	for _, entry := range scopeEntries {
		projected := buildAccountModelProjectionEntry(ctx, account, registry, entry, projection.Source, snapshot)
		if strings.TrimSpace(projected.DisplayModelID) == "" {
			continue
		}
		projection.Entries = append(projection.Entries, projected)
	}

	projection.Entries = dedupeAccountModelProjectionEntries(projection.Entries)
	return projection
}

func resolveAccountModelProjectionScopeEntries(ctx context.Context, account *Account, registry *ModelRegistryService) ([]AccountModelScopeEntry, string, bool, string) {
	if account == nil {
		return nil, "", false, ""
	}

	if scope, ok := ExtractAccountModelScopeV2(account.Extra); ok && scope != nil {
		if !accountModelScopeUsesStructuredEntries(account.Extra) {
			compatEntries := buildLegacyAccountModelProjectionScopeEntries(ctx, account, registry, scope)
			policyMode := normalizeAccountModelPolicyMode(scope.PolicyMode)
			compatPolicyMode := inferAccountModelPolicyMode(compatEntries)
			if compatPolicyMode == AccountModelPolicyModeMapping || policyMode == "" {
				policyMode = compatPolicyMode
			}
			return compatEntries, policyMode, true, accountModelProjectionSourceScope
		}
		return append([]AccountModelScopeEntry(nil), scope.Entries...), scope.PolicyMode, true, accountModelProjectionSourceScope
	}

	if mapping := account.GetModelMapping(); len(mapping) > 0 {
		if registry != nil {
			if inferred := registry.InferAccountModelScopeV2(ctx, account.EffectiveProtocol(), account.Type, mapping); inferred != nil && len(inferred.Entries) > 0 {
				return append([]AccountModelScopeEntry(nil), inferred.Entries...), inferred.PolicyMode, true, accountModelProjectionSourceLegacyMapping
			}
		}
		entries := make([]AccountModelScopeEntry, 0, len(mapping))
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
			entries = append(entries, AccountModelScopeEntry{
				DisplayModelID: displayModelID,
				TargetModelID:  targetModelID,
				Provider:       buildScopeEntryProvider(account.EffectiveProtocol(), targetModelID),
				VisibilityMode: normalizeAccountModelVisibilityMode("", displayModelID, targetModelID),
			})
		}
		entries = normalizeAccountModelScopeEntries(entries)
		if len(entries) > 0 {
			return entries, inferAccountModelPolicyMode(entries), true, accountModelProjectionSourceLegacyMapping
		}
	}

	defaultEntries := buildDefaultAccountModelScopeEntries(ctx, account, registry)
	if len(defaultEntries) == 0 {
		return nil, AccountModelPolicyModeWhitelist, false, accountModelProjectionSourceDefault
	}
	return defaultEntries, AccountModelPolicyModeWhitelist, false, accountModelProjectionSourceDefault
}

func buildDefaultAccountModelScopeEntries(ctx context.Context, account *Account, registry *ModelRegistryService) []AccountModelScopeEntry {
	if account == nil {
		return nil
	}

	if IsProtocolGatewayAccount(account) {
		sourceProtocols := protocolGatewayTestSourceProtocols(account)
		if len(sourceProtocols) == 0 {
			sourceProtocols = GetAccountGatewayAcceptedProtocols(account)
		}
		if len(sourceProtocols) == 0 {
			return buildDefaultAccountModelScopeEntriesForSource(ctx, account, registry, "")
		}

		entries := make([]AccountModelScopeEntry, 0)
		for _, sourceProtocol := range sourceProtocols {
			protocolAccount := ResolveProtocolGatewayInboundAccount(account, sourceProtocol)
			entries = append(entries, buildDefaultAccountModelScopeEntriesForSource(ctx, protocolAccount, registry, sourceProtocol)...)
		}
		return normalizeAccountModelScopeEntries(entries)
	}

	return buildDefaultAccountModelScopeEntriesForSource(ctx, account, registry, "")
}

func buildDefaultAccountModelScopeEntriesForSource(ctx context.Context, account *Account, registry *ModelRegistryService, sourceProtocol string) []AccountModelScopeEntry {
	if account == nil {
		return nil
	}

	sourceProtocol = NormalizeGatewayProtocol(sourceProtocol)
	runtimePlatform := RoutingPlatformForAccount(account)
	if registry != nil {
		if entries, err := registry.GetModelsByPlatform(ctx, runtimePlatform, "runtime", "whitelist"); err == nil && len(entries) > 0 {
			scopeEntries := make([]AccountModelScopeEntry, 0, len(entries))
			for _, entry := range entries {
				displayModelID := strings.TrimSpace(entry.ID)
				if displayModelID == "" {
					continue
				}
				scopeEntries = append(scopeEntries, AccountModelScopeEntry{
					DisplayModelID: displayModelID,
					TargetModelID:  displayModelID,
					Provider:       firstNonEmptyString(NormalizeModelProvider(entry.Provider), NormalizeModelProvider(runtimePlatform)),
					SourceProtocol: sourceProtocol,
					VisibilityMode: AccountModelVisibilityModeDefault,
				})
			}
			return normalizeAccountModelScopeEntries(scopeEntries)
		}
	}

	if !supportsDefaultAccountModelLibrary(runtimePlatform) {
		return nil
	}
	defaults := defaultTestModelCatalog(account)
	scopeEntries := make([]AccountModelScopeEntry, 0, len(defaults))
	for _, item := range defaults {
		displayModelID := strings.TrimSpace(item.ID)
		if displayModelID == "" {
			continue
		}
		scopeEntries = append(scopeEntries, AccountModelScopeEntry{
			DisplayModelID: displayModelID,
			TargetModelID:  displayModelID,
			Provider:       firstNonEmptyString(NormalizeModelProvider(item.Provider), NormalizeModelProvider(runtimePlatform)),
			SourceProtocol: sourceProtocol,
			VisibilityMode: AccountModelVisibilityModeDefault,
		})
	}
	return normalizeAccountModelScopeEntries(scopeEntries)
}

func supportsDefaultAccountModelLibrary(platform string) bool {
	switch normalizeRegistryPlatform(platform) {
	case PlatformOpenAI, PlatformCopilot, PlatformAnthropic, PlatformKiro, PlatformDeepSeek, PlatformGemini, PlatformGrok, PlatformAntigravity, PlatformBaiduDocumentAI:
		return true
	default:
		return false
	}
}

func buildAccountModelProjectionEntry(ctx context.Context, account *Account, registry *ModelRegistryService, entry AccountModelScopeEntry, source string, snapshot *AccountModelProbeSnapshot) AccountModelProjectionEntry {
	displayModelID := strings.TrimSpace(entry.DisplayModelID)
	targetModelID := strings.TrimSpace(entry.TargetModelID)
	if targetModelID == "" {
		targetModelID = displayModelID
	}

	projected := AccountModelProjectionEntry{
		DisplayModelID:    displayModelID,
		TargetModelID:     targetModelID,
		RouteModelID:      resolveAccountProjectionRouteModelID(ctx, registry, account, targetModelID),
		CanonicalID:       normalizeRegistryID(targetModelID),
		DisplayName:       displayModelID,
		Provider:          NormalizeModelProvider(entry.Provider),
		SourceProtocol:    NormalizeGatewayProtocol(entry.SourceProtocol),
		VisibilityMode:    normalizeAccountModelVisibilityMode(entry.VisibilityMode, displayModelID, targetModelID),
		ExposureSource:    strings.TrimSpace(source),
		AvailabilityState: AccountModelAvailabilityUnknown,
		StaleState:        accountModelProjectionStaleState(snapshot),
	}

	var registryEntry *modelregistry.ModelEntry
	if registry != nil {
		if detail, err := registry.GetDetail(ctx, targetModelID); err == nil && detail != nil {
			registryEntry = &detail.ModelEntry
			projected.CanonicalID = firstNonEmptyString(normalizeRegistryID(detail.ID), projected.CanonicalID)
			projected.Provider = firstNonEmptyString(NormalizeModelProvider(detail.Provider), projected.Provider)
			projected.Status = strings.TrimSpace(detail.Status)
			projected.DeprecatedAt = strings.TrimSpace(detail.DeprecatedAt)
			projected.ReplacedBy = strings.TrimSpace(detail.ReplacedBy)
			if projected.DisplayModelID == projected.TargetModelID {
				projected.DisplayName = firstNonEmptyString(strings.TrimSpace(detail.DisplayName), FormatModelCatalogDisplayName(projected.DisplayModelID), projected.DisplayModelID)
			}
		} else if resolution, resolveErr := registry.ExplainResolution(ctx, targetModelID); resolveErr == nil && resolution != nil {
			entryValue := resolution.Entry
			if resolution.ReplacementEntry != nil && normalizeRegistryID(resolution.EffectiveID) != "" {
				entryValue = *resolution.ReplacementEntry
			}
			registryEntry = &entryValue
			projected.CanonicalID = firstNonEmptyString(normalizeRegistryID(resolution.CanonicalID), normalizeRegistryID(entryValue.ID), projected.CanonicalID)
			projected.Provider = firstNonEmptyString(NormalizeModelProvider(entryValue.Provider), projected.Provider)
			if projected.DisplayModelID == projected.TargetModelID {
				projected.DisplayName = firstNonEmptyString(strings.TrimSpace(entryValue.DisplayName), FormatModelCatalogDisplayName(projected.DisplayModelID), projected.DisplayModelID)
			}
		}
	}

	if projected.Provider == "" {
		projected.Provider = buildScopeEntryProvider(account.EffectiveProtocol(), firstNonEmptyString(projected.TargetModelID, projected.DisplayModelID))
	}
	projected.ProviderLabel = FormatProviderLabel(projected.Provider)
	projected.Mode = inferAvailableTestModelMode(firstNonEmptyString(projected.TargetModelID, projected.DisplayModelID), registryEntry)
	if projected.DisplayName == "" {
		if projected.DisplayModelID == projected.TargetModelID {
			projected.DisplayName = firstNonEmptyString(FormatModelCatalogDisplayName(projected.DisplayModelID), projected.DisplayModelID)
		} else {
			projected.DisplayName = projected.DisplayModelID
		}
	}

	if projected.RouteModelID == "" {
		projected.RouteModelID = projected.TargetModelID
	}
	projected.AvailabilityState, projected.StaleState = accountModelProjectionSnapshotState(snapshot, projected)
	return projected
}

func resolveAccountProjectionRouteModelID(ctx context.Context, registry *ModelRegistryService, account *Account, targetModelID string) string {
	targetModelID = strings.TrimSpace(targetModelID)
	if targetModelID == "" {
		return ""
	}
	if registry == nil || account == nil {
		return targetModelID
	}
	if resolved, ok, err := registry.ResolveProtocolModel(ctx, targetModelID, registryRouteForAccount(account)); err == nil && ok && strings.TrimSpace(resolved) != "" {
		return strings.TrimSpace(resolved)
	}
	return targetModelID
}

func dedupeAccountModelProjectionEntries(entries []AccountModelProjectionEntry) []AccountModelProjectionEntry {
	if len(entries) == 0 {
		return nil
	}
	deduped := make(map[string]AccountModelProjectionEntry, len(entries))
	for _, entry := range entries {
		displayModelID := strings.TrimSpace(entry.DisplayModelID)
		if displayModelID == "" || strings.Contains(displayModelID, "*") {
			continue
		}
		existing, ok := deduped[displayModelID]
		if !ok || compareAccountModelProjectionEntry(entry, existing) < 0 {
			deduped[displayModelID] = entry
		}
	}
	result := make([]AccountModelProjectionEntry, 0, len(deduped))
	for _, entry := range deduped {
		result = append(result, entry)
	}
	sort.SliceStable(result, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(result[i].DisplayName))
		rightName := strings.ToLower(strings.TrimSpace(result[j].DisplayName))
		if leftName != rightName {
			return leftName < rightName
		}
		return result[i].DisplayModelID < result[j].DisplayModelID
	})
	return result
}

func compareAccountModelProjectionEntry(left AccountModelProjectionEntry, right AccountModelProjectionEntry) int {
	leftAlias := left.VisibilityMode == AccountModelVisibilityModeAlias
	rightAlias := right.VisibilityMode == AccountModelVisibilityModeAlias
	switch {
	case leftAlias && !rightAlias:
		return -1
	case !leftAlias && rightAlias:
		return 1
	}
	if left.StaleState != right.StaleState {
		return accountModelStaleStateRank(left.StaleState) - accountModelStaleStateRank(right.StaleState)
	}
	if left.AvailabilityState != right.AvailabilityState {
		return accountModelAvailabilityRank(left.AvailabilityState) - accountModelAvailabilityRank(right.AvailabilityState)
	}
	if left.TargetModelID != right.TargetModelID {
		return strings.Compare(left.TargetModelID, right.TargetModelID)
	}
	return strings.Compare(left.DisplayModelID, right.DisplayModelID)
}

func accountModelAvailabilityRank(value string) int {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case AccountModelAvailabilityVerified:
		return 0
	case AccountModelAvailabilityUnknown:
		return 1
	default:
		return 2
	}
}

func accountModelStaleStateRank(value string) int {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case AccountModelStaleStateFresh:
		return 0
	case AccountModelStaleStateStale:
		return 1
	default:
		return 2
	}
}

func normalizeAccountModelAvailabilityState(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case AccountModelAvailabilityVerified:
		return AccountModelAvailabilityVerified
	case AccountModelAvailabilityUnavailable:
		return AccountModelAvailabilityUnavailable
	case AccountModelAvailabilityUnknown:
		return AccountModelAvailabilityUnknown
	default:
		return ""
	}
}

func normalizeAccountModelStaleState(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case AccountModelStaleStateFresh:
		return AccountModelStaleStateFresh
	case AccountModelStaleStateStale:
		return AccountModelStaleStateStale
	case AccountModelStaleStateUnverified:
		return AccountModelStaleStateUnverified
	default:
		return ""
	}
}

func accountModelProjectionSnapshotState(snapshot *AccountModelProbeSnapshot, entry AccountModelProjectionEntry) (string, string) {
	if snapshot == nil {
		return AccountModelAvailabilityUnknown, AccountModelStaleStateUnverified
	}
	if snapshotEntry, ok := accountModelProjectionSnapshotEntryState(snapshot, entry); ok {
		return snapshotEntry.AvailabilityState, snapshotEntry.StaleState
	}

	staleState := accountModelProjectionStaleState(snapshot)
	if staleState == AccountModelStaleStateUnverified {
		return AccountModelAvailabilityUnknown, staleState
	}
	if accountModelSnapshotContainsAny(snapshot,
		entry.DisplayModelID,
		entry.TargetModelID,
		entry.RouteModelID,
		entry.CanonicalID,
	) {
		return AccountModelAvailabilityVerified, staleState
	}
	return AccountModelAvailabilityUnavailable, staleState
}

func accountModelProjectionStaleState(snapshot *AccountModelProbeSnapshot) string {
	if snapshot == nil {
		return AccountModelStaleStateUnverified
	}
	if strings.EqualFold(strings.TrimSpace(snapshot.Source), AccountModelProbeSnapshotSourceModelScopePreview) || strings.EqualFold(strings.TrimSpace(snapshot.ProbeSource), AccountModelProbeSnapshotSourceModelScopePreview) {
		return AccountModelStaleStateUnverified
	}
	if updatedAt, ok := parseSnapshotUpdatedAt(snapshot.UpdatedAt); ok {
		if time.Since(updatedAt) > accountModelProjectionStaleTTL {
			return AccountModelStaleStateStale
		}
		return AccountModelStaleStateFresh
	}
	return AccountModelStaleStateStale
}

func accountModelSnapshotContainsAny(snapshot *AccountModelProbeSnapshot, values ...string) bool {
	if snapshot == nil {
		return false
	}
	seen := make(map[string]struct{})
	for _, modelID := range snapshotModelIDsForAvailableTestModels(snapshot) {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		seen[modelID] = struct{}{}
		if normalized := normalizeRegistryID(modelID); normalized != "" {
			seen[normalized] = struct{}{}
		}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			return true
		}
		if normalized := normalizeRegistryID(value); normalized != "" {
			if _, ok := seen[normalized]; ok {
				return true
			}
		}
	}
	return false
}

func accountModelProjectionSnapshotEntryState(snapshot *AccountModelProbeSnapshot, entry AccountModelProjectionEntry) (AccountModelProbeSnapshotEntry, bool) {
	if snapshot == nil || len(snapshot.Entries) == 0 {
		return AccountModelProbeSnapshotEntry{}, false
	}
	bestRank := 99
	best := AccountModelProbeSnapshotEntry{}
	for _, candidate := range snapshot.Entries {
		rank, ok := accountModelProbeSnapshotEntryMatchRank(candidate, entry)
		if !ok || rank >= bestRank {
			continue
		}
		bestRank = rank
		best = candidate
	}
	if bestRank == 99 {
		return AccountModelProbeSnapshotEntry{}, false
	}
	availabilityState := normalizeAccountModelAvailabilityState(best.AvailabilityState)
	if availabilityState == "" {
		availabilityState = AccountModelAvailabilityUnknown
	}
	staleState := normalizeAccountModelStaleState(best.StaleState)
	if staleState == "" {
		staleState = accountModelProjectionStaleState(snapshot)
	}
	best.AvailabilityState = availabilityState
	best.StaleState = staleState
	return best, true
}

func accountModelProbeSnapshotEntryMatchRank(candidate AccountModelProbeSnapshotEntry, entry AccountModelProjectionEntry) (int, bool) {
	displayModelID := strings.TrimSpace(candidate.DisplayModelID)
	targetModelID := strings.TrimSpace(candidate.TargetModelID)
	entryDisplayID := strings.TrimSpace(entry.DisplayModelID)

	if displayModelID != "" && entryDisplayID != "" && accountModelIDsEqual(displayModelID, entryDisplayID) {
		return 0, true
	}
	for _, target := range []string{entry.TargetModelID, entry.RouteModelID, entry.CanonicalID} {
		if targetModelID != "" && accountModelIDsEqual(targetModelID, target) {
			return 1, true
		}
		if displayModelID != "" && accountModelIDsEqual(displayModelID, target) {
			return 2, true
		}
	}
	return 0, false
}

func accountModelIDsEqual(left string, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	return normalizeRegistryID(left) != "" && normalizeRegistryID(left) == normalizeRegistryID(right)
}

func parseSnapshotUpdatedAt(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed, true
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed, true
	}
	return time.Time{}, false
}
