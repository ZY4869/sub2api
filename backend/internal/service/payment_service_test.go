package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type paymentRepoStub struct {
	orders                  map[string]*PaymentOrder
	ordersByIntent          map[string]*PaymentOrder
	ordersByIdempotency     map[string]*PaymentOrder
	ordersByResume          map[string]*PaymentOrder
	refunds                 map[string]*PaymentRefund
	refundsByIdempotency    map[string]*PaymentRefund
	events                  map[string]*PaymentEvent
	walletAdds              int
	subscriptionAssignments int
	statusUpdates           []string
	txRuns                  int
	failWallet              bool
	failStatus              bool
}

func newPaymentRepoStub() *paymentRepoStub {
	return &paymentRepoStub{
		orders:               map[string]*PaymentOrder{},
		ordersByIntent:       map[string]*PaymentOrder{},
		ordersByIdempotency:  map[string]*PaymentOrder{},
		ordersByResume:       map[string]*PaymentOrder{},
		refunds:              map[string]*PaymentRefund{},
		refundsByIdempotency: map[string]*PaymentRefund{},
		events:               map[string]*PaymentEvent{},
	}
}

func (r *paymentRepoStub) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	r.txRuns++
	return fn(ctx)
}

func (r *paymentRepoStub) CreateOrder(_ context.Context, order *PaymentOrder) error {
	order.ID = int64(len(r.orders) + 1)
	cp := *order
	r.orders[order.OrderNo] = &cp
	if order.ProviderIntentID != "" {
		r.ordersByIntent[order.ProviderIntentID] = &cp
	}
	if order.IdempotencyKeyHash != "" {
		r.ordersByIdempotency[order.IdempotencyKeyHash] = &cp
	}
	if order.ResumeTokenHash != "" {
		r.ordersByResume[order.ResumeTokenHash] = &cp
	}
	return nil
}

func (r *paymentRepoStub) UpdateOrderProviderIntent(_ context.Context, orderNo string, providerIntentID string, status string) error {
	order := r.orders[orderNo]
	order.ProviderIntentID = providerIntentID
	order.Status = status
	r.ordersByIntent[providerIntentID] = order
	return nil
}

func (r *paymentRepoStub) GetOrderByOrderNo(_ context.Context, orderNo string) (*PaymentOrder, error) {
	if order := r.orders[orderNo]; order != nil {
		return order, nil
	}
	return nil, ErrPaymentOrderNotFound
}

func (r *paymentRepoStub) GetOrderByUserIdempotencyHash(_ context.Context, _ int64, hash string) (*PaymentOrder, error) {
	if order := r.ordersByIdempotency[hash]; order != nil {
		return order, nil
	}
	return nil, ErrPaymentOrderNotFound
}

func (r *paymentRepoStub) GetOrderByResumeTokenHash(_ context.Context, hash string) (*PaymentOrder, error) {
	if order := r.ordersByResume[hash]; order != nil {
		return order, nil
	}
	return nil, ErrPaymentOrderNotFound
}

func (r *paymentRepoStub) GetOrderByProviderIntent(_ context.Context, _, providerIntentID string) (*PaymentOrder, error) {
	if order := r.ordersByIntent[providerIntentID]; order != nil {
		return order, nil
	}
	return nil, ErrPaymentOrderNotFound
}

func (r *paymentRepoStub) UpdateOrderStatus(_ context.Context, orderNo, status string, paidAt, refundedAt *time.Time) error {
	if r.failStatus {
		return errors.New("status failed")
	}
	order := r.orders[orderNo]
	order.Status = status
	order.PaidAt = paidAt
	order.RefundedAt = refundedAt
	r.statusUpdates = append(r.statusUpdates, status)
	return nil
}

func (r *paymentRepoStub) CancelOrder(_ context.Context, orderNo string) error {
	r.orders[orderNo].Status = PaymentStatusCancelled
	return nil
}

func (r *paymentRepoStub) ListOrders(context.Context, pagination.PaginationParams, string, string, string, *int64) ([]PaymentOrder, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r *paymentRepoStub) CreateEventIfAbsent(_ context.Context, event *PaymentEvent) (bool, error) {
	if r.events[event.ProviderEventID] != nil {
		return false, nil
	}
	cp := *event
	r.events[event.ProviderEventID] = &cp
	return true, nil
}

func (r *paymentRepoStub) MarkEventProcessed(_ context.Context, _, providerEventID, status, orderNo, reason string) error {
	event := r.events[providerEventID]
	event.EventStatus = status
	event.OrderNo = orderNo
	event.ErrorReason = reason
	return nil
}

