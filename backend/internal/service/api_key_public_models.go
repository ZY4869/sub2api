package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"go.uber.org/zap"
)

type APIKeyPublicModelEntry struct {
	PublicID          string
	AliasID           string
	SourceID          string
	DisplayName       string
	Platform          string
	AvailabilityState string
	StaleState        string
	LifecycleStatus   string
}

type apiKeyPublicProjectionCandidate struct {
	MatchID     string
	AliasID     string
	SourceID    string
	DisplayName string
	Platform    string
	ExposeAlias bool
}

const apiKeyPublicModelsSourcePolicyProjection = "policy_projection"

type apiKeyPublishedPublicCatalogMatch struct {
	Entry      APIKeyPublicModelEntry
	Catalog    *PublishedPublicCatalogEntry
	Binding    APIKeyGroupBinding
	GroupID    *int64
	Account    *Account
	SourceItem PublicModelCatalogItem
}

func (s *GatewayService) GetAPIKeyPublicModels(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
) ([]APIKeyPublicModelEntry, error) {
	if s == nil || s.accountRepo == nil || apiKey == nil {
		return nil, nil
	}
	if publishedEntries, ok, err := s.apiKeyPublishedPublicCatalogModels(ctx, apiKey, platform); err != nil {
		return nil, err
	} else if ok {
		return publishedEntries, nil
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil, nil
	}

	normalizedPlatform := strings.TrimSpace(strings.ToLower(platform))
	mode := apiKey.EffectiveModelDisplayMode()
	entriesByID := make(map[string]APIKeyPublicModelEntry)
	var firstErr error

	for _, binding := range bindings {
		if binding.Group == nil || !binding.Group.IsActive() {
			continue
		}
		bindingPlatform := strings.TrimSpace(binding.Group.Platform)
		projectionPlatform := apiKeyPublicProjectionPlatform(bindingPlatform, normalizedPlatform)
		if normalizedPlatform != "" && projectionPlatform == "" {
			continue
		}

		queryPlatforms := QueryPlatformsForGroupPlatform(bindingPlatform, false)
		accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, binding.GroupID, queryPlatforms)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for i := range accounts {
			account := &accounts[i]
			if account == nil || !account.IsSchedulable() {
				continue
			}
			if !MatchesGroupPlatform(account, projectionPlatform) {
				continue
			}
			accountForProjection := ResolveProtocolGatewayInboundAccount(account, projectionPlatform)
			entries, err := s.publicModelEntriesForAccount(
				ctx,
				accountForProjection,
				mode,
				projectionPlatform,
				binding.ModelPatterns,
				accountForProjection.GetModelMapping(),
			)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			entries = s.filterPublicEntriesByActiveChannel(ctx, binding.GroupID, projectionPlatform, entries)
			entries = filterOpenAIAPIKeyPublicEntriesForRuntimeQuota(accountForProjection, entries)
			for _, entry := range entries {
				if _, exists := entriesByID[entry.PublicID]; exists {
					continue
				}
				entriesByID[entry.PublicID] = entry
			}
		}
	}

	if len(entriesByID) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, nil
	}
	entries := make([]APIKeyPublicModelEntry, 0, len(entriesByID))
	for _, entry := range entriesByID {
		entries = append(entries, entry)
	}
	if apiKey.IsImageOnly() {
		filtered := make([]APIKeyPublicModelEntry, 0, len(entries))
		for _, entry := range entries {
			// image-only key: only expose native image generation models (capability=image_generation).
			native, _ := s.resolvePublicImageCapability(ctx, &entry)
			if native {
				filtered = append(filtered, entry)
			}
		}
		entries = filtered
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PublicID < entries[j].PublicID
	})
	return entries, nil
}

