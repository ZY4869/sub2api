package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type usageFilterAPIKeyItem struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Deleted bool   `json:"deleted"`
}

// UsageHandler handles usage-related requests
type UsageHandler struct {
	usageService  *service.UsageService
	apiKeyService *service.APIKeyService
	opsService    *service.OpsService
}

// NewUsageHandler creates a new UsageHandler
func NewUsageHandler(usageService *service.UsageService, apiKeyService *service.APIKeyService) *UsageHandler {
	return &UsageHandler{
		usageService:  usageService,
		apiKeyService: apiKeyService,
	}
}

func (h *UsageHandler) SetOpsService(opsService *service.OpsService) {
	if h != nil {
		h.opsService = opsService
	}
}

// List handles listing usage records with pagination
// GET /api/v1/usage
func (h *UsageHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)

	var apiKeyID int64
	if apiKeyIDStr := c.Query("api_key_id"); apiKeyIDStr != "" {
		id, err := strconv.ParseInt(apiKeyIDStr, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid api_key_id")
			return
		}

		// [Security Fix] Verify API Key ownership to prevent horizontal privilege escalation
		apiKey, err := h.apiKeyService.GetByIDAllowDeleted(c.Request.Context(), id)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if apiKey.UserID != subject.UserID {
			response.Forbidden(c, "Not authorized to access this API key's usage records")
			return
		}

		apiKeyID = id
	}

	// Parse additional filters
	model := c.Query("model")
	platform := strings.TrimSpace(c.Query("platform"))

	var requestType *int16
	var stream *bool
	if requestTypeStr := strings.TrimSpace(c.Query("request_type")); requestTypeStr != "" {
		parsed, err := service.ParseUsageRequestType(requestTypeStr)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		value := int16(parsed)
		requestType = &value
	} else if streamStr := c.Query("stream"); streamStr != "" {
		val, err := strconv.ParseBool(streamStr)
		if err != nil {
			response.BadRequest(c, "Invalid stream value, use true or false")
			return
		}
		stream = &val
	}

	var billingType *int8
	if billingTypeStr := c.Query("billing_type"); billingTypeStr != "" {
		val, err := strconv.ParseInt(billingTypeStr, 10, 8)
		if err != nil {
			response.BadRequest(c, "Invalid billing_type")
			return
		}
		bt := int8(val)
		billingType = &bt
	}

	// Parse date range
	var startTime, endTime *time.Time
	userTZ := c.Query("timezone") // Get user's timezone from request
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid start_date format, use YYYY-MM-DD")
			return
		}
		startTime = &t
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid end_date format, use YYYY-MM-DD")
			return
		}
		// Use half-open range [start, end), move to next calendar day start (DST-safe).
		t = t.AddDate(0, 0, 1)
		endTime = &t
	}

	params := pagination.PaginationParams{Page: page, PageSize: pageSize}
	filters := usagestats.UsageLogFilters{
		UserID:      subject.UserID, // Always filter by current user for security
		APIKeyID:    apiKeyID,
		Platform:    platform,
		Model:       model,
		RequestType: requestType,
		Stream:      stream,
		BillingType: billingType,
		StartTime:   startTime,
		EndTime:     endTime,
	}

	records, result, err := h.usageService.ListWithFilters(c.Request.Context(), params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.UsageLog, 0, len(records))
	for i := range records {
		out = append(out, *dto.UsageLogFromService(&records[i]))
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

// FilterAPIKeys handles listing usage-backed API keys for the current user.
// GET /api/v1/usage/filter-api-keys
func (h *UsageHandler) FilterAPIKeys(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var startTime, endTime *time.Time
	userTZ := c.Query("timezone")
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid start_date format, use YYYY-MM-DD")
			return
		}
		startTime = &t
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid end_date format, use YYYY-MM-DD")
			return
		}
		t = t.AddDate(0, 0, 1)
		endTime = &t
	}

	keys, err := h.usageService.ListUsageFilterAPIKeys(c.Request.Context(), subject.UserID, startTime, endTime)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]usageFilterAPIKeyItem, 0, len(keys))
	for _, key := range keys {
		items = append(items, usageFilterAPIKeyItem{
			ID:      key.ID,
			Name:    key.Name,
			Deleted: key.Deleted,
		})
	}

	logger.LegacyPrintf(
		"handler.usage",
		"[UsageFilterAPIKeys] user=%d start=%q end=%q count=%d",
		subject.UserID,
		c.Query("start_date"),
		c.Query("end_date"),
		len(items),
	)
	response.Success(c, items)
}

