package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type MetaHandler struct {
	modelCatalogService *service.ModelCatalogService
}

func NewMetaHandler(modelCatalogService *service.ModelCatalogService) *MetaHandler {
	return &MetaHandler{modelCatalogService: modelCatalogService}
}

func (h *MetaHandler) USDCNYExchangeRate(c *gin.Context) {
	rate, err := h.modelCatalogService.GetUSDCNYExchangeRate(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, rate)
}
