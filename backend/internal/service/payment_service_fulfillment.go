package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

func (s *PaymentService) markOrderPaid(ctx context.Context, order *PaymentOrder) error {
	if order.Status == PaymentStatusPaid {
		return nil
	}
	now := time.Now()
	err := s.repo.RunInTx(ctx, func(txCtx context.Context) error {
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
	if err == nil {
		s.sendPaymentSuccessEmailBestEffort(ctx, order)
	}
	return err
}

func (s *PaymentService) sendPaymentSuccessEmailBestEffort(ctx context.Context, order *PaymentOrder) {
	if s == nil || s.emailService == nil || s.emailTemplates == nil || s.userRepo == nil || order == nil {
		return
	}
	if !s.emailTemplates.ShouldSendNotification(ctx, order.UserID, NotificationCategoryPaymentSuccess, order.OrderNo, order.ProductType, time.Now()) {
		return
	}
	user, err := s.userRepo.GetByID(ctx, order.UserID)
	if err != nil || user == nil || user.Email == "" {
		return
	}
	data := map[string]string{
		"SiteName":    paymentNotificationSiteName(ctx, s.settings),
		"OrderNo":     order.OrderNo,
		"ProductType": order.ProductType,
		"Amount":      fmt.Sprintf("%.2f", PaymentMinorToAmount(order.AmountMinor, order.Currency)),
		"Currency":    order.Currency,
	}
	if err := s.emailService.SendTemplatedEmail(ctx, user.Email, EmailTemplatePaymentSuccess, "zh", data); err != nil {
		s.logWarn(ctx, "payment.notification.email_failed", zap.String("order_no", order.OrderNo), zap.Int64("user_id", order.UserID), zap.Error(err))
	}
}

func paymentNotificationSiteName(ctx context.Context, settings *SettingService) string {
	if settings == nil {
		return "Sub2API"
	}
	return settings.GetSiteName(ctx)
}
