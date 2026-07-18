package securityaudit

import (
	"errors"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	service *PromptService
}

func NewAdminHandler(service *PromptService) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) GetConfig(c *gin.Context) {
	response.Success(c, h.service.GetConfig())
}

func (h *AdminHandler) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_audit_invalid_config_request", "prompt audit config request is invalid"))
		return
	}
	cfg, err := h.service.SaveConfig(c.Request.Context(), req, adminID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

func (h *AdminHandler) GetRuntime(c *gin.Context) {
	response.Success(c, h.service.Runtime(c.Request.Context()))
}

func (h *AdminHandler) ProbeEndpoint(c *gin.Context) {
	var req ProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_audit_invalid_probe_request", "prompt audit probe request is invalid"))
		return
	}
	response.Success(c, h.service.Probe(c.Request.Context(), req))
}

func (h *AdminHandler) ListEvents(c *gin.Context) {
	filter, err := eventFilterFromQuery(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result, err := h.service.ListEvents(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *AdminHandler) GetEvent(c *gin.Context) {
	id, err := eventIDParam(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	event, err := h.service.GetEvent(c.Request.Context(), id)
	if errors.Is(err, ErrEventNotFound) {
		response.ErrorFrom(c, infraerrors.NotFound("prompt_audit_event_not_found", "prompt audit event not found"))
		return
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, event)
}

func (h *AdminHandler) DeleteEvent(c *gin.Context) {
	id, err := eventIDParam(c)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result, err := h.service.DeleteEvent(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *AdminHandler) DeletePreview(c *gin.Context) {
	var filter EventFilter
	if err := c.ShouldBindJSON(&filter); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_audit_delete_preview_invalid", "prompt audit delete preview is invalid"))
		return
	}
	preview, err := h.service.PreviewDelete(c.Request.Context(), filter, adminID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, preview)
}

func (h *AdminHandler) DeleteByFilter(c *gin.Context) {
	var req DeleteByFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("prompt_audit_delete_confirmation_invalid", "prompt audit delete confirmation is invalid"))
		return
	}
	result, err := h.service.DeleteByFilter(c.Request.Context(), req, adminID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func eventFilterFromQuery(c *gin.Context) (EventFilter, error) {
	page, err := positiveQuery(c, "page", 1, 0)
	if err != nil {
		return EventFilter{}, err
	}
	pageSize, err := positiveQuery(c, "page_size", 20, 100)
	if err != nil {
		return EventFilter{}, err
	}
	userID, err := optionalInt64Query(c, "user_id")
	if err != nil {
		return EventFilter{}, err
	}
	apiKeyID, err := optionalInt64Query(c, "api_key_id")
	if err != nil {
		return EventFilter{}, err
	}
	groupID, err := optionalInt64Query(c, "group_id")
	if err != nil {
		return EventFilter{}, err
	}
	return EventFilter{
		Page: page, PageSize: pageSize, Decision: c.Query("decision"), RiskLevel: c.Query("risk_level"),
		Endpoint: c.Query("endpoint"), Protocol: c.Query("protocol"), UserID: userID, APIKeyID: apiKeyID,
		GroupID: groupID, PromptHash: c.Query("prompt_hash"), RequestID: c.Query("request_id"), Keyword: c.Query("keyword"),
	}, nil
}

func eventIDParam(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, infraerrors.BadRequest("prompt_audit_invalid_event_id", "prompt audit event id is invalid")
	}
	return id, nil
}

func positiveQuery(c *gin.Context, key string, fallback int, max int) (int, error) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 || (max > 0 && parsed > max) {
		return 0, infraerrors.BadRequest("prompt_audit_invalid_pagination", "prompt audit pagination is invalid")
	}
	return parsed, nil
}

func optionalInt64Query(c *gin.Context, key string) (*int64, error) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return nil, infraerrors.BadRequest("prompt_audit_invalid_filter_id", "prompt audit filter id is invalid")
	}
	return &parsed, nil
}

func adminID(c *gin.Context) int64 {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		return 0
	}
	return subject.UserID
}
