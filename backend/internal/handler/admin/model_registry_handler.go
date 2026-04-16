package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ModelRegistryHandler struct {
	modelRegistryService *service.ModelRegistryService
}

func NewModelRegistryHandler(modelRegistryService *service.ModelRegistryService) *ModelRegistryHandler {
	return &ModelRegistryHandler{modelRegistryService: modelRegistryService}
}

func (h *ModelRegistryHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	filter := service.ModelRegistryListFilter{
		Search:            c.Query("search"),
		Provider:          c.Query("provider"),
		Platform:          c.Query("platform"),
		Exposure:          c.Query("exposure"),
		Status:            c.Query("status"),
		Availability:      c.Query("availability"),
		SortMode:          c.Query("sort_mode"),
		IncludeHidden:     parseBoolDefaultTrue(c.Query("include_hidden")),
		IncludeTombstoned: parseBoolDefaultTrue(c.Query("include_tombstoned")),
		Page:              page,
		PageSize:          pageSize,
	}
	log := logger.FromContext(c.Request.Context()).With(zap.String("component", "handler.admin.model_registry"))
	log.Info("list model registry start", zap.Any("filter", filter))
	items, total, err := h.modelRegistryService.List(c.Request.Context(), filter)
	if err != nil {
		log.Warn("list model registry failed", zap.Error(err))
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *ModelRegistryHandler) Detail(c *gin.Context) {
	model := strings.TrimSpace(c.Query("model"))
	detail, err := h.modelRegistryService.GetDetail(c.Request.Context(), model)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

func (h *ModelRegistryHandler) ListProviders(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.modelRegistryService.ListProviderSummaries(c.Request.Context(), page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *ModelRegistryHandler) UpsertEntry(c *gin.Context) {
	var req service.UpsertModelRegistryEntryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	detail, err := h.modelRegistryService.UpsertEntry(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

func (h *ModelRegistryHandler) ManualAdd(c *gin.Context) {
	var req service.ManualAddModelRegistryEntryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	adminUserID := int64(0)
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		adminUserID = subject.UserID
	}

	log := logger.FromContext(c.Request.Context()).With(
		zap.String("component", "handler.admin.model_registry"),
		zap.Int64("admin_user_id", adminUserID),
		zap.String("model", strings.TrimSpace(req.ID)),
		zap.String("requested_provider", strings.TrimSpace(req.Provider)),
	)
	log.Info("manual add model registry entry start")

	detail, createdRuntime, activated, err := h.modelRegistryService.ManualAddEntry(c.Request.Context(), req)
	if err != nil {
		log.Warn("manual add model registry entry failed", zap.Error(err))
		response.ErrorFrom(c, err)
		return
	}

	log.Info("manual add model registry entry success",
		zap.String("requested_provider", strings.TrimSpace(req.Provider)),
		zap.String("provider", detail.Provider),
		zap.Bool("created_runtime_entry", createdRuntime),
		zap.Bool("activated", activated),
	)
	response.Success(c, service.ManualAddModelRegistryEntryResponse{
		Item:      *detail,
		Activated: activated,
	})
}

func (h *ModelRegistryHandler) SetVisibility(c *gin.Context) {
	var req service.UpdateModelRegistryVisibilityInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	detail, err := h.modelRegistryService.SetVisibility(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

func (h *ModelRegistryHandler) SyncExposures(c *gin.Context) {
	var req service.BatchSyncModelRegistryExposuresInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.modelRegistryService.BatchSyncExposures(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ModelRegistryHandler) MoveProvider(c *gin.Context) {
	var req service.MoveModelRegistryProviderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.modelRegistryService.MoveEntriesToProvider(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *ModelRegistryHandler) Activate(c *gin.Context) {
	var req service.UpdateModelRegistryAvailabilityInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	items, err := h.modelRegistryService.ActivateModels(c.Request.Context(), req.Models)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *ModelRegistryHandler) Deactivate(c *gin.Context) {
	var req service.UpdateModelRegistryAvailabilityInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	items, err := h.modelRegistryService.DeactivateModels(c.Request.Context(), req.Models)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

func (h *ModelRegistryHandler) DeleteEntry(c *gin.Context) {
	model := strings.TrimSpace(c.Query("model"))
	if err := h.modelRegistryService.DeleteEntry(c.Request.Context(), model); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"model": strings.TrimSpace(model)})
}

func (h *ModelRegistryHandler) HardDelete(c *gin.Context) {
	var req service.BatchHardDeleteModelRegistryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	log := logger.FromContext(c.Request.Context()).With(zap.String("component", "handler.admin.model_registry"))
	log.Info("hard delete model registry entries start", zap.Int("model_count", len(req.Models)))
	models, err := h.modelRegistryService.HardDeleteModels(c.Request.Context(), req.Models)
	if err != nil {
		log.Warn("hard delete model registry entries failed", zap.Error(err))
		response.ErrorFrom(c, err)
		return
	}
	log.Info("hard delete model registry entries success", zap.Int("model_count", len(models)), zap.Strings("models", models))
	response.Success(c, gin.H{"models": models})
}

func parseBoolDefaultTrue(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return true
	}
	return value == "1" || value == "true" || value == "yes"
}
