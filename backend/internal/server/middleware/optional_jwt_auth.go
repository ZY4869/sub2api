package middleware

import (
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// NewOptionalJWTAuthMiddleware parses a Bearer token when present and silently
// falls through when credentials are absent or invalid.
func NewOptionalJWTAuthMiddleware(authService *service.AuthService, userService *service.UserService) gin.HandlerFunc {
	return optionalJWTAuth(authService, userService)
}

func optionalJWTAuth(authService *service.AuthService, userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authService == nil || userService == nil {
			c.Next()
			return
		}

		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, service.ErrTokenExpired) {
				c.Next()
				return
			}
			c.Next()
			return
		}
		if claims == nil || claims.UserID <= 0 {
			c.Next()
			return
		}

		user, err := userService.GetByID(c.Request.Context(), claims.UserID)
		if err != nil || user == nil || !user.IsActive() || claims.TokenVersion != user.TokenVersion {
			c.Next()
			return
		}

		c.Set(string(ContextKeyUser), AuthSubject{
			UserID:      user.ID,
			Concurrency: user.Concurrency,
		})
		c.Set(string(ContextKeyUserRole), user.Role)
		c.Next()
	}
}
