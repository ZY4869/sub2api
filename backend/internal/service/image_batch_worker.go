package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

const imageBatchWorkerInterval = 15 * time.Second

var errImageBatchSettlementPending = errors.New("image batch settlement pending")

func (s *ImageBatchService) Start() {
	if s == nil || s.repo == nil || s.forwarder == nil {
		return
	}
	go s.runWorker()
}

func (s *ImageBatchService) Stop() {
	if s == nil || s.stopCh == nil {
		return
	}
	close(s.stopCh)
}

func (s *ImageBatchService) workerInterval() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.ImageBatch.WorkerIntervalSeconds > 0 {
		return time.Duration(s.cfg.ImageBatch.WorkerIntervalSeconds) * time.Second
	}
	return imageBatchWorkerInterval
}

func (s *ImageBatchService) workerClaimLimit() int {
	if s != nil && s.cfg != nil && s.cfg.ImageBatch.WorkerClaimLimit > 0 {
		return s.cfg.ImageBatch.WorkerClaimLimit
	}
	return 4
}

func (s *ImageBatchService) submitTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.ImageBatch.SubmitTimeoutSeconds > 0 {
		return time.Duration(s.cfg.ImageBatch.SubmitTimeoutSeconds) * time.Second
	}
	return 60 * time.Second
}

func (s *ImageBatchService) pollTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.ImageBatch.PollTimeoutSeconds > 0 {
		return time.Duration(s.cfg.ImageBatch.PollTimeoutSeconds) * time.Second
	}
	return 30 * time.Second
}

func (s *ImageBatchService) downloadTimeout() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.ImageBatch.DownloadTimeoutSeconds > 0 {
		return time.Duration(s.cfg.ImageBatch.DownloadTimeoutSeconds) * time.Second
	}
	return 180 * time.Second
}

func (s *ImageBatchService) settlementRetryLimit() int {
	if s != nil && s.cfg != nil {
		return s.cfg.ImageBatch.SettlementRetryLimit
	}
	return 3
}

func (s *ImageBatchService) outputCleanupAfter() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.ImageBatch.OutputCleanupAfterHours > 0 {
		return time.Duration(s.cfg.ImageBatch.OutputCleanupAfterHours) * time.Hour
	}
	return 0
}

func (s *ImageBatchService) providerTimeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func (s *ImageBatchService) runWorker() {
	ticker := time.NewTicker(s.workerInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.ProcessOnce(context.Background())
		case <-s.stopCh:
			return
		}
	}
}

func (s *ImageBatchService) ProcessOnce(ctx context.Context) {
	s.cleanupExpiredOutputs(ctx)
	for i := 0; i < s.workerClaimLimit(); i++ {
		job, items, err := s.repo.ClaimNextRunnableJob(ctx, time.Now().UTC())
		if err != nil || job == nil {
			return
		}
		recordImageBatchQueueLag(job.CreatedAt)
		if job.Status != ImageBatchJobUploading {
			recordImageBatchWorkerRecovered()
		}
		if err := s.processJob(ctx, job, items); err != nil {
			if errors.Is(err, errImageBatchSettlementPending) {
				continue
			}
			errorID := GenerateSafeRequestID()
			_ = s.repo.FailJob(ctx, job.ID, "批量生图任务处理失败，请稍后重试或联系支持。", errorID)
			_ = s.repo.AddEvent(ctx, job.ID, nil, "worker_failed", string(job.Status), string(ImageBatchJobFailed), "", requestIDFromContext(ctx), map[string]any{"error_id": errorID})
			s.releaseJobHoldBestEffort(ctx, job)
			recordImageBatchFailed()
		}
	}
}

func (s *ImageBatchService) cleanupExpiredOutputs(ctx context.Context) {
	after := s.outputCleanupAfter()
	if after <= 0 || s == nil || s.repo == nil {
		return
	}
	count, err := s.repo.CleanupExpiredOutputs(ctx, time.Now().UTC().Add(-after), 100)
	if err == nil {
		recordImageBatchOutputsCleaned(count)
	}
}

