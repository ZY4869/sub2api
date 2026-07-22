package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) GetImageBatchStorageSettings(c *gin.Context) {
	settings, err := h.settingService.GetImageBatchStorageSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *SettingHandler) UpdateImageBatchStorageSettings(c *gin.Context) {
	var req service.ImageBatchStorageSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid image batch storage settings")
		return
	}
	settings, err := h.settingService.SetImageBatchStorageSettings(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *SettingHandler) TestImageBatchStorageSettings(c *gin.Context) {
	var req service.ImageBatchStorageSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid image batch storage settings")
		return
	}
	if err := h.settingService.TestImageBatchStorageSettings(c.Request.Context(), &req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
