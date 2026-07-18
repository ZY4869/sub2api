package routes

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func openAIFamilyAuditAllowed(c *gin.Context, inbound string) bool {
	platform := getGroupPlatform(c)
	decision := service.DecideProtocolCapability(platform, inbound, service.ProtocolCapabilityActionDefault)
	return decision.Supported && decision.Mode == service.ProtocolCapabilityNativePassthrough
}

func openAIGeminiCompatAudit(protocol string, promptAudit *securityaudit.PromptService) gin.HandlerFunc {
	return securityaudit.GatewayMiddlewareWhen(promptAudit, protocol, func(c *gin.Context) bool {
		return strings.TrimSpace(geminiOpenAICompatAuditProtocol(c)) == protocol
	})
}

func geminiOpenAICompatAuditProtocol(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.URL == nil || c.Request.Method != "POST" {
		return ""
	}
	path := strings.ToLower(strings.TrimSpace(c.Request.URL.Path))
	switch {
	case strings.Contains(path, "/openai/chat/completions"):
		return securityaudit.ProtocolOpenAIChat
	case strings.Contains(path, "/openai/embeddings"):
		return securityaudit.ProtocolOpenAIEmbeddings
	case strings.Contains(path, "/openai/images/"), strings.Contains(path, "/openai/videos"):
		return securityaudit.ProtocolOpenAIImages
	default:
		return ""
	}
}

func openAIOnlyAudit(promptAudit *securityaudit.PromptService, protocol string, inbound string) gin.HandlerFunc {
	return securityaudit.GatewayMiddlewareWhen(promptAudit, protocol, func(c *gin.Context) bool {
		if forced, ok := middleware.GetForcePlatformFromContext(c); ok && forced != "" && forced != service.PlatformOpenAI {
			return false
		}
		return openAIFamilyAuditAllowed(c, inbound)
	})
}
