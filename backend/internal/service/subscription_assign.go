package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

// AssignSubscriptionInput 分配订阅输入
type AssignSubscriptionInput struct {
	UserID       int64
	GroupID      int64
	ValidityDays int
	AssignedBy   int64
	Notes        string
}

// AssignSubscription 分配订阅给用户（不允许重复分配）
func (s *SubscriptionService) AssignSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, error) {
	sub, _, err := s.assignSubscriptionWithReuse(ctx, input)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// AssignOrExtendSubscription 分配或续期订阅（用于兑换码等场景）
// 如果用户已有同分组的订阅：
//   - 未过期：从当前过期时间累加天数
//   - 已过期：从当前时间开始计算新的过期时间，并激活订阅
//
// 如果没有订阅：创建新订阅
func (s *SubscriptionService) AssignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	// 检查分组是否存在且为订阅类型
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, false, fmt.Errorf("group not found: %w", err)
	}
	if !group.IsSubscriptionType() {
		return nil, false, ErrGroupNotSubscriptionType
	}

	// 查询是否已有订阅
	existingSub, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
	if err != nil {
		// 不存在记录是正常情况，其他错误需要返回
		existingSub = nil
	}

	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 30
	}
	if validityDays > MaxValidityDays {
		validityDays = MaxValidityDays
	}

	// 已有订阅，执行续期（在事务中完成所有更新）
	if existingSub != nil {
		now := time.Now()
		var newExpiresAt time.Time

		if existingSub.ExpiresAt.After(now) {
			// 未过期：从当前过期时间累加
			newExpiresAt = existingSub.ExpiresAt.AddDate(0, 0, validityDays)
		} else {
			// 已过期：从当前时间开始计算
			newExpiresAt = now.AddDate(0, 0, validityDays)
		}

		// 确保不超过最大过期时间
		if newExpiresAt.After(MaxExpiresAt) {
			newExpiresAt = MaxExpiresAt
		}

		// 开启事务：ExtendExpiry + UpdateStatus + UpdateNotes 在同一事务中完成
		tx, err := s.entClient.Tx(ctx)
		if err != nil {
			return nil, false, fmt.Errorf("begin transaction: %w", err)
		}
		txCtx := dbent.NewTxContext(ctx, tx)

		// 更新过期时间
		if err := s.userSubRepo.ExtendExpiry(txCtx, existingSub.ID, newExpiresAt); err != nil {
			_ = tx.Rollback()
			return nil, false, fmt.Errorf("extend subscription: %w", err)
		}

		// 如果订阅已过期或被暂停，恢复为active状态
		if existingSub.Status != SubscriptionStatusActive {
			if err := s.userSubRepo.UpdateStatus(txCtx, existingSub.ID, SubscriptionStatusActive); err != nil {
				_ = tx.Rollback()
				return nil, false, fmt.Errorf("update subscription status: %w", err)
			}
		}

		// 追加备注
		if input.Notes != "" {
			newNotes := existingSub.Notes
			if newNotes != "" {
				newNotes += "\n"
			}
			newNotes += input.Notes
			if err := s.userSubRepo.UpdateNotes(txCtx, existingSub.ID, newNotes); err != nil {
				_ = tx.Rollback()
				return nil, false, fmt.Errorf("update subscription notes: %w", err)
			}
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			return nil, false, fmt.Errorf("commit transaction: %w", err)
		}

		// 失效订阅缓存
		s.InvalidateSubCache(input.UserID, input.GroupID)
		if s.billingCacheService != nil {
			userID, groupID := input.UserID, input.GroupID
			go func() {
				cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
			}()
		}

		// 返回更新后的订阅
		sub, err := s.userSubRepo.GetByID(ctx, existingSub.ID)
		return sub, true, err // true 表示是续期
	}

	// 没有订阅，创建新订阅
	sub, err := s.createSubscription(ctx, input)
	if err != nil {
		return nil, false, err
	}

	// 失效订阅缓存
	s.InvalidateSubCache(input.UserID, input.GroupID)
	if s.billingCacheService != nil {
		userID, groupID := input.UserID, input.GroupID
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}

	return sub, false, nil // false 表示是新建
}

// createSubscription 创建新订阅（内部方法）
func (s *SubscriptionService) createSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, error) {
	validityDays := input.ValidityDays
	if validityDays <= 0 {
		validityDays = 30
	}
	if validityDays > MaxValidityDays {
		validityDays = MaxValidityDays
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 0, validityDays)
	if expiresAt.After(MaxExpiresAt) {
		expiresAt = MaxExpiresAt
	}

	sub := &UserSubscription{
		UserID:     input.UserID,
		GroupID:    input.GroupID,
		StartsAt:   now,
		ExpiresAt:  expiresAt,
		Status:     SubscriptionStatusActive,
		AssignedAt: now,
		Notes:      input.Notes,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	// 只有当 AssignedBy > 0 时才设置（0 表示系统分配，如兑换码）
	if input.AssignedBy > 0 {
		sub.AssignedBy = &input.AssignedBy
	}

	if err := s.userSubRepo.Create(ctx, sub); err != nil {
		return nil, err
	}

	// 重新获取完整订阅信息（包含关联）
	return s.userSubRepo.GetByID(ctx, sub.ID)
}

func (s *SubscriptionService) assignSubscriptionWithReuse(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error) {
	// 检查分组是否存在且为订阅类型
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil {
		return nil, false, fmt.Errorf("group not found: %w", err)
	}
	if !group.IsSubscriptionType() {
		return nil, false, ErrGroupNotSubscriptionType
	}

	// 检查是否已存在订阅；若已存在，则按幂等成功返回现有订阅
	exists, err := s.userSubRepo.ExistsByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
	if err != nil {
		return nil, false, err
	}
	if exists {
		sub, getErr := s.userSubRepo.GetByUserIDAndGroupID(ctx, input.UserID, input.GroupID)
		if getErr != nil {
			return nil, false, getErr
		}
		if conflictReason, conflict := detectAssignSemanticConflict(sub, input); conflict {
			return nil, false, ErrSubscriptionAssignConflict.WithMetadata(map[string]string{
				"conflict_reason": conflictReason,
			})
		}
		return sub, true, nil
	}

	sub, err := s.createSubscription(ctx, input)
	if err != nil {
		return nil, false, err
	}

	// 失效订阅缓存
	s.InvalidateSubCache(input.UserID, input.GroupID)
	if s.billingCacheService != nil {
		userID, groupID := input.UserID, input.GroupID
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateSubscription(cacheCtx, userID, groupID)
		}()
	}

	return sub, false, nil
}

func detectAssignSemanticConflict(existing *UserSubscription, input *AssignSubscriptionInput) (string, bool) {
	if existing == nil || input == nil {
		return "", false
	}

	normalizedDays := normalizeAssignValidityDays(input.ValidityDays)
	if !existing.StartsAt.IsZero() {
		expectedExpiresAt := existing.StartsAt.AddDate(0, 0, normalizedDays)
		if expectedExpiresAt.After(MaxExpiresAt) {
			expectedExpiresAt = MaxExpiresAt
		}
		if !existing.ExpiresAt.Equal(expectedExpiresAt) {
			return "validity_days_mismatch", true
		}
	}

	existingNotes := strings.TrimSpace(existing.Notes)
	inputNotes := strings.TrimSpace(input.Notes)
	if existingNotes != inputNotes {
		return "notes_mismatch", true
	}

	return "", false
}

func normalizeAssignValidityDays(days int) int {
	if days <= 0 {
		days = 30
	}
	if days > MaxValidityDays {
		days = MaxValidityDays
	}
	return days
}
