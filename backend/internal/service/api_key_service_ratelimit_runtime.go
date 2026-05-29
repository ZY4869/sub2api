package service

import (
	"context"
)

func (s *APIKeyService) GetRateLimitData(ctx context.Context, id int64) (*APIKeyRateLimitData, error) {
	return s.apiKeyRepo.GetRateLimitData(ctx, id)
}

// UpdateRateLimitUsage atomically increments rate limit usage counters in the DB.

func (s *APIKeyService) UpdateRateLimitUsage(ctx context.Context, apiKeyID int64, cost float64) error {
	normalized, err := NormalizeAndValidateNonNegativeBillingAmount(cost)
	if err != nil {
		return err
	}
	cost = normalized
	if cost <= 0 {
		return nil
	}
	return s.apiKeyRepo.IncrementRateLimitUsage(ctx, apiKeyID, cost)
}
