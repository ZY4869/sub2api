package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	documentAIJobPollerName    = "document_ai_job_poller"
	documentAIJobPollInterval  = 15 * time.Second
	documentAIJobPollBatchSize = 50
	documentAIJobPollTaskTTL   = 2 * time.Minute
)

var ErrDocumentAIDisabled = infraerrors.ServiceUnavailable("document_ai_disabled", "document ai service is disabled")

type DocumentAIService struct {
	repo                         DocumentAIJobRepository
	accountRepo                  AccountRepository
	httpUpstream                 HTTPUpstream
	tlsFingerprintProfileService *TLSFingerprintProfileService
	timingWheel                  *TimingWheelService
	settingService               *SettingService

	startOnce sync.Once
	stopOnce  sync.Once
	running   int32
}

func NewDocumentAIService(
	repo DocumentAIJobRepository,
	accountRepo AccountRepository,
	httpUpstream HTTPUpstream,
	tlsFingerprintProfileService *TLSFingerprintProfileService,
	timingWheel *TimingWheelService,
	settingService *SettingService,
) *DocumentAIService {
	return &DocumentAIService{
		repo:                         repo,
		accountRepo:                  accountRepo,
		httpUpstream:                 httpUpstream,
		tlsFingerprintProfileService: tlsFingerprintProfileService,
		timingWheel:                  timingWheel,
		settingService:               settingService,
	}
}

func ProvideDocumentAIService(
	repo DocumentAIJobRepository,
	accountRepo AccountRepository,
	httpUpstream HTTPUpstream,
	tlsFingerprintProfileService *TLSFingerprintProfileService,
	timingWheel *TimingWheelService,
	settingService *SettingService,
) *DocumentAIService {
	svc := NewDocumentAIService(repo, accountRepo, httpUpstream, tlsFingerprintProfileService, timingWheel, settingService)
	svc.Start()
	return svc
}

func (s *DocumentAIService) Start() {
	if s == nil || s.repo == nil || s.accountRepo == nil || s.httpUpstream == nil || s.timingWheel == nil {
		logger.LegacyPrintf("service.document_ai", "[DocumentAI] poller not started (missing deps)")
		return
	}
	s.startOnce.Do(func() {
		s.timingWheel.ScheduleRecurring(documentAIJobPollerName, documentAIJobPollInterval, s.runPoller)
		logger.LegacyPrintf("service.document_ai", "[DocumentAI] poller started interval=%s batch=%d", documentAIJobPollInterval, documentAIJobPollBatchSize)
	})
}

func (s *DocumentAIService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.timingWheel != nil {
			s.timingWheel.Cancel(documentAIJobPollerName)
		}
		logger.LegacyPrintf("service.document_ai", "[DocumentAI] poller stopped")
	})
}

func (s *DocumentAIService) RequireEnabled(ctx context.Context) error {
	if s == nil || s.settingService == nil || !s.settingService.IsDocumentAIEnabled(ctx) {
		return ErrDocumentAIDisabled
	}
	return nil
}

func (s *DocumentAIService) ListModels(ctx context.Context, groupID int64) ([]DocumentAIModelDescriptor, error) {
	if err := s.RequireEnabled(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.accountRepo == nil {
		return BuiltinDocumentAIModels(), nil
	}
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, groupID, PlatformBaiduDocumentAI)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return BuiltinDocumentAIModels(), nil
	}
	return documentAIUnionModelsForAccounts(accounts), nil
}

func (s *DocumentAIService) GetJob(ctx context.Context, jobID string, userID int64) (*DocumentAIJob, error) {
	if err := s.RequireEnabled(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai service is not ready")
	}
	return s.repo.GetByJobIDForUser(ctx, strings.TrimSpace(jobID), userID)
}

