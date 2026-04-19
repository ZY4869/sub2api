package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterDocumentAIRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	settingService *service.SettingService,
	cfg *config.Config,
) {
	if h == nil || h.DocumentAI == nil {
		return
	}
	bodyLimit := middleware.RequestBodyLimit(cfg.Gateway.MaxBodySize)
	clientRequestID := middleware.ClientRequestID()
	documentAI := r.Group("/document-ai/v1")
	documentAI.Use(bodyLimit)
	documentAI.Use(clientRequestID)
	documentAI.Use(gin.HandlerFunc(apiKeyAuth))
	documentAI.Use(middleware.MaintenanceModeGatewayGuard(settingService, "document_ai", middleware.JSONServiceUnavailableWriter))
	{
		documentAI.GET("/models", h.DocumentAI.ListModels)
		documentAI.POST("/jobs", h.DocumentAI.CreateJob)
		documentAI.GET("/jobs/:job_id", h.DocumentAI.GetJob)
		documentAI.GET("/jobs/:job_id/result", h.DocumentAI.GetJobResult)
		documentAI.POST("/models/*modelAction", h.DocumentAI.Parse)
	}
}
