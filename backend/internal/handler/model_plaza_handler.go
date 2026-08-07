package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelPlazaHandler struct {
	modelPlazaService *service.ModelPlazaService
	apiKeyService     *service.APIKeyService
	settingService    *service.SettingService
}

func NewModelPlazaHandler(modelPlazaService *service.ModelPlazaService, apiKeyService *service.APIKeyService, settingService *service.SettingService) *ModelPlazaHandler {
	return &ModelPlazaHandler{
		modelPlazaService: modelPlazaService,
		apiKeyService:     apiKeyService,
		settingService:    settingService,
	}
}

func (h *ModelPlazaHandler) List(c *gin.Context) {
	if h.settingService != nil && !h.settingService.GetAvailableChannelsRuntime(c.Request.Context()).Enabled {
		response.Success(c, publicModelPlazaResponse{Description: "", Groups: []publicModelPlazaGroup{}})
		return
	}
	allowedGroupIDs := map[int64]struct{}{}
	authenticated := false
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 && h.apiKeyService != nil {
		authenticated = true
		groups, err := h.apiKeyService.GetAvailableGroups(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		for i := range groups {
			allowedGroupIDs[groups[i].ID] = struct{}{}
		}
	}
	result, err := h.modelPlazaService.List(c.Request.Context(), allowedGroupIDs, authenticated)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toPublicModelPlazaResponse(result))
}
