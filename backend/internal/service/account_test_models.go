package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

type AvailableTestModel struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	DisplayName    string `json:"display_name"`
	CreatedAt      string `json:"created_at"`
	CanonicalID    string `json:"canonical_id,omitempty"`
	Provider       string `json:"provider,omitempty"`
	ProviderLabel  string `json:"provider_label,omitempty"`
	SourceProtocol string `json:"source_protocol,omitempty"`
	Status         string `json:"status,omitempty"`
	DeprecatedAt   string `json:"deprecated_at,omitempty"`
	ReplacedBy     string `json:"replaced_by,omitempty"`
}

type testModelCandidate struct {
	model      AvailableTestModel
	source     string
	uiPriority int
}

func BuildAvailableTestModels(ctx context.Context, account *Account, registry *ModelRegistryService) []AvailableTestModel {
	if account == nil {
		return []AvailableTestModel{}
	}
	sourceProtocols := protocolGatewayTestSourceProtocols(account)
	if len(sourceProtocols) == 0 {
		return buildAvailableTestModelsForSource(ctx, account, registry, "")
	}

	candidates := make([]testModelCandidate, 0)
	resolutionEntries := []modelregistry.ModelEntry{}
	for _, sourceProtocol := range sourceProtocols {
		protocolAccount := ResolveProtocolGatewayInboundAccount(account, sourceProtocol)
		sourceCandidates, sourceEntries := buildAvailableTestModelCandidatesForSource(ctx, protocolAccount, registry, sourceProtocol)
		candidates = append(candidates, sourceCandidates...)
		if len(sourceEntries) > 0 {
			resolutionEntries = sourceEntries
		}
	}
	return filterAvailableTestModelsForAccount(
		ctx,
		account,
		registry,
		dedupeAndSortAvailableTestModels(candidates, resolutionEntries),
	)
}

func MergeAvailableTestModels(groups ...[]AvailableTestModel) []AvailableTestModel {
	if len(groups) == 0 {
		return []AvailableTestModel{}
	}

	merged := make([]AvailableTestModel, 0)
	for _, group := range groups {
		merged = append(merged, group...)
	}
	if len(merged) == 0 {
		return []AvailableTestModel{}
	}

	deduped := make(map[string]AvailableTestModel, len(merged))
	for _, model := range merged {
		canonicalID := normalizeRegistryID(model.CanonicalID)
		if canonicalID == "" {
			canonicalID = normalizeRegistryID(model.ID)
		}
		key := testModelDedupeKey(canonicalID, model.SourceProtocol)
		existing, ok := deduped[key]
		if !ok || compareAvailableTestModels(model, existing) < 0 {
			deduped[key] = model
		}
	}

	result := make([]AvailableTestModel, 0, len(deduped))
	for _, model := range deduped {
		result = append(result, model)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return compareAvailableTestModels(result[i], result[j]) < 0
	})
	return result
}

func IntersectAvailableTestModels(groups ...[]AvailableTestModel) []AvailableTestModel {
	if len(groups) == 0 {
		return []AvailableTestModel{}
	}

	keyCounts := make(map[string]int)
	bestByKey := make(map[string]AvailableTestModel)

	for idx, group := range groups {
		if len(group) == 0 {
			return []AvailableTestModel{}
		}
		perGroup := make(map[string]AvailableTestModel, len(group))
		for _, model := range group {
			canonicalID := normalizeRegistryID(model.CanonicalID)
			if canonicalID == "" {
				canonicalID = normalizeRegistryID(model.ID)
			}
			key := testModelDedupeKey(canonicalID, model.SourceProtocol)
			existing, ok := perGroup[key]
			if !ok || compareAvailableTestModels(model, existing) < 0 {
				perGroup[key] = model
			}
		}
		if idx == 0 {
			for key, model := range perGroup {
				keyCounts[key] = 1
				bestByKey[key] = model
			}
			continue
		}
		for key, model := range perGroup {
			if keyCounts[key] != idx {
				continue
			}
			keyCounts[key]++
			if existing, ok := bestByKey[key]; !ok || compareAvailableTestModels(model, existing) < 0 {
				bestByKey[key] = model
			}
		}
	}

	result := make([]AvailableTestModel, 0, len(bestByKey))
	for key, count := range keyCounts {
		if count != len(groups) {
			continue
		}
		result = append(result, bestByKey[key])
	}
	sort.SliceStable(result, func(i, j int) bool {
		return compareAvailableTestModels(result[i], result[j]) < 0
	})
	return result
}

