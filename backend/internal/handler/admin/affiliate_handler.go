package admin

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type AffiliateHandler struct {
	affiliateService *service.AffiliateService
}

func NewAffiliateHandler(affiliateService *service.AffiliateService) *AffiliateHandler {
	return &AffiliateHandler{affiliateService: affiliateService}
}

func parseOptionalBoolQuery(c *gin.Context, key string) (*bool, bool) {
	raw, ok := c.GetQuery(key)
	if !ok {
		return nil, false
	}
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return nil, false
	}
	switch raw {
	case "1", "true", "t", "yes", "y", "on":
		v := true
		return &v, true
	case "0", "false", "f", "no", "n", "off":
		v := false
		return &v, true
	default:
		return nil, false
	}
}

func normalizeAffiliateCode(input string) string {
	input = strings.TrimSpace(strings.ToUpper(input))
	input = strings.ReplaceAll(input, "-", "")
	input = strings.ReplaceAll(input, " ", "")
	return input
}

func isAffiliateCodeLike(input string) bool {
	if input == "" {
		return false
	}
	for _, ch := range input {
		if ch >= 'A' && ch <= 'Z' {
			continue
		}
		if ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

func (h *AffiliateHandler) ListUsers(c *gin.Context) {
	if h == nil || h.affiliateService == nil {
		response.InternalError(c, "Service not configured")
		return
	}

	page, pageSize := response.ParsePagination(c)
	params := pagination.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	}

	filters := service.AffiliateAdminUserListFilters{}
	if v, ok := parseOptionalBoolQuery(c, "has_custom_code"); ok {
		filters.HasCustomCode = v
	}
	if v, ok := parseOptionalBoolQuery(c, "has_custom_rate"); ok {
		filters.HasCustomRate = v
	}
	if v, ok := parseOptionalBoolQuery(c, "has_inviter"); ok {
		filters.HasInviter = v
	}

	items, paginationResult, err := h.affiliateService.ListAdminUsers(c.Request.Context(), params, filters)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, paginationResult.Total, page, pageSize)
}

func (h *AffiliateHandler) LookupUsers(c *gin.Context) {
	if h == nil || h.affiliateService == nil {
		response.InternalError(c, "Service not configured")
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}

	items, err := h.affiliateService.LookupAdminUsers(c.Request.Context(), q, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

type updateAffiliateUserRequest struct {
	AffCode                 *string         `json:"aff_code"`
	CustomRebateRatePercent json.RawMessage `json:"custom_rebate_rate_percent"`
}

func (h *AffiliateHandler) UpdateUser(c *gin.Context) {
	if h == nil || h.affiliateService == nil {
		response.InternalError(c, "Service not configured")
		return
	}

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}

	var req updateAffiliateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	update := service.AffiliateAdminUserCustomUpdate{}

	if req.AffCode != nil {
		update.AffCodeSet = true
		code := normalizeAffiliateCode(*req.AffCode)
		if code == "" {
			update.AffCode = nil
		} else {
			if len(code) < 6 || len(code) > 32 || !isAffiliateCodeLike(code) {
				response.BadRequest(c, "Invalid aff_code: must be 6-32 chars, A-Z0-9 only")
				return
			}
			update.AffCode = &code
		}
	}

	if req.CustomRebateRatePercent != nil {
		update.CustomRateSet = true
		raw := bytes.TrimSpace(req.CustomRebateRatePercent)
		if bytes.Equal(raw, []byte("null")) || len(raw) == 0 {
			update.CustomRate = nil
		} else {
			var v float64
			if err := json.Unmarshal(raw, &v); err != nil {
				response.BadRequest(c, "Invalid custom_rebate_rate_percent: "+err.Error())
				return
			}
			if v < 0 {
				v = 0
			}
			if v > 100 {
				v = 100
			}
			update.CustomRate = &v
		}
	}

	if !update.AffCodeSet && !update.CustomRateSet {
		response.BadRequest(c, "No fields to update")
		return
	}

	row, err := h.affiliateService.UpdateAdminUserCustom(c.Request.Context(), userID, update)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		slog.Info("affiliate: admin updated user custom", "admin_user_id", subject.UserID, "user_id", userID, "aff_code_set", update.AffCodeSet, "custom_rate_set", update.CustomRateSet)
	}
	response.Success(c, dto.UserAffiliateFromService(row))
}

func (h *AffiliateHandler) DeleteUserCustom(c *gin.Context) {
	if h == nil || h.affiliateService == nil {
		response.InternalError(c, "Service not configured")
		return
	}

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user_id")
		return
	}

	row, err := h.affiliateService.ResetAdminUserCustom(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		slog.Info("affiliate: admin reset user custom", "admin_user_id", subject.UserID, "user_id", userID)
	}
	response.Success(c, dto.UserAffiliateFromService(row))
}

type batchRateRequest struct {
	UserIDs                 []int64 `json:"user_ids" binding:"required"`
	CustomRebateRatePercent float64 `json:"custom_rebate_rate_percent" binding:"required"`
}

func (h *AffiliateHandler) BatchRate(c *gin.Context) {
	if h == nil || h.affiliateService == nil {
		response.InternalError(c, "Service not configured")
		return
	}

	var req batchRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if len(req.UserIDs) > 2000 {
		response.BadRequest(c, "Too many user_ids (max 2000)")
		return
	}

	updated, err := h.affiliateService.BatchUpdateAdminUserRates(c.Request.Context(), req.UserIDs, req.CustomRebateRatePercent)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok {
		slog.Info("affiliate: admin batch updated rates", "admin_user_id", subject.UserID, "updated", updated, "count", len(req.UserIDs))
	}
	response.Success(c, gin.H{"updated": updated})
}
