package admin

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type adminAPIKeyGroupsUpdater interface {
	AdminUpdateAPIKeyGroups(ctx context.Context, keyID int64, inputs []service.AdminAPIKeyGroupUpdateInput, modelDisplayMode *string) (*service.AdminUpdateAPIKeyGroupsResult, error)
}

type adminAPIKeyGroupsReader interface {
	GetAPIKeyGroups(ctx context.Context, keyID int64) ([]service.APIKeyGroupBinding, error)
}

// AdminAPIKeyHandler handles admin API key management
type AdminAPIKeyHandler struct {
	adminService service.AdminService
}

// NewAdminAPIKeyHandler creates a new admin API key handler
func NewAdminAPIKeyHandler(adminService service.AdminService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{
		adminService: adminService,
	}
}

// AdminUpdateAPIKeyGroupRequest represents the request to update an API key's group
type AdminUpdateAPIKeyGroupRequest struct {
	GroupID          *int64                  `json:"group_id"`
	Groups           []APIKeyGroupBindingReq `json:"groups"`
	ModelDisplayMode *string                 `json:"model_display_mode"`
}

type APIKeyGroupBindingReq struct {
	GroupID       int64    `json:"group_id" binding:"required"`
	Quota         float64  `json:"quota"`
	ModelPatterns []string `json:"model_patterns"`
}

// UpdateGroup handles updating an API key's group binding
// PUT /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) UpdateGroup(c *gin.Context) {
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}

	var req AdminUpdateAPIKeyGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if updater, ok := h.adminService.(adminAPIKeyGroupsUpdater); ok && (len(req.Groups) > 0 || req.GroupID != nil || req.ModelDisplayMode != nil) {
		inputs := make([]service.AdminAPIKeyGroupUpdateInput, 0, len(req.Groups))
		if len(req.Groups) > 0 {
			for _, item := range req.Groups {
				inputs = append(inputs, service.AdminAPIKeyGroupUpdateInput{
					GroupID:       item.GroupID,
					Quota:         item.Quota,
					ModelPatterns: item.ModelPatterns,
				})
			}
		} else if req.GroupID != nil && *req.GroupID > 0 {
			inputs = append(inputs, service.AdminAPIKeyGroupUpdateInput{GroupID: *req.GroupID})
		}

		result, err := updater.AdminUpdateAPIKeyGroups(c.Request.Context(), keyID, inputs, req.ModelDisplayMode)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, gin.H{
			"api_key":                   dto.APIKeyFromService(result.APIKey),
			"auto_granted_group_access": result.AutoGrantedGroupAccess,
			"granted_group_id":          result.GrantedGroupID,
			"granted_group_name":        result.GrantedGroupName,
			"granted_groups":            grantedGroupsDTO(result.GrantedGroups),
		})
		return
	}

	result, err := h.adminService.AdminUpdateAPIKeyGroupID(c.Request.Context(), keyID, req.GroupID, req.ModelDisplayMode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"api_key":                   dto.APIKeyFromService(result.APIKey),
		"auto_granted_group_access": result.AutoGrantedGroupAccess,
		"granted_group_id":          result.GrantedGroupID,
		"granted_group_name":        result.GrantedGroupName,
	})
}

// GetGroups handles fetching an API key's group bindings
// GET /api/v1/admin/api-keys/:id/groups
func (h *AdminAPIKeyHandler) GetGroups(c *gin.Context) {
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}

	reader, ok := h.adminService.(adminAPIKeyGroupsReader)
	if !ok {
		response.Success(c, gin.H{"groups": []dto.APIKeyGroupDTO{}})
		return
	}

	bindings, err := reader.GetAPIKeyGroups(c.Request.Context(), keyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	groups := make([]dto.APIKeyGroupDTO, 0, len(bindings))
	for _, binding := range bindings {
		item := dto.APIKeyGroupDTO{
			GroupID:       binding.GroupID,
			Quota:         binding.Quota,
			QuotaUsed:     binding.QuotaUsed,
			ModelPatterns: append([]string(nil), binding.ModelPatterns...),
		}
		if binding.Group != nil {
			item.GroupName = binding.Group.Name
			item.Platform = binding.Group.Platform
			item.Priority = binding.Group.Priority
		}
		groups = append(groups, item)
	}

	response.Success(c, gin.H{"groups": groups})
}

func grantedGroupsDTO(items []service.AdminGrantedGroupAccess) []dto.APIKeyGroupDTO {
	if len(items) == 0 {
		return nil
	}
	out := make([]dto.APIKeyGroupDTO, 0, len(items))
	for _, item := range items {
		out = append(out, dto.APIKeyGroupDTO{
			GroupID:   item.GroupID,
			GroupName: item.GroupName,
		})
	}
	return out
}
