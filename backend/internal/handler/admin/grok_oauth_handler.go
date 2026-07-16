package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type GrokOAuthHandler struct {
	grokOAuthService *service.GrokOAuthService
	adminService     service.AdminService
	quotaService     *service.GrokQuotaService
	reconciler       service.GrokOAuthReconciler
	modelRegistry    *service.ModelRegistryService
}

func NewGrokOAuthHandler(grokOAuthService *service.GrokOAuthService, adminService service.AdminService) *GrokOAuthHandler {
	return &GrokOAuthHandler{
		grokOAuthService: grokOAuthService,
		adminService:     adminService,
	}
}

func (h *GrokOAuthHandler) SetGrokAdminDependencies(quotaService *service.GrokQuotaService, reconciler service.GrokOAuthReconciler) {
	if h == nil {
		return
	}
	h.quotaService = quotaService
	h.reconciler = reconciler
}

func (h *GrokOAuthHandler) SetModelRegistryService(modelRegistry *service.ModelRegistryService) {
	if h == nil {
		return
	}
	h.modelRegistry = modelRegistry
}

type GrokGenerateAuthURLRequest struct {
	ProxyID     *int64 `json:"proxy_id"`
	RedirectURI string `json:"redirect_uri"`
	BaseURL     string `json:"base_url"`
}