func buildAvailableTestModelsForSource(ctx context.Context, account *Account, registry *ModelRegistryService, sourceProtocol string) []AvailableTestModel {
	candidates, resolutionEntries := buildAvailableTestModelCandidatesForSource(ctx, account, registry, sourceProtocol)
	return filterAvailableTestModelsForAccount(
		ctx,
		account,
		registry,
		dedupeAndSortAvailableTestModels(candidates, resolutionEntries),
	)
}

func buildAvailableTestModelCandidatesForSource(ctx context.Context, account *Account, registry *ModelRegistryService, sourceProtocol string) ([]testModelCandidate, []modelregistry.ModelEntry) {
	candidates, resolutionEntries := buildRegistryTestModelCandidates(ctx, account, registry, sourceProtocol)
	if len(candidates) == 0 {
		candidates, resolutionEntries = buildFallbackTestModelCandidates(ctx, account, registry, sourceProtocol)
	}
	candidates = append(candidates, buildManualTestModelCandidates(account, sourceProtocol)...)
	candidates = filterChatGPTOpenAIKnownTestModelCandidates(account, sourceProtocol, candidates)
	return candidates, resolutionEntries
}

func buildRegistryTestModelCandidates(ctx context.Context, account *Account, registry *ModelRegistryService, sourceProtocol string) ([]testModelCandidate, []modelregistry.ModelEntry) {
	if registry == nil {
		return nil, modelregistry.SeedModels()
	}
	runtimePlatform := RoutingPlatformForAccount(account)
	visibleGrokModels := GrokVisibleModelIDsForAccount(account)
	visibleGrokSet := map[string]struct{}{}
	if runtimePlatform == PlatformGrok && account != nil && account.IsGrokSSO() {
		for _, modelID := range visibleGrokModels {
			if normalized := normalizeRegistryID(modelID); normalized != "" {
				visibleGrokSet[normalized] = struct{}{}
			}
		}
	}

	details, err := registry.adminDetails(ctx)
	if err != nil {
		return nil, modelregistry.SeedModels()
	}

	availableCanonicals := make(map[string]struct{}, len(details))
	for _, detail := range details {
		if !detail.Available {
			continue
		}
		canonicalID := NormalizeModelCatalogModelID(detail.ID)
		if canonicalID == "" {
			canonicalID = normalizeRegistryID(detail.ID)
		}
		if canonicalID != "" {
			availableCanonicals[canonicalID] = struct{}{}
		}
	}

	resolutionEntries := make([]modelregistry.ModelEntry, 0, len(details))
	candidates := make([]testModelCandidate, 0, len(details))
	for _, detail := range details {
		if detail.Hidden || detail.Tombstoned {
			continue
		}
		if !isRegistryDetailAvailableForTestSelection(detail, availableCanonicals) {
			continue
		}
		resolutionEntries = append(resolutionEntries, detail.ModelEntry)
		if !modelregistry.SupportsPlatform(detail.ModelEntry, runtimePlatform) {
			continue
		}
		if !isDirectPlatformTestModelAllowed(account, detail.ModelEntry) {
			continue
		}
		if !modelregistry.HasExposure(detail.ModelEntry, "test") {
			continue
		}
		if len(visibleGrokSet) > 0 {
			if _, ok := visibleGrokSet[normalizeRegistryID(detail.ID)]; !ok {
				continue
			}
		}
		candidates = append(candidates, testModelCandidate{
			model:      buildAvailableTestModelFromRegistryDetail(detail, sourceProtocol),
			source:     strings.TrimSpace(detail.Source),
			uiPriority: detail.UIPriority,
		})
	}
	if len(visibleGrokSet) > 0 {
		seen := make(map[string]struct{}, len(candidates))
		for _, candidate := range candidates {
			seen[normalizeRegistryID(candidate.model.ID)] = struct{}{}
		}
		for _, modelID := range visibleGrokModels {
			normalized := normalizeRegistryID(modelID)
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			candidates = append(candidates, testModelCandidate{
				model: applyAvailableTestModelProvider(AvailableTestModel{
					ID:             modelID,
					Type:           "model",
					DisplayName:    firstNonEmptyTestModelLabel(FormatModelCatalogDisplayName(modelID), modelID),
					SourceProtocol: normalizeTestSourceProtocol(sourceProtocol),
					Status:         "stable",
				}, PlatformGrok),
				source:     "runtime",
				uiPriority: fallbackTestModelPriority(modelID),
			})
		}
	}
	return candidates, resolutionEntries
}

