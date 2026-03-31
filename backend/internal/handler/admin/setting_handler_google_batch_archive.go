package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func googleBatchArchiveSettingsDTO(settings *service.GoogleBatchArchiveSettings) dto.GoogleBatchArchiveSettings {
	if settings == nil {
		settings = service.DefaultGoogleBatchArchiveSettings()
	}
	return dto.GoogleBatchArchiveSettings{
		Enabled:                settings.Enabled,
		PollMinIntervalSeconds: settings.PollMinIntervalSeconds,
		PollMaxIntervalSeconds: settings.PollMaxIntervalSeconds,
		PollBackoffFactor:      settings.PollBackoffFactor,
		PollJitterSeconds:      settings.PollJitterSeconds,
		PollMaxConcurrency:     settings.PollMaxConcurrency,
		PrefetchAfterHours:     settings.PrefetchAfterHours,
		DownloadTimeoutSeconds: settings.DownloadTimeoutSeconds,
		CleanupIntervalMinutes: settings.CleanupIntervalMinutes,
		LocalStorageRoot:       settings.LocalStorageRoot,
	}
}

func (h *SettingHandler) GetGoogleBatchArchiveSettings(c *gin.Context) {
	settings, err := h.settingService.GetGoogleBatchArchiveSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, googleBatchArchiveSettingsDTO(settings))
}

func (h *SettingHandler) UpdateGoogleBatchArchiveSettings(c *gin.Context) {
	var req dto.GoogleBatchArchiveSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settings, err := h.settingService.UpdateGoogleBatchArchiveSettings(c.Request.Context(), &service.GoogleBatchArchiveSettings{
		Enabled:                req.Enabled,
		PollMinIntervalSeconds: req.PollMinIntervalSeconds,
		PollMaxIntervalSeconds: req.PollMaxIntervalSeconds,
		PollBackoffFactor:      req.PollBackoffFactor,
		PollJitterSeconds:      req.PollJitterSeconds,
		PollMaxConcurrency:     req.PollMaxConcurrency,
		PrefetchAfterHours:     req.PrefetchAfterHours,
		DownloadTimeoutSeconds: req.DownloadTimeoutSeconds,
		CleanupIntervalMinutes: req.CleanupIntervalMinutes,
		LocalStorageRoot:       req.LocalStorageRoot,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, googleBatchArchiveSettingsDTO(settings))
}
