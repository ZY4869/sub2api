package service

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
)

func (s *PaymentService) markOrderPaid(ctx context.Context, order *PaymentOrder) error {
	if order.Status == PaymentStatusPaid {
		return nil
	}
	now := time.Now()
	return s.repo.RunInTx(ctx, func(txCtx context.Context) error {
		if order.ProductType == PaymentProductBalanceTopup {
			amount := PaymentMinorToAmount(order.AmountMinor, order.Currency)
			if err := s.repo.AddWalletBalance(txCtx, order.UserID, order.Currency, amount); err != nil {
				return err
			}
			if s.affiliateService != nil {
				s.affiliateService.AccruePaymentTopupRebateBestEffort(txCtx, order.ID, order.UserID, amount)
			}
		}
		if order.ProductType == PaymentProductSubscription {
			var snap paymentOrderSnapshot
			_ = json.Unmarshal(order.SnapshotJSON, &snap)
			if snap.GroupID <= 0 || snap.ValidityDays <= 0 {
				return ErrPaymentInvalidProduct
			}
			if s.subscriptionSvc != nil {
				s.subscriptionSvc.InvalidateSubCache(order.UserID, snap.GroupID)
			}
			if err := s.repo.AssignOrExtendSubscription(txCtx, &AssignSubscriptionInput{
				UserID:       order.UserID,
				GroupID:      snap.GroupID,
				ValidityDays: snap.ValidityDays,
				Notes:        "payment order " + order.OrderNo,
			}); err != nil {
				return err
			}
		}
		if order.ProductType != PaymentProductBalanceTopup && order.ProductType != PaymentProductSubscription {
			return ErrPaymentInvalidProduct
		}
		if err := s.repo.UpdateOrderStatus(txCtx, order.OrderNo, PaymentStatusPaid, &now, nil); err != nil {
			return err
		}
		s.logInfo(ctx, "payment.order.fulfilled",
			zap.String("order_no", order.OrderNo),
			zap.Int64("user_id", order.UserID),
			zap.String("provider", order.Provider),
			zap.String("status", PaymentStatusPaid),
		)
		return nil
	})
}
