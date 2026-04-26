package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ChannelMonitorTemplateHandler struct {
	templateService *service.ChannelMonitorTemplateService
	monitorService  *service.ChannelMonitorService
}

func NewChannelMonitorTemplateHandler(templateService *service.ChannelMonitorTemplateService, monitorService *service.ChannelMonitorService) *ChannelMonitorTemplateHandler {
	return &ChannelMonitorTemplateHandler{
		templateService: templateService,
		monitorService:  monitorService,
	}
}

type createChannelMonitorTemplateRequest struct {
	Name             string            `json:"name" binding:"required"`
	Provider         string            `json:"provider" binding:"required"`
	Description      *string           `json:"description"`
	ExtraHeaders     map[string]string `json:"extra_headers"`
	BodyOverrideMode string            `json:"body_override_mode"`
	BodyOverride     map[string]any    `json:"body_override"`
}

func (h *ChannelMonitorTemplateHandler) List(c *gin.Context) {
	items, err := h.templateService.ListAll(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if items == nil {
		items = make([]*service.ChannelMonitorRequestTemplate, 0)
	}
	response.Success(c, items)
}

func (h *ChannelMonitorTemplateHandler) GetByID(c *gin.Context) {
	templateID, ok := parseChannelMonitorTemplateID(c)
	if !ok {
		return
	}
	item, err := h.templateService.GetByID(c.Request.Context(), templateID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *ChannelMonitorTemplateHandler) Create(c *gin.Context) {
	var req createChannelMonitorTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.templateService.Create(c.Request.Context(), &service.ChannelMonitorRequestTemplate{
		Name:             req.Name,
		Provider:         req.Provider,
		Description:      req.Description,
		ExtraHeaders:     req.ExtraHeaders,
		BodyOverrideMode: req.BodyOverrideMode,
		BodyOverride:     req.BodyOverride,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

type updateChannelMonitorTemplateRequest struct {
	Name             *string            `json:"name"`
	Provider         *string            `json:"provider"`
	Description      **string           `json:"description"`
	ExtraHeaders     *map[string]string `json:"extra_headers"`
	BodyOverrideMode *string            `json:"body_override_mode"`
	BodyOverride     *map[string]any    `json:"body_override"`
}

func (h *ChannelMonitorTemplateHandler) Update(c *gin.Context) {
	templateID, ok := parseChannelMonitorTemplateID(c)
	if !ok {
		return
	}
	var req updateChannelMonitorTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	existing, err := h.templateService.GetByID(c.Request.Context(), templateID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Provider != nil {
		existing.Provider = *req.Provider
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.ExtraHeaders != nil {
		existing.ExtraHeaders = *req.ExtraHeaders
	}
	if req.BodyOverrideMode != nil {
		existing.BodyOverrideMode = *req.BodyOverrideMode
	}
	if req.BodyOverride != nil {
		existing.BodyOverride = *req.BodyOverride
	}
	updated, err := h.templateService.Update(c.Request.Context(), existing)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

func (h *ChannelMonitorTemplateHandler) Delete(c *gin.Context) {
	templateID, ok := parseChannelMonitorTemplateID(c)
	if !ok {
		return
	}
	if err := h.templateService.Delete(c.Request.Context(), templateID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Channel monitor template deleted successfully"})
}

type applyTemplateRequest struct {
	MonitorID int64 `json:"monitor_id" binding:"required"`
}

func (h *ChannelMonitorTemplateHandler) ApplyToMonitor(c *gin.Context) {
	templateID, ok := parseChannelMonitorTemplateID(c)
	if !ok {
		return
	}
	var req applyTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	monitor, err := h.templateService.ApplyToMonitor(c.Request.Context(), templateID, req.MonitorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toAdminChannelMonitorView(h.monitorService, monitor))
}

func (h *ChannelMonitorTemplateHandler) ListAssociatedMonitors(c *gin.Context) {
	templateID, ok := parseChannelMonitorTemplateID(c)
	if !ok {
		return
	}
	monitors, err := h.templateService.ListAssociatedMonitors(c.Request.Context(), templateID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	type associatedMonitor struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	out := make([]associatedMonitor, 0, len(monitors))
	for _, m := range monitors {
		if m == nil {
			continue
		}
		out = append(out, associatedMonitor{ID: m.ID, Name: m.Name})
	}
	response.Success(c, out)
}

func parseChannelMonitorTemplateID(c *gin.Context) (int64, bool) {
	templateID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || templateID <= 0 {
		response.BadRequest(c, "Invalid template ID")
		return 0, false
	}
	return templateID, true
}
