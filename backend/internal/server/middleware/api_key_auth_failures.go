package middleware

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func apiKeyAuthFailureIdentity(c *gin.Context, apiKey string) string {
	if key := strings.TrimSpace(apiKey); key != "" {
		return "key:" + key
	}

	method := ""
	path := ""
	if c != nil && c.Request != nil {
		method = strings.ToUpper(strings.TrimSpace(c.Request.Method))
		if c.Request.URL != nil {
			path = strings.TrimSpace(c.Request.URL.Path)
		}
	}
	clientIP := ""
	if c != nil {
		clientIP = ip.GetTrustedClientIP(c)
	}
	return "missing:" + clientIP + ":" + method + ":" + path
}

func apiKeyAuthFailureRateLimited(ctx context.Context, svc *service.APIKeyService, identity string) bool {
	if svc == nil || strings.TrimSpace(identity) == "" {
		return false
	}
	return errors.Is(svc.CheckAuthFailureRateLimit(ctx, identity), service.ErrAPIKeyRateLimited)
}

func recordAPIKeyAuthFailureRateLimited(ctx context.Context, svc *service.APIKeyService, identity string) bool {
	if svc == nil || strings.TrimSpace(identity) == "" {
		return false
	}
	return errors.Is(svc.RecordAuthFailure(ctx, identity), service.ErrAPIKeyRateLimited)
}

func clearAPIKeyAuthFailure(ctx context.Context, svc *service.APIKeyService, identity string) {
	if svc == nil || strings.TrimSpace(identity) == "" {
		return
	}
	svc.ClearAuthFailure(ctx, identity)
}