func (s *ImageBatchService) processJob(ctx context.Context, job *ImageBatchJob, items []ImageBatchItem) error {
	apiKey, err := s.apiKeyForJob(ctx, job)
	if err != nil {
		return err
	}
	if job.Status == ImageBatchJobSettling {
		if err := s.settleJobHold(ctx, job, apiKey); err != nil {
			if stopErr := s.handleJobSettlementFailure(ctx, job, err); stopErr != nil {
				return stopErr
			}
			return errImageBatchSettlementPending
		}
		recordImageBatchSettlementLag(job.CreatedAt)
		recordImageBatchCompleted(job.CreatedAt)
		return s.repo.CompleteJob(ctx, job.ID, job.ProviderResultFileName)
	}
	if strings.TrimSpace(job.ProviderBatchName) == "" {
		result, err := s.submitProviderJob(ctx, job, items, apiKey)
		if err != nil {
			return err
		}
		resp, ok := result.(*UpstreamHTTPResult)
		if !ok || resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errors.New("image batch submit upstream failed")
		}
		name := strings.TrimSpace(gjson.GetBytes(resp.Body, "name").String())
		if name == "" {
			return errors.New("image batch submit response missing name")
		}
		name = normalizeImageBatchProviderBatchName(job.Provider, name)
		if err := s.repo.MarkJobSubmitted(ctx, job.ID, name); err != nil {
			return err
		}
		job.ProviderBatchName = name
		job.Status = ImageBatchJobSubmitted
		_ = s.repo.AddEvent(ctx, job.ID, nil, "submitted_upstream", string(ImageBatchJobUploading), string(ImageBatchJobSubmitted), "", requestIDFromContext(ctx), map[string]any{"provider": job.Provider})
	}
	return s.pollAndIndexJob(ctx, job, apiKey)
}

func (s *ImageBatchService) pollAndIndexJob(ctx context.Context, job *ImageBatchJob, apiKey *APIKey) error {
	result, _, err := s.forwardProviderPoll(ctx, job, apiKey)
	if err != nil {
		return err
	}
	resp, ok := result.(*UpstreamHTTPResult)
	if !ok || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("image batch poll upstream failed")
	}
	state := strings.ToLower(strings.TrimSpace(gjson.GetBytes(resp.Body, "state").String()))
	resultFile := extractImageBatchResultFileName(resp.Body)
	switch {
	case strings.Contains(state, "fail"):
		s.releaseJobHoldBestEffort(ctx, job)
		recordImageBatchFailed()
		return s.repo.FailJob(ctx, job.ID, "批量生图上游任务失败，请检查输入后重试。", GenerateSafeRequestID())
	case strings.Contains(state, "cancel"):
		s.releaseJobHoldBestEffort(ctx, job)
		recordImageBatchCancelled()
		_, err := s.repo.TryCancelJob(ctx, job.UserID, job.ID, "cancelled upstream")
		return err
	case strings.Contains(state, "succeed"):
		return s.downloadAndIndex(ctx, job, apiKey, resultFile)
	default:
		return s.repo.MarkJobRunning(ctx, job.ID, resultFile)
	}
}

func (s *ImageBatchService) downloadAndIndex(ctx context.Context, job *ImageBatchJob, apiKey *APIKey, resultFile string) error {
	if strings.TrimSpace(resultFile) == "" {
		return s.repo.MarkJobRunning(ctx, job.ID, "")
	}
	if err := s.repo.MarkJobIndexing(ctx, job.ID, resultFile); err != nil {
		return err
	}
	requestCtx, cancel := s.providerTimeoutContext(ctx, s.downloadTimeout())
	defer cancel()
	result, _, err := s.forwarder.ForwardGoogleFileDownload(requestCtx, GoogleBatchForwardInput{
		GroupID:  job.GroupID,
		APIKeyID: job.APIKeyID,
		APIKey:   apiKey,
		UserID:   job.UserID,
		Method:   http.MethodGet,
		Path:     "/download/v1beta/" + strings.TrimPrefix(resultFile, "/") + ":download",
		RawQuery: "alt=media",
	})
	if err != nil {
		recordImageBatchDownloadFailure()
		return err
	}
	stream, ok := result.(*UpstreamHTTPStreamResult)
	if !ok || stream.StatusCode < 200 || stream.StatusCode >= 300 || stream.Body == nil {
		recordImageBatchDownloadFailure()
		return errors.New("image batch result download failed")
	}
	defer func() { _ = stream.Body.Close() }()
	body, err := readImageBatchProviderResult(stream.Body, s.imageBatchDownloadLimitForJob(ctx, job))
	if err != nil {
		recordImageBatchDownloadFailure()
		return err
	}
	if err := s.indexResultPayload(ctx, job, body); err != nil {
		return err
	}
	if err := s.repo.MarkJobSettling(ctx, job.ID, "", ""); err != nil {
		return err
	}
	if err := s.settleJobHold(ctx, job, apiKey); err != nil {
		if stopErr := s.handleJobSettlementFailure(ctx, job, err); stopErr != nil {
			return stopErr
		}
		return errImageBatchSettlementPending
	}
	recordImageBatchSettlementLag(job.CreatedAt)
	recordImageBatchCompleted(job.CreatedAt)
	return s.repo.CompleteJob(ctx, job.ID, resultFile)
}

