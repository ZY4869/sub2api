package admin

import (
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type BlacklistRetestRequest struct {
	AccountIDs     []int64 `json:"account_ids" binding:"required,min=1"`
	ModelID        string  `json:"model_id"`
	ModelInputMode string  `json:"model_input_mode"`
	ManualModelID  string  `json:"manual_model_id"`
	SourceProtocol string  `json:"source_protocol"`
}

type BlacklistRetestModelsRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
}

type BlacklistBatchDeleteRequest struct {
	IDs       []int64 `json:"ids"`
	DeleteAll bool    `json:"delete_all"`
}

type BlacklistRetestAccountResult struct {
	AccountID    int64  `json:"account_id"`
	Success      bool   `json:"success"`
	Restored     bool   `json:"restored"`
	ErrorMessage string `json:"error_message,omitempty"`
	ResponseText string `json:"response_text,omitempty"`
	LatencyMs    int64  `json:"latency_ms,omitempty"`
}

func normalizeBlacklistRetestSourceProtocol(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case service.PlatformOpenAI, service.PlatformAnthropic, service.PlatformGemini:
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return ""
	}
}

func resolveBlacklistRetestModelInput(req BlacklistRetestRequest) (string, string, string) {
	mode := strings.TrimSpace(strings.ToLower(req.ModelInputMode))
	if mode != service.ScheduledTestModelInputModeManual {
		mode = service.ScheduledTestModelInputModeCatalog
	}

	modelID := strings.TrimSpace(req.ModelID)
	if mode == service.ScheduledTestModelInputModeManual {
		modelID = strings.TrimSpace(req.ManualModelID)
	}
	return modelID, mode, normalizeBlacklistRetestSourceProtocol(req.SourceProtocol)
}

func (h *AccountHandler) Blacklist(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	var req BlacklistAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	account, err := h.adminService.BlacklistAccount(c.Request.Context(), accountID, &service.BlacklistAccountInput{
		Source:   req.Source,
		Feedback: req.Feedback,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
}

func (h *AccountHandler) RetestBlacklisted(c *gin.Context) {
	if h.accountTestService == nil {
		response.Error(c, 500, "Account test service is not configured")
		return
	}

	var req BlacklistRetestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	accountIDs := normalizeInt64IDList(req.AccountIDs)
	if len(accountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}
	requestedModelID, modelInputMode, requestedSourceProtocol := resolveBlacklistRetestModelInput(req)
	slog.Info(
		"account_blacklist_retest_start",
		"account_ids", accountIDs,
		"requested_model_id", requestedModelID,
		"model_input_mode", modelInputMode,
		"source_protocol", requestedSourceProtocol,
	)

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

	results := make([]BlacklistRetestAccountResult, len(accountIDs))
	g, gctx := errgroup.WithContext(c.Request.Context())
	g.SetLimit(5)
	var mu sync.Mutex

	for index, accountID := range accountIDs {
		index := index
		accountID := accountID
		g.Go(func() error {
			result := BlacklistRetestAccountResult{AccountID: accountID}
			account := accountByID[accountID]
			switch {
			case account == nil:
				result.ErrorMessage = "account not found"
			case service.NormalizeAccountLifecycleInput(account.LifecycleState) != service.AccountLifecycleBlacklisted:
				result.ErrorMessage = "account is not blacklisted"
			default:
				testResult, err := h.accountTestService.RunTestBackground(gctx, service.ScheduledTestExecutionInput{
					AccountID:      accountID,
					ModelID:        requestedModelID,
					SourceProtocol: requestedSourceProtocol,
				})
				if testResult != nil {
					result.ResponseText = testResult.ResponseText
					result.LatencyMs = testResult.LatencyMs
					if testResult.ErrorMessage != "" {
						result.ErrorMessage = testResult.ErrorMessage
					}
				}
				if err == nil && testResult != nil && testResult.Status == "success" {
					if _, restoreErr := h.adminService.RestoreBlacklistedAccount(gctx, accountID); restoreErr != nil {
						result.ErrorMessage = restoreErr.Error()
					} else {
						result.Success = true
						result.Restored = true
						if h.rateLimitService != nil {
							if _, recoverErr := h.rateLimitService.RecoverAccountAfterSuccessfulTest(gctx, accountID); recoverErr != nil && result.ErrorMessage == "" {
								result.ErrorMessage = recoverErr.Error()
							}
						}
					}
				} else if err != nil && result.ErrorMessage == "" {
					result.ErrorMessage = err.Error()
				}
			}
			slog.Info(
				"account_blacklist_retest_result",
				"account_id", accountID,
				"requested_model_id", requestedModelID,
				"model_input_mode", modelInputMode,
				"source_protocol", requestedSourceProtocol,
				"success", result.Success,
				"restored", result.Restored,
				"error_message", result.ErrorMessage,
			)

			mu.Lock()
			results[index] = result
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()
	response.Success(c, gin.H{"results": results})
}

func (h *AccountHandler) RetestBlacklistedModels(c *gin.Context) {
	var req BlacklistRetestModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	accountIDs := normalizeInt64IDList(req.AccountIDs)
	if len(accountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
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

	response.Success(c, service.MergeAvailableTestModels(modelGroups...))
}

func (h *AccountHandler) BatchDeleteBlacklisted(c *gin.Context) {
	var req BlacklistBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	accountIDs := normalizeInt64IDList(req.IDs)
	switch {
	case req.DeleteAll && len(accountIDs) > 0:
		response.BadRequest(c, "ids and delete_all cannot be provided together")
		return
	case !req.DeleteAll && len(accountIDs) == 0:
		response.BadRequest(c, "either ids or delete_all=true is required")
		return
	}

	result, err := h.adminService.BatchDeleteBlacklistedAccounts(c.Request.Context(), accountIDs, req.DeleteAll)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}
