package admin

import (
	"encoding/json"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
	Name             string          `json:"name" binding:"required"`
	Provider         string          `json:"provider" binding:"required"`
	Description      *string         `json:"description"`
	ExtraHeaders     json.RawMessage `json:"extra_headers"`
	BodyOverrideMode json.RawMessage `json:"body_override_mode"`
	BodyOverride     json.RawMessage `json:"body_override"`
	OpenAIAPIMode    json.RawMessage `json:"openai_api_mode"`
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
		response.ErrorFrom(c, service.ErrChannelMonitorInvalidRequest)
		return
	}
	extraHeaders, _, err := parseChannelMonitorHeaders(req.ExtraHeaders)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	bodyOverride, _, err := parseChannelMonitorBodyOverride(req.BodyOverride)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	bodyOverrideMode, _, err := parseChannelMonitorMode(req.BodyOverrideMode, service.ErrChannelMonitorInvalidOverrideMode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	openAIAPIMode, _, err := parseChannelMonitorMode(req.OpenAIAPIMode, service.ErrChannelMonitorInvalidOpenAIAPIMode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	item, err := h.templateService.Create(c.Request.Context(), &service.ChannelMonitorRequestTemplate{
		Name:             req.Name,
		Provider:         req.Provider,
		Description:      req.Description,
		ExtraHeaders:     extraHeaders,
		BodyOverrideMode: bodyOverrideMode,
		BodyOverride:     bodyOverride,
		OpenAIAPIMode:    openAIAPIMode,
	})
	if err != nil {
		logger.FromContext(c.Request.Context()).Warn(
			"channel_monitor_template_create_failed",
			zap.String("component", "handler.admin.channel_monitor_template"),
			zap.String("provider", req.Provider),
			zap.Bool("has_name", req.Name != ""),
			zap.Error(err),
		)
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

type updateChannelMonitorTemplateRequest struct {
	Name             *string         `json:"name"`
	Provider         *string         `json:"provider"`
	Description      **string        `json:"description"`
	ExtraHeaders     json.RawMessage `json:"extra_headers"`
	BodyOverrideMode json.RawMessage `json:"body_override_mode"`
	BodyOverride     json.RawMessage `json:"body_override"`
	OpenAIAPIMode    json.RawMessage `json:"openai_api_mode"`
}

func (h *ChannelMonitorTemplateHandler) Update(c *gin.Context) {
	templateID, ok := parseChannelMonitorTemplateID(c)
	if !ok {
		return
	}
	var req updateChannelMonitorTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrChannelMonitorInvalidRequest)
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
	if extraHeaders, present, err := parseChannelMonitorHeaders(req.ExtraHeaders); err != nil {
		response.ErrorFrom(c, err)
		return
	} else if present {
		existing.ExtraHeaders = extraHeaders
	}
	if bodyOverrideMode, present, err := parseChannelMonitorMode(req.BodyOverrideMode, service.ErrChannelMonitorInvalidOverrideMode); err != nil {
		response.ErrorFrom(c, err)
		return
	} else if present {
		existing.BodyOverrideMode = bodyOverrideMode
	}
	if bodyOverride, present, err := parseChannelMonitorBodyOverride(req.BodyOverride); err != nil {
		response.ErrorFrom(c, err)
		return
	} else if present {
		existing.BodyOverride = bodyOverride
	}
	if openAIAPIMode, present, err := parseChannelMonitorMode(req.OpenAIAPIMode, service.ErrChannelMonitorInvalidOpenAIAPIMode); err != nil {
		response.ErrorFrom(c, err)
		return
	} else if present {
		existing.OpenAIAPIMode = openAIAPIMode
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
