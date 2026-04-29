package service

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	AccountModelDiagnosticsStatusOK            = "ok"
	AccountModelDiagnosticsStatusDegraded      = "degraded"
	AccountModelDiagnosticsStatusFilteredEmpty = "filtered_empty"
	AccountModelDiagnosticsStatusProbeFailed   = "probe_failed_empty"
	AccountModelDiagnosticsStatusFallbackOnly  = "fallback_only"

	accountDiagnosticsListPageSize = 1000
)

type AccountModelDiagnosticsPreview struct {
	PublicID    string `json:"public_id"`
	AliasID     string `json:"alias_id,omitempty"`
	SourceID    string `json:"source_id"`
	DisplayName string `json:"display_name"`
	Platform    string `json:"platform"`
}

type AccountModelDiagnosticsAPIKeyExposure struct {
	APIKeyID         int64                            `json:"api_key_id"`
	APIKeyName       string                           `json:"api_key_name"`
	ModelDisplayMode string                           `json:"model_display_mode"`
	ModelPatterns    []string                         `json:"model_patterns,omitempty"`
	PublicModels     []AccountModelDiagnosticsPreview `json:"public_models"`
}

type AccountModelDiagnosticsGroupExposure struct {
	GroupID       int64                                   `json:"group_id"`
	GroupName     string                                  `json:"group_name"`
	GroupPlatform string                                  `json:"group_platform"`
	PublicModels  []AccountModelDiagnosticsPreview        `json:"public_models"`
	APIKeys       []AccountModelDiagnosticsAPIKeyExposure `json:"api_keys"`
	Warnings      []string                                `json:"warnings,omitempty"`
}

type AccountModelDiagnosticsResponse struct {
	AccountID               int64                                  `json:"account_id"`
	RoutingPlatform         string                                 `json:"routing_platform"`
	Status                  string                                 `json:"status"`
	ProbeSource             string                                 `json:"probe_source,omitempty"`
	ProbeNotice             string                                 `json:"probe_notice,omitempty"`
	ResolvedUpstreamURL     string                                 `json:"resolved_upstream_url,omitempty"`
	ResolvedUpstreamHost    string                                 `json:"resolved_upstream_host,omitempty"`
	ResolvedUpstreamService string                                 `json:"resolved_upstream_service,omitempty"`
	SavedModels             []string                               `json:"saved_models"`
	DetectedModels          []string                               `json:"detected_models"`
	PublicModelsPreview     []AccountModelDiagnosticsPreview       `json:"public_models_preview"`
	GroupExposures          []AccountModelDiagnosticsGroupExposure `json:"group_exposures"`
	Warnings                []string                               `json:"warnings"`
}

type AccountModelDiagnosticsService struct {
	accountRepo               AccountRepository
	apiKeyRepo                APIKeyRepository
	groupRepo                 GroupRepository
	accountModelImportService *AccountModelImportService
}

func NewAccountModelDiagnosticsService(
	accountRepo AccountRepository,
	apiKeyRepo APIKeyRepository,
	groupRepo GroupRepository,
	accountModelImportService *AccountModelImportService,
) *AccountModelDiagnosticsService {
	return &AccountModelDiagnosticsService{
		accountRepo:               accountRepo,
		apiKeyRepo:                apiKeyRepo,
		groupRepo:                 groupRepo,
		accountModelImportService: accountModelImportService,
	}
}

