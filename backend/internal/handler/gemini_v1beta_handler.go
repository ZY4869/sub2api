package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GeminiV1BetaListModels proxies:
// GET /v1beta/models
func (h *GatewayHandler) GeminiV1BetaListModels(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		googleErrorKey(c, http.StatusUnauthorized, "gateway.gemini.invalid_api_key", "Invalid API key")
		return
	}
	// 检查平台：优先使用强制平台（/antigravity 路由），否则要求 gemini 分组
	forcePlatform, hasForcePlatform := middleware.GetForcePlatformFromContext(c)
	if forcePlatform == service.PlatformKiro || service.IsUnsupportedRuntimePlatform(forcePlatform) {
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.unsupported_platform", "Gemini protocol is not supported for this platform")
		return
	}
	effectivePlatform := service.PlatformGemini
	if hasForcePlatform && strings.TrimSpace(forcePlatform) != "" {
		effectivePlatform = forcePlatform
	} else if apiKey.Group != nil && strings.TrimSpace(apiKey.Group.Platform) != "" {
		effectivePlatform = apiKey.Group.Platform
	}
	publicEntries, err := h.gatewayService.GetAPIKeyPublicModels(c.Request.Context(), apiKey, effectivePlatform)
	if err != nil {
		googleErrorFromServiceError(c, err)
		return
	}
	if len(publicEntries) == 0 {
		applyGeminiModelMetadataSource(c, geminiModelMetadataSourceProjectedEmpty)
		c.JSON(http.StatusOK, gemini.ModelsListResponse{Models: []gemini.Model{}})
		return
	}
	pagedEntries, nextPageToken, err := paginateGeminiPublicModels(publicEntries, c.Query("pageSize"), c.Query("pageToken"))
	if err != nil {
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.invalid_pagination", "Invalid Gemini models pagination parameters")
		return
	}
	applyGeminiModelMetadataSource(c, geminiModelMetadataSourceProjectedEmpty)
	c.JSON(http.StatusOK, apiKeyPublicEntriesToGeminiModelsWithRegistry(pagedEntries, nextPageToken, h.modelRegistryService))
}

// GeminiV1BetaGetModel proxies:
// GET /v1beta/models/{model}
func (h *GatewayHandler) GeminiV1BetaGetModel(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		googleErrorKey(c, http.StatusUnauthorized, "gateway.gemini.invalid_api_key", "Invalid API key")
		return
	}
	// 检查平台：优先使用强制平台（/antigravity 路由），否则要求 gemini 分组
	forcePlatform, hasForcePlatform := middleware.GetForcePlatformFromContext(c)
	if forcePlatform == service.PlatformKiro || service.IsUnsupportedRuntimePlatform(forcePlatform) {
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.unsupported_platform", "Gemini protocol is not supported for this platform")
		return
	}
	modelPath := strings.Trim(strings.TrimSpace(firstNonEmptyString(c.Param("model"), c.Param("modelPath"))), "/")
	if modelPath == "" {
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.missing_model", "Missing model in URL")
		return
	}
	if isGeminiModelOperationsPath(modelPath) {
		h.GeminiV1BetaModelOperations(c)
		return
	}
	modelName := modelPath
	effectivePlatform := service.PlatformGemini
	if hasForcePlatform && strings.TrimSpace(forcePlatform) != "" {
		effectivePlatform = forcePlatform
	} else if apiKey.Group != nil && strings.TrimSpace(apiKey.Group.Platform) != "" {
		effectivePlatform = apiKey.Group.Platform
	}
	publicEntry, ok, err := h.gatewayService.FindAPIKeyPublicModel(c.Request.Context(), apiKey, effectivePlatform, modelName)
	if err != nil {
		googleErrorFromServiceError(c, err)
		return
	}
	if ok {
		applyGeminiModelMetadataSource(c, geminiModelMetadataSourceProjectedEmpty)
		c.JSON(http.StatusOK, apiKeyPublicEntryToGeminiModelWithRegistry(*publicEntry, h.modelRegistryService))
		return
	}
	googleErrorKey(c, http.StatusNotFound, "gateway.gemini.model_not_found", "Model not found")
}