func (s *ImageBatchService) submitProviderJob(ctx context.Context, job *ImageBatchJob, items []ImageBatchItem, apiKey *APIKey) (GoogleBatchUpstreamResult, error) {
	switch normalizeImageBatchProvider(job.Provider) {
	case ImageBatchProviderGeminiAPI:
		body, err := buildGeminiImageBatchPayload(job, items)
		if err != nil {
			return nil, err
		}
		requestCtx, cancel := s.providerTimeoutContext(ctx, s.submitTimeout())
		defer cancel()
		result, _, err := s.forwarder.ForwardGoogleBatches(requestCtx, GoogleBatchForwardInput{
			GroupID:       job.GroupID,
			APIKeyID:      job.APIKeyID,
			APIKey:        apiKey,
			UserID:        job.UserID,
			BillingType:   BillingTypeBalance,
			Method:        http.MethodPost,
			Path:          "/v1beta/models/" + strings.TrimSpace(job.TargetModelID) + ":batchGenerateContent",
			Headers:       http.Header{"Content-Type": []string{"application/json"}},
			Body:          body,
			ContentLength: int64(len(body)),
		})
		return result, err
	case ImageBatchProviderVertex:
		body, err := buildVertexImageBatchPayload(job, items)
		if err != nil {
			return nil, err
		}
		requestCtx, cancel := s.providerTimeoutContext(ctx, s.submitTimeout())
		defer cancel()
		result, _, err := s.forwarder.ForwardSimplifiedVertexBatchPredictionJobs(requestCtx, GoogleBatchForwardInput{
			GroupID:       job.GroupID,
			APIKeyID:      job.APIKeyID,
			APIKey:        apiKey,
			UserID:        job.UserID,
			BillingType:   BillingTypeBalance,
			Method:        http.MethodPost,
			Path:          "/v1/vertex/batchPredictionJobs",
			Headers:       http.Header{"Content-Type": []string{"application/json"}},
			Body:          body,
			ContentLength: int64(len(body)),
		})
		return result, err
	default:
		return nil, ErrImageBatchInvalidProvider
	}
}

func (s *ImageBatchService) forwardProviderPoll(ctx context.Context, job *ImageBatchJob, apiKey *APIKey) (GoogleBatchUpstreamResult, *Account, error) {
	switch normalizeImageBatchProvider(job.Provider) {
	case ImageBatchProviderGeminiAPI:
		requestCtx, cancel := s.providerTimeoutContext(ctx, s.pollTimeout())
		defer cancel()
		return s.forwarder.ForwardGoogleBatches(requestCtx, GoogleBatchForwardInput{
			GroupID:  job.GroupID,
			APIKeyID: job.APIKeyID,
			APIKey:   apiKey,
			UserID:   job.UserID,
			Method:   http.MethodGet,
			Path:     "/v1beta/" + strings.TrimPrefix(strings.TrimSpace(job.ProviderBatchName), "/"),
		})
	case ImageBatchProviderVertex:
		requestCtx, cancel := s.providerTimeoutContext(ctx, s.pollTimeout())
		defer cancel()
		return s.forwarder.ForwardSimplifiedVertexBatchPredictionJobs(requestCtx, GoogleBatchForwardInput{
			GroupID:  job.GroupID,
			APIKeyID: job.APIKeyID,
			APIKey:   apiKey,
			UserID:   job.UserID,
			Method:   http.MethodGet,
			Path:     "/v1/vertex/batchPredictionJobs/" + strings.TrimPrefix(strings.TrimSpace(job.ProviderBatchName), "batchPredictionJobs/"),
		})
	default:
		return nil, nil, ErrImageBatchInvalidProvider
	}
}

func normalizeImageBatchProviderBatchName(provider string, name string) string {
	name = strings.Trim(strings.TrimSpace(name), "/")
	if normalizeImageBatchProvider(provider) == ImageBatchProviderVertex {
		jobName := extractVertexBatchJobName(name)
		if jobName != "" {
			return "batchPredictionJobs/" + jobName
		}
	}
	return name
}

