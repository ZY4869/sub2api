package admin

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"io"
	"strconv"
)

func (h *AccountHandler) ImportModels(c *gin.Context) {
	if h.accountModelImportService == nil {
		response.InternalError(c, "Account model import service unavailable")
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req ImportAccountModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	ctx := c.Request.Context()
	account, err := h.adminService.GetAccount(ctx, accountID)
	if err != nil || account == nil {
		response.NotFound(c, "Account not found")
		return
	}
	result, err := h.accountModelImportService.ImportAccountModels(ctx, account, req.Trigger)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
func (h *AccountHandler) GetAvailableModels(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.NotFound(c, "Account not found")
		return
	}
	models := h.defaultAvailableModels(c.Request.Context(), account)
	if account.Platform == service.PlatformAntigravity || account.Platform == service.PlatformSora || account.IsOAuth() {
		response.Success(c, models)
		return
	}
	mapping := account.GetModelMapping()
	if len(mapping) == 0 {
		response.Success(c, models)
		return
	}
	response.Success(c, filterAvailableModelsByMapping(models, mapping))
}

type availableModelItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

func (h *AccountHandler) defaultAvailableModels(ctx context.Context, account *service.Account) []availableModelItem {
	if account == nil {
		return []availableModelItem{}
	}
	if account.Platform == service.PlatformSora {
		defaults := service.DefaultSoraModels(nil)
		items := make([]availableModelItem, 0, len(defaults))
		for _, model := range defaults {
			items = append(items, availableModelItem{ID: model.ID, Type: model.Type, DisplayName: model.DisplayName, CreatedAt: ""})
		}
		return items
	}
	if h.modelRegistryService != nil {
		for _, exposures := range [][]string{{"test"}, {"runtime", "whitelist"}} {
			entries, err := h.modelRegistryService.GetModelsByPlatform(ctx, account.Platform, exposures...)
			if err != nil || len(entries) == 0 {
				continue
			}
			items := make([]availableModelItem, 0, len(entries))
			for _, entry := range entries {
				displayName := entry.DisplayName
				if displayName == "" {
					displayName = entry.ID
				}
				items = append(items, availableModelItem{ID: entry.ID, Type: "model", DisplayName: displayName, CreatedAt: ""})
			}
			return items
		}
	}
	items := make([]availableModelItem, 0, len(claude.DefaultModels))
	for _, model := range claude.DefaultModels {
		items = append(items, availableModelItem{ID: model.ID, Type: model.Type, DisplayName: model.DisplayName, CreatedAt: model.CreatedAt})
	}
	return items
}
func filterAvailableModelsByMapping(defaults []availableModelItem, mapping map[string]string) []availableModelItem {
	if len(mapping) == 0 {
		return defaults
	}
	index := make(map[string]availableModelItem, len(defaults))
	for _, model := range defaults {
		index[model.ID] = model
	}
	items := make([]availableModelItem, 0, len(mapping))
	for requestedModel := range mapping {
		if model, ok := index[requestedModel]; ok {
			items = append(items, model)
			continue
		}
		items = append(items, availableModelItem{ID: requestedModel, Type: "model", DisplayName: requestedModel, CreatedAt: ""})
	}
	return items
}
