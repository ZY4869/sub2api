package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ChannelMonitorHandler struct {
	monitorService *service.ChannelMonitorService
}

func NewChannelMonitorHandler(monitorService *service.ChannelMonitorService) *ChannelMonitorHandler {
	return &ChannelMonitorHandler{monitorService: monitorService}
}

// ListUserView returns the user-facing channel monitor overview.
// GET /api/v1/channel-monitors
func (h *ChannelMonitorHandler) ListUserView(c *gin.Context) {
	if _, ok := middleware.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	items, err := h.monitorService.ListUserView(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

// GetStatus returns a single monitor status detail.
// GET /api/v1/channel-monitors/:id/status
func (h *ChannelMonitorHandler) GetStatus(c *gin.Context) {
	if _, ok := middleware.GetAuthSubjectFromContext(c); !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	monitorID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || monitorID <= 0 {
		response.BadRequest(c, "Invalid monitor ID")
		return
	}
	out, err := h.monitorService.GetUserDetail(c.Request.Context(), monitorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}
