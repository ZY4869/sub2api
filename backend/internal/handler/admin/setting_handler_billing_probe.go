package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *SettingHandler) GetUpstreamBillingProbeSettings(c *gin.Context) {
	settings, err := h.settingService.GetUpstreamBillingProbeSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UpstreamBillingProbeSettings{
		Enabled:          settings.Enabled,
		BatchConcurrency: settings.BatchConcurrency,
		TimeoutSeconds:   settings.TimeoutSeconds,
	})
}

func (h *SettingHandler) UpdateUpstreamBillingProbeSettings(c *gin.Context) {
	var req dto.UpstreamBillingProbeSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid upstream billing probe settings")
		return
	}
	settings, err := h.settingService.UpdateUpstreamBillingProbeSettings(c.Request.Context(), &service.UpstreamBillingProbeSettings{
		Enabled:          req.Enabled,
		BatchConcurrency: req.BatchConcurrency,
		TimeoutSeconds:   req.TimeoutSeconds,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UpstreamBillingProbeSettings{
		Enabled:          settings.Enabled,
		BatchConcurrency: settings.BatchConcurrency,
		TimeoutSeconds:   settings.TimeoutSeconds,
	})
}
