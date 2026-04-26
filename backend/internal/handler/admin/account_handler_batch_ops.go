package admin

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	batchAccountTestModelInputModeAuto    = "auto"
	batchAccountTestModelInputModeCatalog = service.ScheduledTestModelInputModeCatalog
	batchAccountTestModelInputModeManual  = service.ScheduledTestModelInputModeManual
)

type batchAccountTestModelsRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
}

type batchAccountTestRequest struct {
	AccountIDs     []int64 `json:"account_ids" binding:"required,min=1"`
	ModelID        string  `json:"model_id"`
	Model          string  `json:"model"`
	ModelInputMode string  `json:"model_input_mode"`
	ManualModelID  string  `json:"manual_model_id"`
	SourceProtocol string  `json:"source_protocol"`
	TargetProvider string  `json:"target_provider"`
	TargetModelID  string  `json:"target_model_id"`
	Prompt         string  `json:"prompt"`
	TestMode       string  `json:"test_mode"`
}

type batchAccountTestResult struct {
	AccountID               int64  `json:"account_id"`
	AccountName             string `json:"account_name,omitempty"`
	Platform                string `json:"platform,omitempty"`
	Status                  string `json:"status"`
	ErrorMessage            string `json:"error_message,omitempty"`
	ResponseText            string `json:"response_text,omitempty"`
	LatencyMs               int64  `json:"latency_ms,omitempty"`
	ResolvedModelID         string `json:"resolved_model_id,omitempty"`
	ResolvedPlatform        string `json:"resolved_platform,omitempty"`
	ResolvedSourceProtocol  string `json:"resolved_source_protocol,omitempty"`
	BlacklistAdviceDecision string `json:"blacklist_advice_decision,omitempty"`
	CurrentLifecycleState   string `json:"current_lifecycle_state,omitempty"`
}

type batchAccountTestResponse struct {
	Results []batchAccountTestResult `json:"results"`
}

func normalizeBatchAccountTestModelInputMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case batchAccountTestModelInputModeCatalog:
		return batchAccountTestModelInputModeCatalog
	case batchAccountTestModelInputModeManual:
		return batchAccountTestModelInputModeManual
	default:
		return batchAccountTestModelInputModeAuto
	}
}

func normalizeBatchTestSourceProtocol(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case service.PlatformOpenAI, service.PlatformAnthropic, service.PlatformGemini:
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return ""
	}
}

