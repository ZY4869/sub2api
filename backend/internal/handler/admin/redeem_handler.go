package admin

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RedeemHandler handles admin redeem code management
type RedeemHandler struct {
	adminService  service.AdminService
	redeemService *service.RedeemService
}

// NewRedeemHandler creates a new admin redeem handler
func NewRedeemHandler(adminService service.AdminService, redeemService *service.RedeemService) *RedeemHandler {
	return &RedeemHandler{
		adminService:  adminService,
		redeemService: redeemService,
	}
}

// GenerateRedeemCodesRequest represents generate redeem codes request
type GenerateRedeemCodesRequest struct {
	Count         int     `json:"count" binding:"required,min=1,max=100"`
	Type          string  `json:"type" binding:"required,oneof=balance concurrency subscription invitation"`
	Value         float64 `json:"value"`
	GroupID       *int64  `json:"group_id"`                                    // 订阅类型必填
	ValidityDays  int     `json:"validity_days" binding:"omitempty,max=36500"` // 订阅类型使用，默认30天，最大100年
	ExpiresAt     *string `json:"expires_at"`
	ExpiresInDays *int    `json:"expires_in_days" binding:"omitempty,min=1,max=36500"`
}

// CreateAndRedeemCodeRequest represents creating a fixed code and redeeming it for a target user.
// Type 为 omitempty 而非 required 是为了向后兼容旧版调用方（不传 type 时默认 balance）。
type CreateAndRedeemCodeRequest struct {
	Code          string  `json:"code" binding:"required,min=3,max=128"`
	Type          string  `json:"type" binding:"omitempty,oneof=balance concurrency subscription invitation"` // 不传时默认 balance（向后兼容）
	Value         float64 `json:"value" binding:"required"`
	UserID        int64   `json:"user_id" binding:"required,gt=0"`
	GroupID       *int64  `json:"group_id"`                                    // subscription 类型必填
	ValidityDays  int     `json:"validity_days" binding:"omitempty,max=36500"` // subscription 类型必填，>0
	Notes         string  `json:"notes"`
	ExpiresAt     *string `json:"expires_at"`
	ExpiresInDays *int    `json:"expires_in_days" binding:"omitempty,min=1,max=36500"`
}

type BatchUpdateRedeemCodesRequest struct {
	IDs    []int64               `json:"ids" binding:"required,min=1"`
	Fields redeemCodeBatchFields `json:"fields" binding:"required"`
}

type redeemCodeBatchFields struct {
	Status       *string  `json:"status"`
	Notes        *string  `json:"notes"`
	ExpiresAt    *string  `json:"expires_at"`
	GroupID      *int64   `json:"group_id"`
	Type         *string  `json:"type"`
	Value        *float64 `json:"value"`
	ValidityDays *int     `json:"validity_days" binding:"omitempty,min=-36500,max=36500"`
	present      map[string]bool
}

func (f *redeemCodeBatchFields) UnmarshalJSON(data []byte) error {
	type alias redeemCodeBatchFields
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*f = redeemCodeBatchFields(decoded)
	f.present = make(map[string]bool, len(raw))
	for key := range raw {
		f.present[key] = true
	}
	return nil
}

// List handles listing all redeem codes with pagination
// GET /api/v1/admin/redeem-codes
func (h *RedeemHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	codeType := c.Query("type")
	status := c.Query("status")
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "id")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	// 标准化和验证 search 参数
	search = strings.TrimSpace(search)
	if len(search) > 100 {
		search = search[:100]
	}

	codes, total, err := h.adminService.ListRedeemCodesWithOptions(c.Request.Context(), service.RedeemCodeListInput{
		Page:      page,
		PageSize:  pageSize,
		Type:      codeType,
		Status:    status,
		Search:    search,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.AdminRedeemCode, 0, len(codes))
	for i := range codes {
		out = append(out, *dto.RedeemCodeFromServiceAdmin(&codes[i]))
	}
	response.Paginated(c, out, total, page, pageSize)
}

// GetByID handles getting a redeem code by ID
// GET /api/v1/admin/redeem-codes/:id
func (h *RedeemHandler) GetByID(c *gin.Context) {
	codeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid redeem code ID")
		return
	}

	code, err := h.adminService.GetRedeemCode(c.Request.Context(), codeID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.RedeemCodeFromServiceAdmin(code))
}

