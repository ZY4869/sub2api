package admin

import (
	"errors"
	"io"
	"strconv"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type BlacklistRetestRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
}

type BlacklistRetestAccountResult struct {
	AccountID    int64  `json:"account_id"`
	Success      bool   `json:"success"`
	Restored     bool   `json:"restored"`
	ErrorMessage string `json:"error_message,omitempty"`
	ResponseText string `json:"response_text,omitempty"`
	LatencyMs    int64  `json:"latency_ms,omitempty"`
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
				testResult, err := h.accountTestService.RunTestBackground(gctx, accountID, "")
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

			mu.Lock()
			results[index] = result
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()
	response.Success(c, gin.H{"results": results})
}
