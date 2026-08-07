package service

import (
	"context"
	"fmt"
	"strconv"
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
	if err := s.requireRefundBalanceConfirmation(ctx, order, amount, input.Force); err != nil {
		return nil, err
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
	providerRefund.ID = sanitizePaymentProviderText(providerRefund.ID)
	providerRefund.Status = sanitizePaymentProviderText(providerRefund.Status)
	refund.ProviderRefundID = providerRefund.ID
	refund.Status = normalizeRefundStatus(providerRefund.Status)
	if err := s.repo.UpdateRefundProvider(ctx, refund.RefundNo, refund.ProviderRefundID, refund.Status); err != nil {
		return nil, err
	}
	if refund.Status == PaymentRefundStatusSettled {
		now := time.Now()
		var nextStatus string
		if err := s.repo.RunInTx(ctx, func(txCtx context.Context) error {
			if err := s.deductRefundedWalletBalance(txCtx, order, refund); err != nil {
				return err
			}
			refundedAmount, err := s.repo.SumSuccessfulRefundAmount(txCtx, order.OrderNo)
			if err != nil {
				return err
			}
			nextStatus = PaymentStatusPartialRefunded
			if refundedAmount >= order.AmountMinor {
				nextStatus = PaymentStatusRefunded
			}
			return s.repo.UpdateOrderStatus(txCtx, order.OrderNo, nextStatus, nil, &now)
		}); err != nil {
			return nil, err
		}
		order.Status = nextStatus
		order.RefundedAt = &now
		if order.ProductType == PaymentProductBalanceTopup {
			s.invalidateBalanceCaches(ctx, order.UserID)
		}
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

func (s *PaymentService) requireRefundBalanceConfirmation(ctx context.Context, order *PaymentOrder, amountMinor int64, force bool) error {
	if s == nil || s.repo == nil || order == nil || order.ProductType != PaymentProductBalanceTopup || force {
		return nil
	}
	refundAmount, err := NormalizeAndValidatePositiveBillingAmount(PaymentMinorToAmount(amountMinor, order.Currency))
	if err != nil {
		return err
	}
	current, err := s.repo.GetWalletBalance(ctx, order.UserID, order.Currency)
	if err != nil {
		return err
	}
	currentMoney, err := NewBillingMoneyFromFloat(current)
	if err != nil {
		return err
	}
	refundMoney, err := NewPositiveBillingMoneyFromFloat(refundAmount)
	if err != nil {
		return err
	}
	if currentMoney.Cmp(refundMoney) >= 0 {
		return nil
	}
	return ErrPaymentRefundRequiresForce.WithMetadata(map[string]string{
		"require_force":       "true",
		"order_no":            order.OrderNo,
		"currency":            NormalizePaymentCurrency(order.Currency),
		"current_balance":     strconv.FormatFloat(currentMoney.Float64(), 'f', 8, 64),
		"refund_amount":       strconv.FormatFloat(refundMoney.Float64(), 'f', 8, 64),
		"refund_amount_minor": strconv.FormatInt(amountMinor, 10),
	})
}

func (s *PaymentService) deductRefundedWalletBalance(ctx context.Context, order *PaymentOrder, refund *PaymentRefund) error {
	if s == nil || s.repo == nil || order == nil || refund == nil || order.ProductType != PaymentProductBalanceTopup {
		return nil
	}
	amount, err := NormalizeAndValidatePositiveBillingAmount(PaymentMinorToAmount(refund.AmountMinor, refund.Currency))
	if err != nil {
		return err
	}
	return s.repo.AddWalletBalance(ctx, order.UserID, refund.Currency, -amount)
}

func normalizeRefundStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "received":
		return PaymentRefundStatusReceived
	case "accepted", "processing", "requires_action":
		return PaymentRefundStatusAccepted
	case "succeeded", "success", "settled":
		return PaymentRefundStatusSettled
	case "failed":
		return PaymentRefundStatusFailed
	default:
		return PaymentRefundStatusAccepted
	}
}