func (h *GrokOAuthHandler) GenerateAuthURL(c *gin.Context) {
	var req GrokGenerateAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = GrokGenerateAuthURLRequest{}
	}
	result, err := h.grokOAuthService.GenerateAuthURL(c.Request.Context(), &service.GrokGenerateAuthURLInput{
		ProxyID:     req.ProxyID,
		RedirectURI: req.RedirectURI,
		BaseURL:     req.BaseURL,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type GrokExchangeCodeRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state"`
	RedirectURI string `json:"redirect_uri"`
	ProxyID     *int64 `json:"proxy_id"`
}

func (h *GrokOAuthHandler) ExchangeCode(c *gin.Context) {
	var req GrokExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	tokenInfo, err := h.grokOAuthService.ExchangeCode(c.Request.Context(), &service.GrokExchangeCodeInput{
		SessionID:   req.SessionID,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: req.RedirectURI,
		ProxyID:     req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tokenInfo)
}

type GrokStartDeviceFlowRequest struct {
	ProxyID *int64 `json:"proxy_id"`
	BaseURL string `json:"base_url"`
}

func (h *GrokOAuthHandler) StartDeviceFlow(c *gin.Context) {
	var req GrokStartDeviceFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = GrokStartDeviceFlowRequest{}
	}
	result, err := h.grokOAuthService.StartDeviceFlow(c.Request.Context(), &service.GrokStartDeviceFlowInput{
		ProxyID: req.ProxyID,
		BaseURL: req.BaseURL,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type GrokPollDeviceTokenRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	ProxyID   *int64 `json:"proxy_id"`
}

func (h *GrokOAuthHandler) PollDeviceToken(c *gin.Context) {
	var req GrokPollDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.grokOAuthService.PollDeviceToken(c.Request.Context(), &service.GrokPollDeviceTokenInput{
		SessionID: req.SessionID,
		ProxyID:   req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type CreateGrokAccountFromOAuthRequest struct {
	SessionID               string   `json:"session_id" binding:"required"`
	Code                    string   `json:"code" binding:"required"`
	State                   string   `json:"state"`
	RedirectURI             string   `json:"redirect_uri"`
	ProxyID                 *int64   `json:"proxy_id"`
	Name                    string   `json:"name"`
	Notes                   *string  `json:"notes"`
	Concurrency             int      `json:"concurrency"`
	Priority                int      `json:"priority"`
	RateMultiplier          *float64 `json:"rate_multiplier"`
	LoadFactor              *int     `json:"load_factor"`
	GroupIDs                []int64  `json:"group_ids"`
	ExpiresAt               *int64   `json:"expires_at"`
	AutoPauseOnExpired      *bool    `json:"auto_pause_on_expired"`
	AutoRenewEnabled        *bool    `json:"auto_renew_enabled"`
	AutoRenewPeriod         *string  `json:"auto_renew_period"`
	ConfirmMixedChannelRisk *bool    `json:"confirm_mixed_channel_risk"`
}

func (h *GrokOAuthHandler) CreateAccountFromOAuth(c *gin.Context) {
	var req CreateGrokAccountFromOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	tokenInfo, err := h.grokOAuthService.ExchangeCode(c.Request.Context(), &service.GrokExchangeCodeInput{
		SessionID:   req.SessionID,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: req.RedirectURI,
		ProxyID:     req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	account, err := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{
		Name:                  resolveGrokOAuthAccountName(req.Name, tokenInfo),
		Notes:                 req.Notes,
		Platform:              service.PlatformGrok,
		Type:                  service.AccountTypeOAuth,
		Credentials:           h.grokOAuthService.BuildAccountCredentials(tokenInfo),
		Extra:                 h.grokOAuthService.BuildAccountExtra(tokenInfo),
		ProxyID:               req.ProxyID,
		Concurrency:           req.Concurrency,
		Priority:              req.Priority,
		RateMultiplier:        req.RateMultiplier,
		LoadFactor:            req.LoadFactor,
		GroupIDs:              req.GroupIDs,
		ExpiresAt:             req.ExpiresAt,
		AutoPauseOnExpired:    req.AutoPauseOnExpired,
		AutoRenewEnabled:      req.AutoRenewEnabled,
		AutoRenewPeriod:       req.AutoRenewPeriod,
		SkipMixedChannelCheck: req.ConfirmMixedChannelRisk != nil && *req.ConfirmMixedChannelRisk,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := h.ensureGrokBuildModelScopePersisted(c.Request.Context(), account); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AccountFromService(account))
}

type ReauthorizeGrokAccountFromOAuthRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state"`
	RedirectURI string `json:"redirect_uri"`
	ProxyID     *int64 `json:"proxy_id"`
}

func (h *GrokOAuthHandler) ReauthorizeAccountFromOAuth(c *gin.Context) {
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
	if account.Platform != service.PlatformGrok {
		response.BadRequest(c, "Account platform does not match Grok endpoint")
		return
	}

	var req ReauthorizeGrokAccountFromOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	tokenInfo, err := h.grokOAuthService.ExchangeCode(c.Request.Context(), &service.GrokExchangeCodeInput{
		SessionID:   req.SessionID,
		Code:        req.Code,
		State:       req.State,
		RedirectURI: req.RedirectURI,
		ProxyID:     req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	credentials := service.MergeCredentials(account.Credentials, h.grokOAuthService.BuildAccountCredentials(tokenInfo))
	if account.IsGrokSSO() {
		delete(credentials, "sso_token")
	}
	extraUpdates := h.grokOAuthService.BuildAccountExtra(tokenInfo)
	if scopeUpdates, _, _ := service.EnsureGrokBuildModelScopeUpdates(c.Request.Context(), account, h.modelRegistry, time.Now().UTC()); len(scopeUpdates) > 0 {
		extraUpdates = service.MergeStringAnyMap(extraUpdates, scopeUpdates)
	} else {
		delete(extraUpdates, "model_scope_v2")
	}
	extra := service.MergeStringAnyMap(account.Extra, extraUpdates)
	if _, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{
		Type:        service.AccountTypeOAuth,
		Credentials: credentials,
		Extra:       extra,
	}); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	updated, err := h.adminService.ClearAccountError(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := h.ensureGrokBuildModelScopePersisted(c.Request.Context(), updated); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AccountFromService(updated))
}

func (h *GrokOAuthHandler) RefreshAccount(c *gin.Context) {
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
	tokenInfo, err := h.grokOAuthService.RefreshAccountToken(c.Request.Context(), account)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	credentials := service.MergeCredentials(account.Credentials, h.grokOAuthService.BuildAccountCredentials(tokenInfo))
	if _, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{
		Credentials: credentials,
	}); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	updated, err := h.adminService.ClearAccountError(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := h.ensureGrokBuildModelScopePersisted(c.Request.Context(), updated); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.AccountFromService(updated))
}

func (h *GrokOAuthHandler) QueryQuota(c *gin.Context) {
	if h.quotaService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("GROK_QUOTA_UNAVAILABLE", "Grok quota service is unavailable"))
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	result, err := h.quotaService.QueryQuota(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *GrokOAuthHandler) ResetQuota(c *gin.Context) {
	if h.quotaService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("GROK_QUOTA_UNAVAILABLE", "Grok quota service is unavailable"))
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	result, err := h.quotaService.ResetQuota(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type GrokOAuthReconcileRequest struct {
	DryRun               *bool  `json:"dry_run"`
	Apply                bool   `json:"apply"`
	AfterID              int64  `json:"after_id"`
	Limit                int    `json:"limit"`
	RefreshWindowSeconds *int64 `json:"refresh_window_seconds"`
}

func (h *GrokOAuthHandler) Reconcile(c *gin.Context) {
	if h.reconciler == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("GROK_OAUTH_RECONCILE_UNAVAILABLE", "Grok OAuth reconciliation is unavailable"))
		return
	}
	var req GrokOAuthReconcileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = GrokOAuthReconcileRequest{}
	}
	dryRun := true
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}
	refreshWindow := time.Duration(0)
	if req.RefreshWindowSeconds != nil {
		refreshWindow = time.Duration(*req.RefreshWindowSeconds) * time.Second
	}
	result, err := h.reconciler.ReconcileGrokOAuth(c.Request.Context(), service.GrokOAuthReconcileInput{
		DryRun:        dryRun,
		Apply:         req.Apply,
		AfterID:       req.AfterID,
		Limit:         req.Limit,
		RefreshWindow: refreshWindow,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *GrokOAuthHandler) RuntimeSanity(c *gin.Context) {
	if h.grokOAuthService == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("GROK_OAUTH_UNAVAILABLE", "Grok OAuth service is unavailable"))
		return
	}
	response.Success(c, h.grokOAuthService.RuntimeSanity())
}

func resolveGrokOAuthAccountName(raw string, tokenInfo *service.GrokTokenInfo) string {
	if name := strings.TrimSpace(raw); name != "" {
		return name
	}
	if tokenInfo != nil {
		for _, candidate := range []string{tokenInfo.Email, tokenInfo.Name, tokenInfo.Subject} {
			if trimmed := strings.TrimSpace(candidate); trimmed != "" {
				return trimmed
			}
		}
	}
	return "Grok OAuth Account"
}

func (h *GrokOAuthHandler) ensureGrokBuildModelScopePersisted(ctx context.Context, account *service.Account) error {
	if account == nil || !account.IsGrokOAuth() {
		return nil
	}
	_, err := h.adminService.EnsureGrokBuildModelScope(ctx, account, h.modelRegistry)
	return err
}