func (s *GatewayService) apiKeyPublishedPublicCatalogModels(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
) ([]APIKeyPublicModelEntry, bool, error) {
	if s == nil || s.modelCatalogService == nil || apiKey == nil {
		return nil, false, nil
	}
	matches, active, err := s.apiKeyPublishedPublicCatalogVisibleMatches(ctx, apiKey, platform, "")
	if err != nil || !active {
		return nil, active, err
	}
	entriesByID := make(map[string]APIKeyPublicModelEntry, len(matches))
	for _, match := range matches {
		publicID := strings.TrimSpace(match.Entry.PublicID)
		if publicID == "" {
			continue
		}
		if _, exists := entriesByID[publicID]; exists {
			continue
		}
		entriesByID[publicID] = match.Entry
	}
	entries := make([]APIKeyPublicModelEntry, 0, len(entriesByID))
	for _, entry := range entriesByID {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PublicID < entries[j].PublicID
	})
	return entries, true, nil
}

func (s *GatewayService) findPublishedPublicCatalogModel(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
	modelID string,
) (*APIKeyPublicModelEntry, bool, bool, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false, false, nil
	}
	matches, active, err := s.apiKeyPublishedPublicCatalogVisibleMatches(ctx, apiKey, platform, modelID)
	if err != nil || !active {
		return nil, false, active, err
	}
	if len(matches) == 0 {
		return nil, false, true, nil
	}
	entry := matches[0].Entry
	return &entry, true, true, nil
}

func (s *GatewayService) publicModelEntriesForAccount(
	ctx context.Context,
	account *Account,
	mode string,
	platform string,
	modelPatterns []string,
	mapping map[string]string,
) ([]APIKeyPublicModelEntry, error) {
	if account == nil {
		return nil, nil
	}
	_ = mode
	_ = mapping

	projection := BuildAccountModelProjection(ctx, account, s.modelRegistryService)
	entries := projectAccountModelProjectionToPublicEntries(platform, modelPatterns, projection)
	recordPublicModelProjectionSource(apiKeyPublicModelsSourcePolicyProjection)
	slog.Info(
		"api_key_public_models_policy_projection",
		"account_id", account.ID,
		"platform", platform,
		"source", apiKeyPublicModelsSourcePolicyProjection,
		"policy_mode", firstNonEmptyString(projectionPolicyMode(projection), AccountModelPolicyModeWhitelist),
		"projection_source", projectionSource(projection),
		"count", len(entries),
		"alias_only_count", countAliasOnlyPublicEntries(entries),
	)
	return entries, nil
}

func projectAccountModelProjectionToPublicEntries(
	platform string,
	modelPatterns []string,
	projection *AccountModelProjection,
) []APIKeyPublicModelEntry {
	if projection == nil || len(projection.Entries) == 0 {
		return nil
	}

	projected := make(map[string]APIKeyPublicModelEntry, len(projection.Entries))
	for _, candidate := range projection.Entries {
		publicID := normalizeRegistryID(candidate.DisplayModelID)
		if publicID == "" {
			continue
		}
		targetID := normalizeRegistryID(firstNonEmptyString(candidate.TargetModelID, candidate.RouteModelID))
		if !bindingAllowsProjectedPublicModel(modelPatterns, publicID, targetID) {
			continue
		}
		if _, exists := projected[publicID]; exists {
			continue
		}
		displayName := strings.TrimSpace(candidate.DisplayModelID)
		if candidate.VisibilityMode != AccountModelVisibilityModeAlias {
			displayName = firstNonEmptyString(strings.TrimSpace(candidate.DisplayName), displayName)
		}
		projected[publicID] = APIKeyPublicModelEntry{
			PublicID:          publicID,
			AliasID:           publicID,
			SourceID:          targetID,
			DisplayName:       displayName,
			Platform:          platform,
			AvailabilityState: firstNonEmptyTrimmed(candidate.AvailabilityState, AccountModelAvailabilityUnknown),
			StaleState:        firstNonEmptyTrimmed(candidate.StaleState, AccountModelStaleStateUnverified),
			LifecycleStatus: normalizePublicModelLifecycleStatus(
				candidate.Status,
				candidate.DisplayName,
				candidate.DisplayModelID,
				candidate.TargetModelID,
				candidate.RouteModelID,
			),
		}
	}

	result := make([]APIKeyPublicModelEntry, 0, len(projected))
	for _, entry := range projected {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PublicID < result[j].PublicID
	})
	return result
}

func bindingAllowsProjectedPublicModel(modelPatterns []string, publicID string, targetID string) bool {
	for _, candidate := range []string{publicID, targetID} {
		if _, matched := bindingMatchesModel(modelPatterns, candidate); matched {
			return true
		}
	}
	return false
}