func (r *paymentRepoStub) CreateRefund(_ context.Context, refund *PaymentRefund) error {
	refund.ID = 1
	r.refunds[refund.RefundNo] = refund
	if refund.IdempotencyKeyHash != "" {
		r.refundsByIdempotency[refund.IdempotencyKeyHash] = refund
	}
	return nil
}

func (r *paymentRepoStub) SumSuccessfulRefundAmount(_ context.Context, orderNo string) (int64, error) {
	var total int64
	for _, refund := range r.refunds {
		if refund.OrderNo != orderNo {
			continue
		}
		if refund.Status == PaymentRefundStatusAccepted || refund.Status == PaymentRefundStatusSettled {
			total += refund.AmountMinor
		}
	}
	return total, nil
}

func (r *paymentRepoStub) GetRefundByOrderIdempotencyHash(_ context.Context, _ string, hash string) (*PaymentRefund, error) {
	if refund := r.refundsByIdempotency[hash]; refund != nil {
		return refund, nil
	}
	return nil, ErrPaymentOrderNotFound
}

func (r *paymentRepoStub) UpdateRefundProvider(_ context.Context, refundNo, providerRefundID, status string) error {
	for _, refund := range r.refundsByIdempotency {
		if refund.RefundNo == refundNo {
			refund.ProviderRefundID = providerRefundID
			refund.Status = status
		}
	}
	return nil
}

func (r *paymentRepoStub) AddWalletBalance(context.Context, int64, string, float64) error {
	r.walletAdds++
	if r.failWallet {
		return errors.New("wallet failed")
	}
	return nil
}

func (r *paymentRepoStub) AssignOrExtendSubscription(context.Context, *AssignSubscriptionInput) error {
	r.subscriptionAssignments++
	return nil
}

type airwallexStub struct {
	intentReq  AirwallexPaymentIntentRequest
	refundReq  AirwallexRefundRequest
	failIntent error
	failRefund error
	creates    int
	retrieves  int
}

func (a *airwallexStub) CreatePaymentIntent(_ context.Context, _ PaymentSettings, req AirwallexPaymentIntentRequest) (*AirwallexPaymentIntentResponse, error) {
	a.creates++
	a.intentReq = req
	if a.failIntent != nil {
		return nil, a.failIntent
	}
	return &AirwallexPaymentIntentResponse{ID: "int_123", ClientSecret: "secret", Status: "REQUIRES_PAYMENT_METHOD"}, nil
}

func (a *airwallexStub) RetrievePaymentIntent(context.Context, PaymentSettings, string) (*AirwallexPaymentIntentResponse, error) {
	a.retrieves++
	return &AirwallexPaymentIntentResponse{ID: "int_123", ClientSecret: "secret2"}, nil
}

func (a *airwallexStub) CreateRefund(_ context.Context, _ PaymentSettings, req AirwallexRefundRequest) (*AirwallexRefundResponse, error) {
	a.refundReq = req
	if a.failRefund != nil {
		return nil, a.failRefund
	}
	return &AirwallexRefundResponse{ID: "rf_provider", Status: "succeeded"}, nil
}

func (a *airwallexStub) VerifyWebhookSignature(string, string, string, []byte) error { return nil }

func paymentTestSettings() PaymentSettings {
	return PaymentSettings{
		Enabled:                          true,
		AirwallexEnabled:                 true,
		AirwallexEnv:                     "demo",
		AirwallexClientID:                "client",
		AirwallexAPIKey:                  "key",
		AirwallexAPIKeyConfigured:        true,
		AirwallexWebhookSecret:           "whsec",
		AirwallexWebhookSecretConfigured: true,
		AllowedCurrencies:                []string{"USD", "CNY", "HKD"},
		DefaultCurrency:                  "USD",
		MinTopupAmount:                   1,
		MaxTopupAmount:                   1000,
		SubscriptionPlans: []PaymentSubscriptionPlan{{
			PlanID: "pro", GroupID: 9, ValidityDays: 30, Enabled: true,
			PricesByCurrency: map[string]float64{"USD": 12.5},
		}},
	}
}

func newPaymentServiceTestSubject(repo PaymentRepository, air AirwallexClient) *PaymentService {
	return &PaymentService{
		repo:      repo,
		airwallex: air,
		paymentSettingsOverride: func(context.Context) PaymentSettings {
			return paymentTestSettings()
		},
	}
}

