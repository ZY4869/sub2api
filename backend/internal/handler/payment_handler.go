package handler

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
}

func NewPaymentHandler(paymentService *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

type createPaymentOrderRequest struct {
	ProductType string  `json:"product_type"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	CountryCode string  `json:"country_code"`
	PlanID      string  `json:"plan_id"`
	ReturnURL   string  `json:"return_url"`
}

func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	if h == nil || h.paymentService == nil {
		response.ErrorFrom(c, service.ErrPaymentServiceUnavailable)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req createPaymentOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	payload := struct {
		UserID int64                     `json:"user_id"`
		Body   createPaymentOrderRequest `json:"body"`
	}{
		UserID: subject.UserID,
		Body: createPaymentOrderRequest{
			ProductType: strings.TrimSpace(req.ProductType),
			Amount:      req.Amount,
			Currency:    strings.ToUpper(strings.TrimSpace(req.Currency)),
			CountryCode: strings.ToUpper(strings.TrimSpace(req.CountryCode)),
			PlanID:      strings.TrimSpace(req.PlanID),
			ReturnURL:   strings.TrimSpace(req.ReturnURL),
		},
	}
	input := service.CreatePaymentOrderInput{
		UserID:         subject.UserID,
		ProductType:    strings.TrimSpace(req.ProductType),
		Amount:         req.Amount,
		Currency:       req.Currency,
		CountryCode:    req.CountryCode,
		PlanID:         req.PlanID,
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		ReturnURL:      req.ReturnURL,
	}
	executePaymentCreateOrderIdempotentJSON(c, "payment.orders.create", payload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		result, err := h.paymentService.CreateOrder(ctx, input)
		if err != nil {
			return nil, err
		}
		return paymentOrderResultDTO(result), nil
	}, func(ctx context.Context) (any, error) {
		result, err := h.paymentService.CreateOrder(ctx, input)
		if err != nil {
			return nil, err
		}
		return paymentOrderResultDTO(result), nil
	})
}

func (h *PaymentHandler) GetOrder(c *gin.Context) {
	if h == nil || h.paymentService == nil {
		response.ErrorFrom(c, service.ErrPaymentServiceUnavailable)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	order, err := h.paymentService.GetOrderForUser(c.Request.Context(), subject.UserID, c.Param("order_no"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, paymentOrderDTO(order))
}

func (h *PaymentHandler) ResumeOrder(c *gin.Context) {
	if h == nil || h.paymentService == nil {
		response.ErrorFrom(c, service.ErrPaymentServiceUnavailable)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	result, err := h.paymentService.ResumeOrder(c.Request.Context(), c.Param("resume_token"), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"order":         paymentOrderDTO(result.Order),
		"client_secret": result.ClientSecret,
		"client_id":     result.ClientID,
		"intent_id":     result.IntentID,
		"provider_env":  result.ProviderEnv,
		"payment_mode":  result.PaymentMode,
	})
}

func (h *PaymentHandler) ResumeOrderByOrderNo(c *gin.Context) {
	if h == nil || h.paymentService == nil {
		response.ErrorFrom(c, service.ErrPaymentServiceUnavailable)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	result, err := h.paymentService.ResumeOrderByOrderNo(c.Request.Context(), c.Param("order_no"), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"order":         paymentOrderDTO(result.Order),
		"client_secret": result.ClientSecret,
		"client_id":     result.ClientID,
		"intent_id":     result.IntentID,
		"provider_env":  result.ProviderEnv,
		"payment_mode":  result.PaymentMode,
	})
}

func (h *PaymentHandler) CancelOrder(c *gin.Context) {
	if h == nil || h.paymentService == nil {
		response.ErrorFrom(c, service.ErrPaymentServiceUnavailable)
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	if err := h.paymentService.CancelOrder(c.Request.Context(), subject.UserID, c.Param("order_no")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"order_no": c.Param("order_no")})
}

func (h *PaymentHandler) AirwallexWebhook(c *gin.Context) {
	if h == nil || h.paymentService == nil {
		response.ErrorFrom(c, service.ErrPaymentServiceUnavailable)
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20))
	if err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	timestamp := firstHeader(c, "x-timestamp", "X-Timestamp")
	signature := firstHeader(c, "x-signature", "X-Signature")
	if err := h.paymentService.HandleAirwallexWebhook(c.Request.Context(), timestamp, signature, body); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"received": true})
}

func firstHeader(c *gin.Context, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(c.GetHeader(name)); value != "" {
			return value
		}
	}
	return ""
}

func paymentOrderResultDTO(result *service.CreatePaymentOrderResult) gin.H {
	if result == nil {
		return gin.H{}
	}
	return gin.H{
		"order":         paymentOrderDTO(result.Order),
		"client_secret": result.ClientSecret,
		"client_id":     result.ClientID,
		"intent_id":     result.IntentID,
		"resume_token":  result.ResumeToken,
		"provider_env":  result.ProviderEnv,
		"payment_mode":  result.PaymentMode,
	}
}

func paymentOrderDTO(order *service.PaymentOrder) gin.H {
	if order == nil {
		return gin.H{}
	}
	var snapshot any = map[string]any{}
	if len(order.SnapshotJSON) > 0 {
		_ = json.Unmarshal(order.SnapshotJSON, &snapshot)
	}
	return gin.H{
		"order_no":                order.OrderNo,
		"user_id":                 order.UserID,
		"product_type":            order.ProductType,
		"status":                  order.Status,
		"provider":                order.Provider,
		"provider_env":            order.ProviderEnv,
		"amount_minor":            order.AmountMinor,
		"amount":                  service.PaymentMinorToAmount(order.AmountMinor, order.Currency),
		"refunded_amount_minor":   order.RefundedAmountMinor,
		"refunded_amount":         service.PaymentMinorToAmount(order.RefundedAmountMinor, order.Currency),
		"refundable_amount_minor": order.RefundableAmountMinor,
		"refundable_amount":       service.PaymentMinorToAmount(order.RefundableAmountMinor, order.Currency),
		"currency":                order.Currency,
		"country_code":            order.CountryCode,
		"provider_intent_id":      order.ProviderIntentID,
		"snapshot":                snapshot,
		"paid_at":                 order.PaidAt,
		"refunded_at":             order.RefundedAt,
		"expires_at":              order.ExpiresAt,
		"created_at":              order.CreatedAt,
		"updated_at":              order.UpdatedAt,
	}
}

type AdminPaymentHandler struct {
	paymentService *service.PaymentService
}

func NewAdminPaymentHandler(paymentService *service.PaymentService) *AdminPaymentHandler {
	return &AdminPaymentHandler{paymentService: paymentService}
}

func (h *AdminPaymentHandler) ListOrders(c *gin.Context) {
	if h == nil || h.paymentService == nil {
		response.ErrorFrom(c, service.ErrPaymentServiceUnavailable)
		return
	}
	page, pageSize := response.ParsePagination(c)
	var userID *int64
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			userID = &v
		}
	}
	items, result, err := h.paymentService.ListOrders(c.Request.Context(), servicePagination(page, pageSize), c.Query("status"), c.Query("provider"), c.Query("product_type"), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]gin.H, 0, len(items))
	for i := range items {
		out = append(out, paymentOrderDTO(&items[i]))
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}

type refundPaymentOrderRequest struct {
	AmountMinor int64  `json:"amount_minor"`
	Reason      string `json:"reason"`
}

func (h *AdminPaymentHandler) RefundOrder(c *gin.Context) {
	if h == nil || h.paymentService == nil {
		response.ErrorFrom(c, service.ErrPaymentServiceUnavailable)
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	var req refundPaymentOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	payload := struct {
		AdminID int64                     `json:"admin_id"`
		OrderNo string                    `json:"order_no"`
		Body    refundPaymentOrderRequest `json:"body"`
	}{AdminID: subject.UserID, OrderNo: c.Param("order_no"), Body: req}
	executeAdminPaymentIdempotentJSON(c, "admin.payment.orders.refund", payload, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		refund, err := h.paymentService.RefundOrder(ctx, service.RefundPaymentOrderInput{
			OrderNo:        c.Param("order_no"),
			AmountMinor:    req.AmountMinor,
			Reason:         req.Reason,
			RequestedBy:    subject.UserID,
			IdempotencyKey: c.GetHeader("Idempotency-Key"),
		})
		if err != nil {
			return nil, err
		}
		return gin.H{
			"refund_no":          refund.RefundNo,
			"order_no":           refund.OrderNo,
			"provider_refund_id": refund.ProviderRefundID,
			"amount_minor":       refund.AmountMinor,
			"currency":           refund.Currency,
			"status":             refund.Status,
			"created_at":         refund.CreatedAt,
		}, nil
	})
}

func servicePagination(page, pageSize int) pagination.PaginationParams {
	return pagination.PaginationParams{Page: page, PageSize: pageSize}
}

func executeAdminPaymentIdempotentJSON(
	c *gin.Context,
	scope string,
	payload any,
	ttl time.Duration,
	execute func(context.Context) (any, error),
) {
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		data, err := execute(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, data)
		return
	}

	actorScope := "admin:0"
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		actorScope = "admin:" + strconv.FormatInt(subject.UserID, 10)
	}

	result, err := coordinator.Execute(c.Request.Context(), service.IdempotencyExecuteOptions{
		Scope:          scope,
		ActorScope:     actorScope,
		Method:         c.Request.Method,
		Route:          c.FullPath(),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Payload:        payload,
		RequireKey:     true,
		TTL:            ttl,
	}, execute)
	if err != nil {
		if infraerrors.Code(err) == infraerrors.Code(service.ErrIdempotencyStoreUnavail) {
			service.RecordIdempotencyStoreUnavailable(c.FullPath(), scope, "handler_fail_close")
			logger.LegacyPrintf("handler.idempotency", "[Idempotency] store unavailable: method=%s route=%s scope=%s strategy=fail_close", c.Request.Method, c.FullPath(), scope)
		}
		if retryAfter := service.RetryAfterSecondsFromError(err); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		response.ErrorFrom(c, err)
		return
	}
	if result != nil && result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	response.Success(c, result.Data)
}

func executePaymentCreateOrderIdempotentJSON(
	c *gin.Context,
	scope string,
	payload any,
	ttl time.Duration,
	execute func(context.Context) (any, error),
	replay func(context.Context) (any, error),
) {
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		data, err := execute(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, data)
		return
	}

	actorScope := "user:0"
	if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
		actorScope = "user:" + strconv.FormatInt(subject.UserID, 10)
	}

	result, err := coordinator.Execute(c.Request.Context(), service.IdempotencyExecuteOptions{
		Scope:          scope,
		ActorScope:     actorScope,
		Method:         c.Request.Method,
		Route:          c.FullPath(),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Payload:        payload,
		RequireKey:     true,
		TTL:            ttl,
	}, execute)
	if err != nil {
		if infraerrors.Code(err) == infraerrors.Code(service.ErrIdempotencyStoreUnavail) {
			service.RecordIdempotencyStoreUnavailable(c.FullPath(), scope, "handler_fail_close")
			logger.LegacyPrintf("handler.idempotency", "[Idempotency] store unavailable: method=%s route=%s scope=%s strategy=fail_close", c.Request.Method, c.FullPath(), scope)
		}
		if retryAfter := service.RetryAfterSecondsFromError(err); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		response.ErrorFrom(c, err)
		return
	}
	if result != nil && result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
		if replay != nil {
			data, replayErr := replay(c.Request.Context())
			if replayErr == nil {
				response.Success(c, data)
				return
			}
		}
	}
	response.Success(c, result.Data)
}