func (s *DocumentAIService) SubmitJob(ctx context.Context, input DocumentAISubmitJobInput) (*DocumentAIJob, error) {
	if err := s.RequireEnabled(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil || s.accountRepo == nil {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai service is not ready")
	}
	normalizedInput, err := s.normalizeSubmitInput(input)
	if err != nil {
		return nil, err
	}
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, normalizedInput.GroupID, PlatformBaiduDocumentAI)
	if err != nil {
		return nil, err
	}
	account, targetModelID, err := documentAISelectAccountForDisplayModel(accounts, normalizedInput.Model, DocumentAIJobModeAsync)
	if err != nil {
		return nil, err
	}

	job := newDocumentAIJob(normalizedInput.APIKey, normalizedInput.GroupID, account, DocumentAIJobModeAsync, normalizedInput.Model, normalizedInput.SourceType, normalizedInput.FileName, normalizedInput.ContentType, normalizedInput.FileSize, normalizedInput.FileHash)
	if err := s.repo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create document ai job: %w", err)
	}

	s.logDocumentAIInfo(ctx, "submit_start", job, account,
		zap.String("source_type", normalizedInput.SourceType),
		zap.String("target_model", targetModelID),
	)
	client := newBaiduDocumentAIClient(s.httpUpstream, s.tlsFingerprintProfileService)
	routeInput := normalizedInput
	routeInput.Model = targetModelID
	result, err := client.submitAsyncJob(ctx, account, routeInput)
	if err != nil {
		s.markJobFailed(ctx, "submit_failed", job, documentAIProviderResultJSONFromError(err), err)
		return nil, err
	}
	var providerJobID, providerBatchID, providerRaw *string
	if result.ProviderJobID != "" {
		providerJobID = stringPtr(result.ProviderJobID)
		job.ProviderJobID = providerJobID
	}
	if result.ProviderBatchID != "" {
		providerBatchID = stringPtr(result.ProviderBatchID)
		job.ProviderBatchID = providerBatchID
	}
	if result.ProviderRawJSON != "" {
		providerRaw = stringPtr(result.ProviderRawJSON)
		job.ProviderResultJSON = providerRaw
	}
	job.Status = firstNonEmptyString(result.Status, DocumentAIJobStatusPending)
	if err := s.repo.UpdateAfterSubmit(ctx, job.JobID, providerJobID, providerBatchID, job.Status, providerRaw); err != nil {
		return nil, fmt.Errorf("update document ai job after submit: %w", err)
	}
	s.logDocumentAIInfo(ctx, "submit_success", job, account,
		zap.String("provider_request_id", result.ProviderRequestID),
		zap.String("status", job.Status),
		zap.String("target_model", targetModelID),
	)
	return job, nil
}

func (s *DocumentAIService) ParseDirect(ctx context.Context, input DocumentAIParseDirectInput) (*DocumentAIJob, error) {
	if err := s.RequireEnabled(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil || s.accountRepo == nil {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai service is not ready")
	}
	normalizedInput, err := s.normalizeDirectInput(input)
	if err != nil {
		return nil, err
	}
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, normalizedInput.GroupID, PlatformBaiduDocumentAI)
	if err != nil {
		return nil, err
	}
	account, targetModelID, err := documentAISelectAccountForDisplayModel(accounts, normalizedInput.Model, DocumentAIJobModeDirect)
	if err != nil {
		return nil, err
	}

	job := newDocumentAIJob(normalizedInput.APIKey, normalizedInput.GroupID, account, DocumentAIJobModeDirect, normalizedInput.Model, normalizedInput.SourceType, normalizedInput.FileName, normalizedInput.ContentType, normalizedInput.FileSize, normalizedInput.FileHash)
	if err := s.repo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("create document ai direct job: %w", err)
	}

	s.logDocumentAIInfo(ctx, "direct_start", job, account,
		zap.String("source_type", normalizedInput.SourceType),
		zap.String("file_type", normalizedInput.FileType),
		zap.String("target_model", targetModelID),
	)
	client := newBaiduDocumentAIClient(s.httpUpstream, s.tlsFingerprintProfileService)
	routeInput := normalizedInput
	routeInput.Model = targetModelID
	result, err := client.parseDirect(ctx, account, routeInput)
	if err != nil {
		s.markJobFailed(ctx, "direct_failed", job, documentAIProviderResultJSONFromError(err), err)
		return nil, err
	}
	normalizedJSON, marshalErr := json.Marshal(result.Envelope)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal document ai direct result: %w", marshalErr)
	}
	providerRaw := stringPtr(result.ProviderRawJSON)
	normalizedRaw := stringPtr(string(normalizedJSON))
	if err := s.repo.MarkSucceeded(ctx, job.JobID, providerRaw, normalizedRaw); err != nil {
		return nil, fmt.Errorf("mark document ai direct job succeeded: %w", err)
	}
	now := time.Now().UTC()
	job.Status = DocumentAIJobStatusSucceeded
	job.ProviderResultJSON = providerRaw
	job.NormalizedResultJSON = normalizedRaw
	job.CompletedAt = &now
	s.logDocumentAIInfo(ctx, "direct_success", job, account,
		zap.String("provider_request_id", result.ProviderRequestID),
		zap.String("status", job.Status),
		zap.String("target_model", targetModelID),
	)
	return job, nil
}

