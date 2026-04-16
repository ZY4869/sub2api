package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	gocache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	accountModelProbeSourceUpstream                    = "upstream"
	accountModelProbeSourceVertexExpressCatalog        = "vertex_express_catalog"
	accountModelProbeSourceVertexServiceAccountCatalog = "vertex_service_account_catalog"
	accountModelProbeSourceKiroBuiltinCatalog          = "kiro_builtin_catalog"
	accountModelProbeSourceCopilotStaticCatalog        = "copilot_static_catalog"
	accountModelProbeCacheTTL                          = 5 * time.Minute
)

type AccountModelImportFailure struct {
	Model string `json:"model"`
	Error string `json:"error"`
}

type AccountModelImportModelResult struct {
	SourceModel    string `json:"source_model"`
	CanonicalModel string `json:"canonical_model,omitempty"`
	RegistryModel  string `json:"registry_model,omitempty"`
	Status         string `json:"status"`
	ReasonCode     string `json:"reason_code"`
	Detail         string `json:"detail,omitempty"`
}

type AccountModelImportResult struct {
	AccountID      int64                           `json:"account_id"`
	DetectedModels []string                        `json:"detected_models"`
	ImportedModels []string                        `json:"imported_models"`
	ImportedCount  int                             `json:"imported_count"`
	SkippedCount   int                             `json:"skipped_count"`
	FailedModels   []AccountModelImportFailure     `json:"failed_models,omitempty"`
	ModelResults   []AccountModelImportModelResult `json:"model_results,omitempty"`
	ProbeSource    string                          `json:"probe_source"`
	ProbeNotice    string                          `json:"probe_notice,omitempty"`
	Trigger        string                          `json:"trigger"`
}

type AccountModelProbeSummary struct {
	DetectedModels          []string                 `json:"detected_models"`
	Models                  []AccountModelProbeModel `json:"models,omitempty"`
	ProbeSource             string                   `json:"probe_source"`
	ProbeNotice             string                   `json:"probe_notice,omitempty"`
	ResolvedUpstreamURL     string                   `json:"resolved_upstream_url,omitempty"`
	ResolvedUpstreamHost    string                   `json:"resolved_upstream_host,omitempty"`
	ResolvedUpstreamService string                   `json:"resolved_upstream_service,omitempty"`
}

type AccountModelProbeModel struct {
	ID                 string `json:"id"`
	DisplayName        string `json:"display_name"`
	Provider           string `json:"provider,omitempty"`
	ProviderLabel      string `json:"provider_label,omitempty"`
	SourceProtocol     string `json:"source_protocol,omitempty"`
	UpstreamSource     string `json:"upstream_source,omitempty"`
	Availability       string `json:"availability,omitempty"`
	AvailabilityReason string `json:"availability_reason,omitempty"`
}

type accountModelProbeResult struct {
	Models           []string
	Details          []AccountModelProbeModel
	Source           string
	Notice           string
	ResolvedUpstream ResolvedUpstreamInfo
}

func newAccountModelProbeResult(models []string) *accountModelProbeResult {
	details := make([]AccountModelProbeModel, 0, len(models))
	for _, modelID := range models {
		details = append(details, applyAccountModelProbeProvider(AccountModelProbeModel{
			ID:          modelID,
			DisplayName: FormatModelCatalogDisplayName(modelID),
		}, ""))
	}
	return &accountModelProbeResult{
		Models:  models,
		Details: details,
		Source:  accountModelProbeSourceUpstream,
	}
}

type AccountModelImportService struct {
	modelCatalogService          *ModelCatalogService
	modelRegistryService         *ModelRegistryService
	geminiCompatService          *GeminiMessagesCompatService
	kiroRuntimeService           *KiroRuntimeService
	vertexCatalogService         VertexCatalogProvider
	openAITokenProvider          *OpenAITokenProvider
	httpUpstream                 HTTPUpstream
	proxyRepo                    ProxyRepository
	tlsFingerprintProfileService *TLSFingerprintProfileService
	probeCache                   *gocache.Cache
	probeSF                      singleflight.Group
}

