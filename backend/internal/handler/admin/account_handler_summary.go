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

	filters := service.AccountStatusSummaryFilters{
		Platform:      c.Query("platform"),
		AccountType:   c.Query("type"),
		Search:        search,
		GroupID:       groupID,
		Lifecycle:     c.DefaultQuery("lifecycle", service.AccountLifecycleNormal),
		LimitedView:   service.NormalizeAccountLimitedViewInput(c.DefaultQuery("limited_view", service.AccountLimitedViewAll)),
		LimitedReason: service.NormalizeAccountRateLimitReasonInput(c.Query("limited_reason")),
		RuntimeView:   service.NormalizeAccountRuntimeViewInput(c.DefaultQuery("runtime_view", service.AccountRuntimeViewAll)),
	}
	requestCtx := service.WithAccountLimitedFilters(c.Request.Context(), filters.LimitedView, filters.LimitedReason)
	requestCtx, candidateIDs, err := h.buildRuntimeQueryContext(requestCtx, filters.RuntimeView)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	summary, err := h.adminService.GetAccountStatusSummary(requestCtx, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if filters.RuntimeView == service.AccountRuntimeViewInUseOnly {
		if len(candidateIDs) == 0 {
			summary.InUse = 0
		} else {
			summary.InUse = summary.Total
		}
		response.Success(c, summary)
		return
	}
	if h.concurrencyService == nil && h.sessionLimitCache == nil {
		response.Success(c, summary)
		return
	}
	inUseCount, err := h.countInUseAccounts(c.Request.Context(), filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	summary.InUse = inUseCount
	response.Success(c, summary)
}
