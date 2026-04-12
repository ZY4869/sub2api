package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) GeminiV1BetaCachedContents(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGeminiPassthrough(c, service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiCachedContent})
}

func (h *GatewayHandler) GeminiV1BetaFileSearchStores(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGeminiPassthrough(c, service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiFileSearchStore})
}

func (h *GatewayHandler) GeminiV1BetaDocuments(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGeminiPassthrough(c, service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiDocument})
}

func (h *GatewayHandler) GeminiV1BetaOperations(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGeminiPassthrough(c, service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiOperation})
}

func (h *GatewayHandler) GeminiV1BetaInteractions(c *gin.Context) {
	if h.cfg == nil || !h.cfg.Gateway.GeminiInteractionsEnabled {
		googleErrorWithReason(c, http.StatusNotFound, service.GatewayReasonPublicEndpointUnsupported, "gateway.gemini.interactions_disabled", "Gemini Interactions API is disabled")
		return
	}
	attachGeminiPublicProtocolContext(c)
	h.forwardGeminiPassthrough(c, service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiInteraction})
}

func (h *GatewayHandler) GeminiV1BetaOpenAICompat(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	resourceKind := ""
	switch {
	case strings.Contains(strings.ToLower(strings.TrimSpace(c.Request.URL.Path)), "/openai/files"):
		resourceKind = service.UpstreamResourceKindGeminiFile
	case strings.Contains(strings.ToLower(strings.TrimSpace(c.Request.URL.Path)), "/openai/batches"):
		resourceKind = service.UpstreamResourceKindGeminiBatch
	}
	h.forwardGeminiPassthrough(c, service.GeminiPublicPassthroughInput{ResourceKind: resourceKind})
}

func (h *GatewayHandler) GeminiV1BetaLive(c *gin.Context) {
	if h.cfg == nil || !h.cfg.Gateway.GeminiLiveEnabled {
		googleErrorWithReason(c, http.StatusNotFound, service.GatewayReasonPublicEndpointUnsupported, "gateway.gemini.live_disabled", "Gemini Live API is disabled")
		return
	}
	attachGeminiPublicProtocolContext(c)
	if isOpenAIWSUpgradeRequest(c.Request) {
		h.forwardGeminiLiveWebSocket(c)
		return
	}
	if strings.EqualFold(c.Request.Method, http.MethodPost) && geminiLiveAuthTokenProxyRequested(c.Request.URL.Path) {
		h.forwardGeminiPassthrough(c, service.GeminiPublicPassthroughInput{UpstreamPath: service.GeminiLiveAuthTokensPath})
		return
	}
	googleErrorKey(c, http.StatusUpgradeRequired, "gateway.gemini.live_upgrade_required", "WebSocket upgrade required for Gemini Live")
}

func (h *GatewayHandler) GeminiV1BetaEmbeddings(c *gin.Context, modelName string) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGeminiPassthrough(c, service.GeminiPublicPassthroughInput{
		RequestedModel: strings.TrimSpace(modelName),
	})
}
