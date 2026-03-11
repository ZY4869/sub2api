package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterMetaRoutes(v1 *gin.RouterGroup, h *handler.Handlers) {
	meta := v1.Group("/meta")
	{
		meta.GET("/exchange-rate/usd-cny", h.Meta.USDCNYExchangeRate)
	}
}
