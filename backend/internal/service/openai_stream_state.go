package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func markOpenAIRealSSEStarted(c *gin.Context) {
	if c == nil || c.Request == nil {
		return
	}
	ctx := EnsureRequestMetadata(c.Request.Context())
	SetOpenAIRealSSEStartedMetadata(ctx, true)
	c.Request = c.Request.WithContext(ctx)
}

func isOpenAIKeepaliveSSELine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	if strings.HasPrefix(trimmed, ":") {
		return true
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return false
	}
	data := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
	return data == "" || data == "[DONE]"
}
