package service

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func (s *APIKeyService) TryReserveRequestBillingHold(ctx context.Context, apiKey *APIKey, cfg *config.Config) (*BillingHold, error) {
	if s == nil || apiKey == nil || apiKey.User == nil {
		return nil, ErrInsufficientBalance
	}
	if apiKey.BillingHold != nil && apiKey.BillingHold.Status == BillingHoldStatusHeld {
		return apiKey.BillingHold, nil
	}
	repo := s.billingHoldRepository()
	amount := MinimumRequestHoldUSD(cfg)
	if repo == nil || amount <= 0 {
		return nil, ErrBillingServiceUnavailable
	}
	requestID := resolveUsageBillingRequestID(ctx, "")
	hold, err := repo.Reserve(ctx, &BillingHold{
		RequestID:          requestID,
		RequestFingerprint: BillingHoldRequestFingerprintFromContext(ctx),
		APIKeyID:           apiKey.ID,
		UserID:             apiKey.User.ID,
		Currency:           ModelPricingCurrencyUSD,
		Amount:             amount,
		CurrencyConversion: billingCurrencyConversionFromSettings(ctx, s.settingService),
	})
	if err != nil {
		return nil, err
	}
	if hold != nil && hold.Status == BillingHoldStatusHeld {
		apiKey.BillingHold = hold
		DeductBillingHoldFromUserSnapshot(apiKey.User, hold)
		s.InvalidateAuthCacheByKey(ctx, apiKey.Key)
		s.invalidateBillingBalanceCache(ctx, apiKey.User.ID)
	}
	return hold, nil
}

func (s *APIKeyService) ReleaseRequestBillingHold(ctx context.Context, apiKey *APIKey) {
	if s == nil || apiKey == nil || apiKey.BillingHold == nil {
		return
	}
	hold := apiKey.BillingHold
	if hold.Status != BillingHoldStatusHeld {
		return
	}
	repo := s.billingHoldRepository()
	if repo == nil {
		return
	}
	released, err := repo.Release(ctx, hold.RequestID, hold.APIKeyID)
	if err != nil && !errors.Is(err, ErrBillingHoldNotFound) {
		return
	}
	if released != nil {
		apiKey.BillingHold = released
	} else {
		hold.Status = BillingHoldStatusReleased
	}
	s.InvalidateAuthCacheByKey(ctx, apiKey.Key)
	s.invalidateBillingBalanceCache(ctx, hold.UserID)
}

func (s *APIKeyService) billingHoldRepository() BillingHoldRepository {
	if s == nil || s.apiKeyRepo == nil {
		return nil
	}
	provider, ok := s.apiKeyRepo.(billingHoldRepositoryProvider)
	if !ok {
		return nil
	}
	return provider.BillingHoldRepository()
}

func (s *APIKeyService) invalidateBillingBalanceCache(ctx context.Context, userID int64) {
	if s == nil || s.billingCacheService == nil || userID <= 0 {
		return
	}
	_ = s.billingCacheService.InvalidateUserBalance(ctx, userID)
}
