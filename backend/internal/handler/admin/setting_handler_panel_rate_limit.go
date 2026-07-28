package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) GetPanelRateLimitSettings(c *gin.Context) {
	settings, err := h.settingService.GetPanelRateLimitSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *SettingHandler) UpdatePanelRateLimitSettings(c *gin.Context) {
	var req service.PanelRateLimitSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid panel rate limit settings")
		return
	}
	settings, err := h.settingService.SetPanelRateLimitSettings(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}
