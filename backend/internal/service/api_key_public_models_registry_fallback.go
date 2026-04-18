package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
)

const (
	apiKeyPublicModelsSourceSavedProbe         = "saved_probe"
	apiKeyPublicModelsSourceLiveProbe          = "live_probe"
	apiKeyPublicModelsSourceRestrictedFallback = "restricted_fallback"
	apiKeyPublicModelsSourceRegistryFallback   = "registry_fallback"
	apiKeyPublicModelsSourceVertexCatalog      = "vertex_catalog"
)

func recordPublicModelProjectionSource(source string) {
	protocolruntime.RecordPublicModelProjection(strings.TrimSpace(source))
}

func recordPublicModelRestrictionHit(account *Account) {
	protocolruntime.RecordPublicModelRestrictionHit(publicModelRestrictionReason(account))
}

func publicModelRestrictionReason(account *Account) string {
	switch {
	case account == nil:
		return "unknown"
	case account.Type == AccountTypeBedrock:
		return "bedrock_allowlist"
	case account.Platform == PlatformAntigravity:
		return "antigravity_allowlist"
	default:
		return "account_scope"
	}
}

func buildAccountModelProbeSummaryFromRegistryEntries(
	entries []modelregistry.ModelEntry,
	platform string,
	probeSource string,
) *AccountModelProbeSummary {
	if len(entries) == 0 {
		return nil
	}
	detected := make([]string, 0, len(entries))
	models := make([]AccountModelProbeModel, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		modelID := normalizeRegistryID(entry.ID)
		if modelID == "" {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}
		seen[modelID] = struct{}{}
		detected = append(detected, modelID)
		models = append(models, applyAccountModelProbeProvider(AccountModelProbeModel{
			ID:             modelID,
			DisplayName:    firstNonEmptyString(strings.TrimSpace(entry.DisplayName), FormatModelCatalogDisplayName(modelID)),
			Provider:       normalizeRegistryPlatform(entry.Provider),
			UpstreamSource: probeSource,
		}, platform))
	}
	if len(detected) == 0 {
		return nil
	}
	return &AccountModelProbeSummary{
		DetectedModels: detected,
		Models:         models,
		ProbeSource:    probeSource,
	}
}

func buildAccountModelProbeSummaryFromVertexCatalog(
	models []VertexCatalogModel,
	platform string,
) *AccountModelProbeSummary {
	if len(models) == 0 {
		return nil
	}
	detected := make([]string, 0, len(models))
	items := make([]AccountModelProbeModel, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		modelID := normalizeRegistryID(model.ID)
		if modelID == "" {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}
		seen[modelID] = struct{}{}
		detected = append(detected, modelID)
		items = append(items, applyAccountModelProbeProvider(AccountModelProbeModel{
			ID:                 modelID,
			DisplayName:        firstNonEmptyString(strings.TrimSpace(model.DisplayName), FormatModelCatalogDisplayName(modelID)),
			Provider:           normalizeRegistryPlatform(platform),
			UpstreamSource:     firstNonEmptyString(strings.TrimSpace(model.UpstreamSource), apiKeyPublicModelsSourceVertexCatalog),
			Availability:       strings.TrimSpace(model.Availability),
			AvailabilityReason: strings.TrimSpace(model.AvailabilityReason),
		}, platform))
	}
	if len(detected) == 0 {
		return nil
	}
	return &AccountModelProbeSummary{
		DetectedModels: detected,
		Models:         items,
		ProbeSource:    apiKeyPublicModelsSourceVertexCatalog,
	}
}

func (s *GatewayService) buildAccountModelProbeSummaryFromModelIDs(
	ctx context.Context,
	modelIDs []string,
	platform string,
	probeSource string,
) *AccountModelProbeSummary {
	if len(modelIDs) == 0 {
		return nil
	}
	detected := make([]string, 0, len(modelIDs))
	items := make([]AccountModelProbeModel, 0, len(modelIDs))
	seen := make(map[string]struct{}, len(modelIDs))
	for _, rawID := range modelIDs {
		modelID := normalizeRegistryID(rawID)
		if modelID == "" {
			continue
		}
		item := AccountModelProbeModel{
			ID:             modelID,
			DisplayName:    FormatModelCatalogDisplayName(modelID),
			UpstreamSource: probeSource,
		}
		if s != nil && s.modelRegistryService != nil {
			if resolution, err := s.modelRegistryService.ExplainResolution(ctx, modelID); err == nil && resolution != nil {
				resolvedID := firstNonEmptyString(resolution.EffectiveID, resolution.CanonicalID, resolution.Entry.ID)
				if normalizedResolved := normalizeRegistryID(resolvedID); normalizedResolved != "" {
					item.ID = normalizedResolved
				}
				if displayName := strings.TrimSpace(resolution.Entry.DisplayName); displayName != "" {
					item.DisplayName = displayName
				}
				item.Provider = normalizeRegistryPlatform(resolution.Entry.Provider)
			}
		}
		if _, exists := seen[item.ID]; exists {
			continue
		}
		seen[item.ID] = struct{}{}
		detected = append(detected, item.ID)
		items = append(items, applyAccountModelProbeProvider(item, platform))
	}
	if len(detected) == 0 {
		return nil
	}
	return &AccountModelProbeSummary{
		DetectedModels: detected,
		Models:         items,
		ProbeSource:    probeSource,
	}
}

