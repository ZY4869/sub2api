package handler

import (
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

var opsRetryRequestHeaderAllowlist = []string{
	"anthropic-beta",
	"anthropic-version",
}

func applyOpsAPIKeyAndClientIP(c *gin.Context, apiKey *service.APIKey, entry *service.OpsInsertErrorLogInput) {
	if entry == nil {
		return
	}
	if apiKey != nil {
		entry.APIKeyID = &apiKey.ID
		if apiKey.User != nil {
			entry.UserID = &apiKey.User.ID
		}
		if apiKey.GroupID != nil {
			entry.GroupID = apiKey.GroupID
		}
		// Prefer group platform if present (more stable than inferring from path).
		if apiKey.Group != nil && apiKey.Group.Platform != "" {
			entry.Platform = apiKey.Group.Platform
		}
	}

	var clientIP string
	if ip := strings.TrimSpace(ip.GetTrustedClientIP(c)); ip != "" {
		clientIP = ip
		entry.ClientIP = &clientIP
	}
}

// isCountTokensRequest checks if the request is a count_tokens request
func isCountTokensRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return false
	}
	return strings.Contains(c.Request.URL.Path, "/count_tokens")
}

func extractOpsRetryRequestHeaders(c *gin.Context) *string {
	if c == nil || c.Request == nil {
		return nil
	}

	headers := make(map[string]string, 4)
	for _, key := range opsRetryRequestHeaderAllowlist {
		v := strings.TrimSpace(c.GetHeader(key))
		if v == "" {
			continue
		}
		// Keep headers small even if a client sends something unexpected.
		headers[key] = truncateString(v, 512)
	}
	if len(headers) == 0 {
		return nil
	}

	raw, err := json.Marshal(headers)
	if err != nil {
		return nil
	}
	s := string(raw)
	return &s
}

func applyOpsLatencyFieldsFromContext(c *gin.Context, entry *service.OpsInsertErrorLogInput) {
	if c == nil || entry == nil {
		return
	}
	entry.AuthLatencyMs = getContextLatencyMs(c, service.OpsAuthLatencyMsKey)
	entry.RoutingLatencyMs = getContextLatencyMs(c, service.OpsRoutingLatencyMsKey)
	entry.UpstreamLatencyMs = getContextLatencyMs(c, service.OpsUpstreamLatencyMsKey)
	entry.ResponseLatencyMs = getContextLatencyMs(c, service.OpsResponseLatencyMsKey)
	entry.TimeToFirstTokenMs = getContextLatencyMs(c, service.OpsTimeToFirstTokenMsKey)
}

func applyOpsGeminiMetadataFromContext(c *gin.Context, entry *service.OpsInsertErrorLogInput) {
	if c == nil || c.Request == nil || entry == nil {
		return
	}
	if value, ok := service.GeminiSurfaceMetadataFromContext(c.Request.Context()); ok {
		entry.GeminiSurface = value
	}
	if value, ok := service.BillingRuleIDMetadataFromContext(c.Request.Context()); ok {
		entry.BillingRuleID = value
	}
	if value, ok := service.ProbeActionMetadataFromContext(c.Request.Context()); ok {
		entry.ProbeAction = value
	}
}

func resolveOpsChannelID(c *gin.Context) *int64 {
	if c == nil || c.Request == nil {
		return nil
	}
	state, ok := service.GatewayChannelStateFromContext(c.Request.Context())
	if !ok {
		return nil
	}
	return state.ChannelIDPtr()
}

func getContextLatencyMs(c *gin.Context, key string) *int64 {
	if c == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	v, ok := c.Get(key)
	if !ok {
		return nil
	}
	var ms int64
	switch t := v.(type) {
	case int:
		ms = int64(t)
	case int32:
		ms = int64(t)
	case int64:
		ms = t
	case float64:
		ms = int64(t)
	default:
		return nil
	}
	if ms < 0 {
		return nil
	}
	return &ms
}