func TestPaymentServiceCreateOrderBuildsIntentAndMetrics(t *testing.T) {
	resetPaymentRuntimeMetricsForTest()
	repo := newPaymentRepoStub()
	air := &airwallexStub{}
	svc := newPaymentServiceTestSubject(repo, air)

	result, err := svc.CreateOrder(context.Background(), CreatePaymentOrderInput{
		UserID: 7, ProductType: PaymentProductBalanceTopup, Amount: 12.34, Currency: "USD", CountryCode: "us",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1234), result.Order.AmountMinor)
	require.Equal(t, "US", result.Order.CountryCode)
	require.Equal(t, "int_123", result.IntentID)
	require.Equal(t, "client", result.ClientID)
	require.Equal(t, result.Order.OrderNo, air.intentReq.RequestID)
	require.Equal(t, int64(1234), air.intentReq.AmountMinor)
	require.Equal(t, int64(1), SnapshotPaymentRuntimeMetrics().CreateSuccess)
}

func TestPaymentServiceRejectsUnsupportedCurrency(t *testing.T) {
	resetPaymentRuntimeMetricsForTest()
	svc := newPaymentServiceTestSubject(newPaymentRepoStub(), &airwallexStub{})
	_, err := svc.CreateOrder(context.Background(), CreatePaymentOrderInput{
		UserID: 7, ProductType: PaymentProductBalanceTopup, Amount: 12.34, Currency: "EUR",
	})
	require.ErrorIs(t, err, ErrPaymentUnsupportedCurrency)
	require.Equal(t, int64(1), SnapshotPaymentRuntimeMetrics().CreateFailure)
}

func TestPaymentServiceCreateOrderIdempotencyReplayReusesOrder(t *testing.T) {
	repo := newPaymentRepoStub()
	air := &airwallexStub{}
	svc := newPaymentServiceTestSubject(repo, air)
	input := CreatePaymentOrderInput{
		UserID: 7, ProductType: PaymentProductBalanceTopup, Amount: 12.34, Currency: "USD", IdempotencyKey: "same-key",
	}

	first, err := svc.CreateOrder(context.Background(), input)
	require.NoError(t, err)
	second, err := svc.CreateOrder(context.Background(), input)
	require.NoError(t, err)

	require.Len(t, repo.orders, 1)
	require.Equal(t, first.Order.OrderNo, second.Order.OrderNo)
	require.Equal(t, "secret2", second.ClientSecret)
	require.Equal(t, 1, air.creates)
	require.Equal(t, 1, air.retrieves)
}

func TestPaymentServiceResumeOrderByOrderNoChecksOwnerAndReturnsClientSecret(t *testing.T) {
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup, Status: PaymentStatusPending,
		Provider: PaymentProviderAirwallex, ProviderEnv: "demo", ProviderIntentID: "int_123",
		AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	air := &airwallexStub{}
	svc := newPaymentServiceTestSubject(repo, air)

	result, err := svc.ResumeOrderByOrderNo(context.Background(), order.OrderNo, 7)
	require.NoError(t, err)
	require.Equal(t, order.OrderNo, result.Order.OrderNo)
	require.Equal(t, "secret2", result.ClientSecret)
	require.Equal(t, 1, air.retrieves)

	_, err = svc.ResumeOrderByOrderNo(context.Background(), order.OrderNo, 8)
	require.ErrorIs(t, err, ErrPaymentOrderForbidden)
}

func TestPaymentServiceWebhookPaidRedactsPayloadAndFulfillsInTransaction(t *testing.T) {
	resetPaymentRuntimeMetricsForTest()
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		ID: 3, OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup,
		Status: PaymentStatusPending, Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123",
		AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	repo.ordersByIntent[order.ProviderIntentID] = order
	svc := newPaymentServiceTestSubject(repo, &airwallexStub{})

	body := []byte(`{"id":"evt_1","name":"payment_intent.succeeded","data":{"id":"int_123","client_secret":"secret","customer_email":"a@example.com"}}`)
	require.NoError(t, svc.HandleAirwallexWebhook(context.Background(), "ts", "sig", body))

	require.Equal(t, 1, repo.txRuns)
	require.Equal(t, 1, repo.walletAdds)
	require.Equal(t, PaymentStatusPaid, order.Status)
	var redacted map[string]any
	require.NoError(t, json.Unmarshal(repo.events["evt_1"].PayloadRedactedJSON, &redacted))
	require.Equal(t, "[REDACTED]", redacted["data"].(map[string]any)["client_secret"])
	require.Equal(t, "[REDACTED]", redacted["data"].(map[string]any)["customer_email"])
	require.Equal(t, int64(1), SnapshotPaymentRuntimeMetrics().WebhookSuccess)
}

