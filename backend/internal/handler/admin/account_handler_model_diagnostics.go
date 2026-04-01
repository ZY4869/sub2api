package admin

import (
	"errors"
	"io"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

type accountModelDiagnosticsRequest struct {
	Refresh bool `json:"refresh"`
}

func (h *AccountHandler) DiagnoseModels(c *gin.Context) {
	if h.accountModelDiagnostics == nil {
		response.InternalError(c, "Account model diagnostics service unavailable")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	var req accountModelDiagnosticsRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.accountModelDiagnostics.Diagnose(c.Request.Context(), accountID, req.Refresh)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}