func (s *DocumentAIService) runPoller() {
	if s == nil || s.repo == nil || s.accountRepo == nil || s.httpUpstream == nil {
		return
	}
	if !atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&s.running, 0)

	ctx, cancel := context.WithTimeout(context.Background(), documentAIJobPollTaskTTL)
	defer cancel()

	jobs, err := s.repo.ListPollable(ctx, documentAIJobPollBatchSize)
	if err != nil {
		s.logDocumentAIError(ctx, "poll_failed", nil, nil,
			zap.Error(err),
			zap.String("stage", "list_pollable"),
		)
		return
	}
	now := time.Now().UTC()
	for i := range jobs {
		job := jobs[i]
		if !shouldPollDocumentAIJob(job, now) {
			continue
		}
		s.pollJob(ctx, &job)
	}
}

func (s *DocumentAIService) pollJob(ctx context.Context, job *DocumentAIJob) {
	if job == nil || strings.TrimSpace(job.JobID) == "" || strings.TrimSpace(job.Status) == "" {
		return
	}
	if job.Mode != DocumentAIJobModeAsync {
		return
	}
	if job.ProviderJobID == nil || strings.TrimSpace(*job.ProviderJobID) == "" {
		s.markJobFailed(ctx, "poll_failed", job, nil, infraerrors.New(502, "document_ai_provider_error", "document ai provider_job_id is missing"))
		return
	}

	account, err := s.resolvePollingAccount(ctx, job)
	if err != nil {
		s.markJobFailed(ctx, "poll_failed", job, nil, err)
		return
	}
	client := newBaiduDocumentAIClient(s.httpUpstream, s.tlsFingerprintProfileService)
	s.logDocumentAIInfo(ctx, "poll_start", job, account)
	result, pollErr := client.getAsyncJobStatus(ctx, account, strings.TrimSpace(*job.ProviderJobID))
	if pollErr != nil {
		providerRaw := firstNonNilStringPointer(resultRawJSONPtr(result), documentAIProviderResultJSONFromError(pollErr))
		if result != nil && result.ProviderRawJSON != "" && infraerrors.Reason(pollErr) == "document_ai_job_failed" {
			s.markJobFailed(ctx, "poll_failed", job, providerRaw, pollErr)
			return
		}
		if reason := infraerrors.Reason(pollErr); reason == "document_ai_auth_error" || reason == "document_ai_invalid_request" {
			s.markJobFailed(ctx, "poll_failed", job, providerRaw, pollErr)
			return
		}
		_ = s.repo.TouchLastPolledAt(ctx, job.JobID)
		s.logDocumentAIWarn(ctx, "poll_failed", job, account,
			zap.Error(pollErr),
			zap.String("reason", infraerrors.Reason(pollErr)),
			zap.String("provider_request_id", documentAIProviderRequestIDFromError(pollErr)),
		)
		return
	}
	switch result.Status {
	case DocumentAIJobStatusPending:
		_ = s.repo.TouchLastPolledAt(ctx, job.JobID)
	case DocumentAIJobStatusRunning:
		providerRaw := resultRawJSONPtr(result)
		if err := s.repo.MarkRunning(ctx, job.JobID, providerRaw); err != nil {
			s.logDocumentAIError(ctx, "poll_failed", job, account,
				zap.Error(err),
				zap.String("stage", "mark_running"),
			)
			return
		}
	case DocumentAIJobStatusSucceeded:
		envelope := s.buildAsyncResultEnvelope(ctx, account, job, result)
		normalizedJSON, err := json.Marshal(envelope)
		if err != nil {
			s.logDocumentAIWarn(ctx, "poll_failed", job, account,
				zap.Error(err),
				zap.String("stage", "normalize_result"),
			)
			_ = s.repo.TouchLastPolledAt(ctx, job.JobID)
			return
		}
		if err := s.repo.MarkSucceeded(ctx, job.JobID, resultRawJSONPtr(result), stringPtr(string(normalizedJSON))); err != nil {
			s.logDocumentAIError(ctx, "poll_failed", job, account,
				zap.Error(err),
				zap.String("stage", "mark_succeeded"),
			)
			return
		}
		s.logDocumentAIInfo(ctx, "poll_success", job, account,
			zap.String("provider_request_id", result.ProviderRequestID),
			zap.String("status", result.Status),
		)
	case DocumentAIJobStatusFailed, DocumentAIJobStatusCanceled:
		s.markJobFailed(ctx, "poll_failed", job, resultRawJSONPtr(result), infraerrors.New(502, "document_ai_job_failed", "document ai provider job failed"))
	default:
		_ = s.repo.TouchLastPolledAt(ctx, job.JobID)
	}
}

