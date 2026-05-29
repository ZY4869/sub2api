package service

import (
	"context"
	"fmt"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"strconv"
	"time"
)

func (s *APIKeyService) ValidateKey(ctx context.Context, key string) (*APIKey, *User, error) {
	// 获取API Key
	apiKey, err := s.GetByKey(ctx, key)
	if err != nil {
		return nil, nil, err
	}

	// 检查API Key状态
	if !apiKey.IsActive() {
		return nil, nil, infraerrors.Unauthorized("API_KEY_INACTIVE", "api key is not active")
	}

	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("get user: %w", err)
	}

	// 检查用户状态
	if !user.IsActive() {
		return nil, nil, ErrUserNotActive
	}

	return apiKey, user, nil
}

// TouchLastUsed 通过防抖更新 api_keys.last_used_at，减少高频写放大。
// 该操作为尽力而为，不应阻塞主请求链路。

func (s *APIKeyService) TouchLastUsed(ctx context.Context, keyID int64) error {
	if keyID <= 0 {
		return nil
	}

	now := time.Now()
	if v, ok := s.lastUsedTouchL1.Load(keyID); ok {
		if nextAllowedAt, ok := v.(time.Time); ok && now.Before(nextAllowedAt) {
			return nil
		}
	}

	_, err, _ := s.lastUsedTouchSF.Do(strconv.FormatInt(keyID, 10), func() (any, error) {
		latest := time.Now()
		if v, ok := s.lastUsedTouchL1.Load(keyID); ok {
			if nextAllowedAt, ok := v.(time.Time); ok && latest.Before(nextAllowedAt) {
				return nil, nil
			}
		}

		if err := s.apiKeyRepo.UpdateLastUsed(ctx, keyID, latest); err != nil {
			s.lastUsedTouchL1.Store(keyID, latest.Add(apiKeyLastUsedFailBackoff))
			return nil, fmt.Errorf("touch api key last used: %w", err)
		}
		s.lastUsedTouchL1.Store(keyID, latest.Add(apiKeyLastUsedMinTouch))
		return nil, nil
	})
	return err
}

// IncrementUsage 增加API Key使用次数（可选：用于统计）

func (s *APIKeyService) IncrementUsage(ctx context.Context, keyID int64) error {
	// 使用Redis计数器
	if s.cache != nil {
		cacheKey := fmt.Sprintf("apikey:usage:%d:%s", keyID, timezone.Now().Format("2006-01-02"))
		if err := s.cache.IncrementDailyUsage(ctx, cacheKey); err != nil {
			return fmt.Errorf("increment usage: %w", err)
		}
		// 设置24小时过期
		_ = s.cache.SetDailyUsageExpiry(ctx, cacheKey, 24*time.Hour)
	}
	return nil
}

// GetAvailableGroups 获取用户有权限绑定的分组列表
// 返回用户可以选择的分组：
// - 标准类型分组：公开的（非专属）或用户被明确允许的
// - 订阅类型分组：用户有有效订阅的
