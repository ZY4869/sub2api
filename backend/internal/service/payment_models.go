package service

import (
	"context"
	"encoding/json"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	PaymentProviderAirwallex = "airwallex"

	PaymentProductBalanceTopup = "balance_topup"
	PaymentProductSubscription = "subscription"

	PaymentStatusCreated         = "created"
	PaymentStatusPending         = "pending"
	PaymentStatusPaid            = "paid"
	PaymentStatusFailed          = "failed"
	PaymentStatusCancelled       = "cancelled"
	PaymentStatusExpired         = "expired"
	PaymentStatusPartialRefunded = "partial_refunded"
	PaymentStatusRefunded        = "refunded"

	PaymentRefundStatusReceived = "received"
	PaymentRefundStatusAccepted = "accepted"
	PaymentRefundStatusSettled  = "settled"
	PaymentRefundStatusFailed   = "failed"
)

var (
	ErrPaymentDisabled              = infraerrors.Forbidden("PAYMENT_DISABLED", "payment is disabled")
	ErrPaymentProviderNotConfigured = infraerrors.BadRequest("PAYMENT_PROVIDER_NOT_CONFIGURED", "payment provider is not configured")
	ErrPaymentInvalidProduct        = infraerrors.BadRequest("PAYMENT_INVALID_PRODUCT", "invalid payment product")
	ErrPaymentInvalidAmount         = infraerrors.BadRequest("PAYMENT_INVALID_AMOUNT", "invalid payment amount")
	ErrPaymentUnsupportedCurrency   = infraerrors.BadRequest("PAYMENT_UNSUPPORTED_CURRENCY", "unsupported payment currency")
	ErrPaymentServiceUnavailable    = infraerrors.ServiceUnavailable("PAYMENT_SERVICE_UNAVAILABLE", "payment service unavailable")
	ErrPaymentOrderNotFound         = infraerrors.NotFound("PAYMENT_ORDER_NOT_FOUND", "payment order not found")
	ErrPaymentOrderForbidden        = infraerrors.Forbidden("PAYMENT_ORDER_FORBIDDEN", "payment order does not belong to current user")
	ErrPaymentOrderNotCancelable    = infraerrors.Conflict("PAYMENT_ORDER_NOT_CANCELABLE", "payment order cannot be cancelled")
	ErrPaymentOrderNotRefundable    = infraerrors.Conflict("PAYMENT_ORDER_NOT_REFUNDABLE", "payment order cannot be refunded")
	ErrPaymentWebhookInvalid        = infraerrors.BadRequest("PAYMENT_WEBHOOK_INVALID", "invalid payment webhook")
	ErrPaymentProviderFailed        = infraerrors.ServiceUnavailable("PAYMENT_PROVIDER_FAILED", "payment provider request failed")
)

