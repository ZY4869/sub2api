package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related requests
type UserHandler struct {
	userService *service.UserService
	affiliate   *service.AffiliateService
	identities  *service.AuthIdentityService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService *service.UserService, affiliateService *service.AffiliateService) *UserHandler {
	return &UserHandler{
		userService: userService,
		affiliate:   affiliateService,
	}
}

func (h *UserHandler) GetUserService() *service.UserService {
	if h == nil {
		return nil
	}
	return h.userService
}

func (h *UserHandler) SetAuthIdentityService(identityService *service.AuthIdentityService) {
	h.identities = identityService
}

// ChangePasswordRequest represents the change password request payload
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// UpdateProfileRequest represents the update profile request payload
type UpdateProfileRequest struct {
	Username                        *string `json:"username"`
	UsageModelDisplayMode           *string `json:"usage_model_display_mode"`
	GlobalRealtimeCountdownEnabled  *bool   `json:"global_realtime_countdown_enabled"`
	AccountRealtimeCountdownEnabled *bool   `json:"account_realtime_countdown_enabled"`
	VisualPresetPreference          *string `json:"visual_preset_preference"`
	AccountVisualPresetOverride     *string `json:"account_visual_preset_override"`
}

// GetProfile handles getting user profile
// GET /api/v1/users/me
func (h *UserHandler) GetProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	userData, err := h.userService.GetByID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.UserFromService(userData))
}

// ChangePassword handles changing user password
// POST /api/v1/users/me/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	svcReq := service.ChangePasswordRequest{
		CurrentPassword: req.OldPassword,
		NewPassword:     req.NewPassword,
	}
	err := h.userService.ChangePassword(c.Request.Context(), subject.UserID, svcReq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Password changed successfully"})
}

// UpdateProfile handles updating user profile
// PUT /api/v1/users/me
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	svcReq := service.UpdateProfileRequest{
		Username:                        req.Username,
		UsageModelDisplayMode:           req.UsageModelDisplayMode,
		GlobalRealtimeCountdownEnabled:  req.GlobalRealtimeCountdownEnabled,
		AccountRealtimeCountdownEnabled: req.AccountRealtimeCountdownEnabled,
		VisualPresetPreference:          req.VisualPresetPreference,
		AccountVisualPresetOverride:     req.AccountVisualPresetOverride,
	}
	updatedUser, err := h.userService.UpdateProfile(c.Request.Context(), subject.UserID, svcReq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.UserFromService(updatedUser))
}

func (h *UserHandler) ListAuthIdentities(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.identities == nil {
		response.Success(c, []dto.AuthIdentity{})
		return
	}
	items, err := h.identities.ListByUserID(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result := make([]dto.AuthIdentity, 0, len(items))
	for _, item := range items {
		if mapped := dto.AuthIdentityFromService(item); mapped != nil {
			result = append(result, *mapped)
		}
	}
	response.Success(c, result)
}

func (h *UserHandler) DeleteAuthIdentity(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if h.identities == nil {
		response.ErrorFrom(c, service.ErrAuthIdentityNotFound)
		return
	}
	if err := h.identities.DeleteByUserIDAndProvider(c.Request.Context(), subject.UserID, c.Param("provider")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Auth identity removed"})
}
