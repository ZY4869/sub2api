package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// GetAffiliate returns the current user's affiliate info.
// GET /api/v1/user/aff
func (h *UserHandler) GetAffiliate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.affiliate == nil {
		response.InternalError(c, "Service not configured")
		return
	}

	info, err := h.affiliate.GetMyAffiliateInfo(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, info)
}

// TransferAffiliate transfers the current user's available rebate balance into their main balance.
// POST /api/v1/user/aff/transfer
func (h *UserHandler) TransferAffiliate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h == nil || h.affiliate == nil {
		response.InternalError(c, "Service not configured")
		return
	}

	result, err := h.affiliate.TransferToBalance(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
