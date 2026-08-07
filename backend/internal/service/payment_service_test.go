package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
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
	lastWalletAmount        float64
	walletBalance           float64
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
		walletBalance:        999999,
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

func (r *paymentRepoStub) GetOrderByOrderNoForUpdate(ctx context.Context, orderNo string) (*PaymentOrder, error) {
	return r.GetOrderByOrderNo(ctx, orderNo)
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
		if refund.Status == PaymentRefundStatusSettled {
			total += refund.AmountMinor
		}
	}
	return total, nil
}

func TestPaymentServiceAcceptedRefundDoesNotFinalizeOrder(t *testing.T) {
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		OrderNo: "pay_accepted", UserID: 7, ProductType: PaymentProductBalanceTopup, Status: PaymentStatusPaid,
		Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123", AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	air := &airwallexStub{refundStatus: "ACCEPTED"}
	svc := newPaymentServiceTestSubject(repo, air)

	refund, err := svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 500, Reason: "requested", RequestedBy: 1, IdempotencyKey: "accepted",
	})

	require.NoError(t, err)
	require.Equal(t, PaymentRefundStatusAccepted, refund.Status)
	require.Equal(t, PaymentStatusPaid, order.Status)
	require.Nil(t, order.RefundedAt)
	refundedAmount, err := repo.SumSuccessfulRefundAmount(context.Background(), order.OrderNo)
	require.NoError(t, err)
	require.Zero(t, refundedAmount)
}

func (r *paymentRepoStub) GetRefundByOrderIdempotencyHash(_ context.Context, _ string, hash string) (*PaymentRefund, error) {
	if refund := r.refundsByIdempotency[hash]; refund != nil {
		return refund, nil
	}
	return nil, ErrPaymentOrderNotFound
}

func (r *paymentRepoStub) UpdateRefundProvider(_ context.Context, refundNo, providerRefundID, status string) error {
	for _, refund := range r.refunds {
		if refund.RefundNo == refundNo {
			refund.ProviderRefundID = providerRefundID
			refund.Status = status
		}
	}
	return nil
}

func (r *paymentRepoStub) GetWalletBalance(_ context.Context, _ int64, _ string) (float64, error) {
	return r.walletBalance, nil
}

func (r *paymentRepoStub) AddWalletBalance(_ context.Context, _ int64, _ string, amount float64) error {
	r.walletAdds++
	r.lastWalletAmount = amount
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
	intentReq      AirwallexPaymentIntentRequest
	refundReq      AirwallexRefundRequest
	intentID       string
	clientSecret   string
	refundID       string
	failIntent     error
	failRefund     error
	retrieveStatus string
	refundStatus   string
	creates        int
	retrieves      int
}

func (a *airwallexStub) CreatePaymentIntent(_ context.Context, _ PaymentSettings, req AirwallexPaymentIntentRequest) (*AirwallexPaymentIntentResponse, error) {
	a.creates++
	a.intentReq = req
	if a.failIntent != nil {
		return nil, a.failIntent
	}
	id := a.intentID
	if id == "" {
		id = "int_123"
	}
	secret := a.clientSecret
	if secret == "" {
		secret = "secret"
	}
	return &AirwallexPaymentIntentResponse{ID: id, ClientSecret: secret, Status: "REQUIRES_PAYMENT_METHOD"}, nil
}

func (a *airwallexStub) RetrievePaymentIntent(context.Context, PaymentSettings, string) (*AirwallexPaymentIntentResponse, error) {
	a.retrieves++
	id := a.intentID
	if id == "" {
		id = "int_123"
	}
	secret := a.clientSecret
	if secret == "" {
		secret = "secret2"
	}
	return &AirwallexPaymentIntentResponse{ID: id, ClientSecret: secret, Status: a.retrieveStatus}, nil
}

func (a *airwallexStub) CreateRefund(_ context.Context, _ PaymentSettings, req AirwallexRefundRequest) (*AirwallexRefundResponse, error) {
	a.refundReq = req
	if a.failRefund != nil {
		return nil, a.failRefund
	}
	status := a.refundStatus
	if status == "" {
		status = "succeeded"
	}
	id := a.refundID
	if id == "" {
		id = "rf_provider"
	}
	return &AirwallexRefundResponse{ID: id, Status: status}, nil
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

type paymentAuthCacheInvalidatorStub struct {
	users []int64
}

func (s *paymentAuthCacheInvalidatorStub) InvalidateAuthCacheByKey(context.Context, string) {}

func (s *paymentAuthCacheInvalidatorStub) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	s.users = append(s.users, userID)
}

