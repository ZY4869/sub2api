package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *PaymentService) RefundOrder(ctx context.Context, input RefundPaymentOrderInput) (*PaymentRefund, error) {
	success := false
	defer func() {
		if success {
			paymentRuntimeMetrics.refundSuccess.Add(1)
		} else {
			paymentRuntimeMetrics.refundFailure.Add(1)
		}
	}()
	if s == nil || s.repo == nil || s.airwallex == nil {
		return nil, ErrPaymentServiceUnavailable
	}
	settings := s.paymentSettings(ctx)
	order, err := s.repo.GetOrderByOrderNo(ctx, input.OrderNo)
	if err != nil {
		return nil, err
	}
	if order.Status != PaymentStatusPaid && order.Status != PaymentStatusPartialRefunded {
		return nil, ErrPaymentOrderNotRefundable
	}
	idempotencyHash := hashPaymentToken(input.IdempotencyKey)
	if idempotencyHash != "" {
		if existing, err := s.repo.GetRefundByOrderIdempotencyHash(ctx, order.OrderNo, idempotencyHash); err == nil && existing != nil {
			success = true
			return existing, nil
		}
	}
	amount := input.AmountMinor
	refundedAmount, err := s.repo.SumSuccessfulRefundAmount(ctx, order.OrderNo)
	if err != nil {
		return nil, err
	}
	remaining := order.AmountMinor - refundedAmount
	if remaining <= 0 {
		return nil, ErrPaymentOrderNotRefundable
	}
	if amount <= 0 {
		amount = remaining
	}
	if amount > remaining {
		return nil, ErrPaymentInvalidAmount.WithMetadata(map[string]string{"max_amount_minor": fmt.Sprintf("%d", remaining)})
	}
	refund := &PaymentRefund{
		RefundNo:           "rf_" + randomPaymentHex(12),
		OrderNo:            order.OrderNo,
		AmountMinor:        amount,
		Currency:           order.Currency,
		Reason:             strings.TrimSpace(input.Reason),
		Status:             PaymentRefundStatusReceived,
		RequestedBy:        &input.RequestedBy,
		IdempotencyKeyHash: idempotencyHash,
	}
	if err := s.repo.CreateRefund(ctx, refund); err != nil {
		return nil, err
	}
	s.logInfo(ctx, "payment.refund.create.started",
		zap.String("order_no", order.OrderNo),
		zap.String("refund_no", refund.RefundNo),
		zap.String("provider", order.Provider),
		zap.String("status", refund.Status),
	)
	providerStarted := time.Now()
	providerRefund, err := s.airwallex.CreateRefund(ctx, settings, AirwallexRefundRequest{
		RequestID:   refund.RefundNo,
		IntentID:    order.ProviderIntentID,
		AmountMinor: refund.AmountMinor,
		Currency:    refund.Currency,
		Reason:      refund.Reason,
	})
	recordPaymentProviderLatency(time.Since(providerStarted).Milliseconds())
	if err != nil {
		_ = s.repo.UpdateRefundProvider(ctx, refund.RefundNo, "", PaymentRefundStatusFailed)
		s.logWarn(ctx, "payment.refund.provider_failed",
			zap.String("order_no", order.OrderNo),
			zap.String("refund_no", refund.RefundNo),
			zap.String("provider", order.Provider),
			zap.String("status", PaymentRefundStatusFailed),
			zap.Error(err),
		)
		return nil, err
	}
	refund.ProviderRefundID = providerRefund.ID
	refund.Status = normalizeRefundStatus(providerRefund.Status)
	if err := s.repo.UpdateRefundProvider(ctx, refund.RefundNo, refund.ProviderRefundID, refund.Status); err != nil {
		return nil, err
	}
	if refund.Status == PaymentRefundStatusSettled || refund.Status == PaymentRefundStatusAccepted {
		now := time.Now()
		refundedAmount, err := s.repo.SumSuccessfulRefundAmount(ctx, order.OrderNo)
		if err != nil {
			return nil, err
		}
		nextStatus := PaymentStatusPartialRefunded
		if refundedAmount >= order.AmountMinor {
			nextStatus = PaymentStatusRefunded
		}
		if err := s.repo.UpdateOrderStatus(ctx, order.OrderNo, nextStatus, nil, &now); err != nil {
			return nil, err
		}
		order.Status = nextStatus
		order.RefundedAt = &now
	}
	success = true
	s.logInfo(ctx, "payment.refund.create.succeeded",
		zap.String("order_no", order.OrderNo),
		zap.String("refund_no", refund.RefundNo),
		zap.String("provider", order.Provider),
		zap.String("status", refund.Status),
	)
	return refund, nil
}

func normalizeRefundStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "success", "settled":
		return PaymentRefundStatusSettled
	case "failed":
		return PaymentRefundStatusFailed
	default:
		return PaymentRefundStatusAccepted
	}
}
