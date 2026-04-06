package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// OpenAIOAuthHandler handles OpenAI OAuth-related operations
type OpenAIOAuthHandler struct {
	openaiOAuthService  *service.OpenAIOAuthService
	copilotOAuthService *service.CopilotOAuthService
	adminService        service.AdminService
}

func oauthPlatformFromPath(c *gin.Context) string {
	return service.PlatformOpenAI
}

// NewOpenAIOAuthHandler creates a new OpenAI OAuth handler
func NewOpenAIOAuthHandler(openaiOAuthService *service.OpenAIOAuthService, adminService service.AdminService) *OpenAIOAuthHandler {
	return &OpenAIOAuthHandler{
		openaiOAuthService: openaiOAuthService,
		adminService:       adminService,
	}
}

func (h *OpenAIOAuthHandler) SetCopilotOAuthService(copilotOAuthService *service.CopilotOAuthService) {
	h.copilotOAuthService = copilotOAuthService
}

// OpenAIGenerateAuthURLRequest represents the request for generating OpenAI auth URL
type OpenAIGenerateAuthURLRequest struct {
	ProxyID     *int64 `json:"proxy_id"`
	RedirectURI string `json:"redirect_uri"`
}

// GenerateAuthURL generates OpenAI OAuth authorization URL
// POST /api/v1/admin/openai/generate-auth-url
func (h *OpenAIOAuthHandler) GenerateAuthURL(c *gin.Context) {
	var req OpenAIGenerateAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = OpenAIGenerateAuthURLRequest{}
	}

	result, err := h.openaiOAuthService.GenerateAuthURL(
		c.Request.Context(),
		req.ProxyID,
		req.RedirectURI,
		oauthPlatformFromPath(c),
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// OpenAIExchangeCodeRequest represents the request for exchanging OpenAI auth code
type OpenAIExchangeCodeRequest struct {
	SessionID   string `json:"session_id" binding:"required"`
	Code        string `json:"code" binding:"required"`
	State       string `json:"state" binding:"required"`
	RedirectURI string `json:"redirect_uri"`
	ProxyID     *int64 `json:"proxy_id"`
}

// ExchangeCode exchanges OpenAI authorization code for tokens
// POST /api/v1/admin/openai/exchange-code
func (h *OpenAIOAuthHandler) ExchangeCode(c *gin.Context) {
	var req OpenAIExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tokenInfo, err := h.openaiOAuthService.ExchangeCode(c.Request.Context(), &service.OpenAIExchangeCodeInput{
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

// OpenAIRefreshTokenRequest represents the request for refreshing OpenAI token
type OpenAIRefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
	RT           string `json:"rt"`
	ClientID     string `json:"client_id"`
	ProxyID      *int64 `json:"proxy_id"`
}

// RefreshToken refreshes an OpenAI OAuth token
// POST /api/v1/admin/openai/refresh-token
func (h *OpenAIOAuthHandler) RefreshToken(c *gin.Context) {
	var req OpenAIRefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		refreshToken = strings.TrimSpace(req.RT)
	}
	if refreshToken == "" {
		response.BadRequest(c, "refresh_token is required")
		return
	}

	var proxyURL string
	if req.ProxyID != nil {
		proxy, err := h.adminService.GetProxy(c.Request.Context(), *req.ProxyID)
		if err == nil && proxy != nil {
			proxyURL = proxy.URL()
		}
	}

	// 未指定 client_id 时，根据请求路径平台自动设置默认值，避免 repository 层盲猜
	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		platform := oauthPlatformFromPath(c)
		clientID, _ = openai.OAuthClientConfigByPlatform(platform)
	}

	tokenInfo, err := h.openaiOAuthService.RefreshTokenWithClientID(c.Request.Context(), refreshToken, proxyURL, clientID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

type CopilotDeviceFlowRequest struct {
	ProxyID *int64 `json:"proxy_id"`
}

func (h *OpenAIOAuthHandler) StartCopilotDeviceFlow(c *gin.Context) {
	if h.copilotOAuthService == nil {
		response.InternalError(c, "Copilot OAuth service unavailable")
		return
	}

	var req CopilotDeviceFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = CopilotDeviceFlowRequest{}
	}

	result, err := h.copilotOAuthService.StartDeviceFlow(c.Request.Context(), req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type CopilotDeviceFlowPollRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

type ResolveCopilotDeviceDraftRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	ProxyID   *int64 `json:"proxy_id"`
}

type ResolveCopilotDeviceDraftResponse struct {
	Credentials             map[string]any                 `json:"credentials"`
	Extra                   map[string]any                 `json:"extra"`
	User                    *service.CopilotGitHubUserInfo `json:"user,omitempty"`
	ResolvedUpstreamURL     string                         `json:"resolved_upstream_url,omitempty"`
	ResolvedUpstreamHost    string                         `json:"resolved_upstream_host,omitempty"`
	ResolvedUpstreamService string                         `json:"resolved_upstream_service,omitempty"`
}

func (h *OpenAIOAuthHandler) PollCopilotDeviceFlow(c *gin.Context) {
	if h.copilotOAuthService == nil {
		response.InternalError(c, "Copilot OAuth service unavailable")
		return
	}

	var req CopilotDeviceFlowPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.copilotOAuthService.PollDeviceFlow(c.Request.Context(), req.SessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *OpenAIOAuthHandler) ResolveCopilotDeviceDraft(c *gin.Context) {
	if h.copilotOAuthService == nil {
		response.InternalError(c, "Copilot OAuth service unavailable")
		return
	}

	var req ResolveCopilotDeviceDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	githubToken, userInfo, err := h.copilotOAuthService.GetAuthorizedDeviceSession(req.SessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	probeAccount := &service.Account{
		Platform:    service.PlatformCopilot,
		Type:        service.AccountTypeOAuth,
		Credentials: h.copilotOAuthService.BuildAccountCredentials(githubToken),
		ProxyID:     req.ProxyID,
	}
	if req.ProxyID != nil {
		if proxy, proxyErr := h.adminService.GetProxy(c.Request.Context(), *req.ProxyID); proxyErr == nil && proxy != nil {
			probeAccount.Proxy = proxy
		}
	}

	tokenInfo, err := h.copilotOAuthService.ExchangeGitHubTokenForCopilotToken(c.Request.Context(), probeAccount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	credentials := h.copilotOAuthService.BuildAccountCredentialsWithTokenInfo(githubToken, tokenInfo)
	extra := service.MergeStringAnyMap(
		h.copilotOAuthService.BuildAccountExtra(userInfo),
		map[string]any{
			"provider": "github",
			"source":   "copilot_device_flow",
		},
	)
	extra = service.MergeStringAnyMap(extra, h.copilotOAuthService.BuildAccountUpstreamExtra(tokenInfo, "copilot_device_draft"))
	resolved := service.ResolveUpstreamInfo(tokenInfo.APIBaseURL, service.PlatformCopilot, "copilot_device_draft")

	response.Success(c, ResolveCopilotDeviceDraftResponse{
		Credentials:             credentials,
		Extra:                   extra,
		User:                    userInfo,
		ResolvedUpstreamURL:     resolved.URL,
		ResolvedUpstreamHost:    resolved.Host,
		ResolvedUpstreamService: resolved.Service,
	})
}

// RefreshAccountToken refreshes token for a specific OpenAI account
// POST /api/v1/admin/openai/accounts/:id/refresh
func (h *OpenAIOAuthHandler) RefreshAccountToken(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	// Get account
	account, err := h.adminService.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	platform := oauthPlatformFromPath(c)
	if account.Platform != platform {
		response.BadRequest(c, "Account platform does not match OAuth endpoint")
		return
	}

	// Only refresh OAuth-based accounts
	if !account.IsOAuth() {
		response.BadRequest(c, "Cannot refresh non-OAuth account credentials")
		return
	}

	// Use OpenAI OAuth service to refresh token
	tokenInfo, err := h.openaiOAuthService.RefreshAccountToken(c.Request.Context(), account)
	if err != nil {
		h.adminService.EnsureOpenAIPrivacy(c.Request.Context(), account)
		response.ErrorFrom(c, err)
		return
	}

	// Build new credentials from token info
	newCredentials := h.openaiOAuthService.BuildAccountCredentials(tokenInfo)

	// Preserve non-token settings from existing credentials
	for k, v := range account.Credentials {
		if _, exists := newCredentials[k]; !exists {
			newCredentials[k] = v
		}
	}

	updatedAccount, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{
		Credentials: newCredentials,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.adminService.EnsureOpenAIPrivacy(c.Request.Context(), updatedAccount)

	response.Success(c, dto.AccountFromService(updatedAccount))
}

func (h *OpenAIOAuthHandler) RefreshCopilotAccount(c *gin.Context) {
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
	if account.Platform != service.PlatformCopilot {
		response.BadRequest(c, "Account platform does not match Copilot endpoint")
		return
	}
	if !account.IsOAuth() {
		response.BadRequest(c, "Cannot refresh non-OAuth account credentials")
		return
	}

	updatedAccount, err := refreshCopilotOAuthAccount(c.Request.Context(), h.adminService, h.copilotOAuthService, account)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromService(updatedAccount))
}

// CreateAccountFromOAuth creates a new OpenAI OAuth account from token info
// POST /api/v1/admin/openai/create-from-oauth
func (h *OpenAIOAuthHandler) CreateAccountFromOAuth(c *gin.Context) {
	var req struct {
		SessionID   string  `json:"session_id" binding:"required"`
		Code        string  `json:"code" binding:"required"`
		State       string  `json:"state" binding:"required"`
		RedirectURI string  `json:"redirect_uri"`
		ProxyID     *int64  `json:"proxy_id"`
		Name        string  `json:"name"`
		Concurrency int     `json:"concurrency"`
		Priority    int     `json:"priority"`
		GroupIDs    []int64 `json:"group_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// Exchange code for tokens
	tokenInfo, err := h.openaiOAuthService.ExchangeCode(c.Request.Context(), &service.OpenAIExchangeCodeInput{
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

	// Build credentials from token info
	credentials := h.openaiOAuthService.BuildAccountCredentials(tokenInfo)

	platform := oauthPlatformFromPath(c)

	// Use email as default name if not provided
	name := req.Name
	if name == "" && tokenInfo.Email != "" {
		name = tokenInfo.Email
	}
	if name == "" {
		name = "OpenAI OAuth Account"
	}

	// Create account
	account, err := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{
		Name:        name,
		Platform:    platform,
		Type:        "oauth",
		Credentials: credentials,
		Extra:       nil,
		ProxyID:     req.ProxyID,
		Concurrency: req.Concurrency,
		Priority:    req.Priority,
		GroupIDs:    req.GroupIDs,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.adminService.ForceOpenAIPrivacy(c.Request.Context(), account)

	response.Success(c, dto.AccountFromService(account))
}

type CreateCopilotAccountFromDeviceRequest struct {
	SessionID               string   `json:"session_id" binding:"required"`
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
	ConfirmMixedChannelRisk *bool    `json:"confirm_mixed_channel_risk"`
}

func (h *OpenAIOAuthHandler) CreateCopilotAccountFromDevice(c *gin.Context) {
	if h.copilotOAuthService == nil {
		response.InternalError(c, "Copilot OAuth service unavailable")
		return
	}

	var req CreateCopilotAccountFromDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	githubToken, userInfo, err := h.copilotOAuthService.GetAuthorizedDeviceSession(req.SessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	credentials := h.copilotOAuthService.BuildAccountCredentials(githubToken)
	probeAccount := &service.Account{
		Platform:    service.PlatformCopilot,
		Type:        service.AccountTypeOAuth,
		Credentials: credentials,
		ProxyID:     req.ProxyID,
	}
	tokenInfo, err := h.copilotOAuthService.ExchangeGitHubTokenForCopilotToken(c.Request.Context(), probeAccount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	credentials = h.copilotOAuthService.BuildAccountCredentialsWithTokenInfo(githubToken, tokenInfo)

	extra := service.MergeStringAnyMap(
		h.copilotOAuthService.BuildAccountExtra(userInfo),
		map[string]any{
			"provider": "github",
			"source":   "copilot_device_flow",
		},
	)
	extra = service.MergeStringAnyMap(extra, h.copilotOAuthService.BuildAccountUpstreamExtra(tokenInfo, "copilot_device_create"))

	name := strings.TrimSpace(req.Name)
	if name == "" && userInfo != nil {
		if email := strings.TrimSpace(userInfo.Email); email != "" {
			name = email
		} else if login := strings.TrimSpace(userInfo.Login); login != "" {
			name = login
		} else if displayName := strings.TrimSpace(userInfo.Name); displayName != "" {
			name = displayName
		}
	}
	if name == "" {
		name = "Copilot OAuth Account"
	}

	account, err := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{
		Name:               name,
		Notes:              req.Notes,
		Platform:           service.PlatformCopilot,
		Type:               service.AccountTypeOAuth,
		Credentials:        credentials,
		Extra:              extra,
		ProxyID:            req.ProxyID,
		Concurrency:        req.Concurrency,
		Priority:           req.Priority,
		RateMultiplier:     req.RateMultiplier,
		LoadFactor:         req.LoadFactor,
		GroupIDs:           req.GroupIDs,
		ExpiresAt:          req.ExpiresAt,
		AutoPauseOnExpired: req.AutoPauseOnExpired,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.copilotOAuthService.DeleteDeviceSession(req.SessionID)

	response.Success(c, dto.AccountFromService(account))
}

type ReauthorizeCopilotAccountFromDeviceRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

func (h *OpenAIOAuthHandler) ReauthorizeCopilotAccountFromDevice(c *gin.Context) {
	if h.copilotOAuthService == nil {
		response.InternalError(c, "Copilot OAuth service unavailable")
		return
	}

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
	if account.Platform != service.PlatformCopilot {
		response.BadRequest(c, "Account platform does not match Copilot endpoint")
		return
	}
	if !account.IsOAuth() {
		response.BadRequest(c, "Cannot re-authorize non-OAuth account credentials")
		return
	}

	var req ReauthorizeCopilotAccountFromDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	githubToken, userInfo, err := h.copilotOAuthService.GetAuthorizedDeviceSession(req.SessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	credentials := h.copilotOAuthService.BuildAccountCredentials(githubToken)
	probeAccount := &service.Account{
		Platform:    service.PlatformCopilot,
		Type:        service.AccountTypeOAuth,
		Credentials: credentials,
		ProxyID:     account.ProxyID,
		Proxy:       account.Proxy,
	}
	tokenInfo, err := h.copilotOAuthService.ExchangeGitHubTokenForCopilotToken(c.Request.Context(), probeAccount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	credentials = h.copilotOAuthService.BuildAccountCredentialsWithTokenInfo(githubToken, tokenInfo)

	extra := service.MergeStringAnyMap(account.Extra, h.copilotOAuthService.BuildAccountExtra(userInfo))
	extra = service.MergeStringAnyMap(extra, map[string]any{
		"provider": "github",
		"source":   "copilot_device_flow",
	})
	extra = service.MergeStringAnyMap(extra, h.copilotOAuthService.BuildAccountUpstreamExtra(tokenInfo, "copilot_device_reauthorize"))

	if _, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{
		Type:        service.AccountTypeOAuth,
		Credentials: credentials,
		Extra:       extra,
	}); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	updatedAccount, err := h.adminService.ClearAccountError(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	h.copilotOAuthService.DeleteDeviceSession(req.SessionID)
	response.Success(c, dto.AccountFromService(updatedAccount))
}