func (s *paymentAuthCacheInvalidatorStub) InvalidateAuthCacheByGroupID(context.Context, int64) {}

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

func TestPaymentAmountToMinorUsesFixedPointValidation(t *testing.T) {
	minor, err := PaymentAmountToMinor(0.1+0.2, "USD")
	require.NoError(t, err)
	require.Equal(t, int64(30), minor)

	minor, err = PaymentAmountToMinor(12.5, "JPY")
	require.NoError(t, err)
	require.Equal(t, int64(13), minor)

	for _, amount := range []float64{
		math.NaN(),
		math.Inf(1),
		math.Copysign(0, -1),
		MaxBillingAmount + 1,
	} {
		_, err := PaymentAmountToMinor(amount, "USD")
		require.ErrorIs(t, err, ErrPaymentInvalidAmount)
	}
}

func TestPaymentServiceCreateOrderComparesTopupBoundsByMinorUnit(t *testing.T) {
	settings := paymentTestSettings()
	settings.MinTopupAmount = 1.005
	settings.MaxTopupAmount = 1.014
	repo := newPaymentRepoStub()
	svc := newPaymentServiceTestSubject(repo, &airwallexStub{})
	svc.paymentSettingsOverride = func(context.Context) PaymentSettings {
		return settings
	}

	result, err := svc.CreateOrder(context.Background(), CreatePaymentOrderInput{
		UserID: 7, ProductType: PaymentProductBalanceTopup, Amount: 1.01, Currency: "USD",
	})
	require.NoError(t, err)
	require.Equal(t, int64(101), result.Order.AmountMinor)

	_, err = svc.CreateOrder(context.Background(), CreatePaymentOrderInput{
		UserID: 7, ProductType: PaymentProductBalanceTopup, Amount: 1.02, Currency: "USD",
	})
	require.ErrorIs(t, err, ErrPaymentInvalidAmount)
}

func TestPaymentServiceCreateOrderNormalizesSubscriptionPrice(t *testing.T) {
	settings := paymentTestSettings()
	settings.SubscriptionPlans[0].PricesByCurrency["USD"] = 12.345
	svc := newPaymentServiceTestSubject(newPaymentRepoStub(), &airwallexStub{})
	svc.paymentSettingsOverride = func(context.Context) PaymentSettings {
		return settings
	}

	result, err := svc.CreateOrder(context.Background(), CreatePaymentOrderInput{
		UserID: 7, ProductType: PaymentProductSubscription, Currency: "USD", PlanID: "pro",
	})
	require.NoError(t, err)
	require.Equal(t, int64(1235), result.Order.AmountMinor)
}

func TestPaymentServiceCreateSubscriptionOrderConvertsUSDPlanToCNYWhenOptIn(t *testing.T) {
	settings := paymentTestSettings()
	settings.SubscriptionUSDToCNYRate = 7.15
	svc := newPaymentServiceTestSubject(newPaymentRepoStub(), &airwallexStub{})
	svc.paymentSettingsOverride = func(context.Context) PaymentSettings {
		return settings
	}

	result, err := svc.CreateOrder(context.Background(), CreatePaymentOrderInput{
		UserID: 7, ProductType: PaymentProductSubscription, Currency: "CNY", PlanID: "pro",
	})

	require.NoError(t, err)
	require.Equal(t, int64(8938), result.Order.AmountMinor)
	var snapshot paymentOrderSnapshot
	require.NoError(t, json.Unmarshal(result.Order.SnapshotJSON, &snapshot))
	require.Equal(t, "CNY", snapshot.Currency)
	require.Equal(t, "USD", snapshot.ConvertedFrom)
	require.Equal(t, 7.15, snapshot.ConversionRate)
}

func TestPaymentServiceCreateSubscriptionOrderRequiresOptInForUSDToCNY(t *testing.T) {
	settings := paymentTestSettings()
	settings.SubscriptionUSDToCNYRate = 0
	svc := newPaymentServiceTestSubject(newPaymentRepoStub(), &airwallexStub{})
	svc.paymentSettingsOverride = func(context.Context) PaymentSettings {
		return settings
	}

	_, err := svc.CreateOrder(context.Background(), CreatePaymentOrderInput{
		UserID: 7, ProductType: PaymentProductSubscription, Currency: "CNY", PlanID: "pro",
	})

	require.ErrorIs(t, err, ErrPaymentUnsupportedCurrency)
}

