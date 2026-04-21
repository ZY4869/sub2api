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
	settingService       *service.SettingService
	authService          *service.AuthService
	userService          *service.UserService
	authResolver         func(*gin.Context) bool
}

func NewMetaHandler(modelCatalogService *service.ModelCatalogService) *MetaHandler {
	return &MetaHandler{modelCatalogService: modelCatalogService}
}

func (h *MetaHandler) SetModelRegistryService(modelRegistryService *service.ModelRegistryService) {
	h.modelRegistryService = modelRegistryService
}

func (h *MetaHandler) SetSettingService(settingService *service.SettingService) {
	h.settingService = settingService
}

func (h *MetaHandler) SetOptionalAuthServices(authService *service.AuthService, userService *service.UserService) {
	h.authService = authService
	h.userService = userService
}

func (h *MetaHandler) SetAuthResolverForTest(resolver func(*gin.Context) bool) {
	h.authResolver = resolver
}

func (h *MetaHandler) isModelCatalogAccessibleToGuest(c *gin.Context) bool {
	if h == nil || h.settingService == nil {
		return true
	}
	return h.settingService.IsPublicModelCatalogEnabled(c.Request.Context())
}

func (h *MetaHandler) isAuthenticatedModelCatalogRequest(c *gin.Context) bool {
	if h == nil || c == nil {
		return false
	}
	if h.authResolver != nil {
		return h.authResolver(c)
	}
	if h.authService == nil || h.userService == nil {
		return false
	}

	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader == "" {
		return false
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return false
	}
	tokenString := strings.TrimSpace(parts[1])
	if tokenString == "" {
		return false
	}

	claims, err := h.authService.ValidateToken(tokenString)
	if err != nil {
		return false
	}
	user, err := h.userService.GetByID(c.Request.Context(), claims.UserID)
	if err != nil || !user.IsActive() || claims.TokenVersion != user.TokenVersion {
		return false
	}
	return true
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
	guestAllowed := h.isModelCatalogAccessibleToGuest(c)
	authenticated := false
	if !guestAllowed {
		authenticated = h.isAuthenticatedModelCatalogRequest(c)
		if !authenticated {
			logger.FromContext(c.Request.Context()).Info(
				"public model catalog guest access blocked",
				zap.String("component", "handler.meta"),
				zap.Bool("public_model_catalog_enabled", guestAllowed),
				zap.Bool("has_authorization_header", strings.TrimSpace(c.GetHeader("Authorization")) != ""),
			)
			response.Unauthorized(c, "Authentication required")
			return
		}
	}
	snapshot, err := h.modelCatalogService.PublicModelCatalogSnapshot(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	etagHit := snapshot.ETag != "" && strings.TrimSpace(c.GetHeader("If-None-Match")) == snapshot.ETag
	if snapshot.ETag != "" {
		c.Header("ETag", snapshot.ETag)
		c.Header("Vary", "If-None-Match")
		if etagHit {
			logger.FromContext(c.Request.Context()).Info(
				"public model catalog responded from etag cache",
				zap.String("component", "handler.meta"),
				zap.Bool("etag_hit", true),
				zap.String("catalog_source", snapshot.CatalogSource),
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
		zap.Bool("guest_allowed", guestAllowed),
		zap.Bool("authenticated_request", authenticated),
		zap.String("catalog_source", snapshot.CatalogSource),
		zap.Int("model_count", len(snapshot.Items)),
	)
	response.Success(c, snapshot)
}

func (h *MetaHandler) ModelCatalogDetail(c *gin.Context) {
	guestAllowed := h.isModelCatalogAccessibleToGuest(c)
	authenticated := false
	if !guestAllowed {
		authenticated = h.isAuthenticatedModelCatalogRequest(c)
		if !authenticated {
			response.Unauthorized(c, "Authentication required")
			return
		}
	}

	modelID := strings.TrimSpace(c.Param("model"))
	detail, err := h.modelCatalogService.PublicModelCatalogDetail(c.Request.Context(), modelID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	logger.FromContext(c.Request.Context()).Info(
		"public model catalog detail responded",
		zap.String("component", "handler.meta"),
		zap.String("model", detail.Item.Model),
		zap.Bool("guest_allowed", guestAllowed),
		zap.Bool("authenticated_request", authenticated),
		zap.String("catalog_source", detail.CatalogSource),
		zap.String("example_source", detail.ExampleSource),
	)
	response.Success(c, detail)
}