func NewAccountModelImportService(
	modelCatalogService *ModelCatalogService,
	geminiCompatService *GeminiMessagesCompatService,
	httpUpstream HTTPUpstream,
	proxyRepo ProxyRepository,
) *AccountModelImportService {
	return &AccountModelImportService{
		modelCatalogService: modelCatalogService,
		geminiCompatService: geminiCompatService,
		httpUpstream:        httpUpstream,
		proxyRepo:           proxyRepo,
		probeCache:          gocache.New(accountModelProbeCacheTTL, time.Minute),
	}
}

func (s *AccountModelImportService) SetModelRegistryService(modelRegistryService *ModelRegistryService) {
	s.modelRegistryService = modelRegistryService
}

func (s *AccountModelImportService) SetOpenAITokenProvider(openAITokenProvider *OpenAITokenProvider) {
	s.openAITokenProvider = openAITokenProvider
}

func (s *AccountModelImportService) SetKiroRuntimeService(kiroRuntimeService *KiroRuntimeService) {
	s.kiroRuntimeService = kiroRuntimeService
}

func (s *AccountModelImportService) SetVertexCatalogService(vertexCatalogService VertexCatalogProvider) {
	s.vertexCatalogService = vertexCatalogService
}

func (s *AccountModelImportService) SetTLSFingerprintProfileService(tlsFingerprintProfileService *TLSFingerprintProfileService) {
	s.tlsFingerprintProfileService = tlsFingerprintProfileService
}

func (s *AccountModelImportService) ProbeAccountModels(ctx context.Context, account *Account) (*AccountModelProbeSummary, error) {
	return s.ListAccountModels(ctx, account, true)
}

func (s *AccountModelImportService) ListAccountModels(
	ctx context.Context,
	account *Account,
	forceRefresh bool,
) (*AccountModelProbeSummary, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}

	probeResult, err := s.loadProbeResult(ctx, account, forceRefresh)
	if err != nil {
		return nil, err
	}
	if probeResult == nil {
		return nil, infraerrors.InternalServer("MODEL_IMPORT_PROBE_RESULT_MISSING", "model import probe result is missing")
	}

	detectedModels, _ := normalizeImportedModelIDs(probeResult.Models)
	probeSource := strings.TrimSpace(probeResult.Source)
	if probeSource == "" {
		probeSource = accountModelProbeSourceUpstream
	}
	resolvedUpstream := probeResult.ResolvedUpstream

	return &AccountModelProbeSummary{
		DetectedModels:          detectedModels,
		Models:                  decorateAccountModelProbeDetails(normalizeAccountModelProbeDetails(probeResult.Details, detectedModels), accountModelProbeProviderForPlatform(RoutingPlatformForAccount(account))),
		ProbeSource:             probeSource,
		ProbeNotice:             strings.TrimSpace(probeResult.Notice),
		ResolvedUpstreamURL:     strings.TrimSpace(resolvedUpstream.URL),
		ResolvedUpstreamHost:    strings.TrimSpace(resolvedUpstream.Host),
		ResolvedUpstreamService: strings.TrimSpace(resolvedUpstream.Service),
	}, nil
}

func decorateAccountModelProbeDetails(details []AccountModelProbeModel, provider string) []AccountModelProbeModel {
	for index := range details {
		details[index] = applyAccountModelProbeProvider(details[index], provider)
	}
	return details
}

