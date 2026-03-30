package handler

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (h *GatewayHandler) Models(c *gin.Context) {
	apiKey, _ := middleware2.GetAPIKeyFromContext(c)
	var platform string
	if apiKey != nil && apiKey.Group != nil {
		platform = apiKey.Group.Platform
	}
	if forcedPlatform, ok := middleware2.GetForcePlatformFromContext(c); ok && strings.TrimSpace(forcedPlatform) != "" {
		platform = forcedPlatform
	}
	publicEntries, err := h.gatewayService.GetAPIKeyPublicModels(c.Request.Context(), apiKey, platform)
	if err != nil {
		status := http.StatusBadGateway
		if appErr := infraerrors.FromError(err); appErr != nil {
			status = int(appErr.Code)
		}
		h.errorResponse(c, status, "upstream_error", err.Error())
		return
	}
	if len(publicEntries) > 0 {
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": apiKeyPublicEntriesToClaudeModels(publicEntries)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": []claude.Model{}})
}
func (h *GatewayHandler) AntigravityModels(c *gin.Context) {
	entries := h.registryEntriesForPlatform(c.Request.Context(), service.PlatformAntigravity)
	if len(entries) > 0 {
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": registryEntriesToClaudeModels(entries)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": antigravity.DefaultModels()})
}
func (h *GatewayHandler) registryEntriesForPlatform(ctx context.Context, platform string) []modelregistry.ModelEntry {
	if h.modelRegistryService == nil || strings.TrimSpace(platform) == "" {
		return nil
	}
	entries, err := h.modelRegistryService.GetModelsByPlatform(ctx, platform, "runtime", "whitelist")
	if err != nil {
		return nil
	}
	return entries
}
func registryEntriesToClaudeModels(entries []modelregistry.ModelEntry) []claude.Model {
	models := make([]claude.Model, 0, len(entries))
	for _, entry := range entries {
		displayName := entry.DisplayName
		if displayName == "" {
			displayName = entry.ID
		}
		models = append(models, claude.Model{ID: entry.ID, Type: "model", DisplayName: displayName, CreatedAt: ""})
	}
	return models
}

func apiKeyPublicEntriesToClaudeModels(entries []service.APIKeyPublicModelEntry) []claude.Model {
	models := make([]claude.Model, 0, len(entries))
	for _, entry := range entries {
		displayName := entry.DisplayName
		if displayName == "" {
			displayName = entry.PublicID
		}
		models = append(models, claude.Model{
			ID:          entry.PublicID,
			Type:        "model",
			DisplayName: displayName,
			CreatedAt:   "",
		})
	}
	return models
}

func apiKeyPublicEntriesToGeminiModels(entries []service.APIKeyPublicModelEntry) gemini.ModelsListResponse {
	models := make([]gemini.Model, 0, len(entries))
	for _, entry := range entries {
		models = append(models, apiKeyPublicEntryToGeminiModel(entry))
	}
	return gemini.ModelsListResponse{Models: models}
}

func apiKeyPublicEntryToGeminiModel(entry service.APIKeyPublicModelEntry) gemini.Model {
	displayName := entry.DisplayName
	if displayName == "" {
		displayName = entry.PublicID
	}
	return gemini.Model{
		Name:                       "models/" + entry.PublicID,
		DisplayName:                displayName,
		SupportedGenerationMethods: []string{"generateContent", "streamGenerateContent", "countTokens"},
	}
}

func (h *GatewayHandler) resolveParsedRequestModel(ctx context.Context, parsed *service.ParsedRequest) {
	if parsed == nil {
		return
	}
	if parsed.RawModel == "" {
		parsed.RawModel = parsed.Model
	}
	if h.modelRegistryService == nil {
		return
	}
	resolution, err := h.modelRegistryService.ExplainResolution(ctx, parsed.RawModel)
	if err != nil || resolution == nil {
		return
	}
	if resolution.EffectiveID != "" {
		parsed.Model = resolution.EffectiveID
		return
	}
	if resolution.CanonicalID != "" {
		parsed.Model = resolution.CanonicalID
	}
}
