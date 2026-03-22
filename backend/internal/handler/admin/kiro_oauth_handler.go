package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type KiroOAuthHandler struct {
	kiroOAuthService *service.KiroOAuthService
	adminService     service.AdminService
}

func NewKiroOAuthHandler(kiroOAuthService *service.KiroOAuthService, adminService service.AdminService) *KiroOAuthHandler {
	return &KiroOAuthHandler{
		kiroOAuthService: kiroOAuthService,
		adminService:     adminService,
	}
}

type KiroGenerateAuthURLRequest struct {
	ProxyID     *int64 `json:"proxy_id"`
	RedirectURI string `json:"redirect_uri"`
	Method      string `json:"method"`
	StartURL    string `json:"start_url"`
	Region      string `json:"region"`
}

func (h *KiroOAuthHandler) GenerateAuthURL(c *gin.Context) {
	if h.kiroOAuthService == nil {
		response.InternalError(c, "Kiro OAuth service unavailable")
		return
	}

	var req KiroGenerateAuthURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = KiroGenerateAuthURLRequest{}
	}

	result, err := h.kiroOAuthService.GenerateAuthURL(c.Request.Context(), &service.KiroGenerateAuthURLInput{
		ProxyID:     req.ProxyID,
		RedirectURI: req.RedirectURI,
		Method:      req.Method,
		StartURL:    req.StartURL,
		Region:      req.Region,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

type KiroExchangeCodeRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Code      string `json:"code" binding:"required"`
	State     string `json:"state" binding:"required"`
	ProxyID   *int64 `json:"proxy_id"`
}

func (h *KiroOAuthHandler) ExchangeCode(c *gin.Context) {
	if h.kiroOAuthService == nil {
		response.InternalError(c, "Kiro OAuth service unavailable")
		return
	}

	var req KiroExchangeCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tokenInfo, err := h.kiroOAuthService.ExchangeCode(c.Request.Context(), &service.KiroExchangeCodeInput{
		SessionID: req.SessionID,
		Code:      req.Code,
		State:     req.State,
		ProxyID:   req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, tokenInfo)
}

type CreateKiroAccountFromOAuthRequest struct {
	SessionID               string   `json:"session_id" binding:"required"`
	Code                    string   `json:"code" binding:"required"`
	State                   string   `json:"state" binding:"required"`
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

func (h *KiroOAuthHandler) CreateAccountFromOAuth(c *gin.Context) {
	if h.kiroOAuthService == nil {
		response.InternalError(c, "Kiro OAuth service unavailable")
		return
	}

	var req CreateKiroAccountFromOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tokenInfo, err := h.kiroOAuthService.ExchangeCode(c.Request.Context(), &service.KiroExchangeCodeInput{
		SessionID: req.SessionID,
		Code:      req.Code,
		State:     req.State,
		ProxyID:   req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	account, err := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{
		Name:                  resolveKiroAccountName(req.Name, tokenInfo),
		Notes:                 req.Notes,
		Platform:              service.PlatformKiro,
		Type:                  service.AccountTypeOAuth,
		Credentials:           h.kiroOAuthService.BuildAccountCredentials(tokenInfo),
		Extra:                 h.kiroOAuthService.BuildAccountExtra(tokenInfo),
		ProxyID:               req.ProxyID,
		Concurrency:           req.Concurrency,
		Priority:              req.Priority,
		RateMultiplier:        req.RateMultiplier,
		LoadFactor:            req.LoadFactor,
		GroupIDs:              req.GroupIDs,
		ExpiresAt:             req.ExpiresAt,
		AutoPauseOnExpired:    req.AutoPauseOnExpired,
		SkipMixedChannelCheck: req.ConfirmMixedChannelRisk != nil && *req.ConfirmMixedChannelRisk,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromService(account))
}

type ReauthorizeKiroAccountFromOAuthRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Code      string `json:"code" binding:"required"`
	State     string `json:"state" binding:"required"`
	ProxyID   *int64 `json:"proxy_id"`
}

func (h *KiroOAuthHandler) ReauthorizeAccountFromOAuth(c *gin.Context) {
	if h.kiroOAuthService == nil {
		response.InternalError(c, "Kiro OAuth service unavailable")
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
	if account.Platform != service.PlatformKiro {
		response.BadRequest(c, "Account platform does not match Kiro endpoint")
		return
	}
	if !account.IsOAuth() {
		response.BadRequest(c, "Cannot re-authorize non-OAuth account credentials")
		return
	}

	var req ReauthorizeKiroAccountFromOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	tokenInfo, err := h.kiroOAuthService.ExchangeCode(c.Request.Context(), &service.KiroExchangeCodeInput{
		SessionID: req.SessionID,
		Code:      req.Code,
		State:     req.State,
		ProxyID:   req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	credentials := h.kiroOAuthService.BuildAccountCredentials(tokenInfo)
	credentials = service.MergeCredentials(account.Credentials, credentials)
	extra := service.MergeStringAnyMap(account.Extra, h.kiroOAuthService.BuildAccountExtra(tokenInfo))

	updatedAccount, err := h.adminService.UpdateAccount(c.Request.Context(), accountID, &service.UpdateAccountInput{
		Type:        service.AccountTypeOAuth,
		Credentials: credentials,
		Extra:       extra,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	updatedAccount, err = h.adminService.ClearAccountError(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromService(updatedAccount))
}

func (h *KiroOAuthHandler) RefreshAccount(c *gin.Context) {
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
	if account.Platform != service.PlatformKiro {
		response.BadRequest(c, "Account platform does not match Kiro endpoint")
		return
	}
	if !account.IsOAuth() {
		response.BadRequest(c, "Cannot refresh non-OAuth account credentials")
		return
	}

	updatedAccount, err := refreshKiroOAuthAccount(c.Request.Context(), h.adminService, h.kiroOAuthService, account)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.AccountFromService(updatedAccount))
}

func resolveKiroAccountName(raw string, tokenInfo *service.KiroTokenInfo) string {
	if name := strings.TrimSpace(raw); name != "" {
		return name
	}
	if tokenInfo == nil {
		return "Kiro OAuth Account"
	}
	for _, candidate := range []string{tokenInfo.Email, tokenInfo.Username, tokenInfo.DisplayName} {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return "Kiro OAuth Account"
}