func isDirectPlatformTestModelAllowed(account *Account, entry modelregistry.ModelEntry) bool {
	if account == nil || IsProtocolGatewayAccount(account) {
		return true
	}
	provider := normalizedDirectTestModelProvider(entry.Provider)
	if provider == "" {
		return true
	}

	switch RoutingPlatformForAccount(account) {
	case PlatformOpenAI, PlatformCopilot:
		return provider == PlatformOpenAI
	case PlatformAnthropic, PlatformKiro:
		return provider == PlatformAnthropic
	case PlatformGemini:
		return provider == PlatformGemini
	case PlatformGrok:
		return provider == PlatformGrok
	case PlatformAntigravity:
		return true
	default:
		return true
	}
}

func normalizedDirectTestModelProvider(provider string) string {
	normalized := modelregistry.NormalizePlatform(strings.TrimSpace(provider))
	switch normalized {
	case "claude":
		return PlatformAnthropic
	default:
		return normalized
	}
}

func isRegistryDetailAvailableForTestSelection(detail modelregistry.AdminModelDetail, availableCanonicals map[string]struct{}) bool {
	if detail.Available {
		return true
	}
	if !strings.EqualFold(strings.TrimSpace(detail.Status), "deprecated") {
		return false
	}
	canonicalID := NormalizeModelCatalogModelID(detail.ReplacedBy)
	if canonicalID == "" {
		return false
	}
	_, ok := availableCanonicals[canonicalID]
	return ok
}

func buildFallbackTestModelCandidates(ctx context.Context, account *Account, registry *ModelRegistryService, sourceProtocol string) ([]testModelCandidate, []modelregistry.ModelEntry) {
	resolutionEntries := modelregistry.SeedModels()
	metadata := map[string]modelregistry.AdminModelDetail{}
	if registry != nil {
		if details, err := registry.adminDetails(ctx); err == nil {
			resolutionEntries = make([]modelregistry.ModelEntry, 0, len(details))
			for _, detail := range details {
				if detail.Hidden || detail.Tombstoned || !detail.Available {
					continue
				}
				resolutionEntries = append(resolutionEntries, detail.ModelEntry)
				metadata[normalizeRegistryID(detail.ID)] = detail
			}
		}
	}

	items := defaultTestModelCatalog(account)
	candidates := make([]testModelCandidate, 0, len(items))
	for _, item := range items {
		detail, ok := metadata[normalizeRegistryID(item.ID)]
		status := "stable"
		deprecatedAt := ""
		replacedBy := ""
		uiPriority := fallbackTestModelPriority(item.ID)
		source := "fallback"
		if ok {
			status = strings.TrimSpace(detail.Status)
			deprecatedAt = strings.TrimSpace(detail.DeprecatedAt)
			replacedBy = strings.TrimSpace(detail.ReplacedBy)
			uiPriority = detail.UIPriority
			source = strings.TrimSpace(detail.Source)
		}
		item.SourceProtocol = normalizeTestSourceProtocol(sourceProtocol)
		item.Status = status
		item.DeprecatedAt = deprecatedAt
		item.ReplacedBy = replacedBy
		item = applyAvailableTestModelProvider(item, inferAvailableTestModelProvider(account, sourceProtocol))
		candidates = append(candidates, testModelCandidate{
			model:      item,
			source:     source,
			uiPriority: uiPriority,
		})
	}
	return candidates, resolutionEntries
}

