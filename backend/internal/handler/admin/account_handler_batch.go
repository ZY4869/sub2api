package admin

import (
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"log"
	"sync"
)

func (h *AccountHandler) BatchClearError(c *gin.Context) {
	var req struct {
		AccountIDs []int64 `json:"account_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if len(req.AccountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}
	ctx := c.Request.Context()
	const maxConcurrency = 10
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)
	var mu sync.Mutex
	var successCount, failedCount int
	var errors []gin.H
	for _, id := range req.AccountIDs {
		accountID := id
		g.Go(func() error {
			account, err := h.adminService.ClearAccountError(gctx, accountID)
			if err != nil {
				mu.Lock()
				failedCount++
				errors = append(errors, gin.H{"account_id": accountID, "error": err.Error()})
				mu.Unlock()
				return nil
			}
			if h.tokenCacheInvalidator != nil && account.IsOAuth() {
				if invalidateErr := h.tokenCacheInvalidator.InvalidateToken(gctx, account); invalidateErr != nil {
					log.Printf("[WARN] Failed to invalidate token cache for account %d: %v", accountID, invalidateErr)
				}
			}
			mu.Lock()
			successCount++
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"total": len(req.AccountIDs), "success": successCount, "failed": failedCount, "errors": errors})
}
func (h *AccountHandler) BatchRefresh(c *gin.Context) {
	var req struct {
		AccountIDs []int64 `json:"account_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if len(req.AccountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}
	ctx := c.Request.Context()
	accounts, err := h.adminService.GetAccountsByIDs(ctx, req.AccountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	foundIDs := make(map[int64]bool, len(accounts))
	for _, acc := range accounts {
		if acc != nil {
			foundIDs[acc.ID] = true
		}
	}
	const maxConcurrency = 10
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)
	var mu sync.Mutex
	var successCount, failedCount int
	var errors []gin.H
	var warnings []gin.H
	for _, id := range req.AccountIDs {
		if !foundIDs[id] {
			failedCount++
			errors = append(errors, gin.H{"account_id": id, "error": "account not found"})
		}
	}
	for _, account := range accounts {
		acc := account
		if acc == nil {
			continue
		}
		g.Go(func() error {
			_, warning, err := h.refreshSingleAccount(gctx, acc)
			mu.Lock()
			if err != nil {
				failedCount++
				errors = append(errors, gin.H{"account_id": acc.ID, "error": err.Error()})
			} else {
				successCount++
				if warning != "" {
					warnings = append(warnings, gin.H{"account_id": acc.ID, "warning": warning})
				}
			}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"total": len(req.AccountIDs), "success": successCount, "failed": failedCount, "errors": errors, "warnings": warnings})
}
type BatchUpdateCredentialsRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
	Field      string  `json:"field" binding:"required,oneof=account_uuid org_uuid intercept_warmup_requests"`
	Value      any     `json:"value"`
}

func (h *AccountHandler) BatchUpdateCredentials(c *gin.Context) {
	var req BatchUpdateCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.Field == "intercept_warmup_requests" {
		if _, ok := req.Value.(bool); !ok {
			response.BadRequest(c, "intercept_warmup_requests must be boolean")
			return
		}
	} else {
		if req.Value != nil {
			if _, ok := req.Value.(string); !ok {
				response.BadRequest(c, req.Field+" must be string or null")
				return
			}
		}
	}
	ctx := c.Request.Context()
	type accountUpdate struct {
		ID          int64
		Credentials map[string]any
	}
	updates := make([]accountUpdate, 0, len(req.AccountIDs))
	for _, accountID := range req.AccountIDs {
		account, err := h.adminService.GetAccount(ctx, accountID)
		if err != nil {
			response.Error(c, 404, fmt.Sprintf("Account %d not found", accountID))
			return
		}
		if account.Credentials == nil {
			account.Credentials = make(map[string]any)
		}
		account.Credentials[req.Field] = req.Value
		updates = append(updates, accountUpdate{ID: accountID, Credentials: account.Credentials})
	}
	success := 0
	failed := 0
	successIDs := make([]int64, 0, len(updates))
	failedIDs := make([]int64, 0, len(updates))
	results := make([]gin.H, 0, len(updates))
	for _, u := range updates {
		updateInput := &service.UpdateAccountInput{Credentials: u.Credentials}
		if _, err := h.adminService.UpdateAccount(ctx, u.ID, updateInput); err != nil {
			failed++
			failedIDs = append(failedIDs, u.ID)
			results = append(results, gin.H{"account_id": u.ID, "success": false, "error": err.Error()})
			continue
		}
		success++
		successIDs = append(successIDs, u.ID)
		results = append(results, gin.H{"account_id": u.ID, "success": true})
	}
	response.Success(c, gin.H{"success": success, "failed": failed, "success_ids": successIDs, "failed_ids": failedIDs, "results": results})
}