type userFailedRequestItem struct {
	ID               int64     `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	RequestID        string    `json:"request_id"`
	ClientRequestID  string    `json:"client_request_id"`
	Platform         string    `json:"platform"`
	Model            string    `json:"model"`
	RequestedModel   string    `json:"requested_model"`
	StatusCode       int       `json:"status_code"`
	Phase            string    `json:"phase"`
	Source           string    `json:"error_source"`
	Owner            string    `json:"error_owner"`
	Message          string    `json:"message"`
	RequestPath      string    `json:"request_path"`
	InboundEndpoint  string    `json:"inbound_endpoint"`
	UpstreamEndpoint string    `json:"upstream_endpoint"`
	APIKeyID         *int64    `json:"api_key_id,omitempty"`
}

// ListFailedRequests lists gateway-level failed requests for the current user.
// GET /api/v1/usage/failed-requests
func (h *UsageHandler) ListFailedRequests(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.opsService == nil {
		page, pageSize := response.ParsePagination(c)
		response.Paginated(c, []userFailedRequestItem{}, 0, page, pageSize)
		return
	}

	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}

	startTime, endTime, err := parseUsageQueryTimeRange(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var apiKeyID *int64
	if apiKeyIDStr := strings.TrimSpace(c.Query("api_key_id")); apiKeyIDStr != "" {
		id, err := strconv.ParseInt(apiKeyIDStr, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid api_key_id")
			return
		}
		apiKey, err := h.apiKeyService.GetByIDAllowDeleted(c.Request.Context(), id)
		if err != nil || apiKey == nil || apiKey.UserID != subject.UserID {
			response.NotFound(c, "API key not found")
			return
		}
		apiKeyID = &id
	}

	filter := &service.OpsErrorLogFilter{
		Page:      page,
		PageSize:  pageSize,
		UserID:    &subject.UserID,
		APIKeyID:  apiKeyID,
		Platform:  strings.TrimSpace(c.Query("platform")),
		StartTime: startTime,
		EndTime:   endTime,
		View:      "all",
	}
	result, err := h.opsService.GetErrorLogs(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]userFailedRequestItem, 0, len(result.Errors))
	for _, item := range result.Errors {
		if item == nil {
			continue
		}
		items = append(items, userFailedRequestFromOps(item))
	}
	response.Paginated(c, items, int64(result.Total), result.Page, result.PageSize)
}

func parseUsageQueryTimeRange(c *gin.Context) (*time.Time, *time.Time, error) {
	userTZ := c.Query("timezone")
	now := timezone.NowInUserLocation(userTZ)
	start := timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -6), userTZ)
	end := timezone.StartOfDayInUserLocation(now, userTZ).AddDate(0, 0, 1)

	if startDateStr := strings.TrimSpace(c.Query("start_date")); startDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			return nil, nil, err
		}
		start = t
	}
	if endDateStr := strings.TrimSpace(c.Query("end_date")); endDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			return nil, nil, err
		}
		end = t.AddDate(0, 0, 1)
	}
	if end.Before(start) {
		return nil, nil, fmt.Errorf("invalid date range: end_date must be on or after start_date")
	}
	return &start, &end, nil
}

func userFailedRequestFromOps(item *service.OpsErrorLog) userFailedRequestItem {
	return userFailedRequestItem{
		ID:               item.ID,
		CreatedAt:        item.CreatedAt,
		RequestID:        item.RequestID,
		ClientRequestID:  item.ClientRequestID,
		Platform:         item.Platform,
		Model:            item.Model,
		RequestedModel:   item.RequestedModel,
		StatusCode:       item.StatusCode,
		Phase:            item.Phase,
		Source:           item.Source,
		Owner:            item.Owner,
		Message:          truncateUserFailedRequestMessage(item.Message),
		RequestPath:      item.RequestPath,
		InboundEndpoint:  item.InboundEndpoint,
		UpstreamEndpoint: item.UpstreamEndpoint,
		APIKeyID:         item.APIKeyID,
	}
}

func truncateUserFailedRequestMessage(message string) string {
	message = strings.TrimSpace(message)
	const maxLen = 512
	if len(message) <= maxLen {
		return message
	}
	return message[:maxLen-3] + "..."
}

// GetByID handles getting a single usage record
// GET /api/v1/usage/:id
func (h *UsageHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	usageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid usage ID")
		return
	}

	record, err := h.usageService.GetByID(c.Request.Context(), usageID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 验证所有权
	if record.UserID != subject.UserID {
		response.Forbidden(c, "Not authorized to access this record")
		return
	}

	response.Success(c, dto.UsageLogFromService(record))
}

// GetRequestPreview handles getting a request preview for a single usage record.
// GET /api/v1/usage/:id/request-preview
func (h *UsageHandler) GetRequestPreview(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	usageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid usage ID")
		return
	}

	record, err := h.usageService.GetByID(c.Request.Context(), usageID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if record.UserID != subject.UserID {
		response.Forbidden(c, "Not authorized to access this record")
		return
	}

	preview, err := h.usageService.GetRequestPreview(c.Request.Context(), record)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.UsageRequestPreviewFromService(preview))
}

// Stats handles getting usage statistics
// GET /api/v1/usage/stats
func (h *UsageHandler) Stats(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var apiKeyID int64
	if apiKeyIDStr := c.Query("api_key_id"); apiKeyIDStr != "" {
		id, err := strconv.ParseInt(apiKeyIDStr, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid api_key_id")
			return
		}

		// [Security Fix] Verify API Key ownership to prevent horizontal privilege escalation
		apiKey, err := h.apiKeyService.GetByIDAllowDeleted(c.Request.Context(), id)
		if err != nil {
			response.NotFound(c, "API key not found")
			return
		}
		if apiKey.UserID != subject.UserID {
			response.Forbidden(c, "Not authorized to access this API key's statistics")
			return
		}

		apiKeyID = id
	}

	// 获取时间范围参数
	userTZ := c.Query("timezone") // Get user's timezone from request
	now := timezone.NowInUserLocation(userTZ)
	var startTime, endTime time.Time

	// 优先使用 start_date 和 end_date 参数
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr != "" && endDateStr != "" {
		// 使用自定义日期范围
		var err error
		startTime, err = timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid start_date format, use YYYY-MM-DD")
			return
		}
		endTime, err = timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid end_date format, use YYYY-MM-DD")
			return
		}
		// 与 SQL 条件 created_at < end 对齐，使用次日 00:00 作为上边界（DST-safe）。
		endTime = endTime.AddDate(0, 0, 1)
	} else {
		// 使用 period 参数
		period := c.DefaultQuery("period", "today")
		switch period {
		case "today":
			startTime = timezone.StartOfDayInUserLocation(now, userTZ)
		case "week":
			startTime = now.AddDate(0, 0, -7)
		case "month":
			startTime = now.AddDate(0, -1, 0)
		default:
			startTime = timezone.StartOfDayInUserLocation(now, userTZ)
		}
		endTime = now
	}

	todayStart := timezone.StartOfDayInUserLocation(now, userTZ)
	platform := strings.TrimSpace(c.Query("platform"))

	statsFilters := usagestats.UsageLogFilters{
		UserID:     subject.UserID,
		APIKeyID:   apiKeyID,
		Platform:   platform,
		StartTime:  &startTime,
		EndTime:    &endTime,
		TodayStart: &todayStart,
		TodayEnd:   &now,
	}

	stats, err := h.usageService.GetStatsWithFilters(c.Request.Context(), statsFilters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, stats)
}

// parseUserTimeRange parses start_date, end_date query parameters for user dashboard
// Uses user's timezone if provided, otherwise falls back to server timezone
func parseUserTimeRange(c *gin.Context) (time.Time, time.Time) {
	userTZ := c.Query("timezone") // Get user's timezone from request
	now := timezone.NowInUserLocation(userTZ)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var startTime, endTime time.Time

	if startDate != "" {
		if t, err := timezone.ParseInUserLocation("2006-01-02", startDate, userTZ); err == nil {
			startTime = t
		} else {
			startTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -7), userTZ)
		}
	} else {
		startTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, -7), userTZ)
	}

	if endDate != "" {
		if t, err := timezone.ParseInUserLocation("2006-01-02", endDate, userTZ); err == nil {
			endTime = t.Add(24 * time.Hour) // Include the end date
		} else {
			endTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, 1), userTZ)
		}
	} else {
		endTime = timezone.StartOfDayInUserLocation(now.AddDate(0, 0, 1), userTZ)
	}

	return startTime, endTime
}

// DashboardStats handles getting user dashboard statistics
// GET /api/v1/usage/dashboard/stats
func (h *UsageHandler) DashboardStats(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	stats, err := h.usageService.GetUserDashboardStats(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, stats)
}

// DashboardTrend handles getting user usage trend data
// GET /api/v1/usage/dashboard/trend
func (h *UsageHandler) DashboardTrend(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	startTime, endTime := parseUserTimeRange(c)
	granularity := c.DefaultQuery("granularity", "day")

	trend, err := h.usageService.GetUserUsageTrendByUserID(c.Request.Context(), subject.UserID, startTime, endTime, granularity)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"trend":       trend,
		"start_date":  startTime.Format("2006-01-02"),
		"end_date":    endTime.Add(-24 * time.Hour).Format("2006-01-02"),
		"granularity": granularity,
	})
}

// DashboardModels handles getting user model usage statistics
// GET /api/v1/usage/dashboard/models
func (h *UsageHandler) DashboardModels(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	startTime, endTime := parseUserTimeRange(c)

	stats, err := h.usageService.GetUserModelStats(c.Request.Context(), subject.UserID, startTime, endTime)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"models":     stats,
		"start_date": startTime.Format("2006-01-02"),
		"end_date":   endTime.Add(-24 * time.Hour).Format("2006-01-02"),
	})
}

// BatchAPIKeysUsageRequest represents the request for batch API keys usage
type BatchAPIKeysUsageRequest struct {
	APIKeyIDs []int64 `json:"api_key_ids" binding:"required"`
}

// DashboardAPIKeysUsage handles getting usage stats for user's own API keys
// POST /api/v1/usage/dashboard/api-keys-usage
func (h *UsageHandler) DashboardAPIKeysUsage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req BatchAPIKeysUsageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if len(req.APIKeyIDs) == 0 {
		response.Success(c, gin.H{"stats": map[string]any{}})
		return
	}

	// Limit the number of API key IDs to prevent SQL parameter overflow
	if len(req.APIKeyIDs) > 100 {
		response.BadRequest(c, "Too many API key IDs (maximum 100 allowed)")
		return
	}

	validAPIKeyIDs, err := h.apiKeyService.VerifyOwnership(c.Request.Context(), subject.UserID, req.APIKeyIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if len(validAPIKeyIDs) == 0 {
		response.Success(c, gin.H{"stats": map[string]any{}})
		return
	}

	stats, err := h.usageService.GetBatchAPIKeyUsageStats(c.Request.Context(), validAPIKeyIDs, time.Time{}, time.Time{})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"stats": stats})
}

// DashboardAPIKeyDailyUsage handles daily details for a user's own API key.
// GET /api/v1/usage/dashboard/api-keys/:id/daily
func (h *UsageHandler) DashboardAPIKeyDailyUsage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	apiKeyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || apiKeyID <= 0 {
		response.BadRequest(c, "Invalid API key ID")
		return
	}

	validAPIKeyIDs, err := h.apiKeyService.VerifyOwnership(c.Request.Context(), subject.UserID, []int64{apiKeyID})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if len(validAPIKeyIDs) == 0 {
		response.NotFound(c, "API key not found")
		return
	}

	startTime, endTime := parseUserTimeRange(c)
	if endTime.Before(startTime) {
		startTime = endTime.AddDate(0, 0, -30)
	}
	maxStart := endTime.AddDate(0, 0, -31)
	if startTime.Before(maxStart) {
		startTime = maxStart
	}

	daily, err := h.usageService.GetAPIKeyDailyUsageTrend(c.Request.Context(), apiKeyID, startTime, endTime)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"daily_details": daily,
		"start_date":    startTime.Format("2006-01-02"),
		"end_date":      endTime.Add(-24 * time.Hour).Format("2006-01-02"),
	})
}