func (s *DocumentAIService) buildAsyncResultEnvelope(ctx context.Context, account *Account, job *DocumentAIJob, result *baiduDocumentAIAsyncStatusResult) DocumentAIResultEnvelope {
	envelope := DocumentAIResultEnvelope{
		Provider:      DocumentAIProviderBaidu,
		Mode:          DocumentAIJobModeAsync,
		Model:         job.Model,
		Status:        DocumentAIJobStatusSucceeded,
		ProviderJobID: job.ProviderJobID,
		Text:          "",
	}
	client := newBaiduDocumentAIClient(s.httpUpstream, s.tlsFingerprintProfileService)
	if result != nil && strings.TrimSpace(result.MarkdownResultURL) != "" {
		if text, err := client.downloadResultText(ctx, account, result.MarkdownResultURL); err == nil {
			envelope.Text = text
		} else {
			s.logDocumentAIWarn(ctx, "poll_failed", job, account,
				zap.Error(err),
				zap.String("stage", "download_markdown"),
			)
		}
	}
	if result != nil && strings.TrimSpace(result.JSONResultURL) != "" {
		if payload, err := client.downloadResultJSON(ctx, account, result.JSONResultURL); err == nil {
			normalizeDocumentAIEnvelopeFromJSON(&envelope, payload)
		} else {
			s.logDocumentAIWarn(ctx, "poll_failed", job, account,
				zap.Error(err),
				zap.String("stage", "download_json"),
			)
		}
	}
	if envelope.PageCount == 0 && strings.TrimSpace(envelope.Text) != "" {
		envelope.PageCount = 1
	}
	return envelope
}

func (s *DocumentAIService) resolvePollingAccount(ctx context.Context, job *DocumentAIJob) (*Account, error) {
	if job == nil {
		return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "document ai job is missing")
	}
	if job.AccountID != nil && *job.AccountID > 0 {
		account, err := s.accountRepo.GetByID(ctx, *job.AccountID)
		if err == nil && account != nil && account.IsBaiduDocumentAI() && account.GetBaiduDocumentAIAsyncBearerToken() != "" {
			return account, nil
		}
	}
	if job.GroupID != nil && *job.GroupID > 0 {
		return s.selectAccountForGroup(ctx, *job.GroupID, documentAIAccountNeedAsync)
	}
	return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "no available baidu document ai account")
}