func TestPaymentServiceWebhookDuplicateEventSkipsFulfillment(t *testing.T) {
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		ID: 3, OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup,
		Status: PaymentStatusPending, Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123",
		AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	repo.ordersByIntent[order.ProviderIntentID] = order
	svc := newPaymentServiceTestSubject(repo, &airwallexStub{})

	body := []byte(`{"id":"evt_1","name":"payment_intent.succeeded","data":{"id":"int_123"}}`)
	require.NoError(t, svc.HandleAirwallexWebhook(context.Background(), "ts", "sig", body))
	require.NoError(t, svc.HandleAirwallexWebhook(context.Background(), "ts", "sig", body))

	require.Equal(t, 1, repo.txRuns)
	require.Equal(t, 1, repo.walletAdds)
	require.Equal(t, PaymentStatusPaid, order.Status)
	require.Equal(t, "processed", repo.events["evt_1"].EventStatus)
}

func TestPaymentServiceWebhookPaidDoesNotMarkPaidWhenFulfillmentFails(t *testing.T) {
	repo := newPaymentRepoStub()
	repo.failWallet = true
	order := &PaymentOrder{
		ID: 3, OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup,
		Status: PaymentStatusPending, Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123",
		AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	repo.ordersByIntent[order.ProviderIntentID] = order
	svc := newPaymentServiceTestSubject(repo, &airwallexStub{})

	body := []byte(`{"id":"evt_1","name":"payment_intent.succeeded","data":{"id":"int_123"}}`)
	require.Error(t, svc.HandleAirwallexWebhook(context.Background(), "ts", "sig", body))
	require.Equal(t, PaymentStatusPending, order.Status)
	require.Equal(t, "failed", repo.events["evt_1"].EventStatus)
}

func TestPaymentServiceRefundOrderUsesIdempotencyAndProvider(t *testing.T) {
	resetPaymentRuntimeMetricsForTest()
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup, Status: PaymentStatusPaid,
		Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123", AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	air := &airwallexStub{}
	svc := newPaymentServiceTestSubject(repo, air)

	refund, err := svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 500, Reason: "requested", RequestedBy: 1, IdempotencyKey: "same",
	})
	require.NoError(t, err)
	require.Equal(t, PaymentRefundStatusSettled, refund.Status)
	require.Equal(t, "int_123", air.refundReq.IntentID)
	require.Equal(t, int64(500), air.refundReq.AmountMinor)
	require.Equal(t, PaymentStatusPartialRefunded, order.Status)
	require.NotNil(t, order.RefundedAt)

	again, err := svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 500, RequestedBy: 1, IdempotencyKey: "same",
	})
	require.NoError(t, err)
	require.Equal(t, refund.RefundNo, again.RefundNo)
	require.Equal(t, int64(2), SnapshotPaymentRuntimeMetrics().RefundSuccess)
}

func TestPaymentServiceFullRefundMarksOrderRefunded(t *testing.T) {
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup, Status: PaymentStatusPaid,
		Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123", AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	svc := newPaymentServiceTestSubject(repo, &airwallexStub{})

	refund, err := svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 1500, Reason: "requested", RequestedBy: 1, IdempotencyKey: "full",
	})

	require.NoError(t, err)
	require.Equal(t, PaymentRefundStatusSettled, refund.Status)
	require.Equal(t, PaymentStatusRefunded, order.Status)
	require.NotNil(t, order.RefundedAt)
}

func TestPaymentServiceCumulativePartialRefundMarksFullAndRejectsOverRefund(t *testing.T) {
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup, Status: PaymentStatusPaid,
		Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123", AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	svc := newPaymentServiceTestSubject(repo, &airwallexStub{})

	first, err := svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 500, Reason: "first", RequestedBy: 1, IdempotencyKey: "first",
	})
	require.NoError(t, err)
	require.Equal(t, PaymentRefundStatusSettled, first.Status)
	require.Equal(t, PaymentStatusPartialRefunded, order.Status)

	second, err := svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 1000, Reason: "second", RequestedBy: 1, IdempotencyKey: "second",
	})
	require.NoError(t, err)
	require.Equal(t, PaymentRefundStatusSettled, second.Status)
	require.Equal(t, PaymentStatusRefunded, order.Status)

	_, err = svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 1, Reason: "too much", RequestedBy: 1, IdempotencyKey: "third",
	})
	require.ErrorIs(t, err, ErrPaymentOrderNotRefundable)
}