type PaymentOrder struct {
	ID                    int64
	OrderNo               string
	UserID                int64
	ProductType           string
	Status                string
	Provider              string
	ProviderEnv           string
	AmountMinor           int64
	RefundedAmountMinor   int64
	RefundableAmountMinor int64
	Currency              string
	CountryCode           string
	ProviderIntentID      string
	ResumeTokenHash       string
	IdempotencyKeyHash    string
	SnapshotJSON          json.RawMessage
	PaidAt                *time.Time
	RefundedAt            *time.Time
	ExpiresAt             *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type PaymentEvent struct {
	ID                  int64
	Provider            string
	ProviderEventID     string
	OrderNo             string
	EventType           string
	EventStatus         string
	PayloadHash         string
	PayloadRedactedJSON json.RawMessage
	ProcessedAt         *time.Time
	ErrorReason         string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type PaymentRefund struct {
	ID                 int64
	RefundNo           string
	OrderNo            string
	ProviderRefundID   string
	AmountMinor        int64
	Currency           string
	Reason             string
	Status             string
	RequestedBy        *int64
	IdempotencyKeyHash string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type PaymentSubscriptionPlan struct {
	PlanID           string             `json:"plan_id"`
	Name             string             `json:"name"`
	GroupID          int64              `json:"group_id"`
	ValidityDays     int                `json:"validity_days"`
	PricesByCurrency map[string]float64 `json:"prices_by_currency"`
	Enabled          bool               `json:"enabled"`
}

type PaymentSettings struct {
	Enabled                          bool
	AirwallexEnabled                 bool
	AirwallexEnv                     string
	AirwallexClientID                string
	AirwallexAPIKey                  string
	AirwallexAPIKeyConfigured        bool
	AirwallexWebhookSecret           string
	AirwallexWebhookSecretConfigured bool
	MobileForceQRCodeEnabled         bool
	AllowedCurrencies                []string
	DefaultCurrency                  string
	MinTopupAmount                   float64
	MaxTopupAmount                   float64
	SubscriptionPlans                []PaymentSubscriptionPlan
	FrontendURL                      string
}

type CreatePaymentOrderInput struct {
	UserID         int64
	ProductType    string
	Amount         float64
	Currency       string
	CountryCode    string
	PlanID         string
	GroupID        int64
	ValidityDays   int
	IdempotencyKey string
	ReturnURL      string
}

type CreatePaymentOrderResult struct {
	Order        *PaymentOrder
	ClientSecret string
	ClientID     string
	IntentID     string
	ResumeToken  string
	ProviderEnv  string
	PaymentMode  string
}

type ResumePaymentOrderResult struct {
	Order        *PaymentOrder
	ClientSecret string
	ClientID     string
	IntentID     string
	ProviderEnv  string
	PaymentMode  string
}

type RefundPaymentOrderInput struct {
	OrderNo        string
	AmountMinor    int64
	Reason         string
	RequestedBy    int64
	IdempotencyKey string
}

type AirwallexPaymentIntentRequest struct {
	RequestID   string
	AmountMinor int64
	Currency    string
	OrderNo     string
	Descriptor  string
	ReturnURL   string
	Metadata    map[string]string
}

type AirwallexPaymentIntentResponse struct {
	ID           string
	ClientSecret string
	Status       string
}

type AirwallexRefundRequest struct {
	RequestID        string
	IntentID         string
	PaymentAttemptID string
	AmountMinor      int64
	Currency         string
	Reason           string
}

type AirwallexRefundResponse struct {
	ID     string
	Status string
}

type PaymentRepository interface {
	RunInTx(ctx context.Context, fn func(context.Context) error) error
	CreateOrder(ctx context.Context, order *PaymentOrder) error
	UpdateOrderProviderIntent(ctx context.Context, orderNo string, providerIntentID string, status string) error
	GetOrderByOrderNo(ctx context.Context, orderNo string) (*PaymentOrder, error)
	GetOrderByOrderNoForUpdate(ctx context.Context, orderNo string) (*PaymentOrder, error)
	GetOrderByUserIdempotencyHash(ctx context.Context, userID int64, idempotencyKeyHash string) (*PaymentOrder, error)
	GetOrderByResumeTokenHash(ctx context.Context, tokenHash string) (*PaymentOrder, error)
	GetOrderByProviderIntent(ctx context.Context, provider, providerIntentID string) (*PaymentOrder, error)
	UpdateOrderStatus(ctx context.Context, orderNo, status string, paidAt, refundedAt *time.Time) error
	CancelOrder(ctx context.Context, orderNo string) error
	ListOrders(ctx context.Context, params pagination.PaginationParams, status, provider, productType string, userID *int64) ([]PaymentOrder, *pagination.PaginationResult, error)
	CreateEventIfAbsent(ctx context.Context, event *PaymentEvent) (bool, error)
	MarkEventProcessed(ctx context.Context, provider, providerEventID, status, orderNo, errorReason string) error
	CreateRefund(ctx context.Context, refund *PaymentRefund) error
	GetRefundByOrderIdempotencyHash(ctx context.Context, orderNo string, idempotencyKeyHash string) (*PaymentRefund, error)
	UpdateRefundProvider(ctx context.Context, refundNo, providerRefundID, status string) error
	SumSuccessfulRefundAmount(ctx context.Context, orderNo string) (int64, error)
	AddWalletBalance(ctx context.Context, userID int64, currency string, amount float64) error
	AssignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput) error
}

type AirwallexClient interface {
	CreatePaymentIntent(ctx context.Context, settings PaymentSettings, req AirwallexPaymentIntentRequest) (*AirwallexPaymentIntentResponse, error)
	RetrievePaymentIntent(ctx context.Context, settings PaymentSettings, intentID string) (*AirwallexPaymentIntentResponse, error)
	CreateRefund(ctx context.Context, settings PaymentSettings, req AirwallexRefundRequest) (*AirwallexRefundResponse, error)
	VerifyWebhookSignature(secret string, timestamp string, signature string, body []byte) error
}
