package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type DocsHandler struct {
	docsService *service.APIDocsService
}

func NewDocsHandler(docsService *service.APIDocsService) *DocsHandler {
	return &DocsHandler{docsService: docsService}
}

func (h *DocsHandler) GetAPIReference(c *gin.Context) {
	content, err := h.docsService.GetEffectiveContent(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"content": content,
	})
}
