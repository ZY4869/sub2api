package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ComplianceHandler struct {
	settingService *service.SettingService
}

func NewComplianceHandler(settingService *service.SettingService) *ComplianceHandler {
	return &ComplianceHandler{settingService: settingService}
}

func (h *ComplianceHandler) Status(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.InternalError(c, "Service not configured")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	status, err := h.settingService.GetAdminComplianceStatus(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

func (h *ComplianceHandler) Acknowledge(c *gin.Context) {
	if h == nil || h.settingService == nil {
		response.InternalError(c, "Service not configured")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	status, err := h.settingService.AcknowledgeAdminCompliance(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	logger.With(
		zap.String("component", "audit.admin.compliance"),
		zap.Int64("admin_user_id", subject.UserID),
		zap.String("document_version", status.DocumentVersion),
		zap.String("document_hash", status.DocumentHash),
	).Info("admin compliance acknowledged")
	response.Success(c, status)
}