func TestPaymentServiceCreateOrderSanitizesProviderIntentNUL(t *testing.T) {
	repo := newPaymentRepoStub()
	air := &airwallexStub{intentID: "int_\x00123", clientSecret: "sec\x00ret"}
	svc := newPaymentServiceTestSubject(repo, air)

	result, err := svc.CreateOrder(context.Background(), CreatePaymentOrderInput{
		UserID: 7, ProductType: PaymentProductBalanceTopup, Amount: 12.34, Currency: "USD",
	})

	require.NoError(t, err)
	require.Equal(t, "int_123", result.IntentID)
	require.Equal(t, "secret", result.ClientSecret)
	require.Equal(t, "int_123", result.Order.ProviderIntentID)
	require.NotContains(t, result.Order.ProviderIntentID, "\x00")
}

func TestPaymentSettingsDropUnsafeAmounts(t *testing.T) {
	settings := paymentSettingsFromRaw(map[string]string{
		SettingKeyPaymentAllowedCurrencies:        `["USD","CNY","JPY"]`,
		SettingKeyPaymentDefaultCurrency:          "USD",
		SettingKeyPaymentMinTopupAmount:           "1.005",
		SettingKeyPaymentMaxTopupAmount:           "+Inf",
		SettingKeyPaymentSubscriptionPlans:        `[{"plan_id":"pro","group_id":9,"validity_days":30,"enabled":true,"prices_by_currency":{"USD":0.30000000000000004,"JPY":12.5,"CNY":10000000000}}]`,
		SettingKeyPaymentSubscriptionUSDToCNYRate: "-1",
		SettingKeyPurchaseSubscriptionEnabled:     "true",
	})

	require.Equal(t, 1.01, settings.MinTopupAmount)
	require.Equal(t, 5000.0, settings.MaxTopupAmount)
	require.Len(t, settings.SubscriptionPlans, 1)
	require.Equal(t, 0.3, settings.SubscriptionPlans[0].PricesByCurrency["USD"])
	require.Equal(t, 13.0, settings.SubscriptionPlans[0].PricesByCurrency["JPY"])
	require.Zero(t, settings.SubscriptionUSDToCNYRate)
	_, ok := settings.SubscriptionPlans[0].PricesByCurrency["CNY"]
	require.False(t, ok)
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

func TestPaymentServiceResumeOrderSyncsSucceededIntentAndFulfills(t *testing.T) {
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		ID: 3, OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup,
		Status: PaymentStatusPending, Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123",
		AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	air := &airwallexStub{retrieveStatus: "SUCCEEDED"}
	svc := newPaymentServiceTestSubject(repo, air)

	result, err := svc.ResumeOrderByOrderNo(context.Background(), order.OrderNo, 7)

	require.NoError(t, err)
	require.Equal(t, "secret2", result.ClientSecret)
	require.Equal(t, 1, air.retrieves)
	require.Equal(t, 1, repo.txRuns)
	require.Equal(t, 1, repo.walletAdds)
	require.Equal(t, 15.0, repo.lastWalletAmount)
	require.Equal(t, PaymentStatusPaid, order.Status)
	require.NotNil(t, order.PaidAt)
}

func TestPaymentServiceResumeOrderSyncsTerminalProviderStatus(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider string
		expected string
	}{
		{name: "cancelled", provider: "CANCELLED", expected: PaymentStatusCancelled},
		{name: "expired", provider: "EXPIRED", expected: PaymentStatusExpired},
		{name: "failed", provider: "FAILED", expected: PaymentStatusFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newPaymentRepoStub()
			order := &PaymentOrder{
				OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup,
				Status: PaymentStatusPending, Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123",
				AmountMinor: 1500, Currency: "USD",
			}
			repo.orders[order.OrderNo] = order
			svc := newPaymentServiceTestSubject(repo, &airwallexStub{retrieveStatus: tc.provider})

			_, err := svc.ResumeOrderByOrderNo(context.Background(), order.OrderNo, 7)

			require.NoError(t, err)
			require.Equal(t, tc.expected, order.Status)
			require.Equal(t, []string{tc.expected}, repo.statusUpdates)
			require.Zero(t, repo.walletAdds)
		})
	}
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
	require.Equal(t, 15.0, repo.lastWalletAmount)
	require.Equal(t, PaymentStatusPaid, order.Status)
	var redacted map[string]any
	require.NoError(t, json.Unmarshal(repo.events["evt_1"].PayloadRedactedJSON, &redacted))
	data, ok := redacted["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "[REDACTED]", data["client_secret"])
	require.Equal(t, "[REDACTED]", data["customer_email"])
	require.Equal(t, int64(1), SnapshotPaymentRuntimeMetrics().WebhookSuccess)
}