// Generate handles generating new redeem codes
// POST /api/v1/admin/redeem-codes/generate
func (h *RedeemHandler) Generate(c *gin.Context) {
	var req GenerateRedeemCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	expiresAt, err := parseOptionalRedeemExpiresAtOrDays(req.ExpiresAt, req.ExpiresInDays)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	executeAdminIdempotentJSON(c, "admin.redeem_codes.generate", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		codes, execErr := h.adminService.GenerateRedeemCodes(ctx, &service.GenerateRedeemCodesInput{
			Count:        req.Count,
			Type:         req.Type,
			Value:        req.Value,
			GroupID:      req.GroupID,
			ValidityDays: req.ValidityDays,
			ExpiresAt:    expiresAt,
		})
		if execErr != nil {
			return nil, execErr
		}

		out := make([]dto.AdminRedeemCode, 0, len(codes))
		for i := range codes {
			out = append(out, *dto.RedeemCodeFromServiceAdmin(&codes[i]))
		}
		return out, nil
	})
}

// CreateAndRedeem creates a fixed redeem code and redeems it for a target user in one step.
// POST /api/v1/admin/redeem-codes/create-and-redeem
func (h *RedeemHandler) CreateAndRedeem(c *gin.Context) {
	if h.redeemService == nil {
		response.InternalError(c, "redeem service not configured")
		return
	}

	var req CreateAndRedeemCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	// 向后兼容：旧版调用方（如 Sub2ApiPay）不传 type 字段，默认当作 balance 充值处理。
	// 请勿删除此默认值逻辑，否则会导致旧版调用方 400 报错。
	if req.Type == "" {
		req.Type = "balance"
	}

	if req.Type == "subscription" {
		if req.GroupID == nil {
			response.BadRequest(c, "group_id is required for subscription type")
			return
		}
		if req.ValidityDays == 0 {
			response.BadRequest(c, "validity_days must not be zero for subscription type")
			return
		}
	}
	expiresAt, err := parseOptionalRedeemExpiresAtOrDays(req.ExpiresAt, req.ExpiresInDays)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	executeAdminIdempotentJSON(c, "admin.redeem_codes.create_and_redeem", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		existing, err := h.redeemService.GetByCode(ctx, req.Code)
		if err == nil {
			return h.resolveCreateAndRedeemExisting(ctx, existing, req.UserID)
		}
		if !errors.Is(err, service.ErrRedeemCodeNotFound) {
			return nil, err
		}

		createErr := h.redeemService.CreateCode(ctx, &service.RedeemCode{
			Code:         req.Code,
			Type:         req.Type,
			Value:        req.Value,
			Status:       service.StatusUnused,
			Notes:        req.Notes,
			ExpiresAt:    expiresAt,
			GroupID:      req.GroupID,
			ValidityDays: req.ValidityDays,
		})
		if createErr != nil {
			// Unique code race: if code now exists, use idempotent semantics by used_by.
			existingAfterCreateErr, getErr := h.redeemService.GetByCode(ctx, req.Code)
			if getErr == nil {
				return h.resolveCreateAndRedeemExisting(ctx, existingAfterCreateErr, req.UserID)
			}
			return nil, createErr
		}

		redeemed, redeemErr := h.redeemService.Redeem(ctx, req.UserID, req.Code)
		if redeemErr != nil {
			return nil, redeemErr
		}
		return gin.H{"redeem_code": dto.RedeemCodeFromServiceAdmin(redeemed)}, nil
	})
}

func (h *RedeemHandler) resolveCreateAndRedeemExisting(ctx context.Context, existing *service.RedeemCode, userID int64) (any, error) {
	if existing == nil {
		return nil, infraerrors.Conflict("REDEEM_CODE_CONFLICT", "redeem code conflict")
	}

	// If previous run created the code but crashed before redeem, redeem it now.
	if existing.CanUse() {
		redeemed, err := h.redeemService.Redeem(ctx, userID, existing.Code)
		if err == nil {
			return gin.H{"redeem_code": dto.RedeemCodeFromServiceAdmin(redeemed)}, nil
		}
		if !errors.Is(err, service.ErrRedeemCodeUsed) {
			return nil, err
		}
		latest, getErr := h.redeemService.GetByCode(ctx, existing.Code)
		if getErr == nil {
			existing = latest
		}
	}

	if existing.UsedBy != nil && *existing.UsedBy == userID {
		return gin.H{"redeem_code": dto.RedeemCodeFromServiceAdmin(existing)}, nil
	}

	return nil, infraerrors.Conflict("REDEEM_CODE_CONFLICT", "redeem code already used by another user")
}