func projectionPolicyMode(projection *AccountModelProjection) string {
	if projection == nil {
		return ""
	}
	return strings.TrimSpace(projection.PolicyMode)
}

func projectionSource(projection *AccountModelProjection) string {
	if projection == nil {
		return ""
	}
	return strings.TrimSpace(projection.Source)
}

func projectProbeSummaryToPublicEntries(
	mode string,
	platform string,
	modelPatterns []string,
	mapping map[string]string,
	probeSummary *AccountModelProbeSummary,
	account *Account,
) []APIKeyPublicModelEntry {
	if probeSummary == nil {
		return nil
	}

	detectedSet := make(map[string]AccountModelProbeModel, len(probeSummary.Models))
	for _, detail := range probeSummary.Models {
		sourceID := normalizeRegistryID(detail.ID)
		if sourceID == "" {
			continue
		}
		detail.ID = sourceID
		if strings.TrimSpace(detail.DisplayName) == "" {
			detail.DisplayName = FormatModelCatalogDisplayName(sourceID)
		}
		detectedSet[sourceID] = detail
	}
	for _, modelID := range probeSummary.DetectedModels {
		sourceID := normalizeRegistryID(modelID)
		if sourceID == "" {
			continue
		}
		if _, exists := detectedSet[sourceID]; exists {
			continue
		}
		detectedSet[sourceID] = applyAccountModelProbeProvider(AccountModelProbeModel{
			ID:          sourceID,
			DisplayName: FormatModelCatalogDisplayName(sourceID),
		}, platform)
	}

	candidates := make([]apiKeyPublicProjectionCandidate, 0, len(detectedSet))
	if len(mapping) == 0 {
		if account != nil && account.IsGrokAPIKey() && strings.EqualFold(platform, PlatformGrok) {
			for sourceID, detail := range detectedSet {
				publicID := grokPublicModelForDetectedSource(sourceID)
				candidate, ok := buildAPIKeyPublicProjectionCandidate(mode, publicID, sourceID, platform)
				if !ok {
					continue
				}
				candidate.DisplayName = strings.TrimSpace(detail.DisplayName)
				candidates = append(candidates, candidate)
			}
		} else {
			for sourceID, detail := range detectedSet {
				candidate, ok := buildAPIKeyPublicProjectionCandidate(mode, sourceID, sourceID, platform)
				if !ok {
					continue
				}
				candidate.DisplayName = strings.TrimSpace(detail.DisplayName)
				candidates = append(candidates, candidate)
			}
		}
	} else {
		for alias, source := range mapping {
			candidate, ok := buildAPIKeyPublicProjectionCandidate(mode, alias, source, platform)
			if !ok {
				continue
			}
			candidates = append(candidates, candidate)
		}
		for sourceID, detail := range detectedSet {
			if account != nil && !isRequestedModelSupportedByAccount(context.Background(), nil, account, sourceID) {
				continue
			}
			candidate, ok := buildAPIKeyPublicProjectionCandidate(mode, sourceID, sourceID, platform)
			if !ok {
				continue
			}
			candidate.DisplayName = strings.TrimSpace(detail.DisplayName)
			candidates = append(candidates, candidate)
		}
	}

	projected := make(map[string]APIKeyPublicModelEntry)
	hiddenSourceIDs := make(map[string]struct{})
	for _, candidate := range candidates {
		sourceID, detail, ok := resolveAPIKeyProjectionDetectedDetail(detectedSet, candidate.SourceID)
		if !ok {
			continue
		}
		projectedSourceID := sourceID
		if !strings.EqualFold(platform, PlatformGrok) {
			if rawSourceID := normalizeRegistryID(candidate.SourceID); rawSourceID != "" {
				projectedSourceID = rawSourceID
			}
		}
		publicID := apiKeyPublicProjectionPublicID(platform, candidate, projectedSourceID)
		if publicID == "" {
			publicID = projectedSourceID
		}
		if !bindingMatchesProjectionCandidate(modelPatterns, publicID, candidate) {
			continue
		}
		if !candidate.ExposeAlias && publicID == projectedSourceID {
			if _, hidden := hiddenSourceIDs[projectedSourceID]; hidden {
				continue
			}
		}
		if _, exists := projected[publicID]; exists {
			continue
		}
		displayName := strings.TrimSpace(candidate.AliasID)
		if !candidate.ExposeAlias {
			displayName = strings.TrimSpace(detail.DisplayName)
			if displayName == "" {
				displayName = strings.TrimSpace(candidate.DisplayName)
			}
			if displayName == "" {
				displayName = FormatModelCatalogDisplayName(projectedSourceID)
			}
		}
		projected[publicID] = APIKeyPublicModelEntry{
			PublicID:          publicID,
			AliasID:           candidate.AliasID,
			SourceID:          projectedSourceID,
			DisplayName:       displayName,
			Platform:          platform,
			AvailabilityState: AccountModelAvailabilityUnknown,
			StaleState:        AccountModelStaleStateUnverified,
			LifecycleStatus:   normalizePublicModelLifecycleStatus("", displayName, publicID, projectedSourceID),
		}
		if candidate.ExposeAlias && projectedSourceID != "" {
			hiddenSourceIDs[projectedSourceID] = struct{}{}
		}
	}

	result := make([]APIKeyPublicModelEntry, 0, len(projected))
	for _, entry := range projected {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PublicID < result[j].PublicID
	})
	return result
}