func mergeAccountModelProbeSummaries(left, right *AccountModelProbeSummary) *AccountModelProbeSummary {
	switch {
	case left == nil:
		return right
	case right == nil:
		return left
	}
	merged := &AccountModelProbeSummary{
		DetectedModels: append([]string(nil), left.DetectedModels...),
		Models:         append([]AccountModelProbeModel(nil), left.Models...),
		ProbeSource:    firstNonEmptyString(strings.TrimSpace(left.ProbeSource), strings.TrimSpace(right.ProbeSource)),
	}
	seen := make(map[string]struct{}, len(merged.DetectedModels))
	for _, modelID := range merged.DetectedModels {
		seen[normalizeRegistryID(modelID)] = struct{}{}
	}
	for _, modelID := range right.DetectedModels {
		normalized := normalizeRegistryID(modelID)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		merged.DetectedModels = append(merged.DetectedModels, normalized)
	}
	modelSeen := make(map[string]struct{}, len(merged.Models))
	for _, item := range merged.Models {
		modelSeen[normalizeRegistryID(item.ID)] = struct{}{}
	}
	for _, item := range right.Models {
		normalized := normalizeRegistryID(item.ID)
		if normalized == "" {
			continue
		}
		if _, exists := modelSeen[normalized]; exists {
			continue
		}
		modelSeen[normalized] = struct{}{}
		merged.Models = append(merged.Models, item)
	}
	return merged
}

func (s *GatewayService) runtimeRegistryEntriesForPlatform(ctx context.Context, platform string) ([]modelregistry.ModelEntry, error) {
	if s == nil || s.modelRegistryService == nil || strings.TrimSpace(platform) == "" {
		return nil, nil
	}
	return s.modelRegistryService.GetModelsByPlatform(ctx, platform, "runtime", "whitelist")
}

func (s *GatewayService) restrictedPublicModelSummary(
	ctx context.Context,
	account *Account,
	platform string,
) (*AccountModelProbeSummary, bool, error) {
	if !accountHasExplicitModelRestrictions(account) {
		return nil, false, nil
	}
	recordPublicModelRestrictionHit(account)

	entries, err := s.runtimeRegistryEntriesForPlatform(ctx, platform)
	filteredEntries := make([]modelregistry.ModelEntry, 0, len(entries))
	for _, entry := range entries {
		if !isRequestedModelSupportedByAccount(ctx, s.modelRegistryService, account, entry.ID) {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	summary := buildAccountModelProbeSummaryFromRegistryEntries(filteredEntries, platform, apiKeyPublicModelsSourceRestrictedFallback)
	configuredIDs := accountConfiguredSourceModelIDs(account, platform)
	summary = mergeAccountModelProbeSummaries(summary, s.buildAccountModelProbeSummaryFromModelIDs(ctx, configuredIDs, platform, apiKeyPublicModelsSourceRestrictedFallback))
	if summary != nil {
		recordPublicModelProjectionSource(apiKeyPublicModelsSourceRestrictedFallback)
		slog.Info(
			"api_key_public_models_restricted_projection",
			"account_id", account.ID,
			"platform", platform,
			"source", apiKeyPublicModelsSourceRestrictedFallback,
			"explicit_restriction", true,
			"count", len(summary.DetectedModels),
		)
		return summary, true, nil
	}
	return nil, true, err
}

func (s *GatewayService) registryFallbackPublicModelEntries(
	ctx context.Context,
	account *Account,
	mode string,
	platform string,
	modelPatterns []string,
	mapping map[string]string,
) ([]APIKeyPublicModelEntry, error) {
	registryEntries, err := s.runtimeRegistryEntriesForPlatform(ctx, platform)
	if err != nil {
		return nil, err
	}
	summary := buildAccountModelProbeSummaryFromRegistryEntries(registryEntries, platform, apiKeyPublicModelsSourceRegistryFallback)
	if summary == nil {
		return nil, nil
	}
	recordPublicModelProjectionSource(apiKeyPublicModelsSourceRegistryFallback)
	entries := projectProbeSummaryToPublicEntries(mode, platform, modelPatterns, mapping, summary, account)
	slog.Info(
		"api_key_public_models_registry_fallback",
		"account_id", account.ID,
		"platform", platform,
		"source", apiKeyPublicModelsSourceRegistryFallback,
		"explicit_restriction", accountHasExplicitModelRestrictions(account),
		"count", len(summary.DetectedModels),
		"alias_only_count", countAliasOnlyPublicEntries(entries),
	)
	return entries, nil
}

func filterAPIKeyPublicEntriesByChannel(
	channel *model.Channel,
	platform string,
	entries []APIKeyPublicModelEntry,
) []APIKeyPublicModelEntry {
	if channel == nil || !channel.RestrictModels || len(entries) == 0 {
		return entries
	}
	filtered := make([]APIKeyPublicModelEntry, 0, len(entries))
	for _, entry := range entries {
		requestedModel := strings.TrimSpace(firstNonEmptyString(entry.AliasID, entry.PublicID, entry.SourceID))
		if requestedModel == "" {
			continue
		}
		selectionModel := resolveChannelMappingTarget(channel, platform, requestedModel)
		if selectionModel == "" {
			selectionModel = requestedModel
		}
		if !channelAllowsModel(channel, platform, requestedModel, selectionModel) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func (s *GatewayService) filterPublicEntriesByActiveChannel(
	ctx context.Context,
	groupID int64,
	platform string,
	entries []APIKeyPublicModelEntry,
) []APIKeyPublicModelEntry {
	if s == nil || s.channelService == nil || s.channelService.repo == nil || groupID <= 0 {
		return entries
	}
	channel, err := s.channelService.repo.GetActiveByGroupID(ctx, groupID)
	if err != nil || channel == nil {
		return entries
	}
	return filterAPIKeyPublicEntriesByChannel(channel, platform, entries)
}