func (s *DocumentAIService) normalizeSubmitInput(input DocumentAISubmitJobInput) (DocumentAISubmitJobInput, error) {
	input.Model = strings.TrimSpace(input.Model)
	input.SourceType = trimLower(input.SourceType)
	if input.APIKey == nil || input.APIKey.User == nil {
		return input, infraerrors.Forbidden("document_ai_forbidden", "document ai access requires a valid API key")
	}
	if input.GroupID <= 0 {
		return input, infraerrors.Forbidden("document_ai_forbidden", "This API key is not bound to any Baidu Document AI group")
	}
	if input.Model == "" {
		return input, infraerrors.BadRequest("document_ai_invalid_request", "unsupported document ai model")
	}
	switch input.SourceType {
	case DocumentAISourceTypeFile:
		if len(input.FileBytes) == 0 {
			return input, infraerrors.BadRequest("document_ai_invalid_request", "file is required")
		}
	case DocumentAISourceTypeFileURL:
		if strings.TrimSpace(input.FileURL) == "" {
			return input, infraerrors.BadRequest("document_ai_invalid_request", "file_url is required")
		}
	default:
		return input, infraerrors.BadRequest("document_ai_invalid_request", "unsupported document ai source_type")
	}
	input.FileHash = normalizeDocumentAIFileHash(input.FileHash, input.FileBytes)
	input.FileName = strings.TrimSpace(input.FileName)
	input.ContentType = strings.TrimSpace(input.ContentType)
	if input.FileSize <= 0 && len(input.FileBytes) > 0 {
		input.FileSize = int64(len(input.FileBytes))
	}
	return input, nil
}

func (s *DocumentAIService) normalizeDirectInput(input DocumentAIParseDirectInput) (DocumentAIParseDirectInput, error) {
	input.Model = strings.TrimSpace(input.Model)
	input.SourceType = trimLower(input.SourceType)
	input.FileType = trimLower(input.FileType)
	if input.APIKey == nil || input.APIKey.User == nil {
		return input, infraerrors.Forbidden("document_ai_forbidden", "document ai access requires a valid API key")
	}
	if input.GroupID <= 0 {
		return input, infraerrors.Forbidden("document_ai_forbidden", "This API key is not bound to any Baidu Document AI group")
	}
	if input.Model == "" {
		return input, infraerrors.BadRequest("document_ai_invalid_request", "unsupported direct document ai model")
	}
	if input.FileType != DocumentAIFileTypeImage && input.FileType != DocumentAIFileTypePDF {
		return input, infraerrors.BadRequest("document_ai_invalid_request", "file_type must be image or pdf")
	}
	switch input.SourceType {
	case DocumentAISourceTypeFile:
		if len(input.FileBytes) == 0 {
			return input, infraerrors.BadRequest("document_ai_invalid_request", "file is required")
		}
	case DocumentAISourceTypeFileBase64:
		if strings.TrimSpace(input.FileBase64) == "" {
			return input, infraerrors.BadRequest("document_ai_invalid_request", "file_base64 is required")
		}
		if len(input.FileBytes) == 0 {
			if payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(input.FileBase64)); err == nil {
				input.FileBytes = payload
			}
		}
	default:
		return input, infraerrors.BadRequest("document_ai_invalid_request", "unsupported direct source_type")
	}
	input.FileHash = normalizeDocumentAIFileHash(input.FileHash, input.FileBytes)
	input.FileName = strings.TrimSpace(input.FileName)
	input.ContentType = strings.TrimSpace(input.ContentType)
	if input.FileSize <= 0 && len(input.FileBytes) > 0 {
		input.FileSize = int64(len(input.FileBytes))
	}
	return input, nil
}

type documentAIAccountCapability func(*Account) bool

