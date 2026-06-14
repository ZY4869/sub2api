package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const accountImportGroupBindingBatchSize = 500

type AccountImportGroupBindingSectionRequest struct {
	Platform string  `json:"platform"`
	Type     string  `json:"type"`
	GroupIDs []int64 `json:"group_ids"`
}

type AccountImportGroupBindingRequest struct {
	Sections []AccountImportGroupBindingSectionRequest `json:"sections"`
}

type AccountImportGroupBindingResult struct {
	Success    int      `json:"success"`
	Failed     int      `json:"failed"`
	BoundCount int      `json:"bound_count"`
	Skipped    int      `json:"skipped"`
	Errors     []string `json:"errors,omitempty"`
}

func (h *AccountHandler) CreateImportJob(c *gin.Context) {
	var req DataImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := validateDataHeader(req.Data); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := validateDataImportAccountDefaults(req.AccountDefaults); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	executeAdminIdempotentJSON(c, "admin.accounts.import_data_job", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		job, err := defaultAccountImportJobs.create(len(req.Data.Proxies) + len(req.Data.Accounts))
		if err != nil {
			return nil, err
		}
		h.startImportJob(job, req)
		slog.Info("admin_account_import_job_created",
			"job_id", job.jobID,
			"proxy_count", len(req.Data.Proxies),
			"account_count", len(req.Data.Accounts),
		)
		return CreateAccountImportJobResponse{JobID: job.jobID}, nil
	})
}

func (h *AccountHandler) GetImportJob(c *gin.Context) {
	job, ok := defaultAccountImportJobs.get(strings.TrimSpace(c.Param("job_id")))
	if !ok {
		response.NotFound(c, "Import job not found")
		return
	}
	response.Success(c, job.snapshot())
}

func (h *AccountHandler) CancelImportJob(c *gin.Context) {
	job, ok := defaultAccountImportJobs.get(strings.TrimSpace(c.Param("job_id")))
	if !ok {
		response.NotFound(c, "Import job not found")
		return
	}

	job.mu.Lock()
	if job.status == accountImportJobStatusSucceeded ||
		job.status == accountImportJobStatusPartialFailed ||
		job.status == accountImportJobStatusFailed ||
		job.status == accountImportJobStatusCancelled {
		snapshot := job.snapshotLocked()
		job.mu.Unlock()
		response.Success(c, snapshot)
		return
	}
	job.cancelRequested = true
	job.updatedAt = time.Now().UTC()
	if job.cancel != nil {
		job.cancel()
	}
	snapshot := job.snapshotLocked()
	job.mu.Unlock()

	slog.Info("admin_account_import_job_cancel_requested", "job_id", job.jobID)
	response.Success(c, snapshot)
}