func buildManualTestModelCandidates(account *Account, sourceProtocol string) []testModelCandidate {
	manualModels := AccountManualModelsFromExtra(account.Extra, IsProtocolGatewayAccount(account))
	if len(manualModels) == 0 {
		return nil
	}
	normalizedSourceProtocol := normalizeTestSourceProtocol(sourceProtocol)
	candidates := make([]testModelCandidate, 0, len(manualModels))
	for _, manualModel := range manualModels {
		modelID := strings.TrimSpace(manualModel.ModelID)
		if modelID == "" {
			continue
		}
		manualProtocol := normalizeTestSourceProtocol(manualModel.SourceProtocol)
		if normalizedSourceProtocol != "" && manualProtocol != "" && manualProtocol != normalizedSourceProtocol {
			continue
		}
		if manualProtocol == "" {
			manualProtocol = normalizedSourceProtocol
		}
		provider := NormalizeModelProvider(manualModel.Provider)
		if provider == "" {
			provider = inferAvailableTestModelProvider(account, manualProtocol)
		}
		candidates = append(candidates, testModelCandidate{
			model: applyAvailableTestModelProvider(AvailableTestModel{
				ID:             modelID,
				Type:           "model",
				DisplayName:    firstNonEmptyTestModelLabel(FormatModelCatalogDisplayName(modelID), modelID),
				SourceProtocol: manualProtocol,
				Status:         "manual",
			}, provider),
			source:     "manual",
			uiPriority: -10,
		})
	}
	return candidates
}

func dedupeAndSortAvailableTestModels(candidates []testModelCandidate, resolutionEntries []modelregistry.ModelEntry) []AvailableTestModel {
	if len(candidates) == 0 {
		return []AvailableTestModel{}
	}

	indexEntries := resolutionEntries
	if len(indexEntries) == 0 {
		indexEntries = modelregistry.SeedModels()
	}
	index := modelregistry.BuildIndex(indexEntries)

	grouped := make(map[string][]testModelCandidate, len(candidates))
	for _, candidate := range candidates {
		canonicalID := normalizeRegistryID(candidate.model.ID)
		if resolved, ok := index.ResolveCanonicalID(candidate.model.ID); ok && resolved != "" {
			canonicalID = resolved
		}
		candidate.model.CanonicalID = canonicalID
		candidate.model = applyAvailableTestModelProvider(candidate.model, candidate.model.Provider)
		grouped[testModelDedupeKey(canonicalID, candidate.model.SourceProtocol)] = append(
			grouped[testModelDedupeKey(canonicalID, candidate.model.SourceProtocol)],
			candidate,
		)
	}

	deduped := make([]AvailableTestModel, 0, len(grouped))
	for _, group := range grouped {
		sort.SliceStable(group, func(i, j int) bool {
			return compareTestModelCandidates(group[i], group[j]) < 0
		})
		deduped = append(deduped, group[0].model)
	}

	sort.SliceStable(deduped, func(i, j int) bool {
		return compareAvailableTestModels(deduped[i], deduped[j]) < 0
	})
	return deduped
}