func documentAIAccountNeedDirect(model string) documentAIAccountCapability {
	return func(account *Account) bool {
		return account != nil &&
			account.IsBaiduDocumentAI() &&
			account.GetBaiduDocumentAIDirectToken() != "" &&
			account.GetBaiduDocumentAIDirectAPIURL(model) != ""
	}
}

func documentAIAccountNeedAsync(account *Account) bool {
	return account != nil && account.IsBaiduDocumentAI() && account.GetBaiduDocumentAIAsyncBearerToken() != ""
}

func (s *DocumentAIService) selectAccountForGroup(ctx context.Context, groupID int64, capability documentAIAccountCapability) (*Account, error) {
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, groupID, PlatformBaiduDocumentAI)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		account := accounts[i]
		if capability(&account) {
			return &account, nil
		}
	}
	return nil, infraerrors.ServiceUnavailable("document_ai_provider_error", "no available baidu document ai account")
}

func shouldPollDocumentAIJob(job DocumentAIJob, now time.Time) bool {
	if job.Status != DocumentAIJobStatusPending && job.Status != DocumentAIJobStatusRunning {
		return false
	}
	interval := documentAIJobDynamicInterval(job.CreatedAt, now)
	if job.LastPolledAt == nil {
		return true
	}
	return now.Sub(*job.LastPolledAt) >= interval
}

func documentAIJobDynamicInterval(createdAt time.Time, now time.Time) time.Duration {
	elapsed := now.Sub(createdAt)
	switch {
	case elapsed < 15*time.Second:
		return 15 * time.Second
	case elapsed < 45*time.Second:
		return 30 * time.Second
	case elapsed < 105*time.Second:
		return 60 * time.Second
	case elapsed < 225*time.Second:
		return 120 * time.Second
	default:
		return 300 * time.Second
	}
}

func newDocumentAIJob(apiKey *APIKey, groupID int64, account *Account, mode, model, sourceType, fileName, contentType string, fileSize int64, fileHash string) *DocumentAIJob {
	job := &DocumentAIJob{
		JobID:      uuid.NewString(),
		UserID:     apiKey.User.ID,
		APIKeyID:   apiKey.ID,
		Mode:       mode,
		Model:      model,
		SourceType: sourceType,
		Status:     DocumentAIJobStatusPending,
	}
	if groupID > 0 {
		job.GroupID = &groupID
	}
	if account != nil {
		job.AccountID = &account.ID
	}
	if trimmed := strings.TrimSpace(fileName); trimmed != "" {
		job.FileName = &trimmed
	}
	if trimmed := strings.TrimSpace(contentType); trimmed != "" {
		job.ContentType = &trimmed
	}
	if fileSize > 0 {
		job.FileSize = &fileSize
	}
	if trimmed := strings.TrimSpace(fileHash); trimmed != "" {
		job.FileHash = &trimmed
	}
	return job
}

