package service

import (
	"context"
	"fmt"
)

func (s *APIKeyService) GetAvailableGroups(ctx context.Context, userID int64) ([]Group, error) {
	// 获取用户信息
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// 获取所有活跃分组
	allGroups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups: %w", err)
	}

	// 获取用户的所有有效订阅
	activeSubscriptions, err := s.userSubRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list active subscriptions: %w", err)
	}

	// 构建订阅分组 ID 集合
	subscribedGroupIDs := make(map[int64]bool)
	for _, sub := range activeSubscriptions {
		subscribedGroupIDs[sub.GroupID] = true
	}

	// 过滤出用户有权限的分组
	availableGroups := make([]Group, 0)
	for _, group := range allGroups {
		if s.canUserBindGroupInternal(user, &group, subscribedGroupIDs) {
			availableGroups = append(availableGroups, group)
		}
	}

	return availableGroups, nil
}

// canUserBindGroupInternal 内部方法，检查用户是否可以绑定分组（使用预加载的订阅数据）

func (s *APIKeyService) canUserBindGroupInternal(user *User, group *Group, subscribedGroupIDs map[int64]bool) bool {
	// 订阅类型分组：需要有效订阅
	if group.IsSubscriptionType() {
		return subscribedGroupIDs[group.ID]
	}
	// 标准类型分组：使用原有逻辑
	return user.CanBindGroup(group.ID, group.IsExclusive)
}

func (s *APIKeyService) SearchAPIKeys(ctx context.Context, userID int64, keyword string, limit int) ([]APIKey, error) {
	keys, err := s.apiKeyRepo.SearchAPIKeys(ctx, userID, keyword, limit)
	if err != nil {
		return nil, fmt.Errorf("search api keys: %w", err)
	}
	return keys, nil
}

// GetUserGroupRates 获取用户的专属分组倍率配置
// 返回 map[groupID]rateMultiplier

func (s *APIKeyService) GetUserGroupRates(ctx context.Context, userID int64) (map[int64]float64, error) {
	if s.userGroupRateRepo == nil {
		return nil, nil
	}
	rates, err := s.userGroupRateRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user group rates: %w", err)
	}
	return rates, nil
}

// CheckAPIKeyQuotaAndExpiry checks if the API key is valid for use (not expired, quota not exhausted)
// Returns nil if valid, error if invalid
