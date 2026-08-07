package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type PasskeyHandler struct {
	authService    *service.AuthService
	passkeyService *service.PasskeyService
	settingService *service.SettingService
}

func NewPasskeyHandler(authService *service.AuthService, passkeyService *service.PasskeyService, settingService *service.SettingService) *PasskeyHandler {
	return &PasskeyHandler{
		authService:    authService,
		passkeyService: passkeyService,
		settingService: settingService,
	}
}

type passkeyCredentialResponse struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	CredentialID string  `json:"credential_id"`
	LastUsedAt   *string `json:"last_used_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type beginPasskeyRegistrationRequest struct {
	Password string `json:"password" binding:"required"`
	Name     string `json:"name"`
}

type finishPasskeyRegistrationRequest struct {
	SessionID  string          `json:"session_id" binding:"required"`
	Name       string          `json:"name"`
	Credential json.RawMessage `json:"credential" binding:"required"`
}

type beginPasskeyLoginRequest struct {
	TurnstileToken           string `json:"turnstile_token"`
	TencentCaptchaTicket     string `json:"tencent_captcha_ticket"`
	TencentCaptchaRandstr    string `json:"tencent_captcha_randstr"`
	AliyunCaptchaVerifyParam string `json:"aliyun_captcha_verify_param"`
}

type finishPasskeyLoginRequest struct {
	SessionID  string          `json:"session_id" binding:"required"`
	Credential json.RawMessage `json:"credential" binding:"required"`
}

type renamePasskeyRequest struct {
	Name string `json:"name" binding:"required"`
}

type deletePasskeyRequest struct {
	Password string `json:"password" binding:"required"`
}

func (h *PasskeyHandler) List(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if !h.passkeysEnabled(c.Request.Context()) {
		response.Success(c, []passkeyCredentialResponse{})
		return
	}
	items, err := h.passkeyService.List(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]passkeyCredentialResponse, 0, len(items))
	for i := range items {
		out = append(out, passkeyCredentialToResponse(items[i]))
	}
	response.Success(c, out)
}

func (h *PasskeyHandler) BeginRegistration(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req beginPasskeyRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !h.passkeysEnabled(c.Request.Context()) {
		response.ErrorFrom(c, service.ErrPasskeysDisabled)
		return
	}
	result, err := h.passkeyService.BeginRegistration(c.Request.Context(), subject.UserID, req.Password)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *PasskeyHandler) FinishRegistration(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req finishPasskeyRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !h.passkeysEnabled(c.Request.Context()) {
		response.ErrorFrom(c, service.ErrPasskeysDisabled)
		return
	}
	item, err := h.passkeyService.FinishRegistration(c.Request.Context(), subject.UserID, req.SessionID, req.Name, req.Credential)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, passkeyCredentialToResponse(*item))
}

func (h *PasskeyHandler) BeginLogin(c *gin.Context) {
	var req beginPasskeyLoginRequest
	_ = c.ShouldBindJSON(&req)
	if !h.passkeysEnabled(c.Request.Context()) {
		response.ErrorFrom(c, service.ErrPasskeysDisabled)
		return
	}
	if h.authService != nil {
		if err := h.authService.VerifyCaptcha(c.Request.Context(), authCaptchaProof(req.TurnstileToken, req.TencentCaptchaTicket, req.TencentCaptchaRandstr, req.AliyunCaptchaVerifyParam), ip.GetTrustedClientIP(c)); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	result, err := h.passkeyService.BeginLogin(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *PasskeyHandler) FinishLogin(c *gin.Context) {
	var req finishPasskeyLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !h.passkeysEnabled(c.Request.Context()) {
		response.ErrorFrom(c, service.ErrPasskeysDisabled)
		return
	}
	result, err := h.passkeyService.FinishLogin(c.Request.Context(), req.SessionID, req.Credential)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if h.settingService != nil && h.settingService.IsMaintenanceModeEnabled(c.Request.Context()) && !result.User.IsAdmin() {
		response.ErrorWithDetails(c, http.StatusServiceUnavailable, service.MaintenanceModeMessage, service.MaintenanceModeErrorCode, nil)
		return
	}
	if h.settingService != nil && h.settingService.IsBackendModeEnabled(c.Request.Context()) && !result.User.IsAdmin() {
		response.Forbidden(c, "Backend mode is active. Only admin login is allowed.")
		return
	}
	response.Success(c, AuthResponse{
		AccessToken:  result.TokenPair.AccessToken,
		RefreshToken: result.TokenPair.RefreshToken,
		ExpiresIn:    result.TokenPair.ExpiresIn,
		TokenType:    "Bearer",
		User:         dto.UserFromService(result.User),
	})
}

func (h *PasskeyHandler) Rename(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid passkey id")
		return
	}
	var req renamePasskeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !h.passkeysEnabled(c.Request.Context()) {
		response.ErrorFrom(c, service.ErrPasskeysDisabled)
		return
	}
	if err := h.passkeyService.Rename(c.Request.Context(), subject.UserID, id, req.Name); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Passkey renamed successfully"})
}

func (h *PasskeyHandler) Delete(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid passkey id")
		return
	}
	var req deletePasskeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if !h.passkeysEnabled(c.Request.Context()) {
		response.ErrorFrom(c, service.ErrPasskeysDisabled)
		return
	}
	if err := h.passkeyService.Delete(c.Request.Context(), subject.UserID, id, req.Password); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Passkey deleted successfully"})
}

func (h *PasskeyHandler) passkeysEnabled(ctx context.Context) bool {
	return h != nil && h.passkeyService != nil && h.passkeyService.Enabled() && h.settingService != nil && h.settingService.IsPasskeyEnabled(ctx)
}

func passkeyCredentialToResponse(item service.PasskeyCredential) passkeyCredentialResponse {
	createdAt := item.CreatedAt.UTC().Format(time.RFC3339)
	updatedAt := item.UpdatedAt.UTC().Format(time.RFC3339)
	var lastUsedAt *string
	if item.LastUsedAt != nil {
		formatted := item.LastUsedAt.UTC().Format(time.RFC3339)
		lastUsedAt = &formatted
	}
	return passkeyCredentialResponse{
		ID:           item.ID,
		Name:         item.Name,
		CredentialID: base64.RawURLEncoding.EncodeToString(item.CredentialID),
		LastUsedAt:   lastUsedAt,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}
