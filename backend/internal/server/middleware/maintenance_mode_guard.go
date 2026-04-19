package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func MaintenanceModeUserGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil || !settingService.IsMaintenanceModeEnabled(c.Request.Context()) {
			c.Next()
			return
		}
		role, _ := GetUserRoleFromContext(c)
		if role == service.RoleAdmin {
			c.Next()
			return
		}
		AbortWithMaintenanceJSON(c, role, "json")
	}
}

func MaintenanceModeAuthGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil || !settingService.IsMaintenanceModeEnabled(c.Request.Context()) {
			c.Next()
			return
		}
		path := strings.TrimSpace(c.Request.URL.Path)
		allowedSuffixes := []string{"/auth/login", "/auth/login/2fa", "/auth/logout", "/auth/refresh"}
		for _, suffix := range allowedSuffixes {
			if strings.HasSuffix(path, suffix) {
				c.Next()
				return
			}
		}
		AbortWithMaintenanceJSON(c, "", "json")
	}
}

func MaintenanceModeGatewayGuard(settingService *service.SettingService, gatewayFamily string, writeError GatewayErrorWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if settingService == nil || !settingService.IsMaintenanceModeEnabled(c.Request.Context()) {
			c.Next()
			return
		}
		role, _ := GetUserRoleFromContext(c)
		if role == service.RoleAdmin {
			c.Next()
			return
		}
		AbortWithMaintenanceGateway(c, role, gatewayFamily, writeError)
	}
}
