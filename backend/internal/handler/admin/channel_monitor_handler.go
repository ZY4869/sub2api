package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ChannelMonitorHandler struct {
	monitorService *service.ChannelMonitorService
}

func NewChannelMonitorHandler(monitorService *service.ChannelMonitorService) *ChannelMonitorHandler {
	return &ChannelMonitorHandler{monitorService: monitorService}
}

type adminChannelMonitorView struct {
	ID                 int64             `json:"id"`
	Name               string            `json:"name"`
	Provider           string            `json:"provider"`
	Endpoint           string            `json:"endpoint"`
	IntervalSeconds    int               `json:"interval_seconds"`
	Enabled            bool              `json:"enabled"`
	PrimaryModelID     string            `json:"primary_model_id"`
	AdditionalModelIDs []string          `json:"additional_model_ids"`
	TemplateID         *int64            `json:"template_id,omitempty"`
	ExtraHeaders       map[string]string `json:"extra_headers"`
	BodyOverrideMode   string            `json:"body_override_mode"`
	BodyOverride       map[string]any    `json:"body_override"`
	LastRunAt          *time.Time        `json:"last_run_at,omitempty"`
	NextRunAt          *time.Time        `json:"next_run_at,omitempty"`

	APIKeyConfigured    bool `json:"api_key_configured"`
	APIKeyDecryptFailed bool `json:"api_key_decrypt_failed"`
}

// List monitors.
// GET /api/v1/admin/channel-monitors
func (h *ChannelMonitorHandler) List(c *gin.Context) {
	monitors, err := h.monitorService.ListAll(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]adminChannelMonitorView, 0, len(monitors))
	for _, m := range monitors {
		view := toAdminChannelMonitorView(h.monitorService, m)
		if view != nil {
			out = append(out, *view)
		}
	}
	response.Success(c, out)
}

