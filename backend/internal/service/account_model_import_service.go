package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	accountModelProbeSourceUpstream                 = "upstream"
	accountModelProbeSourceGeminiCLIDefaultFallback = "gemini_cli_default_fallback"
	accountModelProbeSourceKiroBuiltinCatalog       = "kiro_builtin_catalog"
	accountModelProbeSourceCopilotStaticFallback    = "copilot_static_fallback"
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
	DetectedModels []string `json:"detected_models"`
	ProbeSource    string   `json:"probe_source"`
	ProbeNotice    string   `json:"probe_notice,omitempty"`
}

type accountModelProbeResult struct {
	Models []string
	Source string
	Notice string
}

func newAccountModelProbeResult(models []string) *accountModelProbeResult {
	return &accountModelProbeResult{
		Models: models,
		Source: accountModelProbeSourceUpstream,
	}
}

type AccountModelImportService struct {
	modelCatalogService  *ModelCatalogService
	modelRegistryService *ModelRegistryService
	geminiCompatService  *GeminiMessagesCompatService
	openAITokenProvider  *OpenAITokenProvider
	httpUpstream         HTTPUpstream
	proxyRepo            ProxyRepository
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
	}
}

func (s *AccountModelImportService) SetModelRegistryService(modelRegistryService *ModelRegistryService) {
	s.modelRegistryService = modelRegistryService
}

func (s *AccountModelImportService) SetOpenAITokenProvider(openAITokenProvider *OpenAITokenProvider) {
	s.openAITokenProvider = openAITokenProvider
}

func (s *AccountModelImportService) ProbeAccountModels(ctx context.Context, account *Account) (*AccountModelProbeSummary, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}

	probeResult, err := s.detectModels(ctx, account)
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

	return &AccountModelProbeSummary{
		DetectedModels: detectedModels,
		ProbeSource:    probeSource,
		ProbeNotice:    strings.TrimSpace(probeResult.Notice),
	}, nil
}

func (s *AccountModelImportService) ImportAccountModels(ctx context.Context, account *Account, trigger string) (*AccountModelImportResult, error) {
	if account == nil {
		return nil, infraerrors.BadRequest("ACCOUNT_REQUIRED", "account is required")
	}
	if !account.IsActive() {
		return nil, infraerrors.BadRequest("ACCOUNT_INACTIVE", "account must be active to import models")
	}
	log := logger.FromContext(ctx)
	log.Info("account model import: started",
		zap.Int64("account_id", account.ID),
		zap.String("platform", account.Platform),
		zap.String("type", account.Type),
		zap.String("trigger", normalizeImportTrigger(trigger)),
	)
	probeResult, err := s.detectModels(ctx, account)
	if err != nil {
		log.Warn("account model import: detect models failed",
			zap.Int64("account_id", account.ID),
			zap.String("platform", account.Platform),
			zap.String("type", account.Type),
			zap.Error(err),
		)
		return nil, err
	}
	if probeResult == nil {
		return nil, infraerrors.InternalServer("MODEL_IMPORT_PROBE_RESULT_MISSING", "model import probe result is missing")
	}
	if len(probeResult.Models) == 0 {
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
	uniqueDetected, _ := normalizeImportedModelIDs(probeResult.Models)
	result := &AccountModelImportResult{
		AccountID:      account.ID,
		DetectedModels: uniqueDetected,
		ProbeSource:    probeSource,
		ProbeNotice:    strings.TrimSpace(probeResult.Notice),
		Trigger:        normalizeImportTrigger(trigger),
	}
	canonicalRegistryModels := make(map[string]string, len(probeResult.Models))
	for _, sourceModel := range sortImportedSourceModels(probeResult.Models) {
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
			SourcePlatform: account.Platform,
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
			log.Warn("account model import: upsert registry entry failed", zap.Int64("account_id", account.ID), zap.String("platform", account.Platform), zap.String("model", sourceRegistryID), zap.Error(registryErr))
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
		zap.String("platform", account.Platform),
		zap.String("trigger", result.Trigger),
		zap.String("probe_source", result.ProbeSource),
		zap.Int("detected_count", len(result.DetectedModels)),
		zap.Int("imported_count", result.ImportedCount),
		zap.Int("skipped_count", result.SkippedCount),
		zap.Int("failed_count", len(result.FailedModels)),
	)
	return result, nil
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
