package admin

import (
	"errors"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ChannelHandler struct {
	channelService *service.ChannelService
}

func NewChannelHandler(channelService *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{channelService: channelService}
}

type CreateChannelRequest struct {
	Name               string                       `json:"name" binding:"required"`
	Description        string                       `json:"description"`
	Status             string                       `json:"status"`
	RestrictModels     bool                         `json:"restrict_models"`
	BillingModelSource string                       `json:"billing_model_source"`
	GroupIDs           []int64                      `json:"group_ids"`
	ModelMapping       map[string]map[string]string `json:"model_mapping"`
	ModelPricing       []model.ChannelModelPricing  `json:"model_pricing"`
}

type UpdateChannelRequest struct {
	Name               *string                       `json:"name"`
	Description        *string                       `json:"description"`
	Status             *string                       `json:"status"`
	RestrictModels     *bool                         `json:"restrict_models"`
	BillingModelSource *string                       `json:"billing_model_source"`
	GroupIDs           *[]int64                      `json:"group_ids"`
	ModelMapping       *map[string]map[string]string `json:"model_mapping"`
	ModelPricing       *[]model.ChannelModelPricing  `json:"model_pricing"`
}

func (h *ChannelHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.channelService.List(c.Request.Context(), pagination.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}, service.ChannelListFilters{
		Status: c.Query("status"),
		Search: c.Query("search"),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, result.Total, result.Page, result.PageSize)
}

func (h *ChannelHandler) GetByID(c *gin.Context) {
	channelID, ok := parseChannelID(c)
	if !ok {
		return
	}
	channel, err := h.channelService.GetByID(c.Request.Context(), channelID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, channel)
}

func (h *ChannelHandler) Create(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	channel, err := h.channelService.Create(c.Request.Context(), &model.Channel{
		Name:               req.Name,
		Description:        req.Description,
		Status:             req.Status,
		RestrictModels:     req.RestrictModels,
		BillingModelSource: req.BillingModelSource,
		GroupIDs:           req.GroupIDs,
		ModelMapping:       req.ModelMapping,
		ModelPricing:       req.ModelPricing,
	})
	if err != nil {
		handleChannelError(c, err)
		return
	}
	response.Success(c, channel)
}

func (h *ChannelHandler) Update(c *gin.Context) {
	channelID, ok := parseChannelID(c)
	if !ok {
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	channel, err := h.channelService.GetByID(c.Request.Context(), channelID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Description != nil {
		channel.Description = *req.Description
	}
	if req.Status != nil {
		channel.Status = *req.Status
	}
	if req.RestrictModels != nil {
		channel.RestrictModels = *req.RestrictModels
	}
	if req.BillingModelSource != nil {
		channel.BillingModelSource = *req.BillingModelSource
	}
	if req.GroupIDs != nil {
		channel.GroupIDs = *req.GroupIDs
	}
	if req.ModelMapping != nil {
		channel.ModelMapping = *req.ModelMapping
	}
	if req.ModelPricing != nil {
		channel.ModelPricing = *req.ModelPricing
	}

	updated, err := h.channelService.Update(c.Request.Context(), channel)
	if err != nil {
		handleChannelError(c, err)
		return
	}
	response.Success(c, updated)
}

func (h *ChannelHandler) Delete(c *gin.Context) {
	channelID, ok := parseChannelID(c)
	if !ok {
		return
	}
	if err := h.channelService.Delete(c.Request.Context(), channelID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Channel deleted successfully"})
}

func parseChannelID(c *gin.Context) (int64, bool) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid channel ID")
		return 0, false
	}
	return channelID, true
}

func handleChannelError(c *gin.Context, err error) {
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) {
		response.BadRequest(c, validationErr.Error())
		return
	}
	response.ErrorFrom(c, err)
}