// GetByID returns a monitor.
// GET /api/v1/admin/channel-monitors/:id
func (h *ChannelMonitorHandler) GetByID(c *gin.Context) {
	monitorID, ok := parseChannelMonitorID(c)
	if !ok {
		return
	}
	m, err := h.monitorService.GetByID(c.Request.Context(), monitorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	view := toAdminChannelMonitorView(h.monitorService, m)
	if view == nil {
		response.ErrorFrom(c, service.ErrChannelMonitorNotFound)
		return
	}
	response.Success(c, view)
}

type createChannelMonitorRequest struct {
	Name               string            `json:"name" binding:"required"`
	Provider           string            `json:"provider" binding:"required"`
	Endpoint           string            `json:"endpoint" binding:"required"`
	APIKey             *string           `json:"api_key"`
	IntervalSeconds    int               `json:"interval_seconds"`
	Enabled            bool              `json:"enabled"`
	PrimaryModelID     string            `json:"primary_model_id" binding:"required"`
	AdditionalModelIDs []string          `json:"additional_model_ids"`
	TemplateID         *int64            `json:"template_id"`
	ExtraHeaders       map[string]string `json:"extra_headers"`
	BodyOverrideMode   string            `json:"body_override_mode"`
	BodyOverride       map[string]any    `json:"body_override"`
}

// Create creates a monitor.
// POST /api/v1/admin/channel-monitors
func (h *ChannelMonitorHandler) Create(c *gin.Context) {
	var req createChannelMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	m, err := h.monitorService.Create(c.Request.Context(), &service.ChannelMonitor{
		Name:               req.Name,
		Provider:           req.Provider,
		Endpoint:           req.Endpoint,
		IntervalSeconds:    req.IntervalSeconds,
		Enabled:            req.Enabled,
		PrimaryModelID:     req.PrimaryModelID,
		AdditionalModelIDs: req.AdditionalModelIDs,
		TemplateID:         req.TemplateID,
		ExtraHeaders:       req.ExtraHeaders,
		BodyOverrideMode:   req.BodyOverrideMode,
		BodyOverride:       req.BodyOverride,
	}, req.APIKey)
	if err != nil {
		logger.FromContext(c.Request.Context()).Warn(
			"channel_monitor_create_failed",
			zap.String("component", "handler.admin.channel_monitor"),
			zap.String("provider", req.Provider),
			zap.Bool("enabled", req.Enabled),
			zap.Error(err),
		)
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toAdminChannelMonitorView(h.monitorService, m))
}

type updateChannelMonitorRequest struct {
	Name               *string            `json:"name"`
	Provider           *string            `json:"provider"`
	Endpoint           *string            `json:"endpoint"`
	APIKey             *string            `json:"api_key"`
	IntervalSeconds    *int               `json:"interval_seconds"`
	Enabled            *bool              `json:"enabled"`
	PrimaryModelID     *string            `json:"primary_model_id"`
	AdditionalModelIDs *[]string          `json:"additional_model_ids"`
	TemplateID         **int64            `json:"template_id"`
	ExtraHeaders       *map[string]string `json:"extra_headers"`
	BodyOverrideMode   *string            `json:"body_override_mode"`
	BodyOverride       *map[string]any    `json:"body_override"`
}

// Update updates a monitor.
// PUT /api/v1/admin/channel-monitors/:id
func (h *ChannelMonitorHandler) Update(c *gin.Context) {
	monitorID, ok := parseChannelMonitorID(c)
	if !ok {
		return
	}
	var req updateChannelMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	existing, err := h.monitorService.GetByID(c.Request.Context(), monitorID)
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
	if req.Endpoint != nil {
		existing.Endpoint = *req.Endpoint
	}
	if req.IntervalSeconds != nil {
		existing.IntervalSeconds = *req.IntervalSeconds
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.PrimaryModelID != nil {
		existing.PrimaryModelID = *req.PrimaryModelID
	}
	if req.AdditionalModelIDs != nil {
		existing.AdditionalModelIDs = *req.AdditionalModelIDs
	}
	if req.TemplateID != nil {
		existing.TemplateID = *req.TemplateID
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

	updated, err := h.monitorService.Update(c.Request.Context(), existing, req.APIKey)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, toAdminChannelMonitorView(h.monitorService, updated))
}

// Delete deletes a monitor.
// DELETE /api/v1/admin/channel-monitors/:id
func (h *ChannelMonitorHandler) Delete(c *gin.Context) {
	monitorID, ok := parseChannelMonitorID(c)
	if !ok {
		return
	}
	if err := h.monitorService.Delete(c.Request.Context(), monitorID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Channel monitor deleted successfully"})
}

// Run triggers an immediate check and stores histories.
// POST /api/v1/admin/channel-monitors/:id/run
func (h *ChannelMonitorHandler) Run(c *gin.Context) {
	monitorID, ok := parseChannelMonitorID(c)
	if !ok {
		return
	}
	results, err := h.monitorService.RunCheckNow(c.Request.Context(), monitorID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, results)
}

// ListHistories returns the latest histories.
// GET /api/v1/admin/channel-monitors/:id/histories
func (h *ChannelMonitorHandler) ListHistories(c *gin.Context) {
	monitorID, ok := parseChannelMonitorID(c)
	if !ok {
		return
	}
	limit := 50
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}
	items, err := h.monitorService.ListHistory(c.Request.Context(), monitorID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func parseChannelMonitorID(c *gin.Context) (int64, bool) {
	monitorID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || monitorID <= 0 {
		response.BadRequest(c, "Invalid monitor ID")
		return 0, false
	}
	return monitorID, true
}

func toAdminChannelMonitorView(svc *service.ChannelMonitorService, m *service.ChannelMonitor) *adminChannelMonitorView {
	if m == nil {
		return nil
	}
	view := &adminChannelMonitorView{
		ID:                 m.ID,
		Name:               m.Name,
		Provider:           m.Provider,
		Endpoint:           m.Endpoint,
		IntervalSeconds:    m.IntervalSeconds,
		Enabled:            m.Enabled,
		PrimaryModelID:     m.PrimaryModelID,
		AdditionalModelIDs: m.AdditionalModelIDs,
		TemplateID:         m.TemplateID,
		ExtraHeaders:       m.ExtraHeaders,
		BodyOverrideMode:   m.BodyOverrideMode,
		BodyOverride:       m.BodyOverride,
		LastRunAt:          m.LastRunAt,
		NextRunAt:          m.NextRunAt,
		APIKeyConfigured:   m.APIKeyEncrypted != nil && *m.APIKeyEncrypted != "",
	}
	if svc != nil {
		view.APIKeyDecryptFailed = svc.IsAPIKeyDecryptFailed(m)
	}
	return view
}
