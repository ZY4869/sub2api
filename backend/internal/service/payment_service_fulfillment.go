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
		locked, err := s.repo.GetOrderByOrderNoForUpdate(txCtx, order.OrderNo)
		if err != nil {
			return err
		}
		if locked.Status == PaymentStatusPaid {
			order.Status = PaymentStatusPaid
			order.PaidAt = locked.PaidAt
			order.UpdatedAt = locked.UpdatedAt
			return nil
		}
		if locked.ProductType == PaymentProductBalanceTopup {
			amount, err := NormalizeAndValidatePositiveBillingAmount(PaymentMinorToAmount(locked.AmountMinor, locked.Currency))
			if err != nil {
				return err
			}
			if err := s.repo.AddWalletBalance(txCtx, locked.UserID, locked.Currency, amount); err != nil {
				return err
			}
			if s.affiliateService != nil {
				s.affiliateService.AccruePaymentTopupRebateBestEffort(txCtx, locked.ID, locked.UserID, amount)
			}
		}
		if locked.ProductType == PaymentProductSubscription {
			var snap paymentOrderSnapshot
			_ = json.Unmarshal(locked.SnapshotJSON, &snap)
			if snap.GroupID <= 0 || snap.ValidityDays <= 0 {
				return ErrPaymentInvalidProduct
			}
			if s.subscriptionSvc != nil {
				s.subscriptionSvc.InvalidateSubCache(locked.UserID, snap.GroupID)
			}
			if err := s.repo.AssignOrExtendSubscription(txCtx, &AssignSubscriptionInput{
				UserID:       locked.UserID,
				GroupID:      snap.GroupID,
				ValidityDays: snap.ValidityDays,
				Notes:        "payment order " + locked.OrderNo,
			}); err != nil {
				return err
			}
		}
		if locked.ProductType != PaymentProductBalanceTopup && locked.ProductType != PaymentProductSubscription {
			return ErrPaymentInvalidProduct
		}
		if err := s.repo.UpdateOrderStatus(txCtx, locked.OrderNo, PaymentStatusPaid, &now, nil); err != nil {
			return err
		}
		order.Status = PaymentStatusPaid
		order.PaidAt = &now
		order.UpdatedAt = now
		s.logInfo(ctx, "payment.order.fulfilled",
			zap.String("order_no", locked.OrderNo),
			zap.Int64("user_id", locked.UserID),
			zap.String("provider", locked.Provider),
			zap.String("status", PaymentStatusPaid),
		)
		return nil
	})
	if err == nil {
		if order.ProductType == PaymentProductBalanceTopup {
			s.invalidateBalanceCaches(ctx, order.UserID)
		}
		s.sendPaymentSuccessEmailBestEffort(ctx, order)
	}
	return err
}

func (s *PaymentService) invalidateBalanceCaches(ctx context.Context, userID int64) {
	if s == nil || userID <= 0 {
		return
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService == nil {
		return
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.billingCacheService.InvalidateUserBalance(cacheCtx, userID); err != nil {
		s.logWarn(ctx, "payment.balance_cache_invalidate_failed", zap.Int64("user_id", userID), zap.Error(err))
	}
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
