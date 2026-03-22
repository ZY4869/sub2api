package admin

import (
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
)

type OAuthHandler struct{ oauthService *service.OAuthService }

func NewOAuthHandler(oauthService *service.OAuthService) *OAuthHandler {
	return &OAuthHandler{oauthService: oauthService}
}

type AccountHandler struct {
	adminService              service.AdminService
	oauthService              *service.OAuthService
	openaiOAuthService        *service.OpenAIOAuthService
	copilotOAuthService       *service.CopilotOAuthService
	kiroOAuthService          *service.KiroOAuthService
	geminiOAuthService        *service.GeminiOAuthService
	antigravityOAuthService   *service.AntigravityOAuthService
	rateLimitService          *service.RateLimitService
	accountUsageService       *service.AccountUsageService
	accountTestService        *service.AccountTestService
	concurrencyService        *service.ConcurrencyService
	crsSyncService            *service.CRSSyncService
	sessionLimitCache         service.SessionLimitCache
	rpmCache                  service.RPMCache
	tokenCacheInvalidator     service.TokenCacheInvalidator
	accountModelImportService *service.AccountModelImportService
	modelRegistryService      *service.ModelRegistryService
}

func NewAccountHandler(adminService service.AdminService, oauthService *service.OAuthService, openaiOAuthService *service.OpenAIOAuthService, geminiOAuthService *service.GeminiOAuthService, antigravityOAuthService *service.AntigravityOAuthService, rateLimitService *service.RateLimitService, accountUsageService *service.AccountUsageService, accountTestService *service.AccountTestService, concurrencyService *service.ConcurrencyService, crsSyncService *service.CRSSyncService, sessionLimitCache service.SessionLimitCache, rpmCache service.RPMCache, tokenCacheInvalidator service.TokenCacheInvalidator) *AccountHandler {
	return &AccountHandler{adminService: adminService, oauthService: oauthService, openaiOAuthService: openaiOAuthService, geminiOAuthService: geminiOAuthService, antigravityOAuthService: antigravityOAuthService, rateLimitService: rateLimitService, accountUsageService: accountUsageService, accountTestService: accountTestService, concurrencyService: concurrencyService, crsSyncService: crsSyncService, sessionLimitCache: sessionLimitCache, rpmCache: rpmCache, tokenCacheInvalidator: tokenCacheInvalidator}
}
func (h *AccountHandler) SetAccountModelImportService(svc *service.AccountModelImportService) {
	h.accountModelImportService = svc
}
func (h *AccountHandler) SetModelRegistryService(modelRegistryService *service.ModelRegistryService) {
	h.modelRegistryService = modelRegistryService
	if h.accountTestService != nil {
		h.accountTestService.SetModelRegistryService(modelRegistryService)
	}
}
func (h *AccountHandler) SetCopilotOAuthService(copilotOAuthService *service.CopilotOAuthService) {
	h.copilotOAuthService = copilotOAuthService
}
func (h *AccountHandler) SetKiroOAuthService(kiroOAuthService *service.KiroOAuthService) {
	h.kiroOAuthService = kiroOAuthService
}

type CreateAccountRequest struct {
	Name                    string         `json:"name" binding:"required"`
	Notes                   *string        `json:"notes"`
	Platform                string         `json:"platform" binding:"required"`
	Type                    string         `json:"type" binding:"required,oneof=oauth setup-token apikey upstream"`
	Credentials             map[string]any `json:"credentials" binding:"required"`
	Extra                   map[string]any `json:"extra"`
	ProxyID                 *int64         `json:"proxy_id"`
	Concurrency             int            `json:"concurrency"`
	Priority                int            `json:"priority"`
	RateMultiplier          *float64       `json:"rate_multiplier"`
	LoadFactor              *int           `json:"load_factor"`
	GroupIDs                []int64        `json:"group_ids"`
	ExpiresAt               *int64         `json:"expires_at"`
	AutoPauseOnExpired      *bool          `json:"auto_pause_on_expired"`
	ConfirmMixedChannelRisk *bool          `json:"confirm_mixed_channel_risk"`
}
type ImportAccountModelsRequest struct {
	Trigger string `json:"trigger"`
}
type UpdateAccountRequest struct {
	Name                    string         `json:"name"`
	Notes                   *string        `json:"notes"`
	Type                    string         `json:"type" binding:"omitempty,oneof=oauth setup-token apikey upstream"`
	Credentials             map[string]any `json:"credentials"`
	Extra                   map[string]any `json:"extra"`
	ProxyID                 *int64         `json:"proxy_id"`
	Concurrency             *int           `json:"concurrency"`
	Priority                *int           `json:"priority"`
	RateMultiplier          *float64       `json:"rate_multiplier"`
	LoadFactor              *int           `json:"load_factor"`
	Status                  string         `json:"status" binding:"omitempty,oneof=active inactive error"`
	GroupIDs                *[]int64       `json:"group_ids"`
	ExpiresAt               *int64         `json:"expires_at"`
	AutoPauseOnExpired      *bool          `json:"auto_pause_on_expired"`
	ConfirmMixedChannelRisk *bool          `json:"confirm_mixed_channel_risk"`
}
type BulkUpdateAccountsRequest struct {
	AccountIDs              []int64        `json:"account_ids" binding:"required,min=1"`
	Name                    string         `json:"name"`
	ProxyID                 *int64         `json:"proxy_id"`
	Concurrency             *int           `json:"concurrency"`
	Priority                *int           `json:"priority"`
	RateMultiplier          *float64       `json:"rate_multiplier"`
	LoadFactor              *int           `json:"load_factor"`
	Status                  string         `json:"status" binding:"omitempty,oneof=active inactive error"`
	Schedulable             *bool          `json:"schedulable"`
	GroupIDs                *[]int64       `json:"group_ids"`
	Credentials             map[string]any `json:"credentials"`
	Extra                   map[string]any `json:"extra"`
	ConfirmMixedChannelRisk *bool          `json:"confirm_mixed_channel_risk"`
}
type CheckMixedChannelRequest struct {
	Platform  string  `json:"platform" binding:"required"`
	GroupIDs  []int64 `json:"group_ids"`
	AccountID *int64  `json:"account_id"`
}
type AccountWithConcurrency struct {
	*dto.Account
	CurrentConcurrency int      `json:"current_concurrency"`
	CurrentWindowCost  *float64 `json:"current_window_cost,omitempty"`
	ActiveSessions     *int     `json:"active_sessions,omitempty"`
	CurrentRPM         *int     `json:"current_rpm,omitempty"`
}

