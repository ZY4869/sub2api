package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *PaymentService) CreateOrder(ctx context.Context, input CreatePaymentOrderInput) (*CreatePaymentOrderResult, error) {
	if s == nil || s.repo == nil || s.airwallex == nil {
		paymentRuntimeMetrics.createFailure.Add(1)
		return nil, ErrPaymentProviderNotConfigured
	}
	success := false
	defer func() {
		if success {
			paymentRuntimeMetrics.createSuccess.Add(1)
		} else {
			paymentRuntimeMetrics.createFailure.Add(1)
		}
	}()
	settings := s.paymentSettings(ctx)
	if !settings.Enabled || !settings.AirwallexEnabled {
		return nil, ErrPaymentDisabled
	}
	if !settings.AirwallexAPIKeyConfigured || strings.TrimSpace(settings.AirwallexClientID) == "" {
		return nil, ErrPaymentProviderNotConfigured
	}
	currency, err := validatePaymentCurrency(input.Currency, settings.AllowedCurrencies)
	if err != nil {
		return nil, err
	}
	idempotencyHash := hashPaymentToken(input.IdempotencyKey)
	if idempotencyHash != "" {
		if existing, err := s.repo.GetOrderByUserIdempotencyHash(ctx, input.UserID, idempotencyHash); err == nil && existing != nil {
			result, rebuildErr := s.rebuildCreateOrderResult(ctx, settings, existing, "")
			if rebuildErr == nil {
				success = true
			}
			return result, rebuildErr
		}
	}
	orderNo := paymentOrderNo(randomPaymentHex(12))
	resumeToken := randomPaymentHex(24)
	expiresAt := time.Now().Add(30 * time.Minute)
	snapshot, amountMinor, err := s.buildOrderSnapshot(settings, input, currency)
	if err != nil {
		return nil, err
	}
	order := &PaymentOrder{
		OrderNo:            orderNo,
		UserID:             input.UserID,
		ProductType:        input.ProductType,
		Status:             PaymentStatusCreated,
		Provider:           PaymentProviderAirwallex,
		ProviderEnv:        settings.AirwallexEnv,
		AmountMinor:        amountMinor,
		Currency:           currency,
		CountryCode:        strings.ToUpper(strings.TrimSpace(input.CountryCode)),
		ResumeTokenHash:    hashPaymentToken(resumeToken),
		IdempotencyKeyHash: idempotencyHash,
		SnapshotJSON:       snapshot,
		ExpiresAt:          &expiresAt,
	}
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	returnURL := resolvePaymentReturnURL(input.ReturnURL, settings.FrontendURL, order.OrderNo)
	s.logInfo(ctx, "payment.order.create.started",
		zap.String("order_no", order.OrderNo),
		zap.Int64("user_id", order.UserID),
		zap.String("provider", order.Provider),
		zap.String("product_type", order.ProductType),
		zap.String("status", order.Status),
	)
	providerStarted := time.Now()
	intent, err := s.airwallex.CreatePaymentIntent(ctx, settings, AirwallexPaymentIntentRequest{
		RequestID:   order.OrderNo,
		AmountMinor: order.AmountMinor,
		Currency:    order.Currency,
		OrderNo:     order.OrderNo,
		Descriptor:  "Sub2API",
		ReturnURL:   returnURL,
		Metadata: map[string]string{
			"order_no":     order.OrderNo,
			"product_type": order.ProductType,
			"user_id":      fmt.Sprintf("%d", order.UserID),
		},
	})
	recordPaymentProviderLatency(time.Since(providerStarted).Milliseconds())
	if err != nil {
		_ = s.repo.UpdateOrderStatus(ctx, order.OrderNo, PaymentStatusFailed, nil, nil)
		s.logWarn(ctx, "payment.provider.intent_failed",
			zap.String("order_no", order.OrderNo),
			zap.String("provider", order.Provider),
			zap.String("status", PaymentStatusFailed),
			zap.Error(err),
		)
		return nil, err
	}
	if err := s.repo.UpdateOrderProviderIntent(ctx, order.OrderNo, intent.ID, PaymentStatusPending); err != nil {
		return nil, err
	}
	order.ProviderIntentID = intent.ID
	order.Status = PaymentStatusPending
	success = true
	s.logInfo(ctx, "payment.order.create.succeeded",
		zap.String("order_no", order.OrderNo),
		zap.String("provider", order.Provider),
		zap.String("status", order.Status),
	)
	return &CreatePaymentOrderResult{
		Order:        order,
		ClientSecret: intent.ClientSecret,
		ClientID:     settings.AirwallexClientID,
		IntentID:     intent.ID,
		ResumeToken:  resumeToken,
		ProviderEnv:  settings.AirwallexEnv,
	}, nil
}