func normalizeDocumentAIFileHash(provided string, payload []byte) string {
	provided = strings.TrimSpace(provided)
	if provided != "" {
		return provided
	}
	if len(payload) == 0 {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (s *DocumentAIService) markJobFailed(ctx context.Context, event string, job *DocumentAIJob, providerRaw *string, err error) {
	if s == nil || s.repo == nil || job == nil || strings.TrimSpace(job.JobID) == "" || err == nil {
		return
	}
	statusCode, status := infraerrors.ToHTTP(err)
	errorCode := strings.TrimSpace(firstNonEmptyString(status.Reason, fmt.Sprintf("http_%d", statusCode)))
	errorMessage := strings.TrimSpace(firstNonEmptyString(
		status.Metadata["provider_message"],
		status.Message,
		err.Error(),
	))
	if repoErr := s.repo.MarkFailed(ctx, job.JobID, providerRaw, errorCode, errorMessage); repoErr != nil && !errors.Is(repoErr, sql.ErrNoRows) {
		s.logDocumentAIError(ctx, event, job, nil,
			zap.String("status", DocumentAIJobStatusFailed),
			zap.String("error_code", errorCode),
			zap.String("error_message", errorMessage),
			zap.String("provider_request_id", status.Metadata["provider_request_id"]),
			zap.String("stage", "persist_failure"),
			zap.Error(repoErr),
		)
	}
	s.logDocumentAIError(ctx, event, job, nil,
		zap.String("status", DocumentAIJobStatusFailed),
		zap.String("error_code", errorCode),
		zap.String("error_message", errorMessage),
		zap.String("provider_request_id", status.Metadata["provider_request_id"]),
		zap.Error(err),
	)
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func resultRawJSONPtr(result *baiduDocumentAIAsyncStatusResult) *string {
	if result == nil || strings.TrimSpace(result.ProviderRawJSON) == "" {
		return nil
	}
	return stringPtr(result.ProviderRawJSON)
}

func firstNonNilStringPointer(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return value
		}
	}
	return nil
}

func documentAIProviderRequestIDFromError(err error) string {
	if err == nil {
		return ""
	}
	_, status := infraerrors.ToHTTP(err)
	return strings.TrimSpace(status.Metadata["provider_request_id"])
}

func documentAIRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		return strings.TrimSpace(requestID)
	}
	if clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
		return strings.TrimSpace(clientRequestID)
	}
	return ""
}

func documentAIBackgroundRequestID(ctx context.Context) string {
	return firstNonEmptyString(documentAIRequestIDFromContext(ctx), "system:"+documentAIJobPollerName)
}

func (s *DocumentAIService) logDocumentAIInfo(ctx context.Context, event string, job *DocumentAIJob, account *Account, extraFields ...zap.Field) {
	s.logDocumentAI(ctx, "info", event, job, account, extraFields...)
}

func (s *DocumentAIService) logDocumentAIWarn(ctx context.Context, event string, job *DocumentAIJob, account *Account, extraFields ...zap.Field) {
	s.logDocumentAI(ctx, "warn", event, job, account, extraFields...)
}

func (s *DocumentAIService) logDocumentAIError(ctx context.Context, event string, job *DocumentAIJob, account *Account, extraFields ...zap.Field) {
	s.logDocumentAI(ctx, "error", event, job, account, extraFields...)
}

func (s *DocumentAIService) logDocumentAI(ctx context.Context, level string, event string, job *DocumentAIJob, account *Account, extraFields ...zap.Field) {
	requestID := documentAIRequestIDFromContext(ctx)
	if requestID == "" && strings.HasPrefix(strings.TrimSpace(event), "poll_") {
		requestID = documentAIBackgroundRequestID(ctx)
	}

	accountID := int64(0)
	groupID := int64(0)
	apiKeyID := int64(0)
	jobID := ""
	providerJobID := ""
	model := ""
	mode := ""

	if job != nil {
		jobID = strings.TrimSpace(job.JobID)
		model = strings.TrimSpace(job.Model)
		mode = strings.TrimSpace(job.Mode)
		apiKeyID = job.APIKeyID
		if job.AccountID != nil {
			accountID = *job.AccountID
		}
		if job.GroupID != nil {
			groupID = *job.GroupID
		}
		if job.ProviderJobID != nil {
			providerJobID = strings.TrimSpace(*job.ProviderJobID)
		}
	}
	if account != nil && account.ID > 0 {
		accountID = account.ID
	}

	fields := []zap.Field{
		zap.String("component", "service.document_ai"),
		zap.String("request_id", requestID),
		zap.String("job_id", jobID),
		zap.String("provider_job_id", providerJobID),
		zap.Int64("account_id", accountID),
		zap.Int64("group_id", groupID),
		zap.Int64("api_key_id", apiKeyID),
		zap.String("model", model),
		zap.String("mode", mode),
	}
	fields = append(fields, extraFields...)

	log := logger.FromContext(ctx)
	switch level {
	case "error":
		log.Error(event, fields...)
	case "warn":
		log.Warn(event, fields...)
	default:
		log.Info(event, fields...)
	}
}
