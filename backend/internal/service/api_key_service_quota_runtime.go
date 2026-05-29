package service

import (
	"context"
	"fmt"
	"strings"
)

func (s *APIKeyService) CheckAPIKeyQuotaAndExpiry(apiKey *APIKey) error {
	// Check expiration
	if apiKey.IsExpired() {
		return ErrAPIKeyExpired
	}

	// Check quota
	if apiKey.IsQuotaExhausted() {
		return ErrAPIKeyQuotaExhausted
	}

	return nil
}

// UpdateQuotaUsed updates the quota_used field after a request
// Also checks if quota is exhausted and updates status accordingly

func (s *APIKeyService) UpdateQuotaUsed(ctx context.Context, apiKeyID int64, cost float64) error {
	normalized, err := NormalizeAndValidateNonNegativeBillingAmount(cost)
	if err != nil {
		return err
	}
	cost = normalized
	if cost <= 0 {
		return nil
	}

	type quotaStateReader interface {
		IncrementQuotaUsedAndGetState(ctx context.Context, id int64, amount float64) (*APIKeyQuotaUsageState, error)
	}

	if repo, ok := s.apiKeyRepo.(quotaStateReader); ok {
		state, err := repo.IncrementQuotaUsedAndGetState(ctx, apiKeyID, cost)
		if err != nil {
			return fmt.Errorf("increment quota used: %w", err)
		}
		if state != nil && state.Status == StatusAPIKeyQuotaExhausted && strings.TrimSpace(state.Key) != "" {
			s.InvalidateAuthCacheByKey(ctx, state.Key)
		}
		return nil
	}

	// Use repository to atomically increment quota_used
	newQuotaUsed, err := s.apiKeyRepo.IncrementQuotaUsed(ctx, apiKeyID, cost)
	if err != nil {
		return fmt.Errorf("increment quota used: %w", err)
	}

	// Check if quota is now exhausted and update status if needed
	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return nil // Don't fail the request, just log
	}

	// If quota is set and now exhausted, update status
	if apiKey.Quota > 0 && newQuotaUsed >= apiKey.Quota {
		apiKey.Status = StatusAPIKeyQuotaExhausted
		if err := s.apiKeyRepo.Update(ctx, apiKey); err != nil {
			return nil // Don't fail the request
		}
		// Invalidate cache so next request sees the new status
		s.InvalidateAuthCacheByKey(ctx, apiKey.Key)
	}

	return nil
}

func (s *APIKeyService) UpdateGroupQuotaUsed(ctx context.Context, apiKeyID, groupID int64, cost float64) error {
	normalized, err := NormalizeAndValidateNonNegativeBillingAmount(cost)
	if err != nil {
		return err
	}
	cost = normalized
	if cost <= 0 || groupID <= 0 {
		return nil
	}

	type groupQuotaWriter interface {
		IncrementAPIKeyGroupQuotaUsed(ctx context.Context, keyID, groupID int64, amount float64) error
	}

	writer, ok := s.apiKeyRepo.(groupQuotaWriter)
	if !ok {
		return nil
	}
	if err := writer.IncrementAPIKeyGroupQuotaUsed(ctx, apiKeyID, groupID, cost); err != nil {
		return fmt.Errorf("increment api key group quota used: %w", err)
	}
	if apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID); err == nil && apiKey != nil && strings.TrimSpace(apiKey.Key) != "" {
		s.InvalidateAuthCacheByKey(ctx, apiKey.Key)
	}
	return nil
}

// GetRateLimitData returns rate limit usage and window state for an API key.
