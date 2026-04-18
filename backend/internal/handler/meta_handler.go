package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MetaHandler struct {
	modelCatalogService  *service.ModelCatalogService
	modelRegistryService *service.ModelRegistryService
}

func NewMetaHandler(modelCatalogService *service.ModelCatalogService) *MetaHandler {
	return &MetaHandler{modelCatalogService: modelCatalogService}
}

func (h *MetaHandler) SetModelRegistryService(modelRegistryService *service.ModelRegistryService) {
	h.modelRegistryService = modelRegistryService
}

func (h *MetaHandler) USDCNYExchangeRate(c *gin.Context) {
	force := false
	if raw := strings.TrimSpace(c.Query("force")); raw != "" {
		force = raw == "1" || strings.EqualFold(raw, "true")
	}
	rate, err := h.modelCatalogService.GetUSDCNYExchangeRate(c.Request.Context(), force)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, rate)
}

func (h *MetaHandler) ModelRegistry(c *gin.Context) {
	if h.modelRegistryService == nil {
		response.InternalError(c, "model registry service unavailable")
		return
	}
	snapshot, err := h.modelRegistryService.PublicSnapshot(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if snapshot.ETag != "" {
		c.Header("ETag", snapshot.ETag)
		c.Header("Vary", "If-None-Match")
		if strings.TrimSpace(c.GetHeader("If-None-Match")) == snapshot.ETag {
			c.Status(http.StatusNotModified)
			return
		}
	}
	response.Success(c, snapshot)
}

func (h *MetaHandler) ModelCatalog(c *gin.Context) {
	snapshot, err := h.modelCatalogService.PublicModelCatalogSnapshot(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	etagHit := false
	if snapshot.ETag != "" {
		c.Header("ETag", snapshot.ETag)
		c.Header("Vary", "If-None-Match")
		if strings.TrimSpace(c.GetHeader("If-None-Match")) == snapshot.ETag {
			etagHit = true
			logger.FromContext(c.Request.Context()).Info(
				"public model catalog responded from etag cache",
				zap.String("component", "handler.meta"),
				zap.Bool("etag_hit", true),
				zap.Int("model_count", len(snapshot.Items)),
			)
			c.Status(http.StatusNotModified)
			return
		}
	}
	logger.FromContext(c.Request.Context()).Info(
		"public model catalog responded",
		zap.String("component", "handler.meta"),
		zap.Bool("etag_hit", etagHit),
		zap.Int("model_count", len(snapshot.Items)),
	)
	response.Success(c, snapshot)
}
