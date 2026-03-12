package service

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	accountModelProbeSourceUpstream                 = "upstream"
	accountModelProbeSourceGeminiCLIDefaultFallback = "gemini_cli_default_fallback"
)

type AccountModelImportFailure struct {
	Model string `json:"model"`
	Error string `json:"error"`
}

type AccountModelImportResult struct {
	AccountID      int64                       `json:"account_id"`
	DetectedModels []string                    `json:"detected_models"`
	ImportedModels []string                    `json:"imported_models"`
	ImportedCount  int                         `json:"imported_count"`
	SkippedCount   int                         `json:"skipped_count"`
	FailedModels   []AccountModelImportFailure `json:"failed_models,omitempty"`
	ProbeSource    string                      `json:"probe_source"`
	ProbeNotice    string                      `json:"probe_notice,omitempty"`
	Trigger        string                      `json:"trigger"`
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
	modelCatalogService *ModelCatalogService
	geminiCompatService *GeminiMessagesCompatService
	httpUpstream        HTTPUpstream
	proxyRepo           ProxyRepository
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
	if s.modelCatalogService == nil {
		return nil, infraerrors.InternalServer("MODEL_CATALOG_SERVICE_UNAVAILABLE", "model catalog service is unavailable")
	}

	probeSource := strings.TrimSpace(probeResult.Source)
	if probeSource == "" {
		probeSource = accountModelProbeSourceUpstream
	}
	uniqueDetected, skippedCount := normalizeImportedModelIDs(probeResult.Models)
	result := &AccountModelImportResult{
		AccountID:      account.ID,
		DetectedModels: uniqueDetected,
		SkippedCount:   skippedCount,
		ProbeSource:    probeSource,
		ProbeNotice:    strings.TrimSpace(probeResult.Notice),
		Trigger:        normalizeImportTrigger(trigger),
	}

	for _, model := range uniqueDetected {
		if _, err := s.modelCatalogService.UpsertCatalogEntry(ctx, UpsertModelCatalogEntryInput{Model: model}); err != nil {
			result.FailedModels = append(result.FailedModels, AccountModelImportFailure{Model: model, Error: summarizeAccountModelImportError(err)})
			log.Warn("account model import: upsert catalog entry failed", zap.Int64("account_id", account.ID), zap.String("platform", account.Platform), zap.String("model", model), zap.Error(err))
			continue
		}
		result.ImportedModels = append(result.ImportedModels, model)
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
