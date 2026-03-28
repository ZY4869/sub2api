package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
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
	soraMaxBodySize := cfg.Gateway.SoraMaxBodySize
	if soraMaxBodySize <= 0 {
		soraMaxBodySize = cfg.Gateway.MaxBodySize
	}
	soraBodyLimit := middleware.RequestBodyLimit(soraMaxBodySize)
	clientRequestID := middleware.ClientRequestID()
	opsErrorLogger := handler.OpsErrorLoggerMiddleware(opsService)
	endpointNorm := handler.InboundEndpointMiddleware()

	// 未分组 Key 拦截中间件（按协议格式区分错误响应）
	requireGroupAnthropic := middleware.RequireGroupAssignment(settingService, middleware.AnthropicErrorWriter)
	requireGroupGoogle := middleware.RequireGroupAssignment(settingService, middleware.GoogleErrorWriter)

	// API网关（Claude API兼容）
	gateway := r.Group("/v1")
	gateway.Use(bodyLimit)
	gateway.Use(clientRequestID)
	gateway.Use(opsErrorLogger)
	gateway.Use(endpointNorm)
	gateway.Use(gin.HandlerFunc(apiKeyAuth))
	gateway.Use(requireGroupAnthropic)
	{
		// /v1/messages: auto-route based on group platform
		gateway.POST("/messages", func(c *gin.Context) {
			if platform := getGroupPlatform(c); platform == service.PlatformGrok {
				writeGrokMessagesUnsupported(c)
				return
			} else if platform == service.PlatformOpenAI || platform == service.PlatformCopilot {
				h.OpenAIGateway.Messages(c)
				return
			}
			h.Gateway.Messages(c)
		})
		// /v1/messages/count_tokens: OpenAI groups get 404
		gateway.POST("/messages/count_tokens", func(c *gin.Context) {
			if platform := getGroupPlatform(c); platform == service.PlatformGrok {
				writeGrokMessagesUnsupported(c)
				return
			} else if platform == service.PlatformOpenAI || platform == service.PlatformCopilot {
				c.JSON(http.StatusNotFound, gin.H{
					"type": "error",
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Token counting is not supported for this platform",
					},
				})
				return
			}
			h.Gateway.CountTokens(c)
		})
		gateway.GET("/models", h.Gateway.Models)
		gateway.GET("/usage", h.Gateway.Usage)
		gateway.POST("/responses", func(c *gin.Context) {
			if isGrokGroup(c) {
				h.GrokGateway.Responses(c)
				return
			}
			h.OpenAIGateway.Responses(c)
		})
		gateway.POST("/responses/*subpath", func(c *gin.Context) {
			if isGrokGroup(c) {
				h.GrokGateway.Responses(c)
				return
			}
			h.OpenAIGateway.Responses(c)
		})
		gateway.GET("/responses/*subpath", func(c *gin.Context) {
			if isGrokGroup(c) {
				h.GrokGateway.Responses(c)
				return
			}
			h.OpenAIGateway.Responses(c)
		})
		gateway.DELETE("/responses/*subpath", func(c *gin.Context) {
			if isGrokGroup(c) {
				h.GrokGateway.Responses(c)
				return
			}
			h.OpenAIGateway.Responses(c)
		})
		gateway.GET("/responses", func(c *gin.Context) {
			h.OpenAIGateway.ResponsesWebSocket(c)
		})
		gateway.POST("/chat/completions", func(c *gin.Context) {
			if isGrokGroup(c) {
				h.GrokGateway.ChatCompletions(c)
				return
			}
			h.OpenAIGateway.ChatCompletions(c)
		})
		gateway.POST("/images/generations", func(c *gin.Context) {
			if isGrokGroup(c) {
				h.GrokGateway.ImagesGeneration(c)
				return
			}
			writeGrokAliasUnavailable(c, "/v1/images/generations")
		})
		gateway.POST("/images/edits", func(c *gin.Context) {
			if isGrokGroup(c) {
				h.GrokGateway.ImagesEdits(c)
				return
			}
			writeGrokAliasUnavailable(c, "/v1/images/edits")
		})
		gateway.POST("/videos/generations", func(c *gin.Context) {
			if isGrokGroup(c) {
				h.GrokGateway.VideosGeneration(c)
				return
			}
			writeGrokAliasUnavailable(c, "/v1/videos/generations")
		})
		gateway.GET("/videos/:request_id", func(c *gin.Context) {
			if isGrokGroup(c) {
				h.GrokGateway.VideoStatus(c)
				return
			}
			writeGrokAliasUnavailable(c, "/v1/videos/:request_id")
		})
	}

	// Gemini 原生 API 兼容层（Gemini SDK/CLI 直连）
	gemini := r.Group("/v1beta")
	grokV1 := r.Group("/grok/v1")
	grokV1.Use(bodyLimit)
	grokV1.Use(clientRequestID)
	grokV1.Use(opsErrorLogger)
	grokV1.Use(endpointNorm)
	grokV1.Use(middleware.ForcePlatform(service.PlatformGrok))
	grokV1.Use(gin.HandlerFunc(apiKeyAuth))
	grokV1.Use(requireGroupAnthropic)
	{
		grokV1.GET("/models", h.Gateway.Models)
		grokV1.POST("/chat/completions", h.GrokGateway.ChatCompletions)
		grokV1.POST("/responses", h.GrokGateway.Responses)
		grokV1.POST("/responses/*subpath", h.GrokGateway.Responses)
		grokV1.GET("/responses/*subpath", h.GrokGateway.Responses)
		grokV1.DELETE("/responses/*subpath", h.GrokGateway.Responses)
		grokV1.POST("/images/generations", h.GrokGateway.ImagesGeneration)
		grokV1.POST("/images/edits", h.GrokGateway.ImagesEdits)
		grokV1.POST("/videos/generations", h.GrokGateway.VideosGeneration)
		grokV1.GET("/videos/:request_id", h.GrokGateway.VideoStatus)
	}
	gemini.Use(bodyLimit)
	gemini.Use(clientRequestID)
	gemini.Use(opsErrorLogger)
	gemini.Use(endpointNorm)
	gemini.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	gemini.Use(requireGroupGoogle)
	{
		gemini.GET("/models", h.Gateway.GeminiV1BetaListModels)
		gemini.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		// Gin treats ":" as a param marker, but Gemini uses "{model}:{action}" in the same segment.
		gemini.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	// OpenAI Responses API（不带v1前缀的别名）
	r.POST("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGrokGroup(c) {
			h.GrokGateway.Responses(c)
			return
		}
		h.OpenAIGateway.Responses(c)
	})
	r.POST("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGrokGroup(c) {
			h.GrokGateway.Responses(c)
			return
		}
		h.OpenAIGateway.Responses(c)
	})
	r.GET("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGrokGroup(c) {
			h.GrokGateway.Responses(c)
			return
		}
		h.OpenAIGateway.Responses(c)
	})
	r.DELETE("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGrokGroup(c) {
			h.GrokGateway.Responses(c)
			return
		}
		h.OpenAIGateway.Responses(c)
	})
	r.GET("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		h.OpenAIGateway.ResponsesWebSocket(c)
	})
	// OpenAI Chat Completions API（不带v1前缀的别名）
	r.POST("/chat/completions", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGrokGroup(c) {
			h.GrokGateway.ChatCompletions(c)
			return
		}
		h.OpenAIGateway.ChatCompletions(c)
	})
	r.POST("/images/generations", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGrokGroup(c) {
			h.GrokGateway.ImagesGeneration(c)
			return
		}
		writeGrokAliasUnavailable(c, "/images/generations")
	})
	r.POST("/images/edits", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGrokGroup(c) {
			h.GrokGateway.ImagesEdits(c)
			return
		}
		writeGrokAliasUnavailable(c, "/images/edits")
	})
	r.POST("/videos/generations", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGrokGroup(c) {
			h.GrokGateway.VideosGeneration(c)
			return
		}
		writeGrokAliasUnavailable(c, "/videos/generations")
	})
	r.GET("/videos/:request_id", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, func(c *gin.Context) {
		if isGrokGroup(c) {
			h.GrokGateway.VideoStatus(c)
			return
		}
		writeGrokAliasUnavailable(c, "/videos/:request_id")
	})

	// Antigravity 模型列表
	r.GET("/antigravity/models", gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.AntigravityModels)

	// Antigravity 专用路由（仅使用 antigravity 账户，不混合调度）
	antigravityV1 := r.Group("/antigravity/v1")
	antigravityV1.Use(bodyLimit)
	antigravityV1.Use(clientRequestID)
	antigravityV1.Use(opsErrorLogger)
	antigravityV1.Use(endpointNorm)
	antigravityV1.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1.Use(gin.HandlerFunc(apiKeyAuth))
	antigravityV1.Use(requireGroupAnthropic)
	{
		antigravityV1.POST("/messages", h.Gateway.Messages)
		antigravityV1.POST("/messages/count_tokens", h.Gateway.CountTokens)
		antigravityV1.GET("/models", h.Gateway.AntigravityModels)
		antigravityV1.GET("/usage", h.Gateway.Usage)
	}

	antigravityV1Beta := r.Group("/antigravity/v1beta")
	antigravityV1Beta.Use(bodyLimit)
	antigravityV1Beta.Use(clientRequestID)
	antigravityV1Beta.Use(opsErrorLogger)
	antigravityV1Beta.Use(endpointNorm)
	antigravityV1Beta.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1Beta.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	antigravityV1Beta.Use(requireGroupGoogle)
	{
		antigravityV1Beta.GET("/models", h.Gateway.GeminiV1BetaListModels)
		antigravityV1Beta.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		antigravityV1Beta.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	// Sora 专用路由（强制使用 sora 平台）
	soraV1 := r.Group("/sora/v1")
	soraV1.Use(soraBodyLimit)
	soraV1.Use(clientRequestID)
	soraV1.Use(opsErrorLogger)
	soraV1.Use(endpointNorm)
	soraV1.Use(middleware.ForcePlatform(service.PlatformSora))
	soraV1.Use(gin.HandlerFunc(apiKeyAuth))
	soraV1.Use(requireGroupAnthropic)
	{
		soraV1.POST("/chat/completions", h.SoraGateway.ChatCompletions)
		soraV1.GET("/models", h.Gateway.Models)
	}

	// Sora 媒体代理（可选 API Key 验证）
	if cfg.Gateway.SoraMediaRequireAPIKey {
		r.GET("/sora/media/*filepath", gin.HandlerFunc(apiKeyAuth), h.SoraGateway.MediaProxy)
	} else {
		r.GET("/sora/media/*filepath", h.SoraGateway.MediaProxy)
	}
	// Sora 媒体代理（签名 URL，无需 API Key）
	r.GET("/sora/media-signed/*filepath", h.SoraGateway.MediaProxySigned)
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

func writeGrokMessagesUnsupported(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    "invalid_request_error",
			"message": "Grok groups do not support /v1/messages endpoints",
		},
	})
}

func writeGrokUnsupported(c *gin.Context, path string) {
	c.AbortWithStatusJSON(http.StatusNotImplemented, gin.H{
		"error": gin.H{
			"type":    "invalid_request_error",
			"message": "Grok routing for " + path + " is not enabled in this build",
		},
	})
}

func writeGrokAliasUnavailable(c *gin.Context, path string) {
	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
		"error": gin.H{
			"type":    "not_found_error",
			"message": path + " is reserved for Grok groups only",
		},
	})
}
