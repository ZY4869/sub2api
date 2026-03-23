package admin

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
)

func (h *AccountHandler) Create(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		response.BadRequest(c, "rate_multiplier must be >= 0")
		return
	}
	sanitizeExtraBaseRPM(req.Extra)
	skipCheck := req.ConfirmMixedChannelRisk != nil && *req.ConfirmMixedChannelRisk
	result, err := executeAdminIdempotent(c, "admin.accounts.create", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		credentials, extra, scopeErr := h.prepareAccountModelScope(ctx, req.Platform, req.Type, req.Credentials, req.Extra)
		if scopeErr != nil {
			return nil, scopeErr
		}
		account, execErr := h.adminService.CreateAccount(ctx, &service.CreateAccountInput{Name: req.Name, Notes: req.Notes, Platform: req.Platform, Type: req.Type, Credentials: credentials, Extra: extra, ProxyID: req.ProxyID, Concurrency: req.Concurrency, Priority: req.Priority, RateMultiplier: req.RateMultiplier, LoadFactor: req.LoadFactor, GroupIDs: req.GroupIDs, ExpiresAt: req.ExpiresAt, AutoPauseOnExpired: req.AutoPauseOnExpired, SkipMixedChannelCheck: skipCheck})
		if execErr != nil {
			return nil, execErr
		}
		return h.buildAccountResponseWithRuntime(ctx, account), nil
	})
	if err != nil {
		var mixedErr *service.MixedChannelError
		if errors.As(err, &mixedErr) {
			c.JSON(409, gin.H{"error": "mixed_channel_warning", "message": mixedErr.Error()})
			return
		}
		if retryAfter := service.RetryAfterSecondsFromError(err); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		response.ErrorFrom(c, err)
		return
	}
	if result != nil && result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	response.Success(c, result.Data)
}
func (h *AccountHandler) Update(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		response.BadRequest(c, "rate_multiplier must be >= 0")
		return
	}
	sanitizeExtraBaseRPM(req.Extra)
	skipCheck := req.ConfirmMixedChannelRisk != nil && *req.ConfirmMixedChannelRisk
	accountBeforeUpdate, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if accountBeforeUpdate == nil {
		response.NotFound(c, "Account not found")
		return
	}
	accountType := req.Type
	if accountType == "" && accountBeforeUpdate != nil {
		accountType = accountBeforeUpdate.Type
	}
	normalizedStatus := service.NormalizeAdminAccountStatusInput(req.Status)
	credentials, extra, scopeErr := h.prepareAccountModelScope(c.Request.Context(), accountBeforeUpdate.Platform, accountType, req.Credentials, req.Extra)
	if scopeErr != nil {
		response.ErrorFrom(c, scopeErr)
		return
	}
	account, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{Name: req.Name, Notes: req.Notes, Type: req.Type, Credentials: credentials, Extra: extra, ProxyID: req.ProxyID, Concurrency: req.Concurrency, Priority: req.Priority, RateMultiplier: req.RateMultiplier, LoadFactor: req.LoadFactor, Status: normalizedStatus, GroupIDs: req.GroupIDs, ExpiresAt: req.ExpiresAt, AutoPauseOnExpired: req.AutoPauseOnExpired, SkipMixedChannelCheck: skipCheck})
	if err != nil {
		var mixedErr *service.MixedChannelError
		if errors.As(err, &mixedErr) {
			c.JSON(409, gin.H{"error": "mixed_channel_warning", "message": mixedErr.Error()})
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
}
