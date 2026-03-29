package handler

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (h *GatewayHandler) Models(c *gin.Context) {
	apiKey, _ := middleware2.GetAPIKeyFromContext(c)
	var groupID *int64
	var platform string
	if apiKey != nil && apiKey.Group != nil {
		groupID = &apiKey.Group.ID
		platform = apiKey.Group.Platform
	}
	if forcedPlatform, ok := middleware2.GetForcePlatformFromContext(c); ok && strings.TrimSpace(forcedPlatform) != "" {
		platform = forcedPlatform
	}
	if platform == service.PlatformSora {
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": service.DefaultSoraModels(h.cfg)})
		return
	}
	if platform == service.PlatformGrok {
		if entries := h.registryEntriesForPlatform(c.Request.Context(), platform); len(entries) > 0 {
			c.JSON(http.StatusOK, gin.H{"object": "list", "data": registryEntriesToClaudeModels(entries)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": []claude.Model{
			{ID: "grok-4", Type: "model", DisplayName: "grok-4", CreatedAt: ""},
			{ID: "grok-3-beta", Type: "model", DisplayName: "grok-3-beta", CreatedAt: ""},
			{ID: "grok-imagine-image", Type: "model", DisplayName: "grok-imagine-image", CreatedAt: ""},
			{ID: "grok-imagine-video", Type: "model", DisplayName: "grok-imagine-video", CreatedAt: ""},
			{ID: "grok-2-image", Type: "model", DisplayName: "grok-2-image", CreatedAt: ""},
		}})
		return
	}
	if publicEntries := h.gatewayService.GetAPIKeyPublicModels(c.Request.Context(), apiKey, platform); len(publicEntries) > 0 {
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": apiKeyPublicEntriesToClaudeModels(publicEntries)})
		return
	}
	availableModels := h.gatewayService.GetAvailableModels(c.Request.Context(), groupID, "")
	defaultEntries := h.registryEntriesForPlatform(c.Request.Context(), platform)
	if len(availableModels) > 0 {
		if len(defaultEntries) > 0 {
			c.JSON(http.StatusOK, gin.H{"object": "list", "data": filterGatewayModels(defaultEntries, availableModels)})
			return
		}
		models := make([]claude.Model, 0, len(availableModels))
		for _, modelID := range availableModels {
			models = append(models, claude.Model{ID: modelID, Type: "model", DisplayName: modelID, CreatedAt: ""})
		}
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": models})
		return
	}
	if len(defaultEntries) > 0 {
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": registryEntriesToClaudeModels(defaultEntries)})
		return
	}
	if platform == service.PlatformOpenAI || platform == service.PlatformCopilot {
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": openai.DefaultModels})
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": claude.DefaultModels})
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
		SupportedGenerationMethods: []string{"generateContent", "streamGenerateContent"},
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

func filterGatewayModels(entries []modelregistry.ModelEntry, allowed []string) []claude.Model {
	if len(allowed) == 0 {
		return registryEntriesToClaudeModels(entries)
	}
	indexed := make(map[string]modelregistry.ModelEntry, len(entries))
	for _, entry := range entries {
		indexed[entry.ID] = entry
	}
	models := make([]claude.Model, 0, len(allowed))
	for _, item := range allowed {
		entry, ok := indexed[strings.TrimSpace(item)]
		if !ok {
			models = append(models, claude.Model{ID: item, Type: "model", DisplayName: item, CreatedAt: ""})
			continue
		}
		displayName := entry.DisplayName
		if displayName == "" {
			displayName = entry.ID
		}
		models = append(models, claude.Model{ID: entry.ID, Type: "model", DisplayName: displayName, CreatedAt: ""})
	}
	return models
}
