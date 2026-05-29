package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func resolveOpsTracePlatform(c *gin.Context, apiKey *service.APIKey) string {
	if c != nil && c.Request != nil {
		if value, ok := c.Request.Context().Value(ctxkey.Platform).(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if forced, ok := middleware2.GetForcePlatformFromContext(c); ok && strings.TrimSpace(forced) != "" {
		return strings.TrimSpace(forced)
	}
	if apiKey != nil && apiKey.Group != nil && strings.TrimSpace(apiKey.Group.Platform) != "" {
		return strings.TrimSpace(apiKey.Group.Platform)
	}
	return guessPlatformFromPath(c.Request.URL.Path)
}

func inferOpsTraceProtocolIn(c *gin.Context) string {
	if inbound := normalizeOpsTraceProtocolValue(GetInboundEndpoint(c)); inbound != "" {
		return inbound
	}
	return normalizeOpsTraceProtocolValue(inferOpsTraceRoutePath(c))
}

func inferOpsTraceProtocolOut(c *gin.Context, protocolIn string, platform string) string {
	if endpoint := normalizeOpsTraceProtocolValue(resolveOpsUpstreamEndpoint(c, platform)); endpoint != "" {
		return endpoint
	}
	if inbound := normalizeOpsTraceProtocolValue(protocolIn); inbound != "" {
		return inbound
	}
	return normalizeOpsTraceProtocolValue(GetUpstreamEndpoint(c, platform))
}

func inferOpsTraceChannel(c *gin.Context, protocolIn, protocolOut string) string {
	if c != nil && c.Request != nil {
		if state, ok := service.GatewayChannelStateFromContext(c.Request.Context()); ok && state != nil {
			if channelName := strings.TrimSpace(state.ChannelName()); channelName != "" {
				return channelName
			}
		}
	}
	path := strings.ToLower(inferOpsTraceRoutePath(c))
	inboundFamily := opsTraceProtocolFamily(protocolIn)
	outboundFamily := opsTraceProtocolFamily(protocolOut)
	switch {
	case strings.Contains(path, "/publishers/google/models/"):
		return "vertex"
	case outboundFamily == "gemini" && inboundFamily != "gemini":
		return "gemini_compat"
	case outboundFamily == "gemini":
		return "ai_studio"
	case outboundFamily == "anthropic":
		return "anthropic"
	case outboundFamily == "openai":
		return "openai_compat"
	default:
		if outboundFamily != "" {
			return outboundFamily
		}
		return normalizeOpsTraceProtocolValue(protocolOut)
	}
}

func inferOpsTraceRoutePath(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if fullPath := strings.TrimSpace(c.FullPath()); fullPath != "" {
		return fullPath
	}
	if c.Request.URL != nil {
		return strings.TrimSpace(c.Request.URL.Path)
	}
	return ""
}

func inferOpsTraceRequestType(c *gin.Context) string {
	stream := false
	if value, ok := c.Get(opsStreamKey); ok {
		if parsed, ok := value.(bool); ok {
			stream = parsed
		}
	}
	if requestType := resolveOpsRequestType(c, stream); requestType != nil {
		return service.RequestTypeFromInt16(*requestType).String()
	}
	return service.RequestTypeFromLegacy(stream, false).String()
}

func getOpsTraceRequestBody(c *gin.Context) []byte {
	if c == nil {
		return nil
	}
	value, ok := c.Get(opsRequestBodyKey)
	if !ok {
		return nil
	}
	raw, ok := value.([]byte)
	if !ok || len(raw) == 0 {
		return nil
	}
	if len(raw) > opsRequestTraceBodyLimit {
		return append([]byte(nil), raw[:opsRequestTraceBodyLimit]...)
	}
	return append([]byte(nil), raw...)
}

func readOpsTraceStatusCode(c *gin.Context) *int {
	if c == nil {
		return nil
	}
	value, ok := c.Get(service.OpsUpstreamStatusCodeKey)
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return &typed
		}
	case int64:
		if typed > 0 {
			value := int(typed)
			return &value
		}
	}
	return nil
}

func readContextString(c *gin.Context, key ctxkey.Key) string {
	if c == nil || c.Request == nil {
		return ""
	}
	value, _ := c.Request.Context().Value(key).(string)
	return strings.TrimSpace(value)
}

func readOpsTraceModel(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if value, ok := c.Get(opsModelKey); ok {
		if model, ok := value.(string); ok {
			return strings.TrimSpace(model)
		}
	}
	return ""
}