func resolveAPIKeyProjectionDetectedDetail(
	detectedSet map[string]AccountModelProbeModel,
	source string,
) (string, AccountModelProbeModel, bool) {
	candidates := []string{
		normalizeRegistryID(source),
		NormalizeModelCatalogModelID(source),
		modelCatalogDateVersionSuffixPattern.ReplaceAllString(normalizeRegistryID(source), ""),
	}
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = normalizeRegistryID(candidate)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		if detail, ok := detectedSet[candidate]; ok {
			return candidate, detail, true
		}
	}
	return "", AccountModelProbeModel{}, false
}

func (s *GatewayService) FindAPIKeyPublicModel(
	ctx context.Context,
	apiKey *APIKey,
	platform, modelID string,
) (*APIKeyPublicModelEntry, bool, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false, nil
	}
	if entry, ok, active, err := s.findPublishedPublicCatalogModel(ctx, apiKey, platform, modelID); err != nil || ok || active {
		return entry, ok, err
	}
	entries, err := s.GetAPIKeyPublicModels(ctx, apiKey, platform)
	if err != nil {
		return nil, false, err
	}
	for i := range entries {
		if apiKeyPublicEntryMatchesID(entries[i], modelID) {
			entry := entries[i]
			return &entry, true, nil
		}
	}
	return nil, false, nil
}

func (s *GatewayService) ResolveAPIKeySelectionModel(ctx context.Context, apiKey *APIKey, platform, modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ""
	}
	if entry, ok, active, err := s.findPublishedPublicCatalogModel(ctx, apiKey, platform, modelID); err == nil && ok && entry != nil {
		if sourceID := strings.TrimSpace(entry.SourceID); sourceID != "" {
			return sourceID
		}
		return strings.TrimSpace(entry.PublicID)
	} else if err == nil && active {
		return ""
	}
	entry, ok := s.findConfiguredAPIKeyModelByAnyID(ctx, apiKey, platform, modelID)
	if !ok || strings.TrimSpace(entry.PublicID) == "" {
		return modelID
	}
	return entry.PublicID
}

func (s *GatewayService) apiKeyPublishedPublicCatalogVisibleMatches(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
	modelID string,
) ([]apiKeyPublishedPublicCatalogMatch, bool, error) {
	if s == nil || s.modelCatalogService == nil || apiKey == nil {
		return nil, false, nil
	}
	published, active := s.modelCatalogService.activePublishedPublicModelCatalogSnapshot(ctx)
	if !active {
		return nil, false, nil
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil, true, nil
	}
	requestedID := NormalizeModelCatalogModelID(modelID)
	matches := make([]apiKeyPublishedPublicCatalogMatch, 0, len(published.Snapshot.Items))
	for _, item := range published.Snapshot.Items {
		if requestedID != "" && !publicModelCatalogItemMatchesPublicID(item, requestedID) {
			continue
		}
		for _, binding := range bindings {
			match, ok, err := s.publishedPublicCatalogItemForBinding(ctx, binding, platform, item)
			if err != nil {
				return nil, true, err
			}
			if !ok {
				continue
			}
			if apiKey.IsImageOnly() {
				native, _ := s.resolvePublicImageCapability(ctx, &match.Entry)
				if !native && strings.TrimSpace(item.Mode) != "image" {
					continue
				}
			}
			matches = append(matches, match)
			break
		}
	}
	return matches, true, nil
}

