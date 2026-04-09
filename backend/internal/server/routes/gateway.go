package routes

import (
	"log/slog"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterGatewayRoutes 注册 API 网关路由（Claude/OpenAI/Gemini 兼容）
func RegisterGatewayRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	cfg *config.Config,
) {
	bodyLimit := middleware.RequestBodyLimit(cfg.Gateway.MaxBodySize)
	clientRequestID := middleware.ClientRequestID()
	opsErrorLogger := handler.OpsErrorLoggerMiddleware(opsService)
	opsRequestTraceLogger := handler.OpsRequestTraceMiddleware(opsService)
	endpointNorm := handler.InboundEndpointMiddleware()

	// 未分组 Key 拦截中间件（按协议格式区分错误响应）
	requireGroupAnthropic := middleware.RequireGroupAssignment(settingService, middleware.AnthropicErrorWriter)
	requireGroupGoogle := middleware.RequireGroupAssignment(settingService, middleware.GoogleErrorWriter)
	dispatchers := newGatewayRouteDispatchers(h)

	// API网关（Claude API兼容）
	gateway := r.Group("/v1")
	gateway.Use(bodyLimit)
	gateway.Use(clientRequestID)
	gateway.Use(opsErrorLogger)
	gateway.Use(opsRequestTraceLogger)
	gateway.Use(endpointNorm)
	gateway.Use(gin.HandlerFunc(apiKeyAuth))
	gateway.Use(requireGroupAnthropic)
	{
		// /v1/messages: auto-route based on group platform
		gateway.POST("/messages", dispatchers.AnthropicMessages)
		// /v1/messages/count_tokens: OpenAI groups get 404
		gateway.POST("/messages/count_tokens", dispatchers.AnthropicCountTokens)
		gateway.GET("/models", h.Gateway.Models)
		gateway.GET("/usage", h.Gateway.Usage)
		gateway.POST("/responses", dispatchers.OpenAIResponses)
		gateway.POST("/responses/*subpath", dispatchers.OpenAIResponses)
		gateway.GET("/responses/*subpath", dispatchers.OpenAIResponses)
		gateway.DELETE("/responses/*subpath", dispatchers.OpenAIResponses)
		gateway.GET("/responses", dispatchers.OpenAIResponsesWebSocket)
		gateway.POST("/chat/completions", dispatchers.OpenAIChatCompletions)
		gateway.POST("/images/generations", dispatchers.GrokImagesGeneration)
		gateway.POST("/images/edits", dispatchers.GrokImagesEdits)
		gateway.POST("/videos", dispatchers.GrokVideosGeneration)
		gateway.POST("/videos/generations", dispatchers.GrokVideosGeneration)
		gateway.GET("/videos/:request_id", dispatchers.GrokVideosStatus)
	}

	// Gemini 原生 API 兼容层（Gemini SDK/CLI 直连）
	gemini := r.Group("/v1beta")
	grokV1 := r.Group("/grok/v1")
	grokV1.Use(bodyLimit)
	grokV1.Use(clientRequestID)
	grokV1.Use(opsErrorLogger)
	grokV1.Use(opsRequestTraceLogger)
	grokV1.Use(endpointNorm)
	grokV1.Use(middleware.ForcePlatform(service.PlatformGrok))
	grokV1.Use(gin.HandlerFunc(apiKeyAuth))
	grokV1.Use(requireGroupAnthropic)
	{
		grokV1.GET("/models", h.Gateway.Models)
		grokV1.POST("/chat/completions", dispatchers.OpenAIChatCompletions)
		grokV1.POST("/responses", dispatchers.OpenAIResponses)
		grokV1.POST("/responses/*subpath", dispatchers.OpenAIResponses)
		grokV1.GET("/responses/*subpath", dispatchers.OpenAIResponses)
		grokV1.DELETE("/responses/*subpath", dispatchers.OpenAIResponses)
		grokV1.POST("/images/generations", dispatchers.GrokImagesGeneration)
		grokV1.POST("/images/edits", dispatchers.GrokImagesEdits)
		grokV1.POST("/videos", dispatchers.GrokVideosGeneration)
		grokV1.POST("/videos/generations", dispatchers.GrokVideosGeneration)
		grokV1.GET("/videos/:request_id", dispatchers.GrokVideosStatus)
	}
	gemini.Use(bodyLimit)
	gemini.Use(clientRequestID)
	gemini.Use(opsErrorLogger)
	gemini.Use(opsRequestTraceLogger)
	gemini.Use(endpointNorm)
	gemini.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	gemini.Use(requireGroupGoogle)
	{
		gemini.GET("/models", h.Gateway.GeminiV1BetaListModels)
		gemini.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		// Gin treats ":" as a param marker, but Gemini uses "{model}:{action}" in the same segment.
		gemini.POST("/models/*modelAction", dispatchers.GeminiModels)
		gemini.GET("/files", dispatchers.GeminiFiles)
		gemini.POST("/files", dispatchers.GeminiFiles)
		gemini.POST("/files:action", dispatchers.GeminiFiles)
		gemini.GET("/files/*subpath", dispatchers.GeminiFiles)
		gemini.DELETE("/files/*subpath", dispatchers.GeminiFiles)
		gemini.GET("/batches", dispatchers.GeminiBatches)
		gemini.GET("/batches/*subpath", dispatchers.GeminiBatches)
		gemini.POST("/batches/*subpath", dispatchers.GeminiBatches)
		gemini.DELETE("/batches/*subpath", dispatchers.GeminiBatches)
	}
	r.POST("/upload/v1beta/files", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg), requireGroupGoogle, dispatchers.GeminiFilesUpload)
	r.GET("/download/v1beta/files/*subpath", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg), requireGroupGoogle, dispatchers.GeminiFilesDownload)
	googleBatchArchive := r.Group("/google/batch/archive/v1beta")
	googleBatchArchive.Use(bodyLimit)
	googleBatchArchive.Use(clientRequestID)
	googleBatchArchive.Use(opsErrorLogger)
	googleBatchArchive.Use(opsRequestTraceLogger)
	googleBatchArchive.Use(endpointNorm)
	googleBatchArchive.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	googleBatchArchive.Use(requireGroupGoogle)
	{
		googleBatchArchive.GET("/batches/*subpath", dispatchers.GoogleBatchArchiveBatch)
		googleBatchArchive.GET("/files/*subpath", dispatchers.GoogleBatchArchiveFileDownload)
	}
	vertexBatch := r.Group("/v1/projects/:project/locations/:location")
	vertexBatch.Use(bodyLimit)
	vertexBatch.Use(clientRequestID)
	vertexBatch.Use(opsErrorLogger)
	vertexBatch.Use(opsRequestTraceLogger)
	vertexBatch.Use(endpointNorm)
	vertexBatch.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	vertexBatch.Use(requireGroupGoogle)
	{
		vertexBatch.POST("/publishers/google/models/*modelAction", dispatchers.VertexModels)
		vertexBatch.GET("/batchPredictionJobs", dispatchers.VertexBatchPredictionJobs)
		vertexBatch.POST("/batchPredictionJobs", dispatchers.VertexBatchPredictionJobs)
		vertexBatch.GET("/batchPredictionJobs/*subpath", dispatchers.VertexBatchPredictionJobs)
		vertexBatch.POST("/batchPredictionJobs/*subpath", dispatchers.VertexBatchPredictionJobs)
		vertexBatch.DELETE("/batchPredictionJobs/*subpath", dispatchers.VertexBatchPredictionJobs)
	}

	// OpenAI Responses API（不带v1前缀的别名）
	r.POST("/responses", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.OpenAIResponses)
	r.POST("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.OpenAIResponses)
	r.GET("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.OpenAIResponses)
	r.DELETE("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.OpenAIResponses)
	r.GET("/responses", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.OpenAIResponsesWebSocket)
	// OpenAI Chat Completions API（不带v1前缀的别名）
	r.POST("/chat/completions", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.OpenAIChatCompletions)
	r.POST("/images/generations", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.GrokImagesGeneration)
	r.POST("/images/edits", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.GrokImagesEdits)
	r.POST("/videos", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.GrokVideosGeneration)
	r.POST("/videos/generations", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.GrokVideosGeneration)
	r.GET("/videos/:request_id", bodyLimit, clientRequestID, opsErrorLogger, opsRequestTraceLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, dispatchers.GrokVideosStatus)

	// Antigravity 模型列表
	r.GET("/antigravity/models", gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.AntigravityModels)

	// Antigravity 专用路由（仅使用 antigravity 账户，不混合调度）
	antigravityV1 := r.Group("/antigravity/v1")
	antigravityV1.Use(bodyLimit)
	antigravityV1.Use(clientRequestID)
	antigravityV1.Use(opsErrorLogger)
	antigravityV1.Use(opsRequestTraceLogger)
	antigravityV1.Use(endpointNorm)
	antigravityV1.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1.Use(gin.HandlerFunc(apiKeyAuth))
	antigravityV1.Use(requireGroupAnthropic)
	{
		antigravityV1.POST("/messages", dispatchers.AnthropicMessages)
		antigravityV1.POST("/messages/count_tokens", dispatchers.AnthropicCountTokens)
		antigravityV1.GET("/models", h.Gateway.AntigravityModels)
		antigravityV1.GET("/usage", h.Gateway.Usage)
	}

	antigravityV1Beta := r.Group("/antigravity/v1beta")
	antigravityV1Beta.Use(bodyLimit)
	antigravityV1Beta.Use(clientRequestID)
	antigravityV1Beta.Use(opsErrorLogger)
	antigravityV1Beta.Use(opsRequestTraceLogger)
	antigravityV1Beta.Use(endpointNorm)
	antigravityV1Beta.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1Beta.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	antigravityV1Beta.Use(requireGroupGoogle)
	{
		antigravityV1Beta.GET("/models", h.Gateway.GeminiV1BetaListModels)
		antigravityV1Beta.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		antigravityV1Beta.POST("/models/*modelAction", dispatchers.GeminiModels)
	}
}

// getGroupPlatform extracts the group platform from the API Key stored in context.
func getGroupPlatform(c *gin.Context) string {
	if forcedPlatform, ok := middleware.GetForcePlatformFromContext(c); ok && forcedPlatform != "" {
		return forcedPlatform
	}
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok {
		return ""
	}
	if apiKey.Group != nil && apiKey.Group.Platform != "" {
		if !service.IsOpenAIFamily(apiKey.Group.Platform) {
			return apiKey.Group.Platform
		}
	}
	groups := service.GroupsFromContext(c.Request.Context())
	var openAIPlatform string
	for _, group := range groups {
		if group == nil || group.Platform == "" {
			continue
		}
		if service.IsOpenAIFamily(group.Platform) {
			if openAIPlatform == "" {
				openAIPlatform = group.Platform
			}
			continue
		}
		return group.Platform
	}
	if apiKey.Group != nil && apiKey.Group.Platform != "" {
		return apiKey.Group.Platform
	}
	return openAIPlatform
}

func isGrokGroup(c *gin.Context) bool {
	return service.IsGrokPlatform(getGroupPlatform(c))
}

func dispatchMessagesRoute(c *gin.Context, nativeHandler gin.HandlerFunc, compatHandler gin.HandlerFunc) {
	decision := service.DecideProtocolCapability(getGroupPlatform(c), service.EndpointMessages, service.ProtocolCapabilityActionDefault)
	switch decision.Mode {
	case service.ProtocolCapabilityNativePassthrough:
		nativeHandler(c)
	case service.ProtocolCapabilityCompatTranslate:
		compatHandler(c)
	default:
		writeOpenAIGatewayCapabilityError(c, service.GrokMessagesUnsupportedDecision(), "invalid_request_error")
	}
}

func dispatchCountTokensRoute(c *gin.Context, nativeHandler gin.HandlerFunc) {
	decision := service.DecideProtocolCapability(getGroupPlatform(c), service.EndpointMessages, service.ProtocolCapabilityActionCountTokens)
	if decision.Supported && decision.Mode == service.ProtocolCapabilityNativePassthrough {
		nativeHandler(c)
		return
	}
	decision.MessageKey = "gateway.count_tokens.unsupported_platform"
	writeOpenAIGatewayCapabilityError(c, decision, "not_found_error")
}

func dispatchOpenAIRoute(c *gin.Context, inboundEndpoint string, action string, openAIHandler gin.HandlerFunc, grokHandler gin.HandlerFunc) {
	decision := service.DecideProtocolCapability(getGroupPlatform(c), inboundEndpoint, action)
	if !decision.Supported || decision.Mode != service.ProtocolCapabilityNativePassthrough {
		writeOpenAIGatewayCapabilityError(c, decision, "not_found_error")
		return
	}
	if isGrokGroup(c) {
		if grokHandler == nil {
			writeOpenAIGatewayCapabilityError(c, service.PublicEndpointUnsupportedDecision(inboundEndpoint, action), "not_found_error")
			return
		}
		grokHandler(c)
		return
	}
	openAIHandler(c)
}

func dispatchGrokOnlyRoute(c *gin.Context, inboundEndpoint string, grokHandler gin.HandlerFunc) {
	decision := service.DecideProtocolCapability(getGroupPlatform(c), inboundEndpoint, service.ProtocolCapabilityActionDefault)
	if !decision.Supported || decision.Mode != service.ProtocolCapabilityNativePassthrough || !isGrokGroup(c) {
		writeOpenAIGatewayCapabilityError(c, service.GrokAliasReservedDecision(inboundEndpoint), "not_found_error")
		return
	}
	grokHandler(c)
}

func writeOpenAIGatewayCapabilityError(c *gin.Context, decision service.ProtocolCapabilityDecision, errorType string) {
	logOpenAIGatewayCapabilityDecision(c, decision)
	messageKey := decision.MessageKey
	if strings.TrimSpace(messageKey) == "" {
		messageKey = "gateway.public_endpoint.unsupported_platform"
	}
	c.AbortWithStatusJSON(decision.StatusCode, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errorType,
			"message": response.LocalizedMessage(c, messageKey, "%s is not supported for this platform", decision.RequestFormat),
			"code":    decision.Reason,
			"reason":  decision.Reason,
		},
	})
}

func logOpenAIGatewayCapabilityDecision(c *gin.Context, decision service.ProtocolCapabilityDecision) {
	switch decision.Reason {
	case service.GatewayReasonUnsupportedAction:
		protocolruntime.RecordUnsupportedAction(decision.Reason)
		slog.Warn(
			"gateway_unsupported_action",
			"runtime_platform", getGroupPlatform(c),
			"inbound_endpoint", decision.RequestFormat,
			"reason", decision.InternalMismatchKind,
		)
	default:
		protocolruntime.RecordRouteMismatch(decision.InternalMismatchKind)
		slog.Warn(
			"gateway_route_mismatch",
			"runtime_platform", getGroupPlatform(c),
			"inbound_endpoint", decision.RequestFormat,
			"reason", decision.InternalMismatchKind,
		)
	}
}
