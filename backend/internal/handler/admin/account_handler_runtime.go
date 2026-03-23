package admin

import (
	"context"
	"fmt"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type TestAccountRequest struct {
	ModelID string `json:"model_id"`
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
}
type SyncFromCRSRequest struct {
	BaseURL            string   `json:"base_url" binding:"required"`
	Username           string   `json:"username" binding:"required"`
	Password           string   `json:"password" binding:"required"`
	SyncProxies        *bool    `json:"sync_proxies"`
	SelectedAccountIDs []string `json:"selected_account_ids"`
}
type PreviewFromCRSRequest struct {
	BaseURL  string `json:"base_url" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AccountHandler) Test(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	var req TestAccountRequest
	_ = c.ShouldBindJSON(&req)
	modelID := strings.TrimSpace(req.ModelID)
	if modelID == "" {
		modelID = strings.TrimSpace(req.Model)
	}
	if err := h.accountTestService.TestAccountConnection(c, accountID, modelID, req.Prompt); err != nil {
		return
	}
	if service.NormalizeAccountLifecycleInput(account.LifecycleState) == service.AccountLifecycleBlacklisted {
		if _, err := h.adminService.RestoreBlacklistedAccount(c.Request.Context(), accountID); err != nil {
			_ = c.Error(err)
		}
	}
	if h.rateLimitService != nil {
		if _, err := h.rateLimitService.RecoverAccountAfterSuccessfulTest(c.Request.Context(), accountID); err != nil {
			_ = c.Error(err)
		}
	}
}
func (h *AccountHandler) RecoverState(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if h.rateLimitService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Rate limit service unavailable")
		return
	}
	if _, err := h.rateLimitService.RecoverAccountState(c.Request.Context(), accountID, service.AccountRecoveryOptions{InvalidateToken: true}); err != nil {
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
func (h *AccountHandler) SyncFromCRS(c *gin.Context) {
	var req SyncFromCRSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	syncProxies := true
	if req.SyncProxies != nil {
		syncProxies = *req.SyncProxies
	}
	result, err := h.crsSyncService.SyncFromCRS(c.Request.Context(), service.SyncFromCRSInput{BaseURL: req.BaseURL, Username: req.Username, Password: req.Password, SyncProxies: syncProxies, SelectedAccountIDs: req.SelectedAccountIDs})
	if err != nil {
		response.InternalError(c, "CRS sync failed: "+err.Error())
		return
	}
	response.Success(c, result)
}
func (h *AccountHandler) PreviewFromCRS(c *gin.Context) {
	var req PreviewFromCRSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.crsSyncService.PreviewFromCRS(c.Request.Context(), service.SyncFromCRSInput{BaseURL: req.BaseURL, Username: req.Username, Password: req.Password})
	if err != nil {
		response.InternalError(c, "CRS preview failed: "+err.Error())
		return
	}
	response.Success(c, result)
}
func (h *AccountHandler) refreshSingleAccount(ctx context.Context, account *service.Account) (*service.Account, string, error) {
	if !account.IsOAuth() {
		return nil, "", infraerrors.BadRequest("NOT_OAUTH", "cannot refresh non-OAuth account")
	}
	var newCredentials map[string]any
	if account.Platform == service.PlatformCopilot {
		updatedAccount, err := refreshCopilotOAuthAccount(ctx, h.adminService, h.copilotOAuthService, account)
		if err != nil {
			return nil, "", err
		}
		if h.tokenCacheInvalidator != nil {
			if invalidateErr := h.tokenCacheInvalidator.InvalidateToken(ctx, updatedAccount); invalidateErr != nil {
				log.Printf("[WARN] Failed to invalidate token cache for account %d: %v", updatedAccount.ID, invalidateErr)
			}
		}
		return updatedAccount, "", nil
	} else if account.Platform == service.PlatformKiro {
		updatedAccount, err := refreshKiroOAuthAccount(ctx, h.adminService, h.kiroOAuthService, account)
		if err != nil {
			return nil, "", err
		}
		if h.tokenCacheInvalidator != nil {
			if invalidateErr := h.tokenCacheInvalidator.InvalidateToken(ctx, updatedAccount); invalidateErr != nil {
				log.Printf("[WARN] Failed to invalidate token cache for account %d: %v", updatedAccount.ID, invalidateErr)
			}
		}
		return updatedAccount, "", nil
	} else if account.Platform == service.PlatformOpenAI || account.Platform == service.PlatformSora {
		tokenInfo, err := h.openaiOAuthService.RefreshAccountToken(ctx, account)
		if err != nil {
			return nil, "", err
		}
		newCredentials = h.openaiOAuthService.BuildAccountCredentials(tokenInfo)
		for k, v := range account.Credentials {
			if _, exists := newCredentials[k]; !exists {
				newCredentials[k] = v
			}
		}
	} else if account.Platform == service.PlatformGemini {
		tokenInfo, err := h.geminiOAuthService.RefreshAccountToken(ctx, account)
		if err != nil {
			return nil, "", fmt.Errorf("failed to refresh credentials: %w", err)
		}
		newCredentials = h.geminiOAuthService.BuildAccountCredentials(tokenInfo)
		for k, v := range account.Credentials {
			if _, exists := newCredentials[k]; !exists {
				newCredentials[k] = v
			}
		}
	} else if account.Platform == service.PlatformAntigravity {
		tokenInfo, err := h.antigravityOAuthService.RefreshAccountToken(ctx, account)
		if err != nil {
			return nil, "", err
		}
		newCredentials = h.antigravityOAuthService.BuildAccountCredentials(tokenInfo)
		for k, v := range account.Credentials {
			if _, exists := newCredentials[k]; !exists {
				newCredentials[k] = v
			}
		}
		if newProjectID, _ := newCredentials["project_id"].(string); newProjectID == "" {
			if oldProjectID := strings.TrimSpace(account.GetCredential("project_id")); oldProjectID != "" {
				newCredentials["project_id"] = oldProjectID
			}
		}
		if tokenInfo.ProjectIDMissing {
			updatedAccount, updateErr := h.adminService.UpdateAccount(ctx, account.ID, &service.UpdateAccountInput{Credentials: newCredentials})
			if updateErr != nil {
				return nil, "", fmt.Errorf("failed to update credentials: %w", updateErr)
			}
			return updatedAccount, "missing_project_id_temporary", nil
		}
		if account.Status == service.StatusError && strings.Contains(account.ErrorMessage, "missing_project_id:") {
			if _, clearErr := h.adminService.ClearAccountError(ctx, account.ID); clearErr != nil {
				return nil, "", fmt.Errorf("failed to clear account error: %w", clearErr)
			}
		}
	} else {
		tokenInfo, err := h.oauthService.RefreshAccountToken(ctx, account)
		if err != nil {
			return nil, "", err
		}
		newCredentials = make(map[string]any)
		for k, v := range account.Credentials {
			newCredentials[k] = v
		}
		newCredentials["access_token"] = tokenInfo.AccessToken
		newCredentials["token_type"] = tokenInfo.TokenType
		newCredentials["expires_in"] = strconv.FormatInt(tokenInfo.ExpiresIn, 10)
		newCredentials["expires_at"] = strconv.FormatInt(tokenInfo.ExpiresAt, 10)
		if strings.TrimSpace(tokenInfo.RefreshToken) != "" {
			newCredentials["refresh_token"] = tokenInfo.RefreshToken
		}
		if strings.TrimSpace(tokenInfo.Scope) != "" {
			newCredentials["scope"] = tokenInfo.Scope
		}
	}
	updatedAccount, err := h.adminService.UpdateAccount(ctx, account.ID, &service.UpdateAccountInput{Credentials: newCredentials})
	if err != nil {
		return nil, "", err
	}
	if h.tokenCacheInvalidator != nil {
		if invalidateErr := h.tokenCacheInvalidator.InvalidateToken(ctx, updatedAccount); invalidateErr != nil {
			log.Printf("[WARN] Failed to invalidate token cache for account %d: %v", updatedAccount.ID, invalidateErr)
		}
	}
	return updatedAccount, "", nil
}
func (h *AccountHandler) Refresh(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.NotFound(c, "Account not found")
		return
	}
	updatedAccount, warning, err := h.refreshSingleAccount(c.Request.Context(), account)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if warning == "missing_project_id_temporary" {
		response.Success(c, gin.H{"message": "Token refreshed successfully, but project_id could not be retrieved (will retry automatically)", "warning": "missing_project_id_temporary"})
		return
	}
	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), updatedAccount))
}
func (h *AccountHandler) GetStats(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}
	now := timezone.Now()
	endTime := timezone.StartOfDay(now.AddDate(0, 0, 1))
	startTime := timezone.StartOfDay(now.AddDate(0, 0, -days+1))
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
	if h.tokenCacheInvalidator != nil && account.IsOAuth() {
		if invalidateErr := h.tokenCacheInvalidator.InvalidateToken(c.Request.Context(), account); invalidateErr != nil {
			log.Printf("[WARN] Failed to invalidate token cache for account %d: %v", accountID, invalidateErr)
		}
	}
	response.Success(c, h.buildAccountResponseWithRuntime(c.Request.Context(), account))
}
func (h *AccountHandler) GetUsage(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	force := false
	if raw := strings.TrimSpace(c.Query("force")); raw != "" {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			response.BadRequest(c, "Invalid force flag")
			return
		}
		force = parsed
	}
	source := strings.TrimSpace(c.DefaultQuery("source", "active"))
	var usage *service.UsageInfo
	switch source {
	case "", "active":
		usage, err = h.accountUsageService.GetUsage(c.Request.Context(), accountID, force)
	case "passive":
		usage, err = h.accountUsageService.GetPassiveUsage(c.Request.Context(), accountID)
	default:
		response.BadRequest(c, "Invalid usage source")
		return
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, usage)
}
func (h *AccountHandler) ClearRateLimit(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	err = h.rateLimitService.ClearRateLimit(c.Request.Context(), accountID)
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
func (h *AccountHandler) ResetQuota(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if err := h.adminService.ResetAccountQuota(c.Request.Context(), accountID); err != nil {
		response.InternalError(c, "Failed to reset account quota: "+err.Error())
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
	if state == nil || state.UntilUnix <= time.Now().Unix() {
		response.Success(c, gin.H{"active": false})
		return
	}
	response.Success(c, gin.H{"active": true, "state": state})
}
func (h *AccountHandler) ClearTempUnschedulable(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if err := h.rateLimitService.ClearTempUnschedulable(c.Request.Context(), accountID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Temp unschedulable cleared successfully"})
}
func (h *AccountHandler) GetTodayStats(c *gin.Context) {
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

type BatchTodayStatsRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required"`
}

func (h *AccountHandler) GetBatchTodayStats(c *gin.Context) {
	var req BatchTodayStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	accountIDs := normalizeInt64IDList(req.AccountIDs)
	if len(accountIDs) == 0 {
		response.Success(c, gin.H{"stats": map[string]any{}})
		return
	}
	cacheKey := buildAccountTodayStatsBatchCacheKey(accountIDs)
	if cached, ok := accountTodayStatsBatchCache.Get(cacheKey); ok {
		if cached.ETag != "" {
			c.Header("ETag", cached.ETag)
			c.Header("Vary", "If-None-Match")
			if ifNoneMatchMatched(c.GetHeader("If-None-Match"), cached.ETag) {
				c.Status(http.StatusNotModified)
				return
			}
		}
		c.Header("X-Snapshot-Cache", "hit")
		response.Success(c, cached.Payload)
		return
	}
	stats, err := h.accountUsageService.GetTodayStatsBatch(c.Request.Context(), accountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	payload := gin.H{"stats": stats}
	cached := accountTodayStatsBatchCache.Set(cacheKey, payload)
	if cached.ETag != "" {
		c.Header("ETag", cached.ETag)
		c.Header("Vary", "If-None-Match")
	}
	c.Header("X-Snapshot-Cache", "miss")
	response.Success(c, payload)
}

type SetSchedulableRequest struct {
	Schedulable bool `json:"schedulable"`
}

func (h *AccountHandler) SetSchedulable(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req SetSchedulableRequest
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
func (h *AccountHandler) RefreshTier(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	ctx := c.Request.Context()
	account, err := h.adminService.GetAccount(ctx, accountID)
	if err != nil {
		response.NotFound(c, "Account not found")
		return
	}
	if account.Platform != service.PlatformGemini || account.Type != service.AccountTypeOAuth {
		response.BadRequest(c, "Only Gemini OAuth accounts support tier refresh")
		return
	}
	oauthType, _ := account.Credentials["oauth_type"].(string)
	if oauthType != "google_one" {
		response.BadRequest(c, "Only google_one OAuth accounts support tier refresh")
		return
	}
	tierID, extra, creds, err := h.geminiOAuthService.RefreshAccountGoogleOneTier(ctx, account)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	_, updateErr := h.adminService.UpdateAccount(ctx, accountID, &service.UpdateAccountInput{Credentials: creds, Extra: extra})
	if updateErr != nil {
		response.ErrorFrom(c, updateErr)
		return
	}
	response.Success(c, gin.H{"tier_id": tierID, "storage_info": extra, "drive_storage_limit": extra["drive_storage_limit"], "drive_storage_usage": extra["drive_storage_usage"], "updated_at": extra["drive_tier_updated_at"]})
}

type BatchRefreshTierRequest struct {
	AccountIDs []int64 `json:"account_ids"`
}

func (h *AccountHandler) BatchRefreshTier(c *gin.Context) {
	var req BatchRefreshTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = BatchRefreshTierRequest{}
	}
	ctx := c.Request.Context()
	accounts := make([]*service.Account, 0)
	if len(req.AccountIDs) == 0 {
		allAccounts, _, err := h.adminService.ListAccounts(ctx, 1, 10000, "gemini", "oauth", "", "", 0, service.AccountLifecycleNormal)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		for i := range allAccounts {
			acc := &allAccounts[i]
			oauthType, _ := acc.Credentials["oauth_type"].(string)
			if oauthType == "google_one" {
				accounts = append(accounts, acc)
			}
		}
	} else {
		fetched, err := h.adminService.GetAccountsByIDs(ctx, req.AccountIDs)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		for _, acc := range fetched {
			if acc == nil {
				continue
			}
			if acc.Platform != service.PlatformGemini || acc.Type != service.AccountTypeOAuth {
				continue
			}
			oauthType, _ := acc.Credentials["oauth_type"].(string)
			if oauthType != "google_one" {
				continue
			}
			accounts = append(accounts, acc)
		}
	}
	const maxConcurrency = 10
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)
	var mu sync.Mutex
	var successCount, failedCount int
	var errors []gin.H
	for _, account := range accounts {
		acc := account
		g.Go(func() error {
			_, extra, creds, err := h.geminiOAuthService.RefreshAccountGoogleOneTier(gctx, acc)
			if err != nil {
				mu.Lock()
				failedCount++
				errors = append(errors, gin.H{"account_id": acc.ID, "error": err.Error()})
				mu.Unlock()
				return nil
			}
			_, updateErr := h.adminService.UpdateAccount(gctx, acc.ID, &service.UpdateAccountInput{Credentials: creds, Extra: extra})
			mu.Lock()
			if updateErr != nil {
				failedCount++
				errors = append(errors, gin.H{"account_id": acc.ID, "error": updateErr.Error()})
			} else {
				successCount++
			}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	results := gin.H{"total": len(accounts), "success": successCount, "failed": failedCount, "errors": errors}
	response.Success(c, results)
}