func (s *OpenAIGatewayService) apiKeyPublishedPublicCatalogVisibleMatches(
	ctx context.Context,
	apiKey *APIKey,
	platform string,
	modelID string,
) ([]apiKeyPublishedPublicCatalogMatch, bool, error) {
	if s == nil || s.modelCatalogService == nil || apiKey == nil {
		return nil, false, nil
	}
	published, active := s.modelCatalogService.activePublishedPublicModelCatalogSnapshot(ctx)
	if !active {
		return nil, false, nil
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil, true, nil
	}
	requestedID := NormalizeModelCatalogModelID(modelID)
	matches := make([]apiKeyPublishedPublicCatalogMatch, 0, len(published.Snapshot.Items))
	for _, item := range published.Snapshot.Items {
		if requestedID != "" && !publicModelCatalogItemMatchesPublicID(item, requestedID) {
			continue
		}
		for _, binding := range bindings {
			match, ok, err := s.publishedPublicCatalogItemForBinding(ctx, binding, platform, item)
			if err != nil {
				return nil, true, err
			}
			if !ok {
				continue
			}
			if apiKey.IsImageOnly() && strings.TrimSpace(item.Mode) != "image" {
				continue
			}
			matches = append(matches, match)
			break
		}
	}
	return matches, true, nil
}

