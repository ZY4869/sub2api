package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AuditLogHandler struct {
	service       *service.AuditLogService
	adminSecurity *AdminSecurityHelper
}

func NewAuditLogHandler(auditLogService *service.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{service: auditLogService}
}

func (h *AuditLogHandler) SetAdminSecurityHelper(helper *AdminSecurityHelper) {
	if h == nil {
		return
	}
	h.adminSecurity = helper
}

func (h *AuditLogHandler) List(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "Audit log service unavailable")
		return
	}
	page, pageSize := response.ParsePagination(c)
	filter := &service.AuditLogFilter{
		Page:       page,
		PageSize:   pageSize,
		Action:     strings.TrimSpace(c.Query("action")),
		TargetType: strings.TrimSpace(c.Query("target_type")),
		TargetID:   strings.TrimSpace(c.Query("target_id")),
		Status:     strings.TrimSpace(c.Query("status")),
		RequestID:  strings.TrimSpace(c.Query("request_id")),
	}
	if raw := strings.TrimSpace(c.Query("actor_user_id")); raw != "" {
		actorID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || actorID <= 0 {
			response.BadRequest(c, "Invalid actor_user_id")
			return
		}
		filter.ActorUserID = &actorID
	}

	result, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, result.Items, result.Total, result.Page, result.PageSize)
}

func (h *AuditLogHandler) CleanupExpired(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "Audit log service unavailable")
		return
	}
	if !h.requireStepUp(c, "admin.audit_logs.cleanup_expired") {
		return
	}
	deleted, cutoff, err := h.service.CleanupExpired(c.Request.Context())
	if err != nil {
		h.recordAudit(c, "cleanup_expired", "audit_log", "", service.AuditStatusFailure, map[string]any{
			"reason": "cleanup_failed",
		})
		response.ErrorFrom(c, err)
		return
	}
	h.recordAudit(c, "cleanup_expired", "audit_log", "", service.AuditStatusSuccess, map[string]any{
		"deleted": deleted,
		"cutoff":  cutoff.UTC(),
	})
	response.Success(c, gin.H{
		"deleted": deleted,
		"cutoff":  cutoff,
	})
}

func (h *AuditLogHandler) requireStepUp(c *gin.Context, scope string) bool {
	if h != nil && h.adminSecurity != nil {
		return h.adminSecurity.RequireStepUpTotp(c, scope)
	}
	return (*AdminSecurityHelper)(nil).RequireStepUpTotp(c, scope)
}

func (h *AuditLogHandler) recordAudit(c *gin.Context, action string, targetType string, targetID string, status string, metadata map[string]any) {
	if h == nil || h.adminSecurity == nil {
		return
	}
	h.adminSecurity.RecordAudit(c, action, targetType, targetID, status, metadata)
}
