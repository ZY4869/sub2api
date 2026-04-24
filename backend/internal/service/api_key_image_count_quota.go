package service

import (
	"context"
	"fmt"
)

// TryReserveImageCount atomically reserves `count` image outputs for the API key.
//
// It returns:
//   - (true, nil) when the reservation succeeds (or count<=0 is a no-op)
//   - (false, nil) when the reservation would exceed image_max_count
//   - (false, err) on unexpected errors (including ErrAPIKeyNotFound)
func (s *APIKeyService) TryReserveImageCount(ctx context.Context, keyID int64, count int) (bool, error) {
	if s == nil || s.apiKeyRepo == nil || keyID <= 0 || count <= 0 {
		return true, nil
	}
	ok, err := s.apiKeyRepo.TryReserveImageCount(ctx, keyID, count)
	if err != nil {
		return false, fmt.Errorf("try reserve image count: %w", err)
	}
	return ok, nil
}

// RollbackImageCount rolls back (subtracts) `count` image outputs from the API key's used counter.
// This is best-effort and must never underflow below zero.
func (s *APIKeyService) RollbackImageCount(ctx context.Context, keyID int64, count int) error {
	if s == nil || s.apiKeyRepo == nil || keyID <= 0 || count <= 0 {
		return nil
	}
	if err := s.apiKeyRepo.RollbackImageCount(ctx, keyID, count); err != nil {
		return fmt.Errorf("rollback image count: %w", err)
	}
	return nil
}