func (s *GatewayService) publishedPublicCatalogItemForBinding(
	ctx context.Context,
	binding APIKeyGroupBinding,
	requestedPlatform string,
	item PublicModelCatalogItem,
) (apiKeyPublishedPublicCatalogMatch, bool, error) {
	publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
	if s == nil || publicID == "" || binding.Group == nil || !binding.Group.IsActive() {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	if _, matched := bindingMatchesModel(binding.ModelPatterns, publicID); !matched {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	projectionPlatform, ok := publishedCatalogBindingItemPlatform(binding, requestedPlatform, item)
	if !ok {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}

	var resolvedAccount *Account
	if item.SourceAccountID > 0 {
		account, err := s.accountRepo.GetByID(ctx, item.SourceAccountID)
		if err != nil || account == nil {
			recordPublicCatalogPinnedUnavailable(ctx, publishedPublicCatalogEntryFromItem(item), bindingGroupIDPtr(binding), "account_not_found", err)
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
		if !publishedCatalogAccountUsableForBinding(ctx, s.modelRegistryService, account, binding.GroupID, projectionPlatform, item) {
			recordPublicCatalogPinnedUnavailable(ctx, publishedPublicCatalogEntryFromItem(item), bindingGroupIDPtr(binding), "account_not_available_for_group", nil)
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
		resolvedAccount = ResolveProtocolGatewayInboundAccount(account, firstNonEmptyTrimmed(item.SourceProtocol, projectionPlatform, RoutingPlatformForAccount(account)))
		if resolvedAccount == nil {
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
	}

	entry := APIKeyPublicModelEntry{
		PublicID:          publicID,
		AliasID:           publicID,
		SourceID:          NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel)),
		DisplayName:       firstNonEmptyTrimmed(item.DisplayName, item.BaseModel, publicID),
		Platform:          firstNonEmptyTrimmed(item.SourceProtocol, item.Provider, projectionPlatform),
		AvailabilityState: firstNonEmptyTrimmed(item.AvailabilityState, AccountModelAvailabilityUnknown),
		StaleState:        firstNonEmptyTrimmed(item.StaleState, AccountModelStaleStateUnverified),
		LifecycleStatus:   normalizePublicModelLifecycleStatus(item.LifecycleStatus, item.DisplayName, publicID),
	}
	return apiKeyPublishedPublicCatalogMatch{
		Entry:      entry,
		Catalog:    publishedPublicCatalogEntryFromItem(item),
		Binding:    binding,
		GroupID:    bindingGroupIDPtr(binding),
		Account:    resolvedAccount,
		SourceItem: clonePublicModelCatalogItem(item),
	}, true, nil
}

func (s *OpenAIGatewayService) publishedPublicCatalogItemForBinding(
	ctx context.Context,
	binding APIKeyGroupBinding,
	requestedPlatform string,
	item PublicModelCatalogItem,
) (apiKeyPublishedPublicCatalogMatch, bool, error) {
	publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
	if s == nil || publicID == "" || binding.Group == nil || !binding.Group.IsActive() {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	if _, matched := bindingMatchesModel(binding.ModelPatterns, publicID); !matched {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}
	projectionPlatform, ok := publishedCatalogBindingItemPlatform(binding, requestedPlatform, item)
	if !ok {
		return apiKeyPublishedPublicCatalogMatch{}, false, nil
	}

	var resolvedAccount *Account
	if item.SourceAccountID > 0 {
		account, err := s.accountRepo.GetByID(ctx, item.SourceAccountID)
		if err != nil || account == nil {
			recordPublicCatalogPinnedUnavailable(ctx, publishedPublicCatalogEntryFromItem(item), bindingGroupIDPtr(binding), "account_not_found", err)
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
		if !publishedCatalogAccountUsableForBinding(ctx, s.modelRegistryService, account, binding.GroupID, projectionPlatform, item) {
			recordPublicCatalogPinnedUnavailable(ctx, publishedPublicCatalogEntryFromItem(item), bindingGroupIDPtr(binding), "account_not_available_for_group", nil)
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
		resolvedAccount = ResolveProtocolGatewayInboundAccount(account, firstNonEmptyTrimmed(item.SourceProtocol, projectionPlatform, OpenAIPlatformFromContext(ctx)))
		if resolvedAccount == nil || !isOpenAITextRuntimeAccount(resolvedAccount) {
			return apiKeyPublishedPublicCatalogMatch{}, false, nil
		}
	}

	entry := APIKeyPublicModelEntry{
		PublicID:          publicID,
		AliasID:           publicID,
		SourceID:          NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel)),
		DisplayName:       firstNonEmptyTrimmed(item.DisplayName, item.BaseModel, publicID),
		Platform:          firstNonEmptyTrimmed(item.SourceProtocol, item.Provider, projectionPlatform),
		AvailabilityState: firstNonEmptyTrimmed(item.AvailabilityState, AccountModelAvailabilityUnknown),
		StaleState:        firstNonEmptyTrimmed(item.StaleState, AccountModelStaleStateUnverified),
		LifecycleStatus:   normalizePublicModelLifecycleStatus(item.LifecycleStatus, item.DisplayName, publicID),
	}
	return apiKeyPublishedPublicCatalogMatch{
		Entry:      entry,
		Catalog:    publishedPublicCatalogEntryFromItem(item),
		Binding:    binding,
		GroupID:    bindingGroupIDPtr(binding),
		Account:    resolvedAccount,
		SourceItem: clonePublicModelCatalogItem(item),
	}, true, nil
}

func publishedCatalogBindingItemPlatform(binding APIKeyGroupBinding, requestedPlatform string, item PublicModelCatalogItem) (string, bool) {
	if binding.Group == nil {
		return "", false
	}
	bindingPlatform := strings.TrimSpace(strings.ToLower(binding.Group.Platform))
	requestedPlatform = strings.TrimSpace(strings.ToLower(requestedPlatform))
	if requestedPlatform != "" {
		projectionPlatform := apiKeyPublicProjectionPlatform(bindingPlatform, requestedPlatform)
		if projectionPlatform == "" {
			return "", false
		}
		if publicCatalogItemMatchesProtocol(item, requestedPlatform) || publicCatalogItemMatchesProtocol(item, projectionPlatform) {
			return projectionPlatform, true
		}
		return "", false
	}
	if bindingPlatform == PlatformProtocolGateway {
		for _, protocol := range []string{PlatformOpenAI, PlatformAnthropic, PlatformGemini} {
			if publicCatalogItemMatchesProtocol(item, protocol) {
				return protocol, true
			}
		}
		return "", false
	}
	if publicCatalogItemMatchesProtocol(item, bindingPlatform) {
		return bindingPlatform, true
	}
	return "", false
}

func publicCatalogItemMatchesProtocol(item PublicModelCatalogItem, protocol string) bool {
	protocol = strings.TrimSpace(strings.ToLower(protocol))
	if protocol == "" {
		return false
	}
	for _, candidate := range append([]string{item.SourceProtocol, item.Provider}, item.RequestProtocols...) {
		if strings.TrimSpace(strings.ToLower(candidate)) == protocol {
			return true
		}
	}
	return false
}

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

func bindingGroupIDPtr(binding APIKeyGroupBinding) *int64 {
	if binding.GroupID <= 0 {
		return nil
	}
	id := binding.GroupID
	return &id
}

func recordPublicCatalogRouteMiss(ctx context.Context, apiKey *APIKey, groupID *int64, publicModelID string, platform string) {
	protocolruntime.RecordBillingResolverFallback("public_catalog_route_miss")
	fields := publicCatalogLogFields(ctx, nil, groupID, apiKey)
	fields = append(fields,
		zap.String("public_model_id", strings.TrimSpace(publicModelID)),
		zap.String("platform", strings.TrimSpace(platform)),
	)
	logger.FromContext(ctx).Warn("public model catalog route miss", fields...)
}

func (s *GatewayService) findConfiguredAPIKeyModelByAnyID(
	ctx context.Context,
	apiKey *APIKey,
	platform, modelID string,
) (*APIKeyPublicModelEntry, bool) {
	if s == nil || s.accountRepo == nil || apiKey == nil {
		return nil, false
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, false
	}
	bindings := apiKeyBindingsForSelection(apiKey)
	if len(bindings) == 0 {
		return nil, false
	}
	normalizedPlatform := strings.TrimSpace(strings.ToLower(platform))
	mode := apiKey.EffectiveModelDisplayMode()

	for _, binding := range bindings {
		if binding.Group == nil || !binding.Group.IsActive() {
			continue
		}
		bindingPlatform := strings.TrimSpace(binding.Group.Platform)
		projectionPlatform := apiKeyPublicProjectionPlatform(bindingPlatform, normalizedPlatform)
		if normalizedPlatform != "" && projectionPlatform == "" {
			continue
		}
		queryPlatforms := QueryPlatformsForGroupPlatform(bindingPlatform, false)
		accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, binding.GroupID, queryPlatforms)
		if err != nil {
			continue
		}
		for i := range accounts {
			account := &accounts[i]
			if account == nil || !account.IsSchedulable() {
				continue
			}
			if !MatchesGroupPlatform(account, projectionPlatform) {
				continue
			}
			accountForProjection := ResolveProtocolGatewayInboundAccount(account, projectionPlatform)
			entries, err := s.publicModelEntriesForAccount(ctx, accountForProjection, mode, projectionPlatform, binding.ModelPatterns, accountForProjection.GetModelMapping())
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if apiKey.IsImageOnly() {
					native, _ := s.resolvePublicImageCapability(ctx, &entry)
					if !native {
						continue
					}
				}
				if apiKeyPublicEntryMatchesID(entry, modelID) {
					return &entry, true
				}
			}
		}
	}
	return nil, false
}

func apiKeyPublicProjectionPlatform(bindingPlatform string, requestedPlatform string) string {
	bindingPlatform = strings.TrimSpace(strings.ToLower(bindingPlatform))
	requestedPlatform = strings.TrimSpace(strings.ToLower(requestedPlatform))
	if bindingPlatform == "" {
		return ""
	}
	if requestedPlatform == "" || strings.EqualFold(bindingPlatform, requestedPlatform) {
		return bindingPlatform
	}
	if bindingPlatform != PlatformProtocolGateway {
		return ""
	}
	switch requestedPlatform {
	case PlatformOpenAI, PlatformAnthropic, PlatformGemini:
		return requestedPlatform
	default:
		return ""
	}
}

func buildAPIKeyPublicProjectionCandidate(mode, alias, source, platform string) (apiKeyPublicProjectionCandidate, bool) {
	alias = strings.TrimSpace(alias)
	source = strings.TrimSpace(source)
	if alias == "" && source == "" {
		return apiKeyPublicProjectionCandidate{}, false
	}
	if alias == "" {
		alias = source
	}
	if source == "" {
		source = alias
	}
	explicitAlias := shouldExposePublicAlias(platform, alias, source)

	if explicitAlias {
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias,
			Platform:    platform,
			ExposeAlias: true,
		}, true
	}

	switch NormalizeAPIKeyModelDisplayMode(mode) {
	case APIKeyModelDisplayModeSourceOnly:
		return apiKeyPublicProjectionCandidate{
			MatchID:     source,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: source,
			Platform:    platform,
			ExposeAlias: false,
		}, true
	case APIKeyModelDisplayModeAliasAndSource:
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias + " | " + source,
			Platform:    platform,
			ExposeAlias: false,
		}, true
	default:
		return apiKeyPublicProjectionCandidate{
			MatchID:     alias,
			AliasID:     alias,
			SourceID:    source,
			DisplayName: alias,
			Platform:    platform,
			ExposeAlias: false,
		}, true
	}
}

