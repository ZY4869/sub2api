package admin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func requestTraceAdminContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	role, ok := middleware.GetUserRoleFromContext(c)
	return service.WithOpsRequestTraceAdminRawAccess(ctx, ok && role == service.RoleAdmin)
}

// GET /api/v1/admin/ops/request-details
func (h *OpsHandler) ListRequestTraces(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}

	filter, err := buildOpsRequestTraceFilterFromQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	subject, _ := middleware.GetAuthSubjectFromContext(c)
	result, err := h.opsService.ListRequestTraces(requestTraceAdminContext(c), subject.UserID, filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, result.Items, result.Total, result.Page, result.PageSize)
}

// GET /api/v1/admin/ops/request-details/summary
func (h *OpsHandler) GetRequestTraceSummary(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}

	filter, err := buildOpsRequestTraceFilterFromQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	subject, _ := middleware.GetAuthSubjectFromContext(c)
	summary, err := h.opsService.GetRequestTraceSummary(requestTraceAdminContext(c), subject.UserID, filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, summary)
}

// GET /api/v1/admin/ops/request-details/:id
func (h *OpsHandler) GetRequestTraceByID(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}

	id, err := parseOpsRequestTraceID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	subject, _ := middleware.GetAuthSubjectFromContext(c)
	detail, err := h.opsService.GetRequestTraceByID(requestTraceAdminContext(c), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

// GET /api/v1/admin/ops/request-details/:id/raw
func (h *OpsHandler) GetRequestTraceRawByID(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := parseOpsRequestTraceID(c.Param("id"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	detail, err := h.opsService.GetRequestTraceRawByID(requestTraceAdminContext(c), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

// GET /api/v1/admin/ops/request-details/export.csv
func (h *OpsHandler) ExportRequestTracesCSV(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	filter, err := buildOpsRequestTraceFilterFromQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	includeRaw := false
	if rawValue := strings.TrimSpace(c.Query("include_raw")); rawValue != "" {
		parsed, parseErr := parseOptionalBool(rawValue)
		if parseErr != nil {
			response.BadRequest(c, "Invalid include_raw")
			return
		}
		includeRaw = parsed != nil && *parsed
	}

	filename := "request-details-" + time.Now().UTC().Format("20060102-150405") + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Cache-Control", "no-store")

	if _, err := c.Writer.WriteString("\xEF\xBB\xBF"); err != nil {
		response.InternalError(c, "Failed to start export")
		return
	}

	if _, err := h.opsService.ExportRequestTracesCSV(requestTraceAdminContext(c), c.Writer, subject.UserID, filter, includeRaw); err != nil {
		if !c.Writer.Written() {
			response.ErrorFrom(c, err)
			return
		}
		_ = c.Error(err)
	}
}

func buildOpsRequestTraceFilterFromQuery(c *gin.Context) (*service.OpsRequestTraceFilter, error) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > 200 {
		pageSize = 200
	}

	startTime, endTime, err := parseOpsTimeRange(c, "1h")
	if err != nil {
		return nil, err
	}

	filter := &service.OpsRequestTraceFilter{
		Page:              page,
		PageSize:          pageSize,
		StartTime:         &startTime,
		EndTime:           &endTime,
		Status:            strings.TrimSpace(c.Query("status")),
		Platform:          strings.TrimSpace(c.Query("platform")),
		ProtocolIn:        strings.TrimSpace(c.Query("protocol_in")),
		ProtocolOut:       strings.TrimSpace(c.Query("protocol_out")),
		Channel:           strings.TrimSpace(c.Query("channel")),
		RoutePath:         strings.TrimSpace(c.Query("route_path")),
		RequestType:       strings.TrimSpace(c.Query("request_type")),
		FinishReason:      strings.TrimSpace(c.Query("finish_reason")),
		CaptureReason:     strings.TrimSpace(c.Query("capture_reason")),
		RequestedModel:    strings.TrimSpace(c.Query("requested_model")),
		UpstreamModel:     strings.TrimSpace(c.Query("upstream_model")),
		RequestID:         strings.TrimSpace(c.Query("request_id")),
		ClientRequestID:   strings.TrimSpace(c.Query("client_request_id")),
		UpstreamRequestID: strings.TrimSpace(c.Query("upstream_request_id")),
		GeminiSurface:     strings.TrimSpace(c.Query("gemini_surface")),
		BillingRuleID:     strings.TrimSpace(c.Query("billing_rule_id")),
		ProbeAction:       strings.TrimSpace(c.Query("probe_action")),
		Query:             strings.TrimSpace(c.Query("q")),
		Sort:              strings.TrimSpace(c.Query("sort")),
	}

	if filter.Sort == "" {
		filter.Sort = "created_at_desc"
	}

	if filter.UserID, err = parseInt64Query(c, "user_id"); err != nil {
		return nil, err
	}
	if filter.APIKeyID, err = parseInt64Query(c, "api_key_id"); err != nil {
		return nil, err
	}
	if filter.AccountID, err = parseInt64Query(c, "account_id"); err != nil {
		return nil, err
	}
	if filter.GroupID, err = parseInt64Query(c, "group_id"); err != nil {
		return nil, err
	}
	if filter.StatusCode, err = parseIntQuery(c, "status_code"); err != nil {
		return nil, err
	}
	if filter.Stream, err = parseBoolQuery(c, "stream"); err != nil {
		return nil, err
	}
	if filter.HasTools, err = parseBoolQuery(c, "has_tools"); err != nil {
		return nil, err
	}
	if filter.HasThinking, err = parseBoolQuery(c, "has_thinking"); err != nil {
		return nil, err
	}
	if filter.RawAvailable, err = parseBoolQuery(c, "raw_available"); err != nil {
		return nil, err
	}
	if filter.Sampled, err = parseBoolQuery(c, "sampled"); err != nil {
		return nil, err
	}

	return filter, nil
}

func parseOpsRequestTraceID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid request detail id")
	}
	return id, nil
}

func parseInt64Query(c *gin.Context, key string) (*int64, error) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return nil, fmt.Errorf("invalid %s", key)
	}
	return &parsed, nil
}

func parseIntQuery(c *gin.Context, key string) (*int, error) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return nil, fmt.Errorf("invalid %s", key)
	}
	return &parsed, nil
}

func parseBoolQuery(c *gin.Context, key string) (*bool, error) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return nil, nil
	}
	return parseOptionalBool(value)
}

func parseOptionalBool(value string) (*bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes":
		v := true
		return &v, nil
	case "0", "false", "no":
		v := false
		return &v, nil
	default:
		return nil, fmt.Errorf("invalid boolean value")
	}
}
