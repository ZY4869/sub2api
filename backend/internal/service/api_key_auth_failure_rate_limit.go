package service

import (
	"context"
	"strings"
	"time"
)

const (
	apiKeyInvalidCredentialLimitDefault  = 20
	apiKeyInvalidCredentialWindowDefault = time.Hour
)

type apiKeyAuthFailureRateLimitCache interface {
	GetAuthFailureCount(ctx context.Context, key string) (int, error)
	IncrementAuthFailureCount(ctx context.Context, key string, ttl time.Duration) (int, error)
	DeleteAuthFailureCount(ctx context.Context, key string) error
}

func (s *APIKeyService) CheckAuthFailureRateLimit(ctx context.Context, identity string) error {
	cache, ok := s.authFailureCache()
	if !ok {
		return nil
	}
	cacheKey := s.authFailureCacheKey(identity)
	if cacheKey == "" {
		return nil
	}
	count, err := cache.GetAuthFailureCount(ctx, cacheKey)
	if err != nil {
		return nil
	}
	if count >= s.authFailureLimit() {
		return ErrAPIKeyRateLimited
	}
	return nil
}

func (s *APIKeyService) RecordAuthFailure(ctx context.Context, identity string) error {
	cache, ok := s.authFailureCache()
	if !ok {
		return nil
	}
	cacheKey := s.authFailureCacheKey(identity)
	if cacheKey == "" {
		return nil
	}
	count, err := cache.IncrementAuthFailureCount(ctx, cacheKey, s.authFailureWindow())
	if err != nil {
		return nil
	}
	if count > s.authFailureLimit() {
		return ErrAPIKeyRateLimited
	}
	return nil
}

func (s *APIKeyService) ClearAuthFailure(ctx context.Context, identity string) {
	cache, ok := s.authFailureCache()
	if !ok {
		return
	}
	cacheKey := s.authFailureCacheKey(identity)
	if cacheKey == "" {
		return
	}
	_ = cache.DeleteAuthFailureCount(ctx, cacheKey)
}

func (s *APIKeyService) authFailureCache() (apiKeyAuthFailureRateLimitCache, bool) {
	if s == nil || s.cache == nil {
		return nil, false
	}
	cache, ok := s.cache.(apiKeyAuthFailureRateLimitCache)
	return cache, ok
}

func (s *APIKeyService) authFailureCacheKey(identity string) string {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return ""
	}
	return s.authCacheKey("auth_failure:" + identity)
}

func (s *APIKeyService) authFailureLimit() int {
	if s == nil || s.cfg == nil || s.cfg.APIKeyAuth.InvalidCredentialLimit <= 0 {
		return apiKeyInvalidCredentialLimitDefault
	}
	return s.cfg.APIKeyAuth.InvalidCredentialLimit
}

func (s *APIKeyService) authFailureWindow() time.Duration {
	if s == nil || s.cfg == nil || s.cfg.APIKeyAuth.InvalidCredentialWindowSeconds <= 0 {
		return apiKeyInvalidCredentialWindowDefault
	}
	return time.Duration(s.cfg.APIKeyAuth.InvalidCredentialWindowSeconds) * time.Second
}