func apiKeyPublicProjectionPublicID(platform string, candidate apiKeyPublicProjectionCandidate, sourceID string) string {
	if candidate.ExposeAlias {
		if aliasID := normalizeRegistryID(candidate.AliasID); aliasID != "" {
			return aliasID
		}
		if matchID := normalizeRegistryID(candidate.MatchID); matchID != "" {
			return matchID
		}
	}
	if !strings.EqualFold(platform, PlatformGrok) {
		return sourceID
	}
	if aliasID := normalizeRegistryID(candidate.AliasID); aliasID != "" && aliasID != sourceID {
		return aliasID
	}
	if matchID := normalizeRegistryID(candidate.MatchID); matchID != "" && matchID != sourceID {
		return matchID
	}
	if publicID := grokPublicModelForDetectedSource(sourceID); publicID != "" {
		return publicID
	}
	return sourceID
}

func shouldExposePublicAlias(platform, alias, source string) bool {
	alias = strings.TrimSpace(alias)
	source = strings.TrimSpace(source)
	if alias == "" || source == "" || alias == source {
		return false
	}
	if strings.Contains(alias, "*") {
		return false
	}
	if strings.EqualFold(platform, PlatformGemini) && alias == DefaultVertexPublicModelAlias(source) {
		return false
	}
	return true
}

