package service

import (
	"context"
	"fmt"
)

func (s *APIKeyService) Delete(ctx context.Context, id int64, userID int64) error {
	key, ownerID, err := s.apiKeyRepo.GetKeyAndOwnerID(ctx, id)
	if err != nil {
		return fmt.Errorf("get api key: %w", err)
	}

	// 验证当前用户是否为该 API Key 的所有者
	if ownerID != userID {
		return ErrAPIKeyNotFound
	}

	// 清除Redis缓存（使用 userID 而非 apiKey.UserID）
	if s.cache != nil {
		_ = s.cache.DeleteCreateAttemptCount(ctx, userID)
	}
	s.InvalidateAuthCacheByKey(ctx, key)

	if err := s.apiKeyRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	s.lastUsedTouchL1.Delete(id)

	return nil
}

// ValidateKey 验证API Key是否有效（用于认证中间件）
