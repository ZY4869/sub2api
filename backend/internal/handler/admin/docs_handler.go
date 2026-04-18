package admin

import (
	"log"
	"strings"
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
	pageID := strings.TrimSpace(c.Query("page_id"))
	document, err := h.resolveDocument(c, pageID)
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

	pageID := strings.TrimSpace(c.Query("page_id"))
	var err error
	if pageID != "" {
		err = h.docsService.SavePageOverride(c.Request.Context(), pageID, req.Content)
	} else {
		err = h.docsService.SaveOverride(c.Request.Context(), req.Content)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	document, err := h.resolveDocument(c, pageID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	auditAPIDocsAction(c, "save_override", pageID, len(document.EffectiveContent), document.HasOverride)
	response.Success(c, gin.H{
		"effective_content": document.EffectiveContent,
		"default_content":   document.DefaultContent,
		"has_override":      document.HasOverride,
	})
}

func (h *DocsHandler) ClearAPIReferenceOverride(c *gin.Context) {
	pageID := strings.TrimSpace(c.Query("page_id"))
	var err error
	if pageID != "" {
		err = h.docsService.ClearPageOverride(c.Request.Context(), pageID)
	} else {
		err = h.docsService.ClearOverride(c.Request.Context())
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	document, err := h.resolveDocument(c, pageID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	auditAPIDocsAction(c, "clear_override", pageID, 0, document.HasOverride)
	response.Success(c, gin.H{
		"effective_content": document.EffectiveContent,
		"default_content":   document.DefaultContent,
		"has_override":      document.HasOverride,
	})
}

func (h *DocsHandler) resolveDocument(c *gin.Context, pageID string) (*service.APIDocsDocument, error) {
	if strings.TrimSpace(pageID) == "" {
		return h.docsService.GetDocument(c.Request.Context())
	}
	return h.docsService.GetPageDocument(c.Request.Context(), pageID)
}

func auditAPIDocsAction(c *gin.Context, action string, pageID string, contentLength int, hasOverride bool) {
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	role, _ := middleware.GetUserRoleFromContext(c)
	log.Printf(
		"AUDIT: api_docs action=%s page_id=%s at=%s user_id=%d role=%s content_length=%d has_override=%t",
		action,
		pageID,
		time.Now().UTC().Format(time.RFC3339),
		subject.UserID,
		role,
		contentLength,
		hasOverride,
	)
}