func uniqueAccountIDsPreserveOrder(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

func (h *AccountHandler) GetBatchTestModels(c *gin.Context) {
	var req batchAccountTestModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestKey(c, "admin.account.invalid_request", "Invalid request: %s", err.Error())
		return
	}

	accountIDs := uniqueAccountIDsPreserveOrder(req.AccountIDs)
	if len(accountIDs) == 0 {
		response.BadRequestKey(c, "admin.account.account_ids_required", "account_ids is required")
		return
	}

	accounts, err := h.adminService.GetAccountsByIDs(c.Request.Context(), accountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	modelGroups := make([][]service.AvailableTestModel, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		modelGroups = append(modelGroups, service.BuildAvailableTestModels(c.Request.Context(), account, h.modelRegistryService))
	}

	response.Success(c, service.IntersectAvailableTestModels(modelGroups...))
}

func (h *AccountHandler) BatchTest(c *gin.Context) {
	if h.accountTestService == nil {
		response.ErrorKey(c, 500, "admin.account.test_service_missing", "Account test service is not configured")
		return
	}

	var adminUserID *int64
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		v := subject.UserID
		adminUserID = &v
	}
	testRunID := uuid.NewString()
	service.AttachAccountTestOpsContext(c, testRunID, "batch_test", adminUserID)

	var req batchAccountTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestKey(c, "admin.account.invalid_request", "Invalid request: %s", err.Error())
		return
	}

	accountIDs := uniqueAccountIDsPreserveOrder(req.AccountIDs)
	if len(accountIDs) == 0 {
		response.BadRequestKey(c, "admin.account.account_ids_required", "account_ids is required")
		return
	}

	accounts, err := h.adminService.GetAccountsByIDs(c.Request.Context(), accountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	accountByID := make(map[int64]*service.Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			accountByID[account.ID] = account
		}
	}

	modelInputMode := normalizeBatchAccountTestModelInputMode(req.ModelInputMode)
	requestedModelID := resolveRequestedBatchModelID(req, modelInputMode)
	if modelInputMode != batchAccountTestModelInputModeAuto && requestedModelID == "" {
		response.BadRequestKey(c, "admin.account.model_required_for_manual_mode", "model_id is required when model_input_mode is not auto")
		return
	}
	testMode := strings.TrimSpace(req.TestMode)
	if testMode == "" {
		testMode = string(service.AccountTestModeHealthCheck)
	}
	slog.Info(
		"admin_account_batch_test_start",
		"account_count", len(accountIDs),
		"test_mode", testMode,
		"model_input_mode", modelInputMode,
	)

	results := make([]batchAccountTestResult, len(accountIDs))
	startedAt := time.Now()
	g, gctx := errgroup.WithContext(c.Request.Context())
	g.SetLimit(4)
	var mu sync.Mutex

	for idx, accountID := range accountIDs {
		idx := idx
		accountID := accountID
		g.Go(func() error {
			result := batchAccountTestResult{
				AccountID: accountID,
				Status:    "failed",
			}
			account := accountByID[accountID]
			if account == nil {
				result.ErrorMessage = "account not found"
			} else {
				result.AccountName = account.Name
				result.Platform = account.Platform
				input, resolveErr := h.resolveBatchAccountTestExecutionInput(gctx, account, req)
				if resolveErr != nil {
					result.ErrorMessage = resolveErr.Error()
				} else {
					detail, runErr := h.accountTestService.RunTestBackgroundDetailed(gctx, input)
					if detail != nil {
						result.Status = detail.Status
						result.ErrorMessage = detail.ErrorMessage
						result.ResponseText = detail.ResponseText
						result.LatencyMs = detail.LatencyMs
						result.ResolvedModelID = detail.ResolvedModelID
						result.ResolvedPlatform = detail.ResolvedPlatform
						result.ResolvedSourceProtocol = detail.ResolvedSourceProtocol
						result.BlacklistAdviceDecision = detail.BlacklistAdviceDecision
						result.CurrentLifecycleState = detail.CurrentLifecycleState
					}
					if runErr != nil && result.ErrorMessage == "" {
						result.ErrorMessage = runErr.Error()
					}
					if result.ResolvedModelID == "" {
						result.ResolvedModelID = input.ModelID
					}
					if result.ResolvedSourceProtocol == "" {
						result.ResolvedSourceProtocol = input.SourceProtocol
					}
					if result.CurrentLifecycleState == "" {
						result.CurrentLifecycleState = account.LifecycleState
					}
				}
			}

			mu.Lock()
			results[idx] = result
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()

	successCount := 0
	autoBlacklistedCount := 0
	for _, item := range results {
		if item.Status == "success" {
			successCount++
		}
		if item.BlacklistAdviceDecision == string(service.BlacklistAdviceAutoBlacklisted) || item.CurrentLifecycleState == service.AccountLifecycleBlacklisted {
			autoBlacklistedCount++
		}
	}
	slog.Info(
		"admin_account_batch_test_complete",
		"account_count", len(results),
		"success_count", successCount,
		"failed_count", len(results)-successCount,
		"auto_blacklisted_count", autoBlacklistedCount,
		"test_mode", testMode,
		"model_input_mode", modelInputMode,
	)

	duration := time.Since(startedAt)
	failedCount := len(results) - successCount
	status := "success"
	statusCode := 200
	if failedCount > 0 {
		status = "error"
		statusCode = 500
	}

	routePath := strings.TrimSpace(c.FullPath())
	if routePath == "" && c.Request != nil {
		routePath = strings.TrimSpace(c.Request.URL.Path)
	}

	promptPreview := redactPromptPreview(req.Prompt, 120)
	normalizedRequest := map[string]any{
		"test_run_id":        testRunID,
		"account_count":      len(accountIDs),
		"account_id_sample":  limitInt64Slice(accountIDs, 5),
		"requested_model_id": requestedModelID,
		"model_input_mode":   modelInputMode,
		"test_mode":          testMode,
		"source_protocol":    normalizeBatchTestSourceProtocol(req.SourceProtocol),
		"target_provider":    strings.TrimSpace(req.TargetProvider),
		"target_model_id":    strings.TrimSpace(req.TargetModelID),
		"prompt_len":         len(strings.TrimSpace(req.Prompt)),
		"prompt_preview":     promptPreview,
	}

	gatewayResponse := map[string]any{
		"success":               status == "success",
		"success_count":         successCount,
		"failed_count":          failedCount,
		"auto_blacklisted_count": autoBlacklistedCount,
	}

	recordProbeActionTrace(
		c,
		h.opsService,
		testRunID,
		"",
		"batch_test",
		adminUserID,
		nil,
		"",
		normalizeBatchTestSourceProtocol(req.SourceProtocol),
		"",
		routePath,
		requestedModelID,
		normalizedRequest,
		gatewayResponse,
		status,
		statusCode,
		duration,
	)

	// Best-effort: also write an upstream summary trace so ops can clearly distinguish
	// "batch action" vs "upstream execution" when filtering by probe_action.
	var failureSamples []string
	for _, item := range results {
		if item.Status == "success" {
			continue
		}
		if strings.TrimSpace(item.ErrorMessage) == "" {
			continue
		}
		failureSamples = append(failureSamples, logredact.RedactText(item.ErrorMessage, "sso_token"))
		if len(failureSamples) >= 5 {
			break
		}
	}
	upstreamGatewayResponse := map[string]any{
		"success":        status == "success",
		"success_count":  successCount,
		"failed_count":   failedCount,
		"failure_sample": failureSamples,
	}
	recordProbeActionTrace(
		c,
		h.opsService,
		uuid.NewString(),
		testRunID,
		"batch_test_upstream",
		adminUserID,
		nil,
		"",
		normalizeBatchTestSourceProtocol(req.SourceProtocol),
		"",
		routePath,
		requestedModelID,
		normalizedRequest,
		upstreamGatewayResponse,
		status,
		statusCode,
		duration,
	)

	response.Success(c, batchAccountTestResponse{Results: results})
}

func limitInt64Slice(items []int64, limit int) []int64 {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	cloned := make([]int64, limit)
	copy(cloned, items[:limit])
	return cloned
}

func (h *AccountHandler) resolveBatchAccountTestExecutionInput(
	ctx context.Context,
	account *service.Account,
	req batchAccountTestRequest,
) (service.ScheduledTestExecutionInput, error) {
	input := service.ScheduledTestExecutionInput{
		AccountID:      account.ID,
		Prompt:         strings.TrimSpace(req.Prompt),
		TestMode:       strings.TrimSpace(req.TestMode),
		TargetProvider: strings.TrimSpace(req.TargetProvider),
		TargetModelID:  strings.TrimSpace(req.TargetModelID),
	}
	if input.TestMode == "" {
		input.TestMode = string(service.AccountTestModeHealthCheck)
	}

	modelInputMode := normalizeBatchAccountTestModelInputMode(req.ModelInputMode)
	switch modelInputMode {
	case batchAccountTestModelInputModeAuto:
		if input.TargetProvider == "" {
			input.TargetProvider = service.GetAccountGatewayTestProvider(account)
		}
		if input.TargetModelID == "" {
			input.TargetModelID = service.GetAccountGatewayTestModelID(account)
		}
		if !service.IsProtocolGatewayAccount(account) {
			models := service.BuildAvailableTestModels(ctx, account, h.modelRegistryService)
			if len(models) > 0 {
				input.ModelID = strings.TrimSpace(models[0].ID)
				input.SourceProtocol = normalizeBatchTestSourceProtocol(models[0].SourceProtocol)
			}
		}
		if input.SourceProtocol == "" && service.IsProtocolGatewayAccount(account) {
			acceptedProtocols := service.GetAccountGatewayAcceptedProtocols(account)
			if len(acceptedProtocols) == 1 {
				input.SourceProtocol = acceptedProtocols[0]
			}
		}
	default:
		input.ModelID = resolveRequestedBatchModelID(req, modelInputMode)
		input.SourceProtocol = normalizeBatchTestSourceProtocol(req.SourceProtocol)
	}

	return input, nil
}

func resolveRequestedBatchModelID(req batchAccountTestRequest, modelInputMode string) string {
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = strings.TrimSpace(req.Model)
	}
	if modelInputMode == batchAccountTestModelInputModeManual {
		modelID = strings.TrimSpace(req.ManualModelID)
	}
	return modelID
}
