package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func AuthSessionBinding() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c != nil && c.Request != nil {
			ctx := service.WithAuthSessionBinding(c.Request.Context(), ip.GetTrustedClientIP(c), c.GetHeader("User-Agent"))
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}
