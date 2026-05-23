package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"go.uber.org/zap"
)

func (s *PaymentService) GetOrderForUser(ctx context.Context, userID int64, orderNo string) (*PaymentOrder, error) {
	if s == nil || s.repo == nil {
		return nil, ErrPaymentServiceUnavailable
	}
	order, err := s.repo.GetOrderByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ErrPaymentOrderForbidden
	}
	return order, nil
}

func (s *PaymentService) ResumeOrder(ctx context.Context, token string, userID int64) (*ResumePaymentOrderResult, error) {
	success := false
	defer func() {
		if success {
			paymentRuntimeMetrics.resumeSuccess.Add(1)
		} else {
			paymentRuntimeMetrics.resumeFailure.Add(1)
		}
	}()
	if s == nil || s.repo == nil {
		return nil, ErrPaymentServiceUnavailable
	}
	order, err := s.repo.GetOrderByResumeTokenHash(ctx, hashPaymentToken(token))
	if err != nil {
		return nil, err
	}
	result, err := s.resumeExistingOrder(ctx, order, userID)
	if err != nil {
		return nil, err
	}
	success = true
	return result, nil
}

func (s *PaymentService) ResumeOrderByOrderNo(ctx context.Context, orderNo string, userID int64) (*ResumePaymentOrderResult, error) {
	success := false
	defer func() {
		if success {
			paymentRuntimeMetrics.resumeSuccess.Add(1)
		} else {
			paymentRuntimeMetrics.resumeFailure.Add(1)
		}
	}()
	if s == nil || s.repo == nil {
		return nil, ErrPaymentServiceUnavailable
	}
	order, err := s.repo.GetOrderByOrderNo(ctx, orderNo)
	if err != nil {
		return nil, err
	}
	result, err := s.resumeExistingOrder(ctx, order, userID)
	if err != nil {
		return nil, err
	}
	success = true
	return result, nil
}

func (s *PaymentService) resumeExistingOrder(ctx context.Context, order *PaymentOrder, userID int64) (*ResumePaymentOrderResult, error) {
	if order == nil {
		return nil, ErrPaymentOrderNotFound
	}
	if order.UserID != userID {
		return nil, ErrPaymentOrderForbidden
	}
	settings := s.paymentSettings(ctx)
	result := &ResumePaymentOrderResult{Order: order, ClientID: settings.AirwallexClientID, IntentID: order.ProviderIntentID, ProviderEnv: order.ProviderEnv, PaymentMode: resolvePaymentMode(settings)}
	if order.Status != PaymentStatusCreated && order.Status != PaymentStatusPending {
		return result, nil
	}
	if order.ProviderIntentID == "" || s.airwallex == nil {
		return result, nil
	}
	providerStarted := time.Now()
	intent, err := s.airwallex.RetrievePaymentIntent(ctx, settings, order.ProviderIntentID)
	recordPaymentProviderLatency(time.Since(providerStarted).Milliseconds())
	if err != nil {
		s.logWarn(ctx, "payment.order.resume.provider_failed",
			zap.String("order_no", order.OrderNo),
			zap.String("provider", order.Provider),
			zap.String("status", order.Status),
			zap.Error(err),
		)
		return nil, err
	}
	if intent != nil {
		result.ClientSecret = intent.ClientSecret
	}
	return result, nil
}

func (s *PaymentService) CancelOrder(ctx context.Context, userID int64, orderNo string) error {
	order, err := s.GetOrderForUser(ctx, userID, orderNo)
	if err != nil {
		return err
	}
	if order.Status != PaymentStatusCreated && order.Status != PaymentStatusPending {
		return ErrPaymentOrderNotCancelable
	}
	return s.repo.CancelOrder(ctx, orderNo)
}

func (s *PaymentService) ListOrders(ctx context.Context, params pagination.PaginationParams, status, provider, productType string, userID *int64) ([]PaymentOrder, *pagination.PaginationResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, ErrPaymentServiceUnavailable
	}
	return s.repo.ListOrders(ctx, params, status, provider, productType, userID)
}

func (s *PaymentService) rebuildCreateOrderResult(ctx context.Context, settings PaymentSettings, order *PaymentOrder, resumeToken string) (*CreatePaymentOrderResult, error) {
	result := &CreatePaymentOrderResult{Order: order, ClientID: settings.AirwallexClientID, IntentID: order.ProviderIntentID, ResumeToken: resumeToken, ProviderEnv: order.ProviderEnv, PaymentMode: resolvePaymentMode(settings)}
	if order.ProviderIntentID == "" || s.airwallex == nil {
		return result, nil
	}
	intent, err := s.airwallex.RetrievePaymentIntent(ctx, settings, order.ProviderIntentID)
	if err != nil {
		return result, nil
	}
	result.ClientSecret = intent.ClientSecret
	return result, nil
}