func (s *AccountModelImportService) ImportAccountModels(ctx context.Context, account *Account, trigger string, selectedModels ...[]string) (*AccountModelImportResult, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}
	if !account.IsActive() {
		return nil, infraerrors.BadRequest("ACCOUNT_INACTIVE", "account must be active to import models")
	}
	runtimePlatform := RoutingPlatformForAccount(account)
	log := logger.FromContext(ctx)
	log.Info("account model import: started",
		zap.Int64("account_id", account.ID),
		zap.String("platform", runtimePlatform),
		zap.String("type", account.Type),
		zap.String("trigger", normalizeImportTrigger(trigger)),
	)
	probeResult, err := s.loadProbeResult(ctx, account, true)
	if err != nil {
		log.Warn("account model import: detect models failed",
			zap.Int64("account_id", account.ID),
			zap.String("platform", runtimePlatform),
			zap.String("type", account.Type),
			zap.Error(err),
		)
		return nil, err
	}
	if probeResult == nil {
		return nil, infraerrors.InternalServer("MODEL_IMPORT_PROBE_RESULT_MISSING", "model import probe result is missing")
	}
	filteredProbeModels := filterSelectedImportedModels(probeResult.Models, selectedModels...)
	if len(filteredProbeModels) == 0 {
		return nil, infraerrors.BadRequest("MODEL_IMPORT_EMPTY", "no models detected for account")
	}
	if s.modelRegistryService == nil {
		if s.modelCatalogService != nil && s.modelCatalogService.settingRepo != nil {
			s.modelRegistryService = NewModelRegistryService(s.modelCatalogService.settingRepo)
		} else {
			return nil, infraerrors.InternalServer("MODEL_CATALOG_SERVICE_UNAVAILABLE", "model catalog service is unavailable")
		}
	}

	probeSource := strings.TrimSpace(probeResult.Source)
	if probeSource == "" {
		probeSource = accountModelProbeSourceUpstream
	}
	uniqueDetected, _ := normalizeImportedModelIDs(filteredProbeModels)
	result := &AccountModelImportResult{
		AccountID:      account.ID,
		DetectedModels: uniqueDetected,
		ProbeSource:    probeSource,
		ProbeNotice:    strings.TrimSpace(probeResult.Notice),
		Trigger:        normalizeImportTrigger(trigger),
	}
	canonicalRegistryModels := make(map[string]string, len(filteredProbeModels))
	for _, sourceModel := range sortImportedSourceModels(filteredProbeModels) {
		sourceModel = strings.TrimSpace(sourceModel)
		sourceRegistryID := normalizeRegistryID(sourceModel)
		canonicalModel := sourceRegistryID
		if resolved, ok := modelregistry.ResolveToCanonicalID(sourceRegistryID); ok {
			canonicalModel = resolved
		} else if explanation, err := s.modelRegistryService.ExplainResolution(ctx, sourceRegistryID); err == nil && explanation != nil {
			if explanation.EffectiveID != "" {
				canonicalModel = explanation.EffectiveID
			} else if explanation.CanonicalID != "" {
				canonicalModel = explanation.CanonicalID
			}
		}
		if sourceRegistryID == "" {
			result.ModelResults = append(result.ModelResults, AccountModelImportModelResult{
				SourceModel: sourceModel,
				Status:      "failed",
				ReasonCode:  "invalid_model_id",
				Detail:      "model id is empty after normalization",
			})
			result.FailedModels = append(result.FailedModels, AccountModelImportFailure{Model: sourceModel, Error: "invalid model id"})
			continue
		}
		if registryModelID, exists := canonicalRegistryModels[canonicalModel]; exists {
			result.SkippedCount++
			modelResult := AccountModelImportModelResult{
				SourceModel:    sourceModel,
				CanonicalModel: canonicalModel,
				Status:         "skipped",
				ReasonCode:     "duplicate_canonical",
			}
			if registryModelID != "" {
				modelResult.RegistryModel = registryModelID
			}
			result.ModelResults = append(result.ModelResults, modelResult)
			continue
		}
		canonicalRegistryModels[canonicalModel] = ""
		registryResult, registryErr := s.modelRegistryService.UpsertDiscoveredEntry(ctx, UpsertDiscoveredEntryInput{
			ModelID:        sourceRegistryID,
			SourcePlatform: runtimePlatform,
		})
		if registryErr != nil {
			detail := summarizeAccountModelImportError(registryErr)
			result.ModelResults = append(result.ModelResults, AccountModelImportModelResult{
				SourceModel:    sourceModel,
				CanonicalModel: canonicalModel,
				Status:         "failed",
				ReasonCode:     inferAccountModelImportReasonCode(registryErr),
				Detail:         detail,
			})
			result.FailedModels = append(result.FailedModels, AccountModelImportFailure{Model: sourceModel, Error: detail})
			log.Warn("account model import: upsert registry entry failed", zap.Int64("account_id", account.ID), zap.String("platform", runtimePlatform), zap.String("model", sourceRegistryID), zap.Error(registryErr))
			continue
		}
		if registryResult == nil {
			continue
		}
		canonicalModel = registryResult.CanonicalModel
		if canonicalModel == "" {
			canonicalModel = sourceRegistryID
		}
		if registryResult.RegistryModelID != "" {
			canonicalRegistryModels[canonicalModel] = registryResult.RegistryModelID
		}
		if registryResult.Blocked {
			result.SkippedCount++
			result.ModelResults = append(result.ModelResults, AccountModelImportModelResult{
				SourceModel:    sourceModel,
				CanonicalModel: canonicalModel,
				Status:         "skipped",
				ReasonCode:     "blocked_tombstone",
			})
			continue
		}
		if registryResult.Changed {
			result.ImportedModels = append(result.ImportedModels, registryResult.RegistryModelID)
			result.ModelResults = append(result.ModelResults, AccountModelImportModelResult{
				SourceModel:    sourceModel,
				CanonicalModel: canonicalModel,
				RegistryModel:  registryResult.RegistryModelID,
				Status:         importResultStatus(sourceRegistryID, canonicalModel),
				ReasonCode:     importResultReasonCode(sourceRegistryID, canonicalModel, true),
			})
			continue
		}
		result.SkippedCount++
		result.ModelResults = append(result.ModelResults, AccountModelImportModelResult{
			SourceModel:    sourceModel,
			CanonicalModel: canonicalModel,
			RegistryModel:  registryResult.RegistryModelID,
			Status:         importExistingResultStatus(sourceRegistryID, registryResult.RegistryModelID, canonicalModel),
			ReasonCode:     importExistingReasonCode(sourceRegistryID, registryResult.RegistryModelID, canonicalModel),
		})
	}
	result.ImportedCount = len(result.ImportedModels)

	log.Info("account model import: completed",
		zap.Int64("account_id", account.ID),
		zap.String("platform", runtimePlatform),
		zap.String("trigger", result.Trigger),
		zap.String("probe_source", result.ProbeSource),
		zap.Int("detected_count", len(result.DetectedModels)),
		zap.Int("imported_count", result.ImportedCount),
		zap.Int("skipped_count", result.SkippedCount),
		zap.Int("failed_count", len(result.FailedModels)),
	)
	return result, nil
}

