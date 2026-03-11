package service

import (
	"context"
	"fmt"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
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
	Trigger        string                      `json:"trigger"`
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
	detectedModels, err := s.detectModels(ctx, account)
	if err != nil {
		log.Warn("account model import: detect models failed",
			zap.Int64("account_id", account.ID),
			zap.String("platform", account.Platform),
			zap.String("type", account.Type),
			zap.Error(err),
		)
		return nil, err
	}
	if len(detectedModels) == 0 {
		return nil, infraerrors.BadRequest("MODEL_IMPORT_EMPTY", "no models detected for account")
	}
	if s.modelCatalogService == nil {
		return nil, infraerrors.InternalServer("MODEL_CATALOG_SERVICE_UNAVAILABLE", "model catalog service is unavailable")
	}

	uniqueDetected, skippedCount := normalizeImportedModelIDs(detectedModels)
	result := &AccountModelImportResult{
		AccountID:      account.ID,
		DetectedModels: uniqueDetected,
		SkippedCount:   skippedCount,
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