func TestPaymentServiceWebhookSanitizesProviderPayloadNUL(t *testing.T) {
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		ID: 3, OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup,
		Status: PaymentStatusPending, Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123",
		AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	repo.ordersByIntent[order.ProviderIntentID] = order
	svc := newPaymentServiceTestSubject(repo, &airwallexStub{})

	body := []byte(`{"id":"evt_1\u0000","name":"payment_intent.succeeded\u0000","data":{"id":"int_123\u0000","note":"ok\u0000"}}`)
	require.NoError(t, svc.HandleAirwallexWebhook(context.Background(), "ts", "sig", body))

	event := repo.events["evt_1"]
	require.NotNil(t, event)
	require.Equal(t, "evt_1", event.ProviderEventID)
	require.Equal(t, "payment_intent.succeeded", event.EventType)
	require.NotContains(t, string(event.PayloadRedactedJSON), "\x00")
	require.NotContains(t, string(event.PayloadRedactedJSON), `\u0000`)
}

func TestPaymentServiceWebhookPaidInvalidatesAuthCacheAfterTopup(t *testing.T) {
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		ID: 3, OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup,
		Status: PaymentStatusPending, Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123",
		AmountMinor: 10, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	repo.ordersByIntent[order.ProviderIntentID] = order
	authInvalidator := &paymentAuthCacheInvalidatorStub{}
	svc := newPaymentServiceTestSubject(repo, &airwallexStub{})
	svc.SetBillingCacheInvalidators(nil, authInvalidator)

	body := []byte(`{"id":"evt_1","name":"payment_intent.succeeded","data":{"id":"int_123"}}`)
	require.NoError(t, svc.HandleAirwallexWebhook(context.Background(), "ts", "sig", body))

	require.Equal(t, []int64{7}, authInvalidator.users)
	require.Equal(t, 0.1, repo.lastWalletAmount)
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

func TestPaymentServiceRefundOrderRequiresForceWhenTopupBalanceInsufficient(t *testing.T) {
	repo := newPaymentRepoStub()
	repo.walletBalance = 2
	order := &PaymentOrder{
		OrderNo: "pay_force", UserID: 7, ProductType: PaymentProductBalanceTopup, Status: PaymentStatusPaid,
		Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123", AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	air := &airwallexStub{refundStatus: "succeeded"}
	svc := newPaymentServiceTestSubject(repo, air)

	refund, err := svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 500, Reason: "requested", RequestedBy: 1, IdempotencyKey: "force-check",
	})

	require.Nil(t, refund)
	require.ErrorIs(t, err, ErrPaymentRefundRequiresForce)
	require.Equal(t, 0, air.creates)
	require.Empty(t, repo.refunds)
}

func TestPaymentServiceRefundOrderForceDeductsFullTopupBalance(t *testing.T) {
	repo := newPaymentRepoStub()
	repo.walletBalance = 2
	order := &PaymentOrder{
		OrderNo: "pay_force_confirm", UserID: 7, ProductType: PaymentProductBalanceTopup, Status: PaymentStatusPaid,
		Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123", AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	air := &airwallexStub{refundStatus: "succeeded"}
	svc := newPaymentServiceTestSubject(repo, air)

	refund, err := svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 500, Reason: "requested", RequestedBy: 1, IdempotencyKey: "force-confirm", Force: true,
	})

	require.NoError(t, err)
	require.Equal(t, PaymentRefundStatusSettled, refund.Status)
	require.Equal(t, 1, repo.walletAdds)
	require.Equal(t, -5.0, repo.lastWalletAmount)
	require.Equal(t, PaymentStatusPartialRefunded, order.Status)
}

func TestPaymentServiceRefundOrderSanitizesProviderRefundNUL(t *testing.T) {
	repo := newPaymentRepoStub()
	order := &PaymentOrder{
		OrderNo: "pay_1", UserID: 7, ProductType: PaymentProductBalanceTopup, Status: PaymentStatusPaid,
		Provider: PaymentProviderAirwallex, ProviderIntentID: "int_123", AmountMinor: 1500, Currency: "USD",
	}
	repo.orders[order.OrderNo] = order
	svc := newPaymentServiceTestSubject(repo, &airwallexStub{refundID: "rf_\x00provider", refundStatus: "succeeded\x00"})

	refund, err := svc.RefundOrder(context.Background(), RefundPaymentOrderInput{
		OrderNo: order.OrderNo, AmountMinor: 500, Reason: "requested", RequestedBy: 1, IdempotencyKey: "sanitize-refund",
	})

	require.NoError(t, err)
	require.Equal(t, "rf_provider", refund.ProviderRefundID)
	require.Equal(t, PaymentRefundStatusSettled, refund.Status)
	require.NotContains(t, refund.ProviderRefundID, "\x00")
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
