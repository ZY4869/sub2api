package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func AdminComplianceGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil || !settingService.IsAdminComplianceEnabled(c.Request.Context()) {
			c.Next()
			return
		}
		if isAdminComplianceBypassPath(c.FullPath(), c.Request.URL.Path) {
			c.Next()
			return
		}
		subject, ok := GetAuthSubjectFromContext(c)
		if !ok || subject.UserID <= 0 {
			AbortWithError(c, 401, "UNAUTHORIZED", "Authorization required")
			return
		}
		status, err := settingService.GetAdminComplianceStatus(c.Request.Context(), subject.UserID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if status.Required {
			response.ErrorWithDetails(c, 403, "Admin compliance acknowledgement required", "ADMIN_COMPLIANCE_REQUIRED", map[string]string{
				"document_version": status.DocumentVersion,
				"document_hash":    status.DocumentHash,
			})
			return
		}
		c.Next()
	}
}

func isAdminComplianceBypassPath(fullPath string, requestPath string) bool {
	path := strings.TrimSpace(fullPath)
	if path == "" {
		path = strings.TrimSpace(requestPath)
	}
	path = strings.ToLower(path)
	return path == "/api/v1/admin/compliance" ||
		strings.HasPrefix(path, "/api/v1/admin/compliance/")
}
