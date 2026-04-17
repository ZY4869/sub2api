package admin

import (
	"log"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type DocsHandler struct {
	docsService *service.APIDocsService
}

type updateAPIDocsRequest struct {
	Content string `json:"content"`
}

func NewDocsHandler(docsService *service.APIDocsService) *DocsHandler {
	return &DocsHandler{docsService: docsService}
}

func (h *DocsHandler) GetAPIReference(c *gin.Context) {
	document, err := h.docsService.GetDocument(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"effective_content": document.EffectiveContent,
		"default_content":   document.DefaultContent,
		"has_override":      document.HasOverride,
	})
}

func (h *DocsHandler) UpdateAPIReference(c *gin.Context) {
	var req updateAPIDocsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.docsService.SaveOverride(c.Request.Context(), req.Content); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	document, err := h.docsService.GetDocument(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	auditAPIDocsAction(c, "save_override", len(document.EffectiveContent), document.HasOverride)
	response.Success(c, gin.H{
		"effective_content": document.EffectiveContent,
		"default_content":   document.DefaultContent,
		"has_override":      document.HasOverride,
	})
}

func (h *DocsHandler) ClearAPIReferenceOverride(c *gin.Context) {
	if err := h.docsService.ClearOverride(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	document, err := h.docsService.GetDocument(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	auditAPIDocsAction(c, "clear_override", 0, document.HasOverride)
	response.Success(c, gin.H{
		"effective_content": document.EffectiveContent,
		"default_content":   document.DefaultContent,
		"has_override":      document.HasOverride,
	})
}

func auditAPIDocsAction(c *gin.Context, action string, contentLength int, hasOverride bool) {
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	role, _ := middleware.GetUserRoleFromContext(c)
	log.Printf(
		"AUDIT: api_docs action=%s at=%s user_id=%d role=%s content_length=%d has_override=%t",
		action,
		time.Now().UTC().Format(time.RFC3339),
		subject.UserID,
		role,
		contentLength,
		hasOverride,
	)
}
