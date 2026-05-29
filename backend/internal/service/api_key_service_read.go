package service

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

func (s *APIKeyService) List(ctx context.Context, userID int64, params pagination.PaginationParams, filters APIKeyListFilters) ([]APIKey, *pagination.PaginationResult, error) {
	keys, pagination, err := s.apiKeyRepo.ListByUserID(ctx, userID, params, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("list api keys: %w", err)
	}
	return keys, pagination, nil
}

func (s *APIKeyService) VerifyOwnership(ctx context.Context, userID int64, apiKeyIDs []int64) ([]int64, error) {
	if len(apiKeyIDs) == 0 {
		return []int64{}, nil
	}

	validIDs, err := s.apiKeyRepo.VerifyOwnership(ctx, userID, apiKeyIDs)
	if err != nil {
		return nil, fmt.Errorf("verify api key ownership: %w", err)
	}
	return validIDs, nil
}

// GetByID 根据ID获取API Key

func (s *APIKeyService) GetByID(ctx context.Context, id int64) (*APIKey, error) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	s.compileAPIKeyIPRules(apiKey)
	return apiKey, nil
}

func (s *APIKeyService) GetByIDAllowDeleted(ctx context.Context, id int64) (*APIKey, error) {
	if reader, ok := s.apiKeyRepo.(apiKeyDeletedReader); ok {
		apiKey, err := reader.GetByIDAllowDeleted(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get api key: %w", err)
		}
		s.compileAPIKeyIPRules(apiKey)
		return apiKey, nil
	}
	return s.GetByID(ctx, id)
}

// GetByKey 根据Key字符串获取API Key（用于认证）
