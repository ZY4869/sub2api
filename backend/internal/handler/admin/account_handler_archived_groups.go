package admin

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

type UnarchiveAccountsRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
}

func (h *AccountHandler) ListArchivedGroups(c *gin.Context) {
	platform := c.Query("platform")
	accountType := c.Query("type")
	status := service.NormalizeAdminAccountStatusInput(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 100 {
		search = search[:100]
	}

	var groupID int64
	if groupIDStr := c.Query("group"); groupIDStr != "" {
		if groupIDStr == accountListGroupUngroupedQueryValue {
			groupID = service.AccountListGroupUngrouped
		} else {
			parsedGroupID, err := strconv.ParseInt(groupIDStr, 10, 64)
			if err != nil || parsedGroupID < 0 {
				response.BadRequest(c, "Invalid group filter")
				return
			}
			groupID = parsedGroupID
		}
	}

	items, err := h.adminService.ListArchivedGroups(c.Request.Context(), service.ArchivedAccountGroupFilters{
		Platform:    platform,
		AccountType: accountType,
		Status:      status,
		Search:      search,
		GroupID:     groupID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *AccountHandler) UnarchiveAccounts(c *gin.Context) {
	var req UnarchiveAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	executeAdminIdempotentJSON(c, "admin.accounts.unarchive", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.adminService.UnarchiveAccounts(ctx, &service.UnarchiveAccountsInput{
			AccountIDs: req.AccountIDs,
		})
	})
}