func (s *AccountModelDiagnosticsService) Diagnose(
	ctx context.Context,
	accountID int64,
	refresh bool,
) (*AccountModelDiagnosticsResponse, error) {
	if s == nil || s.accountRepo == nil {
		return nil, errors.New("account model diagnostics service unavailable")
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}

	routingPlatform := RoutingPlatformForAccount(account)
	savedSummary := AccountSavedModelProbeSummary(account)
	savedModels := summaryDetectedModels(savedSummary)

	var liveSummary *AccountModelProbeSummary
	var liveErr error
	if refresh || savedSummary == nil {
		if s.accountModelImportService == nil {
			liveErr = errors.New("account model import service unavailable")
		} else {
			liveSummary, liveErr = s.accountModelImportService.ListAccountModels(ctx, account, refresh)
		}
	}

	detectedModels := summaryDetectedModels(liveSummary)
	selectedSummary := savedSummary
	if liveSummary != nil {
		selectedSummary = liveSummary
	}

	accountPreview := diagnosticsPreviewEntries(
		APIKeyModelDisplayModeAliasOnly,
		routingPlatform,
		nil,
		account.GetModelMapping(),
		selectedSummary,
		account,
	)
	groupExposures, groupWarnings := s.buildGroupExposures(ctx, account, selectedSummary)
	resolved := diagnosticsResolvedUpstream(account, selectedSummary)

	warnings := make([]string, 0, 4+len(groupWarnings))
	if liveErr != nil {
		if len(savedModels) > 0 {
			warnings = append(warnings, "live probe failed; current preview is showing the saved models snapshot")
		} else {
			warnings = append(warnings, "live probe failed and no saved models are available")
		}
	}
	if !refresh && len(savedModels) > 0 {
		warnings = append(warnings, "showing saved models first; set refresh=true to force a live reprobe")
	}
	if len(accountPreview) == 0 && (len(savedModels) > 0 || len(detectedModels) > 0) {
		warnings = append(warnings, "models exist on the account, but current mapping filters them all out")
	}
	warnings = append(warnings, groupWarnings...)
	warnings = uniqueSortedStrings(warnings)

	status := resolveAccountModelDiagnosticsStatus(
		account,
		selectedSummary,
		accountPreview,
		groupExposures,
		savedModels,
		detectedModels,
		liveErr,
	)

	slog.Info(
		"admin_account_model_diagnostics",
		"account_id", account.ID,
		"routing_platform", routingPlatform,
		"status", status,
		"probe_source", probeSourceFromSummary(selectedSummary),
		"saved_models", len(savedModels),
		"detected_models", len(detectedModels),
		"public_models", len(accountPreview),
		"group_exposures", len(groupExposures),
		"refresh", refresh,
		"live_probe_failed", liveErr != nil,
	)

	return &AccountModelDiagnosticsResponse{
		AccountID:               account.ID,
		RoutingPlatform:         routingPlatform,
		Status:                  status,
		ProbeSource:             probeSourceFromSummary(selectedSummary),
		ProbeNotice:             probeNoticeFromSummary(liveSummary),
		ResolvedUpstreamURL:     strings.TrimSpace(resolved.URL),
		ResolvedUpstreamHost:    strings.TrimSpace(resolved.Host),
		ResolvedUpstreamService: strings.TrimSpace(resolved.Service),
		SavedModels:             cloneDiagnosticsStringSlice(savedModels),
		DetectedModels:          cloneDiagnosticsStringSlice(detectedModels),
		PublicModelsPreview:     accountPreview,
		GroupExposures:          groupExposures,
		Warnings:                cloneDiagnosticsStringSlice(warnings),
	}, nil
}

