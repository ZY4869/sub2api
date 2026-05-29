package service

import (
	"context"
	"fmt"
)

func (s *APIKeyService) GetByKey(ctx context.Context, key string) (*APIKey, error) {
	cacheKey := s.authCacheKey(key)

	if entry, ok := s.getAuthCacheEntry(ctx, cacheKey); ok {
		if apiKey, used, err := s.applyAuthCacheEntry(key, entry); used {
			if err != nil {
				return nil, fmt.Errorf("get api key: %w", err)
			}
			s.compileAPIKeyIPRules(apiKey)
			return apiKey, nil
		}
	}

	if s.authCfg.singleflight {
		value, err, _ := s.authGroup.Do(cacheKey, func() (any, error) {
			return s.loadAuthCacheEntry(ctx, key, cacheKey)
		})
		if err != nil {
			return nil, err
		}
		entry, _ := value.(*APIKeyAuthCacheEntry)
		if apiKey, used, err := s.applyAuthCacheEntry(key, entry); used {
			if err != nil {
				return nil, fmt.Errorf("get api key: %w", err)
			}
			s.compileAPIKeyIPRules(apiKey)
			return apiKey, nil
		}
	} else {
		entry, err := s.loadAuthCacheEntry(ctx, key, cacheKey)
		if err != nil {
			return nil, err
		}
		if apiKey, used, err := s.applyAuthCacheEntry(key, entry); used {
			if err != nil {
				return nil, fmt.Errorf("get api key: %w", err)
			}
			s.compileAPIKeyIPRules(apiKey)
			return apiKey, nil
		}
	}

	apiKey, err := s.apiKeyRepo.GetByKeyForAuth(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get api key: %w", err)
	}
	apiKey.Key = key
	s.compileAPIKeyIPRules(apiKey)
	return apiKey, nil
}

// Update 更新API Key