// Delete handles deleting a redeem code
// DELETE /api/v1/admin/redeem-codes/:id
func (h *RedeemHandler) Delete(c *gin.Context) {
	codeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid redeem code ID")
		return
	}

	err = h.adminService.DeleteRedeemCode(c.Request.Context(), codeID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Redeem code deleted successfully"})
}

// BatchDelete handles batch deleting redeem codes
// POST /api/v1/admin/redeem-codes/batch-delete
func (h *RedeemHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	deleted, err := h.adminService.BatchDeleteRedeemCodes(c.Request.Context(), req.IDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"deleted": deleted,
		"message": "Redeem codes deleted successfully",
	})
}

// BatchUpdate handles partial batch updates for redeem codes.
// POST /api/v1/admin/redeem-codes/batch-update
func (h *RedeemHandler) BatchUpdate(c *gin.Context) {
	var req BatchUpdateRedeemCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	input, err := req.toServiceInput()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	executeAdminIdempotentJSON(c, "admin.redeem_codes.batch_update", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		updated, execErr := h.adminService.BatchUpdateRedeemCodes(ctx, input)
		if execErr != nil {
			return nil, execErr
		}
		logRedeemCodeBatchUpdateAudit(c, req, updated)
		return gin.H{
			"updated": updated,
			"message": "Redeem codes updated successfully",
		}, nil
	})
}

// Expire handles expiring a redeem code
// POST /api/v1/admin/redeem-codes/:id/expire
func (h *RedeemHandler) Expire(c *gin.Context) {
	codeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid redeem code ID")
		return
	}

	code, err := h.adminService.ExpireRedeemCode(c.Request.Context(), codeID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, dto.RedeemCodeFromServiceAdmin(code))
}

// GetStats handles getting redeem code statistics
// GET /api/v1/admin/redeem-codes/stats
func (h *RedeemHandler) GetStats(c *gin.Context) {
	// Return mock data for now
	response.Success(c, gin.H{
		"total_codes":             0,
		"active_codes":            0,
		"used_codes":              0,
		"expired_codes":           0,
		"total_value_distributed": 0.0,
		"by_type": gin.H{
			"balance":     0,
			"concurrency": 0,
			"trial":       0,
		},
	})
}