func (s *AccountModelImportService) loadProbeResult(
	ctx context.Context,
	account *Account,
	forceRefresh bool,
) (*accountModelProbeResult, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}
	if s == nil {
		return nil, infraerrors.InternalServer("MODEL_IMPORT_SERVICE_UNAVAILABLE", "account model import service is unavailable")
	}
	cacheKey := s.probeCacheKey(account)
	if forceRefresh || s.probeCache == nil {
		result, err := s.detectModels(ctx, account)
		if err != nil {
			return nil, err
		}
		result = s.mergeManualModelsIntoProbeResult(account, result)
		if s.probeCache != nil {
			s.probeCache.Set(cacheKey, cloneAccountModelProbeResult(result), accountModelProbeCacheTTL)
		}
		return cloneAccountModelProbeResult(result), nil
	}
	if cached, ok := s.probeCache.Get(cacheKey); ok {
		if result, castOK := cached.(*accountModelProbeResult); castOK && result != nil {
			return cloneAccountModelProbeResult(result), nil
		}
	}
	value, err, _ := s.probeSF.Do(cacheKey, func() (any, error) {
		if cached, ok := s.probeCache.Get(cacheKey); ok {
			if result, castOK := cached.(*accountModelProbeResult); castOK && result != nil {
				return cloneAccountModelProbeResult(result), nil
			}
		}
		result, detectErr := s.detectModels(ctx, account)
		if detectErr != nil {
			return nil, detectErr
		}
		result = s.mergeManualModelsIntoProbeResult(account, result)
		cloned := cloneAccountModelProbeResult(result)
		s.probeCache.Set(cacheKey, cloned, accountModelProbeCacheTTL)
		return cloneAccountModelProbeResult(cloned), nil
	})
	if err != nil {
		return nil, err
	}
	result, _ := value.(*accountModelProbeResult)
	return cloneAccountModelProbeResult(result), nil
}