func apiKeyPublicEntryMatchesID(entry APIKeyPublicModelEntry, modelID string) bool {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	for _, candidate := range []string{entry.PublicID, entry.AliasID} {
		if strings.TrimSpace(candidate) == modelID {
			return true
		}
	}
	return false
}

func bindingMatchesProjectionCandidate(
	modelPatterns []string,
	publicID string,
	candidate apiKeyPublicProjectionCandidate,
) bool {
	for _, modelID := range []string{
		publicID,
		candidate.MatchID,
		candidate.AliasID,
		candidate.SourceID,
	} {
		if _, matched := bindingMatchesModel(modelPatterns, modelID); matched {
			return true
		}
	}
	return false
}

func countAliasOnlyPublicEntries(entries []APIKeyPublicModelEntry) int {
	count := 0
	for _, entry := range entries {
		if strings.TrimSpace(entry.PublicID) == "" {
			continue
		}
		if strings.TrimSpace(entry.PublicID) != strings.TrimSpace(entry.SourceID) {
			count++
		}
	}
	return count
}

func filterOpenAIAPIKeyPublicEntriesForRuntimeQuota(account *Account, entries []APIKeyPublicModelEntry) []APIKeyPublicModelEntry {
	if len(entries) == 0 || account == nil || !account.IsOpenAI() || !isOpenAIProPlan(account) {
		return entries
	}

	filtered := make([]APIKeyPublicModelEntry, 0, len(entries))
	for _, entry := range entries {
		if shouldHideOpenAIModelForRuntimeQuota(account, apiKeyPublicModelRuntimeQuotaCandidates(account, entry)...) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
