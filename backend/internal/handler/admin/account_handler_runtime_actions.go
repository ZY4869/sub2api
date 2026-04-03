package admin

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type syncFromCRSRequest struct {
	BaseURL            string   `json:"base_url" binding:"required"`
	Username           string   `json:"username" binding:"required"`
	Password           string   `json:"password" binding:"required"`
	SyncProxies        bool     `json:"sync_proxies"`
	SelectedAccountIDs []string `json:"selected_account_ids"`
}

type batchTodayStatsRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required,min=1"`
}

type setSchedulableRequest struct {
	Schedulable bool `json:"schedulable"`
}

type accountTestRequest struct {
	ModelID        string `json:"model_id"`
	Model          string `json:"model"`
	ModelInputMode string `json:"model_input_mode"`
	ManualModelID  string `json:"manual_model_id"`
	RequestAlias   string `json:"request_alias"`
	Prompt         string `json:"prompt"`
	SourceProtocol string `json:"source_protocol"`
	TestMode       string `json:"test_mode"`
}

func (h *AccountHandler) PreviewFromCRS(c *gin.Context) {
	if h.crsSyncService == nil {
		response.Error(c, 500, "CRS sync service is not configured")
		return
	}

	var req syncFromCRSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.crsSyncService.PreviewFromCRS(c.Request.Context(), service.SyncFromCRSInput{
		BaseURL:            req.BaseURL,
		Username:           req.Username,
		Password:           req.Password,
		SyncProxies:        req.SyncProxies,
		SelectedAccountIDs: req.SelectedAccountIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

func (h *AccountHandler) SyncFromCRS(c *gin.Context) {
	if h.crsSyncService == nil {
		response.Error(c, 500, "CRS sync service is not configured")
		return
	}

	var req syncFromCRSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.crsSyncService.SyncFromCRS(c.Request.Context(), service.SyncFromCRSInput{
		BaseURL:            req.BaseURL,
		Username:           req.Username,
		Password:           req.Password,
		SyncProxies:        req.SyncProxies,
		SelectedAccountIDs: req.SelectedAccountIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

func (h *AccountHandler) Test(c *gin.Context) {
	if h.accountTestService == nil {
		response.Error(c, 500, "Account test service is not configured")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	var req accountTestRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = strings.TrimSpace(req.Model)
	}
	if strings.EqualFold(strings.TrimSpace(req.ModelInputMode), "manual") {
		modelID = strings.TrimSpace(req.ManualModelID)
	}
	if modelID == "" {
		modelID = strings.TrimSpace(c.Query("model"))
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = strings.TrimSpace(c.Query("prompt"))
	}
	sourceProtocol := strings.TrimSpace(req.SourceProtocol)
	if sourceProtocol == "" {
		sourceProtocol = strings.TrimSpace(c.Query("source_protocol"))
	}
	testMode := strings.TrimSpace(req.TestMode)
	if testMode == "" {
		testMode = strings.TrimSpace(c.Query("test_mode"))
	}
	if err := h.accountTestService.TestAccountConnection(c, accountID, modelID, prompt, sourceProtocol, testMode); err != nil {
		if !c.Writer.Written() {
			response.ErrorFrom(c, err)
		}
		return
	}
}

func (h *AccountHandler) RecoverState(c *gin.Context) {
	if h.rateLimitService == nil {
		response.Error(c, 500, "Rate limit service is not configured")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	_, err = h.rateLimitService.RecoverAccountState(c.Request.Context(), accountID, service.AccountRecoveryOptions{
		InvalidateToken: true,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
}

func (h *AccountHandler) GetStats(c *gin.Context) {
	if h.accountUsageService == nil {
		response.Error(c, 500, "Account usage service is not configured")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days <= 0 {
		response.BadRequest(c, "Invalid days")
		return
	}

	endTime := time.Now().UTC()
	startTime := endTime.AddDate(0, 0, -days)
	stats, err := h.accountUsageService.GetAccountUsageStats(c.Request.Context(), accountID, startTime, endTime)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, stats)
}

func (h *AccountHandler) ClearError(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	account, err := h.adminService.ClearAccountError(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if h.tokenCacheInvalidator != nil && account != nil && account.IsOAuth() {
		_ = h.tokenCacheInvalidator.InvalidateToken(c.Request.Context(), account)
	}

	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
}

func (h *AccountHandler) GetUsage(c *gin.Context) {
	if h.accountUsageService == nil {
		response.Error(c, 500, "Account usage service is not configured")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	force := parseBoolQueryWithDefault(c.Query("force"), false)
	source := strings.TrimSpace(strings.ToLower(c.Query("source")))

	var usage *service.UsageInfo
	switch source {
	case "", "active":
		usage, err = h.accountUsageService.GetUsage(c.Request.Context(), accountID, force)
	case "passive":
		usage, err = h.accountUsageService.GetPassiveUsage(c.Request.Context(), accountID)
	default:
		response.BadRequest(c, "Invalid source")
		return
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, usage)
}

func (h *AccountHandler) GetTodayStats(c *gin.Context) {
	if h.accountUsageService == nil {
		response.Error(c, 500, "Account usage service is not configured")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	stats, err := h.accountUsageService.GetTodayStats(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, stats)
}

func (h *AccountHandler) GetBatchTodayStats(c *gin.Context) {
	if h.accountUsageService == nil {
		response.Error(c, 500, "Account usage service is not configured")
		return
	}

	var req batchTodayStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	statsByAccount, err := h.accountUsageService.GetTodayStatsBatch(c.Request.Context(), req.AccountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	payload := make(map[string]*service.WindowStats, len(statsByAccount))
	for accountID, stats := range statsByAccount {
		payload[strconv.FormatInt(accountID, 10)] = stats
	}

	response.Success(c, gin.H{"stats": payload})
}

func (h *AccountHandler) ClearRateLimit(c *gin.Context) {
	if h.rateLimitService == nil {
		response.Error(c, 500, "Rate limit service is not configured")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	if err := h.rateLimitService.ClearRateLimit(c.Request.Context(), accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
}

func (h *AccountHandler) ResetQuota(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	if err := h.adminService.ResetAccountQuota(c.Request.Context(), accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
}

func (h *AccountHandler) GetTempUnschedulable(c *gin.Context) {
	if h.rateLimitService == nil {
		response.Error(c, 500, "Rate limit service is not configured")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	state, err := h.rateLimitService.GetTempUnschedStatus(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"active": state != nil,
		"state":  state,
	})
}

func (h *AccountHandler) ClearTempUnschedulable(c *gin.Context) {
	if h.rateLimitService == nil {
		response.Error(c, 500, "Rate limit service is not configured")
		return
	}

	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	if err := h.rateLimitService.ClearTempUnschedulable(c.Request.Context(), accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Temporary unschedulable status cleared"})
}

func (h *AccountHandler) SetSchedulable(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	var req setSchedulableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	account, err := h.adminService.SetAccountSchedulable(c.Request.Context(), accountID, req.Schedulable)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
}
