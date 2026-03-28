package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SetPrivacy handles setting privacy mode for a single OAuth account.
// POST /api/v1/admin/accounts/:id/set-privacy
func (h *AccountHandler) SetPrivacy(c *gin.Context) {
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
	if account == nil {
		response.NotFound(c, "Account not found")
		return
	}
	if account.Type != service.AccountTypeOAuth {
		response.BadRequest(c, "Only OAuth accounts support privacy setting")
		return
	}
	if account.Platform != service.PlatformOpenAI {
		response.BadRequest(c, "Only OpenAI OAuth accounts support privacy setting")
		return
	}

	mode := h.adminService.ForceOpenAIPrivacy(c.Request.Context(), account)
	if mode == "" {
		response.BadRequest(c, "Cannot set privacy: missing access token or privacy client")
		return
	}

	updated, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil || updated == nil {
		if account.Extra == nil {
			account.Extra = make(map[string]any)
		}
		account.Extra["privacy_mode"] = mode
		response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
		return
	}

	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), updated))
}
