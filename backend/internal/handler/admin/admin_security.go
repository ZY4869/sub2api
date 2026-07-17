package admin

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const adminStepUpTotpHeader = "X-Sub2API-Step-Up-TOTP"

type AdminSecurityHelper struct {
	totpService     *service.TotpService
	auditLogService *service.AuditLogService
	stepUpVerifier  func(c *gin.Context, scope string) bool
}

func NewAdminSecurityHelper(totpService *service.TotpService, auditLogService *service.AuditLogService) *AdminSecurityHelper {
	return &AdminSecurityHelper{
		totpService:     totpService,
		auditLogService: auditLogService,
	}
}

func (h *AdminSecurityHelper) RequireStepUpTotp(c *gin.Context, scope string) bool {
	if h != nil && h.stepUpVerifier != nil {
		return h.stepUpVerifier(c, scope)
	}
	if h == nil || h.totpService == nil {
		h.RecordAudit(c, scope, "", "", service.AuditStatusDenied, map[string]any{"reason": "step_up_not_configured"})
		response.ErrorWithDetails(c, http.StatusForbidden, "Step-up verification is not configured", "STEP_UP_NOT_CONFIGURED", map[string]string{
			"method": "totp",
			"scope":  scope,
		})
		return false
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		h.RecordAudit(c, scope, "", "", service.AuditStatusDenied, map[string]any{"reason": "missing_auth_subject"})
		response.Unauthorized(c, "Authentication required")
		return false
	}

	enabled, err := h.totpService.IsTotpEnabledForUser(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return false
	}
	if !enabled {
		h.RecordAudit(c, scope, "", "", service.AuditStatusDenied, map[string]any{"reason": "totp_not_enabled"})
		response.ErrorWithDetails(c, http.StatusForbidden, "TOTP step-up verification is required for this operation", "STEP_UP_TOTP_NOT_ENABLED", map[string]string{
			"method": "totp",
			"scope":  scope,
		})
		return false
	}

	code := strings.TrimSpace(c.GetHeader(adminStepUpTotpHeader))
	if code == "" {
		h.RecordAudit(c, scope, "", "", service.AuditStatusDenied, map[string]any{"reason": "missing_step_up_code"})
		response.ErrorWithDetails(c, http.StatusForbidden, "TOTP step-up verification is required for this operation", "STEP_UP_REQUIRED", map[string]string{
			"method": "totp",
			"scope":  scope,
		})
		return false
	}
	if err := h.totpService.VerifyCode(c.Request.Context(), subject.UserID, code); err != nil {
		h.RecordAudit(c, scope, "", "", service.AuditStatusDenied, map[string]any{"reason": "invalid_step_up_code"})
		response.ErrorFrom(c, err)
		return false
	}

	logger.With(
		zap.String("component", "audit.admin.step_up"),
		zap.Int64("operator_user_id", subject.UserID),
		zap.String("scope", scope),
	).Info("admin step-up verification accepted")
	h.RecordAudit(c, scope, "", "", service.AuditStatusSuccess, map[string]any{"phase": "step_up"})
	return true
}

func (h *AdminSecurityHelper) RecordAudit(c *gin.Context, action string, targetType string, targetID string, status string, metadata map[string]any) {
	if h == nil || h.auditLogService == nil || c == nil {
		return
	}
	var actorID *int64
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		id := subject.UserID
		actorID = &id
	}
	role, _ := middleware.GetUserRoleFromContext(c)
	requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string)
	if strings.TrimSpace(requestID) == "" {
		requestID, _ = c.Request.Context().Value(ctxkey.ClientRequestID).(string)
	}
	_ = h.auditLogService.Record(c.Request.Context(), &service.AuditLog{
		ActorUserID: actorID,
		ActorRole:   role,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Status:      status,
		RequestID:   requestID,
		ClientIP:    ip.GetTrustedClientIP(c),
		UserAgent:   c.GetHeader("User-Agent"),
		Metadata:    metadata,
	})
}
