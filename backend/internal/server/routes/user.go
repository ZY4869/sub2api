package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers authenticated user routes.
func RegisterUserRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	settingService *service.SettingService,
) {
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.MaintenanceModeUserGuard(settingService))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	{
		user := authenticated.Group("/user")
		{
			user.GET("/profile", h.User.GetProfile)
			user.PUT("/password", h.User.ChangePassword)
			user.PUT("", h.User.UpdateProfile)
			user.GET("/aff", h.User.GetAffiliate)
			user.POST("/aff/transfer", h.User.TransferAffiliate)

			totp := user.Group("/totp")
			{
				totp.GET("/status", h.Totp.GetStatus)
				totp.GET("/verification-method", h.Totp.GetVerificationMethod)
				totp.POST("/send-code", h.Totp.SendVerifyCode)
				totp.POST("/setup", h.Totp.InitiateSetup)
				totp.POST("/enable", h.Totp.Enable)
				totp.POST("/disable", h.Totp.Disable)
			}
		}

		keys := authenticated.Group("/keys")
		{
			keys.GET("", h.APIKey.List)
			keys.GET("/:id", h.APIKey.GetByID)
			keys.POST("", h.APIKey.Create)
			keys.PUT("/:id", h.APIKey.Update)
			keys.DELETE("/:id", h.APIKey.Delete)
		}

		groups := authenticated.Group("/groups")
		{
			groups.GET("/available", h.APIKey.GetAvailableGroups)
			groups.GET("/model-catalog", h.APIKey.GetGroupModelCatalog)
			groups.GET("/model-options", h.APIKey.GetGroupModelOptions)
			groups.GET("/rates", h.APIKey.GetUserGroupRates)
		}

		channels := authenticated.Group("/channels")
		{
			channels.GET("/available", h.AvailableChannel.List)
		}

		channelMonitors := authenticated.Group("/channel-monitors")
		{
			channelMonitors.GET("", h.ChannelMonitor.ListUserView)
			channelMonitors.GET("/:id/status", h.ChannelMonitor.GetStatus)
		}

		usage := authenticated.Group("/usage")
		{
			usage.GET("", h.Usage.List)
			usage.GET("/filter-api-keys", h.Usage.FilterAPIKeys)
			usage.GET("/:id/request-preview", h.Usage.GetRequestPreview)
			usage.GET("/:id", h.Usage.GetByID)
			usage.GET("/stats", h.Usage.Stats)
			usage.GET("/dashboard/stats", h.Usage.DashboardStats)
			usage.GET("/dashboard/trend", h.Usage.DashboardTrend)
			usage.GET("/dashboard/models", h.Usage.DashboardModels)
			usage.POST("/dashboard/api-keys-usage", h.Usage.DashboardAPIKeysUsage)
		}

		announcements := authenticated.Group("/announcements")
		{
			announcements.GET("", h.Announcement.List)
			announcements.POST("/:id/read", h.Announcement.MarkRead)
		}

		docs := authenticated.Group("/docs")
		{
			docs.GET("/api", h.Docs.GetAPIReference)
		}

		redeem := authenticated.Group("/redeem")
		{
			redeem.POST("", h.Redeem.Redeem)
			redeem.GET("/history", h.Redeem.GetHistory)
		}

		subscriptions := authenticated.Group("/subscriptions")
		{
			subscriptions.GET("", h.Subscription.List)
			subscriptions.GET("/active", h.Subscription.GetActive)
			subscriptions.GET("/progress", h.Subscription.GetProgress)
			subscriptions.GET("/summary", h.Subscription.GetSummary)
		}
	}
}