const accountListGroupUngroupedQueryValue = "ungrouped"

func (h *AccountHandler) CheckMixedChannel(c *gin.Context) {
	var req CheckMixedChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if len(req.GroupIDs) == 0 {
		response.Success(c, gin.H{"has_risk": false})
		return
	}
	accountID := int64(0)
	if req.AccountID != nil {
		accountID = *req.AccountID
	}
	err := h.adminService.CheckMixedChannelRisk(c.Request.Context(), accountID, req.Platform, req.GroupIDs)
	if err != nil {
		var mixedErr *service.MixedChannelError
		if errors.As(err, &mixedErr) {
			response.Success(c, gin.H{"has_risk": true, "error": "mixed_channel_warning", "message": mixedErr.Error(), "details": gin.H{"group_id": mixedErr.GroupID, "group_name": mixedErr.GroupName, "current_platform": mixedErr.CurrentPlatform, "other_platform": mixedErr.OtherPlatform}})
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"has_risk": false})
}
func (h *AccountHandler) Delete(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	err = h.adminService.DeleteAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Account deleted successfully"})
}
func (h *AccountHandler) BulkUpdate(c *gin.Context) {
	var req BulkUpdateAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		response.BadRequest(c, "rate_multiplier must be >= 0")
		return
	}
	sanitizeExtraBaseRPM(req.Extra)
	skipCheck := req.ConfirmMixedChannelRisk != nil && *req.ConfirmMixedChannelRisk
	hasUpdates := req.Name != "" || req.ProxyID != nil || req.Concurrency != nil || req.Priority != nil || req.RateMultiplier != nil || req.LoadFactor != nil || req.Status != "" || req.Schedulable != nil || req.GroupIDs != nil || len(req.Credentials) > 0 || len(req.Extra) > 0
	if !hasUpdates {
		response.BadRequest(c, "No updates provided")
		return
	}
	result, err := h.adminService.BulkUpdateAccounts(c.Request.Context(), &service.BulkUpdateAccountsInput{AccountIDs: req.AccountIDs, Name: req.Name, ProxyID: req.ProxyID, Concurrency: req.Concurrency, Priority: req.Priority, RateMultiplier: req.RateMultiplier, LoadFactor: req.LoadFactor, Status: req.Status, Schedulable: req.Schedulable, GroupIDs: req.GroupIDs, Credentials: req.Credentials, Extra: req.Extra, SkipMixedChannelCheck: skipCheck})
	if err != nil {
		var mixedErr *service.MixedChannelError
		if errors.As(err, &mixedErr) {
			c.JSON(409, gin.H{"error": "mixed_channel_warning", "message": mixedErr.Error()})
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
func (h *AccountHandler) GetAntigravityDefaultModelMapping(c *gin.Context) {
	response.Success(c, domain.DefaultAntigravityModelMapping)
}
func sanitizeExtraBaseRPM(extra map[string]any) {
	if extra == nil {
		return
	}
	raw, ok := extra["base_rpm"]
	if !ok {
		return
	}
	v := service.ParseExtraInt(raw)
	if v < 0 {
		v = 0
	} else if v > 10000 {
		v = 10000
	}
	extra["base_rpm"] = v
}
