package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ContentModerationAuditHandler struct {
	service *service.ContentModerationService
}

func NewContentModerationAuditHandler(contentModerationService *service.ContentModerationService) *ContentModerationAuditHandler {
	return &ContentModerationAuditHandler{service: contentModerationService}
}

// List handles GET /api/v1/admin/moderation/audits
func (h *ContentModerationAuditHandler) List(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "Content moderation service unavailable")
		return
	}

	page, pageSize := response.ParsePagination(c)
	filter := &service.ContentModerationAuditFilter{
		Page:            page,
		PageSize:        pageSize,
		RequestID:       strings.TrimSpace(c.Query("request_id")),
		ClientRequestID: strings.TrimSpace(c.Query("client_request_id")),
		Provider:        strings.TrimSpace(c.Query("provider")),
		Model:           strings.TrimSpace(c.Query("model")),
		SourceEndpoint:  strings.TrimSpace(c.Query("source_endpoint")),
		ContentHash:     strings.TrimSpace(c.Query("content_hash")),
	}

	if rawHit := strings.TrimSpace(c.Query("hit")); rawHit != "" {
		parsed, err := strconv.ParseBool(rawHit)
		if err != nil {
			response.BadRequest(c, "Invalid hit filter")
			return
		}
		filter.Hit = &parsed
	}
	if rawUserID := strings.TrimSpace(c.Query("user_id")); rawUserID != "" {
		userID, err := strconv.ParseInt(rawUserID, 10, 64)
		if err != nil || userID <= 0 {
			response.BadRequest(c, "Invalid user_id")
			return
		}
		filter.UserID = &userID
	}

	result, err := h.service.ListAudits(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]dto.ContentModerationAudit, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, *dto.ContentModerationAuditFromService(result.Items[i]))
	}
	response.Paginated(c, items, result.Total, result.Page, result.PageSize)
}

// Detail handles GET /api/v1/admin/moderation/audits/:id
func (h *ContentModerationAuditHandler) Detail(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "Content moderation service unavailable")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid audit ID")
		return
	}

	item, err := h.service.GetAuditByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		logger.With(
			zap.String("component", "audit.admin.content_moderation.detail"),
			zap.Int64("operator_user_id", subject.UserID),
			zap.Int64("audit_id", id),
			zap.String("provider", strings.TrimSpace(item.Provider)),
			zap.String("source_endpoint", strings.TrimSpace(item.SourceEndpoint)),
			zap.String("request_id", strings.TrimSpace(item.RequestID)),
			zap.String("client_request_id", strings.TrimSpace(item.ClientRequestID)),
		).Info("admin viewed content moderation audit detail")
	}

	response.Success(c, dto.ContentModerationAuditFromService(item))
}
