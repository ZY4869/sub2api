package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func rejectUnsupportedPlatform(c *gin.Context, platform string) bool {
	if err := service.EnsureSupportedPrimaryPlatform(platform); err != nil {
		response.ErrorFrom(c, err)
		return true
	}
	return false
}

func (h *AccountHandler) rejectUnsupportedAccountByID(c *gin.Context, accountID int64) (*service.Account, bool) {
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return nil, true
	}
	if account == nil {
		response.NotFound(c, response.LocalizedMessage(c, "admin.account.not_found", "Account not found"))
		return nil, true
	}
	if err := service.EnsureSupportedAccountPlatform(account); err != nil {
		response.ErrorFrom(c, err)
		return nil, true
	}
	return account, false
}
