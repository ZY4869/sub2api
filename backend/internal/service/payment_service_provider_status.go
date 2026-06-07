package service

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *PaymentService) syncRetrievedPaymentIntentStatus(ctx context.Context, order *PaymentOrder, intent *AirwallexPaymentIntentResponse) error {
	if s == nil || s.repo == nil || order == nil || intent == nil {
		return nil
	}
	if order.Provider != PaymentProviderAirwallex {
		return nil
	}
	if order.Status != PaymentStatusCreated && order.Status != PaymentStatusPending {
		return nil
	}
	nextStatus := paymentStatusFromAirwallexIntentStatus(intent.Status)
	switch nextStatus {
	case PaymentStatusPaid:
		if err := s.markOrderPaid(ctx, order); err != nil {
			s.logWarn(ctx, "payment.order.provider_status_fulfill_failed",
				zap.String("order_no", order.OrderNo),
				zap.String("provider", order.Provider),
				zap.String("provider_status", normalizeAirwallexIntentStatus(intent.Status)),
				zap.Error(err),
			)
			return err
		}
	case PaymentStatusFailed, PaymentStatusCancelled, PaymentStatusExpired:
		if err := s.repo.UpdateOrderStatus(ctx, order.OrderNo, nextStatus, nil, nil); err != nil {
			return err
		}
		order.Status = nextStatus
		order.UpdatedAt = time.Now()
		s.logInfo(ctx, "payment.order.provider_status_synced",
			zap.String("order_no", order.OrderNo),
			zap.String("provider", order.Provider),
			zap.String("provider_status", normalizeAirwallexIntentStatus(intent.Status)),
			zap.String("status", nextStatus),
		)
	}
	return nil
}

func paymentStatusFromAirwallexIntentStatus(status string) string {
	status = normalizeAirwallexIntentStatus(status)
	switch status {
	case "SUCCEEDED", "SUCCESS", "PAID":
		return PaymentStatusPaid
	case "CANCELLED", "CANCELED":
		return PaymentStatusCancelled
	case "EXPIRED":
		return PaymentStatusExpired
	}
	if strings.Contains(status, "SUCCEED") || strings.Contains(status, "SUCCESS") {
		return PaymentStatusPaid
	}
	if strings.Contains(status, "CANCEL") {
		return PaymentStatusCancelled
	}
	if strings.Contains(status, "EXPIRE") {
		return PaymentStatusExpired
	}
	if strings.Contains(status, "FAIL") || strings.Contains(status, "DECLIN") || strings.Contains(status, "REJECT") {
		return PaymentStatusFailed
	}
	return ""
}

func normalizeAirwallexIntentStatus(status string) string {
	status = strings.ToUpper(strings.TrimSpace(status))
	status = strings.ReplaceAll(status, "-", "_")
	status = strings.ReplaceAll(status, " ", "_")
	return status
}