func (s *AccountModelImportService) probeCacheKey(account *Account) string {
	if account == nil {
		return ""
	}
	groupIDs := append([]int64(nil), account.GroupIDs...)
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })
	groupParts := make([]string, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		groupParts = append(groupParts, fmt.Sprintf("%d", groupID))
	}

	baseURL := strings.TrimSpace(account.GetBaseURL())
	projectID := ""
	location := ""
	if RoutingPlatformForAccount(account) == PlatformGemini {
		baseURL = strings.TrimSpace(geminiBaseURLForLogging(account))
		projectID = strings.TrimSpace(account.GetCredential("project_id"))
		if account.IsGeminiVertexSource() {
			projectID = strings.TrimSpace(account.GetGeminiVertexProjectID())
			location = strings.TrimSpace(account.GetGeminiVertexLocation())
		}
	}
	if RoutingPlatformForAccount(account) == PlatformAntigravity && account.Type == AccountTypeOAuth {
		projectID = strings.TrimSpace(account.GetCredential("project_id"))
	}
	acceptedProtocols := append([]string(nil), GetAccountGatewayAcceptedProtocols(account)...)
	sort.Strings(acceptedProtocols)
	manualModels := AccountManualModelsFromExtra(account.Extra, IsProtocolGatewayAccount(account))

	return strings.Join([]string{
		"groups=" + strings.Join(groupParts, ","),
		fmt.Sprintf("account=%d", account.ID),
		"platform=" + strings.TrimSpace(strings.ToLower(RoutingPlatformForAccount(account))),
		"type=" + strings.TrimSpace(strings.ToLower(account.Type)),
		"auth=" + accountModelImportAuthMode(account),
		"base=" + baseURL,
		"project=" + projectID,
		"location=" + location,
		"gateway=" + strings.TrimSpace(GetAccountGatewayProtocol(account)),
		"accepted=" + strings.Join(acceptedProtocols, ","),
		"mapping=" + accountModelImportMappingSignature(account.GetModelMapping()),
		"manual=" + accountManualModelsSignature(manualModels),
	}, "|")
}

func accountModelImportAuthMode(account *Account) string {
	if account == nil {
		return ""
	}
	switch {
	case account.IsGeminiVertexExpress():
		return "vertex_express_key"
	case account.IsGeminiVertexAI():
		return "vertex_service_account"
	case account.Type == AccountTypeOAuth && account.IsGeminiCodeAssist():
		return "gemini_code_assist_oauth"
	case account.Type == AccountTypeOAuth:
		return "oauth"
	case account.Type == AccountTypeAPIKey:
		return "api_key"
	case account.Type == AccountTypeUpstream:
		return "upstream"
	default:
		return strings.TrimSpace(strings.ToLower(account.Type))
	}
}

func accountModelImportMappingSignature(mapping map[string]string) string {
	if len(mapping) == 0 {
		return ""
	}
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(keys))
	for _, key := range keys {
		ordered[key] = mapping[key]
	}
	payload, err := json.Marshal(ordered)
	if err != nil {
		return ""
	}
	return string(payload)
}

func cloneAccountModelProbeResult(result *accountModelProbeResult) *accountModelProbeResult {
	if result == nil {
		return nil
	}
	return &accountModelProbeResult{
		Models:           append([]string(nil), result.Models...),
		Details:          append([]AccountModelProbeModel(nil), result.Details...),
		Source:           result.Source,
		Notice:           result.Notice,
		ResolvedUpstream: result.ResolvedUpstream,
	}
}

