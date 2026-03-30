package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ScheduledTestHandler handles admin scheduled-test-plan management.
type ScheduledTestHandler struct {
	scheduledTestSvc *service.ScheduledTestService
}

// NewScheduledTestHandler creates a new ScheduledTestHandler.
func NewScheduledTestHandler(scheduledTestSvc *service.ScheduledTestService) *ScheduledTestHandler {
	return &ScheduledTestHandler{scheduledTestSvc: scheduledTestSvc}
}

type createScheduledTestPlanRequest struct {
	AccountID              int64  `json:"account_id" binding:"required"`
	ModelID                string `json:"model_id"`
	Model                  string `json:"model"`
	ModelInputMode         string `json:"model_input_mode"`
	ManualModelID          string `json:"manual_model_id"`
	RequestAlias           string `json:"request_alias"`
	SourceProtocol         string `json:"source_protocol"`
	CronExpression         string `json:"cron_expression" binding:"required"`
	Enabled                *bool  `json:"enabled"`
	MaxResults             int    `json:"max_results"`
	AutoRecover            *bool  `json:"auto_recover"`
	NotifyPolicy           string `json:"notify_policy"`
	NotifyFailureThreshold int    `json:"notify_failure_threshold"`
	RetryIntervalMinutes   int    `json:"retry_interval_minutes"`
	MaxRetries             int    `json:"max_retries"`
}

type updateScheduledTestPlanRequest struct {
	ModelID                string  `json:"model_id"`
	Model                  string  `json:"model"`
	ModelInputMode         *string `json:"model_input_mode"`
	ManualModelID          *string `json:"manual_model_id"`
	RequestAlias           *string `json:"request_alias"`
	SourceProtocol         *string `json:"source_protocol"`
	CronExpression         string  `json:"cron_expression"`
	Enabled                *bool   `json:"enabled"`
	MaxResults             int     `json:"max_results"`
	AutoRecover            *bool   `json:"auto_recover"`
	NotifyPolicy           string  `json:"notify_policy"`
	NotifyFailureThreshold int     `json:"notify_failure_threshold"`
	RetryIntervalMinutes   int     `json:"retry_interval_minutes"`
	MaxRetries             int     `json:"max_retries"`
}

// ListByAccount GET /admin/accounts/:id/scheduled-test-plans
func (h *ScheduledTestHandler) ListByAccount(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid account id")
		return
	}

	plans, err := h.scheduledTestSvc.ListPlansByAccount(c.Request.Context(), accountID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, plans)
}

// Create POST /admin/scheduled-test-plans
func (h *ScheduledTestHandler) Create(c *gin.Context) {
	var req createScheduledTestPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	plan := &service.ScheduledTestPlan{
		AccountID:              req.AccountID,
		ModelID:                strings.TrimSpace(req.ModelID),
		ModelInputMode:         strings.TrimSpace(req.ModelInputMode),
		ManualModelID:          strings.TrimSpace(req.ManualModelID),
		RequestAlias:           strings.TrimSpace(req.RequestAlias),
		SourceProtocol:         strings.TrimSpace(req.SourceProtocol),
		CronExpression:         req.CronExpression,
		Enabled:                true,
		MaxResults:             req.MaxResults,
		NotifyPolicy:           strings.TrimSpace(req.NotifyPolicy),
		NotifyFailureThreshold: req.NotifyFailureThreshold,
		RetryIntervalMinutes:   req.RetryIntervalMinutes,
		MaxRetries:             req.MaxRetries,
	}
	if plan.ModelID == "" {
		plan.ModelID = strings.TrimSpace(req.Model)
	}
	if req.Enabled != nil {
		plan.Enabled = *req.Enabled
	}
	if req.AutoRecover != nil {
		plan.AutoRecover = *req.AutoRecover
	}

	created, err := h.scheduledTestSvc.CreatePlan(c.Request.Context(), plan)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, created)
}

// Update PUT /admin/scheduled-test-plans/:id
func (h *ScheduledTestHandler) Update(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}

	existing, err := h.scheduledTestSvc.GetPlan(c.Request.Context(), planID)
	if err != nil {
		response.NotFound(c, "plan not found")
		return
	}

	var req updateScheduledTestPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if strings.TrimSpace(req.ModelID) != "" || strings.TrimSpace(req.Model) != "" {
		existing.ModelID = strings.TrimSpace(req.ModelID)
		if existing.ModelID == "" {
			existing.ModelID = strings.TrimSpace(req.Model)
		}
	}
	if req.ModelInputMode != nil {
		existing.ModelInputMode = strings.TrimSpace(*req.ModelInputMode)
	}
	if req.ManualModelID != nil {
		existing.ManualModelID = strings.TrimSpace(*req.ManualModelID)
	}
	if req.RequestAlias != nil {
		existing.RequestAlias = strings.TrimSpace(*req.RequestAlias)
	}
	if req.SourceProtocol != nil {
		existing.SourceProtocol = strings.TrimSpace(*req.SourceProtocol)
	}
	if req.CronExpression != "" {
		existing.CronExpression = req.CronExpression
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.MaxResults > 0 {
		existing.MaxResults = req.MaxResults
	}
	if req.AutoRecover != nil {
		existing.AutoRecover = *req.AutoRecover
	}
	if req.NotifyPolicy != "" {
		existing.NotifyPolicy = strings.TrimSpace(req.NotifyPolicy)
	}
	if req.NotifyFailureThreshold > 0 {
		existing.NotifyFailureThreshold = req.NotifyFailureThreshold
	}
	if req.RetryIntervalMinutes > 0 {
		existing.RetryIntervalMinutes = req.RetryIntervalMinutes
	}
	if req.MaxRetries > 0 {
		existing.MaxRetries = req.MaxRetries
	}

	updated, err := h.scheduledTestSvc.UpdatePlan(c.Request.Context(), existing)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, updated)
}

// Delete DELETE /admin/scheduled-test-plans/:id
func (h *ScheduledTestHandler) Delete(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}

	if err := h.scheduledTestSvc.DeletePlan(c.Request.Context(), planID); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ListResults GET /admin/scheduled-test-plans/:id/results
func (h *ScheduledTestHandler) ListResults(c *gin.Context) {
	planID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid plan id")
		return
	}

	limit := 50
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	results, err := h.scheduledTestSvc.ListResults(c.Request.Context(), planID, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, results)
}
