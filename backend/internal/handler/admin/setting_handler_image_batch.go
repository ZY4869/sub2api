package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) GetImageBatchSettings(c *gin.Context) {
	response.Success(c, dto.ImageBatchSettings{
		Enabled: h.settingService.IsImageBatchEnabled(c.Request.Context()),
	})
}

func (h *SettingHandler) UpdateImageBatchSettings(c *gin.Context) {
	var req dto.ImageBatchSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid image batch settings")
		return
	}
	if err := h.settingService.SetImageBatchEnabled(c.Request.Context(), req.Enabled); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, req)
}