func (h *AccountHandler) BindImportJobGroups(c *gin.Context) {
	job, ok := defaultAccountImportJobs.get(strings.TrimSpace(c.Param("job_id")))
	if !ok {
		response.NotFound(c, "Import job not found")
		return
	}
	var req AccountImportGroupBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	jobSnapshot := job.snapshot()
	if jobSnapshot.Status != accountImportJobStatusSucceeded && jobSnapshot.Status != accountImportJobStatusPartialFailed {
		response.BadRequest(c, "Import job is not completed")
		return
	}

	result, err := h.bindImportJobGroups(c.Request.Context(), jobSnapshot.CreatedAccountsSummary, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *AccountHandler) startImportJob(job *accountImportJob, req DataImportRequest) {
	jobCtx, cancel := context.WithCancel(context.Background())
	job.mu.Lock()
	job.cancel = cancel
	job.updatedAt = time.Now().UTC()
	job.mu.Unlock()

	go func() {
		defer cancel()
		now := time.Now().UTC()
		job.mu.Lock()
		job.status = accountImportJobStatusRunning
		job.startedAt = &now
		job.updatedAt = now
		job.mu.Unlock()

		result, err := h.importDataWithProgress(jobCtx, req, func(processed int, result DataImportResult) bool {
			job.mu.Lock()
			defer job.mu.Unlock()
			job.progress.Processed = processed
			job.result = cloneDataImportResult(result)
			job.updatedAt = time.Now().UTC()
			return !job.cancelRequested
		})

		finishedAt := time.Now().UTC()
		job.mu.Lock()
		job.result = cloneDataImportResult(result)
		job.finishedAt = &finishedAt
		job.updatedAt = finishedAt
		switch {
		case errors.Is(err, context.Canceled) || job.cancelRequested:
			job.status = accountImportJobStatusCancelled
			job.errorMessage = "cancelled"
		case err != nil:
			job.status = accountImportJobStatusFailed
			job.errorMessage = err.Error()
		case result.AccountFailed > 0 || result.ProxyFailed > 0:
			job.status = accountImportJobStatusPartialFailed
			job.progress.Processed = job.progress.Total
		default:
			job.status = accountImportJobStatusSucceeded
			job.progress.Processed = job.progress.Total
		}
		snapshot := job.snapshotLocked()
		job.mu.Unlock()

		slog.Info("admin_account_import_job_finished",
			"job_id", snapshot.JobID,
			"status", snapshot.Status,
			"account_created", snapshot.Result.AccountCreated,
			"account_failed", snapshot.Result.AccountFailed,
			"proxy_created", snapshot.Result.ProxyCreated,
			"proxy_failed", snapshot.Result.ProxyFailed,
		)
	}()
}

func (h *AccountHandler) bindImportJobGroups(ctx context.Context, accounts []DataImportCreatedAccount, req AccountImportGroupBindingRequest) (AccountImportGroupBindingResult, error) {
	if len(req.Sections) == 0 {
		return AccountImportGroupBindingResult{}, fmt.Errorf("sections is required")
	}
	accountsByKey := make(map[string][]int64)
	for _, account := range accounts {
		key := importJobGroupBindingKey(account.Platform, account.Type)
		accountsByKey[key] = append(accountsByKey[key], account.AccountID)
	}

	result := AccountImportGroupBindingResult{}
	for _, section := range req.Sections {
		groupIDs := normalizeImportJobGroupIDs(section.GroupIDs)
		if len(groupIDs) == 0 {
			continue
		}
		accountIDs := accountsByKey[importJobGroupBindingKey(section.Platform, section.Type)]
		if len(accountIDs) == 0 {
			result.Skipped++
			continue
		}
		for start := 0; start < len(accountIDs); start += accountImportGroupBindingBatchSize {
			end := start + accountImportGroupBindingBatchSize
			if end > len(accountIDs) {
				end = len(accountIDs)
			}
			batch := append([]int64(nil), accountIDs[start:end]...)
			batchGroupIDs := append([]int64(nil), groupIDs...)
			updateResult, err := h.adminService.BulkUpdateAccounts(ctx, &service.BulkUpdateAccountsInput{
				AccountIDs: batch,
				GroupIDs:   &batchGroupIDs,
			})
			if err != nil {
				result.Failed += len(batch)
				result.Errors = append(result.Errors, err.Error())
				continue
			}
			result.Success += updateResult.Success
			result.Failed += updateResult.Failed
			result.BoundCount += updateResult.Success
			for _, item := range updateResult.Results {
				if item.Success || item.Error == "" {
					continue
				}
				result.Errors = append(result.Errors, item.Error)
			}
		}
	}

	slog.Info("admin_account_import_job_group_binding_completed",
		"success", result.Success,
		"failed", result.Failed,
		"skipped", result.Skipped,
	)
	return result, nil
}

func cloneDataImportResult(result DataImportResult) DataImportResult {
	result.CreatedAccounts = append([]DataImportCreatedAccount(nil), result.CreatedAccounts...)
	result.Errors = append([]DataImportError(nil), result.Errors...)
	return result
}

func importJobGroupBindingKey(platform, accountType string) string {
	return strings.TrimSpace(platform) + ":" + strings.TrimSpace(accountType)
}

func normalizeImportJobGroupIDs(groupIDs []int64) []int64 {
	seen := make(map[int64]struct{}, len(groupIDs))
	out := make([]int64, 0, len(groupIDs))
	for _, id := range groupIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
