package admin

import (
	"context"
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

type accountTestServicePort interface {
	SetModelRegistryService(modelRegistryService *service.ModelRegistryService)
	TestAccountConnection(c *gin.Context, accountID int64, modelID string, prompt string, sourceProtocol string, targetProvider string, targetModelID string, testMode string) error
	RunTestBackgroundDetailed(ctx context.Context, input service.ScheduledTestExecutionInput) (*service.BackgroundAccountTestResult, error)
	RunTestBackground(ctx context.Context, input service.ScheduledTestExecutionInput) (*service.ScheduledTestResult, error)
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
	accountTestService        accountTestServicePort
	concurrencyService        *service.ConcurrencyService
	crsSyncService            *service.CRSSyncService
	sessionLimitCache         service.SessionLimitCache
	rpmCache                  service.RPMCache
	tokenCacheInvalidator     service.TokenCacheInvalidator
	accountModelImportService *service.AccountModelImportService
	accountModelDiagnostics   *service.AccountModelDiagnosticsService
	modelRegistryService      *service.ModelRegistryService
	opsService                *service.OpsService
}

func NewAccountHandler(adminService service.AdminService, oauthService *service.OAuthService, openaiOAuthService *service.OpenAIOAuthService, geminiOAuthService *service.GeminiOAuthService, antigravityOAuthService *service.AntigravityOAuthService, rateLimitService *service.RateLimitService, accountUsageService *service.AccountUsageService, accountTestService accountTestServicePort, concurrencyService *service.ConcurrencyService, crsSyncService *service.CRSSyncService, sessionLimitCache service.SessionLimitCache, rpmCache service.RPMCache, tokenCacheInvalidator service.TokenCacheInvalidator) *AccountHandler {
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
func (h *AccountHandler) SetAccountModelDiagnosticsService(svc *service.AccountModelDiagnosticsService) {
	h.accountModelDiagnostics = svc
}
func (h *AccountHandler) SetCopilotOAuthService(copilotOAuthService *service.CopilotOAuthService) {
	h.copilotOAuthService = copilotOAuthService
}
func (h *AccountHandler) SetKiroOAuthService(kiroOAuthService *service.KiroOAuthService) {
	h.kiroOAuthService = kiroOAuthService
}

func (h *AccountHandler) SetOpsService(opsService *service.OpsService) {
	h.opsService = opsService
}

type CreateAccountRequest struct {
	Name                    string         `json:"name" binding:"required"`
	Notes                   *string        `json:"notes"`
	Platform                string         `json:"platform" binding:"required"`
	GatewayProtocol         string         `json:"gateway_protocol" binding:"omitempty,oneof=openai anthropic gemini mixed"`
	Type                    string         `json:"type" binding:"required,oneof=oauth setup-token apikey upstream sso"`
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
	Trigger string   `json:"trigger"`
	Models  []string `json:"models"`
}
type UpdateAccountRequest struct {
	Name                    string         `json:"name"`
	Notes                   *string        `json:"notes"`
	GatewayProtocol         string         `json:"gateway_protocol" binding:"omitempty,oneof=openai anthropic gemini mixed"`
	Type                    string         `json:"type" binding:"omitempty,oneof=oauth setup-token apikey upstream sso"`
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
	// Either AccountIDs or Filters must be provided. The handler enforces this.
	AccountIDs              []int64                    `json:"account_ids"`
	Filters                 *BulkUpdateAccountsFilters `json:"filters,omitempty"`
	Name                    string                     `json:"name"`
	ProxyID                 *int64                     `json:"proxy_id"`
	Concurrency             *int                       `json:"concurrency"`
	Priority                *int                       `json:"priority"`
	RateMultiplier          *float64                   `json:"rate_multiplier"`
	LoadFactor              *int                       `json:"load_factor"`
	Status                  string                     `json:"status" binding:"omitempty,oneof=active inactive error"`
	Schedulable             *bool                      `json:"schedulable"`
	GroupIDs                *[]int64                   `json:"group_ids"`
	Credentials             map[string]any             `json:"credentials"`
	Extra                   map[string]any             `json:"extra"`
	ConfirmMixedChannelRisk *bool                      `json:"confirm_mixed_channel_risk"`
}

// BulkUpdateAccountsFilters reuses the same semantics as the admin accounts list endpoint.
// All fields are optional; when omitted, the default list behavior applies.
type BulkUpdateAccountsFilters struct {
	Platform      string `json:"platform,omitempty"`
	Type          string `json:"type,omitempty"`
	Status        string `json:"status,omitempty"`
	Group         string `json:"group,omitempty"` // numeric group id or "ungrouped"
	Search        string `json:"search,omitempty"`
	Lifecycle     string `json:"lifecycle,omitempty"`
	PrivacyMode   string `json:"privacy_mode,omitempty"`
	LimitedView   string `json:"limited_view,omitempty"`
	LimitedReason string `json:"limited_reason,omitempty"`
	RuntimeView   string `json:"runtime_view,omitempty"`
}
type BlacklistAccountRequest struct {
	Source   string                          `json:"source"`
	Feedback *service.BlacklistFeedbackInput `json:"feedback"`
}
type CheckMixedChannelRequest struct {
	Platform        string  `json:"platform" binding:"required"`
	GatewayProtocol string  `json:"gateway_protocol" binding:"omitempty,oneof=openai anthropic gemini mixed"`
	GroupIDs        []int64 `json:"group_ids"`
	AccountID       *int64  `json:"account_id"`
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
		response.BadRequestKey(c, "admin.account.invalid_request", "Invalid request: %s", err.Error())
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
	platform := service.RoutingPlatformFromValues(req.Platform, withGatewayProtocol(req.Platform, "", nil, req.GatewayProtocol, ""))
	err := h.adminService.CheckMixedChannelRisk(c.Request.Context(), accountID, platform, req.GroupIDs)
	if err != nil {
		var mixedErr *service.MixedChannelError
		if errors.As(err, &mixedErr) {
			response.Success(c, gin.H{"has_risk": true, "error": "mixed_channel_warning", "message": mixedChannelWarningMessage(c, mixedErr), "details": gin.H{"group_id": mixedErr.GroupID, "group_name": mixedErr.GroupName, "current_platform": mixedErr.CurrentPlatform, "other_platform": mixedErr.OtherPlatform}})
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
		response.BadRequestKey(c, "admin.account.invalid_id", "Invalid account ID")
		return
	}
	err = h.adminService.DeleteAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": response.LocalizedMessage(c, "admin.account.deleted", "Account deleted successfully")})
}
func (h *AccountHandler) BulkUpdate(c *gin.Context) {
	var req BulkUpdateAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestKey(c, "admin.account.invalid_request", "Invalid request: %s", err.Error())
		return
	}
	if req.RateMultiplier != nil && *req.RateMultiplier < 0 {
		response.BadRequestKey(c, "admin.account.rate_multiplier_invalid", "rate_multiplier must be >= 0")
		return
	}
	normalizedStatus := service.NormalizeAdminAccountStatusInput(req.Status)
	sanitizeExtraBaseRPM(req.Extra)
	skipCheck := req.ConfirmMixedChannelRisk != nil && *req.ConfirmMixedChannelRisk
	hasUpdates := req.Name != "" || req.ProxyID != nil || req.Concurrency != nil || req.Priority != nil || req.RateMultiplier != nil || req.LoadFactor != nil || normalizedStatus != "" || req.Schedulable != nil || req.GroupIDs != nil || len(req.Credentials) > 0 || len(req.Extra) > 0
	if !hasUpdates {
		response.BadRequestKey(c, "admin.account.no_updates", "No updates provided")
		return
	}

	// Target selection: either explicit account_ids or filters (mutually exclusive).
	if len(req.AccountIDs) > 0 && req.Filters != nil {
		response.BadRequestKey(c, "admin.account.invalid_request", "Provide either account_ids or filters, not both")
		return
	}
	if len(req.AccountIDs) == 0 {
		if req.Filters == nil {
			response.BadRequestKey(c, "admin.account.invalid_request", "account_ids or filters is required")
			return
		}
		targetIDs, err := h.resolveBulkUpdateAccountIDsByFilters(c, req.Filters)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if len(targetIDs) == 0 {
			response.BadRequestKey(c, "admin.account.invalid_request", "No accounts matched the filters")
			return
		}
		req.AccountIDs = targetIDs
	}

	result, err := h.adminService.BulkUpdateAccounts(c.Request.Context(), &service.BulkUpdateAccountsInput{AccountIDs: req.AccountIDs, Name: req.Name, ProxyID: req.ProxyID, Concurrency: req.Concurrency, Priority: req.Priority, RateMultiplier: req.RateMultiplier, LoadFactor: req.LoadFactor, Status: normalizedStatus, Schedulable: req.Schedulable, GroupIDs: req.GroupIDs, Credentials: req.Credentials, Extra: req.Extra, SkipMixedChannelCheck: skipCheck})
	if err != nil {
		var mixedErr *service.MixedChannelError
		if errors.As(err, &mixedErr) {
			c.JSON(409, gin.H{"error": "mixed_channel_warning", "message": mixedChannelWarningMessage(c, mixedErr)})
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
