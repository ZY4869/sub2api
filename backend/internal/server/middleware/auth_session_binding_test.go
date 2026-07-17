package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAuthSessionBindingMiddlewareSetsTrustedClientContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(AuthSessionBinding())
	r.GET("/auth/check", func(c *gin.Context) {
		binding, ok := service.AuthSessionBindingFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, "192.0.2.10", binding.ClientIP)
		require.Equal(t, "sub2api-test/1.0", binding.UserAgent)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/auth/check", nil)
	req.RemoteAddr = "192.0.2.10:3456"
	req.Header.Set("User-Agent", "sub2api-test/1.0")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}
