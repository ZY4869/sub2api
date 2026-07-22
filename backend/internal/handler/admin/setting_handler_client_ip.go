package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) GetClientIPSettings(c *gin.Context) {
	settings, err := h.settingService.GetClientIPSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *SettingHandler) UpdateClientIPSettings(c *gin.Context) {
	var req service.ClientIPSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid client IP settings")
		return
	}
	settings, err := h.settingService.SetClientIPSettings(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}
