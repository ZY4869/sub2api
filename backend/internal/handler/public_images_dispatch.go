package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) DispatchPublicImagesGeneration(c *gin.Context) {
	if h == nil || h.Gateway == nil || h.OpenAIGateway == nil || h.GrokGateway == nil {
		return
	}
	decision, ok := h.Gateway.ResolvePublicImageRoute(c, EndpointImagesGen)
	if !ok {
		return
	}
	switch decision.ResolvedProvider {
	case service.PlatformGrok:
		h.GrokGateway.ImagesGeneration(c)
	case service.PlatformGemini:
		h.Gateway.GeminiOpenAICompatImagesGeneration(c)
	default:
		h.OpenAIGateway.ImagesGeneration(c)
	}
}

func (h *Handlers) DispatchPublicImagesEdits(c *gin.Context) {
	if h == nil || h.Gateway == nil || h.OpenAIGateway == nil || h.GrokGateway == nil {
		return
	}
	decision, ok := h.Gateway.ResolvePublicImageRoute(c, EndpointImagesEdits)
	if !ok {
		return
	}
	switch decision.ResolvedProvider {
	case service.PlatformGrok:
		h.GrokGateway.ImagesEdits(c)
	default:
		h.OpenAIGateway.ImagesEdits(c)
	}
}
