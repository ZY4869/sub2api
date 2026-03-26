package admin

import (
	"context"
	"log"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) refreshSingleAccount(ctx context.Context, account *service.Account) (*service.Account, string, error) {
	if account == nil {
		return nil, "", service.ErrAccountNotFound
	}

	refreshed, err := h.adminService.RefreshAccountCredentials(ctx, account.ID)
	if err != nil {
		return nil, "", err
	}
	if refreshed == nil {
		refreshed = account
	}

	warning := ""
	if h.tokenCacheInvalidator != nil && refreshed.IsOAuth() {
		if err := h.tokenCacheInvalidator.InvalidateToken(ctx, refreshed); err != nil {
			warning = err.Error()
		}
	}

	return refreshed, warning, nil
}

func (h *AccountHandler) Refresh(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	refreshed, warning, err := h.refreshSingleAccount(c.Request.Context(), account)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if warning != "" {
		log.Printf("[WARN] Failed to invalidate token cache for account %d after refresh: %v", accountID, warning)
	}

	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), refreshed))
}

func (h *AccountHandler) RefreshTier(c *gin.Context) {
	h.Refresh(c)
}

func (h *AccountHandler) BatchRefreshTier(c *gin.Context) {
	h.BatchRefresh(c)
}
