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

type opsRequestTraceCleanupRequest struct {
	Mode   string                               `json:"mode"`
	Filter *opsRequestTraceCleanupFilterPayload `json:"filter"`
}

type opsRequestTraceCleanupFilterPayload struct {
	TimeRange         string `json:"time_range"`
	StartTime         string `json:"start_time"`
	EndTime           string `json:"end_time"`
	Status            string `json:"status"`
	Platform          string `json:"platform"`
	ProtocolIn        string `json:"protocol_in"`
	ProtocolOut       string `json:"protocol_out"`
	Channel           string `json:"channel"`
	RoutePath         string `json:"route_path"`
	RequestType       string `json:"request_type"`
	FinishReason      string `json:"finish_reason"`
	CaptureReason     string `json:"capture_reason"`
	RequestedModel    string `json:"requested_model"`
	UpstreamModel     string `json:"upstream_model"`
	RequestID         string `json:"request_id"`
	ClientRequestID   string `json:"client_request_id"`
	UpstreamRequestID string `json:"upstream_request_id"`
	GeminiSurface     string `json:"gemini_surface"`
	BillingRuleID     string `json:"billing_rule_id"`
	ProbeAction       string `json:"probe_action"`
	Query             string `json:"q"`
	UserID            *int64 `json:"user_id"`
	APIKeyID          *int64 `json:"api_key_id"`
	AccountID         *int64 `json:"account_id"`
	GroupID           *int64 `json:"group_id"`
	StatusCode        *int   `json:"status_code"`
	Stream            *bool  `json:"stream"`
	HasTools          *bool  `json:"has_tools"`
	HasThinking       *bool  `json:"has_thinking"`
	RawAvailable      *bool  `json:"raw_available"`
	Sampled           *bool  `json:"sampled"`
}

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

// POST /api/v1/admin/ops/request-details/cleanup
func (h *OpsHandler) CleanupRequestTraces(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req opsRequestTraceCleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}

	mode := service.OpsRequestTraceCleanupMode(strings.ToLower(strings.TrimSpace(req.Mode)))
	filter, err := buildOpsRequestTraceFilterFromPayload(req.Filter)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.opsService.CleanupRequestTraces(requestTraceAdminContext(c), mode, filter, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
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

func buildOpsRequestTraceFilterFromPayload(payload *opsRequestTraceCleanupFilterPayload) (*service.OpsRequestTraceFilter, error) {
	if payload == nil {
		return nil, nil
	}
	filter := &service.OpsRequestTraceFilter{
		Status:            strings.TrimSpace(payload.Status),
		Platform:          strings.TrimSpace(payload.Platform),
		ProtocolIn:        strings.TrimSpace(payload.ProtocolIn),
		ProtocolOut:       strings.TrimSpace(payload.ProtocolOut),
		Channel:           strings.TrimSpace(payload.Channel),
		RoutePath:         strings.TrimSpace(payload.RoutePath),
		RequestType:       strings.TrimSpace(payload.RequestType),
		FinishReason:      strings.TrimSpace(payload.FinishReason),
		CaptureReason:     strings.TrimSpace(payload.CaptureReason),
		RequestedModel:    strings.TrimSpace(payload.RequestedModel),
		UpstreamModel:     strings.TrimSpace(payload.UpstreamModel),
		RequestID:         strings.TrimSpace(payload.RequestID),
		ClientRequestID:   strings.TrimSpace(payload.ClientRequestID),
		UpstreamRequestID: strings.TrimSpace(payload.UpstreamRequestID),
		GeminiSurface:     strings.TrimSpace(payload.GeminiSurface),
		BillingRuleID:     strings.TrimSpace(payload.BillingRuleID),
		ProbeAction:       strings.TrimSpace(payload.ProbeAction),
		Query:             strings.TrimSpace(payload.Query),
		UserID:            payload.UserID,
		APIKeyID:          payload.APIKeyID,
		AccountID:         payload.AccountID,
		GroupID:           payload.GroupID,
		StatusCode:        payload.StatusCode,
		Stream:            payload.Stream,
		HasTools:          payload.HasTools,
		HasThinking:       payload.HasThinking,
		RawAvailable:      payload.RawAvailable,
		Sampled:           payload.Sampled,
	}

	start, end, err := parseOpsTimeRangeValues(payload.StartTime, payload.EndTime, payload.TimeRange)
	if err != nil {
		return nil, err
	}
	if start != nil {
		filter.StartTime = start
	}
	if end != nil {
		filter.EndTime = end
	}
	return filter, nil
}

func parseOpsTimeRangeValues(startRaw, endRaw, timeRangeRaw string) (*time.Time, *time.Time, error) {
	parseTS := func(raw string) (*time.Time, error) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, nil
		}
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return &t, nil
		}
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}

	start, err := parseTS(startRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid start_time")
	}
	end, err := parseTS(endRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid end_time")
	}
	if start != nil && end != nil && start.After(*end) {
		return nil, nil, fmt.Errorf("invalid time range: start_time must be <= end_time")
	}
	if start != nil || end != nil {
		return start, end, nil
	}

	timeRangeRaw = strings.TrimSpace(timeRangeRaw)
	if timeRangeRaw == "" {
		return nil, nil, nil
	}
	duration, ok := parseOpsDuration(timeRangeRaw)
	if !ok {
		return nil, nil, fmt.Errorf("invalid time_range")
	}
	endValue := time.Now()
	startValue := endValue.Add(-duration)
	return &startValue, &endValue, nil
}