func readImageBatchProviderResult(reader io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(reader)
	}
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, ErrImageBatchDownloadTooLarge
	}
	return body, nil
}

func (s *ImageBatchService) handleJobSettlementFailure(ctx context.Context, job *ImageBatchJob, err error) error {
	recordImageBatchSettlementFailure()
	retryCount := 0
	if s != nil && s.repo != nil {
		next, repoErr := s.repo.IncrementJobSettlementRetry(ctx, job.ID)
		if repoErr != nil {
			return repoErr
		}
		retryCount = next
	}
	limit := s.settlementRetryLimit()
	errorID := GenerateSafeRequestID()
	if limit <= 0 || retryCount >= limit {
		message := "批量生图已完成，但结算多次失败，系统已释放冻结额度，请联系支持处理。"
		_ = s.repo.AddEvent(ctx, job.ID, nil, "settlement_failed", string(ImageBatchJobSettling), string(ImageBatchJobFailed), message, requestIDFromContext(ctx), map[string]any{"error_id": errorID, "retry_count": retryCount})
		s.releaseJobHoldBestEffort(ctx, job)
		recordImageBatchFailed()
		return s.repo.FailJob(ctx, job.ID, message, errorID)
	}
	message := "批量生图已完成，但结算暂时失败，系统将自动重试。"
	_ = s.repo.MarkJobSettling(ctx, job.ID, message, errorID)
	_ = s.repo.AddEvent(ctx, job.ID, nil, "settlement_retry", string(ImageBatchJobSettling), string(ImageBatchJobSettling), message, requestIDFromContext(ctx), map[string]any{"error_id": errorID, "retry_count": retryCount, "retry_limit": limit})
	return nil
}

func (s *ImageBatchService) settleJobHold(ctx context.Context, job *ImageBatchJob, apiKey *APIKey) error {
	if apiKey == nil || apiKey.BillingHold == nil || apiKey.BillingHold.Status != BillingHoldStatusHeld {
		return nil
	}
	if apiKey.BillingHold.RequestID == "" {
		apiKey.BillingHold.RequestID = job.HoldRequestID
	}
	if apiKey.BillingHold.RequestID == "" {
		return nil
	}
	if apiKey.User == nil {
		apiKey.User = &User{ID: job.UserID}
	}
	// Google batch archive settlement runs through usage billing when usage can be read.
	// Settling the minimum hold to zero here makes completed zero-usage or deferred
	// settlement jobs idempotently release the reserve instead of leaving money frozen.
	repo := s.apiKeyService.billingHoldRepository()
	if repo == nil {
		return ErrBillingServiceUnavailable
	}
	settled, err := repo.Settle(ctx, apiKey.BillingHold.RequestID, job.APIKeyID, 0)
	if err != nil && !errors.Is(err, ErrBillingHoldNotFound) && !errors.Is(err, ErrBillingHoldAlreadyFinished) {
		return err
	}
	if settled != nil {
		apiKey.BillingHold = settled
	} else {
		apiKey.BillingHold.Status = BillingHoldStatusSettled
	}
	s.apiKeyService.InvalidateAuthCacheByKey(ctx, apiKey.Key)
	s.apiKeyService.invalidateBillingBalanceCache(ctx, job.UserID)
	return nil
}

func (s *ImageBatchService) apiKeyForJob(ctx context.Context, job *ImageBatchJob) (*APIKey, error) {
	if s.apiKeyService == nil || job == nil {
		return nil, ErrImageBatchUnavailable
	}
	apiKey, err := s.apiKeyService.GetByID(ctx, job.APIKeyID)
	if err != nil {
		return nil, err
	}
	apiKey.BillingHold = &BillingHold{
		RequestID: job.HoldRequestID,
		APIKeyID:  job.APIKeyID,
		UserID:    job.UserID,
		Status:    BillingHoldStatusHeld,
		Currency:  ModelPricingCurrencyUSD,
	}
	return apiKey, nil
}

func (s *ImageBatchService) releaseJobHoldBestEffort(ctx context.Context, job *ImageBatchJob) {
	apiKey, err := s.apiKeyForJob(ctx, job)
	if err == nil {
		s.apiKeyService.ReleaseRequestBillingHold(ctx, apiKey)
	}
}
