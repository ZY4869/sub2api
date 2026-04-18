package server

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterRoutes_RegistersMetaRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handlers := &handler.Handlers{
		Auth:          &handler.AuthHandler{},
		User:          &handler.UserHandler{},
		Meta:          &handler.MetaHandler{},
		APIKey:        &handler.APIKeyHandler{},
		Usage:         &handler.UsageHandler{},
		Redeem:        &handler.RedeemHandler{},
		Subscription:  &handler.SubscriptionHandler{},
		Announcement:  &handler.AnnouncementHandler{},
		Admin:         &handler.AdminHandlers{},
		Gateway:       &handler.GatewayHandler{},
		OpenAIGateway: &handler.OpenAIGatewayHandler{},
		GrokGateway:   &handler.GrokGatewayHandler{},
		Setting:       &handler.SettingHandler{},
		Totp:          &handler.TotpHandler{},
	}

	noAuth := func(c *gin.Context) {}
	registerRoutes(
		router,
		handlers,
		middleware2.JWTAuthMiddleware(noAuth),
		middleware2.AdminAuthMiddleware(noAuth),
		middleware2.APIKeyAuthMiddleware(noAuth),
		nil,
		nil,
		nil,
		nil,
		&config.Config{
			Gateway: config.GatewayConfig{
				MaxBodySize: 1,
			},
		},
		nil,
	)

	routes := router.Routes()
	require.True(t, hasRoute(routes, "GET", "/api/v1/meta/exchange-rate/usd-cny"))
	require.True(t, hasRoute(routes, "GET", "/api/v1/meta/model-catalog"))
	require.True(t, hasRoute(routes, "GET", "/api/v1/meta/model-registry"))
}

func hasRoute(routes gin.RoutesInfo, method, path string) bool {
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}