func (s *AccountModelDiagnosticsService) buildGroupExposures(
	ctx context.Context,
	account *Account,
	summary *AccountModelProbeSummary,
) ([]AccountModelDiagnosticsGroupExposure, []string) {
	if s == nil || s.groupRepo == nil || s.apiKeyRepo == nil {
		return []AccountModelDiagnosticsGroupExposure{}, []string{"group or api key repositories are unavailable for diagnostics"}
	}

	groupIDs := accountDiagnosticsGroupIDs(account)
	if len(groupIDs) == 0 {
		return []AccountModelDiagnosticsGroupExposure{}, []string{"the account is not bound to any group, so no downstream api key can expose it"}
	}

	exposures := make([]AccountModelDiagnosticsGroupExposure, 0, len(groupIDs))
	warnings := make([]string, 0)

	for _, groupID := range groupIDs {
		exposure := AccountModelDiagnosticsGroupExposure{
			GroupID:      groupID,
			PublicModels: []AccountModelDiagnosticsPreview{},
			APIKeys:      []AccountModelDiagnosticsAPIKeyExposure{},
		}

		group, err := s.groupRepo.GetByIDLite(ctx, groupID)
		if err != nil || group == nil {
			exposure.Warnings = append(exposure.Warnings, "failed to load current group metadata")
			exposures = append(exposures, exposure)
			warnings = append(warnings, "some group metadata could not be loaded")
			continue
		}
		exposure.GroupName = group.Name
		exposure.GroupPlatform = group.Platform

		apiKeys, _, err := s.apiKeyRepo.ListByGroupID(
			ctx,
			groupID,
			pagination.PaginationParams{Page: 1, PageSize: accountDiagnosticsListPageSize},
		)
		if err != nil {
			exposure.Warnings = append(exposure.Warnings, "failed to load api keys bound to this group")
			exposures = append(exposures, exposure)
			warnings = append(warnings, "some group api key exposures could not be loaded")
			continue
		}

		sort.SliceStable(apiKeys, func(i, j int) bool { return apiKeys[i].ID < apiKeys[j].ID })
		union := make(map[string]AccountModelDiagnosticsPreview)
		for _, apiKey := range apiKeys {
			binding := apiKey.GroupBindingByID(groupID)
			modelPatterns := []string(nil)
			if binding != nil {
				modelPatterns = binding.ModelPatterns
			}
			publicModels := diagnosticsPreviewEntries(
				apiKey.EffectiveModelDisplayMode(),
				group.Platform,
				modelPatterns,
				account.GetModelMapping(),
				summary,
				account,
			)
			exposure.APIKeys = append(exposure.APIKeys, AccountModelDiagnosticsAPIKeyExposure{
				APIKeyID:         apiKey.ID,
				APIKeyName:       apiKey.Name,
				ModelDisplayMode: apiKey.EffectiveModelDisplayMode(),
				ModelPatterns:    append([]string(nil), modelPatterns...),
				PublicModels:     publicModels,
			})
			for _, item := range publicModels {
				if _, exists := union[item.PublicID]; exists {
					continue
				}
				union[item.PublicID] = item
			}
		}
		if len(apiKeys) == 0 {
			exposure.Warnings = append(exposure.Warnings, "this group currently has no bound api keys")
		}

		exposure.PublicModels = previewMapValues(union)
		exposure.Warnings = uniqueSortedStrings(exposure.Warnings)
		exposures = append(exposures, exposure)
	}

	sort.SliceStable(exposures, func(i, j int) bool { return exposures[i].GroupID < exposures[j].GroupID })
	if !hasGroupExposureModels(exposures) {
		warnings = append(warnings, "no group/api key combination currently exposes this account to downstream /models")
	}

	return exposures, uniqueSortedStrings(warnings)
}

func resolveAccountModelDiagnosticsStatus(
	account *Account,
	summary *AccountModelProbeSummary,
	accountPreview []AccountModelDiagnosticsPreview,
	groupExposures []AccountModelDiagnosticsGroupExposure,
	savedModels []string,
	detectedModels []string,
	liveErr error,
) string {
	if hasGroupExposureModels(groupExposures) {
		if isFallbackOnlyProbeSource(probeSourceFromSummary(summary)) {
			return AccountModelDiagnosticsStatusFallbackOnly
		}
		if liveErr != nil && len(savedModels) > 0 && !shouldTreatSavedDiagnosticsSnapshotAsOK(account) {
			return AccountModelDiagnosticsStatusDegraded
		}
		return AccountModelDiagnosticsStatusOK
	}
	if len(accountPreview) > 0 || len(savedModels) > 0 || len(detectedModels) > 0 {
		return AccountModelDiagnosticsStatusFilteredEmpty
	}
	if liveErr != nil || summary == nil {
		return AccountModelDiagnosticsStatusProbeFailed
	}
	return AccountModelDiagnosticsStatusFilteredEmpty
}

