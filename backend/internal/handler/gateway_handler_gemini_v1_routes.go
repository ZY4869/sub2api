package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) GatewayV1ModelsList(c *gin.Context) {
	if h.gatewayV1UsesGeminiSurface(c) {
		h.GeminiV1BetaListModels(c)
		return
	}
	h.Models(c)
}

func (h *GatewayHandler) GatewayV1ModelsGet(c *gin.Context) {
	if h.gatewayV1UsesGeminiSurface(c) {
		h.GeminiV1BetaGetModel(c)
		return
	}
	h.errorResponse(c, http.StatusNotFound, "not_found_error", "Model detail is only supported on the Gemini /v1 surface")
}

func (h *GatewayHandler) GatewayV1ModelsAction(c *gin.Context) {
	if h.gatewayV1UsesGeminiSurface(c) {
		h.GeminiV1BetaModels(c)
		return
	}
	h.errorResponse(c, http.StatusNotFound, "not_found_error", "Gemini model actions are only supported for Gemini groups")
}

func (h *GatewayHandler) GeminiV1AlphaAuthTokens(c *gin.Context) {
	h.GeminiV1BetaLive(c)
}

func (h *GatewayHandler) gatewayV1UsesGeminiSurface(c *gin.Context) bool {
	if c == nil {
		return false
	}
	if forcePlatform, ok := middleware.GetForcePlatformFromContext(c); ok && strings.TrimSpace(forcePlatform) != "" {
		return forcePlatform == service.PlatformGemini
	}
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.Group == nil {
		return false
	}
	return strings.TrimSpace(apiKey.Group.Platform) == service.PlatformGemini
}