func filterSelectedImportedModels(models []string, selectedModels ...[]string) []string {
	if len(selectedModels) == 0 || len(selectedModels[0]) == 0 {
		return append([]string{}, models...)
	}
	selected := make(map[string]struct{}, len(selectedModels[0]))
	for _, modelID := range selectedModels[0] {
		if normalized := normalizeRegistryID(modelID); normalized != "" {
			selected[normalized] = struct{}{}
		}
	}
	if len(selected) == 0 {
		return append([]string{}, models...)
	}
	filtered := make([]string, 0, len(models))
	for _, modelID := range models {
		if normalized := normalizeRegistryID(modelID); normalized != "" {
			if _, ok := selected[normalized]; ok {
				filtered = append(filtered, modelID)
			}
		}
	}
	return filtered
}

func sortImportedSourceModels(models []string) []string {
	items := append([]string(nil), models...)
	sort.SliceStable(items, func(i, j int) bool {
		left := normalizeRegistryID(items[i])
		right := normalizeRegistryID(items[j])
		if len(left) == len(right) {
			return left < right
		}
		return len(left) < len(right)
	})
	return items
}

func normalizeAccountModelProbeDetails(details []AccountModelProbeModel, detectedModels []string) []AccountModelProbeModel {
	if len(detectedModels) == 0 {
		return []AccountModelProbeModel{}
	}
	detailByID := make(map[string]AccountModelProbeModel, len(details))
	for _, detail := range details {
		modelID := strings.TrimSpace(detail.ID)
		if modelID == "" {
			continue
		}
		detail.ID = modelID
		if strings.TrimSpace(detail.DisplayName) == "" {
			detail.DisplayName = FormatModelCatalogDisplayName(modelID)
		}
		detail = applyAccountModelProbeProvider(detail, detail.Provider)
		detail.SourceProtocol = NormalizeGatewayProtocol(detail.SourceProtocol)
		detail.UpstreamSource = strings.TrimSpace(detail.UpstreamSource)
		detail.Availability = strings.TrimSpace(detail.Availability)
		detail.AvailabilityReason = strings.TrimSpace(detail.AvailabilityReason)
		if _, exists := detailByID[modelID]; !exists {
			detailByID[modelID] = detail
		}
	}
	result := make([]AccountModelProbeModel, 0, len(detectedModels))
	for _, modelID := range detectedModels {
		if detail, ok := detailByID[modelID]; ok {
			result = append(result, detail)
			continue
		}
		result = append(result, applyAccountModelProbeProvider(AccountModelProbeModel{
			ID:          modelID,
			DisplayName: FormatModelCatalogDisplayName(modelID),
		}, ""))
	}
	sort.SliceStable(result, func(i, j int) bool {
		leftSortKey := FinalDisplayNameSortKey(result[i].Provider, result[i].ProviderLabel, result[i].DisplayName, result[i].ID)
		rightSortKey := FinalDisplayNameSortKey(result[j].Provider, result[j].ProviderLabel, result[j].DisplayName, result[j].ID)
		if leftSortKey != rightSortKey {
			return leftSortKey < rightSortKey
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func (s *AccountModelImportService) mergeManualModelsIntoProbeResult(account *Account, result *accountModelProbeResult) *accountModelProbeResult {
	if result == nil {
		return nil
	}
	manualModels := AccountManualModelsFromExtra(account.Extra, IsProtocolGatewayAccount(account))
	if len(manualModels) == 0 {
		return result
	}
	mergedModels := append([]string(nil), result.Models...)
	mergedDetails := append([]AccountModelProbeModel(nil), result.Details...)
	seen := make(map[string]struct{}, len(mergedDetails))
	for _, detail := range mergedDetails {
		if modelID := NormalizeModelCatalogModelID(detail.ID); modelID != "" {
			seen[modelID] = struct{}{}
		}
	}
	for _, manualModel := range manualModels {
		modelID := NormalizeModelCatalogModelID(manualModel.ModelID)
		if modelID == "" {
			continue
		}
		mergedModels = append(mergedModels, modelID)
		if _, exists := seen[modelID]; exists {
			for index := range mergedDetails {
				if NormalizeModelCatalogModelID(mergedDetails[index].ID) != modelID {
					continue
				}
				if mergedDetails[index].SourceProtocol == "" {
					mergedDetails[index].SourceProtocol = manualModel.SourceProtocol
				}
				mergedDetails[index] = applyAccountModelProbeProvider(mergedDetails[index], manualModel.Provider)
				break
			}
			continue
		}
		seen[modelID] = struct{}{}
		mergedDetails = append(mergedDetails, applyAccountModelProbeProvider(AccountModelProbeModel{
			ID:             modelID,
			DisplayName:    FormatModelCatalogDisplayName(modelID),
			SourceProtocol: manualModel.SourceProtocol,
			UpstreamSource: "manual",
			Availability:   "manual",
		}, manualModel.Provider))
	}
	result.Models = mergedModels
	result.Details = mergedDetails
	return result
}

func accountModelProbeProviderForPlatform(platform string) string {
	return ProviderForPlatform(platform)
}

func applyAccountModelProbeProvider(detail AccountModelProbeModel, provider string) AccountModelProbeModel {
	normalized := NormalizeModelProvider(provider)
	if normalized == "" {
		normalized = NormalizeModelProvider(detail.Provider)
	}
	if normalized == "" {
		normalized = NormalizeModelProvider(detail.SourceProtocol)
	}
	detail.Provider = normalized
	if normalized != "" {
		detail.ProviderLabel = FormatProviderLabel(normalized)
	}
	return detail
}

func accountManualModelsSignature(models []AccountManualModel) string {
	if len(models) == 0 {
		return ""
	}
	payload, err := json.Marshal(models)
	if err != nil {
		return ""
	}
	return string(payload)
}

func summarizeAccountModelImportError(err error) string {
	if err == nil {
		return ""
	}
	message := infraerrors.Message(err)
	if message != "" && message != infraerrors.UnknownMessage {
		return message
	}
	return err.Error()
}

func normalizeImportTrigger(trigger string) string {
	switch trigger {
	case "create", "manual":
		return trigger
	case "":
		return "manual"
	default:
		return fmt.Sprintf("custom:%s", trigger)
	}
}

func normalizeImportedModelIDs(models []string) ([]string, int) {
	unique := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	skipped := 0
	for _, model := range models {
		normalized := NormalizeModelCatalogModelID(model)
		if normalized == "" {
			skipped++
			continue
		}
		if _, exists := seen[normalized]; exists {
			skipped++
			continue
		}
		seen[normalized] = struct{}{}
		unique = append(unique, normalized)
	}
	return unique, skipped
}

func importResultStatus(sourceModelID string, canonicalModel string) string {
	if canonicalModel != "" && sourceModelID != canonicalModel {
		return "merged"
	}
	return "imported"
}

func importResultReasonCode(sourceModelID string, canonicalModel string, changed bool) string {
	if !changed {
		return "already_exists"
	}
	if canonicalModel != "" && sourceModelID != canonicalModel {
		return "merged_canonical"
	}
	return "imported_new"
}

func importExistingResultStatus(sourceModelID string, registryModelID string, canonicalModel string) string {
	if registryModelID != "" && registryModelID != sourceModelID {
		return "merged"
	}
	if canonicalModel != "" && canonicalModel != sourceModelID {
		return "merged"
	}
	return "skipped"
}

func importExistingReasonCode(sourceModelID string, registryModelID string, canonicalModel string) string {
	if importExistingResultStatus(sourceModelID, registryModelID, canonicalModel) == "merged" {
		return "merged_canonical"
	}
	return "already_exists"
}

func inferAccountModelImportReasonCode(err error) string {
	if err == nil {
		return "persist_failed"
	}
	if infraerrors.Reason(err) == "MODEL_RUNTIME_PLATFORM_UNSUPPORTED" {
		return "unsupported_runtime_platform"
	}
	if infraerrors.Reason(err) == "MODEL_REQUIRED" {
		return "invalid_model_id"
	}
	return "persist_failed"
}