func filterAvailableTestModelsForAccount(
	ctx context.Context,
	account *Account,
	registry *ModelRegistryService,
	models []AvailableTestModel,
) []AvailableTestModel {
	if len(models) == 0 || account == nil || !accountHasExplicitModelRestrictions(account) {
		return models
	}

	filtered := make([]AvailableTestModel, 0, len(models))
	for _, model := range models {
		requestedIDs := []string{
			strings.TrimSpace(model.ID),
			strings.TrimSpace(model.CanonicalID),
		}
		allowed := false
		for _, requestedID := range requestedIDs {
			if requestedID == "" {
				continue
			}
			if isRequestedModelSupportedByAccount(ctx, registry, account, requestedID) {
				allowed = true
				break
			}
		}
		if allowed {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func compareTestModelCandidates(left testModelCandidate, right testModelCandidate) int {
	if left.model.CanonicalID == left.model.ID && right.model.CanonicalID != right.model.ID {
		return -1
	}
	if right.model.CanonicalID == right.model.ID && left.model.CanonicalID != left.model.ID {
		return 1
	}
	if isDeprecatedTestModel(left.model) != isDeprecatedTestModel(right.model) {
		if !isDeprecatedTestModel(left.model) {
			return -1
		}
		return 1
	}
	if sourceRank := compareTestModelSource(left.source, right.source); sourceRank != 0 {
		return sourceRank
	}
	if protocolRank := compareSourceProtocol(left.model.SourceProtocol, right.model.SourceProtocol); protocolRank != 0 {
		return protocolRank
	}
	if left.uiPriority != right.uiPriority {
		return left.uiPriority - right.uiPriority
	}
	if len(left.model.ID) != len(right.model.ID) {
		return len(left.model.ID) - len(right.model.ID)
	}
	return strings.Compare(left.model.ID, right.model.ID)
}

func compareAvailableTestModels(left AvailableTestModel, right AvailableTestModel) int {
	if isDeprecatedTestModel(left) != isDeprecatedTestModel(right) {
		if !isDeprecatedTestModel(left) {
			return -1
		}
		return 1
	}
	leftPriority := fallbackTestModelPriority(left.ID)
	rightPriority := fallbackTestModelPriority(right.ID)
	if leftPriority != rightPriority {
		return leftPriority - rightPriority
	}
	leftSortLabel := FinalDisplayNameSortKey(
		left.Provider,
		left.ProviderLabel,
		firstNonEmptyTestModelLabel(left.DisplayName, left.ID),
		left.ID,
	)
	rightSortLabel := FinalDisplayNameSortKey(
		right.Provider,
		right.ProviderLabel,
		firstNonEmptyTestModelLabel(right.DisplayName, right.ID),
		right.ID,
	)
	if leftSortLabel != rightSortLabel {
		return strings.Compare(leftSortLabel, rightSortLabel)
	}
	if protocolRank := compareSourceProtocol(left.SourceProtocol, right.SourceProtocol); protocolRank != 0 {
		return protocolRank
	}
	return strings.Compare(left.ID, right.ID)
}

func compareTestModelSource(left string, right string) int {
	return testModelSourceRank(left) - testModelSourceRank(right)
}

func testModelSourceRank(source string) int {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "seed":
		return 0
	case "runtime":
		return 1
	default:
		return 2
	}
}

func compareSourceProtocol(left string, right string) int {
	return sourceProtocolRank(left) - sourceProtocolRank(right)
}

func sourceProtocolRank(sourceProtocol string) int {
	switch normalizeTestSourceProtocol(sourceProtocol) {
	case PlatformOpenAI:
		return 0
	case PlatformAnthropic:
		return 1
	case PlatformGemini:
		return 2
	default:
		return 3
	}
}

func testModelDedupeKey(canonicalID string, sourceProtocol string) string {
	sourceProtocol = normalizeTestSourceProtocol(sourceProtocol)
	if sourceProtocol == "" {
		return canonicalID
	}
	return canonicalID + "::" + sourceProtocol
}

func normalizeTestSourceProtocol(sourceProtocol string) string {
	switch NormalizeGatewayProtocol(sourceProtocol) {
	case PlatformOpenAI, PlatformAnthropic, PlatformGemini:
		return NormalizeGatewayProtocol(sourceProtocol)
	default:
		return ""
	}
}

func protocolGatewayTestSourceProtocols(account *Account) []string {
	if !IsProtocolGatewayAccount(account) {
		return nil
	}
	acceptedProtocols := GetAccountGatewayAcceptedProtocols(account)
	result := make([]string, 0, len(acceptedProtocols))
	for _, protocol := range acceptedProtocols {
		if normalized := normalizeTestSourceProtocol(protocol); normalized != "" {
			result = append(result, normalized)
		}
	}
	return result
}

func isDeprecatedTestModel(model AvailableTestModel) bool {
	return strings.EqualFold(strings.TrimSpace(model.Status), "deprecated")
}

func fallbackTestModelPriority(modelID string) int {
	if entry, ok := modelregistry.SeedModelByID(modelID); ok && entry.UIPriority > 0 {
		return entry.UIPriority
	}
	return 5000
}

func firstNonEmptyTestModelLabel(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func inferAvailableTestModelProvider(account *Account, sourceProtocol string) string {
	if normalized := NormalizeModelProvider(sourceProtocol); normalized != "" {
		return normalized
	}
	if account == nil {
		return ""
	}
	return ProviderForPlatform(RoutingPlatformForAccount(account))
}

func applyAvailableTestModelProvider(model AvailableTestModel, provider string) AvailableTestModel {
	normalized := NormalizeModelProvider(provider)
	if normalized == "" {
		normalized = NormalizeModelProvider(model.Provider)
	}
	if normalized == "" {
		normalized = NormalizeModelProvider(model.SourceProtocol)
	}
	model.Provider = normalized
	if normalized != "" {
		model.ProviderLabel = FormatProviderLabel(normalized)
	}
	return model
}

func defaultTestModelCatalog(account *Account) []AvailableTestModel {
	if account == nil {
		return []AvailableTestModel{}
	}

	switch RoutingPlatformForAccount(account) {
	case PlatformKiro:
		items := KiroBuiltinModelCatalog()
		result := make([]AvailableTestModel, 0, len(items))
		for _, item := range items {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				CreatedAt:   item.CreatedAt,
				Status:      "stable",
			})
		}
		return decorateDefaultTestModels(result, PlatformKiro)
	case PlatformGemini:
		result := make([]AvailableTestModel, 0, len(geminicli.DefaultModels))
		for _, item := range geminicli.DefaultModels {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				CreatedAt:   item.CreatedAt,
				Status:      "stable",
			})
		}
		return decorateDefaultTestModels(result, PlatformGemini)
	case PlatformAntigravity:
		items := antigravity.DefaultModels()
		result := make([]AvailableTestModel, 0, len(items))
		for _, item := range items {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				CreatedAt:   item.CreatedAt,
				Status:      "stable",
			})
		}
		return decorateDefaultTestModels(result, PlatformAntigravity)
	case PlatformOpenAI, PlatformCopilot:
		result := make([]AvailableTestModel, 0, len(openai.DefaultModels))
		for _, item := range openai.DefaultModels {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				Status:      "stable",
			})
		}
		return decorateDefaultTestModels(result, ProviderForPlatform(RoutingPlatformForAccount(account)))
	case PlatformGrok:
		models := GrokVisibleModelIDsForAccount(account)
		result := make([]AvailableTestModel, 0, len(models))
		for _, modelID := range models {
			result = append(result, AvailableTestModel{
				ID:          modelID,
				Type:        "model",
				DisplayName: modelID,
				Status:      "stable",
			})
		}
		return decorateDefaultTestModels(result, PlatformGrok)
	default:
		result := make([]AvailableTestModel, 0, len(claude.DefaultModels))
		for _, item := range claude.DefaultModels {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				CreatedAt:   item.CreatedAt,
				Status:      "stable",
			})
		}
		return decorateDefaultTestModels(result, ProviderForPlatform(RoutingPlatformForAccount(account)))
	}
}

func decorateDefaultTestModels(items []AvailableTestModel, provider string) []AvailableTestModel {
	for index := range items {
		items[index] = applyAvailableTestModelProvider(items[index], provider)
	}
	return items
}