func shouldTreatSavedDiagnosticsSnapshotAsOK(account *Account) bool {
	return isGeminiAIStudioSourceAccount(account)
}

func diagnosticsPreviewEntries(
	mode string,
	platform string,
	modelPatterns []string,
	mapping map[string]string,
	summary *AccountModelProbeSummary,
	account *Account,
) []AccountModelDiagnosticsPreview {
	entries := projectProbeSummaryToPublicEntries(mode, platform, modelPatterns, mapping, summary, account)
	if len(entries) == 0 {
		return []AccountModelDiagnosticsPreview{}
	}

	out := make([]AccountModelDiagnosticsPreview, 0, len(entries))
	for _, entry := range entries {
		out = append(out, AccountModelDiagnosticsPreview{
			PublicID:    entry.PublicID,
			AliasID:     entry.AliasID,
			SourceID:    entry.SourceID,
			DisplayName: entry.DisplayName,
			Platform:    entry.Platform,
		})
	}
	return out
}

func diagnosticsResolvedUpstream(account *Account, summary *AccountModelProbeSummary) ResolvedUpstreamInfo {
	if summary != nil {
		return ResolveUpstreamInfo(
			summary.ResolvedUpstreamURL,
			firstNonEmptyTrimmed(summary.ResolvedUpstreamService, RoutingPlatformForAccount(account)),
			summary.ProbeSource,
		)
	}
	if account == nil {
		return ResolvedUpstreamInfo{}
	}
	return ResolveUpstreamInfo(
		account.GetExtraString("upstream_url"),
		firstNonEmptyTrimmed(account.GetExtraString("upstream_service"), RoutingPlatformForAccount(account)),
		account.GetExtraString("upstream_probe_source"),
	)
}

func accountDiagnosticsGroupIDs(account *Account) []int64 {
	if account == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(account.GroupIDs))
	out := make([]int64, 0, len(account.GroupIDs))
	appendID := func(id int64) {
		if id <= 0 {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range account.GroupIDs {
		appendID(id)
	}
	for _, group := range account.Groups {
		if group != nil {
			appendID(group.ID)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func hasGroupExposureModels(exposures []AccountModelDiagnosticsGroupExposure) bool {
	for _, exposure := range exposures {
		if len(exposure.PublicModels) > 0 {
			return true
		}
	}
	return false
}

func previewMapValues(items map[string]AccountModelDiagnosticsPreview) []AccountModelDiagnosticsPreview {
	if len(items) == 0 {
		return []AccountModelDiagnosticsPreview{}
	}
	out := make([]AccountModelDiagnosticsPreview, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].PublicID < out[j].PublicID })
	return out
}

func cloneDiagnosticsStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}

func summaryDetectedModels(summary *AccountModelProbeSummary) []string {
	if summary == nil {
		return nil
	}
	return normalizeAccountModelProbeSnapshotModels(summary.DetectedModels)
}

func probeSourceFromSummary(summary *AccountModelProbeSummary) string {
	if summary == nil {
		return ""
	}
	return strings.TrimSpace(summary.ProbeSource)
}

func probeNoticeFromSummary(summary *AccountModelProbeSummary) string {
	if summary == nil {
		return ""
	}
	return strings.TrimSpace(summary.ProbeNotice)
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func isFallbackOnlyProbeSource(source string) bool {
	switch strings.TrimSpace(source) {
	case accountModelProbeSourceGrokSSOCapability,
		accountModelProbeSourceKiroBuiltinCatalog,
		accountModelProbeSourceCopilotStaticCatalog,
		accountModelProbeSourceVertexExpressCatalog,
		accountModelProbeSourceVertexServiceAccountCatalog:
		return true
	default:
		return false
	}
}
