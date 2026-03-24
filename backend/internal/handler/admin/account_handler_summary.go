package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) GetStatusSummary(c *gin.Context) {
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

	summary, err := h.adminService.GetAccountStatusSummary(c.Request.Context(), service.AccountStatusSummaryFilters{
		Platform:    c.Query("platform"),
		AccountType: c.Query("type"),
		Search:      search,
		GroupID:     groupID,
		Lifecycle:   c.DefaultQuery("lifecycle", service.AccountLifecycleNormal),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}