// Export handles exporting redeem codes to CSV
// GET /api/v1/admin/redeem-codes/export
func (h *RedeemHandler) Export(c *gin.Context) {
	codeType := c.Query("type")
	status := c.Query("status")
	sortBy := c.DefaultQuery("sort_by", "id")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// Get all codes without pagination (use large page size)
	codes, _, err := h.adminService.ListRedeemCodesWithOptions(c.Request.Context(), service.RedeemCodeListInput{
		Page:      1,
		PageSize:  10000,
		Type:      codeType,
		Status:    status,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Create CSV buffer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	if err := writer.Write([]string{"id", "code", "type", "value", "status", "used_by", "used_by_email", "used_at", "created_at", "expires_at"}); err != nil {
		response.InternalError(c, "Failed to export redeem codes: "+err.Error())
		return
	}

	// Write data rows
	for _, code := range codes {
		usedBy := ""
		if code.UsedBy != nil {
			usedBy = fmt.Sprintf("%d", *code.UsedBy)
		}
		usedByEmail := ""
		if code.User != nil {
			usedByEmail = code.User.Email
		}
		usedAt := ""
		if code.UsedAt != nil {
			usedAt = code.UsedAt.Format("2006-01-02 15:04:05")
		}
		expiresAt := ""
		if code.ExpiresAt != nil {
			expiresAt = code.ExpiresAt.Format("2006-01-02 15:04:05")
		}
		if err := writer.Write([]string{
			fmt.Sprintf("%d", code.ID),
			code.Code,
			code.Type,
			fmt.Sprintf("%.2f", code.Value),
			code.Status,
			usedBy,
			usedByEmail,
			usedAt,
			code.CreatedAt.Format("2006-01-02 15:04:05"),
			expiresAt,
		}); err != nil {
			response.InternalError(c, "Failed to export redeem codes: "+err.Error())
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		response.InternalError(c, "Failed to export redeem codes: "+err.Error())
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=redeem_codes.csv")
	c.Data(200, "text/csv; charset=utf-8", append([]byte{0xEF, 0xBB, 0xBF}, buf.Bytes()...))
}

func (req *BatchUpdateRedeemCodesRequest) toServiceInput() (*service.BatchUpdateRedeemCodesInput, error) {
	fields := req.Fields
	input := &service.BatchUpdateRedeemCodesInput{IDs: req.IDs}
	if fields.present["status"] {
		status := strings.ToLower(strings.TrimSpace(derefStringPtr(fields.Status)))
		switch status {
		case service.StatusUnused, service.StatusExpired, service.StatusDisabled:
			input.Status = &status
		case service.StatusUsed:
			return nil, infraerrors.BadRequest("REDEEM_CODE_STATUS_INVALID", "cannot batch mark redeem codes as used")
		default:
			return nil, infraerrors.BadRequest("REDEEM_CODE_STATUS_INVALID", "status must be unused, expired, or disabled")
		}
	}
	if fields.present["notes"] {
		notes := strings.TrimSpace(derefStringPtr(fields.Notes))
		input.Notes = &notes
	}
	if fields.present["expires_at"] {
		expiresAt, err := parseOptionalRedeemExpiresAt(fields.ExpiresAt)
		if err != nil {
			return nil, err
		}
		input.ExpiresAtSet = true
		input.ExpiresAt = expiresAt
	}
	if fields.present["group_id"] {
		input.GroupIDSet = true
		input.GroupID = fields.GroupID
	}
	if fields.present["type"] {
		codeType := strings.ToLower(strings.TrimSpace(derefStringPtr(fields.Type)))
		switch codeType {
		case service.RedeemTypeBalance, service.RedeemTypeConcurrency, service.RedeemTypeSubscription, service.RedeemTypeInvitation:
			input.Type = &codeType
		default:
			return nil, infraerrors.BadRequest("REDEEM_CODE_TYPE_INVALID", "invalid redeem code type")
		}
	}
	if fields.present["value"] {
		if fields.Value == nil {
			return nil, infraerrors.BadRequest("REDEEM_CODE_VALUE_INVALID", "value must be a number")
		}
		input.Value = fields.Value
	}
	if fields.present["validity_days"] {
		if fields.ValidityDays == nil {
			return nil, infraerrors.BadRequest("REDEEM_CODE_VALIDITY_INVALID", "validity_days must be a number")
		}
		input.ValidityDays = fields.ValidityDays
	}
	return input, nil
}

func derefStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func logRedeemCodeBatchUpdateAudit(c *gin.Context, req BatchUpdateRedeemCodesRequest, updated int64) {
	fields := []zap.Field{
		zap.String("component", "audit.admin.redeem_codes.batch_update"),
		zap.Int("requested_count", len(req.IDs)),
		zap.Int64("updated_count", updated),
		zap.Strings("fields", req.Fields.fieldNames()),
	}
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		fields = append(fields, zap.Int64("operator_user_id", subject.UserID))
	}
	if requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		fields = append(fields, zap.String("request_id", strings.TrimSpace(requestID)))
	}
	if clientRequestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
		fields = append(fields, zap.String("client_request_id", strings.TrimSpace(clientRequestID)))
	}
	logger.With(fields...).Info("admin redeem codes batch updated")
}

func (f redeemCodeBatchFields) fieldNames() []string {
	names := make([]string, 0, len(f.present))
	for _, key := range []string{"status", "notes", "expires_at", "group_id", "type", "value", "validity_days"} {
		if f.present[key] {
			names = append(names, key)
		}
	}
	return names
}

func parseOptionalRedeemExpiresAt(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			if !parsed.After(time.Now()) {
				return nil, infraerrors.BadRequest("REDEEM_CODE_EXPIRES_AT_INVALID", "expires_at must be in the future")
			}
			return &parsed, nil
		}
	}
	return nil, infraerrors.BadRequest("REDEEM_CODE_EXPIRES_AT_INVALID", "expires_at must be a valid date or RFC3339 timestamp")
}

func parseOptionalRedeemExpiresAtOrDays(raw *string, days *int) (*time.Time, error) {
	if days != nil && raw != nil && strings.TrimSpace(*raw) != "" {
		return nil, infraerrors.BadRequest("REDEEM_CODE_EXPIRY_CONFLICT", "expires_at and expires_in_days cannot be used together")
	}
	if days != nil {
		if *days <= 0 {
			return nil, infraerrors.BadRequest("REDEEM_CODE_EXPIRES_IN_DAYS_INVALID", "expires_in_days must be greater than zero")
		}
		expiresAt := time.Now().AddDate(0, 0, *days)
		return &expiresAt, nil
	}
	return parseOptionalRedeemExpiresAt(raw)
}
